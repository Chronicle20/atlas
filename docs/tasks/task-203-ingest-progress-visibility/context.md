# Ingest Progress Visibility — Implementation Context

Companion to [`plan.md`](plan.md). Everything here is grounded in the code on
this branch at planning time; `file:line` references are to the worktree.

## 1. What exists today

An operator uploads WZ archives and clicks Process Data. `atlas-data` running
`MODE=rest` renders a Kubernetes Job from the `atlas-data-ingest-job-template`
ConfigMap and submits it (`runtime/rest/jobs.go:194` `JobCreator.Create`). That
Job runs `MODE=ingest` (`runtime/ingest/run.go:23`), which fans eleven
registered workers out over the scope's archives (`data/runwz.go:45`
`RunWorkers`). The run takes tens of minutes; the Map worker alone has
previously exceeded a thirty-minute watchdog cutoff.

The UI shows none of it. Setup and Baselines poll `GET /api/data/status` every
five seconds and render a document count (`useSeed.ts:298`) — a numerator with
no denominator.

Three pieces are already in place and are reused rather than replaced:

- `GET /api/data/process` exists and is routed (`runtime/rest/resource.go:26`)
  but has **zero consumers** — verified: nothing under `services/atlas-ui/src`
  references it (only `POST`, at `seed.service.ts:241,269`). Changing its shape
  is therefore not a breaking change in practice.
- A per-run Redis registry is written from both sides: the REST pod stamps the
  Job name at creation (`jobs.go:206`) and the ingest pod refreshes an
  `:updatedAt` heartbeat every thirty seconds (`runtime/ingest/heartbeat.go:49`).
- `wzinput.ResolveScope` (`wzinput/scope.go:21`) already implements exactly the
  scope rules FR-4.1 asks for, including the `X-Atlas-Operator: 1` gate.

## 2. Key files

### Backend (`services/atlas-data/atlas.com/data`, module `atlas-data`)

| Path | Role |
|---|---|
| `runtime/rest/jobs.go` | `JobCreator`, `renderJob`, `sanitizeLabel`, key-suffix helpers, Redis registry construction |
| `runtime/rest/resource.go` | `InitResource`, `processCreate`, `processStatus` |
| `runtime/rest/watchdog.go` | `sweep`, `jobIsStuck`, `deleteStuckJob` |
| `runtime/ingest/run.go` | `MODE=ingest` entry point; builds `workers.Params` from env |
| `runtime/ingest/heartbeat.go` | `runHeartbeat`, `ingestJobSuffixFromEnv` |
| `data/runwz.go` | `RunWorkers`, `runOne`, `splitPrerequisites` |
| `data/workers/registry.go` | `Registered` — the eleven-worker canonical list |
| `data/status.go` | `StatusRestModel` — the JSON:API model template to copy |
| `wzinput/scope.go` | `ResolveScope`, `ErrSharedRequiresOperator` |
| `main.go` | mode switch, `rdb`/`JobCreator`/`Watchdog` wiring, route registration (`:175–201`) |
| `../../docs/rest.md` | the `/api/data/process` contract |

### Shared lib

`libs/atlas-redis/registry.go` — generic `Registry[K, V]` with `Get`, `Put`,
`PutWithTTL`, `Update`, `Remove`. Values are JSON-marshalled automatically
(`registry.go:30`), so a struct value type needs no bespoke encoding.

### Frontend (`services/atlas-ui/src`)

| Path | Role |
|---|---|
| `services/api/seed.service.ts` | `fetchJsonApi` (unwraps `data.attributes`), `postProcess`, all status fetchers |
| `lib/headers.tsx` | `tenantHeaders`, `canonicalHeaders` (bakes in `X-Atlas-Operator: 1`), `CanonicalSelection` |
| `lib/hooks/api/useSeed.ts` | tenant-scope polling hooks (`refetchInterval: 5000, staleTime: 0`) |
| `lib/hooks/api/useCanonicalData.ts` | shared-scope equivalents |
| `components/features/setup/SetupRow.tsx` | the row primitive; has a `warning?: ReactNode` slot |
| `pages/SetupPage.tsx` | "Process Data" row at ~`:374` |
| `pages/BaselinesPage.tsx` | `publishDisabled` at `:153`, "Publish Baseline" row at ~`:309` |

## 3. Decisions carried in from the design

**Redis is the system of record; Kubernetes is a second opinion.** The Job
object is the thing that disappears (`ttlSecondsAfterFinished: 3600`, Watchdog
deletion), so it cannot be the source of truth for a feature whose whole point
is surviving its disappearance. Kubernetes is consulted only to demote a stale
`running` to `unknown`, and terminal phases are served with no Kubernetes call
at all.

