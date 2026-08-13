# Backend Audit — atlas-transports (task-198 branch diff)

- **Service Path:** `services/atlas-transports/atlas.com/transports`
- **Diff Range:** `354fb1257..1c1b7798f`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-07
- **Build:** PASS (reported clean by task owner; not re-run per instructions)
- **Tests:** PASS (`go test -race -count=1 ./...` reported clean; not re-run per instructions)
- **Overall:** NEEDS-WORK

## Scope Note

`atlas-transports` has no GORM/DB layer anywhere in the module (`grep -rn "gorm.DB" transport/ instance/` — zero matches outside `_test.go`; no `entity.go`/`administrator.go`/`provider.go` file exists in either package). State lives in in-memory/Redis-backed registries (`transport/route_registry.go`, `instance/instance_registry.go`). This predates the branch and is out of scope to relitigate; DOM checks that presuppose a GORM entity layer (DOM-02, DOM-03, DOM-10, DOM-11, DOM-16, DOM-27) are marked N/A below with the absence cited as evidence, not silently skipped.

Both `transport/` and `instance/` have `model.go` → classified **domain package**, full DOM checklist run. No new service, no new external HTTP client, no packet/dispatcher work, no auth surface → SCAFFOLD/EXT/SEC checklists not triggered.

## Domain Checklist Results

