# Backend Audit — task-292-map-definition-field-split

- **Service Paths:** `services/atlas-data`, `services/atlas-maps`
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-09-03
- **Build:** PASS (per `tools/verify.sh`, not re-run — see instructions)
- **Tests:** PASS (per `tools/verify.sh`, not re-run — see instructions)
- **Overall:** NEEDS-WORK

## Scope

Diff `9613e7259..9f6679907` restricted to `services/atlas-data` and
`services/atlas-maps`, per instructions. No `libs/` changes. Reviewed every
changed file plus the minimum targeted lookups needed to resolve rule
applicability (sibling `rest.go` files to compare `Transform` usage, the
`document.Registry` implementation backing the new object registry, and the
`paginate`/`rest.RegisterHandler` aliases).

## Build & Test Results

Not re-run per instruction — `tools/verify.sh` (flagless) already confirmed
green on this HEAD, both modules under `-race`. This audit spent its budget on
guideline conformance only.

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | Yes | `field/rest.go`, `map/object/rest.go` are new; `map/processor.go`, `map/reader.go` changed |
| FILE placement (FILE-01..06) | Yes | Every changed Go package |
| SUB sub-domain (SUB-01..04) | Yes | `atlas-maps/field` has `resource.go`, no `model.go` |
| REST (DOM-06..09,12..15,17..19,32) | Yes | `field/resource.go`, `map/resource.go`, `map/rest.go` register/serve routes |
| Constants reuse (DOM-21) | Yes | New types `MapObjectDefinition`, `ObjKindEnvironment`/`ObjKindObstacle` const block |
| Testing (DOM-10,20,24,33) | Yes | New `_test.go` files; `character.Processor` and `map._map.Processor` interfaces gained methods |
| Cache (DOM-29) | Yes | `object_registry.go` is a new package-level singleton registry |
| Messaging (DOM-30) | No | No `AndEmit`/`message.Emit`/`producer.ProviderImpl` call sites in the diff (grep confirmed) |
| Multi-tenancy (DOM-31) | Yes | `field/resource.go`, `field/registry.go` read/pass tenant state; new `/api/fields` route |
| Migration hygiene (DOM-34,35) | No | No symbol moved between service and `libs/atlas-*` |
| Deploy & topics (DOM-22,23) | No | No `libs/atlas-*` module added, no Kafka topic env var added |
| Runtime safety (DOM-26) | Yes (whole-repo guard) | `tools/goroutine-guard.sh` run — exit 0, no bare `go` statements introduced |
| Channel wire values (DOM-25) | No | Diff does not touch `atlas-channel`/`atlas-packet`; no client-interpreted byte in the new event/REST surface |
| Resilience (DOM-27,28) | No | No changed handler writes `http.StatusInternalServerError`; no `model.Decorator`/enrichment path changed |
| External clients (EXT-01..04) | No | No `requests.RootUrl`/`GetRequest[T]`/`PostRequest[T]` call added |
| Scaffolding (SCAFFOLD-01..09) | No | No `services/atlas-<svc>/` added, no channel Writer/Handler registered, `routes.conf` untouched |
| Security (SEC-01..04) | No | Neither service touches auth/tokens/redirects/secrets in this diff |

## Checklist Results

