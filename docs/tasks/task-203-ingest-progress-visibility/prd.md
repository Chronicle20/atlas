# Ingest Progress Visibility — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-08
---

## 1. Overview

When an operator uploads WZ archives and triggers data processing, atlas-data
(running `MODE=rest`) renders a Kubernetes Job from the
`atlas-data-ingest-job-template` ConfigMap and submits it
(`runtime/rest/jobs.go:194`). That Job runs `MODE=ingest`, which fans eleven
registered workers out over the scope's WZ archives
(`data/runwz.go:45`, `data/workers/registry.go`). The run takes tens of minutes
on a full data set, and the Map worker alone has previously exceeded a
thirty-minute watchdog cutoff.

From the web UI, none of that is visible. The Setup and Baselines pages poll
`GET /api/data/status` every five seconds and render a document count
(`src/lib/hooks/api/useSeed.ts:298`). That count is a numerator with no
denominator: it climbs while the run proceeds and then stops climbing, but
nothing distinguishes "the run finished cleanly" from "the run is between
workers" from "the watchdog deleted a wedged Job an hour ago". An operator who
wants to know whether it is safe to publish a canonical baseline has to leave
the UI and inspect `kubectl get jobs` directly.

The backend is most of the way there already. `GET /api/data/process` exists and
is registered (`runtime/rest/resource.go:26`); it returns each ingest Job's
`active`/`succeeded`/`failed` counts and its scope/region/version labels. No UI
code calls it. A per-run Redis registry already exists and is written from both
sides — the REST pod stamps the Job name at creation (`jobs.go:206`) and the
ingest pod refreshes an `:updatedAt` heartbeat every thirty seconds
(`runtime/ingest/heartbeat.go:49`). This feature closes the remaining gaps:
scope the read endpoint to the caller, publish per-worker progress onto the
existing Redis channel, record a durable terminal outcome that survives Job
garbage collection and watchdog deletion, and surface all of it in the UI —
including gating the baseline publish control on it.

## 2. Goals

Primary goals:

- An operator can determine, entirely within the web UI, whether an ingest run
  for a given scope/region/version is in flight, has completed successfully, or
  has terminated abnormally.
- Progress is expressed against a known denominator: the eleven registered
  workers, each with an individual state and timing.
- A terminal outcome remains readable after the Kubernetes Job object is gone,
  whether it was garbage-collected (`ttlSecondsAfterFinished: 3600`) or deleted
  by the Watchdog (`runtime/rest/watchdog.go:95`).
- The Baselines page refuses to publish a canonical baseline while a shared-scope
  ingest for that region/version is in flight or ended abnormally.
- `GET /api/data/process` stops disclosing other tenants' ingest activity.

Non-goals:

- Sub-worker progress (items processed within a worker, e.g. maps ingested out of
  total maps). The eleven-worker denominator is the granularity for this task.
  Worker-internal instrumentation, including the Map worker specifically, is
  explicitly deferred.
- Cancelling or restarting an in-flight ingest run from the UI.
- Changing Watchdog timeout behaviour or the `TimeoutSecs: 7200` value set at
  `main.go:115`.
- A cross-tenant operator dashboard listing every ingest run in the namespace.
- Persisting ingest run history to Postgres. Run records live in Redis only
  (see §6).
- Streaming transports (SSE/WebSocket). Progress is delivered by polling.

## 3. User Stories

- As an operator, I want to see which of the eleven ingest workers have finished
  and which is currently running, so that I can estimate how much of a long run
  remains.
- As an operator, I want the UI to tell me an ingest run finished successfully,
  so that I can publish a canonical baseline without checking Kubernetes Job
  status by hand.
- As an operator, I want an ingest run that the Watchdog killed to be reported as
  abnormally terminated rather than silently disappearing, so that I do not
  publish a baseline over a half-ingested data set.
- As an operator, I want the Publish Baseline button disabled while a shared-scope
  ingest is running for that region/version, so that a mistimed click cannot
  capture a partial data set.
- As a tenant-scoped user of the Setup page, I want to see my own tenant's ingest
  progress and not other tenants' runs.

## 4. Functional Requirements

### FR-1 Run record

- **FR-1.1** atlas-data MUST maintain a *run record* per
  `(scope, region, majorVersion, minorVersion)` describing the most recent ingest
  run for that triple. A new run for the same triple overwrites the previous
  record; there is no run history.
