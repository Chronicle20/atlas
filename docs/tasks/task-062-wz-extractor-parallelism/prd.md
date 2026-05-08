# atlas-wz-extractor Parallelism — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-03
---

## 1. Overview

`atlas-wz-extractor` parses every `*.wz` archive staged for a tenant and produces two output trees: HaRepacker-compatible XML (consumed by `atlas-data`) and PNG icon/map images (consumed by `atlas-assets`). Today the service runs as a single replica and processes WZ files **sequentially** inside one goroutine; only `RenderMaps` (Map.wz) is internally parallelized via a `runtime.NumCPU()` worker pool. A full extraction currently takes on the order of 10+ minutes per tenant/version, and that floor will rise as new WZ-derived stages or files are added.

This task introduces two layers of parallelism:

1. **Within-pod fan-out** across the top-level `*.wz` files using a bounded worker pool, replacing today's `for _, wzPath := range wzFiles` loop.
2. **Cross-pod sharding** by promoting each `(WZ-file, stage)` unit of work to a Kafka command, modeled directly on `atlas-data`'s `START_WORKER` / `COMMAND_TOPIC_DATA` pattern. Multiple `atlas-wz-extractor` replicas in one consumer group then share work via Kafka partition assignment, with per-job state tracked in Redis so a multi-pod extraction has a coherent progress view and a coherent tenant lock.

The shared input/output PVCs (`atlas-wz-input-pvc`, `atlas-data-pvc`, `atlas-assets-pvc`) are already `ReadWriteMany` on Longhorn, so cross-pod read/write needs no storage migration. The bottleneck this task targets is CPU-bound parse + serialize + render work, not I/O.

## 2. Goals

Primary goals:

- Reduce wall-clock time of a full per-tenant extraction by parallelizing across the top-level `*.wz` files within a single pod.
- Allow horizontal scale-out by partitioning extraction work across multiple `atlas-wz-extractor` replicas via Kafka, with one extraction job spanning multiple pods.
- Provide multi-pod-safe per-tenant exclusion so the same tenant cannot be extracted concurrently across pods.
- Provide a per-job progress endpoint that returns an aggregate view (units total / completed / failed / running) regardless of which pod each unit ran on.
- Preserve the current "continue on individual file/property error" semantics — partial extractions remain valid.

Non-goals:

- Rewriting the WZ parser (`wz/`) to stream a single image's parse across multiple goroutines.
- Changing the output directory layout (`{outputDir}/{tenantId}/{region}/{maj.min}/...`) — `atlas-data` and `atlas-assets` continue to read the existing tree.
- Re-architecting `atlas-data`'s import pipeline (`atlas-data` is unchanged by this task).
- Multi-tenant fairness, queueing, or cancellation of in-flight extractions.
- Changing the existing zip-upload flow (`PATCH /api/wz/input`) or the `xmlOnly` / `imagesOnly` query parameters.

## 3. User Stories

- As a platform operator, I want a single extraction to finish substantially faster on one pod, so that bringing up a new tenant or version doesn't block downstream work for ~10 minutes.
- As a platform operator, I want to scale `atlas-wz-extractor` horizontally for large workloads, so that adding replicas reduces wall-clock time without code changes.
- As a platform operator, I want a single REST call to know whether an extraction is still running, completed, completed-with-errors, or failed, so I don't need to scrape pod logs to know when to trigger `atlas-data`.
- As a platform operator, I want per-WZ-file progress visible during the run, so I can see which unit is taking the longest.
- As a developer adding a new WZ-derived output, I want the new output to participate in the parallel job model by default, so I don't reintroduce serial processing.

## 4. Functional Requirements

### 4.1 Unit of Work

Define an `ExtractionUnit` as a `(jobId, tenant, WZ file path, stage flags)` tuple, where stage flags mirror the existing `xmlOnly` / `imagesOnly` query parameters. The minimum unit granularity is a single WZ file, not an image inside a WZ file. Map.wz keeps its existing `RenderMaps` internal worker pool; from the cross-pod perspective Map.wz is still one unit.

