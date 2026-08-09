# Ingest Progress Visibility — Design

Version: v1
Status: Approved for planning
Created: 2026-08-08
PRD: [`prd.md`](prd.md)
---

## 1. Scope of this document

The PRD fixes *what* is built. This document fixes *how*: the ownership split
between the two pods, the shape of the record they share, the three
pre-existing defects that this feature would otherwise silently inherit, and
the alternatives rejected along the way. It also settles the three open
questions in PRD §9.

Everything below is grounded in the code as it exists on this branch;
`file:line` references are to the worktree at design time.

## 2. Findings that shape the design

Four facts about the current code determine most of the design. Three of them
are defects that the feature would inherit or trip over; they are in scope
because the feature cannot be correct without fixing them.

### F-1 `Registry.Update` silently drops the key's TTL — blocking

`libs/atlas-redis/registry.go:100` writes the updated value with
`pipe.Set(ctx, rk, newData, 0)`. In Redis, `SET` without an expiry option
**clears** any existing TTL. So the sequence the PRD mandates —
`PutWithTTL(7d)` at init (FR-3.2), then `Update` per worker transition
(FR-2.3) — produces a key that loses its TTL on the very first worker
transition and then lives forever. §6 of the PRD ("The 7-day TTL is refreshed
on every write") is not achievable with the registry as written.

The escape hatch of calling `EXPIRE` on the raw client is closed:
`tools/redis-key-guard.sh` bans keyed go-redis commands outside
`libs/atlas-redis`.

**Decision:** add `Registry.UpdateWithTTL(ctx, key, ttl, fn)` to
`libs/atlas-redis/registry.go` — an exact copy of `Update` whose pipelined
`Set` passes `ttl` instead of `0` — plus unit tests (the existing
`registry_test.go` already exercises `Update` against miniredis, so the TTL
assertion has a home). `Update` itself is left alone: changing its semantics
would touch every existing caller, including live monster-store state.

This is the only change outside `services/atlas-data` and `services/atlas-ui`.
`libs/atlas-redis` is an existing lib, already in `go.work` and the root
`Dockerfile`, so no registration lists change and
`tools/service-registration-guard.sh` is unaffected.

### F-2 The Watchdog reads a heartbeat key that tenant-scope runs never write — blocking

`Create` writes the heartbeat under
`ingestJobKeySuffix(scope, …)` = `tenants/<uuid>:GMS:83.1`
(`runtime/rest/jobs.go:207`, raw scope). The Watchdog reconstructs the same
suffix from the Job's *labels* via `ingestJobKeySuffixFromLabels`
(`jobs.go:55`), which reads `j.Labels["scope"]` — and that label was written as
`sanitizeLabel(scope)` (`jobs.go:233`), which maps `/` to `-`, yielding
`tenants-<uuid>:GMS:83.1`.

The two strings differ for every tenant-scope run. `jobIsStuck`
(`watchdog.go:84`) therefore never finds a heartbeat for a tenant ingest and
falls back to `j.CreationTimestamp` — i.e. for tenant runs the Watchdog is
still the creation-time cutoff the heartbeat was added to replace. Shared-scope
runs are unaffected (`sanitizeLabel("shared") == "shared"`).

This matters here because FR-3.1 makes the Watchdog a *writer* of the run
record. Left unfixed, it would write phase `stuck` to a key nothing reads,
while the record the UI polls stayed `running` forever.

**Decision:** fix `ingestJobKeySuffixFromLabels` to reconstruct the raw scope:
`shared` when `labels["scope"] == "shared"`, otherwise
`"tenants/" + labels["tenant"]` (the `tenant` label is stored raw at
`jobs.go:278`, unsanitized). Return `""` when the tenant label is missing on a
non-shared Job, preserving today's "skip" contract. A regression test asserts
`ingestJobKeySuffixFromLabels(renderJob(...))` round-trips to
`ingestJobKeySuffix(...)` for both scopes — the assertion that was missing.

Fixing this also repairs the tenant-scope heartbeat check as a side effect.
That is a behaviour change to the Watchdog (tenant runs now honour the
heartbeat instead of the creation timestamp), and it is strictly the intended
behaviour of the existing `TimeoutSecs: 7200` design (`main.go:115`).

### F-3 The run-record registry cannot hang off `JobCreator`

FR-4.5 requires `GET /api/data/process` to serve the Redis record even when the
Kubernetes client is unavailable. But `JobCreator.Registry` only exists when
`NewJobCreatorInClusterWithRedis` succeeds; on failure `main.go:96` sets
`jc = nil` and the registry goes with it. `rdb` is also currently scoped inside
the `MODE == "rest"` block (`main.go:91`).

**Decision:** hoist `rdb` in `main.go` to the enclosing scope, construct the run
registry from it independently of the `JobCreator`, and change the signature to
`restruntime.InitResource(jc, runReg)`. The handler's two dependencies become
independently nil-able: no `jc` degrades the live-Job cross-check (FR-4.3/4.5),
no `runReg` degrades to `phase: "none"`.

### F-4 `data` cannot import `runtime/ingest`

`runtime/ingest` imports `atlas-data/data` (`runtime/ingest/run.go:4`). The
per-worker instrumentation point is `runOne` inside `data.RunWorkers`
(`data/runwz.go:60`), so progress reporting must enter the `data` package
through an interface the `data` package itself declares, with the concrete
Redis-backed implementation living in `runtime/ingest`. See §4.2.

## 3. Architecture

Three writers and one reader share one Redis key per
`(scope, region, major, minor)`.

```
POST /api/data/process ──> JobCreator.Create
   (REST pod)               ├─ renders Job (+ INGEST_RUN_ID env)          ┐
                            ├─ PutWithTTL <suffix>            (job name)  │
                            ├─ PutWithTTL <suffix>:updatedAt  (heartbeat) │ FR-3.2
                            └─ PutWithTTL <suffix>:run        (record,    │
                                              phase=running, all pending) ┘

Job pod (MODE=ingest) ──> ingest.Run
                            ├─ heartbeat goroutine (unchanged)
                            ├─ progress.Init      → adopt/reset record   FR-2.1
                            ├─ RunWorkers(… sink) → per-worker Update    FR-2.2
                            └─ progress.Finish    → succeeded | failed   FR-2.4

Watchdog (REST pod)   ──> deleteStuckJob
                            ├─ delete Job
                            ├─ Remove <suffix>, <suffix>:updatedAt (as today)
                            └─ Update <suffix>:run → phase=stuck, reason FR-3.1

GET /api/data/process ──> processStatus
   (REST pod)               ├─ ResolveScope (shared with POST)           FR-4.1
                            ├─ Get <suffix>:run  ──┐
                            ├─ Get <suffix>:updatedAt (heartbeat) ├ merge FR-4.3
                            ├─ List Jobs (label-selected to triple)┘
                            └─ JSON:API ingestRun                        FR-4.6
```

Redis is the system of record for progress; Kubernetes is a corroborating
second opinion used only to demote a stale `running` to `unknown`. That
direction is deliberate: the Job object is the thing that disappears
(`ttlSecondsAfterFinished: 3600`, Watchdog deletion), so it cannot be the
source of truth for a feature whose whole point is surviving its disappearance.

### 3.1 Run identity and the superseded-pod hazard

Nothing today tells an ingest pod which run it *is*. If an operator triggers a
second ingest for the same triple while the first pod is still alive, both pods
would write to the same key: the old pod would keep flipping workers to
`succeeded` inside the new run's record, and its eventual `Finish` would stamp
a terminal phase over a run that is still going.

**Decision:** `Create` generates a run id and injects it into the rendered Job
as env `INGEST_RUN_ID` (alongside the `SCOPE`/`REGION`/… vars at
`jobs.go:237`). Every write from the ingest pod is guarded: read the record,
and if `record.runId != os.Getenv("INGEST_RUN_ID")`, drop the write. The guard
lives inside the `UpdateWithTTL` mutator function, so it is evaluated against
the freshly-read value on every optimistic-lock retry — a stale pod can never
win the race.

`runId` is a UUID (`google/uuid` is already an atlas-data dependency). The
Job name is *not* reused as the identity: the ingest pod has no supported way
to learn its own Job name without a downward-API field ref, and `renderJob`
already knows the id before the Create call.

Consequence for FR-2.6: when `INGEST_RUN_ID` is unset (compose / unit-test
path, the same condition that already disables the heartbeat via
`ingestJobSuffixFromEnv`), progress publication is skipped silently. One gate,
both signals.

## 4. Components

### 4.1 The record type — `runtime/ingest/…` or shared?

The record is written by `runtime/rest` (create, watchdog), written by
`runtime/ingest` (worker transitions), and read by `runtime/rest` (status
handler). It cannot live in either package without one importing the other.

**Decision:** a new leaf package `services/atlas-data/atlas.com/data/ingestrun`
holding the record type, the phase/state enums, the key-suffix helper, and the
registry constructor. It imports only `libs/atlas-redis` and stdlib, so both
runtime packages and the REST model can depend on it with no cycle.

The two existing duplicated helpers (`ingestJobNamespace` and
`newIngestJobRegistry`, copy-pasted between `runtime/rest/jobs.go:38` and
`runtime/ingest/heartbeat.go:27`) move into this package as the single
definition, with both runtime packages importing it. That is a targeted
de-duplication of code this feature touches, not an unrelated refactor — and it
is what makes F-2's round-trip test expressible.

```go
package ingestrun

type Phase string   // none | running | succeeded | failed | stuck | unknown
type WorkerState string // pending | running | succeeded | failed | skipped

type WorkerEntry struct {
    Name       string     `json:"name"`
    State      WorkerState`json:"state"`
    StartedAt  *time.Time `json:"startedAt,omitempty"`
    FinishedAt *time.Time `json:"finishedAt,omitempty"`
    Error      string     `json:"error,omitempty"`
}

type Record struct {
    RunId      string        `json:"runId"`
    JobName    string        `json:"jobName"`
    Scope      string        `json:"scope"`
    Region     string        `json:"region"`
    Version    string        `json:"version"`   // "<major>.<minor>"
    Tenant     string        `json:"tenant,omitempty"`
    Phase      Phase         `json:"phase"`
    StartedAt  time.Time     `json:"startedAt"`
    FinishedAt *time.Time    `json:"finishedAt,omitempty"`
    Reason     string        `json:"reason,omitempty"`
    Workers    []WorkerEntry `json:"workers"`
}
```

`Record` is a plain serialisable struct, not an immutable model with a Builder.
The project's immutable-model + Builder convention (CLAUDE.md, DOM checklist)
governs *domain* models flowing through processors; this is a wire/storage DTO
for a Redis JSON blob mutated inside an optimistic-lock closure, in the same
category as the existing `processStatusJob` and `StatusRestModel` shapes.
Applying a Builder here would add a copy step inside a hot retry loop for no
invariant. Mutation is nonetheless confined: all transitions go through named
methods on `Record` (`WithWorkerRunning`, `WithWorkerTerminal`, `WithPhase`)
that return a modified copy, so the `UpdateWithTTL` mutator stays a pure
function of its input and is safe to re-run on retry.

The worker roster comes from `workers.Registered` (FR-1.6) — but `ingestrun`
must not import `data/workers` (that would drag the WZ/gorm/minio dependency
graph into the REST path). The roster is passed *in* as a `[]string` by each
caller; `runtime/rest` obtains it from a tiny exported accessor
`workers.RegisteredNames()`, which is a pure `[]string` helper with no new
dependency edge for `ingestrun` itself.

### 4.2 The progress sink — how instrumentation reaches `runOne`

Per F-4, `data` declares the interface:

```go
// data/progress.go
type ProgressSink interface {
    WorkerStarted(ctx context.Context, name string)
    WorkerFinished(ctx context.Context, name string, err error, skipped bool)
}
```

Wiring options considered:

- **(a) A field on `workers.Params`.** Rejected: `Params` is a value struct
  passed to every worker; adding a behavioural dependency to it widens the
  worker contract for something no worker uses.
- **(b) A new parameter on `RunWorkers(l, db, mc, sink)`.** Rejected: forces
  every existing caller (including `data/runwz_test.go`) to thread a nil.
- **(c) A variadic option: `RunWorkers(l, db, mc, opts ...RunOption)` with
  `WithProgress(sink)`.** **Chosen.** Existing call sites compile unchanged, the
  test path defaults to a no-op sink, and there is exactly one place
  (`runOne`) that consults it.

`runOne` becomes:

```go
sink.WorkerStarted(tctx, w.Name())
… OpenArchive …
  ErrCategoryAbsent → sink.WorkerFinished(tctx, w.Name(), nil, true); return nil
  other error       → sink.WorkerFinished(tctx, w.Name(), err, false); return err
err := w.Run(...)
sink.WorkerFinished(tctx, w.Name(), err, false)
return err
```

This single chokepoint covers both the sequential prerequisite loop and the
parallel fan-out (FR-2.2) because both already call `runOne`. The default sink
is a no-op struct, so `runwz_test.go` and the compose path need no changes and
FR-2.6 falls out for free.

`ProgressSink` also becomes the natural home for the FR-8 observability
requirement: the Redis-backed implementation logs
`worker=<name> state=<state> duration=<d>` at info on every `WorkerFinished`,
so an operator without the UI gets the same information from pod logs.

### 4.3 Context discipline for progress writes

Two hazards, both handled inside the Redis sink implementation rather than at
call sites:

1. **Cancelled context at terminal write.** When a worker fails, the errgroup
   cancels `gctx`; the enclosing `Finish(failed)` write would then run under a
   dead context and always fail — defeating FR-2.4 exactly when it matters
   most. Every write therefore runs under
   `context.WithoutCancel(ctx)` wrapped in `context.WithTimeout(…, 5s)`.
2. **A wedged Redis stalling ingest.** The 5s ceiling bounds the worst case to
   ~22 × 5s across a multi-minute run, and the write result is discarded after a
   warn-level log (FR-2.5). An ingest that would have succeeded cannot fail
   because Redis did.

No new goroutines: writes are synchronous at the transition points, so
`tools/goroutine-guard.sh` is satisfied without an allow comment.

### 4.4 The read handler

`processStatus` is rewritten as:

1. `wzinput.ResolveScope(r, t)` — the existing shared resolver
   (`wzinput/scope.go:21`), which already implements exactly the FR-4.1 rules.
   `processCreate` is switched to it in the same change, deleting its inline
   duplicate at `runtime/rest/resource.go:39-51`. This is the NFR's "must reuse
   the same code path" requirement, satisfied by adopting the resolver that
   already exists rather than inventing a third copy. (`data/status.go:120` has
   a fourth copy that resolves to a *tenant id* rather than a scope key; it is
   left alone — different return type, different consumer, out of scope.)
   Error mapping: `ErrSharedRequiresOperator` → 403, anything else → 400.
2. `Get(<suffix>:run)`. `ErrNotFound` (or a nil registry) → `phase: "none"`
   with an empty worker list and a zero `workersTotal` (FR-4.4).
3. If the record's phase is `running`, cross-check:
   - `Get(<suffix>:updatedAt)` — a heartbeat within the Watchdog's staleness
     window means the pod is alive; report `running` regardless of what
     Kubernetes says (this covers the window between Job creation and pod
     scheduling, and any Job-list hiccup).
   - Otherwise list Jobs with a label selector narrowed to the triple —
     `atlas-data-ingest=true,scope=<sanitizeLabel(scope)>,region=<region>,version=<major>.<minor>`
     — per the NFR's "filtered by label selector, not client-side" rule. Note
     the selector must use the *sanitized* scope, matching `renderJob`
     (`jobs.go:233`); the raw scope only ever appears in Redis keys. This is the
     same raw-vs-label asymmetry that produced F-2, so both sides get a test.
   - No live Job and no fresh heartbeat → `unknown` (FR-4.3). The stored record
     is *not* rewritten; `unknown` is computed at read time, so a record that
     later proves alive (a slow Job list, a restarted pod) recovers on the next
     poll without a repair path.
4. Terminal phases (`succeeded`, `failed`, `stuck`) are returned as stored, with
   no Kubernetes call at all. This is what makes the record readable after
   garbage collection.
5. Marshal via `jsonapi.Marshal` with a new `IngestRunRestModel`
   (`GetName() == "ingestRun"`, id `<scope>:<region>:<major>.<minor>`),
   following `StatusRestModel` (`data/status.go:20`) as the template.
   `workersTotal` / `workersComplete` are computed in the REST model, not
   stored — derived values have no business being persisted twice.
   `workersComplete` counts `succeeded` + `skipped` + `failed`, i.e. workers
   that will not change again.

`jc == nil` no longer short-circuits to 503 (FR-4.5); it only disables step 3's
Job list, which degrades a `running` record to `unknown` only when the
heartbeat is also absent. The 503 branch at `resource.go:93` is deleted.

### 4.5 UI

- `seed.service.ts` gains `getIngestRun(tenant)` and
  `getCanonicalIngestRun(sel)`, both thin wrappers over the existing
  `fetchJsonApi` helper with `tenantHeaders` / `canonicalHeaders`
  (`src/lib/headers.tsx`) — the same pairing every other status fetcher uses.
  `canonicalHeaders` already bakes in `X-Atlas-Operator: 1`, so FR-5.2's
  operator requirement needs no new plumbing.
- `useIngestRun()` in `useSeed.ts` and `useCanonicalIngestRun(sel)` in
  `useCanonicalData.ts`, both `refetchInterval: 5000, staleTime: 0`, matching
  the neighbouring hooks (`useSeed.ts:294`, `useCanonicalData.ts:49`).
  `retry: false` plus rendering from `query.isError` rather than a toast gives
  FR-5.7 — an unreachable endpoint yields a steady "progress unavailable" panel
  and a 5s retry cadence that never escalates.
- One shared presentational component,
  `src/components/features/setup/IngestProgressPanel.tsx`, taking
  `{ run, isError }` and owning no fetching. Both pages mount the same
  component with a different hook, so the two surfaces cannot drift.
- `BaselinesPage` extends `publishDisabled` (`BaselinesPage.tsx:153`) with
  `blockingPhases = ["running", "stuck", "failed", "unknown"]`. `none` and
  `succeeded` do not block (FR-5.6). The reason string is rendered next to the
  disabled button; the existing `SetupRow` `warning` slot
  (`SetupRow.tsx:8`) is the natural place for it.

## 5. Alternatives considered

**Per-worker Redis keys instead of one record.** Eleven keys removes write
contention entirely and needs no optimistic lock. Rejected: the read becomes a
SCAN or eleven GETs per poll per open page, the "run phase" has no home, and
partial-write states (six worker keys, no run key) become representable. The
contention it avoids is not real — eleven transitions over minutes at
`INGEST_MAX_PARALLEL=4` is nothing against `updateMaxRetries = 1000`
(`registry.go:74`).

**Kubernetes as the source of truth, Redis as a cache.** Rejected on the
PRD's own terms: the Job is deleted at `ttlSecondsAfterFinished: 3600` and by
the Watchdog, and the terminal outcome must outlive both. Job status also has
no concept of per-worker progress.

**Postgres run history.** Genuinely better for durability and would remove the
"lost record reports `none`" caveat. Explicitly a PRD non-goal, and it costs a
migration plus a new table on a service whose ingest path is already the
slowest thing in the system. The mitigation for the caveat is FR-5.6: `none`
never blocks publishing, so an evicted record degrades to today's behaviour
rather than wedging the control.

**Kafka progress events.** Rejected: no consumer wants a stream, atlas-data has
no ingest topic, and it would add a topic registration to a feature whose PRD
explicitly claims no deployment change.

**Sub-worker progress (maps ingested / total).** PRD non-goal. Worth noting the
design does not preclude it: adding `Processed`/`Total` to `WorkerEntry` and a
third sink method is additive, and the record's TTL/locking story is unchanged.

**SSE/WebSocket streaming.** PRD non-goal. The five-second poll matches every
other status surface on these two pages; a second transport for one panel is
not justified.

## 6. Resolved open questions (PRD §9)

**Q1 — `stuck` run's per-worker record.** Preserve it exactly as the ingest pod
left it. The worker that was `running` when the Watchdog fired stays `running`;
the Watchdog writes only the run-level `phase` and `reason`. Marking it
`failed` would assert something we do not know (the pod may have been wedged in
`OpenArchive`, or mid-write, or simply slow), and the diagnostic value of the
record *is* "MAP was the one still going." The UI renders a worker in state
`running` under a terminal run phase as "interrupted" — a presentation
concern, derived from the pair, requiring no extra stored state.

**Q2 — elapsed-time source.** The record's `startedAt`, stamped by the REST pod
at Job creation (FR-3.2), never overwritten by the ingest pod. Three reasons:
the Kubernetes `startTime` vanishes with the Job while the panel must keep
rendering a duration; one clock (the REST pod's) is used for both the running
and terminal cases, so a completed run's total does not silently change
provenance; and scheduler-pending time is time the operator waited, which is
what the question "how long has this been going?" means at the console.
Consequence: the ingest pod's `Init` (FR-2.1) *adopts* an existing record —
preserving `runId`, `jobName`, `startedAt` — and only seeds the worker roster
and confirms `phase: running`. It writes a fresh record with its own
`startedAt` only when none exists (a Redis eviction between create and pod
start).

**Q3 — the Setup page's document-count badge.** Keep it, with the new panel as
a separate row beneath. The two answer different questions — "how many rows
landed" versus "which workers have run" — and a completed run with a zero
document count is a real, diagnosable state that folding them together would
hide. Zero regression risk on a page that is already the operator's default
landing surface.

## 7. Error handling summary

| Condition | Behaviour |
|---|---|
| Redis write fails (any progress write) | warn log, ingest continues (FR-2.5) |
| Redis read fails on GET | standard error response via `WriteErrorResponse` |
| Record absent | 200, `phase: "none"` (FR-4.4) |
| Registry nil (no Redis) | 200, `phase: "none"` |
| `jc == nil` / k8s list error | serve the stored record; skip the cross-check (FR-4.5) |
| Record `running`, no Job, stale heartbeat | `phase: "unknown"` (FR-4.3), record untouched |
| `INGEST_RUN_ID` mismatch (superseded pod) | write dropped, debug log (§3.1) |
| `SCOPE`/`REGION` unset in ingest pod | progress skipped silently (FR-2.6) |
| `scope=shared` without operator header | 403 |
| Unknown `scope` value | 400 |

## 8. Testing strategy

Go (`services/atlas-data`, `go test -race ./...`):

- `libs/atlas-redis`: `UpdateWithTTL` sets the value **and** retains a TTL;
  `Update` still clears it (locking F-1 in place so a future refactor cannot
  silently undo the fix).
- `ingestrun`: record transitions (`WithWorkerRunning`, `WithWorkerTerminal`,
  `WithPhase`) including `skipped` not blocking `succeeded` (FR-1.5), and
  roster derivation from a passed name list (FR-1.6).
- `runtime/rest/jobs_test.go`: `ingestJobKeySuffixFromLabels(renderJob(…))`
  round-trips to `ingestJobKeySuffix(…)` for `shared` and `tenants/<uuid>`
  (F-2), and `renderJob` injects `INGEST_RUN_ID`.
- `runtime/rest/resource_test.go`: the FR-4 matrix — default scope, shared
  without/with operator, bogus scope, absent record → `none`, terminal record
  with a nil `jc`, `running` + no Job + stale heartbeat → `unknown`, and the
  label selector actually narrowing to the triple. `fake.NewSimpleClientset`
  plus miniredis, both already used in this package's tests
  (`watchdog_test.go`, `jobs_test.go`).
- `runtime/rest/watchdog_test.go`: `deleteStuckJob` writes `stuck` with a
  reason, still removes the two heartbeat keys, and leaves per-worker states
  untouched (Q1).
- `data/runwz_test.go`: a recording sink observes
  `pending → running → succeeded` per worker, `skipped` on `ErrCategoryAbsent`,
  and STRING terminal before any other worker starts (the FR ordering
  acceptance criterion) — assertable in-process without Redis.
- `runtime/ingest`: the run-id guard drops a stale pod's write; a failing
  registry does not propagate an error out of the sink (FR-2.5).

TypeScript (`services/atlas-ui`, `npm run build` — which type-checks tests):

- `IngestProgressPanel` renders each phase, surfaces `reason`/`error` for
  `failed`/`stuck` (FR-5.4), and shows "progress unavailable" on `isError`
  (FR-5.7).
- `publishDisabled` truth table across all six phases (FR-5.5/5.6).

## 9. Verification checklist

Per CLAUDE.md §Build & Verification, the affected modules are
`libs/atlas-redis`, `services/atlas-data`, `services/atlas-ui`:

1. `go test -race ./...` and `go vet ./...` clean in `libs/atlas-redis` and
   `services/atlas-data`.
2. `go build ./...` clean in `services/atlas-data`.
3. `docker buildx bake atlas-data` — required: `services/atlas-data/go.mod` is
   not expected to change, but `libs/atlas-redis` is a shared lib and the bake
   is the only check that catches a Dockerfile `COPY` gap. (No new lib is
   added, so no `Dockerfile`/`go.work` edit is expected — the bake proves it.)
4. `tools/redis-key-guard.sh` — all Redis access is via `libs/atlas-redis`
   registries; `UpdateWithTTL` is added *inside* the lib.
5. `tools/goroutine-guard.sh` — no new goroutines.
6. `tools/lint.sh --check` from the repo root.
7. `npm run build` in `services/atlas-ui`.

Guards not applicable: service-registration (no new service), template
opcode/duplicate/movement (no template change), skill-job-id, buff-duration.

## 10. Documentation

`services/atlas-data/docs/rest.md` — the `GET /api/data/process` section
(currently lines 81–90: "Lists active/recent ingest Jobs", raw-JSON shape, 503
on absent client) is rewritten for the scope parameter, the JSON:API
`ingestRun` shape, the six phases, the five worker states, and the removal of
the 503 branch. The `POST` section gains a sentence noting the run record is
initialised at Job creation.
