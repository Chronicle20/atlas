# Shared Service Bootstrap — Design

Task: task-118-shared-service-bootstrap
Status: Proposed
Date: 2026-07-02
Inputs: `docs/tasks/task-118-shared-service-bootstrap/prd.md` (approved)

---

## 1. Summary

Three shared-lib additions and one fleet migration:

1. `libs/atlas-kafka/producer` gains `Provider` + `ProviderImpl` (DUP-1) — a verbatim move of the 51-way-identical service wrapper.
2. `libs/atlas-service` gains `CreateLogger` (DUP-2) with an emit-time snake_case field-key normalization hook (CP-9).
3. `libs/atlas-service` gains `Bootstrap(serviceName, opts...) *Runtime` (DUP-3) owning logger, teardown manager, tracer, a readiness controller (OPS-2 endpoint half), and opt-in configuration-projection wiring.
4. All 58 service `main.go`s migrate onto `Bootstrap`; every local `logger/init.go` (56), `kafka/producer/producer.go` wrapper (52), projection catch-up block (4), and atlas-merchant's private teardown-manager copy are deleted.

The Bootstrap API is functional options returning a `Runtime` handle (Decision D4). Readiness stays mounted under the existing `/api/` base path via one explicit line in each `main.go` (D5). The projection option wires around the four services' service-local `projection` packages through a tiny interface rather than extracting those packages into a lib (D6). atlas-renders is the single special case: it adopts Bootstrap without a tracer and keeps its root-mounted mux and `/healthz` probe (D7).

## 2. Measured baseline (2026-07-02 snapshot; re-measure post task-114/116 rebase per FR-6.2)

All counts verified against the worktree at design time:

| Fact | Value | Evidence |
|---|---|---|
| Service `main.go`s | 58 | `find services -name main.go -path '*/atlas.com/*'` |
| `logger/init.go` copies | 56 (53 byte-identical + 3 drifted-but-semantically-identical: atlas-map-actions, atlas-reactor-actions, atlas-portal-actions) | md5sum over the 56: 53× one hash, 3× another; diff shows renamed hook struct + reordered decls only |
| `kafka/producer/producer.go` wrappers | 52 (51 byte-identical + atlas-quest, which defines only `type Provider`, no `ProviderImpl`) | md5sum: 51× one hash, 1× quest |
| Files importing a service-local `kafka/producer` package | 232 | `grep -rl '"atlas-<svc>/kafka/producer"'` |
| Files importing BOTH the local wrapper and `libs/atlas-kafka/producer` | ~20+ (aliased imports; e.g. `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/producer.go`) | grep sweep |
| Services mounting `/readyz` today | 4: atlas-login, atlas-channel, atlas-world, atlas-character-factory (all under `SetBasePath("/api/")` → effective `/api/readyz`) | `grep -l MountReadiness */main.go` |
| `parseProjectionCatchupTimeout` copies | 4, byte-identical | the same four `main.go`s |
| Projection readiness gates | login/channel/character-factory: `caughtUp.CaughtUpNow() && !shuttingDown`; world: `configuration.SnapshotReady() && !shuttingDown` | `main.go` of each |
| Projection topics read | login/channel: `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS` + `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS` (+ `SERVICE_ID`); world/character-factory: tenant topic only | `main.go` of each |
| Services with no REST server, no local logger pkg, no tracer | 1: atlas-renders (raw `mux.Router`, `logrus.JSONFormatter`, root-mounted routes, `/healthz` already probed by `deploy/k8s/base/atlas-renders.yaml`) | its `main.go`, `logger.go`, manifest |
| Services with a private teardown-manager copy | 1: atlas-merchant (`atlas.com/merchant/service/teardown.go`, byte-equivalent to `libs/atlas-service/teardown.go`) — **not in the PRD; discovered during design** | grep `GetTeardownManager` outside main.go |
| `producer.GetManager().Close` teardown line in main.go | 52 | grep |
| logrus version | v1.9.4 fleet-wide | go.mod |
| `logrus.Entry.log()` duplicates the entry (incl. `Data` map) before firing hooks | confirmed in v1.9.4 source (`entry.Dup()` at entry.go:227) | module cache read |

The last row is the safety proof for the CP-9 hook: a logrus hook may mutate `entry.Data` in place without racing callers that retain a derived `*Entry`, because hooks always fire on a per-emission copy.

## 3. Design decisions

### D1 — Producer move (DUP-1)

