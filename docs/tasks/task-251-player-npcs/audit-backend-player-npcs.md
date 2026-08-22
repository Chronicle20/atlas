# Backend Audit — atlas-player-npcs

- **Service Path:** services/atlas-player-npcs
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-22
- **Build:** PASS
- **Tests:** all packages pass (`go test ./... -count=1`), no failures
- **Overall:** NEEDS-WORK

## Build & Test Results

```
$ cd services/atlas-player-npcs/atlas.com/player-npcs && go build ./...
(no output — success)

$ go test ./... -count=1
ok  	atlas-player-npcs/allocation
ok  	atlas-player-npcs/character
ok  	atlas-player-npcs/configuration
ok  	atlas-player-npcs/data/map
ok  	atlas-player-npcs/data/npc
ok  	atlas-player-npcs/eligibility
ok  	atlas-player-npcs/inventory
ok  	atlas-player-npcs/kafka/consumer/character
ok  	atlas-player-npcs/kafka/consumer/playernpc
ok  	atlas-player-npcs/playernpc
ok  	atlas-player-npcs/position
ok  	atlas-player-npcs/ranking
ok  	atlas-player-npcs/routing
ok  	atlas-player-npcs/snapshot
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (01-05,11,16) | Yes | `playernpc/` has `model.go`, `entity.go`, `rest.go`, `provider.go` |
| FILE placement (01-06) | Yes | every package audited |
| SUB sub-domain (01-04) | No | no package has `resource.go` without `model.go` — `playernpc` has both; kafka consumer packages have neither `resource.go` nor `model.go` |
| REST (06-09,12-19,32) | Yes | `playernpc/rest.go`, `playernpc/resource.go`, `playernpc/processor.go`, registers HTTP routes |
| Constants reuse (DOM-21) | Yes | new consts (`allocation.PoolMin/PoolMax`, `EventType`, error codes) and a new `Model`/`Entity` type declared |
| Testing (DOM-10,20,24,33) | Yes | `_test.go` files throughout; `playernpc.Processor`/`character.Processor`/etc. interfaces defined; producer emit paths reached |
| Cache (DOM-29) | No | no `cache.go`, no cached processor/struct state anywhere in the service |
| Messaging (DOM-30) | Yes | `playernpc/producer.go`, `kafka/consumer/playernpc/consumer.go`, `kafka/consumer/character/consumer.go` call `producer.ProviderImpl` |
| Multi-tenancy (DOM-31) | Yes | every domain/client package has `rest.go`; `tenant.MustFromContext` used throughout |
| Migration hygiene (DOM-34/35) | No | no symbols moved between service and `libs/atlas-*` in this diff |
| Deploy & topics (DOM-22/23) | Yes | adds `COMMAND_TOPIC_PLAYER_NPC` and `EVENT_TOPIC_PLAYER_NPC_STATUS` |
| Runtime safety (DOM-26) | Yes | non-test Go files changed throughout |
| Channel wire values (DOM-25) | No | service does not touch `atlas-channel`/`atlas-packet`, and emits semantic JSON events, not client-interpreted bytes |
| Resilience (DOM-27/28) | Partial | DOM-27: no handler writes a bare `http.StatusInternalServerError` (all route through `server.WriteErrorResponse`) — N/A by absence; DOM-28: no `model.Decorator`/enrichment fallback path exists — N/A |
| External clients (EXT-01-04) | Yes | `character`, `configuration`, `data/map`, `data/npc`, `inventory`, `ranking` all call `requests.GetRequest[T]` against other atlas services |
| Scaffolding (SCAFFOLD-01-09) | Yes | diff adds `services/atlas-player-npcs/` |
| Security (SEC-01-04) | No | service handles no tokens, auth, redirects, or secrets |

## Checklist Results

### playernpc (domain — has `model.go`, `entity.go`, `rest.go`, `provider.go`, `processor.go`, `resource.go`, `builder.go`, `administrator.go`, `producer.go`, `errors.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` has `NewBuilder()`, fluent setters, validating `Build()` | PASS | `playernpc/builder.go:47` `NewBuilder()`; `:200` `Build()`; `:229` `validate()` rejects empty `characterId`/`name` |
| DOM-02 | `Model.ToEntity()` in `entity.go` | FAIL | No `func (m Model) ToEntity()` exists anywhere in `entity.go`; the model→entity conversion is a free function `MakeEntity(tenantId uuid.UUID, m Model) Entity` at `playernpc/entity.go:159` |
| DOM-03 | `func Make(` returns `(Model, error)` | PASS | `playernpc/entity.go:103` `func Make(e Entity, equipmentEntities []EquipmentEntity) (Model, error)` |
| DOM-04 | `rest.go` has `func Transform(` | FAIL | `Transform` is defined in `playernpc/resource.go:73`, not `rest.go` — `rest.go` holds handlers/route registration instead (see FILE-02 below) |
| DOM-05 | `rest.go` has `TransformSlice`, used by list handlers | FAIL | No `TransformSlice` function exists anywhere in the package; the list handler `handleGetPlayerNpcs` (`playernpc/rest.go:293`) uses `model.SliceMap(Transform)(...)` inline instead |
| DOM-11 | Providers lazy via `database.Query`/`SliceQuery`, no eager `FixedProvider` wrap | PASS | `playernpc/provider.go:12-34` mirrors the canonical example in `patterns-provider.md` (deferred `db.Where().First()` inside the returned closure, `FixedProvider` wraps the already-fetched result — this is the documented pattern, not the anti-pattern); `:38` `getByMapPagedProvider` uses `database.PagedQuery` directly |
| DOM-16 | `administrator.go` holds writes, called by processor | FAIL | `administrator.go:19` `createPlayerNpc` exists and is correct, but `processor.go`'s `Deploy` does **not** call it — it inlines its own `tx.Create(&entity)` (`playernpc/processor.go:303`) and `tx.Create(&eqEntities[i])` (`:309`) directly inside the transaction, duplicating `createPlayerNpc`'s logic instead of delegating to it |

### playernpc (FILE placement)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in `processor.go` | PASS | `playernpc/processor.go:88` interface, `:120` `NewProcessor`, `:145+` `ProcessorImpl` methods |
| FILE-02 | `RestModel`, `Transform`, JSON:API methods in `rest.go` | FAIL | `RestModel`+`Transform`+`GetName/GetID/SetID` live in `playernpc/resource.go` (104 lines); the request model `DeployRestModel`+its JSON:API methods live in `playernpc/requests.go`; `rest.go` (506 lines) instead holds `HandlerDependency`, route registration (`InitializeRoutes`), and every HTTP handler — the file contents of `rest.go`/`resource.go` are inverted relative to the repo's own reference convention (`services/atlas-notes/atlas.com/notes/note/`: `resource.go` = handlers+routes, `rest.go` = RestModel+Transform, confirmed at `services/atlas-notes/.../note/resource.go:18` `InitializeRoutes`, `.../note/rest.go`) |
| FILE-03 | Cross-service request functions in `requests.go` | FAIL | `playernpc/requests.go` contains no cross-service request functions at all — it holds `DeployRestModel`/`PositionRestModel` (this service's own inbound REST input model). `playernpc` calls no other atlas service directly (that's delegated to `character`/`inventory`/`ranking`/`configuration`/`data/npc`/`data/map`, each of which has its own correctly-placed `requests.go`), so this file has no legitimate FILE-03 content and is misused |
| FILE-04 | Entity/Migration/TableName in `entity.go` | PASS | `playernpc/entity.go:33` `Entity` struct, `:60` `TableName()`, `:206` `Migration(db *gorm.DB) error` |
| FILE-05 | Builder/Model/administrator/provider/enums in their named files | FAIL | Builder/Model/administrator/provider are correctly placed, but the `EventType` enum (`const EventTypeDeployed ... EventTypeRepositioned`) is declared in `playernpc/processor.go:53-61`, not a `state.go` — the fleet convention for enums the rule points at exists elsewhere in the repo (e.g. `services/atlas-account/atlas.com/account/account/state.go`) |
| FILE-06 | No package-named catch-all combining ≥2 responsibilities | N/A | No file mixes ≥2 of the FILE-01..05 categories in one file; the violation here is misplacement (FILE-02/03/05), not a catch-all |

### playernpc (REST — resource.go / rest.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | `playernpc/processor.go:120` `NewProcessor(l logrus.FieldLogger, ...)` |
| DOM-07 | Handlers pass `d.Logger()` into `NewProcessor` | PASS | `playernpc/rest.go:196` `processorFor` — `l, ctx, db := d.Logger(), d.Context(), d.DB()` then `NewProcessor(l, ctx, db, ...)` |
| DOM-08 | POST/PATCH via `RegisterInputHandler[T]` | FAIL | POST `deploy_player_npc` correctly uses `registerDeploy` (input-handler variant, `rest.go:177`); but PATCH `redeploy_player_npc` (`rest.go:184`) registers via `registerGet` — the plain `GetHandler` variant, not an input-handler variant, for a `Methods(http.MethodPatch)` route |
| DOM-09 | Every `Transform(` call site checks its error | PASS | `playernpc/rest.go:211` `rm, err := model.Map(Transform)(...)` checked; `:293` `rms, err := model.SliceMap(Transform)(...)` checked |
| DOM-12 | No `os.Getenv()` in handlers | PASS | no `os.Getenv` occurrence anywhere in the package |
| DOM-13 | No cross-domain orchestration in handlers | FAIL | `handleGetEligibility` (`playernpc/rest.go:440-506`) directly constructs `character.NewProcessor(l, ctx)` (`:477`) and `configuration.NewProcessor(l, ctx)` (`:489`), fetches the character, reads config, counts existing NPCs, and calls `eligibility.Evaluate` — all in the handler, not through `playernpc.Processor` |
| DOM-14 | Handlers call processor methods, never providers directly | FAIL | `handleGetPlayerNpc` (`rest.go:310`) calls `getPlayerNpcModel(d.DB().WithContext(d.Context()), id)` directly instead of `processorFor(d).GetById(id)` (which exists on `Processor`); `handleGetPlayerNpcs` (`rest.go:286`) calls package-level `pagedPlayerNpcsByMap` (`rest.go:255`, itself calling `getByMapPagedProvider` and `getPlayerNpcModel` directly) instead of `processorFor(d).GetByMap(...)`; `handleGetEligibility` (`rest.go:491`) calls `countByName(db.WithContext(ctx), ...)`, an `administrator.go` function, directly |
| DOM-15 | No `db.Create`/`Save`/`Delete` in handlers | PASS | no such call in `rest.go`/`resource.go` |
| DOM-17 | Domain errors map to specific HTTP status | PASS | `writeDeployError` (`rest.go:143`) maps `CodeUnresolvable`→422, `CodeDuplicate`/`CodePoolExhausted`/`CodeMapFull`/`CodeIneligible`→409; `gorm.ErrRecordNotFound`→404 at `rest.go:311`, `:352`, `:378` |
| DOM-18 | `RestModel` implements `GetName/GetID/SetID` | PASS | `playernpc/resource.go:46-64` |
| DOM-19 | Request models flat, no nested Data/Type/Attributes | PASS | `DeployRestModel` (`requests.go:15`) has a plain `*PositionRestModel` sub-struct, not a JSON:API envelope |
| DOM-32 | Routes register through `server.RegisterHandler`/`RegisterInputHandler[T]`, no custom error helper | FAIL | The local `registerHandler`/`registerInputHandler` (`rest.go:84`, `:99`) do **not** delegate to `server.RegisterHandler`/`server.RegisterInputHandler[T]` — they re-implement the span/tenant-parsing chain manually via `server.RetrieveSpan`+`server.ParseTenant` (the delegation the DOM-32 procedure requires tracing to and confirming was never made); `writeError` (`rest.go:126`) is a hand-rolled custom error-response helper, not a raw status write nor `server.WriteErrorResponse` |

### playernpc (Messaging — DOM-30)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-30 | DB-write operations emit through `AndEmit`+`message.Buffer`, not a bare `producer.ProviderImpl` from the success path | PASS (by procedure scope) | The only `producer.ProviderImpl(` call for domain events is inside `playernpc/producer.go:29` itself, which the checklist's own verification procedure explicitly excludes from the grep ("call sites outside `producer.go` itself"). No `message.Buffer`/`AndEmit` is used — `Deploy`/`Redeploy`/`RemoveById` commit their transaction first and call `p.emit(...)` only afterward (`processor.go:293-317`), which the package doc (`processor.go:36-42`) acknowledges accepts a commit-succeeds/emit-fails window. This is architecturally the gap `AndEmit` exists to close, but it is not a rule violation under the checklist's literal grep scope |

### playernpc (Multi-tenancy — DOM-31)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-31 | Tenant/trace travel in context only | PASS | No `tenantId` field on `RestModel` (`resource.go:19-43`) or `DeployRestModel` (`requests.go:15-21`); tenant read via `tenant.MustFromContext(p.ctx)` (`processor.go:158`) and `tenant.MustFromContext(ctx)` (`rest.go:485`) |

### playernpc (Testing — DOM-10/20/24/33)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-10 | Test DB setup calls `database.RegisterTenantCallbacks` | PASS | `playernpc/administrator_test.go:30` `database.RegisterTenantCallbacks(l, db)` inside the shared `testDatabase(t)` helper reused by `resource_test.go`/`processor_test.go` |
| DOM-20 | Tests table-driven (`tests := []struct{} + t.Run`) | WARN | `errors_test.go:12` is properly table-driven; `processor_test.go` and `resource_test.go` use a long sequence of individual `t.Run("name", func(t *testing.T){...})` blocks without a driving `[]struct{}` table — not a hard violation of any single test, but a real deviation from the documented shape for most of the suite |
| DOM-24 | Emit-path test packages install the producer stub | PASS | `playernpc/producer_test.go:17-19` `TestMain` calls `producertest.InstallCapturing()` |
| DOM-33 | Interface changes update every mock | N/A | Greenfield service — no `mock` package exists anywhere under `services/atlas-player-npcs/`; interfaces are newly introduced, not re-signed |

### playernpc (Runtime safety — DOM-26)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-26 | Every goroutine via `routine.Go` | PASS | The only `go func`/goroutine spawn in the service is `main.go:64` `routine.Go(l, rt.Context(), func(_ context.Context) {...})` |

### playernpc (Resilience — DOM-27/28)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-27 | Transient DB errors surface as 503 via `server.WriteErrorResponse` | N/A | No handler writes a bare `http.StatusInternalServerError` in non-test code (only `resource_test.go:257`, a test fixture); every generic error path already routes through `server.WriteErrorResponse` |
| DOM-28 | Enrichment/decorator paths degrade loudly | N/A | No `model.Decorator` or enrichment fallback path exists anywhere in the service |

### character / configuration / data/map / data/npc / inventory / ranking (client packages — EXT-01..04)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| EXT-01 | Target `RestModel` implements `SetToOneReferenceID`/`SetToManyReferenceIDs` | PASS: `character` (`character/rest.go:48,52`), `inventory` (`inventory/rest.go:95,99,185,189`) — FAIL: `data/map` (`data/map/rest.go` — `RestModel`, `GroundRequestRestModel`, `GroundResultRestModel` all lack both methods), `data/npc` (`data/npc/rest.go` — `RestModel` lacks both), `ranking` (`ranking/rest.go` — `RestModel` lacks both), `configuration` (`configuration/rest.go` — `RestModel` lacks both) |
| EXT-02 | `httptest`-backed integration test with a representative JSON:API fixture | PASS: `configuration/rest_test.go:29` `httptest.NewServer(...)`; `ranking/rest_test.go:28` `httptest.NewServer(...)` — FAIL: `character/rest_test.go`, `data/map/rest_test.go`, `data/npc/rest_test.go`, `inventory/rest_test.go` contain only direct `jsonapi.Unmarshal`+`Extract` unit tests, no `httptest.NewServer` anywhere in those four packages |
| EXT-03 | Only genuine 404s map to "not found"; transport/decode/5xx bubble up unmodified | PASS | None of the six client packages perform their own broad-error-to-"not found" mapping — no `errors.Is(err, requests.ErrNotFound)` reclassification exists outside `ranking/processor.go:35` and `configuration/processor.go:35`, both of which use it correctly (fallback-to-default on genuine not-found, not a blanket catch) |
| EXT-04 | URL composed via `requests.RootUrl(<DOMAIN>)`/equivalent, not hardcoded DNS | PASS | All six packages use `requests.RootUrlFor(ctx, "<DOMAIN>")` (`character/requests.go:17`, `configuration/requests.go:15`, `data/map/requests.go:18`, `data/npc/requests.go:16`, `inventory/requests.go:16`, `ranking/requests.go:17`) — the context-aware sibling of `requests.RootUrl`, no hardcoded DNS anywhere |

### kafka/consumer/playernpc, kafka/consumer/character (sub-domain / support — no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01..04 | N/A | Neither package has `resource.go` (they are Kafka handler packages, not REST sub-domain packages) — trigger `resource.go` present + no `model.go` never fires |
| DOM-30 | `producer.ProviderImpl` outside `producer.go` wrapped in `AndEmit` unless the operation has no DB write of its own | PASS (documented-exception reasoning) | `kafka/consumer/playernpc/consumer.go:56` (`NewOutcomeEmitter`) and `kafka/consumer/character/consumer.go:154` (`emitDeployCommand`) both call `producer.ProviderImpl` directly, but neither wraps a DB write of its own — they report/trigger downstream operations whose writes (if any) already committed through `playernpc.Processor`'s own transaction. This matches the spirit of "Operations over non-DB state" (checklist's documented exception #2) even though neither is literally an in-memory registry |
| DOM-24 | Emit-path tests stub the producer | PASS | `kafka/consumer/character/consumer_test.go:29-32` installs `producertest.InstallCapturing()` in `TestMain`; `kafka/consumer/playernpc/consumer_test.go:90,97` injects a stub `OutcomeEmitter` per test (`captureEmitter`/`discardEmitter`) |

### allocation / eligibility / position / routing / snapshot (support packages)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | No `model.go`/`entity.go`/`rest.go`/`processor.go`/`resource.go` in any of these five packages | N/A | Confirmed by directory listing — each holds only pure algorithm/lookup code (`allocation/allocation.go:1`, `eligibility/eligibility.go:1`, `position/types.go:1`, `routing/routing.go:1`, `snapshot/snapshot.go:1` package docs) |

## Deploy & Scaffolding

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-22 | New `libs/atlas-*` module wired into Dockerfile/go.work | N/A | No new `libs/atlas-*` module added by this diff |
| DOM-23 | Topic env vars in base configmap, both overlays, no manifest literal | PASS | `COMMAND_TOPIC_PLAYER_NPC`, `EVENT_TOPIC_PLAYER_NPC_STATUS`, `EVENT_TOPIC_CHARACTER_STATUS` all `configmap=PRESENT manifest_literal=CLEAN` (verified via the checklist's own script against `deploy/k8s/base/env-configmap.yaml`/`deploy/k8s/base/atlas-player-npcs.yaml`); present in `deploy/k8s/overlays/main/kustomization.yaml:114,207` and `deploy/k8s/overlays/pr/kustomization.yaml:234,327` |
| SCAFFOLD-01 | `.github/config/services.json` `go-service` entry | PASS | `.github/config/services.json:393-397` |
| SCAFFOLD-02 | Base manifest exists, listed in base `kustomization.yaml` | PASS | `deploy/k8s/base/atlas-player-npcs.yaml` exists; listed at `deploy/k8s/base/kustomization.yaml:57` |
| SCAFFOLD-03 | `docker-bake.hcl` entry, `go.work` `use()` entry | PASS | `docker-bake.hcl:84`; `go.work:74` |
| SCAFFOLD-04 | Ingress location block in `deploy/shared/routes.conf` | PASS | `deploy/shared/routes.conf:561-562` |
| SCAFFOLD-05 | Generated routes regenerated and committed | PASS | `tools/gen-routes.sh` reports "gen-routes: up to date"; no working-tree diff produced on re-run |
| SCAFFOLD-06 | docker-compose entry alongside peers | FAIL | No `atlas-player-npcs:` service block exists in `deploy/compose/docker-compose.core.yml`, the file that carries every other service's peer entry (e.g. `atlas-account`, `atlas-ban`) |
| SCAFFOLD-07 | New channel Writer/Handler seeded in tenant templates | N/A | This diff registers no new `atlas-channel` `Writer`/`Handler` |
| SCAFFOLD-08 | Bruno collection present | PASS | `services/atlas-player-npcs/.bruno/collection.bru` and 7 request files |
| SCAFFOLD-09 | Overlay enumerations / `ATLAS_DB_NAMES` / DB bootstrap | PASS | `tools/service-registration-guard.sh` exit: "service-registration-guard: clean"; `atlas-player-npcs` present in `ATLAS_DB_NAMES` (`deploy/k8s/overlays/pr/kustomization.yaml:351`) |

## Security Review

Not applicable — `SEC-*` trigger did not fire: `atlas-player-npcs` handles no authentication tokens, no login/logout/session flow, no redirect/callback handlers, and no secrets.

## Not evaluable from the diff

- DOM-20 (table-driven tests): graded from the test files read (`errors_test.go`, `processor_test.go`, `resource_test.go`, `administrator_test.go`, `model_test.go`); a full line-by-line audit of every `t.Run` block across all 12 `_test.go` files in the service was not performed given the review's scope — the WARN above reflects the pattern observed in the files actually read, not an exhaustive sweep.
- FILE-01..06 for `kafka/message/playernpc/kafka.go` and `kafka/message/character/kafka.go`: these hold wire envelope types only; not graded against FILE-01..05 since none of those categories (Processor/RestModel/cross-service requests/Entity/Builder-Model-administrator-provider-state) apply to a pure Kafka message-envelope file, and the checklist defines no separate rule for that file's placement.
- DOM-21 (constants reuse) was checked only for the specific values that looked most likely to collide with `libs/atlas-constants` (object-id derivation, script-id pool bounds); a full symbol-by-symbol diff against every existing `libs/atlas-constants` value was not performed.

## Summary

### Blocking (must fix)
- DOM-02: `playernpc/entity.go` — no `Model.ToEntity()` method; only a free `MakeEntity(tenantId, m)` function.
- DOM-04 / FILE-02: `Transform` lives in `playernpc/resource.go`, not `rest.go`.
- DOM-05: no `TransformSlice` function anywhere in `playernpc`.
- DOM-08: `playernpc/rest.go:184` registers the PATCH redeploy route via the plain `GetHandler` variant, not an input-handler variant.
- DOM-13/DOM-14: `playernpc/rest.go` handlers (`handleGetPlayerNpc:310`, `handleGetPlayerNpcs:286`, `handleGetEligibility:440-506`) bypass `playernpc.Processor` and call providers/administrator functions and other packages' processors directly.
- DOM-16: `playernpc/processor.go:303,309` inlines `tx.Create` in `Deploy` instead of calling `administrator.go`'s `createPlayerNpc`.
- DOM-32: `playernpc/rest.go`'s local `registerHandler`/`registerInputHandler` (`:84`,`:99`) do not delegate to `server.RegisterHandler`/`server.RegisterInputHandler[T]`; `writeError` (`:126`) is a custom error-response helper.
- FILE-03: `playernpc/requests.go` holds a domain input model instead of cross-service request functions.
- FILE-05: `EventType` enum declared in `playernpc/processor.go:53-61` instead of a `state.go`.
- EXT-01: `data/map`, `data/npc`, `ranking`, `configuration` `RestModel`s lack `SetToOneReferenceID`/`SetToManyReferenceIDs`.
- EXT-02: `character`, `data/map`, `data/npc`, `inventory` have no `httptest`-backed integration test.
- SCAFFOLD-06: no `atlas-player-npcs` entry in `deploy/compose/docker-compose.core.yml`.

### Non-Blocking (should fix)
- DOM-20: `processor_test.go`/`resource_test.go` use sequential `t.Run` blocks rather than a driving `[]struct{}` table for most of the suite.
- DOM-30: domain-event emission happens strictly after transaction commit via a bare `producer.ProviderImpl` call inside `producer.go`, not through `AndEmit`/`message.Buffer` — passes the checklist's literal grep scope but leaves the commit-succeeds/emit-fails window the pattern exists to close (acknowledged in the package's own doc comment).
