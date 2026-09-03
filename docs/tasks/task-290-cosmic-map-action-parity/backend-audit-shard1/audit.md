# Backend Audit — task-290-cosmic-map-action-parity (shard 1 of 2)

- **Service Path(s):** services/atlas-saga-orchestrator, libs/atlas-saga, services/atlas-quest, services/atlas-character
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-09-03
- **Range:** `3f37fe6da..9c2b9be13`
- **Build:** PASS (module-local, all four modules)
- **Tests:** PASS (module-local `go test ./... -count=1`, all four modules)
- **Overall:** NEEDS-WORK

This shard reviews only the range `3f37fe6da..9c2b9be13`. A prior audit at
`docs/tasks/task-290-cosmic-map-action-parity/backend-audit/audit.md` covers
`9613e7259..3f37fe6da` and is not re-reviewed here, except where this range's
diff visibly addresses one of its findings (noted inline).

## Build & Test Results

```
services/atlas-saga-orchestrator/atlas.com/saga-orchestrator : go build OK, go test OK (all subpackages ok)
libs/atlas-saga                                               : go build OK, go test OK
services/atlas-quest/atlas.com/quest                          : go build OK, go test OK
services/atlas-character/atlas.com/character                  : go build OK, go test OK (pending_change 230s, rest fast)
```
No failures.

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| FILE (FILE-01..06) | Yes | Every changed Go package runs unconditionally. |
| RUNTIME safety (DOM-26) | Yes (family opens; rule N/A) | Non-test Go files changed; no bare `go` statement added (grep across the whole diff found none outside `_test.go`). |
| DOM structure (DOM-01..05,11,16) | Yes | `area_info` (atlas-character) and `medal_map` (atlas-quest) both have `model.go`+`entity.go`+`builder.go`+`provider.go`(area_info only)/rest.go. |
| SUB (SUB-01..04) | No | No changed package has `resource.go` without `model.go`. `area_info`/`medal_map` (both services) have `model.go`; saga-orchestrator's `area_info`, `npc_spawn`, `drops`, `field`, `reactor`, `quest` packages have neither `model.go` nor `resource.go` (support/client packages). |
| REST (DOM-06..09,12..15,17..19,32) | Yes | `area_info/resource.go` (character), `medal_map/resource.go` (quest) register routes; `processor.go` changed in numerous saga-orchestrator packages. |
| Constants reuse (DOM-21) | No | No new type/const-block/numeric-literal classification; all additions are struct fields or `Action`/topic string constants following the existing per-package convention. |
| Testing (DOM-10,20,24,33) | Yes | Diff touches many `_test.go` files; `character.Processor`, `quest.Processor`, `system_message.Processor` interfaces all re-signed/extended with mocks updated in the same diff. |
| Cache (DOM-29) | No | No `cache.go` touched; no new cached processor state. |
| Messaging (DOM-30) | Yes | `character.Processor.WarpToMapAndEmit`, `system_message.Processor.PlaySound/ChangeMusic/BoatEffect` all use `message.Emit`/`producer.ProviderImpl` directly (system_message has no DB write to keep atomic — matches every pre-existing sibling method in the same file). |
| Multi-tenancy (DOM-31) | Yes | `area_info/rest.go` (both services) and `medal_map/rest.go` changed; no tenant/trace field found on any REST model. |
| Migration hygiene (DOM-34,35) | No | No symbol moved between a service and a `libs/atlas-*` module. |
| Deploy & topics (DOM-22,23) | No | No new `libs/atlas-*` module; no Kafka topic env var added/renamed (new `Action`/command-type string constants ride the existing `EnvCommandTopic`/`quest.EnvCommandTopic`). |
| Resilience (DOM-27,28) | Yes | New DB-backed handlers (`area_info/resource.go`, `medal_map/resource.go`) in DB-backed services; no `model.Decorator`/enrichment path changed. |
| External clients (EXT-01..04) | Yes | `saga-orchestrator/area_info`, `saga-orchestrator/quest` (medal-map + quest-data), `saga-orchestrator/npc_spawn`, `saga-orchestrator/drops`, `saga-orchestrator/field`, `saga-orchestrator/reactor` all call another atlas service via `requests.*Request[T]`. |
| Scaffolding (SCAFFOLD-01..09) | No | No new `services/atlas-<svc>/` directory, no new channel Writer/Handler, `deploy/shared/routes.conf` untouched. |
| Security (SEC-01..04) | No | None of the four modules handle auth tokens, revocation, redirects, or secrets. |

## Checklist Results