### transport

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists | PASS | `transport/builder.go` (untouched by this diff; `NewBuilder`, fluent setters, `Build()` present — confirmed pre-existing, not modified in `354fb1257..1c1b7798f`) |
| DOM-02 | ToEntity() method | N/A | No `entity.go` in package; no GORM persistence — confirmed `find . -iname entity.go` returns nothing |
| DOM-03 | Make(Entity) function | N/A | Same as DOM-02 |
| DOM-04 | Transform function | PASS | `transport/rest.go:158` `func Transform(m Model) (RestModel, error)` |
| DOM-05 | TransformSlice function / list handlers use it | WARN | No `func TransformSlice` in `transport/rest.go`; `transport/resource.go:116` calls `model.SliceMap(transformer)(...)` directly. This is the alternate composition explicitly sanctioned in `ai-guidance.md` ("Useful Composition": `ops.SliceMap(Transform)(ops.FixedProvider(models))(ops.ParallelMap())()`), and predates this diff — the line was `model.SliceMap(Transform)(...)` before, only the argument changed to the conditional `transformer`. Not a new violation; flagged non-blocking because the guideline is internally inconsistent (file-responsibilities.md wants a named `TransformSlice`, ai-guidance.md's own example doesn't use one) |
| DOM-06 | Processor accepts FieldLogger | PASS | `transport/processor.go:50` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` |
| DOM-07 | Handlers pass d.Logger() | PASS | `transport/resource.go:34,95,116` all `NewProcessor(d.Logger(), d.Context())` |
| DOM-08 | POST/PATCH use RegisterInputHandler | PASS (N/A — none touched) | No POST/PATCH routes touched by this diff in `transport/resource.go`; existing routes unaffected |
| DOM-09 | Transform errors handled | PASS | `transport/resource.go:100-105` (`restModel, transformErr := transformer(route)` then `if transformErr != nil`), same pattern at the `AllRoutesProvider` call site |
| DOM-10 | Test DB has tenant callbacks | N/A | No SQL/GORM test DB in this package — registries seeded directly via Builders in tests (e.g. `transport/evaluate_test.go:16-33`) |
| DOM-11 | Providers use lazy evaluation | N/A | No `provider.go`; registry reads are synchronous map lookups, not `database.Query` |
| DOM-12 | No os.Getenv() in handlers | PASS | `grep -n "os.Getenv" transport/resource.go` — zero matches |
| DOM-13 | No cross-domain logic in handlers | PASS | `transport/resource.go` handlers call only `transport` processor/registry; `wantsSchedule` (resource.go:49-60) is pure query-param parsing, not cross-domain orchestration |
| DOM-14 | Handlers don't call providers directly | PASS | `transport/resource.go:34,95,116` all route through `NewProcessor(...)` |
| DOM-15 | No direct entity creation in handlers | PASS | `grep -n "db.Create\|db.Save\|db.Delete" transport/resource.go` — zero matches (no DB in package) |
| DOM-16 | administrator.go exists for write ops | N/A | No DB; writes (if any) happen via processor→registry, no `administrator.go` pattern applies |
| DOM-17 | Domain error → HTTP status mapping | PASS | `transport/resource.go` unchanged mapping (404 not-found, 500 on transform/provider error) — touched lines add no new error branches |
| DOM-18 | JSON:API interface on REST models | PASS | `transport/rest.go:43-119` `RestModel` has `GetID`, `SetID`, `GetName`, `SetToOneReferenceID`, `SetToManyReferenceIDs` — new fields (`BoardingWindowSeconds` etc., `NextTransitionAt`, `NextState`) added without touching these methods |
| DOM-19 | Request models use flat structure | N/A | No new request models — read-only diff, per task scope |
| DOM-20 | Table-driven tests | PASS | `transport/evaluate_test.go:68-158` (`tests := []struct{...}` + `t.Run`); `transport/resource_include_test.go` uses `t.Run` subtests |
| DOM-21 | No duplication of atlas-constants types | PASS | Only new types added are `Transition` (`transport/model.go`, domain-specific, not a constants type) and a test-local `routeListDoc` struct (`transport/resource_include_test.go:19`) — neither overlaps `libs/atlas-constants` |
| DOM-22 | Dockerfile mentions per direct lib require | N/A | No `go.mod` change in this branch (confirmed `git diff 354fb1257..1c1b7798f -- go.mod go.sum` empty) |
| DOM-23 | Kafka topic naming convention | N/A | No new/changed Kafka topic constants in this diff |
| DOM-24 | Kafka producer stubbed in tests that emit | N/A | `grep -n "AndEmit\|message.Emit\|producer.Provide" transport/*_test.go` — zero matches in touched/new test files |
| DOM-25 | Client-interpreted byte values config-resolved | N/A | No channel/socket wire-code literals in this diff |
| DOM-26 | Goroutines spawned via routine.Go | PASS | `grep -rnE '^\s*go (func\|[A-Za-z_])' transport/*.go` (excluding `_test.go`) — zero new bare `go` statements in the diff |
| DOM-27 | Transient DB errors map to 503 | N/A | No DB in this service; no `database.Connect` call anywhere in the module |
| DOM-28 | No silent degradation in decorators/enrichment | PASS (N/A) | No `model.Decorator` implementations touched by this diff |

### instance

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists | PASS | `instance/builder.go` (untouched, pre-existing `NewRouteBuilder`, fluent setters) |
| DOM-02 | ToEntity() method | N/A | No `entity.go`; no GORM |
| DOM-03 | Make(Entity) function | N/A | Same |
| DOM-04 | Transform function | PASS | `instance/rest.go` `TransformRoute`, `TransformInstanceStatus` (both pre-existing, extended with new fields by this diff) |
| DOM-05 | TransformSlice function | WARN | Same as `transport` — `instance/resource.go:39` uses `model.SliceMap(TransformRoute)(...)` directly, not a package-local `TransformSlice`; pre-existing pattern, sanctioned by ai-guidance.md's composition example |
| DOM-06 | Processor accepts FieldLogger | PASS | `instance/processor.go:65` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` |
| DOM-07 | Handlers pass d.Logger() | PASS | `instance/resource.go:38,62,87` all `NewProcessor(d.Logger(), d.Context())` |
| DOM-08 | POST/PATCH use RegisterInputHandler | PASS | `instance/resource.go:26` `rest.RegisterInputHandler[StartTransportRestModel](l)(si)(...)` for the one POST route (unchanged by this diff) |
| DOM-09 | Transform errors handled | PASS | `instance/resource.go:66-70` (`rm, err := TransformRoute(route)` then `if err != nil`) |
| DOM-10 | Test DB has tenant callbacks | N/A | No SQL DB; registries seeded via Builders |
| DOM-11 | Providers use lazy evaluation | N/A | No `provider.go` |
| DOM-12 | No os.Getenv() in handlers | PASS | `grep -n "os.Getenv" instance/resource.go` — zero matches |
| DOM-13 | No cross-domain logic in handlers | PASS | Handlers call only `instance` processor/registry |
| DOM-14 | Handlers don't call providers directly | **FAIL** | `instance/resource.go:115-119` — `GetInstanceRouteStatusHandler` calls `ir := getInstanceRegistry(); instances := ir.GetInstancesByRoute(t.Id(), routeId)` directly from the handler, bypassing `Processor`. This is the **only** call site in the entire module where a `resource.go` handler reaches a registry directly instead of via `NewProcessor(...)` — every other registry access (`getInstanceRegistry()`/`getRouteRegistry()`) happens inside `instance/processor.go` or `transport/processor.go` (confirmed by `grep -n "getInstanceRegistry()\|getRouteRegistry()" instance/*.go transport/*.go`, all other hits are in `processor.go` files). The `Processor` interface (`instance/processor.go:22`) has no method exposing `GetInstancesByRoute`, so the handler had no processor path available — this diff's FR-6.1 tenant fix (`instance/resource.go:110-119`) touches this exact block (adding `t := tenant.MustFromContext(d.Context())` immediately above the bypass) and should have added a `Processor` method (e.g. `GetInstancesByRoute(routeId uuid.UUID) []TransportInstance` that internally reads `p.ctx`'s tenant and calls the registry) rather than perpetuating direct registry access at the handler layer. Severity: Important — this is a File-Responsibilities / anti-pattern violation (`anti-patterns.md` "Handlers calling provider functions directly"), not downgraded despite being adjacent to pre-existing code, because it is unique in the codebase, not a repeated convention |
| DOM-15 | No direct entity creation in handlers | PASS | No `db.Create/Save/Delete` — no DB in package |
| DOM-16 | administrator.go exists for write ops | N/A | No DB |
| DOM-17 | Domain error → HTTP status mapping | PASS | `instance/resource.go` mapping unchanged by this diff (400 on `StartTransportAndEmit` failure, 404 on route-not-found, 500 on transform failure) |
| DOM-18 | JSON:API interface on REST models | PASS | `instance/rest.go` `RouteRestModel`/`InstanceStatusRestModel` `GetID`/`SetID`/`GetName` untouched by field additions |
| DOM-19 | Request models use flat structure | N/A | No new request models in this diff |
| DOM-20 | Table-driven tests | PASS | `instance/resource_tenant_test.go:71-89` uses `t.Run` subtests; `instance/rest_test.go` two focused unit tests (acceptable — narrow scope, not a multi-case matrix) |
| DOM-21 | No duplication of atlas-constants types | PASS | No new types in `instance/` diff beyond REST fields (`uint32`, `string`) — no map/world/channel id redeclaration |
| DOM-22 | Dockerfile mentions | N/A | No go.mod change |
| DOM-23 | Kafka topic naming | N/A | No topic changes |
| DOM-24 | Kafka producer stubbed in tests | N/A | `grep -n "AndEmit\|message.Emit\|producer.Provide" instance/*_test.go` (new/touched files) — zero matches. Note: `instance/resource.go:87` (`StartTransportAndEmit`) is unmodified by this diff and not exercised by any new/touched test |
| DOM-25 | Client-interpreted byte values | N/A | No wire-code literals touched |
| DOM-26 | Goroutines via routine.Go | PASS | No new bare `go` statements in `instance/*.go` diff |
| DOM-27 | Transient DB errors → 503 | N/A | No DB |
| DOM-28 | No silent degradation in decorators | N/A | No decorators touched |

## File Responsibilities Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor logic in processor.go | PASS | `transport/processor.go`, `instance/processor.go` — no `ProcessorImpl` methods added outside these files by this diff |
| FILE-02 | RestModel + Transform + JSON:API methods in rest.go | PASS | `transport/rest.go`, `instance/rest.go` — all new fields, `TransformSummary`, and JSON:API methods live in `rest.go` |
| FILE-03 | Cross-service request funcs in requests.go | N/A | No `requests.go` in either package; no external HTTP client calls added |
| FILE-04 | Entity + Migration + TableName in entity.go | N/A | No `entity.go` — no GORM entities in this service |
| FILE-05 | Builder/Model/administrator/provider/state placement | PASS | `Transition` type and `Evaluate`/`timeOfDay`/`materializeBoundary` helpers added to `transport/model.go` — correct: this is domain state-machine logic, belongs in `model.go` per file-responsibilities.md `model.go` = "immutable domain objects" + `state.go` = "state transition helpers" (the helpers here are private to `Evaluate`'s computation, not standalone state enum machinery — acceptable in `model.go` alongside the existing `processStateChange`/`UpdateState`) |
| FILE-06 | No package-named catch-all file | PASS | No `transport.go` or `instance.go` catch-all file introduced; each new symbol lands in its designated file (`model.go` for `Transition`/`Evaluate`, `rest.go` for wire fields/`TransformSummary`, `resource.go` for `wantsSchedule`) |

## Immutability / Builder Pattern

| Check | Status | Evidence |
|-------|--------|----------|
| No setters added to `transport.Model` / `instance.RouteModel` | PASS | `git diff 354fb1257..1c1b7798f -- transport/builder.go instance/builder.go` is empty — builder files untouched, no new setters |
| `Model.Evaluate` is pure (no mutation) | PASS | `transport/model.go` — `func (m Model) Evaluate(now time.Time) Transition` takes value receiver, returns a new `Transition` value, mutates nothing |
| Test helpers use Builder pattern, no `*_testhelpers.go` | PASS | `git diff --name-status 354fb1257..1c1b7798f -- '*_test.go'` shows no `*_testhelpers.go` file; all new tests (`evaluate_test.go`, `rest_test.go`, `resource_include_test.go`, `resource_tenant_test.go`, `instance/rest_test.go`) build fixtures via `NewBuilder(...)`/`NewRouteBuilder(...)`/`NewTripScheduleBuilder(...)` |

## Project-Specific Rule Verification

| Rule | Status | Evidence |
|------|--------|----------|
| FR-6.5 duration rule — new fields are `…Seconds uint32`, legacy nanosecond fields untouched | PASS | `transport/rest.go:28-31` (`BoardingWindowSeconds`/`PreDepartureSeconds`/`TravelDurationSeconds`/`CycleIntervalSeconds`, all `uint32`); `CycleInterval time.Duration` field unchanged in place at line 27, still `json:"cycleInterval"`. Same for `instance/rest.go:20-21` (`BoardingWindowSeconds`/`TravelDurationSeconds` added; `BoardingWindow`/`TravelDuration time.Duration` left untouched at their original field names/types) |
| One clock seam — `timeNow` in `transport/scheduler.go:7` | PASS | `grep -n "timeNow" transport/*.go` (non-test) shows exactly two use sites, both reading the same package-level var: `transport/scheduler.go:22` and `transport/rest.go:129` (`m.Evaluate(timeNow().UTC())`). No second `var …Now = time.Now` declared anywhere in the diff. (Pre-existing `time.Now()` direct call at `transport/processor.go:138` is untouched by this diff, as noted in the task scope, and is not part of this branch's changes) |
| Read-only surface — no new mutation endpoints/auth/secrets | PASS | `git diff 354fb1257..1c1b7798f -- transport/resource.go instance/resource.go` shows only a new `?include=schedule` GET-path branch and the FR-6.1 tenant read fix — no new `Methods(http.MethodPost\|Patch\|Delete)` route registration, no new import of any auth/secret-handling package |
| Immutable domain models — no setters on `transport.Model`/`instance.RouteModel` | PASS | Confirmed above (empty `builder.go` diffs) |
| Trip-schedule timestamps carry a stale date; `nextTransitionAt` strictly future | PASS | `transport/model.go` `materializeBoundary` (new) always adds 24h when the boundary would not be strictly after `now` — `if !at.After(now) { at = at.Add(24 * time.Hour) }`; guarded by `TestEvaluate_NextAtIsAlwaysInTheFuture` (`transport/evaluate_test.go:187-206`), which asserts `got.NextAt.After(now)` for every 7-minute tick across a full day |
| Two unreachable tail branches pruned, behavior preserved | PASS (verified independently, not taken on faith) | Manually traced: in the non-midnight-crossing branch, `nextTrip` is always either `inTransitTrip` (whose selection condition `nowTimeOfDay` between its own departure/arrival makes `nowTimeOfDay.Before(arrival)` always true when re-checked) or `futureTrip` (whose selection condition `departure > now` implies `arrival > now` for a same-day trip, so `nowTimeOfDay.Before(arrival)` is again always true) — so the pre-existing `else if futureTrip != nil / else if arrivedTrip != nil / else OutOfService` tail in `354fb1257`'s `processStateChange` (see `git show 354fb1257:.../model.go:198-204`) could never execute. `Model.Evaluate`'s tail correctly collapses to `return Transition{State: OutOfService}`. Guard: pre-existing `transport/state_test.go` passes unedited (`git diff` shows zero changes to that file), and the new `TestEvaluate_StateMatchesProcessStateChange` (`transport/evaluate_test.go:174-184`) additionally pins `Evaluate(now).State == processStateChange(now)` |

## Summary

### Blocking (must fix)

- **DOM-14 (instance/resource.go:115-119)** — `GetInstanceRouteStatusHandler` calls `getInstanceRegistry()` directly instead of through `NewProcessor(...)`, the only such handler-level registry bypass in the codebase. The FR-6.1 tenant-scoping fix this branch lands touches this exact block; it should add a `Processor` method (e.g. `GetInstancesByRoute(routeId uuid.UUID) []TransportInstance`, reading `p.ctx`'s tenant internally) instead of leaving/extending the direct registry call at the handler layer.

### Non-Blocking (should fix)

- **DOM-05 (transport/resource.go:116, instance/resource.go:39)** — Neither package defines a `TransformSlice` function; list handlers call `model.SliceMap(Transform)`/`model.SliceMap(TransformRoute)` inline. This matches `ai-guidance.md`'s own "Useful Composition" example and predates this diff (only the transformer argument changed), so it is not a regression introduced by this branch — flagged as pre-existing guideline-internal inconsistency rather than a new defect.