- **FR-1.2** The run record MUST contain: a run identifier, the Kubernetes Job
  name, scope, region, version, tenant id (when scope is `tenants/<id>`), the run
  phase, run start time, run end time (when terminal), a per-worker entry for
  each of the eleven registered workers, and — for abnormal terminations — a
  human-readable reason.
- **FR-1.3** Run phase MUST be one of `running`, `succeeded`, `failed`,
  `stuck`. `stuck` is written by the Watchdog when it deletes a Job for heartbeat
  staleness. `failed` is written when the ingest process returns an error.
- **FR-1.4** Per-worker state MUST be one of `pending`, `running`, `succeeded`,
  `failed`, `skipped`, and MUST carry `startedAt` and `finishedAt` where
  applicable, plus an error message when the state is `failed`.
- **FR-1.5** `skipped` MUST be used for the `ErrCategoryAbsent` case
  (`data/runwz.go:63`) — a category genuinely absent from a monolithic `Data.wz`,
  such as v12 having no Quest. A skipped worker MUST NOT make a run non-succeeded.
- **FR-1.6** The worker roster in a freshly created run record MUST be derived
  from `workers.Registered`, so adding a twelfth worker changes the denominator
  without a second edit. The eleven current names are `ITEM`, `MOB`, `NPC`,
  `REACTOR`, `SKILL`, `QUEST`, `STRING`, `MAP`, `CHARACTER`, `UI`, `COMMODITY`.

### FR-2 Writing progress from the ingest pod

- **FR-2.1** The ingest pod MUST initialise the run record — phase `running`, all
  workers `pending`, `startedAt` stamped — before the prerequisite phase begins
  in `data.RunWorkers`.
- **FR-2.2** The ingest pod MUST transition a worker to `running` immediately
  before invoking it and to its terminal state immediately after, at the single
  `runOne` chokepoint in `data/runwz.go:60`. Both the sequential prerequisite
  phase (STRING) and the parallel fan-out phase MUST be covered.
- **FR-2.3** Worker-state writes MUST be safe under the parallel fan-out
  (`INGEST_MAX_PARALLEL`, default 4). Concurrent updates to the shared record
  MUST use the registry's optimistic-lock `Update` rather than read-modify-write
  via `Get`+`Put`.
- **FR-2.4** The ingest pod MUST write the terminal run phase (`succeeded` or
  `failed`, with the error) before the process exits, including when a worker
  error aborts the errgroup.
- **FR-2.5** A failure to write progress to Redis MUST NOT fail or abort the
  ingest run. Progress writes are best-effort and log at warn level, matching the
  existing heartbeat's behaviour (`runtime/ingest/heartbeat.go:55`).
- **FR-2.6** When the run record's key suffix cannot be derived from environment
  (the compose / unit-test path where `SCOPE`/`REGION` are unset — see
  `ingestJobSuffixFromEnv`, `heartbeat.go:76`), progress publication MUST be
  skipped silently, exactly as the heartbeat is today.

### FR-3 Writing terminal outcomes from the REST pod

- **FR-3.1** When the Watchdog deletes a Job for heartbeat staleness
  (`watchdog.go:95`), it MUST write run phase `stuck` with a reason naming the
  staleness timeout, instead of removing the run record. It MUST still remove the
  `<suffix>` and `<suffix>:updatedAt` heartbeat keys as it does today.
- **FR-3.2** `POST /api/data/process` MUST initialise (or reset) the run record
  for the target triple at Job-creation time, so a run that dies before the
  ingest pod's first write is still represented.

### FR-4 Read endpoint

- **FR-4.1** `GET /api/data/process` MUST resolve scope with the same rules
  `POST /api/data/process` uses today (`runtime/rest/resource.go:39`): absent or
  `tenant` resolves to `tenants/<callerTenantId>`; `shared` resolves to the
  canonical dataset and requires `X-Atlas-Operator: 1`; any other value is a 400.
- **FR-4.2** The endpoint MUST return only the run for the resolved
  scope/region/version. It MUST NOT return runs belonging to other tenants. This
  replaces the current unfiltered namespace-wide list
  (`runtime/rest/resource.go:97`).
- **FR-4.3** The response MUST merge the Redis run record with live Kubernetes
  Job status for the same triple. When the two disagree — the record says
  `running` but no Job exists and the heartbeat is absent — the endpoint MUST
  report the run as `unknown` rather than `running`.
