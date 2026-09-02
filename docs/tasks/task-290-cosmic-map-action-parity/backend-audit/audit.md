# Backend Audit — task-290-cosmic-map-action-parity

- **Service Path(s):** services/atlas-map-actions, services/atlas-monsters, services/atlas-saga-orchestrator; libs/atlas-saga, libs/atlas-script-core; tools/catalog-lint, tools/gen-map-action-schema
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-09-02
- **Range:** `9613e7259..3f37fe6da`
- **Build:** PASS (module-local, all touched modules)
- **Tests:** PASS (module-local `go test ./... -count=1`, all touched modules)
- **Overall:** NEEDS-WORK

## Build & Test Results

```
services/atlas-map-actions/atlas.com/map-actions   : go build OK, go test OK (script pkg 0.059s)
services/atlas-monsters/atlas.com/monsters          : go build OK, go test OK (all subpackages ok)
services/atlas-saga-orchestrator/atlas.com/saga-orchestrator : go build OK, go test OK
libs/atlas-saga                                     : go build OK, go test OK
libs/atlas-script-core                              : go build OK, go test OK
tools/catalog-lint                                  : go build OK, go test OK (9.5s)
tools/gen-map-action-schema                         : go build OK, go test OK
```
No failures. `tools/verify.sh` (flagless) was explicitly excluded from this review per instructions (already in flight separately).

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| FILE (FILE-01..06) | Yes | Every changed Go package runs unconditionally. |
| DOM structure (DOM-01,02,03,04,05,11,16) | Yes | `script/` (map-actions) has `model.go`+`entity.go`+`rest.go`; `condition/` (libs) has `model.go`+`builder.go`. |
| SUB (SUB-01..04) | Yes | `world/` (atlas-monsters) has `resource.go`, no `model.go`. `monster/` (saga-orchestrator) has no `resource.go` → SUB N/A for that package. |
| REST (DOM-06..09,12..15,17..19,32) | Yes | `rest.go`/`processor.go`/`resource.go` changed in multiple packages. |
| Constants reuse (DOM-21) | No | Diff adds struct fields to existing types only; no new type/const-block/numeric-literal classification declared. |
| Testing (DOM-10,20,24,33) | Yes | Diff touches many `_test.go` files; `SpawnMonster` method re-signed on `saga-orchestrator/monster.Processor`. |
| Cache (DOM-29) | No | No `cache.go` touched; no new cached processor state. |
| Messaging (DOM-30) | No | No new `AndEmit`/`message.Emit`/`producer.ProviderImpl` call site added by the diff. |
| Multi-tenancy (DOM-31) | Yes | `rest.go` files changed in several packages; MapId/Instance/WorldId/ChannelId are domain routing fields, not tenant/trace identifiers — verified none of the new fields carry tenant/trace state. |
| Migration hygiene (DOM-34,35) | No | No symbols moved between a service and a `libs/atlas-*` module. |
| Deploy & topics (DOM-22,23) | No | No new `libs/atlas-*` module; no Kafka topic env var added/renamed. |
| Runtime safety (DOM-26) | Yes (family opens; rule N/A) | Non-test Go files changed, but no bare `go` statement added by the diff. |
| Channel wire values (DOM-25) | No | Diff does not touch `atlas-channel`/`atlas-packet`; no client-interpreted byte added. |
| Resilience (DOM-27,28) | No | No `model.Decorator`/enrichment fallback path changed. |
| External clients (EXT-01..04) | Yes | `saga-orchestrator/monster` package (processor.go, rest.go changed) calls atlas-monsters via `requests.PostRequest[T]`. |
| Scaffolding (SCAFFOLD-01..09) | No | No new `services/atlas-<svc>/` directory, no new channel Writer/Handler, `deploy/shared/routes.conf` untouched. |
| Security (SEC-01..04) | No | None of the touched services handle auth tokens, revocation, redirects, or secrets. |

## Checklist Results

