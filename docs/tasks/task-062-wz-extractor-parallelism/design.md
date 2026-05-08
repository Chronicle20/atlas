# atlas-wz-extractor Parallelism — Design

Status: Draft
Created: 2026-05-08
Companion: `prd.md` (v1, approved)

---

## 1. Scope of this design

The PRD (§3–§10) is approved as the contract: API shape, response codes, env vars, acceptance criteria. This document does **not** revisit those decisions. It covers:

- **How** the dispatcher / consumer / job-store / lock fit together inside `atlas-wz-extractor` and what package boundaries make that maintainable.
- **Alternatives considered** for the few mechanical sub-decisions the PRD deferred (Redis client, Map.wz granularity, lock-TTL refresh, idempotent finalize, within-pod parallelism path).
- **Failure-mode traversal** for each cross-pod hazard (redelivery, crashed dispatcher, lock expiry mid-job, finalize race) so the plan can write tests against named scenarios.

Where the PRD already chose, we restate the choice with one-sentence justification and move on. Where the PRD deferred to design, the trade-off is laid out below.

---

## 2. Architectural overview

```
                                  ┌─────────────────────────┐
   POST /api/wz/extractions ─────▶│  Dispatcher (1 pod)     │
                                  │  - Redis NX tenant lock │
                                  │  - wipeCharacterCache   │
                                  │  - Create job + units   │
                                  │  - Emit N Kafka msgs    │
                                  └────────────┬────────────┘
                                               │  N × START_EXTRACTION_UNIT
                                               ▼
                          ┌────────────────────────────────────────┐
                          │   Kafka topic COMMAND_TOPIC_WZ_EXTRACTION  │
                          │   ≥ 16 partitions, group "wz-extractor-extraction" │
                          └─────┬───────────────┬───────────────┬──┘
                                │               │               │
                          ┌─────▼─────┐   ┌─────▼─────┐   ┌─────▼─────┐
                          │  Pod A    │   │  Pod B    │   │  Pod C    │
                          │ Consumer  │   │ Consumer  │   │ Consumer  │
                          │ ExtractUnit│  │ ExtractUnit│  │ ExtractUnit│
                          └─────┬─────┘   └─────┬─────┘   └─────┬─────┘
                                │               │               │
                                └─────┬─────────┴────┬──────────┘
                                      │              │
                                      ▼              ▼
                              ┌────────────┐   ┌────────────────────┐
                              │  Redis     │   │ PVCs (RWX/Longhorn)│
                              │ job state  │   │ atlas-data-pvc     │
                              │ tenant lock│   │ atlas-assets-pvc   │
                              └────────────┘   └────────────────────┘

   GET /api/wz/extractions/jobs/{jobId} ─▶ any pod ─▶ Redis (read-only)
```

Everything outside the two new boxes (Redis state + Kafka topic) is pre-existing infra:

- `libs/atlas-kafka` — manager, consumer/producer, `TenantHeaderParser`, `SpanHeaderParser`. Already used by atlas-data exactly the way we need.
- `libs/atlas-redis` — `Connect()`, `Lock` with NX+TTL, plus `keys.go` namespacing helpers. Already used by atlas-buffs / atlas-npc-shops / atlas-messengers / atlas-portals.
- `libs/atlas-tenant`, `libs/atlas-rest/server`, `libs/atlas-tracing` — same as today.

The PRD's "audit existing libs first" directive (per project memory) applies: nothing new is built that `libs/atlas-redis` or `libs/atlas-kafka` already provides. The only new code is wiring + domain-specific job/unit state structures.

---

## 3. Package layout

```
services/atlas-wz-extractor/atlas.com/wz-extractor/
├── extraction/
│   ├── processor.go            # MODIFIED: Extract (whole-list) + ExtractUnit (single file)
│   ├── pool.go                 # NEW: bounded worker pool used by Extract (within-pod fallback path)
│   ├── dispatcher.go           # NEW: handleExtract orchestration (lock, wipe, create job, publish)
│   ├── resource.go             # MODIFIED: register POST + GET-by-id + existing GET endpoints
│   ├── job_handler.go          # NEW: GET /api/wz/extractions/jobs/{jobId}
│   ├── job/                    # NEW package
│   │   ├── model.go            # Job, Unit, Status enums (immutable + Builder per atlas conventions)
│   │   ├── store.go            # Store interface + Redis impl
│   │   ├── store_test.go       # uses miniredis
│   │   └── keys.go             # key naming, kept private to this package
│   ├── lock/                   # NEW package
│   │   ├── tenant_lock.go      # wraps libs/atlas-redis Lock with auto-refresh goroutine
│   │   └── tenant_lock_test.go
│   ├── mutex.go                # DELETED
│   ├── mutex_test.go           # DELETED
│   ├── status.go               # unchanged (filesystem-scan endpoints)
│   └── ...                     # rest unchanged
├── kafka/                      # NEW (mirrors atlas-data layout)
│   ├── consumer/
│   │   └── extraction/
│   │       ├── consumer.go     # InitConsumers + InitHandlers
│   │       └── kafka.go        # EnvCommandTopic, CommandStartExtractionUnit, command, body types
│   └── producer/
│       └── producer.go         # ProviderImpl mirroring atlas-data
└── main.go                     # MODIFIED: wire Redis, Kafka producer/consumer
```