- **FR-4.4** When no run record exists for the triple, the endpoint MUST return
  200 with a record whose phase is `none`, not a 404. "No ingest has been run"
  is a valid, actionable answer.
- **FR-4.5** When the Kubernetes client is unavailable (compose, or not running
  `MODE=rest`), the endpoint MUST still serve the Redis run record if one exists,
  degrading only the live-Job cross-check. It MUST NOT blanket-503 as it does
  today (`runtime/rest/resource.go:93`).
- **FR-4.6** The response MUST be a JSON:API document, replacing the current raw
  JSON shape. The endpoint has no existing consumers, so this is not a breaking
  change in practice.

### FR-5 UI

- **FR-5.1** The Setup page MUST render an ingest progress panel for the active
  tenant's scope, polling the endpoint on the same five-second cadence the
  existing status queries use (`useSeed.ts:308`).
- **FR-5.2** The Baselines page MUST render the same panel for the selected
  region/version at `shared` scope, sending the operator header.
- **FR-5.3** The panel MUST show the overall phase, elapsed time (or total
  duration when terminal), a completed-of-total worker count, and a per-worker
  list with each worker's state and duration.
- **FR-5.4** Phases `failed` and `stuck` MUST be visually distinct from
  `succeeded` and MUST surface the recorded reason or error text.
- **FR-5.5** The Baselines page's `publishDisabled` condition
  (`src/pages/BaselinesPage.tsx:153`) MUST additionally be true when the
  shared-scope run for the selected region/version has phase `running`, `stuck`,
  `failed`, or `unknown`. The disabled control MUST explain why.
- **FR-5.6** Phases `none` and `succeeded` MUST NOT block publishing. `none` is
  the pre-existing state for every scope that has never been ingested through
  this mechanism and must not regress today's behaviour.
- **FR-5.7** Polling MUST stop escalating when the endpoint is unreachable or
  returns 503; the panel MUST degrade to an explicit "progress unavailable"
  state rather than an error toast loop.

## 5. API Surface

### GET /api/data/process (modified)

Query parameters:

- `scope` (optional): `""` or `"tenant"` (default) — the caller's own tenant;
  `"shared"` — the version-scoped canonical dataset, requires
  `X-Atlas-Operator: 1`.

Region and version come from the standard tenant headers, as they do for
`POST /api/data/process`.

Response `200`, JSON:API, resource type `ingestRun`, id
`<scope>:<region>:<major>.<minor>`:

```json
{
  "data": {
    "type": "ingestRun",
    "id": "tenants/0a1b.../GMS/83.1",
    "attributes": {
      "runId": "…",
      "jobName": "ingest-t-0a1b2c3d-gms-83-1-x7f2qa",
      "scope": "tenants/0a1b…",
      "region": "GMS",
      "version": "83.1",
      "tenant": "0a1b…",
      "phase": "running",
      "startedAt": "2026-08-08T10:14:02Z",
      "finishedAt": null,
      "reason": null,
      "workersTotal": 11,
      "workersComplete": 7,
      "workers": [
        {
          "name": "STRING",
          "state": "succeeded",
          "startedAt": "2026-08-08T10:14:03Z",
          "finishedAt": "2026-08-08T10:16:41Z",
          "error": null
        },
        {
          "name": "MAP",
          "state": "running",
          "startedAt": "2026-08-08T10:16:42Z",
          "finishedAt": null,
          "error": null
        }
      ]
    }
  }
}
```

`phase` is one of `none`, `running`, `succeeded`, `failed`, `stuck`, `unknown`.
Worker `state` is one of `pending`, `running`, `succeeded`, `failed`, `skipped`.

Error cases:

- `400 Bad Request` — invalid `scope` value.
- `403 Forbidden` — `scope=shared` without `X-Atlas-Operator: 1`.

`503` is no longer returned for an absent Kubernetes client (FR-4.5); a Redis
outage that prevents reading the record yields the standard error response.

### POST /api/data/process (unchanged wire contract)

Behaviour extended per FR-3.2 — the run record is initialised at Job creation.
Request and response shapes are unchanged.

## 6. Data Model

No relational schema change. No migration.

Run records live in the existing env-prefixed Redis namespace `data-ingest`
(`jobs.go:38`), alongside the two keys already in use:

| Key | Type | Written by | TTL |
|---|---|---|---|
| `<prefix>:data-ingest:<suffix>` | job name (string) | REST pod at create | 1 h (existing) |
| `<prefix>:data-ingest:<suffix>:updatedAt` | RFC3339 (string) | ingest pod heartbeat | 1 h (existing) |
| `<prefix>:data-ingest:<suffix>:run` | run record (JSON) | both pods | **7 days** (new) |

`<suffix>` is `<scope>:<region>:<major>.<minor>`, unchanged from
`ingestJobKeySuffix` (`jobs.go:49`). `<prefix>` comes from
`libs/atlas-redis` `KeyPrefix()` and is environment-aware, so PR overlays are
already isolated.

The run record is stored under a **separate `:run` key** rather than extending
the existing keys, because those are typed `Registry[string, string]` and are
read by the Watchdog's staleness check. The run record is a
`Registry[string, RunRecord]`; `libs/atlas-redis` marshals values as JSON
automatically (`registry.go:30`), so no bespoke encoding is needed.

The whole record — including all eleven worker entries — lives under one key so
a read is a single `Get`. Concurrent worker transitions during the parallel
phase use `Registry.Update`, whose optimistic `WATCH` loop
(`registry.go:76`) is sized for far heavier contention than eleven workers at
`INGEST_MAX_PARALLEL=4`.

The 7-day TTL is refreshed on every write, so an in-flight run cannot expire
mid-run. Redis is not a durable store: a flush or eviction loses the record, and
the endpoint then reports `none`. This is the accepted trade-off — recording
runs in Postgres was considered and deliberately deferred (§2 non-goals). It is
the reason FR-5.6 treats `none` as non-blocking: a lost record must not
permanently wedge the publish control.

## 7. Service Impact

**atlas-data** (`services/atlas-data/atlas.com/data`) — all backend changes:

- `runtime/rest/jobs.go` — run-record registry constructor and key suffix; run
  record initialised in `Create` (FR-3.2).
- `runtime/rest/resource.go` — `processStatus` rewritten: scope resolution
  (FR-4.1/4.2), Redis+Kubernetes merge (FR-4.3), `none`/`unknown` phases
  (FR-4.3/4.4), degraded-mode behaviour (FR-4.5), JSON:API marshalling (FR-4.6).
- `runtime/rest/watchdog.go` — `deleteStuckJob` writes phase `stuck` (FR-3.1).
- `runtime/ingest/` — new run-record writer alongside the existing heartbeat;
  initialisation and terminal write around the `RunWorkers` call in `run.go`.
- `data/runwz.go` — `runOne` instrumented with per-worker transitions (FR-2.2),
  including the `ErrCategoryAbsent` skip path (FR-1.5) and the prerequisite
  phase.
- New JSON:API rest model for `ingestRun`.
- `services/atlas-data/docs/rest.md` — the `GET /api/data/process` section
  (lines 81–90) rewritten for the new scope rules and JSON:API shape.

**atlas-ui** (`services/atlas-ui`):

- `src/services/api/seed.service.ts` — a `getIngestRun` fetcher for tenant scope
  and a canonical-headers variant for shared scope, alongside the existing
  `postProcess` calls.
- `src/lib/hooks/api/useSeed.ts` — polling query hook, five-second interval.
- New progress panel component under `src/components/features/`.
- `src/pages/SetupPage.tsx` — panel mounted for tenant scope (FR-5.1).
- `src/pages/BaselinesPage.tsx` — panel mounted for shared scope (FR-5.2) and
  `publishDisabled` extended (FR-5.5).
- New TypeScript types for the `ingestRun` resource.

**Deployment**: no change. No new ConfigMap, Secret, RBAC rule, service
registration, or Kafka topic. The `atlas-data` ServiceAccount's existing Job
permissions and the existing Redis connection are sufficient.

## 8. Non-Functional Requirements

**Multi-tenancy.** FR-4.1/4.2 close a real disclosure: `processStatus` currently
lists every ingest Job in the namespace with its `tenant` label
(`resource.go:97`). Scope resolution MUST reuse the same code path as
`processCreate` rather than a parallel implementation, so the operator gate on
`shared` cannot drift between the two verbs. Redis keys are already
environment-prefixed and scope-qualified.

