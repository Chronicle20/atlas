# Backend Audit — task-240 (atlas-character / atlas-configurations / atlas-data / libs/atlas-constants / libs/atlas-packet)

- **Service Path(s):** `services/atlas-character`, `services/atlas-configurations`, `services/atlas-data`, `libs/atlas-constants`, `libs/atlas-packet`
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-19
- **Range:** merge-base `d9ec287b8`..`3bc7ebd21`
- **Build:** PASS
- **Tests:** All passing (see below); 0 failed
- **Overall:** NEEDS-WORK

## Build & Test Results

```
cd services/atlas-character/atlas.com/character && go build ./...   -> ok
cd services/atlas-configurations/atlas.com/configurations && go build ./... -> ok
cd services/atlas-data/atlas.com/data && go build ./...             -> ok
cd libs/atlas-constants && go build ./...                            -> ok
cd libs/atlas-packet && go build ./...                               -> ok

go test ./... -count=1 (all five modules): all packages `ok` or `[no test files]`,
no failures. Notable: atlas-character/pending_change took 179.6s (pre-existing,
unrelated to this diff) which forced running that module's tests outside the
default 2-minute timeout.
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | Yes | `equipslot` package (new) has `model.go`, `entity.go`, `rest.go` |
| FILE placement (FILE-01..06) | Yes | Every changed Go package |
| SUB (SUB-01..04) | Yes | `cashpackage` has `resource.go`, no `model.go` |
| REST (DOM-06..09,12..15,17..19,32) | Yes | `equipslot` and `cashpackage` both have `resource.go`/`rest.go`/`processor.go` |
| Constants reuse (DOM-21) | Yes | `libs/atlas-constants/inventory/slot/constants.go` adds a `pendant2` slot entry; `equipslot` adds new domain types |
| Testing (DOM-10,20,24,33) | Yes | Diff adds/changes `_test.go` files in `equipslot`, `cashpackage`, `libs/atlas-packet` |
| Cache (DOM-29) | No | No `cache.go`, no cached processor/struct state in any changed package |
| Messaging (DOM-30) | No | No `producer.go`; no `AndEmit`/`message.Emit`/`producer.ProviderImpl` call sites added by this diff (pre-existing `producer.ProviderImpl` lines in `data/processor.go` untouched by the hunks) |
| Multi-tenancy (DOM-31) | Yes | `equipslot/rest.go`, `equipslot/processor.go` reads `tenant.MustFromContext` |
| Migration hygiene (DOM-34,35) | No | Diff adds new packages; does not move/extract symbols between service and `libs/atlas-*` |
| Deploy & topics (DOM-22,23) | No | No new `libs/atlas-*` module added; no Kafka topic env var added/renamed |
| Runtime safety (DOM-26) | No | No `go func`/bare `go` statement added by this diff (grepped every changed non-test `.go` file's added lines) |
| Channel wire values (DOM-25) | Fired (family), out of scope | `libs/atlas-packet` and a template JSON are touched, but this is the packet coverage matrix / IDA export, explicitly owned by the concurrent `packet-completeness-critic` per this review's instructions — not re-audited here |
| Resilience (DOM-27,28) | No | Neither `equipslot/resource.go` nor `cashpackage/resource.go` writes `http.StatusInternalServerError` literally; no `model.Decorator`/enrichment path touched |
| External clients (EXT-01..04) | No | No `requests.RootUrl`/`requests.GetRequest[T]`/`requests.PostRequest[T]` call sites in any changed package |
| Scaffolding (SCAFFOLD-01..09) | No | No new `services/atlas-<svc>/` directory; no channel Writer/Handler registration change in scope (owned by concurrent reviewer); `deploy/shared/routes.conf` untouched |
| Security (SEC-01..04) | No | None of the touched services handle auth/tokens/redirects/secrets in this diff |

## Checklist Results

### equipslot (domain package — services/atlas-character)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` exists with `NewBuilder()` | FAIL | No `builder.go` in `services/atlas-character/atlas.com/character/equipslot/` — `ls` confirms absence |
| DOM-02 | `Model.ToEntity()` in `entity.go` | FAIL | `grep -rn "ToEntity"` over the package returns no match; `entity.go` (1-41) has no such method |
| DOM-03 | `Make(Entity) (Model, error)` in `entity.go` | FAIL | `grep -rn "func Make("` over the package returns no match; the package instead has `modelFromEntity(e Entity) Model` in `model.go:33-39` — wrong file, wrong signature (no error return) |
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | PASS | `rest.go:32` `func Transform(m Model) (RestModel, error)` |
| DOM-05 | `TransformSlice` in `rest.go`, used by list handlers | FAIL | `rest.go` defines no `TransformSlice`; `resource.go:41` uses `model.SliceMap(Transform)(...)` instead — the required function does not exist |
| DOM-11 | Providers use `database.Query`/`SliceQuery`, not eager `FixedProvider` | N/A | Rule's own trigger ("package has `provider.go`") does not fire — no `provider.go` file exists in this package (see FILE-05 finding: the reader lives in `administrator.go` instead) |
| DOM-16 | `administrator.go` holds writes | PASS | `administrator.go:24-61` defines `Extend`, the write path, called from `processor.go:38` |
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | `processor.go:26` `func NewProcessor(l logrus.FieldLogger, ...)` |
| DOM-07 | Handlers pass `d.Logger()` to `NewProcessor` | PASS | `resource.go:35`, `resource.go:63` |
| DOM-08 | POST routes use `RegisterInputHandler[T]` | PASS | `resource.go:27` `rest.RegisterInputHandler[ExtendInputRestModel](l)(db)(si)("extend_equip_slot", ...)` |
| DOM-09 | Every `Transform(` call site checks its error | PASS | `resource.go:41-46`, `resource.go:85-90` both check `err` |
| DOM-12 | No `os.Getenv()` in `resource.go` | PASS | `grep -n "os.Getenv" resource.go` — zero matches |
| DOM-13 | No cross-domain orchestration in handlers | PASS | `resource.go` handlers call only this package's own `Processor` |
| DOM-14 | Handlers call processor methods, never providers | PASS | `resource.go:35,63,72` all call `NewProcessor(...).GetActive`/`.Extend` |
| DOM-15 | No `db.Create`/`db.Save`/`db.Delete` in `resource.go` | PASS | `grep -n "db.Create\|db.Save\|db.Delete" resource.go` — zero matches |
| DOM-17 | Domain errors map to 400/404/409 | WARN | `resource.go` routes every processor error through `server.WriteErrorResponse` uniformly; no explicit not-found (404) branch is visible for `Extend`/`GetActive` failures (they surface as whatever `WriteErrorResponse` derives from the raw `gorm` error) — not evaluable further without reading `WriteErrorResponse`'s classification (see Not evaluable) |
| DOM-18 | `RestModel`s implement `GetName/GetID/SetID` | PASS | `rest.go:19-30` (`RestModel`), `rest.go:57-67` (`ExtendInputRestModel`) |
| DOM-19 | Request models flat, no nested Data/Type/Attributes | PASS | `rest.go:50-55` `ExtendInputRestModel` is flat |
| DOM-32 | Routes resolve to `server.RegisterHandler`/`RegisterInputHandler[T]` | PASS | `resource.go:24,27` use `rest.RegisterHandler`/`rest.RegisterInputHandler[T]`, which delegate to `server.RetrieveSpan`/`server.ParseTenant` — traced to `services/atlas-character/atlas.com/character/rest/handler.go:68-93`; no manual tenant-header parsing, no custom error writer |
| DOM-31 | Tenant/trace never a REST field | PASS | `rest.go:12-17,50-55` — no tenant/trace field on either `RestModel`; `processor.go:31` reads tenant only via `tenant.MustFromContext(ctx)` |
| FILE-01 | Processor interface/ctor/methods in `processor.go` | PASS | `processor.go:14-53`, entire `Processor`/`ProcessorImpl` definition confined to this file |
| FILE-02 | RestModel/Transform/JSON:API methods in `rest.go` | PASS | `rest.go` (entire file) |
| FILE-03 | Cross-service request funcs in `requests.go` | N/A | No `requests.RootUrl`/`GetRequest`/`PostRequest` call sites in the package |
| FILE-04 | Entity struct/Migration/TableName in `entity.go` | PASS | `entity.go:17,37,39` |
| FILE-05 | Builder/Model/writes/readers each in their own file | FAIL | Builder missing entirely (see DOM-01); reader `GetActive` (`administrator.go:66-77`) is in `administrator.go`, not `provider.go` — no `provider.go` file exists |
| FILE-06 | No file bundling ≥2 responsibilities | FAIL | `administrator.go` bundles the "writes" responsibility (`Extend`, lines 24-61) AND the "readers" responsibility (`GetActive`, lines 66-77) in a single file |
| DOM-20 | Table-driven tests (`tests := []struct{...}` + `t.Run`) | WARN | `administrator_test.go` and `resource_test.go` use sequential `t.Run(...)` blocks (e.g. `administrator_test.go:60,74,90,108,126,133,145,164`) but never declare a `tests := []struct{...}` table — deviates from the literal DOM-20 shape, though each case is still isolated via `t.Run` |
| DOM-10 | Test DB setup calls `database.RegisterTenantCallbacks` | FAIL | `administrator_test.go:17-34` `testDB(t)` opens a GORM DB (`gorm.Open`, line 25) and never calls `database.RegisterTenantCallbacks`; `resource_test.go` reuses the same `testDB` helper (`resource_test.go:67,108`). Mitigating factor noted but not exculpatory: `administrator.go` filters by `tenant_id` explicitly in every query rather than relying on GORM's tenant-scoping callback, so the specific risk the rule targets does not manifest here — but the rule's own trigger ("tests open a GORM DB directly") and pass criterion (the callback is called) are unmet regardless |
| DOM-24 | Emit-reaching tests stub the producer | N/A | No `AndEmit`/`message.Emit`/`producer.Produce` call sites reachable from `equipslot`'s test entry points |
| DOM-33 | Interface change updates all mocks | N/A | `Processor` is a brand-new interface introduced by this diff, not a change to an existing one |
| DOM-21 | No redeclaration of a shared constant | PASS | `libs/atlas-constants/inventory/slot/constants.go:69` adds `{Type: "pendant2", Position: -59}` to the existing shared `Slots` table; `equipslot`/its tests resolve the slot via `slot.GetSlotByType("pendant2")` (`administrator_test.go:41`), not a local literal |