Add `libs/atlas-kafka/producer/provider.go`:

```go
// Provider resolves a topic token to a ready-to-use MessageProducer.
type Provider func(token string) MessageProducer

// ProviderImpl is the canonical provider: span + tenant header decorators
// over the manager-owned writer for the token's topic.
func ProviderImpl(l logrus.FieldLogger) func(ctx context.Context) Provider {
	return func(ctx context.Context) Provider {
		sd := SpanHeaderDecorator(ctx)
		td := TenantHeaderDecorator(ctx)
		return func(token string) MessageProducer {
			return Produce(l)(ManagerWriterProvider(l)(token))(sd, td)
		}
	}
}
```

This is the service wrapper verbatim with intra-package references (`producer.X` → `X`). The return type changes from the anonymous `func(token string) producer.MessageProducer` to the named `Provider` — identical underlying type, assignable at every call site, and it satisfies FR-1.1's signature.

Migration mechanics:

- All 232 importing files: rewrite `"atlas-<svc>/kafka/producer"` → `"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"`. Since both packages are named `producer`, unaliased call sites (`producer.Provider`, `producer.ProviderImpl`) compile unchanged.
- Dual-import files (local + lib, one aliased): drop the local import, rename the alias's uses to the surviving `producer` identifier. `goimports` + compile errors make misses impossible to land.
- Delete the 52 `kafka/producer/producer.go` wrappers. Sibling per-domain builders under `kafka/producer/<domain>/` and the two `producer_test.go` files (atlas-reactors, atlas-marriages) are kept with imports rewritten.
- atlas-quest: same import rewrite; it simply never references `ProviderImpl` (FR-1.4 satisfied — nothing forces adoption).

No behavior change: same decorators, same order, same manager. The lib's existing `producer_test.go`/`manager_test.go` already cover `Produce`/manager; add one test pinning `ProviderImpl`'s decorator composition (headers present, order) using the existing `producertest` fakes.

### D2 — `CreateLogger` into `libs/atlas-service` (DUP-2)

`libs/atlas-service/logger.go` (package `service`):

```go
func CreateLogger(serviceName string) *logrus.Logger
```

Body is the canonical 53-way-identical implementation: stdout, `service.name` hook, `ecslogrus.Formatter`, `LOG_LEVEL` parse with silent fallback on invalid values — plus the CP-9 normalization hook registered **last** (D3). The 3 drifted variants are semantically identical (verified by diff) and unify without behavior change. All 56 `logger/init.go` files and their now-empty `logger` packages are deleted; `main.go`s obtain the logger from `Bootstrap` (D4). `CreateLogger` stays exported (FR-2.1) for tests and any non-Bootstrap use.

New deps for `libs/atlas-service/go.mod`: `sirupsen/logrus`, `go.elastic.co/ecslogrus`, `google/uuid` (D6), `libs/atlas-tracing` (D4). No import cycles: atlas-tracing imports only otel + logrus; nothing in `libs/` imports atlas-service except services themselves. `libs/atlas-service` is already in `go.work` and both Dockerfile COPY blocks — no manifest edits.

### D3 — snake_case normalization hook (CP-9)

An unexported logrus hook (`fieldKeyNormalizerHook`) registered by `CreateLogger` at all levels, firing after the `service.name` hook. Algorithm, per key in `entry.Data`:

1. **Pass through** if the key contains a dot (`service.name`, any ECS/namespaced `x.y` key) or contains no uppercase letter (already snake_case, or plain lowercase).
2. Otherwise **convert** camelCase → snake_case: insert `_` at a lower/digit→upper boundary and at the last upper of an upper-run followed by a lower (`characterId` → `character_id`, `characterID` → `character_id`, `HTTPServer` → `http_server`, `worldId2` → `world_id2`), then lowercase.
3. **Collision rule**: if the converted key already exists in `entry.Data`, the existing (explicitly snake_case) value **wins** and the camelCase entry is dropped. Rationale: an author who wrote both spellings made the snake_case one deliberately; the rule is deterministic regardless of map iteration order (unlike last-writer-wins). Documented + unit-tested per FR-3.2.

Mutation is in place on `entry.Data` — safe per the logrus v1.9.4 `Dup()` proof in §2. Conversion allocates only when a key actually needs rewriting (check-before-convert), so already-migrated call sites cost one string scan per key; no map copy. That satisfies the NFR "no pathological allocation".