**One key, one `Get`.** The whole record — all worker entries — lives under
`<prefix>:data-ingest:<suffix>:run`. Per-worker keys were rejected: the read
becomes a SCAN or eleven GETs per poll per open page, the run phase has no
home, and partial-write states become representable. The contention it would
avoid is not real (eleven transitions over minutes at `INGEST_MAX_PARALLEL=4`
against `updateMaxRetries = 1000`).

**A separate `:run` key, not an extension of the existing ones.** Those are
typed `Registry[string, string]` and read by the Watchdog's staleness check.

**`Record` is a DTO, not an immutable model with a Builder.** The project's
Builder convention governs *domain* models flowing through processors; this is
a wire/storage shape mutated inside an optimistic-lock closure, in the same
category as the existing `processStatusJob` and `StatusRestModel`. Mutation is
still confined to named copy-returning methods, which is what keeps the closure
pure across retries.

**Resolved PRD open questions:**
- *Q1 — a `stuck` run's per-worker record.* Preserved exactly as the ingest pod
  left it. Marking the in-flight worker `failed` would assert something we do
  not know; "MAP was the one still going" is the diagnostic value. The UI
  renders `running` under a terminal phase as "interrupted" — derived, not
  stored.
- *Q2 — elapsed-time source.* The record's `startedAt`, stamped by the REST pod
  at Job creation and never overwritten. The Kubernetes `startTime` vanishes
  with the Job; one clock covers both the running and terminal cases; and
  scheduler-pending time is time the operator waited.
- *Q3 — the Setup document-count badge.* Kept, with the panel as a separate row
  beneath. A completed run with a zero document count is a real, diagnosable
  state that folding them together would hide.

**Rejected alternatives:** Kubernetes as source of truth (the Job is deleted,
and Job status has no per-worker concept); Postgres run history (better
durability, but a PRD non-goal and a migration on the slowest path in the
system — mitigated by FR-5.6, where `none` never blocks publishing); Kafka
progress events (no consumer wants a stream, and it would add a topic
registration to a feature that claims no deployment change); SSE/WebSocket (a
second transport for one panel).

## 4. Three pre-existing defects fixed in scope

These are not opportunistic cleanups — the feature cannot be correct without
them.

**F-1 — `Registry.Update` silently drops the key's TTL.** `registry.go:100`
writes with `pipe.Set(ctx, rk, newData, 0)`, and in Redis a `SET` without an
expiry option *clears* any existing TTL. `PutWithTTL(7d)` at init followed by
`Update` per transition produces a key that loses its TTL on the very first
transition and then lives forever. The `EXPIRE`-on-the-raw-client escape hatch
is closed by `tools/redis-key-guard.sh`. Fixed by adding `UpdateWithTTL`
alongside `Update`; `Update`'s semantics are left alone because changing them
would touch every existing caller, including live monster-store state. This is
the only change outside `services/atlas-data` and `services/atlas-ui`.

**F-2 — the Watchdog reads a heartbeat key that tenant-scope runs never write.**
`Create` writes the heartbeat under the raw scope (`jobs.go:207`) —
`tenants/<uuid>:GMS:83.1`. The Watchdog reconstructs the suffix from the Job's
labels (`jobs.go:55`), where `scope` was written as `sanitizeLabel(scope)`
(`jobs.go:233`), mapping `/` to `-` — `tenants-<uuid>:GMS:83.1`. The two differ
for every tenant-scope run, so `jobIsStuck` (`watchdog.go:84`) never finds a
heartbeat for a tenant ingest and falls back to `j.CreationTimestamp` — i.e.
for tenant runs the Watchdog is still the creation-time cutoff the heartbeat was
added to replace. (`sanitizeLabel("shared") == "shared"`, so shared runs are
unaffected — which is why this went unnoticed.) It becomes blocking here because
FR-3.1 makes the Watchdog a *writer*: unfixed, it would write `stuck` to a key
nothing reads while the record the UI polls stayed `running` forever. Fixed by
rebuilding the raw scope from the unsanitized `tenant` label (`jobs.go:278`).
Repairing the tenant-scope heartbeat check is a deliberate side effect and is
the intended behaviour of the existing timeout.

**F-3 — the run registry cannot hang off `JobCreator`.** FR-4.5 requires the
endpoint to serve the Redis record when Kubernetes is unavailable, but
`JobCreator.Registry` only exists when `NewJobCreatorInClusterWithRedis`
succeeds; on failure `main.go:96` sets `jc = nil` and takes the registry with
it. `rdb` is also scoped inside the `MODE == "rest"` block (`main.go:91`).
Fixed by hoisting `rdb` and constructing an `IngestRegistries` bundle
independently, so the handler's two dependencies are independently nil-able.