### cashpackage (sub-domain package — services/atlas-data)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Business logic not in handler | FAIL | `resource.go:39-40` (`handleGetCashPackagesRequest`) and `resource.go:57-58` (`handleGetCashPackageRequest`) construct `NewStorage(...)` and call `s.AllPagedProvider(...)`/`s.GetById(...)` directly in the handler; the package's own `Processor` (`processor.go:14-17`) exposes no read method at all — reads never route through it |
| SUB-02 | No `db.Create`/`db.Save`/`db.Delete` in `resource.go` | PASS | `grep -n "db.Create\|db.Save\|db.Delete" resource.go` — zero matches |
| SUB-03 | POST routes use `RegisterInputHandler[T]` | N/A | `resource.go:23-24` registers only `http.MethodGet` routes — no POST |
| SUB-04 | No manual JSON parsing in `resource.go` | PASS | `grep -n "json.NewDecoder\|json.Unmarshal\|io.ReadAll" resource.go` — zero matches |
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | `processor.go:25` `func NewProcessor(l logrus.FieldLogger, ...)` |
| DOM-07 | Handlers pass `d.Logger()` to `NewProcessor`/`NewStorage` | PASS | `resource.go:39,57` pass `d.Logger()` |
| DOM-09 | Every `Transform(` call site checks its error | N/A | `resource.go` never calls a package-local `Transform(` — it delegates entirely to `document.Storage`'s internal marshaling |
| DOM-12 | No `os.Getenv()` in `resource.go` | PASS | `grep -n "os.Getenv" resource.go` — zero matches |
| DOM-13 | No cross-domain orchestration in handlers | PASS | Handlers stay within `cashpackage`/`document` |
| DOM-14 | Handlers call processor methods, never providers directly | FAIL | Same evidence as SUB-01: `resource.go:39-40,57-58` bypass `Processor` and call `document.Storage` (the package's de facto provider) directly. No documented circular-dependency exception applies — this is same-package read access, not a cross-domain circular dependency (`anti-patterns.md` "Exception: Cross-Domain Read-Only Views with Circular Dependencies", lines 99-129, requires a genuine circular package dependency, none exists here) |
| DOM-15 | No `db.Create`/`db.Save`/`db.Delete` in `resource.go` | PASS | Same evidence as SUB-02 |
| DOM-17 | Domain errors map to 400/404/409 | PASS | `resource.go:34-37` (400 via `WriteBadRequest` on bad page params), `resource.go:61-62` (404 via `w.WriteHeader(http.StatusNotFound)` on miss) |
| DOM-18 | RestModel implements `GetName/GetID/SetID` | PASS | `rest.go:12-27` |
| DOM-19 | Request models flat | N/A | No request/input model defined in this package (read-only surface) |
| DOM-32 | Routes resolve to `server.RegisterHandler`/`RegisterInputHandler[T]` | PASS | `resource.go:20` `rest.RegisterHandler(l)(si)`, traced to `services/atlas-data/atlas.com/data/rest/handler.go:24` `var RegisterHandler = server.RegisterHandler` |
| DOM-31 | Tenant/trace never a REST field | PASS | `rest.go:7-10` — no tenant/trace field |
| FILE-01 | Processor interface/ctor/methods in `processor.go` | PASS | `processor.go:14-61` |
| FILE-02 | RestModel/Transform/JSON:API methods in `rest.go` | PASS | `rest.go` (entire file) |
| FILE-03 | Cross-service request funcs in `requests.go` | N/A | No `requests.RootUrl`/`GetRequest`/`PostRequest` call sites |
| FILE-04 | Entity struct/Migration/TableName in `entity.go` | N/A | Package has no GORM entity of its own — it persists through `atlas-data/document`'s generic `Storage`/`Registry`; none of `type entity struct`/`func Migration(`/`TableName()` appear anywhere in the package to grade as misplaced |
| FILE-05 | Builder/Model/writes/readers each in their own file | N/A | Package defines no domain `Model` distinct from `RestModel`, no `Create*`/`Update*`/`Delete*` writes, no `database.Query`/`SliceQuery` readers — it is built entirely on the shared `document.Storage` abstraction, which is out of this diff's scope |
| FILE-06 | No file bundling ≥2 responsibilities | PASS | Files are `processor.go`, `reader.go`, `registry.go`, `resource.go`, `rest.go`; `reader.go` is a genuine single-purpose XML-ingestion utility (not one of the enumerated responsibilities bundled with another) |
| DOM-20 | Table-driven tests | FAIL | `reader_test.go:30-65` (`TestReadCashPackages`) is a single flat test with a manual nested `for` loop over a `want` slice — no `t.Run`, no `tests := []struct{...}` table |
| DOM-20 | Table-driven tests | WARN | `resource_test.go:45-120` (`TestCashPackageResourceIntegration`) uses sequential `t.Run(...)` sub-tests (lines 54, 73, 97, 109) but never declares a `tests := []struct{...}` table |
| DOM-10 | Test DB setup calls `database.RegisterTenantCallbacks` | PASS | `resource_test.go:138` `database.RegisterTenantCallbacks(logrus.StandardLogger(), db)` after `gorm.Open()` (line 123) |
| DOM-24 | Emit-reaching tests stub the producer | N/A | No `AndEmit`/`message.Emit`/`producer.Produce` call sites reachable from this package's tests |
| DOM-33 | Interface change updates all mocks | N/A | `Processor` is new in this diff |
| DOM-21 | No redeclaration of a shared constant/type | PASS | `RestModel`, `Processor` etc. are new domain-specific types for a new WZ document kind — no existing `libs/atlas-constants/` equivalent found for "cash package" data |
| EXT-01..04 | External atlas-service client rules | N/A | No `requests.*Request[T]` call site added |

## Not evaluable from the diff

- DOM-17 (`equipslot`): whether `server.WriteErrorResponse` correctly classifies a `gorm.ErrRecordNotFound`-adjacent error from `Extend`/`GetActive` into 404 (vs. a generic 500/503) would require reading `WriteErrorResponse`'s implementation in `libs/atlas-rest/server`, which is outside the changed-file surface and not itself touched by this diff.
- Channel wire values (DOM-25) / packet coverage matrix content in `libs/atlas-packet/cash/*`, `field/clientbound/*`, and `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`: explicitly out of this review's scope per instructions (owned by the concurrent `packet-completeness-critic`).

## Summary

### Blocking (must fix)
- DOM-01: `equipslot` has no `builder.go` / `NewBuilder()` (`services/atlas-character/atlas.com/character/equipslot/` — file absent).
- DOM-02: `equipslot/entity.go` has no `Model.ToEntity()` method.
- DOM-03: `equipslot/entity.go` has no `func Make(Entity) (Model, error)`; `model.go:33` has a same-purpose but non-conforming `modelFromEntity` in the wrong file with the wrong signature.
- DOM-05: `equipslot/rest.go` has no `TransformSlice`; `resource.go:41` uses `model.SliceMap(Transform)` instead.
- FILE-05 / FILE-06: `equipslot/administrator.go:66-77` bundles a reader (`GetActive`) into the writes file; no `provider.go` exists.
- SUB-01 / DOM-14: `cashpackage/resource.go:39-40,57-58` call `document.Storage` (`NewStorage(...).AllPagedProvider`/`.GetById`) directly from the handler; the package's `Processor` interface (`processor.go:14-17`) has no read method at all.
- DOM-10: `equipslot/administrator_test.go:17-34` (`testDB`, reused by `resource_test.go`) opens a GORM DB without `database.RegisterTenantCallbacks`.
- DOM-20: `cashpackage/reader_test.go:30-65` is not table-driven / has no `t.Run` at all.

### Non-Blocking (should fix)
- DOM-17: `equipslot/resource.go` has no visible not-found (404) branch distinct from its generic `WriteErrorResponse` path for `Extend`/`GetActive` failures — flagged WARN pending confirmation of `WriteErrorResponse`'s own classification (see Not evaluable).
- DOM-20 (WARN): `equipslot/administrator_test.go`, `equipslot/resource_test.go`, and `cashpackage/resource_test.go` use `t.Run` sub-tests but not the literal `tests := []struct{...}` table shape.

## Not evaluable from the diff
See above.