### `atlas-maps/field` (sub-domain: `resource.go`, no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | FAIL | `services/atlas-maps/atlas.com/maps/field/rest.go` has no `func Transform(`. A natural `Model` exists (`character.FieldOccupancy`) but is converted to `RestModel` inline in the handler instead. |
| DOM-05 | `TransformSlice` used by list handlers, no inline loop in `resource.go` | FAIL | `services/atlas-maps/atlas.com/maps/field/resource.go:93-113` — hand-builds `RestModel` in a `for _, o := range occ` loop directly in the handler, with filtering interleaved. |
| DOM-07 | Handler passes `d.Logger()` into `NewProcessor` | PASS | `services/atlas-maps/atlas.com/maps/field/resource.go:91` — `character.NewProcessor(d.Logger(), d.Context())`. |
| DOM-08 | POST/PATCH via `RegisterInputHandler[T]` | N/A | Package registers only `GET /fields` — `services/atlas-maps/atlas.com/maps/field/resource.go:28`. |
| DOM-09 | Every `Transform(` call site checks error | N/A | No `Transform` call site exists (see DOM-04). |
| DOM-12 | No `os.Getenv()` in handlers | PASS | `services/atlas-maps/atlas.com/maps/field/resource.go` — no `os` import. |
| DOM-13 | No cross-domain orchestration in handler | WARN | `handleGetFields` performs filtering, sorting, and RestModel construction itself (lines 91-127) rather than delegating to a processor — the orchestration concern DOM-13 targets, though the cross-domain call itself (`character.NewProcessor(...)`) is a single, legitimate parent-domain lookup. |
| DOM-14 | Handler calls processor methods, not providers | PASS | `character.NewProcessor(d.Logger(), d.Context()).GetFieldsWithCharacters(t)` — `field/resource.go:91`, a `Processor` method, not a provider. |
| DOM-15 | No `db.Create`/`Save`/`Delete` in handler | PASS | No GORM writes present; package has no DB access at all. |
| DOM-17 | Domain errors map to specific HTTP status | PASS | Only failure paths are `400` for bad `page[...]`/`filter[...]` values — `field/resource.go:73,87`; no domain error path exists otherwise. |
| DOM-18 | REST model implements JSON:API interface | PASS | `services/atlas-maps/atlas.com/maps/field/rest.go:22-33` — `GetID`/`GetName`/`SetID`. |
| DOM-19 | Request models flat | N/A | No request model — GET-only endpoint. |
| DOM-31 | Tenant travels in context only | PASS | `field/rest.go:13-20` — `RestModel` has no tenant field; `field/resource.go:77` reads tenant via `tenant.MustFromContext(d.Context())`, never from a query/path param. Two explicit tests assert cross-tenant isolation: `field/resource_test.go:105-143` (`TestGetFieldsTenantIsolation`). |
| DOM-32 | Routes register via `server.RegisterHandler`/`RegisterInputHandler[T]` | PASS | `field/resource.go:28` uses `rest.RegisterHandler`, aliased directly to `server.RegisterHandler` at `services/atlas-maps/atlas.com/maps/rest/handler.go:28`. |
| SUB-01 | Business logic in package's own or parent processor, not handler | FAIL | `field/resource.go:93-127` — filtering by world/channel/map, sorting, and RestModel construction all execute inside `handleGetFields` itself; the package has no `processor.go` of its own, and the parent (`character.Processor`) only supplies the raw occupancy snapshot. |
| SUB-02 | Writes via administrator, no `db.Create`/`Save` in `resource.go` | PASS | No writes in this package. |
| SUB-03 | POST via `RegisterInputHandler[T]` | N/A | No POST route. |
| SUB-04 | No manual JSON parsing in `resource.go` | PASS | No `json.NewDecoder`/`Unmarshal`/`io.ReadAll` in `field/resource.go`. |

### `atlas-maps/map/character` (domain package, existing `model.go` unaffected by this diff; interface/registry methods added)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-29 | Cache/registry is an application-scoped singleton via `GetCache()`-equivalent | PASS | `services/atlas-maps/atlas.com/maps/map/character/registry.go:18-29` — package-level `registry`/`once sync.Once`, accessed via `getRegistry()`; `GetFieldsWithCharacters` (added lines 116-131) reuses the existing singleton, not a per-processor cache. |
| DOM-31 | Tenant travels in context only on the public surface | PASS (scope-limited) | `GetFieldsWithCharacters(t tenant.Model)` is an unexported-package-internal `Processor` method, not a REST model/request/path/query field — in scope per the DOM-31 "internal function parameter" carve-out (`patterns-multitenancy-context.md` §Scope). The tenant value itself originates from `tenant.MustFromContext(d.Context())` in the caller (`field/resource.go:77`). |
| DOM-33 | Interface change updates every mock | PASS | `character.Processor` gained `GetFieldsWithCharacters`; all three mocks updated in-diff: `services/atlas-maps/atlas.com/maps/map/character/mock/processor.go:46-51`, `services/atlas-maps/atlas.com/maps/map/monster/processor_test.go:661-663`, `services/atlas-maps/atlas.com/maps/map/processor_test.go:83-85`. |
| DOM-20 | Table-driven tests | WARN | `services/atlas-maps/atlas.com/maps/map/character/registry_test.go:17-119` — uses `t.Run` subtests but not the `tests := []struct{...}` shape; each subtest has distinct setup so a literal table would be awkward, but the guideline's stated shape is not followed. |

