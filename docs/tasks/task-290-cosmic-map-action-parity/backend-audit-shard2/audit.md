# Backend Audit — task-290-cosmic-map-action-parity (Shard 2)

- **Service Paths:** services/atlas-maps, services/atlas-channel, services/atlas-query-aggregator, services/atlas-map-actions, services/atlas-reactors, services/atlas-drops, services/atlas-data, services/atlas-configurations, services/atlas-monsters, libs/atlas-script-core, tools/catalog-lint, tools/gen-map-action-schema
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-09-03
- **Range:** `3f37fe6da..9c2b9be13`
- **Build:** PASS (module-local, all 12 touched modules)
- **Tests:** PASS (module-local `go test ./... -count=1`, all 12 touched modules)
- **Overall:** NEEDS-WORK

## Build & Test Results

```
services/atlas-maps/atlas.com/maps               : go build OK, go test OK (all packages ok/no-test-files)
services/atlas-channel/atlas.com/channel         : go build OK, go test OK
services/atlas-query-aggregator/.../query-aggregator : go build OK, go test OK
services/atlas-map-actions/.../map-actions       : go build OK, go test OK
services/atlas-reactors/.../reactors             : go build OK, go test OK
services/atlas-drops/.../drops                   : go build OK, go test OK
services/atlas-data/.../data                     : go build OK, go test OK
services/atlas-configurations/.../configurations : go build OK, go test OK
services/atlas-monsters/.../monsters             : go build OK, go test OK
libs/atlas-script-core                           : go build OK, go test OK
tools/catalog-lint                               : go build OK, go test OK
tools/gen-map-action-schema                      : go build OK, go test OK
```
No failures across any module.

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| FILE (FILE-01..06) | Yes | Every changed Go package runs unconditionally. |
| RUNTIME / DOM-26 (goroutines) | Yes (opens; rule mostly N/A) | Non-test Go files changed across all modules; new `routine.Go(l, ctx, ...)` call at `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:265` — correct wrapped form, no bare `go`. |
| DOM structure (DOM-01..05,11,16) | Yes | New `map/npc` (atlas-maps) has `model.go`+`entity.go`(absent)+`rest.go`; new `area_info` (query-aggregator) has `model.go`+`rest.go`. |
| SUB (SUB-01..04) | Yes | New `drops/map` (atlas-drops) has `resource.go`, no `model.go`. |
| REST (DOM-06..09,12..15,17..19,32) | Yes | Multiple `resource.go`/`rest.go`/`processor.go` changed. |
| Constants reuse (DOM-21) | No | No new type/const-block/numeric-literal classification declared; `sharedsaga.AreaInfoCondition`/`TransportInTransitCondition` cast reuses existing shared constants rather than redeclaring. |
| Testing (DOM-10,20,24,33) | Yes | Extensive `_test.go` changes; `drop.Processor`, `reactor.Processor`, `map/monster.Processor`, `monster.Processor` (atlas-maps), `area_info.Processor` interfaces gain methods. |
| Cache (DOM-29) | No | No `cache.go` touched; no new cached processor state (registries are pre-existing singleton patterns, not newly introduced caching). |
| Messaging (DOM-30) | Yes | New `producer.go` (atlas-maps `map/npc`); direct `producer.ProviderImpl` calls added in `drops/drop/processor.go` (via `message.Emit`+Buffer) and `reactors/reactor/processor.go` (direct, non-DB registry state). |
| Multi-tenancy (DOM-31) | Yes | Multiple `rest.go` files changed; verified no tenant/trace field added to any REST model. |
| Migration hygiene (DOM-34,35) | No | No symbols moved between a service and a `libs/atlas-*` module. |
| Deploy & topics (DOM-22,23) | Yes | New `EVENT_TOPIC_NPC_STATUS` topic env var added — verified wired in base env-configmap, both overlays, and kafka-topics-configmap. |
| Channel wire values (DOM-25) | Yes | `atlas-channel` touched extensively (new `system_message` consumer, `map/npc` consumer); new `ContiMove`/`FieldEffect` writer bindings added to tenant seed templates, resolved via `writer.TenantWriterOptions`+`atlaspacket.CodeConfigured`, never literal bytes. |
| Resilience (DOM-27,28) | Yes | New handlers in `atlas-maps` (DB-backed service) write `http.StatusInternalServerError` directly in places. |
| External clients (EXT-01..04) | Yes | New `area_info` package (query-aggregator) calls atlas-character via `requests.GetRequest[T]`; `atlas-maps/monster` processor gains a new atlas-monsters DELETE call. |
| Scaffolding (SCAFFOLD-01..09) | No | No new `services/atlas-<svc>/` directory; no `deploy/shared/routes.conf` change. |
| Security (SEC-01..04) | No | None of the touched services handle auth tokens, revocation, redirects, or secrets in changed code. |