### script (atlas-map-actions) — domain package

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go / validating Build() | N/A | `builder.go` untouched by this diff; not evaluated (out of surface). |
| DOM-02/03 | `entity.go` ToEntity/Make | PASS | `services/atlas-map-actions/atlas.com/map-actions/script/entity.go:113` `convertJsonCondition` builds via the condition `Builder`; `entity.go:179` `convertRuleToJson` mirrors the reverse conversion. Consistent with pre-existing shape, field additions only. |
| DOM-04/05 | rest.go Transform/TransformSlice | PASS | `script/rest.go:66` `transformRule`; no inline transform loops added in `resource.go` (resource.go untouched by this diff). |
| DOM-09 | Transform error-checked | N/A | `resource.go` (which calls `Transform`) is not in the diff. |
| DOM-19 | Request models flat | PASS | `script/rest.go:32-42` `RestRuleModel` — all new fields (`Values`, `Step`, `WorldId`, `ChannelId`, `IncludeEquipped`) are scalar/slice, no nested Data/Attributes. |
| DOM-30 | AndEmit + message.Buffer for writes | N/A | No new Kafka emit path added; `executor.go` calls `e.sagaP.Create(s)`, an existing saga-creation call, unchanged pattern. |
| DOM-20 | Table-driven tests | **FAIL** | `script/executor_test.go`: 4 of 6 new/changed `Test*` funcs are standalone, not table-driven — `TestExecuteOperationUnknownTypeErrors` (executor_test.go:39), `TestExecuteOperationsAbortsAfterUnknownOperation` (executor_test.go:61), `TestExecuteSpawnMonsterCarriesFieldInstance` (executor_test.go:95), `TestExecuteSpawnMonsterMonsterIdsPicksFromSet` (executor_test.go:187). Only `TestExecuteSpawnMonsterSpawnIfAbsent` (executor_test.go:140) and `TestExecuteSpawnMonsterIdParamValidation` (executor_test.go:229) use `tests := []struct{...}` + `t.Run`. `script/evaluator_test.go`: 4 of 5 new funcs standalone — `TestEvaluateViaQueryAggregatorCarriesEveryField` (evaluator_test.go:80), `TestEvaluateViaQueryAggregatorInOperatorUsesValues` (evaluator_test.go:127), `TestEvaluateViaQueryAggregatorRejectsNonIntegerScalarValue` (evaluator_test.go:164), `TestEvaluateViaQueryAggregatorRejectsNonIntegerValuesEntry` (evaluator_test.go:189); only `TestEvaluateMapIdOperators` (evaluator_test.go:25) is table-driven. `script/rest_test.go`: all 3 new funcs standalone — `TestExtractConditionCarriesEveryField` (rest_test.go:27), `TestTransformRuleRoundTripsEveryConditionField` (rest_test.go:67), `TestConvertJsonConditionCarriesEveryField` (rest_test.go:100); zero use `tests := []struct{...}` + `t.Run`. |
| DOM-24 | Producer stub for emit-reaching tests | PASS | `executor_test.go:20-35` injects a recording `mapactionsaga.Processor` fake into `e.sagaP`, so `ExecuteOperation`'s `e.sagaP.Create(s)` never reaches the real Kafka producer — no `AndEmit`/`message.Emit` call is exercised. |

### validation (atlas-map-actions) — support package

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-05 | model.go holds domain Model | PASS | `services/atlas-map-actions/atlas.com/map-actions/validation/model.go:10-16` — `ConditionInput` gains `Values []int` only; no misplaced responsibility introduced. |
| DOM-21 | Constants reuse | N/A | `Values []int` is a plain field addition, not a new type/const/numeric classification. |