Mirroring `services/atlas-data/atlas.com/data/kafka/{consumer,producer}/` is deliberate. The PRD specifies that pattern; copying the structure means a future reader spots both implementations are the same shape and `backend-guidelines-reviewer` doesn't flag a one-off.

`extraction.Processor` becomes the single owner of "process one WZ file." `dispatcher.go` and `kafka/consumer/extraction/consumer.go` both call `processor.ExtractUnit` — they don't duplicate the per-WZ logic.

---

## 4. Decision log

### 4.1 Redis client — reuse `libs/atlas-redis` (decided)

The PRD's open question #1 asks whether the project standardizes on a Redis client. Survey result: yes, `libs/atlas-redis` (which wraps `github.com/redis/go-redis/v9`) is used by atlas-buffs, atlas-npc-shops, atlas-messengers, atlas-portals, atlas-pets. It already exposes `Connect(l)` (reads `REDIS_URL` / `REDIS_PASSWORD`) and a `Lock` type with NX + TTL + Extend. We use both.

**Implication for env vars.** The PRD listed `WZ_REDIS_ADDR` / `WZ_REDIS_DB` as candidates. Since we reuse the shared library, we inherit `REDIS_URL` and `REDIS_PASSWORD` instead — same names every other service uses. The PRD's §4.10 envvar list updates to: `COMMAND_TOPIC_WZ_EXTRACTION`, `WZ_EXTRACT_PARALLELISM` (new); `REDIS_URL`, `REDIS_PASSWORD` (reused).

### 4.2 Within-pod parallelism — Option A (partition count) (PRD-decided, restated)

PRD §4.7 chose partition-count parallelism over async-handler-pool. Restated rationale: at-least-once delivery + commit-after-async is the failure mode the manager-based consumer in `libs/atlas-kafka` does **not** handle (it commits after the synchronous handler returns). Option B would require either a custom commit hook or accepting that a pod crash re-runs every unit currently in-flight, which conflicts with the "redelivery → another pod, not double-running on this one" property we want. Option A trivially gives that.

**Cost.** A single `replicas=1` deployment is bounded by partition count, not `runtime.NumCPU()`. Default partition count of 16 ≥ NumCPU on every realistic node, so this is a non-issue in practice. We document partition count in `services/atlas-wz-extractor/docs/kafka.md` and require it ≥ `WZ_EXTRACT_PARALLELISM`.

**Within-pod fallback path.** The new `pool.go` (a `runtime.NumCPU()`-bounded worker pool calling `ExtractUnit` directly) exists for two reasons: (1) PRD §10 acceptance criterion requires that `Extract` no longer iterates serially; (2) it is the fallback the dispatcher would use if Redis or Kafka were unreachable, but since we reject that fallback (see §4.5 below), the pool is currently used only by `Extract`-the-method which is retained for tests and any future single-process caller. Documented as such in the file header so reviewers don't add a "looks dead" warning.

### 4.3 Map.wz granularity — keep as one unit (decided)

PRD open question #4. Three options on the table:

| Option | Description | Cost | Benefit |
|---|---|---|---|
| **A (chosen)** | Map.wz is one unit. `RenderMaps` keeps its internal `runtime.NumCPU()` pool. | Map.wz wall-clock is the bottom of the wall-clock floor (`max(Map.wz, sum(others)/replicas)`). | Simplest. No state explosion. Idempotent on redelivery (full re-render overwrites). |
| B | Subdivide Map.wz into N units of `(Map.wz, image-id-range)`. | Aggregation: each sub-unit must publish a partial counter; finalizer logic gets a 2-level hierarchy. | Cross-pod parallelism on the longest unit. |
| C | Keep Map.wz as one unit *and* let it spawn nested START messages (recursive job model). | Full job-model recursion: cycles, partial-failure semantics, redelivery semantics all double in complexity. | Same as B without the upfront partition planning. |

Option A wins because: (a) `RenderMaps` already gives within-pod parallelism on Map.wz, (b) the PRD's §8 NFR target of "<50% of today's wall-clock at replicas=1" is achievable without subdividing (today's serial loop runs Map.wz alongside ~10 other WZ files; parallelizing those ~10 already shaves substantially), and (c) the simplicity of A keeps the finalize/idempotency invariants tractable.