### `atlas-data/map` (domain package: `model.go`, `entity.go`, `rest.go`, `processor.go`, `reader.go` all pre-exist; `object`-related additions layered in)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-07 | Handler passes `d.Logger()` into `NewProcessor` | PASS | `services/atlas-data/atlas.com/data/map/rest.go:275` — `NewProcessor(d.Logger(), d.Context(), db).GetObjects(s, mapId)`. |
| DOM-09 | Every `Transform(` call site checks error | N/A | No `Transform` call site added; `GetObjects` returns `[]object.RestModel` built by the reader, not via a `Transform` call. |
| DOM-11 | Providers lazy via `database.Query`/`SliceQuery` | N/A | No new `provider.go`; `objectProvider` (added `map/processor.go:306-312`) wraps an already-fetched `Storage.ByIdProvider` result in `model.FixedProvider` — but this pattern is identical to the pre-existing `npcProvider`/`reactorProvider` siblings in the same file, not a new deviation introduced by this diff. |
| DOM-13/14/15 | Handler orchestration/writes | PASS | `handleGetMapObjectsRequest` (`map/rest.go:263-288`) calls `NewProcessor(...).GetObjects(s, mapId)`, no direct provider/db access, no writes. |
| DOM-17 | Domain errors map to specific status | PASS (consistent with pre-existing siblings) | `map/rest.go:276-280` returns `404` on any `GetObjects` error, identical to the unmodified `GetReactors`/`GetNpcs` handlers in the same file — not a new deviation. |
| DOM-18 | REST model implements JSON:API interface | PASS | `services/atlas-data/atlas.com/data/map/object/rest.go:20-31`. |
| DOM-21 | No redeclaration of a shared `atlas-constants` type | PASS | `grep -rli obstacle libs/atlas-constants/` returned no hits; `ObjKindEnvironment`/`ObjKindObstacle` (`map/object_registry.go:17-20`) have no shared-library equivalent. World/channel/map IDs in `field/resource.go` correctly reuse `world.Id`/`channel.Id`/`_map.Id` from `libs/atlas-constants` rather than redeclaring. |
| DOM-29 | Registry is an application-scoped singleton | PASS | `services/atlas-data/atlas.com/data/map/object_registry.go:34-44` — package-level `moRg`/`moOnce sync.Once`, `GetMapObjectRegistry()` accessor; the underlying `document.Registry[I,M]` (`services/atlas-data/atlas.com/data/document/registry.go:10-20`, unchanged) provides per-tenant `sync.RWMutex` guarding. |
| DOM-31 | Tenant travels in context only, internal params exempt | PASS | `InitObj(t tenant.Model, dir string)` / `ResolveObjKind(t tenant.Model, ...)` (`map/object_registry.go:50,86`) are internal functions, not REST-surface — in scope per the DOM-31 internal-parameter carve-out; callers derive `t` from `tenant.MustFromContext` (`data/processor.go` diff, `workers/mapw.go` diff). |
| DOM-33 | Interface change updates every mock | PASS | `Processor.GetObjects` added; mock updated same diff: `services/atlas-data/atlas.com/data/map/mock/processor.go:74-79`. |
| DOM-20 | Table-driven tests | WARN | `services/atlas-data/atlas.com/data/map/object_registry_test.go:32-108` and `reader_test.go:872-980` (new `TestGetObjects*` functions) use plain `func Test...(t *testing.T)` bodies, not `tests := []struct{...}` + `t.Run`. `reader_test.go:618-633` (`TestReactorDelayMillis`, pre-existing pattern reused) does follow the table-driven shape correctly. |

### `atlas-data/map/object` (support package: `rest.go` only)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | FAIL | `services/atlas-data/atlas.com/data/map/object/rest.go` — no `func Transform(`. `RestModel` is built directly by `getObjects` in `map/reader.go:350-401`. This mirrors the pre-existing, unmodified `portal`/`reactor`/`npc`/`monster` sibling packages (none of which define `Transform` either), but per the audit mindset prevalence does not exempt a new package from the same rule. |
| DOM-05 | `TransformSlice` used by list handler | N/A | No inline `Transform`-loop exists in `map/resource.go`'s object handler — `GetObjects` returns pre-built values from storage, not a hand-rolled per-item transform in the handler. The absence of `Transform` itself is captured under DOM-04, not DOM-05. |
| FILE-02 | `RestModel`/`GetName`/`GetID`/`SetID` all live in `rest.go` | PASS | `services/atlas-data/atlas.com/data/map/object/rest.go:6-31` — everything present, single file. |
| FILE-06 | No catch-all file carrying ≥2 responsibilities | PASS | Package has exactly one file, `rest.go`, holding only the REST-model responsibility. |