### world (atlas-monsters) — sub-domain package (`resource.go`, no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Business logic outside handler | PASS | `world/resource.go:199-204` only interprets the processor's return signal (`m.UniqueId() == 0`) to pick an HTTP status; the actual suppression decision is made in `monster/processor.go:259-278` (`Create`). |
| SUB-02 | Writes via administrator, no db.Create in resource.go | PASS | `world/resource.go:194` calls `p.Create(f, input)` (processor), no direct DB/registry write in the handler. |
| SUB-03 | POST uses RegisterInputHandler[T] | PASS | `world/resource.go:38` `rest.RegisterInputHandler[monster.RestModel](l)(si)(createMonsterInMap, handleCreateMonsterInMap)`. |
| SUB-04 | No manual JSON parsing | PASS | No `json.NewDecoder`/`json.Unmarshal`/`io.ReadAll` in `world/resource.go` (grep confirmed absent). |
| DOM-17 | Domain errors mapped to specific HTTP status | WARN | `world/resource.go:196` maps every `p.Create` error to `http.StatusBadRequest` regardless of whether the underlying failure is a transient registry/monster-information lookup error vs. a genuine validation failure — pre-existing pattern, not newly introduced by this diff (the new code only adds the `StatusNoContent` branch above it), so not graded FAIL against this diff, flagged as pre-existing context for the reviewer. |
| DOM-20 | Table-driven tests | **FAIL** | `world/resource_create_test.go:29` `TestCreateMonsterInMapSuppressedSpawnReturns204` is a single standalone func, not `tests := []struct{...}` + `t.Run`. |
| DOM-24 | Producer stub for emit-reaching tests | PASS | `world/resource_create_test.go:38` seeds the field by calling `monster.GetMonsterRegistry().CreateMonster(...)` directly — that registry method (`monster/registry.go:396-404`) performs only in-memory `Put`/`Add` calls, no Kafka emit. The suppressed-spawn code path under test returns before reaching `GetMonsterRegistry().CreateMonster`/any emit in `monster/processor.go:259-286`, so the test never reaches an emit path. |

### monster (atlas-monsters) — domain package (has `model.go`, changed: `processor.go`, `processor_test.go`, `rest.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | N/A | Constructor (`NewProcessor`) not touched by this diff. |
| DOM-19 | Request models flat | PASS | `monster/rest.go:38` adds `SpawnIfAbsent bool` as a flat scalar to the existing `RestModel`. |
| DOM-20 | Table-driven tests | **FAIL** | `monster/processor_test.go`: all 5 new funcs are standalone `func Test...(t *testing.T)`, none use `tests := []struct{...}` + `t.Run` — `TestCreateSpawnIfAbsentSuppressesWhenPresent` (processor_test.go:2676), `TestCreateSpawnIfAbsentCreatesWhenAbsent` (processor_test.go:2723), `TestCreateSpawnIfAbsentMatchesOnTemplateNotUniqueId` (processor_test.go:2759), `TestCreateWithoutSpawnIfAbsentStacks` (processor_test.go:2799), `TestCreateSpawnIfAbsentIsFieldScoped` (processor_test.go:2834). |
| DOM-24 | Producer stub for emit-reaching tests | PASS | New tests construct `&ProcessorImpl{..., emit: func(topic.Token, model.Provider[[]kafka.Message]) error { return nil }}` (e.g. `processor_test.go:2683`) — the pre-existing, package-wide per-test no-op emit injection pattern (used identically 22 times across this file), functionally equivalent to the "per-test injection of a no-op producer" carve-out. |
| — | **CARRIED-FORWARD ITEM** — mapId override + field instance mismatch | **FAIL (see dedicated section below)** | `services/atlas-map-actions/atlas.com/map-actions/script/executor.go:196-234`, corroborated by `services/atlas-monsters/atlas.com/monsters/monster/processor.go:259-278` and `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go:2086-2089`. |