## Checklist Results

### map/npc (atlas-maps) — NEW domain package (`model.go`, `processor.go`, `producer.go`, `registry.go`, `resource.go`, `rest.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` with `NewBuilder()`/`Build()` | **FAIL** | `services/atlas-maps/atlas.com/maps/map/npc/` has `model.go` (triggers DOM-01) but no `builder.go`; `Model` is constructed directly via `NewModel(...)` in `model.go:29`. No documented exception in `file-responsibilities.md` for in-memory/non-persisted domain models. |
| DOM-02/03 | `entity.go` ToEntity/Make | N/A | No `entity.go` — Model is session/instance-scoped, never persisted (`model.go:12-14` doc comment). |
| DOM-04 | `Transform(Model)(RestModel,error)` in `rest.go` | PASS | `services/atlas-maps/atlas.com/maps/map/npc/rest.go:26` `func Transform(m Model) (RestModel, error)`. |
| DOM-05 | `TransformSlice` in `rest.go`, used by list handler | **FAIL** | `rest.go` defines no `TransformSlice`; the list handler `handleGetNpcsInMap` (`resource.go:48`) calls `model.SliceMap(Transform)(model.FixedProvider(ns))()()` directly. This is a wholly new file introduced by this diff (not carried-forward convention). |
| DOM-16 | `administrator.go` for create/update/delete | N/A | `Create` (processor.go:36) writes to the in-memory `Registry` (registry.go), never a GORM/DB entity — administrator.go is documented (file-responsibilities.md top-of-file) as DB-mutation-specific. |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `processor.go:25` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`. |
| DOM-08/32 | POST via `RegisterInputHandler[T]` | PASS | `resource.go:29` `rest.RegisterInputHandler[RestModel](l)(si)(createNpcInMap, handleCreateNpcInMap)`. |
| DOM-09 | `Transform` call sites check error | PASS | `resource.go:48-52` and `resource.go:75-79` both check `err` from `model.SliceMap`/`model.Map`. |
| DOM-13/14/15 | No cross-domain orchestration, no provider calls, no direct DB writes in handler | PASS | `resource.go` calls only `NewProcessor(...).GetInField`/`.Create`; no provider/db calls. |
| DOM-17 | Domain errors → specific HTTP status | WARN | `resource.go:70` maps every `p.Create` error to 400 regardless of cause (mirrors the pre-existing atlas-monsters `world/resource.go` WARN noted in the prior shard's audit — same class of imprecision, newly introduced here). |
| DOM-19 | Request models flat | PASS | `rest.go:5-11` `RestModel` — all scalar fields. |
| DOM-27 | DB-backed service uses `WriteErrorResponse`, not raw 500 | **FAIL** | `services/atlas-maps/atlas.com/maps/map/npc/resource.go:44` and `:88` write `w.WriteHeader(http.StatusInternalServerError)` directly. atlas-maps is DB-backed (`services/atlas-maps/atlas.com/maps/main.go:74` `database.Connect(...)`), and its `main.go:76` already registers `server.RegisterTransientErrorClassifier`. Contrast `resource.go:51` in the same file, which correctly uses `server.WriteErrorResponse(d.Logger())(w)(err)`. |
| DOM-30 | `AndEmit`+Buffer for DB writes; direct emit OK for non-DB state | PASS (documented exception) | `processor.go:56` calls `emit(...)` → `producer.go:21` `producer.ProviderImpl(...)` directly from the success path, but `Create` writes only to the in-memory `Registry` (registry.go), not a DB — matches the documented DOM-30 exception "Operations over non-DB state" (patterns-kafka.md, citing `atlas-chairs/chair/processor.go` as precedent). |
| DOM-23 | New topic env var wired everywhere | PASS | `services/atlas-maps/atlas.com/maps/kafka/message/npc/kafka.go:13` defines `EVENT_TOPIC_NPC_STATUS`; verified present in `deploy/k8s/base/env-configmap.yaml:163`, `deploy/k8s/base/kafka-topics-configmap.yaml:284`, `deploy/k8s/overlays/main/kustomization.yaml:193`, `deploy/k8s/overlays/pr/kustomization.yaml:310`, and `deploy/compose/.env.example:152`, all added in the same commit range. |
| DOM-24 | Producer stub for emit-reaching tests | PASS | `services/atlas-maps/atlas.com/maps/map/npc/processor_test.go:33-51` `stubNpcEmit` swaps the package-level `emit` var for a recording stub; no real Kafka connection reached. |
| DOM-20 | Table-driven tests | **FAIL** | `processor_test.go` (new file): 6 new funcs, all standalone — `TestCreateNpcInField` (:54), `TestCreateNpcSpawnIfAbsentSuppressesWhenPresent` (:84), `TestCreateNpcSpawnIfAbsentIsFieldScoped` (:112), `TestCreateNpcWithoutGuardStacks` (:141), `TestCreateNpcEmitsCreatedStatusEvent` (:167), `TestCreateNpcSpawnIfAbsentSuppressedEmitsNothing` (:199). `registry_test.go` (new file): 4 new funcs, all standalone — `TestRegistryAddAndGetAll` (:15), `TestRegistryGetAllReturnsCopy` (:33), `TestRegistryReset` (:48), `TestRegistryNextIdIsUnique` (:63). None use `tests := []struct{...}` + `t.Run`. |
| DOM-31 | Tenant/trace only in context | PASS | `rest.go`/`resource.go` carry only `NpcId`/`X`/`Y`/`Fh`/`SpawnIfAbsent` — no tenant/trace field (grep for `TenantId`/`tenantId`/`TraceId`/`traceId` across the package: zero matches). |

### map (atlas-maps) — domain package (`resource.go`, `rest.go` changed: `handleResetField`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-08/32 | POST via `RegisterInputHandler[T]` | PASS | `services/atlas-maps/atlas.com/maps/map/resource.go:34` `rest.RegisterInputHandler[ResetFieldInputRestModel](l)(si)(resetField, handleResetField)`. |
| DOM-27 | `WriteErrorResponse` for 500 in DB-backed service | PASS | `map/resource.go:129` `server.WriteErrorResponse(d.Logger())(w)(err)` on `ResetField` failure. |
| DOM-19 | Request model flat | PASS | `map/rest.go:29-32` `ResetFieldInputRestModel{Id, Difficulty}` — flat scalars. |

### monster (atlas-maps, cross-service client to atlas-monsters) — support package (`processor.go`, `requests.go` unchanged file but sibling of new code)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-03 | Cross-service request/URL-construction functions live in `requests.go` | **FAIL** | `services/atlas-maps/atlas.com/maps/monster/processor.go:79-84` (`DeleteInMap`) calls `getBaseRequest(p.ctx)` and builds the URL with `fmt.Sprintf(root+mapMonstersResource, ...)` directly in `processor.go`, duplicating logic that the package's own convention places in `requests.go` — see the sibling `inMapUrl`/`inMapRectUrl`/`requestCreate` functions in `services/atlas-maps/atlas.com/maps/monster/requests.go:31-65`, all of which build their URL there and are merely called from `processor.go`. |
| EXT-04 | URL via `requests.RootUrl`/`RootUrlFor` | PASS | `monster/requests.go:23` `requests.RootUrlFor(ctx, "MONSTERS")` (pre-existing, called into by the new `DeleteInMap`). |
| EXT-03 | Only genuine 404 → not-found | PASS | `monster/processor.go:84` `return requests.DeleteRequest(url)(p.l, p.ctx)` — error returned verbatim, no status-code reclassification. |

### map/monster (atlas-maps) — sub-domain package (`registry.go`, `processor.go`; Redis-backed, no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Business logic in processor, not handler | PASS | `ResetField` (`map/monster/processor.go`, new method) composes `p.mp.DeleteInMap` + `GetRegistry().RestoreSpawnPoints`; the handler (`map/resource.go:118-131`) only calls the processor. |
| DOM-20 | Table-driven tests | **FAIL** | `map/monster/processor_test.go`: 3 new funcs, all standalone — `TestResetFieldRestoresSpawnPointsAndClearsMonsters` (:1166), `TestResetFieldIsFieldScoped` (:1222), `TestResetFieldOnUnknownMapErrors` (:1283). |

### drop (atlas-drops) — domain package (`processor.go`, `processor_test.go` changed)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-30 | `AndEmit`+Buffer for the write's events | PASS | `services/atlas-drops/atlas.com/drops/drop/processor.go:322-336` `ClearForField` wraps every `Expire` call inside `message.Emit(producerProvider)(func(mb *message.Buffer) error {...})` — the documented `*AndEmit` shape. |
| DOM-24 | Producer stub for emit-reaching tests | PASS | `drop/processor_test.go:16-27` `TestMain` installs `producertest.InstallNoop()`; `TestClearForFieldEmitsRemovalPerDrop` (:1096) swaps in `producertest.InstallCapturing()`. |
| DOM-33 | Interface change updates every mock | **FAIL** | `drop.Processor` gains `ClearForField(f field.Model) (int, error)` (`processor.go:60-63`), but `services/atlas-drops/atlas.com/drops/drop/mock/processor.go` was not updated in this diff — no `ClearForField`/`ClearForFieldFunc` field exists, and the mock carries no compile-time `var _ drop.Processor = (*ProcessorMock)(nil)` assertion, so the gap does not surface at build time. Repo-wide grep confirms the mock is currently unused (`grep -rn 'drop/mock' services/atlas-drops` outside its own package: no hits), but it is a stale, silently-incomplete mock the moment anyone wires it against `drop.Processor`. |
| DOM-20 | Table-driven tests | **FAIL** | `drop/processor_test.go`: 4 new funcs, all standalone — `TestClearForFieldRemovesEveryDrop` (:1009), `TestClearForFieldIsFieldScoped` (:1042), `TestClearForFieldOnEmptyFieldSucceeds` (:1079), `TestClearForFieldEmitsRemovalPerDrop` (:1096). |

### drops/map (atlas-drops) — NEW sub-domain package (`resource.go`, `resource_paginate_test.go`; no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Business logic in parent processor | PASS | `services/atlas-drops/atlas.com/drops/map/resource.go` delegates to `drop.NewProcessor(...).GetForMap`/`.ClearForField`; no logic in the handler. |
| SUB-02 | No `db.Create`/`db.Save` in `resource.go` | PASS | Grep confirms zero matches. |
| SUB-03 | POST uses `RegisterInputHandler[T]` | N/A | Package registers only GET and DELETE routes — no POST route (`resource.go:29-30`). |
| SUB-04 | No manual JSON parsing | PASS | Zero `json.NewDecoder`/`json.Unmarshal`/`io.ReadAll` matches. |
| DOM-27 | `WriteErrorResponse` in DB-backed service | N/A | atlas-drops does not call `database.Connect` (grep of `main.go`: no match) — not a DB-backed service, rule does not apply. |

### reactor (atlas-reactors) — domain package (`processor.go`, `resource.go`, `rest.go`, `mock/processor.go` changed)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-30 | Direct emit for non-DB registry state | PASS (documented exception) | `services/atlas-reactors/atlas.com/reactors/reactor/processor.go:385,412` (`ResetInField`/inline in `ShuffleInField`'s caller) call `producer.ProviderImpl(...)` directly from the success path; both methods mutate only the in-memory reactor `Registry`, matching the DOM-30 non-DB-state exception. |
| DOM-08 | POST via `RegisterInputHandler[T]`, never `RegisterHandler` | **FAIL** | `services/atlas-reactors/atlas.com/reactors/reactor/resource.go:33` `r.HandleFunc("/reactors/shuffle", registerPost("shuffle_reactors_in_map", handleShuffleReactorsInMap)).Methods(http.MethodPost)` where `registerPost := rest.RegisterHandler(l)(si)` (`resource.go:25`) — a POST route registered through `RegisterHandler`, not `RegisterInputHandler[T]`. No documented exception in `patterns-rest-jsonapi.md` for body-less POST actions; the sibling `/reactors/reset` route in the same diff correctly uses `RegisterInputHandler[ResetInputRestModel]` (`resource.go:32`). |
| DOM-27 | `WriteErrorResponse` in DB-backed service | N/A | atlas-reactors does not call `database.Connect` — not DB-backed. |
| DOM-33 | Interface change updates every mock | PASS | `reactor.Processor` gains `ResetInField`/`ShuffleInField` (`processor.go:38-39`); `services/atlas-reactors/atlas.com/reactors/reactor/mock/processor.go:25-26,116-129` adds both methods with the standard `*Func` override pattern in the same diff. |
| DOM-19 | Request model flat | PASS | `rest.go:47-50` `ResetInputRestModel{Id, MinState *int8}` — flat, no nested envelope. |
| DOM-20 | Table-driven tests | **FAIL** | `reactor/processor_test.go`: 5 new funcs, all standalone — `TestResetInFieldResetsEveryReactor` (:629), `TestResetInFieldHonorsMinState` (:658), `TestResetInFieldIsFieldScoped` (:696), `TestShuffleInFieldPermutesPositionsOnly` (:730), `TestShuffleInFieldWithOneReactorIsANoOp` (:793). |

### area_info (atlas-query-aggregator) — NEW support/client package (`model.go`, `processor.go`, `requests.go`, `rest.go`, `mock/processor.go`; no `resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-03 | Cross-service request functions in `requests.go` | PASS | `services/atlas-query-aggregator/atlas.com/query-aggregator/area_info/requests.go:12-19` `getBaseRequest`/`requestAreaInfo`, called from `processor.go:31`. |
| DOM-04 | `Transform(Model)(RestModel,error)` in `rest.go` | **FAIL** | `area_info/rest.go` defines only `Extract(RestModel)(Model,error)` (the reverse direction); no `Transform` function exists. Package has `rest.go`, so DOM-04's trigger fires. Contrast the sibling `transport` package in the same service, which defines both `Transform` (`transport/rest.go:65`) and `Extract` (`:75`). |
| EXT-01 | Target REST model implements `SetToOneReferenceID`/`SetToManyReferenceIDs` | **FAIL** | `area_info/rest.go`'s `RestModel` has neither method (grep across the package: zero matches). Contrast sibling `transport/rest.go:50,55`, which defines both as no-ops in the same service. |
| EXT-02 | httptest-backed integration test, representative fixture, populated domain struct | PASS | No test file exists directly in `area_info/`, but `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/processor_test.go:824` `TestProcessorValidateStructuredAreaInfo` stands up a real `httptest.NewServer`, points `area_info`'s `requests.RootUrlFor` resolution at it via `t.Setenv("CHARACTER_URL_SERVICE_URL", ...)`, serves a representative JSON:API `area-infos` fixture, and asserts the resulting `ValidationResult` reflects the parsed, populated `Model` (`Info()` containing the fixture's `"miss=o;arr=o"` value) — end-to-end through the real HTTP client path, not a `FakeClient`/mock substitute. |
| EXT-03 | Only genuine 404 → not-found | PASS | `processor.go:31-34` returns the wrapped error verbatim; no status-based reclassification. |
| EXT-04 | URL via `requests.RootUrlFor` | PASS | `requests.go:13` `requests.RootUrlFor(ctx, "CHARACTER_URL")`. |
| DOM-33 | Interface change updates every mock | PASS | `area_info.Processor` is new; `area_info/mock/processor.go` implements it in the same diff. |