### 4.2 Within-pod fan-out

- Replace the serial `for _, wzPath := range wzFiles` loop in `extraction.processorImpl.runExtraction` with a bounded worker pool.
- Default worker count: `runtime.NumCPU()`.
- Configurable override: env var `WZ_EXTRACT_PARALLELISM` (positive integer; invalid/zero values fall back to default and log a warning, matching the existing `WZ_EXTRACT_MAX_MAP_PIXELS` precedent).
- Within a single pod's processing of one job, units run concurrently up to that worker count. `wipeCharacterCache` (called once before the loop today) must run **before** any unit starts.
- A failure in one unit must not abort sibling units; the unit logs its error and the job-level counter records the failure (preserving today's "continue on error" semantics).

### 4.3 Cross-pod sharding (Kafka job model)

Modeled on `atlas-data`'s pattern in `services/atlas-data/atlas.com/data/kafka/consumer/data/`:

- New env var: `COMMAND_TOPIC_WZ_EXTRACTION` resolves the topic name.
- New command type: `START_EXTRACTION_UNIT` with a body of `{ jobId, wzFile, xmlOnly, imagesOnly }`.
- Tenant context is propagated via the standard `consumer.TenantHeaderParser` (same as atlas-data).
- Consumers register at `kafka.LastOffset`, persistent handler config (parity with atlas-data).
- A single consumer group ID (e.g. `wz-extractor-extraction`) ensures each unit is delivered to exactly one replica.
- The "dispatcher" (the pod that received the `POST /api/wz/extractions` request) emits one Kafka message per WZ file under the job. It does not itself process those units — it returns 202 with the `jobId` and lets the consumer group (which it is part of) pick them up.

### 4.4 Job state in Redis

A new Redis key namespace tracks job progress so any pod (and the REST status endpoint on any pod) can read coherent state:

- `wz-extractor:job:{jobId}` — hash with fields:
  - `tenantId`, `region`, `majorVersion`, `minorVersion`
  - `status` ∈ `{ pending, running, completed, completed_with_errors, failed }`
  - `unitsTotal`, `unitsCompleted`, `unitsFailed`
  - `xmlOnly`, `imagesOnly` (booleans)
  - `createdAt`, `updatedAt`, `completedAt` (RFC3339)
- `wz-extractor:job:{jobId}:units` — hash mapping `wzFileName -> { status, startedAt, completedAt, error? }` where `status ∈ { pending, running, succeeded, failed }`.
- `wz-extractor:tenant-lock:{tenantId}:{region}:{maj.min}` — string with TTL, set NX by the dispatcher and released after the job terminates. Replaces the in-process `extraction.tenantMutexRegistry`.

TTL on job records: 24h after `updatedAt` (long enough for an operator to inspect post-completion, short enough not to leak Redis space). The tenant lock has a separate TTL (e.g. 60 minutes) refreshed while units are running so a crashed dispatcher cannot wedge a tenant indefinitely.

### 4.5 Dispatcher behavior (`POST /api/wz/extractions`)

1. Parse tenant headers and stage flags as today.
2. Compute the WZ file list by globbing `INPUT_WZ_DIR/{tenantPath}/*.wz`.
3. If the file list is empty, return `400 Bad Request` (today's behavior is `202 Accepted` followed by an async error log — this is a regression worth fixing in passing).
4. Acquire the Redis tenant lock NX. On conflict, return `409 Conflict`.
5. Generate `jobId` (UUIDv4). Create the job record + unit records in Redis with `status=pending`.
6. Publish one `START_EXTRACTION_UNIT` Kafka message per WZ file, including `jobId`, `wzFile`, and stage flags. Tenant header propagated by the existing producer plumbing.
7. Set job `status=running`.
8. Return `202 Accepted` with `{ "jobId": "...", "unitsTotal": N }`.

The dispatcher does **not** execute any unit synchronously, even with `replicas=1`. Single-replica clusters still go through Kafka — the consumer in the same pod picks the messages back up. This keeps behavior uniform across replica counts.

### 4.6 Consumer behavior

For each `START_EXTRACTION_UNIT`:

1. Update the unit record in Redis: `status=running`, `startedAt=now`.
2. Run the existing per-WZ logic from `processorImpl.runExtraction`, scoped to one file:
   - `wz.Open` → optional XML serialize → optional icon extract → optional minimap extract → optional `RenderMaps` (when `wzFile=Map.wz`).
   - Continue-on-error within the unit (matching today's semantics).
   - The unit is "succeeded" if `wz.Open` succeeded and the file was processed; per-property errors logged but do not flip the unit to failed.
   - The unit is "failed" if `wz.Open` itself fails or any stage returns a non-recoverable error today's code returns to the caller (currently those errors are logged, not returned — preserve that, so unit-level failure is rare and reserved for the "couldn't even open the file" case).
3. On completion: update the unit record (`status=succeeded|failed`, `completedAt=now`, optional `error`), and atomically increment the job's `unitsCompleted` / `unitsFailed` counter.
4. After the increment, re-read job counters: if `unitsCompleted + unitsFailed == unitsTotal`, set job final status:
   - `unitsFailed == 0` → `completed`
   - `0 < unitsFailed < unitsTotal` → `completed_with_errors`
   - `unitsFailed == unitsTotal` → `failed`
   Then release the tenant lock.

The "I'm the last one home" check uses Redis atomic increment + read; whichever consumer brings the counter to total is the one that finalizes the job. This is the same race-safe terminator pattern used by other Atlas services.

### 4.7 Within-pod parallelism inside the consumer

A single consumer pod can be assigned multiple unit messages by Kafka. Today the kafka consumer machinery in this repo processes messages from one partition serially per handler. To get within-pod parallelism on top of cross-pod parallelism we have two options:

- **Option A — partition count = unit count**: Configure the topic with enough partitions (e.g. ≥ 16) and rely on Kafka assignment to distribute units. Single-pod parallelism is then bounded by partition count visible to that pod.
- **Option B — handler dispatches to a worker pool**: The handler enqueues the unit on an in-pod bounded worker pool (`WZ_EXTRACT_PARALLELISM` workers) and returns once enqueued; commit-on-completion semantics need care.

**Decision (this PRD):** Option A. It avoids the at-least-once / commit-after-async-completion pitfalls and keeps the consumer handler synchronous, matching atlas-data. Topic must be created with at least `WZ_EXTRACT_PARALLELISM` partitions; default 16. Documented in `docs/kafka.md` for ops.

### 4.8 Progress endpoint

New endpoint:

- `GET /api/wz/extractions/jobs/{jobId}` → JSON:API resource `wzExtractionJob` (see §5).

Existing endpoint kept:

- `GET /api/wz/extractions` (today's filesystem-scan view) — unchanged behavior, used as the "is anything on disk for this tenant?" check.

The new endpoint reads only Redis. It must work identically regardless of which pod handles the GET.

### 4.9 Failure semantics

- Unit-level failure: continue, increment `unitsFailed`, the job ultimately reports `completed_with_errors` or `failed`.
- Pod crash mid-unit: with Kafka at-least-once delivery, the unit is redelivered to another pod. Idempotency note: today's per-WZ logic over-writes XML/PNG output, so re-running a unit is safe. `wipeCharacterCache` is **not** re-run on redelivery (it ran once at dispatcher time before any unit message was published).
- Stale tenant lock: TTL refresh on every unit completion; if all consumers die, the lock expires within `lockTTL` and the tenant becomes re-extractable.
- Status endpoint with unknown `jobId`: `404 Not Found`.

### 4.10 Configuration / observability

- New env vars: `COMMAND_TOPIC_WZ_EXTRACTION`, `WZ_EXTRACT_PARALLELISM`, `WZ_REDIS_ADDR` / `WZ_REDIS_DB` (or whatever Redis env vars are already standard in this project — survey before final naming).
- Existing `WZ_EXTRACT_MAX_MAP_PIXELS` keeps its current meaning.
- Per-unit logs already exist (`Processing [Mob.wz]`, `Map render batch complete`, etc.). Add `jobId` and `wzFile` as structured log fields on every unit log so logs across pods can be correlated.
- Add OpenTelemetry spans `wz.extraction.dispatch` (one per job) and `wz.extraction.unit` (one per unit) under the existing tracer.

## 5. API Surface

### 5.1 `POST /api/wz/extractions` (modified)

Headers (unchanged): `TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`.
Query params (unchanged): `xmlOnly`, `imagesOnly`.

Response (changed):

```
202 Accepted
Content-Type: application/json
{
  "jobId": "f7c2e1aa-6c3c-4f6c-9b5e-2f9b1fbf3d41",
  "unitsTotal": 11,
  "status": "running"
}
```

Error responses:

- `400 Bad Request` — no `*.wz` files staged for this tenant (was previously a `202` followed by an async error log).
- `409 Conflict` — tenant already has an extraction in flight (Redis lock held).

### 5.2 `GET /api/wz/extractions/jobs/{jobId}` (new)

```
200 OK
Content-Type: application/vnd.api+json

{
  "data": {
    "type": "wzExtractionJob",
    "id": "f7c2e1aa-6c3c-4f6c-9b5e-2f9b1fbf3d41",
    "attributes": {
      "tenantId": "4ec40a5a-e596-4613-b498-e42450505e91",
      "region": "GMS",
      "majorVersion": 83,
      "minorVersion": 1,
      "status": "running",
      "xmlOnly": false,
      "imagesOnly": false,
      "unitsTotal": 11,
      "unitsCompleted": 7,
      "unitsFailed": 0,
      "createdAt": "2026-05-03T18:00:00Z",
      "updatedAt": "2026-05-03T18:04:21Z",
      "completedAt": null,
      "units": [
        { "wzFile": "Character.wz", "status": "succeeded", "startedAt": "...", "completedAt": "...", "error": null },
        { "wzFile": "Map.wz",       "status": "running",   "startedAt": "...", "completedAt": null,    "error": null },
        ...
      ]
    }
  }
}
```

`404 Not Found` for unknown `jobId`. JSON:API resource type follows the project's `GetName()` convention.

### 5.3 `GET /api/wz/extractions` (unchanged)

Same filesystem-scan semantics as today, kept for compatibility.

### 5.4 Kafka command

Topic: env var `COMMAND_TOPIC_WZ_EXTRACTION`. Headers: standard tenant + span headers (`consumer.TenantHeaderParser`, `consumer.SpanHeaderParser`).

```json
{
  "type": "START_EXTRACTION_UNIT",
  "body": {
    "jobId": "f7c2e1aa-6c3c-4f6c-9b5e-2f9b1fbf3d41",
    "wzFile": "Map.wz",
    "xmlOnly": false,
    "imagesOnly": false
  }
}
```

Consumer group: `wz-extractor-extraction`. Partition count: ≥ `WZ_EXTRACT_PARALLELISM` (default 16). Documented in `docs/kafka.md`.

## 6. Data Model

No relational schema changes. New Redis namespace under prefix `wz-extractor:` (see §4.4). Document the schema in `docs/storage.md`.

In-process state to remove: `extraction/mutex.go`'s `tenantMutexRegistry` (replaced by Redis tenant lock). Tests for the in-process mutex (`extraction/mutex_test.go`) are replaced by tests against the Redis lock implementation; if the rest of the service ever uses the in-process registry, those callers move to the Redis lock.

No multi-tenant scoping changes — every Redis key is implicitly scoped by `tenantId` (either embedded directly in the key or as a hash field).

## 7. Service Impact

**`atlas-wz-extractor`** (primary):

- New Kafka producer (`COMMAND_TOPIC_WZ_EXTRACTION`).
- New Kafka consumer (group `wz-extractor-extraction`) registered in `main.go` via the standard curried `InitConsumers` / `InitHandlers` pattern.
- Redis client wired into `main.go` (this service does not currently use Redis — confirm and add the standard atlas Redis lib if not already a dep).
- `extraction.Processor.Extract` refactored: today's signature processes the whole file list; new helper processes a single WZ file (`ExtractUnit(l, ctx, wzFile, xmlOnly, imagesOnly)`). The whole-list method remains for any in-pod-only callers.
- Per-tenant lock moves to Redis.
- New REST handler for `GET /api/wz/extractions/jobs/{jobId}`.
- Deployment manifest (`deploy/k8s/atlas-wz-extractor.yaml`):
  - Add `resources.requests` and `resources.limits` (CPU and memory).
  - Allow `replicas: N` (initial value still `1`; a follow-up rollout decides when to scale up).
  - Optionally add an `HorizontalPodAutoscaler` keyed on CPU.

**`atlas-data`**: no change. It still consumes the XML output from the shared PVC.

**`atlas-assets`**: no change.

**`atlas-ui`** / any current REST caller of `POST /api/wz/extractions`: must accept the new response shape (`{jobId, unitsTotal, status}`) and may opt into polling the new job endpoint. Today's callers (likely scripts) should still work because the response is JSON and 202 — but verify all in-repo callers as part of implementation.

**Kafka cluster**: a new topic must exist with the required partition count. Document in `deploy/k8s/atlas-wz-extractor.yaml` or in the topic-provisioning manifest used by the rest of the project.

**Redis**: this service starts depending on the existing project Redis instance. Confirm it is reachable from `atlas` namespace pods.

## 8. Non-Functional Requirements

**Performance:**

- Within-pod (`replicas=1`, `WZ_EXTRACT_PARALLELISM=NumCPU`): a representative full extraction must complete in under 50% of today's wall-clock time on the same node, **or** the spike report explains why CPU is no longer the bound. Measured against a baseline run on the existing prod node before the change is deployed.
- Cross-pod (`replicas=3`): scaling-up reduces wall-clock further. We do not commit a specific scaling factor in this PRD (Kafka rebalance overhead, Longhorn write contention, and per-WZ size skew make a flat number misleading); the design phase should propose a target.

**Concurrency safety:**

- Two simultaneous `POST` requests for the same tenant must result in exactly one running extraction (one returns 409). Verified by the Redis-lock test.
- Two pods cannot run the same unit twice concurrently — Kafka consumer-group assignment guarantees this.
- Idempotency on redelivery: a unit re-processed after pod crash must overwrite outputs without corruption.

**Multi-tenancy:**

- All Redis keys are tenant-scoped (either by embedding the tenant id in the key or by storing it as an attribute and using job-id-as-key with a separate `tenantId -> jobId` index).
- Tenant context flows through Kafka via the standard headers. Consumers reconstruct `tenant.Model` via `consumer.TenantHeaderParser` exactly like atlas-data.

**Observability:**

- Per-unit logs include `jobId` and `wzFile` structured fields.
- OTel spans: `wz.extraction.dispatch` (parent), `wz.extraction.unit` (one per unit), `wz.extraction.unit.<stage>` (xml / icons / minimaps / mapRender) — last tier optional but recommended.
- Existing log line `"map rendered"` etc. unchanged.

**Security:**

- No new external surface. Kafka and Redis are intra-cluster.
- The `jobId` is a UUIDv4 — non-guessable enough that the GET endpoint doesn't need additional authz beyond what the rest of the service already enforces (which today is "tenant header trust"). If the project later tightens authz, this endpoint follows.

**Resource limits:**

- Pod must declare CPU and memory `requests`/`limits`. Initial proposal: `requests: { cpu: 1, memory: 2Gi }`, `limits: { cpu: 4, memory: 8Gi }` — to be tuned in the design phase against measured peak working set.

## 9. Open Questions

1. **Redis library / addressing.** Does this project already standardize on a Go Redis client across other services? If so, reuse it. If not, the design phase picks one (likely `go-redis/v9`) and documents it.
2. **Topic provisioning.** Is the Kafka topic created by infra automation, or by the consumer at startup? `atlas-data` relies on infra automation — the design phase should confirm the same path is available here and that we can specify partition count there.
3. **Deployment resource sizing.** Concrete CPU/memory `requests`/`limits` need a measurement pass during/after the within-pod work lands. The PRD numbers above are placeholders.
4. **Map.wz inside a unit.** Map.wz today is far larger than the other WZ files (uses `RenderMaps` parallelism internally). Should it remain a single unit (simple) or be sub-divided into multiple units of `(Map.wz, image-id-range)` to balance the partitioning? Treating it as one unit keeps the design simple; leaving the existing `RenderMaps` internal pool intact still gives within-pod parallelism for that one unit. The design phase decides whether the simplicity wins or the imbalance is too costly.
5. **HPA.** Auto-scaling on CPU is the obvious choice, but extraction is bursty and short-lived. An HPA may scale up only after the work is mostly done. Consider whether HPA is included now or is a follow-up after measuring real-world burst durations.
6. **Backwards compatibility on `POST` response.** Today the response is `{"status": "started"}`. New shape adds `jobId` and `unitsTotal`. Are there callers (scripts, internal tooling) parsing the old shape that need a transition period? Worth one quick repo-wide grep before merge.

## 10. Acceptance Criteria

- [ ] `extraction/processor.go` no longer iterates `wzFiles` serially; the whole-list path uses a bounded worker pool, and a new `ExtractUnit` exists for single-file processing.
- [ ] `WZ_EXTRACT_PARALLELISM` env var is honored, with fallback to `runtime.NumCPU()` and a logged warning on invalid values.
- [ ] `POST /api/wz/extractions` publishes one `START_EXTRACTION_UNIT` Kafka message per WZ file under a freshly generated `jobId`, returns `202 { jobId, unitsTotal, status }`, and does not run any unit synchronously.
- [ ] A Kafka consumer in group `wz-extractor-extraction` handles `START_EXTRACTION_UNIT` messages by invoking `ExtractUnit` and updating Redis job/unit state.
- [ ] Two concurrent `POST` calls for the same tenant return exactly one `202` and one `409`, with state visible in Redis.
- [ ] When all units finalize, the job's `status` is one of `completed | completed_with_errors | failed` per the §4.6 rules and the tenant lock is released.
- [ ] `GET /api/wz/extractions/jobs/{jobId}` returns the JSON:API resource described in §5.2 and is correct from any pod (i.e. the dispatcher pod and any non-dispatcher pod return the same data).
- [ ] `GET /api/wz/extractions/jobs/{unknownId}` returns `404`.
- [ ] Unit-level failure does not abort sibling units; final `unitsFailed > 0` produces `completed_with_errors`.
- [ ] Pod crash mid-unit results in redelivery to another pod; final job state is correct (no double-count, no stuck `running`).
- [ ] Deployment manifest declares `resources.requests` and `resources.limits` and supports `replicas > 1` (kept at `1` for initial rollout).
- [ ] Topic creation / partition count documented in `services/atlas-wz-extractor/docs/kafka.md`.
- [ ] Redis schema documented in `services/atlas-wz-extractor/docs/storage.md`.
- [ ] Per-unit logs include `jobId` and `wzFile` structured fields.
- [ ] All affected Go packages build and tests pass; new tests cover: bounded worker pool, Redis tenant lock, dispatcher conflict handling, finalizer race, 404 path.
- [ ] Manual smoke: a representative tenant extraction with `replicas=1` finishes faster than today's baseline; the same extraction with `replicas=3` finishes faster still (numbers reported in the design/plan documents, not committed in the PRD).