### monster (saga-orchestrator) — support package (processor.go, rest.go changed; no resource.go, no model.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor/ProcessorImpl in processor.go | PASS | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/monster/processor.go:11-38` — `Processor` interface, `ProcessorImpl`, constructor, and method all in `processor.go`. |
| FILE-02 | RestModel/Transform in rest.go | PASS | `monster/rest.go:12-83` — `SpawnInputRestModel`, `SpawnResponseRestModel`, `SpawnRequest.ToRestModel()` all in `rest.go`. |
| FILE-03 | Cross-service calls in requests.go | PASS | `monster/requests.go` (unchanged by diff, confirmed present) holds `requestSpawnMonster`. |
| DOM-33 | Interface change updates every mock | PASS | `Processor.SpawnMonster` re-signed from `(f, monsterId, x, y, fh, team)` to `(f, req SpawnRequest)` (`processor.go:13`). Repo-wide grep for `monster.Processor` found only one non-test reference — `saga/handler.go` (updated in the same diff, `handler.go:2092-2104`). No mock implementation of this interface exists anywhere in the repo (`saga/mock/processor.go` does not reference it); nothing to update. |
| EXT-01 | Target REST model implements SetToOneReferenceID/SetToManyReferenceIDs | **FAIL** | `monster/rest.go:29-58` `SpawnResponseRestModel` has no `SetToOneReferenceID`/`SetToManyReferenceIDs` methods (grep across the package found none). Pre-existing gap in a file this diff modifies (adds `SpawnIfAbsent` to the sibling input model in the same file) — in scope because `rest.go` is a changed file. |
| EXT-02 | httptest-backed integration test with representative fixture | **FAIL** | No `_test.go` file exists anywhere under `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/monster/` (directory listing confirmed). No integration test covers `requestSpawnMonster`/`SpawnMonster`, including the new `SpawnIfAbsent` field this diff adds to the wire payload. |
| EXT-03 | Only genuine 404 → not-found; other failures bubble | PASS | `monster/processor.go:31-34` — `err` from `requestSpawnMonster` is logged and returned verbatim, no status-code-based reclassification that could conflate a 5xx/transport failure with "not found". |
| EXT-04 | URL via requests.RootUrl, not hardcoded DNS | PASS | `monster/requests.go:16` `requests.RootUrlFor(ctx, "MONSTERS")` (file unchanged by this diff but confirmed as the URL-composition path the changed `processor.go` calls into). |
| DOM-20 | Table-driven tests | N/A | No test file exists in this package to grade (see EXT-02 finding above — the gap is the absence of tests, not their shape). |

### saga (saga-orchestrator) — domain-adjacent support file (`handler.go` changed only)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-31 | Tenant/trace only in context | PASS | `saga/handler.go:2086-2089` builds `field.NewBuilder(payload.WorldId, payload.ChannelId, payload.MapId).SetInstance(payload.Instance).Build()` — WorldId/ChannelId/MapId/Instance are domain routing fields carried on the saga payload, not tenant or trace identifiers; no tenant/trace value is read from a REST model or request body here. |
| DOM-13 | No cross-domain orchestration in handlers | N/A | `saga/handler.go` is saga-step orchestration by design, not a REST `resource.go` handler — DOM-13's trigger (`package has resource.go`) does not fire for this file. |

### condition (libs/atlas-script-core) — domain package (model.go + builder.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go / validating Build() | PASS | `libs/atlas-script-core/condition/builder.go:90-92` — `Build()` still validates `conditionType`/`operator`, and the updated guard `if b.value == "" && len(b.values) == 0` (builder.go:91) correctly requires one of `value`/`values`. |
| DOM-19 | (n/a family, no rest.go) | N/A | Package has no `rest.go`. |
| DOM-20 | Table-driven tests | **FAIL** | `libs/atlas-script-core/condition/model_test.go`: all 7 new funcs standalone, none use `tests := []struct{...}` + `t.Run` — `TestBuilderSetValues` (model_test.go:8), `TestBuilderAddValue` (model_test.go:21), `TestBuilderValuesOmittedDefaultsNil` (model_test.go:31), `TestBuildRequiresValueOrValues` (model_test.go:44), `TestBuildAcceptsValuesWithoutValue` (model_test.go:54), `TestBuildStillRequiresType` (model_test.go:61), `TestBuildStillRequiresOperator` (model_test.go:71). |

### atlas-saga (libs) — support (payloads.go changed)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | Constants reuse | N/A | `SpawnMonsterPayload.SpawnIfAbsent bool` (`libs/atlas-saga/payloads.go:520`) is a field addition, not a new type/const. |
| DOM-20 | Table-driven tests | **FAIL** | `libs/atlas-saga/payloads_test.go`: all 5 new funcs standalone — `TestDestroyAssetFromSlotPayloadTemplateIdRoundTrip` (payloads_test.go:12), `TestSkillBookUseSagaType` (payloads_test.go:41), `TestCharacterCreatePayloadCarriesApAndSp` (payloads_test.go:50), `TestSpawnMonsterPayloadRoundTripsSpawnIfAbsent` (payloads_test.go:92), `TestSpawnMonsterPayloadOmitsSpawnIfAbsentWhenFalse` (payloads_test.go:123). None use `tests := []struct{...}` + `t.Run`. |
| DOM-34/35 | Migration hygiene | N/A | No symbol moved between a service and a `libs/atlas-*` module — this is an additive field on an existing shared payload type. |