Ordering caveat (documented in the lib): keys added by hooks registered *after* the normalizer escape normalization. `CreateLogger` registers it last, and no service registers extra hooks today (verified: zero `AddHook` calls outside `logger/` packages).

Unit tests: conversion table (camel, acronym runs, digits, single-word, already-snake), dotted-key passthrough, collision (both spellings present → snake value survives), idempotence (running twice is a no-op), and a formatter-integration test asserting the emitted JSON from a `*logrus.Logger` built by `CreateLogger` carries `character_id`.

`docs/observability.md` gains a section: snake_case is canonical, the hook normalizes legacy camelCase at emit time, new code should write snake_case directly.

**Rejected alternative — fleet call-site rename:** ~1,500 `WithField` call sites (619 `characterId`, 462 `worldId`, 271 `transactionId`, …) across 50+ services; enormous mechanical diff, and nothing prevents reintroduction the day after. The hook is smaller, drift-proof, and reversible.

### D4 — `Bootstrap` API shape (DUP-3)

**Chosen: functional options returning a `Runtime` handle** (`libs/atlas-service/bootstrap.go`, package `service`):

```go
func Bootstrap(serviceName string, opts ...Option) *Runtime

type Option func(*bootstrapConfig)
func WithoutTracer() Option                                  // atlas-renders only
func WithReadinessGate(fn func() bool) Option                // ANDed into Ready()
func WithConfigProjection(baseGroupId string, build ProjectionBuilder) Option // D6

type Runtime struct { /* unexported fields */ }
func (r *Runtime) Logger() *logrus.Logger
func (r *Runtime) Context() context.Context        // teardown context
func (r *Runtime) WaitGroup() *sync.WaitGroup
func (r *Runtime) TeardownFunc(f func())
func (r *Runtime) TeardownManager() *Manager       // for callees typed on *service.Manager (e.g. login's buildListener)
func (r *Runtime) Ready() bool                     // readiness controller output (D5)
func (r *Runtime) AwaitProjectionCatchUp()         // D6; Fatal on timeout; panics if no projection option (misuse guard)
func (r *Runtime) Wait()                           // tdm.Wait() + "Service shutdown." log
```

`Bootstrap` executes, in order:

1. `CreateLogger(serviceName)`; log `"Starting main service."` (absorbs today's first line).
2. `GetTeardownManager()` (existing singleton, unchanged — stray direct callers keep working).
3. `tracing.InitTracer(serviceName)` unless `WithoutTracer`; on error `l.WithError(err).Fatal(...)` — preserving today's fail-fast (FR-4.5). Registers `tracing.Teardown(l)(tc)` immediately (today it's registered later in main.go, but `Manager.TeardownFunc` goroutines all fire concurrently on `doneChan` close, so registration order was never semantically meaningful).
4. Readiness controller: a `shuttingDown atomic.Bool` plus a teardown func that flips it and logs, exactly as the 4 services hand-roll today. `Ready()` returns `!shuttingDown && AND(all WithReadinessGate fns)`.
5. Projection wiring when the option is present (D6).

**What stays in `main.go`** (FR-4.3): DB/Redis connections, consumer registration, REST server builder + route initializers (including the `MountReadiness` line, D5), socket services, tasks, service-specific teardowns — and the one-line `rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })`. Absorbing the producer close into Bootstrap was rejected: it would couple `libs/atlas-service` to `libs/atlas-kafka` for one line, and atlas-renders (no Kafka) would need an opt-out.

A minimal migrated `main.go` (the ~50-service common shape):

```go
func main() {
	rt := service.Bootstrap(serviceName)
	l := rt.Logger()

	db := database.Connect(l, database.SetMigrations(...))     // unchanged
	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	...                                                        // consumers unchanged
	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath("/api/").
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
```

**Alternatives considered:**

- **B — config struct** (`Bootstrap(service.Config{Name: ..., Projection: ...})`): fine for 3 knobs, but every future knob lands in one struct and zero-value ambiguity creeps in (is `Tracer: false` "off" or "unset"?). Options match the repo's builder/curried idiom (REST `Builder`, `database.Connect(l, configurators...)`, `producer.GetManager(configurators...)`).
- **C — run-callback inversion** (`service.Run(name, func(rt *Runtime) { ... })` owning `Wait()`): guarantees `Wait` is never forgotten, but turns every `main` body into a closure, makes early-`return`-vs-`Fatal` semantics subtle, and is a bigger rewrite of 58 files for no measured failure mode (no service today forgets `tdm.Wait()`).