### area_info (atlas-character) — domain package (model.go+entity.go+builder.go+provider.go+administrator.go+processor.go+rest.go+resource.go, all new)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go / validating Build() | **FAIL** | `services/atlas-character/atlas.com/character/area_info/builder.go:38-44` — `Build()` returns `Model{...}` with no invariant check (e.g. no non-zero `characterId`/`area` check). Sibling `pending_change/builder.go:104` shows the same non-validating style is a known, pre-existing service-wide gap — new code should not perpetuate it without at least matching the "recommended for new code" `BuildWithValidation()` pattern that `services/atlas-quest/atlas.com/quest/quest/builder.go:94-96` documents as the fleet's own remediation for this exact rule. |
| DOM-02 | entity.go `ToEntity()` | **FAIL** | No `func (m Model) ToEntity()` anywhere in `area_info/entity.go` (grep across the package returned zero matches). |
| DOM-03 | entity.go `Make(Entity) (Model, error)` | **FAIL** | No `func Make(` anywhere in `area_info/entity.go`; conversion instead happens via `modelFromEntity` in `provider.go:3-9`, wrong name and (per FILE-05) arguably wrong file too. |
| DOM-04/05 | rest.go Transform/TransformSlice | PASS | `area_info/rest.go:15` `Transform`; `area_info/rest.go:32` `TransformSlice`; `resource.go:57` calls `TransformSlice`, no inline loop. |
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | `area_info/processor.go:20` `NewProcessor(l logrus.FieldLogger, ...)`. |
| DOM-07 | Handler passes `d.Logger()` into `NewProcessor` | PASS | `area_info/resource.go:51,75,110` `NewProcessor(d.Logger(), d.Context(), d.DB())`. |
| DOM-08 | POST/PATCH via `RegisterInputHandler[T]` | N/A | Package registers GET/GET/PUT only, no POST/PATCH route. |
| DOM-09 | Transform error-checked | PASS | `resource.go:57,60` and `resource.go:85,88` and `resource.go:116,119` all check `err` immediately after `Transform`/`TransformSlice`/`model.Map(Transform)`. |
| DOM-11 | provider.go readers are lazy `database.Query`/`SliceQuery` | N/A | `provider.go` contains only `modelFromEntity` (a mapper, no query function at all) — the package's actual readers are misplaced in `administrator.go` (see FILE-05 FAIL below), so DOM-11's FixedProvider-misuse check has nothing to grade in this file. |
| DOM-12 | No `os.Getenv` in handlers | PASS | Grep of `resource.go` for `os.Getenv` returns nothing. |
| DOM-13 | No cross-domain orchestration in handlers | PASS | Each handler calls exactly one `Processor` method. |
| DOM-14 | Handlers call processor, not provider | PASS | All three handlers call `NewProcessor(...).Get*`/`Put`, never a provider function directly. |
| DOM-15 | No `db.Create`/`Save`/`Delete` in handlers | PASS | None present in `resource.go`. |
| DOM-16 | administrator.go holds writes | PASS | `area_info/administrator.go:8` `upsert(...)` is the package's only write, called from `processor.go:26`. |
| DOM-17 | Domain errors → specific status | PASS | `resource.go:77-80` maps `gorm.ErrRecordNotFound` to 404 explicitly before falling through to `WriteErrorResponse`. |
| DOM-18 | REST models implement JSON:API interface | PASS | `RestModel` (`rest.go:8,13,17`) and `PutRestModel` (`rest.go:54,58,62`) both have `GetName`/`GetID`/`SetID`. |
| DOM-19 | Request models flat | PASS | `PutRestModel` (`rest.go:52-56`) is `{Id, Info}`, no nesting. |
| DOM-27 | 500 branches use `WriteErrorResponse` in DB-backed service | PASS | `resource.go:53,60,81,88,112,119` all use `server.WriteErrorResponse`; the only bare `w.WriteHeader` calls are the 400/404 branches (`resource.go:35,41,78`), which DOM-27 exempts. `main.go:89` registers `server.RegisterTransientErrorClassifier` (pre-existing). |
| DOM-31 | Tenant/trace only in context | PASS | `RestModel`/`PutRestModel` carry `CharacterId`/`Area`/`Info` only; tenant comes from `d.Context()` via the pre-existing `rest.RegisterHandler`/`RegisterInputHandler` wrapper. |
| DOM-32 | Routes via `server.RegisterHandler`/`RegisterInputHandler[T]` | PASS | `resource.go:21,25` uses the service-local `rest.RegisterHandler`/`rest.RegisterInputHandler[T]` aliases; traced to `services/atlas-character/atlas.com/character/rest/handler.go:68-95`, which composes `server.RetrieveSpan` + `server.ParseTenant` (the same primitives `server.RegisterHandler` itself composes) — the dominant fleet idiom the guideline explicitly calls out as not a deviation. No manual tenant-header parsing, no custom error-response helper (uses `server.WriteErrorResponse` directly). |
| FILE-01 | Processor in processor.go | PASS | `area_info/processor.go` holds `Processor`/`ProcessorImpl`/`NewProcessor` and all methods. |
| FILE-02 | RestModel in rest.go | PASS | `area_info/rest.go` holds both REST models and `Transform`/`Extract`/`TransformSlice`. |
| FILE-03 | Cross-service requests in requests.go | N/A | Package makes no calls to another atlas service. |
| FILE-04 | entity struct/Migration/TableName in entity.go | PASS | `area_info/entity.go:9,16,21` all three present. |
| FILE-05 | Builder/Model/writes/readers correctly placed | **FAIL** | `area_info/administrator.go:19` (`getByCharacterIdAndArea`) and `administrator.go:28` (`getAllByCharacterId`) are readers placed in `administrator.go` instead of `provider.go`. `provider.go` (10 lines) holds only the entity→Model mapper. Confirmed against this service's own established convention: `services/atlas-character/atlas.com/character/pending_change/provider.go:39,51,67,79,95,111` places every reader (`getById`, `getByCharacterId`, `getPendingByNameLower`, etc.) in `provider.go`, while `pending_change/administrator.go:24,76,98` holds only writes (`create`, `transition`, `markNotified`). |
| FILE-06 | No catch-all file | PASS | Each new file carries exactly one responsibility (builder/entity/model/processor/provider/administrator/rest/resource) — the FILE-05 misplacement above is a placement error, not a multi-responsibility collapse. |
| DOM-20 | Table-driven tests | **FAIL** | `area_info/processor_test.go`: all 5 new funcs standalone, none use `tests := []struct{...}` + `t.Run` — `TestUpsertAreaInfoReplacesWholeString` (:58), `TestGetByAreaMissingReturnsEmpty` (:82), `TestAreaInfoIsPerCharacter` (:96), `TestAreaInfoIsTenantScoped` (:121). |
| DOM-24 | Producer stub for emit-reaching tests | N/A | No test in this package reaches an `AndEmit`/`message.Emit`/producer path — `area_info` never emits Kafka. |