**Performance.** One `Get` per poll per open UI page, plus one Kubernetes Job
lookup. The Kubernetes lookup MUST be filtered by label selector to the resolved
triple rather than listing the namespace and filtering client-side. Worker
transitions add at most 22 Redis writes across a multi-minute run — negligible
against the existing 30-second heartbeat.

**Correctness under concurrency.** The parallel fan-out means up to
`INGEST_MAX_PARALLEL` simultaneous transitions on one key; FR-2.3 mandates the
optimistic-lock path. A read-modify-write via `Get`+`Put` would silently drop
worker states and is explicitly disallowed.

**Failure isolation.** FR-2.5 is a hard requirement: an ingest run that would
have succeeded MUST NOT fail because a progress write failed. Redis is
best-effort telemetry here, not the system of record for the ingested data
itself.

**Observability.** Worker transitions SHOULD log at info level with worker name
and duration, giving the same information in logs for operators debugging
without the UI.

**Backward compatibility.** `GET /api/data/process` changes shape and gains a
scope filter. Verified to have zero consumers in this repository: no reference
exists anywhere in `services/atlas-ui/src` (only `POST` is called, at
`seed.service.ts:241,269`).

**Guards.** The Redis work goes through `libs/atlas-redis` registries, satisfying
`tools/redis-key-guard.sh`. Any new goroutine MUST be spawned via `routine.Go`
per `tools/goroutine-guard.sh`; the existing heartbeat spawn in
`runtime/ingest/run.go:44` is the pattern to follow.

## 9. Open Questions

- Should a `stuck` run's per-worker record be preserved as-is (showing which
  worker was running when the Watchdog fired), or should the in-flight worker be
  marked `failed`? Preserving it is more informative for diagnosis; the design
  phase should settle the exact rendering.
- Elapsed time for an in-flight run can be computed either from `startedAt` in
  the record or from the Kubernetes Job's `startTime`. These can disagree when
  the Job was pending in the scheduler. Design phase should pick one and say why.
- Whether the Setup page's existing document-count badge
  (`SetupPage.tsx:181`) should remain alongside the new panel or be folded into
  it.

## 10. Acceptance Criteria

- [ ] `GET /api/data/process` with no `scope` returns only the calling tenant's
      run record; a second tenant's concurrent run is not visible in the response.
- [ ] `GET /api/data/process?scope=shared` without `X-Atlas-Operator: 1` returns
      403; with the header it returns the canonical scope's record.
- [ ] `GET /api/data/process?scope=bogus` returns 400.
- [ ] The response validates as a JSON:API document with resource type
      `ingestRun`.
- [ ] A triple that has never been ingested returns 200 with `phase: "none"`,
      not 404.
- [ ] During a live ingest, successive polls show workers advancing
      `pending` → `running` → `succeeded`, and `workersComplete` increases
      monotonically to 11.
- [ ] STRING reaches `succeeded` before any other worker leaves `pending`,
      matching the prerequisite ordering in `data/runwz.go`.
- [ ] A completed run reports `phase: "succeeded"` and remains readable after the
      Kubernetes Job is garbage-collected past `ttlSecondsAfterFinished: 3600`.
- [ ] A Watchdog-deleted Job leaves `phase: "stuck"` with a reason naming the
      timeout, and the run remains readable after the Job object is gone.
- [ ] A worker returning an error yields `phase: "failed"`, that worker in state
      `failed` with its error text, and the run record still lists the states of
      workers that had already completed.
- [ ] A monolithic `Data.wz` scope lacking a category records that worker as
      `skipped` and still reports `phase: "succeeded"`.
- [ ] Injecting a Redis write failure during a run does not fail the ingest: the
      run completes and the data lands.
- [ ] The Setup page shows the tenant-scope panel, updating without a manual
      refresh, and shows "progress unavailable" rather than looping errors when
      the endpoint is unreachable.
- [ ] The Baselines page's Publish Baseline button is disabled with an
      explanatory message while the shared-scope run for the selected
      region/version is `running`, `stuck`, `failed`, or `unknown`, and enabled
      when the phase is `succeeded` or `none`.
- [ ] `go test -race ./...`, `go vet ./...`, and `go build ./...` are clean in
      `services/atlas-data`.
- [ ] `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, and
      `tools/lint.sh --check` are clean from the repo root.
- [ ] `npm run build` (which type-checks tests) is clean in `services/atlas-ui`.
- [ ] `services/atlas-data/docs/rest.md` documents the new `GET
      /api/data/process` contract.