### D5 — Readiness controller & mounting (OPS-2 endpoint half)

- **Baseline semantic** (FR-5.1): `Ready()` = `!shuttingDown` (AND any gates). "Ready once the REST server is serving" is enforced structurally, not by a flag: the probe can only reach `/readyz` after `ListenAndServe` is up, and the route only exists on the served router. Before serving → connection refused → not ready; after SIGTERM → teardown flips `shuttingDown` → 503. This exactly generalizes the 4 hand-rolled implementations.
- **Mounting** stays an explicit `AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready))` line in each `main.go`, under the existing `SetBasePath("/api/")` — effective path `/api/readyz`, identical to the 4 probed services (FR-5.3, `bug_readiness_probe_path_under_api_basepath`). Keeping the line in `main.go` follows FR-4.3 (Bootstrap doesn't own the REST builder); the acceptance grep ("every REST-serving service mounts /readyz") is the drift guard.
- **OPS-3 (root-mounted `/readyz`): not now.** The shared server builds a single `PathPrefix(basePath)` subrouter (`libs/atlas-rest/server/server.go:64-76`); a root mount needs either a second router layer or `SetRouterProducer` surgery in all services. That is OPS-3's decision to make once, with the k8s manifests (OPS-1) in hand. This task guarantees the endpoint exists at the same effective path the 4 existing probes already use, so OPS-1 can standardize on `/api/readyz`.
- `Runtime.Ready` is a plain `func() bool` (method value) rather than a `server.RouteInitializer`, so `libs/atlas-service` takes **no dependency on `libs/atlas-rest`** — atlas-renders (D7) reuses the same controller on its raw mux.

### D6 — Configuration-projection option (FR-4.2)

The four projection implementations share the main.go *choreography* but not their types: each service has its own `configuration/projection` package, and the `State`/apply-loop halves are genuinely service-specific (login/channel drive socket listener registries; world/character-factory bridge tenant-config snapshots). **The lib therefore owns the choreography and wires around the service packages via a two-method interface** — it does not absorb the packages:

```go
type ProjectionTopics struct{ ServiceStatus, TenantStatus string } // read from env by the lib

type Projection interface {
	Start(ctx context.Context, l logrus.FieldLogger, wg *sync.WaitGroup, groupId string) error
	WaitCaughtUp(ctx context.Context) error
}

type ProjectionBuilder func(t ProjectionTopics) Projection
```

The existing service `Subscriber`+`CaughtUp` pairs already satisfy this shape (`sub.Start(ctx, l, wg, groupId)`, `caughtUp.WaitCaughtUp(ctx)`) — each service's builder closure composes its own `State`/`CaughtUp`/`Subscriber` (capturing `serviceId` where needed) and returns a small adapter.

`WithConfigProjection(baseGroupId, build)` makes `Bootstrap`:

1. Read `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS` + `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS`; warn when the tenant topic is unset (see note below).
2. Generate the per-process group id: `fmt.Sprintf("%s - projection - %s", baseGroupId, uuid.New().String())` — preserving the replay-from-FirstOffset behavior the 4 services depend on.
3. Call `build(topics)` and `Start(...)` bound to the teardown context/waitgroup; `Fatal` on start error (today's semantics).
4. Arm `Runtime.AwaitProjectionCatchUp()`: `context.WithTimeout(rt.Context(), parseProjectionCatchupTimeout())` → `WaitCaughtUp` → `Fatal` on error. `parseProjectionCatchupTimeout` (env `PROJECTION_CATCHUP_TIMEOUT_S`, 5-minute default) moves into the lib as an unexported helper; the 4 byte-identical copies are deleted (acceptance grep → 0).

**Catch-up gate placement stays with `main.go`** via the explicit `rt.AwaitProjectionCatchUp()` call, because the four services legitimately differ: world/character-factory start the REST server *before* gating (so `/readyz` serves 503 during catch-up), login/channel gate before building listeners. One call in the service's chosen position preserves each ordering exactly while the logic lives once in the lib.

**Readiness gates stay service-supplied** via `WithReadinessGate`: login/channel/character-factory pass `caughtUp.CaughtUpNow`, world passes `configuration.SnapshotReady`. Auto-ANDing `CaughtUpNow` inside the option (FR-5.2's sketch) would silently change world's gate; an explicit gate preserves all four (and FR-5.2's intent — catch-up state is ANDed into readiness — is met by construction in each service's option list).

Two accepted, documented micro-changes in log/startup behavior:

- Subscriber start moves to `Bootstrap()` time — in login/channel that is slightly *earlier* than today (before regular consumer registration). The projection consumes independent compacted topics; ordering against other consumers is not load-bearing.
- Warn condition unifies to "tenant topic unset" (world/character-factory's rule). Login/channel today warn only when *both* topics are unset; after unification they'd also warn when only the tenant topic is missing — a strictly more-informative warning in a misconfiguration corner.

**Rejected alternative — extract the projection packages into a shared lib** (e.g. `libs/atlas-config-projection`): the subscriber/caught-up/envelope halves are near-identical, but `State` and apply loops are service-specific; a clean extraction is real design work with its own blast radius, belongs with RR-3's follow-ups (the PRD's non-goal), and isn't needed to delete the duplicated main.go block. Per the audit-existing-libs rule: no existing `libs/atlas-*` overlaps this space today.

### D7 — atlas-renders special case (FR-5.4)

atlas-renders is the one non-standard service: raw `mux.Router` at root (no base path), image-serving routes, `/healthz` already wired to liveness+readiness probes, plain `logrus.JSONFormatter` logger, no tracer, no Kafka, no graceful teardown.

Migration: `rt := service.Bootstrap(serviceName, service.WithoutTracer())`; delete `logger.go`. It keeps its raw mux (moving onto the shared REST builder would apply `CommonHeader`, forcing `Content-Type: application/json` onto PNG routes — wrong, and exactly the kind of silent behavior change this task forbids). It keeps `/healthz` (probed by the live manifest — FR-5.3's "don't change probed URLs" applies) and additionally mounts `/readyz` at root backed by `rt.Ready`, giving it SIGTERM-aware readiness it lacks today; graceful shutdown wraps `http.Server` with `rt.Context()`/`rt.Wait()` instead of bare `ListenAndServe`.

Accepted observable change (flagged per PRD §8): its log format changes from plain `JSONFormatter` to the fleet-standard ecslogrus + `service.name` + snake_case — a unification, called out in the PR. Adding a tracer was rejected: `TRACE_ENDPOINT` isn't in its deployment and behavior preservation wins; `WithoutTracer` keeps FR-4.4's letter ("no direct `tracing.InitTracer` calls") and spirit.

Also discovered and folded in: **atlas-merchant's private teardown manager** (`atlas.com/merchant/service/teardown.go`, byte-equivalent to the lib's) is deleted; its `main.go` migrates onto the lib `Runtime` like everyone else.

## 4. Migration strategy

Commit structure (PRD §8 reviewability requirement — lib commits separated from mechanical sweeps):

1. `libs/atlas-kafka/producer`: `provider.go` + test.
2. `libs/atlas-service`: `logger.go` + normalization hook + tests.
3. `libs/atlas-service`: `bootstrap.go`/`runtime.go`/options + tests (incl. projection option against a fake `Projection`).
4. One commit per service (58), each: `main.go` rewrite, `logger/` deletion, producer-wrapper deletion + import rewrites, `/readyz` line. Ordered by shape cohort so review ramps from trivial to complex:
   - Cohort A: plain kafka+REST services (atlas-fame shape) — the bulk.
   - Cohort B: +database/+redis/+tasks variants (same template, more retained lines).
   - Cohort C: atlas-quest (Provider-only), atlas-merchant (private teardown deletion), atlas-saga-orchestrator and other dual-import-heavy services.
   - Cohort D: the 4 projection services (login, channel, world, character-factory) — behavior-critical, reviewed line-by-line against the current blocks.
   - Cohort E: atlas-renders.
5. Docs: `docs/architectural-improvements.md` (DUP-1/2/3 ✓, CP-9 ✓, OPS-2 endpoint-half note), `docs/observability.md` (snake_case convention).

Rebase gate (FR-6.1/6.2): no fleet edits until task-114 and task-116 are merged and this branch is rebased onto them. First plan step after rebase: re-run the §2 measurement commands; if the canonical wrapper/main.go shapes moved (outbox-era emit paths), update `provider.go`'s body and the migration template to the *post-rebase* canon before any sweep. The plan must treat §2's numbers as expiring.

## 5. Error handling

- Tracer init failure, projection subscriber start failure, projection catch-up timeout: `Fatal` inside the lib with the same log messages — identical process-exit semantics to today (FR-4.5).
- `AwaitProjectionCatchUp()` without `WithConfigProjection`: panic with a clear message (programmer error, caught by the service's own startup in dev, not a silent no-op).
- Invalid `LOG_LEVEL` / `PROJECTION_CATCHUP_TIMEOUT_S`: silently keep defaults, as today.
- Normalization hook never returns an error (logrus hooks that error print to stderr and drop nothing; there is no failure path in a key rewrite).

## 6. Testing & verification

Lib unit tests (all in the two libs, `go test -race`):

- `ProviderImpl`: span+tenant headers present and in today's order, via `producertest` fakes.
- `CreateLogger`: formatter/hook wiring; emitted JSON contains `service.name` and normalized keys.
- Normalizer: conversion table, dotted passthrough, collision rule, idempotence (§D3).
- `Bootstrap`/`Runtime`: readiness flips 200→503 across a simulated teardown; gates AND correctly; projection option drives a fake `Projection` (start called with generated group id; `AwaitProjectionCatchUp` honors timeout env; Fatal paths asserted via logrus test hooks where practical).

Fleet verification (CLAUDE.md, all mandatory): `go test -race ./...`, `go vet ./...`, `go build ./...` per changed module; `docker buildx bake all-go-services` (every service is touched); `tools/redis-key-guard.sh`.

Runtime verification (acceptance criteria): on a deployed/locally-run previously-readiness-less service — `/api/readyz` → 200, SIGTERM → 503 before exit; one live log line showing `character_id`-style keys; the 4 existing services' `/api/readyz` behavior spot-checked unchanged (probe paths in `deploy/k8s/base/atlas-{world,character-factory}.yaml` remain valid).

Acceptance greps (from the PRD, run in CI-shape at the end): zero `*/kafka/producer/producer.go`, zero `*/logger/init.go`, zero `parseProjectionCatchupTimeout` under `services/`, zero `tracing.InitTracer` in `main.go`s, `MountReadiness` present in every REST-serving `main.go`.

## 7. Risks & mitigations

| Risk | Mitigation |
|---|---|
| task-114/116 land shapes that invalidate §2 (outbox-era producer wiring, rewritten main.go s) | Hard rebase gate + re-measure step (§4); the design's decisions (D1–D7) are shape-independent, only the verbatim bodies get re-derived |
| Dual-import files: alias unification breaks compile in odd corners | Purely compile-time; per-service commits keep failures localized; `goimports` + `go build` per cohort |
| Hook mutates entry data concurrently | Disproved for logrus v1.9.4 (`Dup()` before hooks, §2); pin: a race test logging from parallel goroutines through one shared derived entry under `-race` |
| snake_case conversion surprises Loki dashboards/alerts mid-transition | Keys converge to the spelling that already dominates queries going forward; docs updated; camelCase disappears from new logs at deploy time — call out in PR description for anyone with saved camelCase queries |
| Projection micro-changes (earlier subscriber start, unified warn condition) hide a real regression | Both enumerated in D6 and in the Cohort D commit messages; catch-up gate position and readiness gates preserved exactly |
| atlas-renders formatter change breaks a log consumer | Its logs were already JSON; ecslogrus is JSON with ECS envelope; flagged in PR |
| A `main.go` outside the 4 shapes resists the template | 58 is enumerable; the plan lists every service explicitly — no "and the rest" bucket |

## 8. Resolution of PRD open questions

1. **Bootstrap API shape** → functional options + `Runtime` handle (D4); handle exposes logger, context, waitgroup, teardown, `Ready`, `AwaitProjectionCatchUp`, `Wait`.
2. **Root-mounted `/readyz`** → No; preserve `/api/readyz` exactly; OPS-3 decides relocation together with OPS-1's manifests (D5).
3. **snake_case mechanism** → emit-time hook confirmed; collision rule = explicit snake_case key wins; safety proven against logrus v1.9.4 internals (D3).
4. **Non-REST services** → exactly one (atlas-renders); handled per D7, documented, keeps `/healthz`, gains root `/readyz`.
5. **Post-114/116 producer shape** → `ProviderImpl` moves as found *at rebase time*; the §4 re-measure step owns adopting any outbox-era canon; if 114 deletes `ProviderImpl` usage fleet-wide, the lib exports whatever the post-114 canonical wrapper is instead (decision deferred to the measured reality, per FR-6.2).