### FILE-* — every changed package (domain, sub-domain, support alike)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor interface/constructor/methods in `processor.go` | PASS | `atlas-data/map/processor.go` (`GetObjects`, `objectProvider` added at lines 306-315) and `atlas-maps/map/character/processor.go` (`GetFieldsWithCharacters` at lines 54-56) — both additions land in the existing `processor.go`, not a stray file. |
| FILE-02 | RestModel/Transform/JSON:API methods in `rest.go` | PASS (field, object) / see DOM-04 for the Transform gap | `field/rest.go`, `object/rest.go` — RestModel + `GetName`/`GetID`/`SetID` correctly placed; `Transform` itself is simply absent (DOM-04 finding above), not misplaced. |
| FILE-03 | Cross-service request functions in `requests.go` | N/A | No `requests.RootUrl`/`GetRequest`/`PostRequest` call added anywhere in the diff. |
| FILE-04 | Entity struct/Migration/TableName in `entity.go` | N/A | No new entity/table added — map objects are stored as an existing JSON document field (`RestModel.Objects`), not a new GORM entity. |
| FILE-05 | Builder/Model/writes/readers placed per table | PASS | No new `Model`, `Builder`, or write path introduced; `object_registry.go`'s `MapObjectDefinition` is a value type for the registry, not a domain `Model`. |
| FILE-06 | No package-named catch-all file | PASS | `object_registry.go` holds a single cohesive responsibility (the obstacle registry) — a "genuine single-purpose utility" per the FILE-06 exception, matching the pre-existing `model_registry.go`/`string_registry.go` naming convention in the same package. `field/resource.go` and `field/rest.go` are each single-responsibility (handler vs. REST model) even though the handler additionally does the transform work that should live in `rest.go` (captured as DOM-04/05/SUB-01 above, not a FILE-06 finding). |

## Security Review

SEC-* family did not trigger — neither `atlas-data` nor `atlas-maps` changes in this diff touch authentication, authorization, tokens, redirects, or secret handling. No security section produced.

## Not evaluable from the diff

- DOM-27 (transient DB errors → 503): the new `GET /api/data/maps/{mapId}/objects` handler maps every `GetObjects` error to 404, identical to sibling handlers in the same unmodified file. Confirming whether that pre-existing convention itself is correct would require auditing the un-changed `map/resource.go` handlers wholesale, which is out of this diff's scope — recorded as N/A above on the rule's literal trigger (no `http.StatusInternalServerError` write in the changed handler), not evaluated further.
- The `document.Registry[I,M]` mutex/tenant-partitioning correctness backing `GetMapObjectRegistry()` was read as a targeted lookup (its contract is directly load-bearing for DOM-29/DOM-31 on the new registry) but its own test coverage was not audited — it is unchanged library code, not part of this diff.

## Summary

### Blocking (must fix)
- DOM-04/DOM-05/SUB-01 — `services/atlas-maps/atlas.com/maps/field/resource.go:91-127`: filtering, sorting, and `Model → RestModel` construction all happen inline in the `GET /fields` handler; no `Transform`/`TransformSlice` exists in `field/rest.go`, and no processor owns this logic.
- DOM-04 — `services/atlas-data/atlas.com/data/map/object/rest.go`: no `Transform(Model) (RestModel, error)` defined despite the package having `rest.go`.

### Non-Blocking (should fix)
- DOM-20 — `services/atlas-maps/atlas.com/maps/map/character/registry_test.go`, `services/atlas-data/atlas.com/data/map/object_registry_test.go`, `services/atlas-data/atlas.com/data/map/reader_test.go` (`TestGetObjects*`), `services/atlas-maps/atlas.com/maps/field/resource_test.go` (`TestGetFieldsFilters`): new tests use `t.Run`/plain bodies rather than the `tests := []struct{...}` + `t.Run` table-driven shape the guideline specifies.
- DOM-13 — `services/atlas-maps/atlas.com/maps/field/resource.go:91-127`: filtering/sorting logic sits in the handler (same root cause as the DOM-04/05/SUB-01 blocking finding; listed separately because DOM-13 grades handler orchestration specifically).