### tools/catalog-lint, tools/gen-map-action-schema — CLI tooling (not a service; no Processor/RestModel/Entity constructs exist)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File responsibilities | N/A | Neither package has a `resource.go`, `rest.go`, `entity.go`, `builder.go`+domain `model.go`, or `administrator.go` — these are standalone CLI generators/linters with no REST/domain surface for FILE-* to grade. |
| DOM-20 | Table-driven tests | **FAIL** | `tools/catalog-lint/mapactions_test.go`: 12 `func Test...` , 0 use `tests := []struct{...}` + `t.Run`. `tools/gen-map-action-schema/main_test.go`: 5 `func Test...`, 0 table-driven. |

## Carried-forward item — mapId override paired with field instance (executor.go:196-220)

**Assessment: this hazard is introduced by this branch, not pre-existing as framed in the review prompt.**

Verified by diff against merge-base `9613e7259`:
- Merge-base `executor.go` (`git show 9613e7259:.../script/executor.go`, old line ~183) hard-coded `Instance: uuid.Nil` on every `SpawnMonsterPayload`, **regardless of** the `mapId` override at old lines 155-163. Under that code, the `(mapId, instance)` pair sent downstream was always `(overrideOrDefaultMapId, uuid.Nil)` — internally consistent, because `uuid.Nil` is a valid "no instance" value for any map.
- This branch (`services/atlas-map-actions/atlas.com/map-actions/script/executor.go:229`) changes `Instance: uuid.Nil` to `Instance: f.Instance()` — the caller's *own* current field instance UUID — while leaving the `mapId` override at `executor.go:206-213` untouched and independent. This is Design F3's intended fix for the common case (spawn stays on the caller's own map), and it is correct for every currently-seeded script (none of which sets `mapId`, confirmed by `grep -rl '"mapId"' deploy/seed/**/map-actions` returning zero seed-document matches — only `docs/map_script_schema.json` documents the parameter).
- The two changes compose incorrectly: when a future document sets `mapId` to a map other than the caller's current map, the resulting `SpawnMonsterPayload` carries `(overrideMapId, callerInstanceUUID)` — an instance UUID that has no necessary relationship to `overrideMapId`. `saga/handler.go:2086-2089` builds the field straight from these two payload fields (`field.NewBuilder(payload.WorldId, payload.ChannelId, payload.MapId).SetInstance(payload.Instance).Build()`), and `monster/processor.go:264` (`GetInField(f)`) and `world/resource.go`'s registry scope the `spawnIfAbsent` idempotency check to that composite, possibly-mismatched field identity — silently defeating the guard's purpose (design D5's own stated intent: "two instances of the same map each get their own monster") for the override case, and potentially orphaning the spawned monster under a field identity no normal entrant to `overrideMapId` will ever query.
- No test in this diff exercises `mapId` override together with a non-nil instance: `executor_test.go`'s `TestExecuteSpawnMonsterCarriesFieldInstance` (executor_test.go:95) uses an explicit instance but does not set `mapId`; no test sets both.
- `tools/catalog-lint`'s `checkMapActionSpawnGuards` (`tools/catalog-lint/mapactions.go:214-236`) only checks for the presence of `"spawnIfAbsent": "true"`; it has no (and structurally cannot have, at lint time) awareness of runtime field-instance identity, so it cannot catch this.

**Severity: Important.** It is dormant today (no seed script uses `mapId`), so it does not affect current production behavior, but it ships as a live, undocumented defect in a feature this same PR promotes to a first-class, catalog-lint-enforced schema parameter (`mapId`) combined with a newly-real per-instance idempotency guard (`spawnIfAbsent`). The moment a future map-action targets a different (especially instanced) map via `mapId`, the spawn silently associates with the wrong field identity with no error, no test, and no lint signal.

