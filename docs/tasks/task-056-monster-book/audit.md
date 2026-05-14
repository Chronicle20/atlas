# Plan Audit — task-056-monster-book

**Plan Path:** docs/tasks/task-056-monster-book/plan.md
**Audit Date:** 2026-05-05
**Branch:** task-056-monster-book
**Base Branch:** main (range `9a28353c6..HEAD`, 36 commits)

## Executive Summary

The Monster Book implementation is end-to-end and broadly faithful to the plan. All 40 plan
tasks have artifacts on disk; every affected Go service builds and every Go test suite
passes (including new tests for `card.upsertCard`, `collection.computeBookLevel`,
`monsterbook` consumer, query-aggregator's `MonsterBookCountCondition`, and atlas-quest's
`buildStartConditions`). The atlas-ui widget tests pass (692 tests). Two material divergences
warrant attention: (1) the atlas-channel outbound consumer (Task 30) was implemented but the
plan-required unit tests for the three-packet/one-packet fan-out were not written; (2)
atlas-quest only emits the `monsterBookCount` condition from `buildStartConditions`, not
from `ValidateEndRequirements` as the plan called out — verify this is intentional or add
the branch.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Service skeleton | DONE | services/atlas-monster-book/atlas.com/monster-book/{go.mod, main.go, logger/logger.go} (commit 1136e2a42) |
| 2 | collection entity + migration | DONE | collection/entity.go, collection/entity_test.go (commit 3d5d0e773) |
| 3 | collection model + builder | DONE | collection/{model.go, builder.go, builder_test.go} (c3a7fe224) |
| 4 | collection administrator + provider | DONE | collection/{administrator.go, administrator_test.go, provider.go} (a058e8160) |
| 5 | card entity + migration | DONE | card/{entity.go, entity_test.go} (b8e62d207) |
| 6 | card model + builder | DONE | card/{model.go, builder.go, builder_test.go} (70ef0ffcf) |
| 7 | card administrator (idempotent upsert) | DONE | card/administrator.go:19 `upsertCard` with `LastEventId` idempotency guard (lines 47-49); card/administrator_test.go (376096bff) |
| 8 | Kafka producer + buffer scaffold | DONE | kafka/{producer/producer.go, message/message.go, consumer/consumer.go} (afa8cbccd) |
| 9 | Kafka message types | DONE | kafka/message/{character/kafka.go, monsterbook/kafka.go} (0a7496a93). Topic constants `EnvCommandTopic`, `EnvEventTopicStatus`; envelopes `Command[B]` / `StatusEvent[B]` |
| 10 | card.Processor `AddAndEmit` | DONE | card/processor.go:63-104. `Add(mb)` curried buffer signature; `AddAndEmit` wraps `message.Emit(producer.ProviderImpl(...))`. Buffers `CARD_ADDED` only; emits via tenant-keyed message provider. card/processor_test.go covers Add/AddAndEmit/Duplicate paths |
| 11 | collection.Processor (recompute, EXP, cover) | DONE | collection/processor.go:128-203 RecomputeAndEmit, :205-243 SetCoverAndEmit. computeBookLevel matches Cosmic formula (level*10 increments). EXPERIENCE_CHANGED envelope shape (lines 32-48 + 188-200) populates `CharacterId`, `Type`, `Body.Distributions[].ExperienceType=MONSTER_BOOK`, `Amount=expBonus` — a strict subset of atlas-channel's `character.StatusEvent[ExperienceChangedStatusEventBody]` (channel/kafka/message/character/kafka.go:99-127). Compatible because unknown JSON fields are zero-valued on the consumer side. collection/processor_test.go covers formula + cover validation |
| 12 | Inbound consumer CARD_PICKED_UP | DONE | kafka/consumer/monsterbook/consumer.go:47-73. Wraps tx + `message.Emit` correctly: `cp.Add(mb) -> if Inserted -> colp.RecomputeAndEmit(mb)`. SET_COVER handler at :75-85. consumer_test.go validates idempotency, transaction rollback |
| 13 | Character lifecycle consumer (cascade delete) | DONE | kafka/consumer/character/{consumer.go, consumer_test.go} (commit 8068df5b7) |
| 14 | REST handlers | DONE | character/resource.go:32-35 registers GET /monster-book, PATCH /monster-book, GET /monster-book/cards (paginated), GET /monster-book/cards/{cardId}; rest/handler.go added ParseCardId; collection/rest.go and card/rest.go provide JSON:API models (f797761af) |
| 15 | Wire main.go | DONE | services/atlas-monster-book/atlas.com/monster-book/main.go (47dc7d120) |
| 16 | Dockerfile | DONE | services/atlas-monster-book/Dockerfile (4307fa55b) |
| 17 | docker-compose entry | DONE | deploy/compose/docker-compose.core.yml:340-349 (3fdb95832) |
| 18 | ITEM.CONSUMED_ON_PICKUP message + producer | DONE | services/atlas-inventory/atlas.com/inventory/kafka/message/pickup/kafka.go (Command, NewCommandProvider, EnvCommandTopic = COMMAND_TOPIC_ITEM_CONSUMED_ON_PICKUP) (ab36daf38) |
| 19 | Branch in AttemptItemPickUp | DONE | services/atlas-inventory/atlas.com/inventory/compartment/processor.go:1149-1167. Branch checks `inventoryType == TypeValueUse && cm.ConsumeOnPickup()`, emits `pickupMsg.NewCommandProvider`, then `dropProcessor.RequestPickUp(mb)` — bypasses inventory mutation. Test added in compartment/processor_test.go (736a49c1b) |
| 20 | Verify atlas-inventory build + tests | DONE | `go build ./...` + `go test ./...` PASS (this audit) |
| 21 | atlas-consumables ITEM.CONSUMED_ON_PICKUP consumer | PARTIAL | services/atlas-consumables/atlas.com/consumables/kafka/consumer/pickup/consumer.go implements Init/Handler + `cardItemPrefix` (238) gating (commit ab89af94e). However plan Step 3 required `consumer_test.go`; **no test file exists** for the new consumer. Plan Step 5 ("Run tests") cannot be satisfied |
| 22 | Wire atlas-consumables consumer | DONE | atlas-consumables main.go registers pickup.InitConsumers + InitHandlers (b4ed8dbc3) |
| 23 | atlas-consumables build + Docker | DONE | `go build ./...` + `go test ./...` PASS |
| 24 | MonsterBookSetCard writer (0x53) | DONE | libs/atlas-packet/character/clientbound/monsterbook/set_card.go + set_card_test.go (dce73f2e4) |
| 25 | MonsterBookSetCover writer (0x54) | DONE | libs/atlas-packet/character/clientbound/monsterbook/set_cover.go + set_cover_test.go (639f2bfed) |
| 26 | Recv handler MonsterBookCover (0x39) | DONE | libs/atlas-packet/character/serverbound/monsterbook/{cover.go, cover_test.go}; services/atlas-channel/atlas.com/channel/socket/handler/monster_book_cover.go; channel monsterbook/{processor.go, producer.go}; channel kafka/message/monsterbook/kafka.go (6f431d418) |
| 27 | Register handler + writers in atlas-channel main.go | DONE | services/atlas-channel/atlas.com/channel/main.go:507-508 writers, :584 handler (db183aa52) |
| 28 | Decoder/handler unit tests | DONE | services/atlas-channel/atlas.com/channel/socket/handler/monster_book_cover_test.go (379b0129b) |
| 29 | REST client to atlas-monster-book | DONE | services/atlas-channel/atlas.com/channel/monsterbook/{rest.go, requests.go} + Get extension on processor.go:65-67 (57844d91c) |
| 30 | Outbound consumer MONSTER_BOOK status | PARTIAL | services/atlas-channel/atlas.com/channel/kafka/consumer/monsterbook/consumer.go (a16b4622f). handleCardAdded correctly fans out: SetCard (always), CardGetEffect self + Foreign (only when `!Body.Full`). handleCoverChanged sends SetCover. **Plan Step 2 required tests** for "three-packet fan-out (added) and one-packet fan-out (full)" — no consumer_test.go exists in this package. Step 4 ("Run tests PASS") therefore not actually executed |
| 31 | Wire outbound consumer in main.go | DONE | atlas-channel main.go:34 import, :177 InitConsumers, :281 InitHandlers (16e2455f8) |
| 32 | MonsterBookCoverDecorator on character info | DONE | character/model.go:61 coverCardId field + :347-357 getter/setter; character/builder.go:59,148,193; character/processor.go:33 interface, :172-181 implementation (graceful degradation on REST errors); socket/handler/character_info_request.go:31 appends decorator. Step 5 (cover-in-info-packet decision): cover is NOT serialized in the info packet — `CoverCardId` is read but no clientbound packet field embeds it; the cover is only reflected via the SetCover packet pushed when COVER_CHANGED fires (e52015d09) |
| 33 | atlas-saga MonsterBookCountCondition constant | DONE | libs/atlas-saga/validation.go:45 (a961b8803) |
| 34 | atlas-query-aggregator monsterBookCount evaluator | DONE | validation/model.go:56 constant, :125 SetType allow-list, :561 Evaluate (no-context fallback), :848-852 EvaluateWithContext using `ctx.GetMonsterBookTotalUniqueCards()`; validation/context.go:84,471 `monsterbook.NewProcessor` injection, :205-235 GetMonsterBookTotalUniqueCards/MonsterBookProcessor/With…, :519 SetMonsterBookProcessor builder; new package monsterbook/{processor.go, rest.go}; validation/rest.go:300 accept-list; validation/model_test.go:2016-2148 7 test cases (208d3cf41) |
| 35 | atlas-quest emits the condition | PARTIAL | data/quest/rest.go:74-77 adds `MonsterBookCountMin uint32` field on local RequirementsRestModel (intentional — atlas-data has not yet been updated to read `mbmin` from WZ); data/validation/model.go:24 `MonsterBookCountCondition` constant; data/validation/processor.go:97-106 emits condition from `buildStartConditions` when `MonsterBookCountMin > 0`. **Plan Step 2 explicitly required the branch in BOTH `buildStartConditions` AND `ValidateEndRequirements`**; reviewing processor.go:219-271 shows `ValidateEndRequirements` only handles Item + MesoMin and never references MonsterBookCountMin. Two interpretations: (a) intentional because end-requirements are a stricter subset (only items/meso) and the start-requirement branch suffices, (b) gap that lets a quest define `mbmin` as a turn-in gate that won't be enforced. Tests at processor_test.go:118-148 only cover the start path |
| 36 | E2E build verification | DONE | All six Go services + atlas-ui pass build/test in this audit (see Build & Test Results) |
| 37 | UI API client service | DONE | services/atlas-ui/src/services/api/monster-book.service.ts + types/monster-book.ts + services/api/index.ts export (5ccc27c17) |
| 38 | UI widget component | DONE | components/features/characters/MonsterBookWidget.tsx + __tests__/MonsterBookWidget.test.tsx (e047a9582) |
| 39 | Mount widget on character detail page | DONE | pages/CharacterDetailPage.tsx:19 import, :192 `<MonsterBookWidget characterId={...} />` (bdb001a91) |
| 40 | Final verification + audit handoff | DONE | This audit covers Steps 1-2; Steps 3-4-7 are the user-facing handoff phase |