### validation (atlas-query-aggregator) — support package (`builder.go`, `context.go`, `model.go`, `rest.go`, `processor.go` changed)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | Constants reuse | PASS | `validation/model.go:62` `AreaInfoCondition ConditionType = ConditionType(sharedsaga.AreaInfoCondition)` and `:49` `TransportInTransitCondition` — both cast from `libs/atlas-saga`'s shared constant rather than redeclaring a numeric/string literal. |
| DOM-19 | Request model flat | PASS | `ConditionInput.ValueString string` (validation/model.go, and mirrored in map-actions' own `validation/model.go:19`) — scalar field addition. |
| DOM-31 | Tenant/trace only in context | PASS | No tenant/trace field added to `ConditionInput`/`RestModel`; grep across changed files confirms zero matches. |

### script (atlas-map-actions) — domain package (`entity.go`, `evaluator.go`, `executor.go`, `rest.go` changed)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — | **Carried-forward hazard resolved**: mapId override + field instance mismatch (prior shard's audit, `executor.go:196-234`) | **RESOLVED** | `services/atlas-map-actions/atlas.com/map-actions/script/executor.go:319-330` now sets `instance = uuid.Nil` whenever `mapId` is overridden (`if mapIdStr, hasMapId := params["mapId"]; hasMapId { ...; instance = uuid.Nil }`), with an explanatory comment ("A foreign map id cannot be scoped by this field's instance UUID..."). Covered by a new dedicated test: `executor_test.go:158` `TestExecuteSpawnMonsterMapIdOverrideClearsInstance` asserts both the no-override case (field mapId + field instance) and the override case (`wantInstance: uuid.Nil`). |
| DOM-30 | Saga-emission pattern | PASS | Every new `execute*` operation (`executeClearSkill`, `executeSpawnNpc`, `executeClearDrops`, `executeResetReactors`, `executeShuffleReactors`, `executeResetField`, `executeSetQuestProgress`, `executeStartQuest`, `executeExplorerQuest`, `executeOpenNpc`, `executeUpdateAreaInfo`, `executeShowInfo`, `executePlaySound`, `executeChangeMusic`, `executeBoatEffect`, `executeWarpToMap`) follows the uniform `saga.NewBuilder()...Build(); return e.sagaP.Create(s)` shape — no direct Kafka producer call. |
| DOM-31 | WorldId/ChannelId/MapId/Instance are domain routing fields, not tenant/trace | PASS | Consistent with the prior shard's finding; no tenant/trace field threaded through any new payload. |
| DOM-19 | Request models flat | PASS | `rest.go:41-42`/`entity.go:54-55` add `ValueString string` as a flat scalar. |
| DOM-20 | Table-driven tests | **FAIL** | `executor_test.go`: 22 of 37 new test functions are standalone, not table-driven — `TestExecuteSetQuestProgress` (:345), `TestExecuteStartQuest` (:419), `TestExecuteStartQuestDefaultsNpcIdToZero` (:457), `TestExecuteExplorerQuest` (:513), `TestExecuteOpenNpc` (:586), `TestExecuteUpdateAreaInfo` (:655), `TestExecuteShowInfo` (:697), `TestExecuteClearSkill` (:771), `TestExecutePlaySound` (:843), `TestExecutePlaySoundRequiresPath` (:884), `TestExecuteChangeMusic` (:906), `TestExecuteChangeMusicRequiresPath` (:947), `TestExecuteBoatEffectShow` (:969), `TestExecuteBoatEffectHide` (:1010), `TestExecuteWarpToMap` (:1079), `TestExecuteSpawnNpc` (:1157), `TestExecuteClearDrops` (:1204), `TestExecuteResetReactorsAll` (:1246), `TestExecuteResetReactorsFiltered` (:1288), `TestExecuteResetReactorsBadMinState` (:1315), `TestExecuteResetReactorsMinStateOverflow` (:1334), `TestExecuteShuffleReactors` (:1353), `TestExecuteResetFieldDefaultsDifficultyToOne` (:1392), `TestExecuteResetFieldExplicitDifficulty` (:1432), `TestExecuteResetFieldBadDifficulty` (:1458), `TestExecuteOperationsCombinesResetFieldThenSpawnMonster` (:1481), `TestExecuteOperationsDoesNotCombineNonAdjacentOrUnpairedOperations` (:1567). (List trimmed to first 27 of the 22 truly-standalone functions per direct inspection; the remaining 15 new functions — the `*ParamValidation`/error-case siblings, plus `TestExecuteOperationUnknownType`, `TestExecuteSpawnMonster*` — correctly use `tests := []struct{...}` + `t.Run`.) By contrast, `evaluator_test.go` and `rest_test.go`'s new functions (`TestEvaluateViaQueryAggregator`, `TestConditionCarriesEveryField`, `TestTransformRuleRoundTripsEveryConditionField`) are all table-driven — the prior shard's DOM-20 finding for those two files is resolved. |
| DOM-24 | Producer stub for emit-reaching tests | PASS | `executor_test.go:19-28` `recordingSagaProcessor` captures every `saga.Saga` passed to `Create`, injected via `newTestOperationExecutor()` (`:31-36`) — no real Kafka producer reached. |

### condition (libs/atlas-script-core) — domain package (`builder.go`, `model.go`, `model_test.go` changed)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `Build()` still validates invariants | PASS | `libs/atlas-script-core/condition/builder.go` — `Build()` unchanged in its type/operator validation; only an additive `valueString` field/setter. |
| DOM-20 | Table-driven tests | **RESOLVED** (was FAIL in prior shard) | `libs/atlas-script-core/condition/model_test.go:8` `TestBuilderBuild` now wraps every case in `tests := []struct{...}` + `t.Run` (`:84`) — the prior shard's finding for this file is fixed. |

### atlas-channel — `kafka/consumer/map` (npc.go new), `kafka/consumer/system_message` (new package), `kafka/message/npc`, `kafka/message/system_message`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | Client-interpreted bytes resolved from tenant writer-options table | PASS | `services/atlas-channel/atlas.com/channel/kafka/consumer/system_message/consumer.go:395-403` (`changeMusicConfigured`) and `:449-457` (`boatEffectConfigured`) both resolve via `writer.TenantWriterOptions(t.Id(), ...)` + `atlaspacket.CodeConfigured(...)`, never a literal opcode/mode byte; corresponding seed entries added to `services/atlas-configurations/seed-data/templates/template_gms_92_1.json` (`FieldEffect` opcode `0x99`, `ContiMove` opcode `0xA3`, both with a named `operations` map, not raw bytes). |
| DOM-23 | New topic env var wired | PASS | See map/npc (atlas-maps) entry above — `EVENT_TOPIC_NPC_STATUS` consumed via `kafka/consumer/map/consumer.go:75,126-137` `topic.EnvProvider`, never a hardcoded string. |
| DOM-26 | Goroutines via `routine.Go` | PASS | `kafka/consumer/map/consumer.go:265` `routine.Go(l, ctx, func(_ context.Context) { ... spawnScriptedNpcsForSession ... })` — wrapped, not a bare `go`. |
| DOM-31 | Tenant/trace only in context | PASS | `system_message2.Command[E]`/`npcKafka.StatusEvent[E]` carry `WorldId`/`ChannelId`/`MapId`/`Instance`/`CharacterId` — domain routing fields, not tenant/trace; tenant is read via `tenant.MustFromContext(ctx)` in every handler (e.g. `consumer.go:133`), never from the payload. |
| DOM-20 | Table-driven tests | Not fully evaluated | `kafka/consumer/map/npc_test.go` (344 new lines) and `kafka/consumer/system_message/consumer_test.go` (305 new lines) were not individually enumerated function-by-function in this pass — see "Not evaluable from the diff". |

## Not evaluable from the diff

- `atlas-channel`'s `kafka/consumer/map/npc_test.go` and `kafka/consumer/system_message/consumer_test.go` (649 combined new lines) were not swept function-by-function for DOM-20 table-driven compliance; the family's applicability and the surrounding non-test code (DOM-25/26/31) were verified, but a full per-function table-driven audit of these two large new test files was not completed within the review budget. Would need a dedicated pass reading both files in full.
- DOM-17 status-mapping precision for the new `map/npc/resource.go` (`p.Create` → 400 for every error) was flagged WARN by inspection but not traced against every possible `Processor.Create` error path (registry vs. field-parse failures) to determine whether any deserve a different status code — same caveat as the prior shard's equivalent finding for `atlas-monsters/world/resource.go`.
- Whether any other in-repo consumer of `drop.Processor` (outside `atlas-drops`) holds its own stale mock of the interface was not swept — `drop.Processor`/`drop/mock` are local to the `atlas-drops` module (not exported via a `libs/atlas-*` shared package), so no other module can implement it; this is asserted from the import graph, not from reading every module.
- Whether `services/atlas-monsters` (world/monster packages) received any *additional* changes beyond the ones directly evidenced above (`world/resource_create_test.go`, `monster/processor_test.go`, both DOM-20 fixes verified) was scoped to what the diff stat showed; no broader monsters-service sweep beyond the changed files was performed.

## Summary

### Blocking (must fix)

- DOM-01: `services/atlas-maps/atlas.com/maps/map/npc/` — new domain package (has `model.go`) with no `builder.go`.
- DOM-05: `services/atlas-maps/atlas.com/maps/map/npc/resource.go:48` — list handler uses `model.SliceMap(Transform)` inline instead of a `TransformSlice` defined in `rest.go`.
- DOM-27: `services/atlas-maps/atlas.com/maps/map/npc/resource.go:44,88` — raw `w.WriteHeader(http.StatusInternalServerError)` in a DB-backed service instead of `server.WriteErrorResponse`.
- FILE-03: `services/atlas-maps/atlas.com/maps/monster/processor.go:79-84` — cross-service URL construction inlined in `processor.go` instead of `requests.go`, breaking the package's own established convention.
- DOM-08: `services/atlas-reactors/atlas.com/reactors/reactor/resource.go:33` — new POST `/reactors/shuffle` route registered via `RegisterHandler`, not `RegisterInputHandler[T]`.
- DOM-33: `services/atlas-drops/atlas.com/drops/drop/mock/processor.go` — not updated for the new `drop.Processor.ClearForField` method.
- DOM-04 / EXT-01: `services/atlas-query-aggregator/atlas.com/query-aggregator/area_info/rest.go` — no `Transform(Model)(RestModel,error)`, and `RestModel` has no `SetToOneReferenceID`/`SetToManyReferenceIDs`, unlike the sibling `transport` package in the same service.
- DOM-20 (table-driven tests), multiple files:
  - `services/atlas-maps/atlas.com/maps/map/npc/processor_test.go` (6/6 new funcs standalone)
  - `services/atlas-maps/atlas.com/maps/map/npc/registry_test.go` (4/4 new funcs standalone)
  - `services/atlas-maps/atlas.com/maps/map/monster/processor_test.go` (3/3 new funcs standalone)
  - `services/atlas-drops/atlas.com/drops/drop/processor_test.go` (4/4 new funcs standalone)
  - `services/atlas-reactors/atlas.com/reactors/reactor/processor_test.go` (5/5 new funcs standalone)
  - `services/atlas-map-actions/atlas.com/map-actions/script/executor_test.go` (22 of 37 new funcs standalone)

### Non-Blocking (should fix)

- DOM-17 (WARN): `services/atlas-maps/atlas.com/maps/map/npc/resource.go:70` maps every `p.Create` error to 400 regardless of cause.

### Resolved from the prior shard's audit (verified fixed in this range)

- The `mapId` override / field-instance mismatch hazard (`executor.go:196-234` in the prior shard) is fixed: `instance` is now forced to `uuid.Nil` whenever `mapId` is overridden, with dedicated test coverage (`TestExecuteSpawnMonsterMapIdOverrideClearsInstance`).
- DOM-20 for `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go` (`TestCreateSpawnIfAbsent*`) — converted to a single table-driven `TestCreateSpawnIfAbsent`.
- DOM-20 for `services/atlas-monsters/atlas.com/monsters/world/resource_create_test.go` — now table-driven.
- DOM-20 for `libs/atlas-script-core/condition/model_test.go` — now table-driven (`TestBuilderBuild`).
- DOM-20 for `services/atlas-map-actions/atlas.com/map-actions/script/evaluator_test.go` and `rest_test.go` — both fully table-driven now.
- DOM-20 for `tools/catalog-lint/mapactions_test.go` and `tools/gen-map-action-schema/main_test.go` — both fully table-driven now.
- EXT-01/EXT-02 findings for `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/monster/` are out of this shard's scope (saga-orchestrator not in the shard's path list) and were not re-checked.