**Trade-off accepted.** With `replicas=N` large, Map.wz becomes the long pole. The Plan's "manual smoke" measurement should record per-WZ-file wall-clock so a future task can decide whether subdivision is warranted. We don't pre-build hooks for it.

### 4.4 Tenant lock TTL refresh — auto-refresh goroutine (decided)

PRD §4.4 sets the lock TTL at "e.g. 60 minutes" and §4.9 says it's "refreshed while units are running." The PRD leaves the refresh mechanism unspecified.

**Problem.** If a single unit (Map.wz) takes >60min on a slow pod, the lock can expire mid-job and a second `POST` could be admitted. Unit-completion-only refresh is insufficient — Map.wz is one unit and only refreshes once.

**Decision.** The dispatcher spawns a refresh goroutine bound to the job's lifetime, calling `Lock.Extend(ctx, key)` every `lockTTL/3` until the job finalizes (`unitsCompleted + unitsFailed == unitsTotal`) or the context is cancelled (pod shutdown). The goroutine watches the same Redis job-record's `status` to know when to stop.

**Concrete params.** `lockTTL = 60 * time.Minute`, refresh every 20 minutes. 60min was kept from the PRD; 20min is the standard "refresh at one-third TTL" cushion that survives one missed refresh.

**Crashed-dispatcher fallback.** If the dispatcher pod dies before the job finalizes, the refresh goroutine dies with it. Lock expires within 60min; tenant becomes re-extractable. In-flight Kafka units continue to run on whichever consumer they were dispatched to — those are independent of the dispatcher pod. After expiry a second `POST` would be admitted; if it overlaps in-flight units from the first job, they over-write the same output paths idempotently. Final job state for the first job remains correct as long as the consumer pool is still alive; otherwise it stays `running` until the 24h job-record TTL reaps it. We accept this — a 60min crashed-dispatcher window after which a tenant unblocks is acceptable for an operations-tool extraction.

### 4.5 Dispatcher always goes through Kafka (PRD-decided, restated)

PRD §4.5 closes with "single-replica clusters still go through Kafka." This is restated because it's the linchpin of the simplicity story: there is exactly one path through the system at runtime, regardless of replica count.