**Completion Rate:** 37/40 DONE, 3/40 PARTIAL (92.5% strictly DONE; 100% have implementation artifacts on disk)
**Skipped without approval:** 0
**Partial implementations:** 3 (Tasks 21, 30, 35)

## Skipped / Deferred Tasks

### Task 21 — atlas-consumables pickup consumer test (PARTIAL)

Plan Step 3 specifies "Write consumer test" with a concrete table-driven structure
(card path emits CARD_PICKED_UP; non-card path no-ops; type mismatch no-ops). No
`consumer_test.go` exists under `services/atlas-consumables/atlas.com/consumables/kafka/consumer/pickup/`.
The consumer logic is small (~33 lines) but uncovered. Impact: the `cardItemPrefix`
classification (`itemId/10000 == 238`) is not asserted, making accidental regressions
possible.

### Task 30 — atlas-channel MONSTER_BOOK.CARD_ADDED outbound consumer test (PARTIAL)

Plan Step 2 spelled out two specific test cases:
- "three-packet fan-out (added)" — SetCard + CharacterEffect self + CharacterEffectForeign
- "one-packet fan-out (full)" — SetCard only, no effects

The implementation in `services/atlas-channel/atlas.com/channel/kafka/consumer/monsterbook/consumer.go`
correctly distinguishes these via `if !e.Body.Full`. However, no `consumer_test.go` was
written in this package. The fan-out logic (especially the foreign-effect call
through `_map.NewProcessor(...).ForOtherSessionsInMap`) is not asserted by tests.
Impact: regressions in the Full check or the foreign-broadcast path won't be caught.