**F-4 (not a defect, a constraint).** `runtime/ingest` imports `atlas-data/data`
(`run.go:4`), and the instrumentation point is `runOne` inside
`data.RunWorkers`. Progress reporting therefore enters through an interface the
`data` package declares, with the Redis implementation in `runtime/ingest`.

## 5. Traps

- **Raw scope vs sanitized label.** Redis keys use `tenants/<uuid>`; Kubernetes
  labels use `tenants-<uuid>`. The status handler's label selector must use the
  *sanitized* form (matching `renderJob`); the key suffix must use the raw form.
  This asymmetry is F-2; both sides get a test.
- **`Update` clears TTLs.** Never use `Update` on the run key. Task 1's
  `TestRegistry_Update_ClearsTTL` pins the behaviour so a future refactor
  cannot silently undo the reason `UpdateWithTTL` exists.
- **Cancelled context at the terminal write.** When a worker fails the errgroup
  cancels `gctx`, and `Finish(failed)` is precisely the write that must land.
  Every progress write runs under `context.WithoutCancel(ctx)` +
  `WithTimeout(5s)`.
- **Typed-nil interface.** A nil `*redisSink` passed through `data.WithProgress`
  produces a non-nil interface whose methods then run on a nil receiver.
  `runtime/ingest/run.go` builds the option slice conditionally instead.
- **`Record` mutators must deep-copy `Workers`.** Copying the struct shares the
  slice's backing array; the mutator may re-run up to 1000 times inside the
  optimistic lock.
- **Existing `watchdog_test.go` fixtures** set `scope: "tenants-t1"` with no
  `tenant` label. After the F-2 fix those cases correctly resolve to `""` and
  skip. Add the `tenant` label to the fixtures — do not weaken the assertion.
- **`tools/lint.sh --check` false-fails without node 22 on PATH.** Source nvm
  first. It can also false-fail on cross-worktree golangci-lint lock contention.
- **`npm run build` is the UI gate, not `vitest` alone** — the build is what
  type-checks the test files.
- **`docker buildx bake atlas-data` is mandatory** because `libs/atlas-redis`
  changed. `go build` against `go.work` will not catch a missing `COPY libs/...`
  in the shared root `Dockerfile`. No new lib is added, so no `Dockerfile` or
  `go.work` edit is expected — the bake is what proves it.

## 6. Design contradiction resolved

Design §7's error table says "`jc == nil` / k8s list error → serve the stored
record; skip the cross-check", while §4.4 says `jc == nil` "degrades a `running`
record to `unknown` only when the heartbeat is also absent". **§4.4 wins** — it
is the FR-4.3 requirement, and `unknown` is the conservative choice for the
publish gate. The plan implements:

| Situation | Reported phase |
|---|---|
| fresh heartbeat (within `DefaultWatchdogTimeoutSecs`) | `running`, no Kubernetes call |
| no fresh heartbeat, `jc == nil` | `unknown` |
| no fresh heartbeat, Job list **errors** | `running` (a failed list is not evidence of absence) |
| no fresh heartbeat, list succeeds, no live Job | `unknown` |

`unknown` is computed at read time and never stored, so a record that later
proves alive recovers on the next poll with no repair path.

## 7. Test infrastructure already available

- `miniredis` (`github.com/alicebob/miniredis/v2`) — used in
  `runtime/rest/watchdog_test.go` and `libs/atlas-redis/registry_test.go`.
  `mr.TTL(key)` returns `0` when a key has no expiry; `mr.Close()` simulates an
  outage.
- `k8s.io/client-go/kubernetes/fake` `NewSimpleClientset` — used in
  `runtime/rest/jobs_test.go` and `resource_test.go`. It applies label
  selectors server-side, so a selector test is genuine.
- Tenant context in a handler test: `tenant.Create(uuid, region, major, minor)`
  then `tenant.WithContext(ctx, tm)` (`libs/atlas-tenant/processor.go:30,90`).
- JSON:API marshalling in a handler test needs a non-nil `ServerInformation` —
  copy the `fakeServerInfo` stub from `data/item/string_search_test.go:453`.
- Redis key prefix in assertions: `redis.KeyPrefix()` (`libs/atlas-redis/keys.go:29`),
  which is `"atlas"` when `ATLAS_ENV` is unset.
- UI component tests live in `components/features/<area>/__tests__/` and use
  vitest + `@testing-library/react`; see
  `components/features/baselines/__tests__/BaselineTargetPicker.test.tsx`.

## 8. Deployment

No change. No new ConfigMap, Secret, RBAC rule, service registration, Kafka
topic, or database migration. The `atlas-data` ServiceAccount's existing Job
permissions and the existing Redis connection suffice.
`tools/service-registration-guard.sh` is unaffected.