**Recommendation:** does not need to block this PR's merge on its own (it changes no currently-observable behavior — no seed document uses `mapId`), but it should not be left as an implicit, un-filed gap either. File it as an explicit, named follow-up task before this branch is considered fully closed — the fix requires a design decision (e.g., reject `mapId` override outright when `f.Instance() != uuid.Nil`, fail loud in `executeSpawnMonster`; or thread a real target-instance resolution through the saga) rather than a trivial code change, which is why it is a follow-up and not a same-PR fix.

## Security Review

SEC-* family did not fire — none of the three changed services (`atlas-map-actions`, `atlas-monsters`, `atlas-saga-orchestrator`) handle authentication, tokens, revocation, redirects, or secrets in the changed code. No security-relevant code paths in this diff.

## Not evaluable from the diff

- EXT-02 gap disposition (missing `saga-orchestrator/monster` package tests) is settled as a FAIL from the diff itself (no test file exists) — no further reading was needed, so this is not listed as "not evaluable."
- Whether other in-repo `SpawnMonsterPayload` producers (`services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go`, `services/atlas-reactor-actions/atlas.com/reactor/script/executor.go`) have their own `mapId`-override-equivalent hazards was not investigated — those files are unchanged by this diff and outside the stated scope (changed services: atlas-map-actions, atlas-monsters, atlas-saga-orchestrator). Named here because they share the payload type this diff extends; a full sweep of every `SpawnMonsterPayload` producer for the same class of defect was not performed.
- DOM-17 status-mapping correctness in `world/resource.go`'s pre-existing `StatusBadRequest`-for-all-`p.Create`-errors branch was noted (WARN, not FAIL) but not fully traced against every possible `p.Create` error path (registry failures, `monsterInformation` lookup failures) to determine which, if any, are actually transient/5xx-worthy — that would require reading `monster/processor.go`'s full error surface beyond what this diff touches.

## Summary

### Blocking (must fix)

- DOM-20: `services/atlas-map-actions/atlas.com/map-actions/script/executor_test.go` — 4 of 6 new tests not table-driven (executor_test.go:39,61,95,187).
- DOM-20: `services/atlas-map-actions/atlas.com/map-actions/script/evaluator_test.go` — 4 of 5 new tests not table-driven (evaluator_test.go:80,127,164,189).
- DOM-20: `services/atlas-map-actions/atlas.com/map-actions/script/rest_test.go` — 3 of 3 new tests not table-driven (rest_test.go:27,67,100).
- DOM-20: `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go` — 5 of 5 new tests not table-driven (processor_test.go:2676,2723,2759,2799,2834).
- DOM-20: `services/atlas-monsters/atlas.com/monsters/world/resource_create_test.go` — 1 of 1 new test not table-driven (resource_create_test.go:29).
- DOM-20: `libs/atlas-saga/payloads_test.go` — 5 of 5 new tests not table-driven (payloads_test.go:12,41,50,92,123).
- DOM-20: `libs/atlas-script-core/condition/model_test.go` — 7 of 7 new tests not table-driven (model_test.go:8,21,31,44,54,61,71).
- DOM-20: `tools/catalog-lint/mapactions_test.go` (12/12) and `tools/gen-map-action-schema/main_test.go` (5/5) not table-driven.
- EXT-01: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/monster/rest.go:29-58` — `SpawnResponseRestModel` has no `SetToOneReferenceID`/`SetToManyReferenceIDs`.
- EXT-02: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/monster/` — no test file exists at all; no httptest integration coverage of `SpawnMonster`/`requestSpawnMonster`, including the new `SpawnIfAbsent` wire field.
- Carried-forward: `services/atlas-map-actions/atlas.com/map-actions/script/executor.go:196-234` — `mapId` override combined with `f.Instance()` produces a mismatched field identity for spawn/idempotency purposes. Newly introduced by this branch's Instance fix (previously inert under hard-coded `uuid.Nil`); dormant today, no test coverage of the combination. Recommend filing as an explicit follow-up task rather than blocking this PR outright, given no current behavior regression — see dedicated section above for full reasoning.

### Non-Blocking (should fix)

- DOM-17 (WARN): `services/atlas-monsters/atlas.com/monsters/world/resource.go:196` maps every `p.Create` error to 400 regardless of cause; pre-existing, not introduced by this diff.