**Implication for failure of Kafka or Redis at dispatch time.** If Redis is unreachable: `POST` returns 503. If Kafka producer fails to publish: dispatcher releases the Redis lock and returns 500 with the partial unitsTotal in the error body so the operator sees what was attempted. We do not fall back to in-process execution. (This is a behavior change from today's "extraction always runs once `POST` returns 202"; documented in §7.)

### 4.6 Idempotent finalize — SETNX on the unit's terminal state (decided)

Kafka at-least-once means a unit message can be redelivered after the consumer has already incremented the counter. Without protection, `unitsCompleted` overshoots `unitsTotal` and the finalize check `unitsCompleted + unitsFailed == unitsTotal` either fires twice (writing `completed` twice — harmless but ugly) or fires never (if redelivery counts toward only one outcome and races push past total).

**Decision.** The unit record is the source of truth. The increment is gated on a transition from `pending|running` to a terminal state:

```
WATCH wz-extractor:job:{jobId}:units
GET   field=wzFile  ->  current unit JSON
if current.status in {succeeded, failed}: discard, log "already terminal", do nothing
MULTI
HSET  wz-extractor:job:{jobId}:units field=wzFile value=<new JSON with terminal status>
HINCRBY wz-extractor:job:{jobId} unitsCompleted +1   (or unitsFailed)
HSET  wz-extractor:job:{jobId} updatedAt <now>
EXEC
```

If `EXEC` returns nil (concurrent change), the consumer retries up to N times. If still failing, log and skip. The check "is this unit already terminal" guards both the redelivery race and the concurrent-finalize race in a single transaction.

The "last one home" finalizer (PRD §4.6 step 4) reads the counters after `EXEC`, computes `(unitsCompleted + unitsFailed == unitsTotal)`, and uses a separate `SET status NX` (only set if currently `running`) to declare the terminal job status. Whichever consumer wins that `SET` releases the lock; losers are no-ops.

This keeps everything in standard Redis primitives (no Lua script). `libs/atlas-redis` doesn't expose WATCH/MULTI today, but go-redis client supports it directly via `client.Watch(...)`. We use the underlying client; the abstraction in `libs/atlas-redis` is not violated because we are adding new repository code that *uses* the lib's connection, not modifying the lib.

### 4.7 Job-record schema (decided)

PRD §4.4 sketched the keys. Concrete schema:

```
Key:   wz-extractor:job:{jobId}                       Type: HASH
Fields:
  tenantId          string (UUIDv4)
  region            string
  majorVersion      string (decimal)
  minorVersion      string (decimal)
  status            string ∈ {pending, running, completed, completed_with_errors, failed}
  unitsTotal        string (decimal int)
  unitsCompleted    string (decimal int)  ← HINCRBY target
  unitsFailed       string (decimal int)  ← HINCRBY target
  xmlOnly           string ("true"|"false")
  imagesOnly        string ("true"|"false")
  createdAt         string (RFC3339)
  updatedAt         string (RFC3339)
  completedAt       string (RFC3339|"")
TTL:    24h, set on job creation, refreshed on every counter update.

Key:   wz-extractor:job:{jobId}:units                 Type: HASH
Fields: <wzFileBaseName> -> JSON {
  status      string ∈ {pending, running, succeeded, failed}
  startedAt   string (RFC3339|null)
  completedAt string (RFC3339|null)
  error       string|null
}
TTL:    same as parent job key.

Key:   wz-extractor:tenant-lock:{tenantId}:{region}:{maj}.{min}   Type: STRING
Value: jobId (so the holder is identifiable for debugging)
TTL:   60 minutes; auto-refreshed by the dispatcher.
```

`tenantId` is **not** prefixed onto the job key. The job key is keyed by jobId (UUIDv4) so the public GET endpoint can look up by jobId without knowing the tenant. Multi-tenancy is enforced by the tenant header on `POST` (we trust the header, same as the rest of the service today) and by the lock key, which **is** tenant-scoped. The job hash records `tenantId` as a field so an inspector can see "whose job is this".

Why not include tenantId in the job key? Because the GET endpoint receives only `{jobId}` in the URL and we don't want to require the tenant header on a status read (operators copy-paste jobIds from logs). A jobId is a UUIDv4 — non-guessable, and the same authz model (header-trust) protects everything else, so this is consistent.

### 4.8 Dispatcher empty-input behavior — 400 (PRD-decided, restated)

PRD §4.5 step 3: empty file list returns `400 Bad Request`. Today's behavior was `202` followed by an async error log. This is a behavior change that operators may not expect; we mitigate by making the error message explicit ("no .wz files staged for tenant X under path Y; upload via PATCH /api/wz/input first") so the operator's first read of the response tells them exactly what to do. No deprecation period — the previous behavior was a regression, not a feature.

### 4.9 wipeCharacterCache — runs once, on dispatcher, before publishing (decided)

PRD §4.2 says it must run before any unit starts and §4.9 says it is **not** re-run on redelivery. Implementation: dispatcher calls `wipeCharacterCache(imgOutPath)` immediately after acquiring the lock, before creating the job record. If wipe fails, log a warning (today's behavior) and proceed.

This isn't put in a "preflight unit" Kafka message because that would force a sequencing constraint between preflight and other units (preflight must complete before any other unit runs). Cross-partition ordering isn't free in Kafka. Doing it synchronously on the dispatcher is simpler and matches today's "wipe runs once before processing" invariant.

**Cost.** If the dispatcher pod and the consumer pods are on different filesystems, wipe runs on the dispatcher's mount. Since both use the same RWX `atlas-assets-pvc`, this is a non-issue. Documented in `docs/storage.md`.

### 4.10 Within-pod `WZ_EXTRACT_PARALLELISM` env var meaning

In Option A, `WZ_EXTRACT_PARALLELISM` no longer controls a worker-pool size at runtime — the consumer is synchronous, parallelism comes from partition assignment. We re-purpose the env var to two consistent meanings:

1. **Topic provisioning hint.** The deployment manifest's topic creation specifies partitions ≥ `WZ_EXTRACT_PARALLELISM`. (Atlas's topic provisioning is documented per-service; we do the same.)
2. **In-process pool size for the legacy `Extract` (whole-list) path.** Used by tests and by `pool.go` when the dispatcher path is bypassed.

This avoids env-var bloat. `runtime.NumCPU()` remains the default. Invalid values fall back to default with a warning, matching `WZ_EXTRACT_MAX_MAP_PIXELS` precedent.

---

## 5. Detailed component design

### 5.1 `extraction.Processor` interface and refactor

```go
type Processor interface {
    // Extract preserves today's call-site (whole-list, in-process). Used by
    // tests and by any caller that wants in-process fallback. Internally now
    // backed by a bounded worker pool over ExtractUnit.
    Extract(l logrus.FieldLogger, ctx context.Context, xmlOnly, imagesOnly bool) error

    // ExtractUnit is the per-WZ-file worker. Idempotent on output paths
    // (overwrites). Returns an error only when wz.Open fails (the
    // "couldn't even open the file" case from PRD §4.6 step 2).
    ExtractUnit(l logrus.FieldLogger, ctx context.Context, wzFile string, xmlOnly, imagesOnly bool) error
}
```

Behavior contract for `ExtractUnit`:

- Returns **nil** if wz.Open succeeded and the per-stage logic ran to completion (logging stage errors as today). Even if a stage like XML serialization fails for a particular property, the unit succeeded. This matches today's continue-on-error.
- Returns **non-nil** only when the file cannot be opened. Under Option A this becomes "unit failed" in Redis.
- `ctx` carries the tenant. The consumer rebuilds tenant context from Kafka headers via `consumer.TenantHeaderParser`, then calls `ExtractUnit`. The dispatcher passes its own request-derived context.

`Extract` becomes a thin loop using `pool.go`:
1. List `*.wz` files.
2. Run `wipeCharacterCache` (only when `!xmlOnly`).
3. Submit each path to the bounded pool, which calls `ExtractUnit`.
4. Aggregate errors (today's "unable to open" paths surface as warnings).

This preserves today's call-site for any in-process test and gives a clean within-pod fan-out per PRD §4.2.

### 5.2 `dispatcher.go` orchestration

```go
func (d *dispatcher) handleExtract(...) http.HandlerFunc {
    // 1. Parse tenant + flags from headers/query (today's logic).
    // 2. Glob WZ files. If empty → 400.
    // 3. Acquire Redis tenant lock (NX). On false → 409.
    //    On Redis error → release nothing, return 503.
    // 4. wipeCharacterCache (warn on error, continue).
    // 5. Generate jobId (UUIDv4). Create job record + N unit records
    //    (status=pending). Set job status=running.
    //    On Redis error → release lock, return 500.
    // 6. Publish N Kafka messages (one per WZ file).
    //    On producer error after K messages: leave the K already-published
    //    units to their fate, set job status=failed with an error note,
    //    release lock, return 500.
    // 7. Spawn refresh goroutine (§4.4).
    // 8. Return 202 with {jobId, unitsTotal, status:"running"}.
}
```

The dispatcher is `Logger + ctx + httpResponseWriter + processor + jobStore + tenantLock + producer`. All deps injected via the existing `InitResource` plumbing. No goroutine survives the request beyond the refresh goroutine, and that one is bounded by job lifetime.

**Why not a saga.** A saga (`libs/atlas-saga`) would be appropriate if any of these steps had a meaningful undo (publish a compensating message, etc.). Here:

- Step 4 wipe is irreversible by design (cache rebuilds itself).
- Step 5 Redis writes are reversed only by deleting the job key, which we do anyway via TTL or explicit delete on early failure.
- Step 6 Kafka publishes are not undoable (consumers may already be processing).

A saga's overhead doesn't pay back here; a linear sequence with explicit error branches is clearer.

### 5.3 Consumer

`kafka/consumer/extraction/` mirrors `services/atlas-data/atlas.com/data/kafka/consumer/data/`:

```go
const (
    EnvCommandTopic              = "COMMAND_TOPIC_WZ_EXTRACTION"
    CommandStartExtractionUnit   = "START_EXTRACTION_UNIT"
)

type command[E any] struct {
    Type string `json:"type"`
    Body E      `json:"body"`
}

type startExtractionUnitBody struct {
    JobId      string `json:"jobId"`
    WzFile     string `json:"wzFile"`
    XmlOnly    bool   `json:"xmlOnly"`
    ImagesOnly bool   `json:"imagesOnly"`
}
```

`InitConsumers` registers with `consumer.SpanHeaderParser` + `consumer.TenantHeaderParser` and `kafka.LastOffset` (parity with atlas-data). `InitHandlers` adapts via `message.AdaptHandler(message.PersistentConfig(handleStartExtractionUnit(p, store, lock)))`.

```go
func handleStartExtractionUnit(p extraction.Processor, store job.Store, lock lock.TenantLock) message.Handler[command[startExtractionUnitBody]] {
    return func(l logrus.FieldLogger, ctx context.Context, c command[startExtractionUnitBody]) {
        if c.Type != CommandStartExtractionUnit { return }

        l = l.WithFields(logrus.Fields{"jobId": c.Body.JobId, "wzFile": c.Body.WzFile})

        // (a) Idempotent transition pending → running.
        if !store.MarkUnitRunning(ctx, c.Body.JobId, c.Body.WzFile) {
            l.Info("unit already terminal; skipping (redelivery)")
            return
        }

        // (b) Run the work.
        runErr := p.ExtractUnit(l, ctx, c.Body.WzFile, c.Body.XmlOnly, c.Body.ImagesOnly)

        // (c) Atomic terminal transition + counter increment (§4.6).
        finalState := job.UnitSucceeded
        if runErr != nil { finalState = job.UnitFailed }
        finalized := store.FinalizeUnit(ctx, c.Body.JobId, c.Body.WzFile, finalState, runErr)

        // (d) "Last one home" → declare job terminal status, release lock.
        if finalized.AllDone {
            terminal := job.JobCompleted
            switch {
            case finalized.UnitsFailed == finalized.UnitsTotal: terminal = job.JobFailed
            case finalized.UnitsFailed > 0:                     terminal = job.JobCompletedWithErrors
            }
            if store.MarkJobTerminal(ctx, c.Body.JobId, terminal) {
                _ = lock.Release(ctx, finalized.LockKey)
                l.WithField("status", terminal).Info("job finalized")
            }
        }
    }
}
```

`MarkUnitRunning`, `FinalizeUnit`, and `MarkJobTerminal` are the only methods that touch Redis transactions; all WATCH/MULTI/EXEC stays in `job/store.go` so the handler is testable against a fake `job.Store`.

### 5.4 `job.Store` interface

```go
type Store interface {
    Create(ctx context.Context, job Job) error
    Get(ctx context.Context, jobId string) (Job, []Unit, error)   // for GET endpoint
    MarkJobRunning(ctx context.Context, jobId string) error
    MarkUnitRunning(ctx context.Context, jobId, wzFile string) (claimed bool, err error)
    FinalizeUnit(ctx context.Context, jobId, wzFile string, terminal UnitStatus, runErr error) (Counters, error)
    MarkJobTerminal(ctx context.Context, jobId string, terminal JobStatus) (claimed bool, err error)
    Delete(ctx context.Context, jobId string) error                // tests + admin tools
}

type Counters struct {
    UnitsTotal, UnitsCompleted, UnitsFailed int
    AllDone                                  bool
    LockKey                                  string  // composed from tenant fields stored on the job
}
```

Keeping `LockKey` on `Counters` is a deliberate convenience: the consumer doesn't need a separate Redis read to figure out which tenant lock to release. The lock key is composed from the tenant fields written into the job hash at creation time, so it's a pure function of `Get(jobId)`. The store returns it inline to save a round-trip on the hot path.

`Job` and `Unit` follow atlas immutable-model conventions (private fields + getters + Builder). Builders go in `model.go`.

### 5.5 `lock.TenantLock` wrapper

Wraps `libs/atlas-redis.Lock` with:

- `Acquire(ctx, key, jobId) (bool, error)` — `SET NX EX` with `value=jobId` so debugging tools can see which job holds it.
- `StartRefresh(ctx, key) (cancel func())` — spawns a goroutine that calls `client.Expire(key, lockTTL)` every `lockTTL/3` until `cancel()` or `ctx.Done()`.
- `Release(ctx, key)` — `DEL`. (We don't gate release on owner-match; if the lock has expired and a second job re-acquired it, our `DEL` would unlock the second job. To avoid that we use Redis Lua compare-and-delete: `if GET == jobId then DEL`. This is the standard "Redis Distributed Lock" pattern. Implemented via go-redis `Eval`.)

### 5.6 `GET /api/wz/extractions/jobs/{jobId}` handler

Reads `Job` + units from `job.Store.Get`. JSON:API resource type `wzExtractionJob` per PRD §5.2. Returns 404 on `redis.Nil` from the underlying lookup. No tenant check on read (UUIDv4 jobId is the bearer token; consistent with the rest of the service's authz model).

The endpoint is mounted on the existing `/wz/extractions` subrouter:

```go
ext.HandleFunc("/jobs/{jobId}", register("get_extraction_job", handleJobStatus(...))).Methods(http.MethodGet)
```

### 5.7 main.go wiring

```go
rc := atlas.Connect(l)              // libs/atlas-redis Connect
defer rc.Close()
js := job.NewStore(rc, "wz-extractor")
tl := lock.NewTenantLock(rc, "wz-extractor", 60*time.Minute)
p  := extraction.NewProcessor(inputDir, outputXmlDir, outputImgDir)

// Producer manager teardown (parity with atlas-data).
tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

// Consumer registration.
cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
extconsumer.InitConsumers(l)(cmf)("wz-extractor-extraction")
if err := extconsumer.InitHandlers(l)(p, js, tl)(consumer.GetManager().RegisterHandler); err != nil {
    l.WithError(err).Fatal("Unable to register kafka handlers.")
}

// REST.
server.New(l).
    AddRouteInitializer(extraction.InitResource(p, js, tl, tdm.WaitGroup(), extraction.Dirs{...})(GetServer())).
    ...
```

Existing `wg *sync.WaitGroup` parameter on `InitResource` stays — it's used today to track the in-flight async extraction goroutine. Under the new model the dispatcher is synchronous (the unit work is on Kafka, not on a goroutine), so the wg use-count drops to zero. We leave the parameter in place for API compatibility but it becomes effectively a no-op. The plan documents this as removable in a follow-up.

---

## 6. Failure-mode traversal

| # | Scenario | Expected behavior | Mechanism |
|---|---|---|---|
| 1 | Two `POST` for same tenant arrive same instant. | One returns 202, other returns 409. | Redis NX on tenant lock. Verified by integration test against miniredis. |
| 2 | Dispatcher publishes K of N messages, then producer errors. | Job marked `failed` with error note. Lock released. The K already-published units run (their work is wasted but harmless — outputs overwrite). | Dispatcher catches producer error, calls `MarkJobTerminal(failed)`, releases lock. The K units that already ran will see `MarkUnitRunning → claimed=false` because we set the unit JSON to `{status: "skipped"}` in the same MarkJobTerminal transaction. (Pseudo-code in §4.6.) |
| 3 | Consumer pod crashes mid-`ExtractUnit`. | Unit is redelivered to another pod. | Kafka at-least-once. Consumer manager commits offset only after handler returns. |
| 4 | Unit message redelivered after the unit had already finalized as succeeded. | Second consumer logs "unit already terminal" and discards. Counters not double-incremented. | `MarkUnitRunning` returns `claimed=false` if current status is terminal (§4.6). Consumer returns early. |
| 5 | Two consumers race to call `FinalizeUnit` for the same wzFile (same redelivery scenario, but both reach `FinalizeUnit` at once). | One commits, the other's WATCH detects concurrent change and retries; on retry it sees terminal status and discards. | WATCH/MULTI/EXEC retry-on-nil. |
| 6 | Two consumers race to be "last one home" (both observe `unitsCompleted + unitsFailed == unitsTotal` after their own increment). | Exactly one's `MarkJobTerminal` succeeds. The other's CAS fails. Lock released exactly once. | `SET status NX` (only-if-running) gates the terminal write; `Release` is idempotent (DEL of an already-deleted key is no-op). |
| 7 | Dispatcher pod dies between lock acquire and message publish. | Lock TTL expires after 60min; tenant becomes re-extractable. Job record has `status=pending`, no unit messages were sent. Eventually 24h TTL reaps the orphan job record. | TTL on both lock and job record. Operator can also DELETE the job key manually. |
| 8 | Dispatcher pod dies after publish but before refresh-goroutine starts. | Refresh goroutine never runs; lock TTL is the original 60min. Within 60min the in-flight units finalize on the consumer pods (which are independent of the dispatcher). | Consumer-side finalize is independent of dispatcher liveness. Worst case: lock expires before final unit finishes; another `POST` admitted; outputs overwrite idempotently; first-job's terminal status is still computed correctly when its last unit lands. |
| 9 | Map.wz unit takes 90 minutes (>lockTTL). | Refresh goroutine extends lock every 20min; lock survives. | §4.4. |
| 10 | Redis becomes unreachable mid-job. | Refresh fails → eventually lock expires. Consumers' Redis writes fail; consumer logs error; Kafka offset is **not** committed (handler returns error), so unit redelivers when Redis is back. | Consumer handler returns error on Redis failure; manager re-delivers per persistent config. |
| 11 | `GET /jobs/{jobId}` for unknown jobId. | 404. | `redis.Nil` mapped to 404. |
| 12 | `POST` while no `*.wz` files exist for tenant. | 400 with explanatory message. | PRD §4.5 step 3. |

---

## 7. Behavior changes operators will notice

Listed up-front so the rollout note in the eventual PR is comprehensive:

1. **`POST` response shape** changes from `{"status":"started"}` to `{"jobId":"…","unitsTotal":N,"status":"running"}`. Existing callers that only check the 202 status code keep working. A repo-wide grep for callers of `POST /api/wz/extractions` is in the plan.
2. **`POST` empty-input** changes from 202+async-error-log to **400**.
3. **`POST` while another extraction in flight** changes from blocking-then-running-sequentially to **409 Conflict**. (Today's in-process mutex serialized; under Redis lock we reject.)
4. **New endpoint** `GET /api/wz/extractions/jobs/{jobId}`.
5. **New env vars required at startup**: `REDIS_URL`, `COMMAND_TOPIC_WZ_EXTRACTION`. Service refuses to start without them. Optional: `WZ_EXTRACT_PARALLELISM`, `REDIS_PASSWORD`.
6. **New Kafka topic** must exist before the service is healthy. Same posture as atlas-data today.
7. **New Redis dependency** — confirms intra-cluster reachability.
8. **Multi-pod deploy** is now supported; initial rollout still `replicas: 1`.

Of these, items 2 and 3 are the only ones a script could regress on. The plan must include grepping for repo-internal callers and updating them if they exist.

---

## 8. Testing strategy

Plan-level (`plan.md` will enumerate; here we set the categories):

- **Unit tests**:
  - `pool.go` bounded fan-out (within-pod parallelism, error propagation).
  - `job.Store` with `miniredis`: Create, Get, MarkUnitRunning idempotency, FinalizeUnit transactional retry, MarkJobTerminal CAS.
  - `lock.TenantLock` with `miniredis`: Acquire NX, refresh extends TTL, Release uses owner-match Lua.
  - `dispatcher.go`: empty input → 400; lock conflict → 409; Redis down → 503; Kafka producer error → 500 with job marked failed.
  - Consumer handler: redelivery returns early; failed `ExtractUnit` increments unitsFailed; last-one-home transitions job correctly.

- **Integration tests**:
  - End-to-end with embedded Kafka (`kafka-go` test harness or testcontainers) + `miniredis`: dispatch → consume → finalize. Run with `partitions=4` and 12 mock units to exercise partition-balanced consumption inside one pod.
  - Two concurrent dispatchers same tenant → exactly one 202.
  - Crash simulation: kill the consumer goroutine mid-handler before commit; reissue the message; verify counter sanity.

- **Manual smoke** (per PRD §8 NFR + §10 acceptance):
  - `replicas=1`, `WZ_EXTRACT_PARALLELISM=NumCPU`: full extraction wall-clock vs. baseline.
  - `replicas=3`: full extraction wall-clock vs. `replicas=1`.
  - Numbers go in plan.md, not here, since they depend on cluster sizing.

The `Test Helper Pattern` rule from CLAUDE.md applies: use the project Builder pattern; no `*_testhelpers.go`. `job.Job` and `job.Unit` get builders in `model.go`.

---

## 9. Observability

Per PRD §4.10 and §8 Observability:

- Every per-unit log line picks up `jobId` and `wzFile` structured fields via `l.WithFields(...)` in the consumer handler before any work runs. This wraps today's logs (`"Processing [Mob.wz]"`, `"map rendered"`, etc.) without needing per-line edits inside `ExtractUnit`.
- New OTel spans:
  - `wz.extraction.dispatch` — root span on the dispatcher; attributes `jobId`, `tenantId`, `unitsTotal`.
  - `wz.extraction.unit` — per-unit span on the consumer; attributes `jobId`, `wzFile`. Parents from the Kafka span context (`SpanHeaderParser` already propagates).
  - `wz.extraction.unit.<stage>` (xml, icons, minimaps, mapRender) — child spans inside `ExtractUnit`. Optional; recommended.
- Job state itself is also observable via `GET /jobs/{jobId}` (operator-facing) and Redis directly (debugger-facing).

---

## 10. Rollout plan

1. Land code on `task-062-wz-extractor-parallelism` branch with `replicas: 1` in the manifest. Topic is provisioned with `partitions=16`.
2. Verify on a staging tenant: full extraction completes, status endpoint reflects state, lock conflicts behave.
3. Measure single-pod wall-clock vs. baseline (PRD §8 NFR target).
4. Change manifest to `replicas: 3` in a follow-up PR (or the same PR, behind a feature flag of the manifest itself — left to the plan). Re-measure.
5. Decide on HPA in a separate task. Burst durations from steps 3–4 inform whether HPA-on-CPU is useful or whether extraction is too short-lived.

CPU/memory `requests`/`limits` are written in the manifest based on step-3 measurements; the plan keeps them as `requests: { cpu: 1, memory: 2Gi }` / `limits: { cpu: 4, memory: 8Gi }` from the PRD as a starting point and tightens after measurement.

---

## 11. Open items handed to plan.md

- Concrete partition count: PRD says `≥ 16, default 16`. Plan to confirm whether topic provisioning lives in `deploy/k8s/atlas-wz-extractor.yaml` or in a Kafka-topic CR/manifest used by the rest of the project.
- Concrete CPU/memory `requests`/`limits` after step-3 measurement.
- Whether to remove the now-unused `*sync.WaitGroup` parameter from `extraction.InitResource` immediately or in a follow-up (favors follow-up — keep this PR focused).
- Whether to add a small admin endpoint `DELETE /jobs/{jobId}` for clearing stuck job records (operationally useful but out of PRD scope; flag for explicit accept/reject in plan).
- Whether to grep for callers of `POST /api/wz/extractions` and update them; PRD §7 says verify, plan task says do it.

---

## 12. What this design does NOT do

Listed for the reviewer who looks for over-reach:

- No multi-tenant fairness, queueing, or cancellation.
- No streaming of a single image's parse across goroutines (PRD non-goal).
- No `atlas-data` import-pipeline changes.
- No change to the zip-upload flow (`PATCH /api/wz/input`).
- No new shared library. Everything new is service-local under `services/atlas-wz-extractor/atlas.com/wz-extractor/`.
- No new `libs/atlas-*` directory. Per memory rule, the existing `libs/atlas-redis` and `libs/atlas-kafka` cover what we need.
