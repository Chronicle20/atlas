# Backend Audit — task-278 post-merge commits (map environment object state)

- **Service Path(s):** services/atlas-data, services/atlas-maps, services/atlas-channel
- **Scope:** Go files changed in `9b7817a13..5efddd01b` (4 post-merge commits)
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-09-02
- **Build:** PASS (all three modules)
- **Tests:** PASS (all three modules, no `FAIL` lines; new tests exercised)
- **Overall:** NEEDS-WORK

## Build & Test Results

```
cd services/atlas-data/atlas.com/data && go build ./...    -> exit 0, no output
cd services/atlas-data/atlas.com/data && go test ./... -count=1  -> all ok, no failures
cd services/atlas-maps/atlas.com/maps && go build ./...    -> exit 0, no output
cd services/atlas-maps/atlas.com/maps && go test ./... -count=1  -> all ok, no failures
cd services/atlas-channel/atlas.com/channel && go build ./... -> exit 0, no output
cd services/atlas-channel/atlas.com/channel && go test ./... -count=1 -> all ok, no failures
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| FILE-01..06 | Yes | Every changed package audited (no exemptions) |
| DOM structure (01-05,11,16) | Yes | `services/atlas-data/atlas.com/data/map/object/rest.go` (rest.go, no model.go); `services/atlas-maps/atlas.com/maps/data/map/object/{model.go,rest.go}` (model.go + rest.go, no entity.go/provider.go); `services/atlas-data/atlas.com/data/map/{rest.go,resource.go,processor.go}` (rest.go/processor.go changed, pre-existing model.go/entity.go present but unchanged) |
| SUB-01..04 | Yes (partially) | `services/atlas-maps/atlas.com/maps/map/environment/` has `resource.go`, no `model.go` — sub-domain. Only `processor.go`/`registry.go`/`producer.go` changed; `resource.go` itself untouched in this commit range |
| REST (DOM-06..09,12..15,17..19,32) | Yes | `services/atlas-data/atlas.com/data/map/resource.go` gets a new route + handler `handleGetMapObjectsRequest` |
| Constants reuse (DOM-21) | Checked, N/A | No new type/const duplicates `libs/atlas-constants/`; `field.ObjectKind` reused as-is (`libs/atlas-constants/field/constants.go:19-23`), new fields are plain `uint32 State`/`DefaultState`, not new constant classifications |
| Testing (DOM-10,20,24,33) | Yes | Diff adds `reader_object_test.go`, `resource_object_test.go`, `processor_default_test.go`, extends `consumer_test.go`; `_map.Processor` interface gained `GetObjects` |
| Cache (DOM-29) | Yes | `services/atlas-maps/atlas.com/maps/map/environment/registry.go` — package singleton, only additive field/method in this diff |
| Messaging (DOM-30) | Yes | `services/atlas-maps/atlas.com/maps/map/environment/producer.go` changed |
| Multi-tenancy (DOM-31) | Yes | `services/atlas-data/atlas.com/data/map/rest.go` (rest.go changed); `services/atlas-maps/atlas.com/maps/data/map/object/processor.go` reads tenant via context |
| Deploy & topics (DOM-22,23) | N/A | No `libs/atlas-*` module added, no topic env var added or renamed |
| Runtime safety (DOM-26) | Checked, PASS | `git diff` shows no added bare `go ` statements outside `_test.go` |
| Channel wire values (DOM-25) | Yes | `services/atlas-channel` touched (`consumer.go`, `kafka.go`) |
| Resilience (DOM-27,28) | Checked, N/A | New handler `handleGetMapObjectsRequest` writes `http.StatusNotFound`/`http.StatusBadRequest`, never `http.StatusInternalServerError` — DOM-27 trigger absent; no `model.Decorator` touched — DOM-28 N/A |
| External clients (EXT-01..04) | Yes | `services/atlas-maps/atlas.com/maps/data/map/object/processor.go` calls `requests.DrainProvider` against atlas-data |
| Scaffolding (SCAFFOLD-01..09) | N/A | No new `services/atlas-<svc>/` directory, no new channel Writer/Handler registration, no `routes.conf` change |
| Security (SEC-01..04) | N/A | No token/auth/secret/redirect code touched anywhere in this diff |

## Checklist Results

### services/atlas-data/atlas.com/data/map (`_map`, domain)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-04 | `Transform(Model)(RestModel,error)` in `rest.go` | N/A | `rest.go` changed only to add `Objects` field/reference wiring to the pre-existing `RestModel`; `_map` package's own `Transform`/`TransformSlice` funcs are pre-existing and untouched by this diff |
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | `processor.go` unchanged constructor signature; new `GetObjects`/`objectProvider` methods added at `services/atlas-data/atlas.com/data/map/processor.go:306-317`, same receiver |
| DOM-07 | Handlers pass `d.Logger()` into `NewProcessor` | PASS | `services/atlas-data/atlas.com/data/map/resource.go:275` `NewProcessor(d.Logger(), d.Context(), db).GetObjects(s, mapId)` |
| DOM-09 | Every `Transform(` call site checks error | N/A | New handler never calls `Transform` for objects (object package has no `Transform`, see finding below); no new `Transform(` call site introduced |
| DOM-11 | Providers lazy via `database.Query`/`SliceQuery` | N/A | No `provider.go` changed; `objectProvider` in `processor.go` wraps an already-lazy `s.ByIdProvider` call, consistent with sibling `npcProvider`/`reactorProvider` (unchanged pattern) |
| DOM-12 | No `os.Getenv()` in handlers | PASS | `handleGetMapObjectsRequest` (resource.go:260-286) contains no `os.Getenv` |
| DOM-13 | No cross-domain orchestration in handlers | PASS | Handler only calls `NewProcessor(...).GetObjects` then paginates/marshals — resource.go:260-286 |
| DOM-14 | Handlers call processor methods, not providers | PASS | resource.go:275 calls `.GetObjects(s, mapId)`, not a provider function |
| DOM-15 | No `db.Create`/`Save`/`Delete` in handlers | PASS | Handler is read-only; no write calls in resource.go:260-286 |
| DOM-17 | Domain errors map to specific HTTP status | PASS | Handler maps param-parse failure to 400 (`server.WriteBadRequest`, resource.go:264-267) and any provider error to 404 (resource.go:277-280) — identical shape to sibling `handleGetMapReactorsRequest`; endpoint has no validation/conflict class of its own to mis-map |
| DOM-18 | RestModel implements JSON:API interface | PASS | `services/atlas-data/atlas.com/data/map/object/rest.go:12,16,20` `GetName`/`GetID`/`SetID` |
| DOM-19 | Request models flat | N/A | No request body model added (GET only) |
| DOM-31 | Tenant/trace travel in context only | PASS | `object.RestModel` (rest.go) carries no tenant field; handler uses `d.Context()` only, no tenant query/path param introduced |
| DOM-32 | Routes register via `server.RegisterHandler`/`RegisterInputHandler[T]` | PASS | resource.go:31 `registerGet := rest.RegisterHandler(l)(si)`; new route at resource.go:40 `r.HandleFunc("/{mapId}/objects", registerGet("get_map_objects", handleGetMapObjectsRequest(db)))` — same wrapper as every sibling route in the file |
| DOM-04 (object pkg) | `Transform(Model)(RestModel,error)` in `rest.go` | **FAIL** | `services/atlas-data/atlas.com/data/map/object/rest.go` has `rest.go` (trigger fires) but defines no `Transform` function — only `GetName`/`GetID`/`SetID` (lines 12-22). Mirrors sibling `portal`/`reactor` packages' identical gap, but those are unchanged in this diff; this file is new |
| FILE-01..06 | File placement | PASS | `object/rest.go` holds only `RestModel` + JSON:API methods; no catch-all file introduced |

### services/atlas-data/atlas.com/data/map/mock (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-33 | Mock updated for interface change | PASS | `_map.Processor` gained `GetObjects` (processor.go:33); mock updated at `services/atlas-data/atlas.com/data/map/mock/processor.go:74-79` |

### services/atlas-data/atlas.com/data/map/object (support — has `rest.go`, no `model.go`/`resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-04 | `Transform` in `rest.go` | **FAIL** | See above — `object/rest.go` (23 lines) has no `Transform` |
| DOM-18 | JSON:API interface | PASS | `object/rest.go:12-22` |
| FILE-02 | RestModel/JSON:API methods live in `rest.go` | PASS | All three methods in `rest.go`, no catch-all |
| DOM-20 | Reading `reader_object_test.go` (see below) | **FAIL** | See `_map` test section |

### services/atlas-data/atlas.com/data/map — new tests

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-20 | Table-driven pattern | **FAIL** | `services/atlas-data/atlas.com/data/map/reader_object_test.go:29-70` `TestGetObjects` walks 3 parallel l2-parsing scenarios (numeric, non-numeric, absent) via sequential `if os[N].Name != ...` assertions instead of `tests := []struct{...}` + `t.Run` — this is the textbook table-driven case (same function, parallel inputs, identical assertion shape) |
| DOM-20 | Table-driven pattern | WARN | `resource_object_test.go` `TestMapObjects_Endpoint` (single end-to-end scenario) is not table-driven, but is a single integration scenario with no natural parallel cases — lower confidence finding, recorded non-blocking |
| DOM-10 | Test DB setup calls `RegisterTenantCallbacks` | PASS | Reuses pre-existing `setupStorageTestDB` helper, `services/atlas-data/atlas.com/data/map/storage_test.go:84` `database.RegisterTenantCallbacks(...)` |
| DOM-24 | producertest stub for emit path | N/A | Neither new test file reaches `AndEmit`/`message.Emit`/`producer.Produce` |

### services/atlas-maps/atlas.com/maps/data/map/object (domain — has `model.go`, no `entity.go`/`resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` with `NewBuilder()`/`Build()` | **FAIL** | Package has `model.go` (`services/atlas-maps/atlas.com/maps/data/map/object/model.go:1-16`) but no `builder.go` file exists in the package (`ls` confirms only `model.go processor.go requests.go rest.go`) |
| DOM-04 | `Transform(Model)(RestModel,error)` in `rest.go` | **FAIL** | `services/atlas-maps/atlas.com/maps/data/map/object/rest.go:32` defines only `Extract(rm RestModel) (Model, error)`, no `Transform` in the reverse direction |
| DOM-11 | Providers lazy | N/A | No `provider.go` |
| DOM-16 | `administrator.go` for writes | N/A | Package performs no writes (read-only client) |
| EXT-01 | `SetToOneReferenceID`/`SetToManyReferenceIDs` present | PASS | `rest.go:24-30` both present as no-ops |
| EXT-02 | httptest-backed integration test asserting populated struct | **FAIL** | `services/atlas-maps/atlas.com/maps/data/map/object/` contains no `_test.go` file at all (`ls` confirms `model.go processor.go requests.go rest.go` only). Sibling client `services/atlas-maps/atlas.com/maps/data/map/monster/processor_drain_test.go` has the equivalent httptest-backed test; `object` has none of its own. `map/environment/processor_default_test.go` exercises the client indirectly through `environment.Processor.Set`, but that is a different package, not "the client package or its sibling `_test.go`" |
| EXT-03 | Only genuine 404s map to not-found | PASS | `processor.go:44-52` `GetDefaultState` bubbles any provider error unchanged (via `requests.DrainProvider`) — no blanket "not found" conflation; `ErrUnknownObject` is returned only when the name is absent from a successfully-fetched slice (`processor.go:53`), not from an HTTP error |
| EXT-04 | URL via `requests.RootUrl(<DOMAIN>)` | PASS | `services/atlas-maps/atlas.com/maps/data/map/object/requests.go:16` `requests.RootUrlFor(ctx, "DATA")` — same tenant-aware root-URL resolver used identically by every sibling client in `data/map/` (`reactor`, `script`, `monster`, `info`) |
| FILE-01..06 | File placement | PASS | `model.go`/`processor.go`/`requests.go`/`rest.go` split, no catch-all |
| DOM-31 | Tenant travels in context only | PASS | `processor.go:27` `t: tenant.MustFromContext(ctx)`; `RestModel` (rest.go) carries no tenant field |

### services/atlas-maps/atlas.com/maps/map/environment (sub-domain — has `resource.go`, no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Business logic in own/parent processor, not handler | PASS | New `defaultStateOf`/registry-lookup logic lives in `processor.go:52-66`, not in (unchanged) `resource.go` |
| SUB-02..04 | writes/POST/manual-JSON in `resource.go` | N/A (out of scope) | `resource.go` itself is unchanged in this commit range; not re-audited per task scope instructions |
| DOM-29 | Cache is an application-scoped singleton via accessor, not per-instance state | PASS | `registry.go:31-39` `getRegistry()` uses `sync.Once`; `ProcessorImpl` (processor.go) holds no cached registry field, only calls the package-level `getRegistry()` accessor |
| DOM-30 | DB write + emit atomic via `AndEmit`/`message.Buffer` | PASS (documented exception) | `Set`/`Reset` operate over the in-memory `Registry`, not a DB transaction — `patterns-kafka.md`'s "Operations over non-DB state" exception applies (`registry.go` has no `gorm`/DB import) |
| DOM-33 | Mock updated for interface change | N/A | `environment.Processor`'s `Set`/`Reset` method signatures are unchanged by this diff (only bodies changed); no mock exists or needs updating |
| FILE-01..06 | File placement | PASS | New logic added to `processor.go`/`registry.go`/`producer.go`, matching their existing responsibilities; no catch-all |

### services/atlas-channel/atlas.com/channel/kafka/consumer/map and kafka/message/map (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | Client-interpreted bytes resolved from tenant writer-options, not literals | PASS | `consumer.go`'s new `state := o.State` / `if kind == field.ObjectKindObstacle { state = 0 }` (consumer.go, `handleStatusEventEnvironmentReset`) selects a **value carried by the domain event**, not a hardcoded client opcode; the actual wire encoding still goes through `announceObjectState` → `fieldcb.SetObjectStateWriter`, unchanged in this diff |
| DOM-24 | producertest stub for emit path | N/A | `handleStatusEventEnvironmentReset` emits via `writer.Producer` (socket write), not `producer.Produce`/`message.Emit`/`AndEmit`; no Kafka emit path reached |
| DOM-20 | Table-driven tests | WARN | New `TestHandleStatusEventEnvironmentReset_RestoresCarriedDefaultState` (`consumer_test.go`, appended after line 1191) is a single scenario, not table-driven — file already has a table-driven test elsewhere (`TestAnnounceObjectState_WriterSelection`, line 807-843) for comparison, but this new test asserts one distinct cross-service contract and isn't a natural parallel-case candidate; non-blocking |
| FILE-01..06 | File placement | PASS | `kafka.go` in both services holds only message structs (`EnvironmentObject`, `EnvironmentReset`), no catch-all |
| DOM-31 | Tenant travels in context only | PASS | `EnvironmentObject`/`EnvironmentReset` (both `kafka.go`s) carry no tenant field, only `Kind`/`Name`/`State` |

## Not evaluable from the diff

- SUB-02/03/04 for `services/atlas-maps/atlas.com/maps/map/environment/resource.go`: file is unchanged in this commit range; would require reading it in full (out of the stated review surface — task explicitly excludes re-auditing unchanged code).
- DOM-11 (`objectProvider`/`npcProvider` laziness pattern) for `services/atlas-data/atlas.com/data/map/processor.go`: the underlying `s.ByIdProvider` implementation lives in `storage.go`, which is unchanged and outside the diff; assumed lazy by consistency with the identical pre-existing `npcProvider`/`reactorProvider` shape, not independently re-verified against `storage.go`.

## Summary

### Blocking (must fix)
- DOM-01: `services/atlas-maps/atlas.com/maps/data/map/object/model.go` has `model.go` with no `builder.go`/`NewBuilder()`.
- DOM-04: `services/atlas-maps/atlas.com/maps/data/map/object/rest.go` has no `Transform(Model)(RestModel,error)` (only `Extract`).
- DOM-04: `services/atlas-data/atlas.com/data/map/object/rest.go` has no `Transform(Model)(RestModel,error)`.
- EXT-02: `services/atlas-maps/atlas.com/maps/data/map/object/` has zero test files — no httptest-backed integration test proving the client unmarshals a representative atlas-data JSON:API fixture into a populated `Model`.
- DOM-20: `services/atlas-data/atlas.com/data/map/reader_object_test.go:29-70` `TestGetObjects` tests 3 parallel l2-parsing input scenarios via sequential `if` checks instead of the required `tests := []struct{...}` + `t.Run` table-driven pattern.

### Non-Blocking (should fix)
- DOM-20 (WARN): `resource_object_test.go` `TestMapObjects_Endpoint` is a single-scenario test, not table-driven.
- DOM-20 (WARN): `consumer_test.go` `TestHandleStatusEventEnvironmentReset_RestoresCarriedDefaultState` is a single-scenario test, not table-driven.
- DOM-04 (pre-existing, out of scope): sibling `services/atlas-data/atlas.com/data/map/portal/rest.go` and `.../reactor/rest.go` have the same missing-`Transform` shape as the new `object/rest.go`; unchanged by this diff, noted for awareness only.
- EXT-02 (pre-existing, out of scope): sibling `services/atlas-maps/atlas.com/maps/data/map/reactor/` and `.../script/` client packages also have no dedicated httptest-backed test of their own (only `monster` and `info` do); unchanged by this diff, noted for awareness only.