### Task 35 — atlas-quest MonsterBookCountMin in ValidateEndRequirements (PARTIAL)

Plan Step 2 explicitly required adding the `monsterBookCount` builder branch in
**both** `buildStartConditions` and `ValidateEndRequirements`. Only the former is
implemented (`processor.go:97-106`); `ValidateEndRequirements` (lines 219-271)
remains unchanged and never references `MonsterBookCountMin`. If a quest declares
`mbmin` as an end-requirement (e.g., turn-in gate), it will silently pass.

A separate concern noted by the user: data flows through a new local `MonsterBookCountMin`
field on `RequirementsRestModel`, but atlas-data has **not** been updated to populate
it from the WZ `mbmin` element. The constant exists end-to-end and the validation
chain works once data is wired, but until atlas-data extracts `mbmin`, no quest
will actually carry a non-zero `MonsterBookCountMin` to atlas-quest. This is
explicitly flagged in plan Task 35 ("note: data flows through a new
MonsterBookCountMin field…"); it is best documented as a follow-up rather than a
plan deviation, but tracking it here for visibility.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-monster-book | PASS | PASS | all packages compile; card/collection/consumer tests green |
| atlas-inventory | PASS | PASS | including new consume-on-pickup branch test in compartment/processor_test.go |
| atlas-consumables | PASS | PASS | builds, but pickup consumer has no tests (Task 21 gap) |
| atlas-channel | PASS | PASS | builds, but monster-book outbound consumer has no tests (Task 30 gap); recv handler test exists |
| atlas-quest | PASS | PASS | new TestBuildStartConditions_MonsterBookCountMin_* tests pass |
| atlas-query-aggregator | PASS | PASS | new TestCondition_MonsterBookCount and accept-list tests pass |
| atlas-ui | PASS (tsc -b) | PASS (vitest: 73 files, 692 tests) | npm run build wrapper failed in this audit's WSL shell context (CMD.EXE/UNC paths); tsc and vitest run cleanly via direct binary invocation with node v24.12.0 on PATH |

## Overall Assessment

- **Plan Adherence:** MOSTLY_COMPLETE
- **Recommendation:** NEEDS_REVIEW — three test gaps and one possible end-requirement
  branch divergence to confirm. None block merge functionally; all are test-coverage
  or scope-confirmation issues.

## Action Items

1. Add `consumer_test.go` for `services/atlas-consumables/atlas.com/consumables/kafka/consumer/pickup/`
   covering: card path emits `CARD_PICKED_UP` to `COMMAND_TOPIC_MONSTER_BOOK`; non-card item
   logs and no-ops; mismatched `Type` field no-ops; `cardItemPrefix` boundary (2380000 vs
   2390000).
2. Add `consumer_test.go` for `services/atlas-channel/atlas.com/channel/kafka/consumer/monsterbook/`
   asserting: `handleCardAdded` with `Full=false` invokes the SetCard writer + self
   CharacterEffect + foreign CharacterEffect-for-others; `handleCardAdded` with `Full=true`
   invokes only the SetCard writer; tenant-mismatch path no-ops; `handleCoverChanged`
   with mismatched `Type` no-ops.
3. Confirm Task 35 intent. Either (a) add the `MonsterBookCountMin` branch to
   `ValidateEndRequirements` and a corresponding test, or (b) document explicitly in
   plan/PRD that monster-book counts only gate quest-start (not turn-in) and remove the
   plan's "and `ValidateEndRequirements`" wording.
4. Track the atlas-data WZ `mbmin` extraction as a follow-up task (out of scope for this
   plan but required for the condition to actually fire on real quest definitions).

---

# Backend Audit — task-056-monster-book

- **Reviewer:** backend-guidelines-reviewer
- **Branch:** `task-056-monster-book`
- **Range:** `9a28353c6..HEAD`
- **Date:** 2026-05-05
- **Build:** PASS (atlas-monster-book, atlas-channel, atlas-consumables, atlas-inventory, atlas-query-aggregator, atlas-quest)
- **Tests:** all packages pass (`go test ./... -count=1` clean for every touched service)
- **Overall:** NEEDS-WORK

## Build & Test Results

- `services/atlas-monster-book/atlas.com/monster-book` — `go build ./...` clean; `go test ./... -count=1` PASS (card, collection, kafka/consumer/character, kafka/consumer/monsterbook all green; other packages have no test files but compile)
- `services/atlas-channel/atlas.com/channel` — build clean; tests pass (handler, monster_book_cover_test.go included)
- `services/atlas-consumables/atlas.com/consumables` — build clean; tests pass
- `services/atlas-inventory/atlas.com/inventory` — build clean; tests pass (compartment processor tests cover new consume-on-pickup branch)
- `services/atlas-query-aggregator/atlas.com/query-aggregator` — build clean; tests pass
- `services/atlas-quest/atlas.com/quest` — build clean; tests pass

## Domain Discovery

`atlas-monster-book` packages classified:

| Package | Type | Notes |
|---------|------|-------|
| `card` | domain (has `model.go`) | full DOM checklist applies |
| `collection` | domain (has `model.go`) | full DOM checklist applies |
| `character` | sub-domain (resource.go only) | SUB checklist applies (route registration) |
| `rest` | support | thin wrappers over `atlas-rest/server` |
| `kafka/consumer/{character,monsterbook}` | support (consumer wiring) | not domains |
| `kafka/message/{character,monsterbook}` | support (envelopes) | not domains |
| `kafka/producer` | support | not a domain |

External REST clients added: `services/atlas-channel/atlas.com/channel/monsterbook/` and `services/atlas-query-aggregator/atlas.com/query-aggregator/monsterbook/` — EXT checklist applies.

## Domain Checklist — `card` (atlas-monster-book)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists | PASS | `services/atlas-monster-book/atlas.com/monster-book/card/builder.go:11-71` (`NewModelBuilder`, fluent setters, `Build()` validates characterId/cardId/level) |
| DOM-02 | `ToEntity()` method | FAIL | No `ToEntity()` in `card/entity.go` (only `Make(entity)` and table name). Round-trip Model->entity is implemented inline in `card/administrator.go:30-37` and `:55-67` instead of via a `Model.ToEntity()` method. |
| DOM-03 | `Make(Entity)` function | PASS | `card/builder.go:61` (`func Make(e entity) (Model, error)`) |
| DOM-04 | `Transform` function | PASS | `card/rest.go:26` |
| DOM-05 | `TransformSlice` function | FAIL | No `TransformSlice` in `card/rest.go`. The list handler at `character/resource.go:114` uses `model.SliceMap(card.Transform)(...)` inline instead of a domain-defined `card.TransformSlice`. |
| DOM-06 | Processor accepts `FieldLogger` | PASS | `card/processor.go:39` (`l logrus.FieldLogger`); `collection/processor.go:66` likewise |
| DOM-07 | Handlers pass `d.Logger()` | PASS | `character/resource.go:44, 61, 82, 130` all call `card.NewProcessor(d.Logger(), ...)` / `collection.NewProcessor(d.Logger(), ...)`. No `logrus.StandardLogger()` references found. |
| DOM-08 | POST/PATCH use `RegisterInputHandler` | PASS | `character/resource.go:33` registers PATCH via `rest.RegisterInputHandler[collection.PatchInput]`. No POST routes are exposed (commands flow via Kafka). |
| DOM-09 | Transform errors handled | FAIL | `character/resource.go:50` `rm, _ := collection.Transform(m)`; line 71 same pattern; line 136 `rm, _ := card.Transform(m)`. Three Transform error returns silently dropped. |
| DOM-10 | Test DB has tenant callbacks | FAIL | `card/administrator_test.go:11`, `collection/administrator_test.go:11`, `card/processor_test.go` (uses `newDB`), `collection/processor_test.go` open SQLite with `Migration(db)` but never call `database.RegisterTenantCallbacks(l, db)`. The producer code relies on `tenant.MustFromContext(ctx)` and passes `tenantId` explicitly into every WHERE/INSERT, so isolation is enforced by hand-rolled clauses in `card/provider.go:11-25` and `card/administrator.go:26,33,55-66,73`. The tests pass but do not exercise the GORM callback path; if a future read switches to `database.Query` without an explicit tenant filter, no test will catch the leak. |
| DOM-11 | Providers use lazy evaluation | PASS | `card/provider.go:10-26` uses `database.Query`/`database.SliceQuery`; `collection/provider.go:10-14` likewise |
| DOM-12 | No `os.Getenv()` in handlers | PASS | `grep "os.Getenv" character/resource.go` -> no match |
| DOM-13 | No cross-domain logic in handlers | PASS | `character/resource.go` handlers call only `card.NewProcessor` / `collection.NewProcessor` and their own methods; no orchestration logic in handlers |
| DOM-14 | Handlers don't call providers directly | PASS | All reads go through `cp.GetByCharacterId`, `cp.GetByCharacterIdAndCardId`, `p.GetByCharacterId`. No direct `byCharacterIdEntityProvider` usage in `character/resource.go`. |
| DOM-15 | No direct entity creation in handlers | PASS | No `db.Create`/`db.Save`/`db.Delete` in `character/resource.go` |
| DOM-16 | `administrator.go` exists for writes | PASS | `card/administrator.go` and `collection/administrator.go` |
| DOM-17 | Domain error -> HTTP status mapping | FAIL | `character/resource.go:47, 66, 86, 116`: every error returns `http.StatusInternalServerError` regardless of cause. `handlePatch` line 63 uses `StatusUnprocessableEntity` for *any* `SetCoverAndEmit` error — the same envelope is returned for "cover requires owned card" (validation, should be 400/422), "cardId out of range" (400), and a transient DB failure (500). `handleGetCard` returns 404 for ANY error, including connection failures. No mapping of `gorm.ErrRecordNotFound` -> 404 in `handleGet` (the processor masks it for `GetByCharacterId` but not callers' errors), and no distinction between validation and infrastructure errors. |
| DOM-18 | JSON:API interface on REST models | PASS | `card/rest.go:15-24` and `collection/rest.go:15-24, 43-52` define `GetName()`, `GetID()`, `SetID()` on `RestModel` and `PatchInput` |
| DOM-19 | Request models use flat structure | PASS | `collection.PatchInput` (`collection/rest.go:38-52`) is flat; no nested Data/Type/Attributes |
| DOM-20 | Table-driven tests | WARN | Tests use direct `if/else` assertions and shared `cases := map[...]...` literals (`card/builder_test.go:32-48`, `collection/processor_test.go:31-42`) but do not consistently use `tests := []struct{...}` + `t.Run`. Functionality is covered; style is inconsistent with the guideline. |
| DOM-21 | No duplication of atlas-constants types | FAIL | Multiple violations:<br>- `card/model.go:9-22` redeclares card-id classification: `MinCardId = 2380000`, `MaxCardId = 2389999`, `IsCardId(itemId)`, and re-derives "card item" via `itemId >= 2380000`. `libs/atlas-constants/item/constants.go:43` already exposes `ClassificationConsumableMonsterCard = Classification(238)` and `:121` exposes `GetClassification(itemId Id) Classification`. The check should be `item.GetClassification(item.Id(itemId)) == item.ClassificationConsumableMonsterCard`.<br>- `card/model.go` and `collection/model.go` use raw `uint32` for `cardId` (an item id) and `characterId`. `libs/atlas-constants/item/constants.go:5` defines `type Id uint32`; `libs/atlas-constants/character/constants.go:3` defines `type Id uint32`. The new builder/Model/processor APIs should accept `item.Id` and `character.Id`, not `uint32`.<br>- `services/atlas-consumables/atlas.com/consumables/kafka/consumer/pickup/consumer.go:22,53` declares `const cardItemPrefix = 238` and gates routing on `cmd.ItemId/10000 != cardItemPrefix`. This duplicates `item.GetClassification` exactly and re-encodes `ClassificationConsumableMonsterCard` as a magic number.<br>- `services/atlas-channel/atlas.com/channel/monsterbook/processor.go:17-23,53,59,65` and `services/atlas-query-aggregator/atlas.com/query-aggregator/monsterbook/processor.go:13-19,69,75,82` use `uint32` for both `characterId` and `coverCardId`.<br>- `services/atlas-channel/atlas.com/channel/character/model.go:343-358`, `builder.go:59,148,193`, and `processor.go:175` add `coverCardId uint32` to the channel Character model — same wrapping issue.<br>- Saga-side `MonsterBookCountCondition` (`libs/atlas-saga/validation.go:45`) is a string token, which is fine; the underlying value is the integer count of unique cards (not an id), so atlas-constants does not apply for the count itself. |

## Domain Checklist — `collection` (atlas-monster-book)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists | PASS | `collection/builder.go:23-67`; also includes `CloneModelBuilder(m Model)` for immutable updates |
| DOM-02 | `ToEntity()` method | FAIL | `collection/entity.go` defines only `Migration` and `TableName()`. Entity construction is inlined into `collection/administrator.go:21-28` (`upsertStats`) instead of being expressed as `Model.ToEntity()`. |
| DOM-03 | `Make(Entity)` function | PASS | `collection/builder.go:78` |
| DOM-04 | `Transform` function | PASS | `collection/rest.go:26` |
| DOM-05 | `TransformSlice` function | WARN | The collection domain only ever transforms a single `Model` (per-character), so a `TransformSlice` is arguably unnecessary; however the guideline expects it on every domain `rest.go`. Mark WARN, not FAIL: scope-justified omission. |
| DOM-06 | Processor accepts `FieldLogger` | PASS | `collection/processor.go:66` |
| DOM-07 | Handlers pass `d.Logger()` | PASS | `character/resource.go:44, 61` |
| DOM-08 | POST/PATCH use `RegisterInputHandler` | PASS | `character/resource.go:33` |
| DOM-09 | Transform errors handled | FAIL | Same as `card`: `character/resource.go:50, 71` discard `Transform` errors |
| DOM-10 | Test DB has tenant callbacks | FAIL | Same as `card` |
| DOM-11 | Providers use lazy evaluation | PASS | `collection/provider.go:10-14` |
| DOM-12 | No `os.Getenv()` in handlers | PASS | (handlers live in `character/resource.go` which is shared) |
| DOM-13 | No cross-domain logic in handlers | PASS | |
| DOM-14 | Handlers don't call providers directly | PASS | |
| DOM-15 | No direct entity creation in handlers | PASS | |
| DOM-16 | `administrator.go` exists for writes | PASS | `collection/administrator.go` |
| DOM-17 | Domain error -> HTTP status mapping | FAIL | Same as `card` (handlers in shared file; PATCH always maps every domain error to 422 regardless of cause). Specifically: `setCover` returns `errors.New("collection row does not exist; cover requires owned card")` for missing rows (`collection/administrator.go:64`) but the handler can't distinguish that from a connection failure. |
| DOM-18 | JSON:API interface on REST models | PASS | `collection/rest.go:15-24, 43-52` |
| DOM-19 | Request models use flat structure | PASS | `PatchInput` is flat |
| DOM-20 | Table-driven tests | WARN | Same style nit as `card` |
| DOM-21 | No duplication of atlas-constants types | FAIL | `collection/model.go:14, 17, 21` use `uint32` for `characterId` and `coverCardId`. Should use `character.Id` and `item.Id`. |

## Sub-Domain Checklist — `character` (atlas-monster-book)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Has processor or uses parent processor | PASS | Handlers delegate to `card.NewProcessor` and `collection.NewProcessor`; no business logic in `character/resource.go` |
| SUB-02 | Has administrator for writes | PASS | Writes go through `collection.SetCoverAndEmit` -> `collection/administrator.go:setCover` |
| SUB-03 | Uses `RegisterInputHandler[T]` for POST | PASS | PATCH uses `rest.RegisterInputHandler[collection.PatchInput]` (`character/resource.go:33`); no POST routes exist |
| SUB-04 | No manual JSON parsing | PASS | No `json.NewDecoder`/`json.Unmarshal`/`io.ReadAll` in `character/resource.go` |

## External HTTP Client Checklist — `atlas-channel/monsterbook`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| EXT-01 | JSON:API target struct implements relationship interfaces | FAIL | `services/atlas-channel/atlas.com/channel/monsterbook/rest.go:5-25` — `CollectionRestModel` has `GetName/GetID/SetID` but **no** `SetToOneReferenceID` or `SetToManyReferenceIDs`. Per `libs/atlas-rest/CLAUDE.md:3-26`, every JSON:API target struct decoded by `requests.GetRequest[T]` must implement these stubs. The upstream `services/atlas-monster-book/atlas.com/monster-book/collection/rest.go:5-13` does not currently emit a `relationships` block, so today this happens to work — but it is a foot-gun: any future addition of relationships on the upstream resource will surface as a misleading "not found"/lookup error in atlas-channel. |
| EXT-02 | httptest-backed integration test exists | FAIL | `find services/atlas-channel/atlas.com/channel/monsterbook/ -name '*_test*'` returns nothing. There is no `httptest.NewServer`-backed test for the REST client at all. The decode path is therefore exercised only at runtime. |
| EXT-03 | Errors distinguish 404 from other failures | FAIL | `services/atlas-channel/atlas.com/channel/monsterbook/processor.go:60-67` — `GetByCharacterId` simply returns `p.ByCharacterIdProvider(characterId)()` without any `errors.Is(err, requests.ErrNotFound)` mapping. The caller `character/processor.go:175-181` (`MonsterBookCoverDecorator`) collapses every error path — connection refused, decode failure, 5xx, 404 — into the same "return undecorated model" branch. A 5xx outage is silently absorbed. |
| EXT-04 | Service URL not hardcoded; uses `RootUrl(domain)` | PASS | `services/atlas-channel/atlas.com/channel/monsterbook/requests.go:15-17` (`requests.RootUrl("MONSTER_BOOK")`) |

## External HTTP Client Checklist — `atlas-query-aggregator/monsterbook`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| EXT-01 | JSON:API target struct implements relationship interfaces | FAIL | `services/atlas-query-aggregator/atlas.com/query-aggregator/monsterbook/rest.go:10-40` — same omission as the channel client. |
| EXT-02 | httptest-backed integration test exists | FAIL | No `*_test*` files under `services/atlas-query-aggregator/atlas.com/query-aggregator/monsterbook/`. The condition evaluator's "graceful degrade to 0 on error" (`validation/context.go:209-218`) hides decode failures from validation tests. |
| EXT-03 | Errors distinguish 404 from other failures | FAIL | `monsterbook/processor.go:82-88` returns the raw error; the caller `validation/context.go:213-217` maps **every** error (404, 5xx, decode failure, network outage) to `0` unique cards. A deploy bug or a downed monster-book service silently turns "you have 25 cards" into "you have 0 cards" and gates the player out of monster-card-collector quests. |
| EXT-04 | Service URL not hardcoded; uses `RootUrl(domain)` | PASS | `monsterbook/requests.go:15-17` |

## Cross-Cutting Findings

### Kafka envelope drift (NEEDS-WORK)

The `monsterbook` envelope (`Command[B]`/`StatusEvent[B]`) is **redefined** in three places:

- `services/atlas-monster-book/atlas.com/monster-book/kafka/message/monsterbook/kafka.go:21-63`
- `services/atlas-channel/atlas.com/channel/kafka/message/monsterbook/kafka.go` (full duplicate including types and constants)
- `services/atlas-consumables/atlas.com/consumables/kafka/message/monsterbook/kafka.go:1-21` (subset — only `CardPickedUpBody`)

Same envelope, three sources of truth. The pattern is consistent with sibling Kafka envelopes (each service redeclares them) so this is not a new violation, but the new code propagates the antipattern and increases the surface area of drift risk. The same applies to `pickup`:

- `services/atlas-inventory/atlas.com/inventory/kafka/message/pickup/kafka.go:11-32` (producer side)
- `services/atlas-consumables/atlas.com/consumables/kafka/message/pickup/kafka.go:1-16` (consumer side)

These are byte-compatible today; any divergence will go undetected by the type system.

### `card.upsertCard` performs three SQL round-trips per call (WARN)

`services/atlas-monster-book/atlas.com/monster-book/card/administrator.go:19-70` does:

1. `SELECT ... First(&existing)`
2. `UPDATE` (or `INSERT`) based on `existing.Level`
3. A separate `UPDATE last_event_id` in the at-cap branch

A single `INSERT ... ON CONFLICT DO UPDATE` (matching `collection/administrator.go:upsertStats:29-34`) would be one round-trip and would side-step the lost-update race when two CARD_PICKED_UP commands with different eventIds arrive concurrently — the current select-then-update is racy if two replicas of the consumer ever process the same character (group rebalance window). Idempotency by `lastEventId` mitigates duplicate eventIds but does not protect against legitimate concurrent picks of two different cards.

### Channel `Collection` model lives outside the data domain (NEEDS-WORK)

`services/atlas-channel/atlas.com/channel/monsterbook/processor.go:16-30` declares `Collection` as a struct in the same file as the processor. It has private fields with getters but **no builder** — direct struct literal at `rest.go:29-37` (`Extract`) is the only constructor. The atlas-channel side does not need a builder for a value that is read-only on the wire, but it diverges from the immutable-model + builder pattern the guidelines require; there's no `Builder()`, no validation, and clients can construct zero-valued instances without checks. Same critique applies to `services/atlas-query-aggregator/atlas.com/query-aggregator/monsterbook/processor.go:13-40`.

### `RecomputeAndEmit` rebuilds slices, not counts (WARN — performance)

`collection/processor.go:128-202` calls `cp.GetByCharacterIdAndIsSpecial(characterId, false)` then `(..., true)`, materializing every card row to `len()` it. A `SELECT count(*) ... GROUP BY is_special` provider would be O(1) network round-trips and would not blow up for whales with 1000+ cards. Acceptable for v1 but worth a follow-up.

### Test isolation: `producer.ResetInstance` pattern (PASS)

`services/atlas-monster-book/atlas.com/monster-book/kafka/consumer/monsterbook/consumer_test.go:37-44` correctly stubs the Kafka producer manager via `kafkaProducer.ResetInstance` + a `noopWriter` and resets in `t.Cleanup`. This avoids the "test needs a real broker" trap that the SetCover test (`collection/processor_test.go:89-115`) explicitly avoids by exercising only the validation path.

## Security Review

`atlas-monster-book` is not an auth/token service; SEC-* checks are not applicable. No hardcoded secrets, JWT parsing, or redirect handlers are introduced.

## Backend Summary

### Blocking (must fix)

- **DOM-21** (`card`, `collection`, `atlas-consumables/pickup`, `atlas-channel/monsterbook`, `atlas-channel/character`, `atlas-query-aggregator/monsterbook`): replace raw `uint32` for cardId/characterId with `item.Id` / `character.Id`; replace `MinCardId`/`MaxCardId`/`IsCardId(itemId uint32)` (`card/model.go:9-22`) and `cardItemPrefix=238` (`atlas-consumables/.../pickup/consumer.go:22`) with `item.GetClassification(item.Id(...)) == item.ClassificationConsumableMonsterCard`.
- **DOM-09** (`atlas-monster-book/character/resource.go:50, 71, 136`): handle the `Transform` error returns; never use `_, _ := ...Transform(...)`.
- **DOM-17** (`atlas-monster-book/character/resource.go:47, 63, 66, 86, 116, 133`): map domain errors to HTTP statuses. `errors.New("cover requires owned card")` and `errors.New("cardId out of range")` should map to 422; `gorm.ErrRecordNotFound` to 404; everything else to 500. Today every PATCH error is 422 and every GET error is 500/404 with no signal to the client.
- **DOM-02** (`card/entity.go`, `collection/entity.go`): add `func (m Model) ToEntity() entity` and route writes through it. The current code constructs entity literals inside `upsertCard`/`upsertStats` (administrator.go) duplicating the field mapping that `Make(entity)` already encodes for the inverse direction.
- **EXT-01** (`atlas-channel/monsterbook/rest.go`, `atlas-query-aggregator/monsterbook/rest.go`): add no-op `SetToOneReferenceID(_, _ string) error` and `SetToManyReferenceIDs(_ string, _ []string) error` on both `CollectionRestModel`s. Required by `libs/atlas-rest/CLAUDE.md`.
- **EXT-02** (both clients): add httptest-backed integration tests that serve a real JSON:API fixture and assert the decoded `Collection` is populated. Without these, a missing relationship stub or schema drift surfaces only in production as silent "no monster book".
- **EXT-03** (both clients): distinguish `requests.ErrNotFound` from other errors. Currently a 5xx outage of atlas-monster-book degrades silently to "0 cards" in `validation/context.go:213-217` — this fails open and is sufficient to break monster-card-collector quest gating without any visible error.

### Non-Blocking (should fix)

- **DOM-05** (`card/rest.go`): add a `TransformSlice` so `character/resource.go:114` can use it instead of inlining `model.SliceMap(card.Transform)`. Aligns with the guideline.
- **DOM-10** (`card/administrator_test.go:11-21`, `collection/administrator_test.go:11-21`): call `database.RegisterTenantCallbacks(l, db)` in `newDB`. The current code is safe because every WHERE clause filters by tenantId explicitly, but the test does not enforce that invariant; a future provider that omits the explicit filter will pass tests and leak data.
- **DOM-20**: convert `processor_test.go` and `builder_test.go` to `tests := []struct{...}` + `t.Run(name, ...)` for failure-case clarity.
- **Concurrency** (`card/administrator.go:upsertCard`): replace the read-then-write with `INSERT ... ON CONFLICT (tenant_id, character_id, card_id) DO UPDATE SET level = ..., last_event_id = ...` to remove the lost-update race during consumer rebalance. Mirrors `collection/upsertStats`.
- **Performance** (`collection/RecomputeAndEmit`): replace the two `GetByCharacterIdAndIsSpecial` slice loads with a single grouped count provider.
- **Architecture** (`atlas-channel/monsterbook/processor.go`, `atlas-query-aggregator/monsterbook/processor.go`): introduce a `Builder` for the read-only `Collection` value to be consistent with the immutable-model pattern (low-priority; current code is not mutating but lacks invariant enforcement).
- **Kafka envelope duplication**: long-term, extract the `monsterbook` and `pickup` envelopes into a shared `libs/atlas-kafka-events/` (or similar). The current per-service redeclaration is the established pattern, so this is a project-wide concern, not specific to this task.