### medal_map (atlas-quest) — domain package (model.go+entity.go+builder.go+administrator.go+processor.go+rest.go+resource.go, all new)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go / validating Build() | **FAIL** | `services/atlas-quest/atlas.com/quest/medal_map/builder.go:38-44` — `Build()` returns `Model{...}` with no invariant enforcement, same gap as area_info above. `services/atlas-quest/atlas.com/quest/quest/builder.go:94-96`'s own `BuildWithValidation()` shows this exact service's fleet remediation for the rule; medal_map (new code in this diff) does not adopt it. |
| DOM-02 | entity.go `ToEntity()` | **FAIL** | No `ToEntity()` in `medal_map/entity.go` (grep returned nothing). |
| DOM-03 | entity.go `Make(Entity) (Model, error)` | **FAIL** | No `func Make(` in `medal_map/entity.go`. Sibling `services/atlas-quest/atlas.com/quest/quest/entity.go:33` `func Make(e Entity) (Model, error)` shows the pattern exists and is followed elsewhere in this exact service; medal_map's own `administrator.go` builds `entity{...}` literals directly instead. |
| DOM-04/05 | rest.go Transform | PASS | `medal_map/rest.go:47` `Transform(result RecordResult) (RestModel, error)`; no `TransformSlice` needed (single-record response). |
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | `medal_map/processor.go:23` `NewProcessor(l logrus.FieldLogger, ...)`. |
| DOM-07 | Handler passes `d.Logger()` | PASS | `medal_map/resource.go:18` `NewProcessor(d.Logger(), d.Context(), d.DB())`. |
| DOM-08 | POST via `RegisterInputHandler[T]` | PASS | `medal_map/resource.go:18` `rest.RegisterInputHandler[PostRestModel](l)(db)(si)(...)`. |
| DOM-09 | Transform error-checked | PASS | `resource.go` (per diff) checks `err` immediately after `model.Map(Transform)(...)`. |
| DOM-11 | provider.go lazy readers | N/A | No `provider.go` exists in this package at all — the reader lives in `administrator.go` (see FILE-05 FAIL). |
| DOM-12/13/14/15 | Handler hygiene | PASS | No `os.Getenv`, no cross-domain orchestration, calls only `Processor.Record`, no direct DB write. |
| DOM-16 | administrator.go holds writes | PASS | `medal_map/administrator.go:14` `recordIfAbsent(...)`. |
| DOM-17 | Domain errors → specific status | PASS (by absence) | `Record` only surfaces DB errors; no validation/not-found class exists for this endpoint, so there is nothing to mis-map. |
| DOM-18/19 | REST models / flat requests | PASS | `PostRestModel`/`RestModel` (`rest.go:5,26`) both implement `GetName`/`GetID`/`SetID`; both flat. |
| DOM-27 | 500 branches use `WriteErrorResponse` | PASS | `resource.go` error branches use `server.WriteErrorResponse`; `main.go:67` registers the transient classifier (pre-existing). |
| DOM-31 | Tenant/trace only in context | PASS | `PostRestModel`/`RestModel` carry no tenant/trace field. |
| DOM-32 | Routes via wrapper delegating to server primitives | PASS | Same local-wrapper pattern as area_info, traced to `services/atlas-quest/atlas.com/quest/rest/handler.go:68-95` (same shape as atlas-character's). |
| FILE-01/02/04 | Processor/RestModel/entity placement | PASS | Each in its own file. |
| FILE-03 | Cross-service requests | N/A | Package makes no external-service calls. |
| FILE-05 | Builder/Model/writes/readers correctly placed | **FAIL** | `medal_map/administrator.go:27` (`countByCharacterAndQuest`, a `db.Model(&entity{}).Where(...).Count(...)` reader) is placed in `administrator.go` alongside the write `recordIfAbsent` (`administrator.go:14`); the package has **no `provider.go` at all**. Confirmed against this service's own convention: `services/atlas-quest/atlas.com/quest/quest/provider.go:11,27,33,44,61` places every reader in `provider.go`, while `quest/administrator.go:13,30,55,74,96,134,156,178` holds only writes. |
| FILE-06 | No catch-all file | PASS | No single file combines ≥2 of the graded responsibilities. |
| DOM-20 | Table-driven tests | **FAIL** | `medal_map/processor_test.go`: all 4 new funcs standalone — `TestRecordMedalMapDeduplicates` (:58), `TestRecordMedalMapCountsDistinctMaps` (:80), `TestMedalMapsArePerQuest` (:99), `TestMedalMapsArePerCharacterAndTenant` (:117). None use `tests := []struct{...}` + `t.Run`. |
| DOM-24 | Producer stub for emit-reaching tests | N/A | `Record` never emits Kafka. |

### saga-orchestrator/area_info — support/client package (processor.go, requests.go, rest.go, mock/processor.go — no model.go, no resource.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in processor.go | PASS | `area_info/processor.go:9-14,26` `Processor`/`ProcessorImpl`/`NewProcessor`/`Put`. |
| FILE-02 | RestModel in rest.go | PASS | `area_info/rest.go`. |
| FILE-03 | Cross-service requests in requests.go | PASS | `area_info/requests.go:17,30` `PutAreaInfo`/`GetAreaInfo`. |
| FILE-06 | No catch-all file | PASS | Three files, one responsibility each. |
| DOM-33 | Interface change updates every mock | PASS | `area_info/mock/processor.go:8-19` implements the new `Processor.Put` exactly; `var _ area_info.Processor = (*ProcessorMock)(nil)` compiles. |
| EXT-01 | Target REST model implements SetToOneReferenceID/SetToManyReferenceIDs | **FAIL** | `area_info/rest.go` `RestModel` (used as the `PutRequest[RestModel]`/`GetRequest[RestModel]` target type in `requests.go:19,31`) has no `SetToOneReferenceID`/`SetToManyReferenceIDs` (grep returned nothing). |
| EXT-02 | httptest-backed integration test | **FAIL** | No `_test.go` file exists anywhere under `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/area_info/` (directory listing confirmed empty of test files). No coverage of `PutAreaInfo`/`GetAreaInfo` against a representative fixture. |
| EXT-03 | Only genuine 404 → not-found | PASS | `area_info/processor.go:28-31` logs and returns the `PutAreaInfo` error verbatim, no status-based reclassification. |
| EXT-04 | URL via `requests.RootUrlFor` | PASS | `area_info/requests.go:15` `requests.RootUrlFor(ctx, BaseUrl)` where `BaseUrl = "CHARACTER_URL"` (`requests.go:12`). |
| DOM-20 | Table-driven tests | N/A | No test file exists to grade (the gap is EXT-02's absence of tests, not their shape). |

### saga-orchestrator/quest — support/client package additions (medal_map_requests.go, medal_map_rest.go, quest_data_requests.go, quest_data_rest.go, processor.go changes, mock/processor_mock.go new)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in processor.go | PASS | `quest/processor.go` holds `RequestExplorerQuest` alongside the pre-existing interface methods. |
| FILE-02 | RestModel in rest.go(-suffixed files) | PASS | `medal_map_rest.go`, `quest_data_rest.go` hold their respective REST models; `medalMapPostRestModel`/`medalMapRestModel`/`questDataRestModel` all implement `GetName`/`GetID`/`SetID`. |
| FILE-03 | Cross-service requests in requests.go(-suffixed files) | PASS | `medal_map_requests.go`, `quest_data_requests.go` hold `postMedalMap`/`requestQuestData`. |
| FILE-06 | No catch-all file | PASS | Clean four-way split, unlike `field`/`reactor` below. |
| DOM-33 | Interface change updates every mock | PASS | `quest/mock/processor_mock.go:24-30,53-56` (new file) implements the extended `Processor` interface including `RequestExplorerQuest`; `var _ questp.Processor = (*ProcessorMock)(nil)` compiles. |
| EXT-01 | Target REST model implements SetToOneReferenceID/SetToManyReferenceIDs | **FAIL** | Neither `medalMapRestModel` (`medal_map_rest.go:23-46`) nor `questDataRestModel` (`quest_data_rest.go:14-32`) implement `SetToOneReferenceID`/`SetToManyReferenceIDs` (grep across both files returned nothing). |
| EXT-02 | httptest-backed integration test | **FAIL** | No `_test.go` file exists at the top level of `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/quest/` (only the pre-existing `quest/state/*_test.go` subpackage tests). `postMedalMap`, `requestQuestData`, and the whole of `RequestExplorerQuest` — a new method that force-starts a quest, records a medal map, and conditionally resolves+writes progress — ship with **zero** unit or integration coverage in this package; the only coverage of `RequestExplorerQuest`'s behavior is indirect, through the saga handler's mocked `quest.Processor` in `saga/handler_test.go` (see below), which never exercises the real HTTP-calling implementation. |
| EXT-03 | Only genuine 404 → not-found | PASS | `quest/processor.go`'s `RequestExplorerQuest` logs and returns `postMedalMap`/`requestQuestData` errors verbatim, no status-based reclassification. |
| EXT-04 | URL via `requests.RootUrlFor` | PASS | `medal_map_requests.go:16` (`"QUESTS"`) and `quest_data_requests.go:15` (`"DATA"`), both `requests.RootUrlFor(ctx, ...)`. |
| DOM-20 | Table-driven tests | N/A | No test file exists in this package to grade (see EXT-02 above). |

### saga-orchestrator/npc_spawn — support/client package (new)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/02/03 | Processor/RestModel/requests split | PASS | `npc_spawn/processor.go` (Processor only), `npc_spawn/rest.go` (`SpawnInputRestModel`/`SpawnResponseRestModel`/`SpawnRequest`), `npc_spawn/requests.go` (`requestSpawnNpc`). Clean split — the correct counter-example to `field`/`reactor` below. |
| EXT-01 | Target model implements SetToOneReferenceID/SetToManyReferenceIDs | **FAIL** | `npc_spawn/rest.go:32-53` — grep for `SetToOneReferenceID`/`SetToManyReferenceIDs` in this file returns nothing; `SpawnResponseRestModel` has `GetName`/`GetID`/`SetID` only. Contrast with `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/monster/rest.go:63-67`, changed in this same diff, which explicitly adds both stub methods with a comment citing the exact api2go gotcha this rule exists for — `npc_spawn`'s otherwise-identical `SpawnResponseRestModel` does not get the same fix. |
| EXT-02 | httptest-backed integration test | **FAIL** | No `_test.go` file exists under `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/npc_spawn/` at all. |
| EXT-03 | Only genuine 404 → not-found | PASS | `npc_spawn/processor.go:29-32` logs and returns the error verbatim. |
| EXT-04 | URL via `requests.RootUrlFor` | PASS | `npc_spawn/requests.go:11` `requests.RootUrlFor(ctx, "MAPS")`. |

### saga-orchestrator/drops — support/client package (new)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/03 | Processor/requests split | PASS | `drops/processor.go` (Processor only), `drops/requests.go` (`requestClearDrops`). No `rest.go` needed — DELETE with no response body. |
| EXT-01 | Target model relationship stubs | N/A | `requestClearDrops` returns `requests.EmptyBodyRequest` (a bodyless DELETE, `requests/delete.go:14`) — no JSON:API response is unmarshaled, so there is no target REST model to grade. |
| EXT-02 | httptest-backed integration test | PASS | `drops/drops_test.go:24` `TestClearDropsIssuesDeleteToTheRightField` and `:61` `TestClearDropsPropagatesUpstreamFailure` both use `httptest.NewServer` and assert real HTTP method/path/error-propagation behavior — the strongest EXT-02 coverage in this shard. |
| EXT-03 | Only genuine handling of failure | PASS | `drops_test.go:61-80` explicitly asserts a non-2xx/204 response surfaces as an error, not swallowed. |
| EXT-04 | URL via `requests.RootUrlFor` | PASS | `drops/requests.go:15` `requests.RootUrlFor(ctx, "DROPS")`. |
| DOM-20 | Table-driven tests | **FAIL** | `drops/drops_test.go`: both new funcs standalone (`TestClearDropsIssuesDeleteToTheRightField` :24, `TestClearDropsPropagatesUpstreamFailure` :61), neither uses `tests := []struct{...}` + `t.Run`. |

### saga-orchestrator/field — support/client package (new)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in processor.go | PASS | `field/processor.go:14-19,26,34` `Processor`/`ProcessorImpl`/`NewProcessor`/`ResetField`. |
| FILE-02 | RestModel in rest.go | **FAIL** | `field/processor.go:44-58` defines `ResetFieldInputRestModel` with its `GetName`/`GetID` methods inside `processor.go` — there is no `rest.go` in this package at all (directory listing: `processor.go`, `requests.go` only). |
| FILE-03 | Cross-service requests in requests.go | **FAIL** | `field/processor.go:36-39,60-67` defines `getBaseRequest` and `requestResetField` inside `processor.go` — the package has no `requests.go` at all (`ls services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/field/` shows only `processor.go`). |
| FILE-06 | No catch-all file | **FAIL** | `field/processor.go` (70 lines, entirely new) carries all three of Processor + RestModel + requests responsibilities — the exact `task-102 wallet.go` collapse pattern the guideline calls out by name. |
| EXT-01 | Target model relationship stubs | N/A | `requestResetField` targets `requests.Request[struct{}]` (`processor.go:60`) — an empty response type with no fields to decode; `MakePostRequest`'s `createOrUpdate` skips `unmarshalResponse` entirely when `contentLength == 0` (`libs/atlas-rest/requests/post.go:88-96`), so no relationship-stub gap exists here. |
| EXT-02 | httptest-backed integration test | **FAIL** | No `_test.go` file exists under `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/field/` at all. |
| EXT-03 | Only genuine 404 → not-found | PASS | `field/processor.go:26-29` wraps and returns the error verbatim, no reclassification. |
| EXT-04 | URL via `requests.RootUrlFor` | PASS | `field/processor.go:38` `requests.RootUrlFor(ctx, "MAPS")`. |

### saga-orchestrator/reactor — support/client package (pre-existing `processor.go`, extended by this diff)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-02/03/06 | RestModel / requests placement, no catch-all | **FAIL** | `reactor/processor.go` is (and, per its own pre-existing shape before this diff, already was) the package's only file — `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/` contains just `processor.go` and a `drop/` subpackage. This diff adds two more REST models (`ResetReactorsInputRestModel`, `ShuffleReactorsInputRestModel`, `processor.go:199-217`) and two more request functions (`requestResetReactors`, `requestShuffleReactors`, `processor.go:229-243`) into that same collapsed file, growing rather than fixing the FILE-02/FILE-03/FILE-06 collapse. Per the Mindset guidance, prevalence of the pre-existing collapse does not exempt new content added to it in this diff. |
| EXT-01 | Target model relationship stubs | N/A | Both new request functions target `requests.Request[struct{}]` (`processor.go:236,242`) — same no-body-to-decode situation as `field` above. |
| EXT-02 | httptest-backed integration test | **FAIL** | No `_test.go` file exists under `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/` (top level) at all — `ResetReactors`/`ShuffleReactors` are entirely untested. |
| EXT-03 | Only genuine 404 → not-found | PASS | `processor.go:93,102` wrap and return the error verbatim. |
| EXT-04 | URL via `requests.RootUrlFor` | PASS | `getReactorsBaseRequest` (pre-existing, `processor.go:185`) used by both new functions. |

### saga-orchestrator/character, /skill, /monster, /system_message — interface extensions to pre-existing packages

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-33 | Interface change updates every mock | PASS | `character/mock/processor.go` gains `WarpToMapAndEmitFunc`/`WarpToMapFunc` matching `character/processor.go:26-27`'s new interface methods; `system_message/mock/processor.go` (new file) implements all 14 `system_message.Processor` methods including the 3 new ones; both compile via `var _ X.Processor = (*ProcessorMock)(nil)`. |
| DOM-30 | AndEmit + message.Buffer for DB-coupled writes | PASS | `character.Processor.WarpToMapAndEmit` (`character/processor.go:108-112`) uses `message.Emit(p.p)(...)`, matching every sibling `*AndEmit` method in the file. `system_message.Processor.PlaySound/ChangeMusic/BoatEffect` (`system_message/processor.go:117-129`) call `producer.ProviderImpl(...)` directly with no DB write on this path — matches every pre-existing sibling in the same file (e.g. `UiLock`, `processor.go:109-111`); DOM-30's substantive requirement (atomicity between a DB write and its event) does not apply because there is no DB write. |
| EXT-01 (fix confirmed) | `monster` REST model relationship stubs | PASS (fixes prior audit finding) | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/monster/rest.go:63-67` — `SpawnResponseRestModel.SetToOneReferenceID`/`SetToManyReferenceIDs` added in this diff, closing the EXT-01 blocking finding from the prior shard's audit (`docs/tasks/task-290-cosmic-map-action-parity/backend-audit/audit.md` line 171). |
| DOM-20 | Table-driven tests | N/A | No new `_test.go` content in `character`, `skill`, or `monster` (beyond the `rest.go` fix above) in this shard's range; `system_message` has no new test file either. |

### saga (saga-orchestrator) — handler.go / model.go / event_acceptance.go / character_extractor.go / unmarshal wiring

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-31 | Tenant/trace only in context | PASS | All new payload types (`WarpToMapPayload`, `ExplorerQuestPayload`, `SpawnNpcPayload`, `ClearDropsPayload`, `ResetReactorsPayload`, `ShuffleReactorsPayload`, `ResetFieldPayload`, `PlaySoundPayload`, `ChangeMusicPayload`, `BoatEffectPayload`, `ClearSkillPayload` — all in `libs/atlas-saga/payloads.go`) carry `WorldId`/`ChannelId`/domain fields, never a tenant or trace identifier. |
| DOM-33 | Interface change updates every mock | PASS | `Handler` interface gains `WithNpcSpawnProcessor`/`WithDropsProcessor`/`WithFieldProcessor`/`WithAreaInfoProcessor`; `HandlerImpl` implements all four (`handler.go:332,887,899,910`) — no separate mock of `Handler` exists in the repo (it is the top-level orchestration type, not injected elsewhere), so there is nothing further to update. |
| — | **Handler struct-copy bug in three new injector methods** | **FAIL (blocking, no rule ID — see dedicated section below)** | `saga/handler.go:887-916` (`WithNpcSpawnProcessor`, `WithDropsProcessor`, `WithFieldProcessor`). |
| — | **Stale exhaustiveness test — 9 of 10 new Actions missing from `allActions`** | **FAIL (blocking, no rule ID — see dedicated section below)** | `saga/event_acceptance_test.go:14-64` (`allActions`), vs. `libs/atlas-saga/model.go`'s 10 new `Action` constants. |
| DOM-20 | Table-driven tests | **FAIL** | `saga/handler_test.go`: 4 new funcs standalone, none use `tests := []struct{...}` + `t.Run` — `TestHandleUpdateAreaInfo` (:1841), `TestHandleExplorerQuest_Routing` (:1899), `TestHandleExplorerQuest_WritesProgressWhenNewlyRecorded` (:1948), `TestHandleExplorerQuest_SkipsProgressWhenNotNewlyRecorded` (:2001). Contrast with this same file's own pre-existing convention for handler tests, e.g. `TestHandleWarpToPortal` (:170) and `TestHandleAwardAsset` (:353), both `tests := []struct{...}` + `t.Run`. |
| — | **Zero test coverage for 10 of 14 new/changed handler methods** | Non-blocking (see dedicated section) | `handleWarpToMap`, `handleClearSkill`, `handleSpawnNpc`, `handleClearDrops`, `handleResetReactors`, `handleShuffleReactors`, `handleResetField`, `handlePlaySound`, `handleChangeMusic`, `handleBoatEffect` have no test anywhere in `saga/handler_test.go` (grep for each handler name and for `WithNpcSpawnProcessor`/`WithDropsProcessor`/`WithFieldProcessor` in the test file returns nothing). Only `handleUpdateAreaInfo` and `handleExplorerQuest` (of the 12 new handlers this diff adds) are exercised. |
| DOM-24 | Producer stub for emit-reaching tests | PASS | `TestHandleUpdateAreaInfo`/`TestHandleExplorerQuest_*` inject `ProcessorMock`s for every processor on the emit path (`systemMessageMock.ProcessorMock`, `questmock.ProcessorMock`), never reaching a real Kafka producer. |

### libs/atlas-saga — model.go, payloads.go, validation.go, unmarshal.go (+ tests)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-19 | Payload structs flat | PASS | All 11 new payload types (`WarpToMapPayload`, `ClearSkillPayload`, `ExplorerQuestPayload`, `PlaySoundPayload`, `ChangeMusicPayload`, `BoatEffectPayload`, `SpawnNpcPayload`, `ClearDropsPayload`, `ResetReactorsPayload`, `ShuffleReactorsPayload`, `ResetFieldPayload`) are flat scalar/pointer fields, no nesting. |
| DOM-21 | No redeclared constant/type | PASS | New `Action`/condition constants (`WarpToMap`, `ClearSkill`, `ExplorerQuest`, `PlaySound`, `ChangeMusic`, `BoatEffect`, `SpawnNpc`, `ClearDrops`, `ResetReactors`, `ShuffleReactors`, `ResetField`, `TransportInTransitCondition`, `AreaInfoCondition`) are new string values in the existing `Action`/condition typed-const blocks — no collision with `libs/atlas-constants`. |
| DOM-20 | Table-driven tests | **FAIL** (new file) / **PASS** (fixed pre-existing file) | `libs/atlas-saga/unmarshal_test.go` (entirely new, 395 lines): 52 of 53 `func Test...` are standalone, only `TestUnmarshalRebalanceAPStep` (:17) uses `tests := []struct{...}` + `t.Run` — e.g. `TestCreateAndEquipAssetPayload_UseAverageStats_RoundTrip` (:170), `TestUnmarshalAwaitInventoryCreatedStep` (:230), `TestUnmarshalEvolvePetStep` (:258), and 49 more of the same shape. Conversely, `libs/atlas-saga/payloads_test.go` was **rewritten in this diff to table-driven form** (`TestDestroyAssetFromSlotPayloadTemplateIdRoundTrip`, `TestSkillBookUseSagaType`, `TestCharacterCreatePayload_GmAndMeso_RoundTrip` all converted to `tests := []struct{...}` + `t.Run`), closing the prior shard's audit finding for that file (`docs/tasks/task-290-cosmic-map-action-parity/backend-audit/audit.md` line 124). |
| DOM-24 | Producer stub | N/A | Pure payload/marshal library code, no emit path. |

## Two findings with no numbered rule ID

Per the reviewer mindset, a genuine defect discovered while reviewing an
applicable family's surface is reported even without a matching rule ID —
both below surfaced while grading DOM-33/Testing.

### 1. Handler struct-copy bug — `WithNpcSpawnProcessor`/`WithDropsProcessor`/`WithFieldProcessor` silently drop every other processor field

`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go:887-916`:

```go
func (h *HandlerImpl) WithNpcSpawnProcessor(npcSpawnP npc_spawn.Processor) Handler {
	return &HandlerImpl{
		l:         h.l,
		ctx:       h.ctx,
		t:         h.t,
		footholdP: h.footholdP,
		npcSpawnP: npcSpawnP,
	}
}

func (h *HandlerImpl) WithDropsProcessor(dropsP drops.Processor) Handler {
	return &HandlerImpl{
		l:      h.l,
		ctx:    h.ctx,
		t:      h.t,
		dropsP: dropsP,
	}
}

func (h *HandlerImpl) WithFieldProcessor(fieldP fieldclient.Processor) Handler {
	return &HandlerImpl{
		l:      h.l,
		ctx:    h.ctx,
		t:      h.t,
		fieldP: fieldP,
	}
}
```

All three construct a brand-new `&HandlerImpl{}` literal naming only a
handful of fields, which zeroes every other processor field (`charP`,
`monsterP`, `questP`, `partyP`, ... — 30+ fields on `HandlerImpl`). This is
the exact defect class the diff's own `WithAreaInfoProcessor` doc comment
(`handler.go:330-331`) calls out by name:

> "WithAreaInfoProcessor uses the same shallow-copy form as WithMtsProcessor,
> and for the reason spelled out there: the field-by-field siblings silently
> nil any field they forget."

`WithAreaInfoProcessor` (`handler.go:332-336`) correctly uses `c := *h;
c.areaInfoP = areaInfoP; return &c` — the shallow-copy form that preserves
every already-set field. The three other new injectors added in this same
diff do not, including one (`WithNpcSpawnProcessor`) that copies an
apparently-unrelated `footholdP` field alongside the new one — visible
evidence of copy-paste from a different, unrelated `With*` method rather than
deliberate design.

**Blast radius:** these three methods are declared as `Handler` interface
injector seams (doc comments: "injector seam for handle*'s ... call") but are
never actually called anywhere in the repository outside their own
declaration — confirmed by grep across the whole module. `NewHandler`
(`handler.go:319-336`) already sets every field including the three new
processors at construction, so production code has no reason to call them.
But their entire stated purpose is dependency injection for **tests** (the
same role `WithAreaInfoProcessor`/`WithSystemMessageProcessor`/
`WithQuestProcessor` play in this diff's own `TestHandleUpdateAreaInfo`/
`TestHandleExplorerQuest_*`), and no test in this diff exercises any of the
three. If a future test (or production caller) chains, e.g.,
`.WithCharacterProcessor(x).WithNpcSpawnProcessor(y)`, `charP` silently
becomes `nil` and the next handler invocation on that `Handler` panics on a
nil-pointer method call — with no compiler error, no test failure today, and
no signal until the exact combination is exercised.

**Severity: Important (blocking).** Dead code today with a self-evident bug
that will surface as a confusing nil-pointer panic in whichever future test
or code path is the first to actually use these seams for their stated
purpose — exactly the kind of "builds and unit-tests clean while being
unreachable/broken in production" defect class `TestHandleExplorerQuest_Routing`'s
own doc comment (`handler_test.go:1836-1839`) explicitly names as the thing
this diff was trying to guard against elsewhere.

### 2. `TestAcceptanceTable_EveryActionRepresented`'s `allActions` list not updated for 9 of 10 new Actions

`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance_test.go:14-64`
declares `allActions`, with its own doc comment: "Keep this list in sync when
new Actions are added. The coverage test below fails if acceptanceTable
lacks an entry for any of these." This diff adds 10 new `Action` constants to
`libs/atlas-saga/model.go` (`WarpToMap`, `ClearSkill`, `ExplorerQuest`,
`PlaySound`, `ChangeMusic`, `BoatEffect`, `SpawnNpc`, `ClearDrops`,
`ResetReactors`, `ShuffleReactors`, `ResetField`), but `allActions`
(`event_acceptance_test.go:21`) only gained `sharedsaga.ClearSkill`. The
other 9 are absent from the list, so `TestAcceptanceTable_EveryActionRepresented`
does not actually verify `acceptanceTable` has entries for them.

This diff's own `event_acceptance.go` changes do add correct entries for all
10 (confirmed present at `event_acceptance.go:203-217,333,345-347,355-359`),
so there is no live defect today — but the exhaustiveness test that exists
specifically to catch a **future** regression of this kind (an `Action`
constant added without a matching `acceptanceTable` entry, silently leaving
that saga step permanently `Pending`) no longer covers 9 of the service's
newest, least-battle-tested actions. This is the same "routing/table
omission passes tests" defect class the diff's own new
`TestHandleExplorerQuest_Routing` test was written to guard against for one
specific action — the guard for the other 9 was left incomplete.

**Severity: Important (blocking).** Not a live behavioral bug, but a
regression-guard gap in a test whose entire purpose is guarding against
exactly this class of gap, introduced in the same diff that added the
actions it fails to cover.

## Security Review

SEC-* family did not fire — none of the four modules in this shard
(`atlas-saga-orchestrator`, `atlas-saga`, `atlas-quest`, `atlas-character`)
handle authentication, tokens, revocation, redirects, or secrets in the
changed code.

## Not evaluable from the diff

- Whether `RequestExplorerQuest`'s `strconv.Atoi(questData.EndRequirements.InfoEx[0])` silent-fallback-to-0 (`quest/processor.go`) matches atlas-data's actual `infoEx` wire format was not checked — that would require reading atlas-data's quest REST model/response shape, which is outside every service in this shard's scope.
- Whether any other in-repo caller besides `saga/handler.go` invokes `character.Processor.WarpToMap`/`WarpToMapAndEmit`, `system_message.Processor.PlaySound/ChangeMusic/BoatEffect`, or the three buggy `With*Processor` injectors in a chained pattern that would trigger the struct-copy bug today was checked only within this shard's four modules (none found); a repo-wide grep outside these modules was not performed.
- Whether `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/processor.go`'s pre-existing FILE-02/03/06 collapse (present before this diff) was already flagged by an earlier audit of that package was not checked — this shard's prior audit (`backend-audit/audit.md`) did not cover `atlas-saga-orchestrator/reactor` at all (it covered `monster`/`world` in `atlas-monsters`, not `reactor` in `atlas-saga-orchestrator`), so this is reported fresh here rather than treated as a duplicate.

## Summary

### Blocking (must fix)

- DOM-01: `services/atlas-character/atlas.com/character/area_info/builder.go:38-44` — `Build()` has no invariant validation.
- DOM-01: `services/atlas-quest/atlas.com/quest/medal_map/builder.go:38-44` — same.
- DOM-02/DOM-03: `services/atlas-character/atlas.com/character/area_info/entity.go` — no `ToEntity()`/`Make()`.
- DOM-02/DOM-03: `services/atlas-quest/atlas.com/quest/medal_map/entity.go` — same.
- FILE-05: `services/atlas-character/atlas.com/character/area_info/administrator.go:19,28` — readers misplaced, belong in `provider.go`.
- FILE-05: `services/atlas-quest/atlas.com/quest/medal_map/administrator.go:27` — reader misplaced, no `provider.go` exists.
- FILE-02/FILE-03/FILE-06: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/field/processor.go` — Processor + RestModel + requests collapsed into one file (task-102 pattern).
- FILE-02/FILE-03/FILE-06: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/processor.go:199-243` — this diff adds more REST-model/request content to an already-collapsed file.
- EXT-01: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/area_info/rest.go` — `RestModel` missing relationship stubs.
- EXT-01: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/quest/medal_map_rest.go` and `quest_data_rest.go` — both target models missing relationship stubs.
- EXT-01: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/npc_spawn/rest.go:32-53` — `SpawnResponseRestModel` missing relationship stubs (sibling `monster/rest.go` got the fix in this same diff, `npc_spawn` did not).
- EXT-02: no test file at all in `saga-orchestrator/area_info`, `saga-orchestrator/quest` (top level), `saga-orchestrator/field`, `saga-orchestrator/npc_spawn`, `saga-orchestrator/reactor` — five new/extended external-client surfaces with zero coverage.
- Handler struct-copy bug (no rule ID): `saga/handler.go:887-916` — `WithNpcSpawnProcessor`/`WithDropsProcessor`/`WithFieldProcessor` drop every Handler field except the ones named, contradicting the diff's own `WithAreaInfoProcessor` fix for the identical defect class two methods above. See dedicated section.
- Stale exhaustiveness test (no rule ID): `saga/event_acceptance_test.go:14-64` — `allActions` missing 9 of 10 new `Action` constants, silently weakening `TestAcceptanceTable_EveryActionRepresented`. See dedicated section.
- DOM-20: not table-driven — `area_info/processor_test.go` (character, 4/4), `medal_map/processor_test.go` (quest, 4/4), `drops/drops_test.go` (2/2), `saga/handler_test.go`'s 4 new funcs, `libs/atlas-saga/unmarshal_test.go` (52/53).

### Non-Blocking (should fix)

- Zero test coverage for 10 of 12 new saga handler methods (`handleWarpToMap`, `handleClearSkill`, `handleSpawnNpc`, `handleClearDrops`, `handleResetReactors`, `handleShuffleReactors`, `handleResetField`, `handlePlaySound`, `handleChangeMusic`, `handleBoatEffect`) — only `handleUpdateAreaInfo`/`handleExplorerQuest` are tested. Not independently rule-scored (no DOM-20/24 violation per se — these methods simply have no test to grade), but worth closing before this surface is considered done.
