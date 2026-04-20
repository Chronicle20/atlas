# Task 014 — Conversation Reward Notices: Task Checklist

Last Updated: 2026-04-19

Reference: `plan.md` (full detail), `context.md` (key files/decisions), `prd.md` (requirements).

---

## Phase A — Foundational types & packet (libs)

- [x] **A1** Added `ShowEffect bool` to `AwardItemActionPayload`, `AwardExperiencePayload`, `AwardMesosPayload`, `DestroyAssetPayload`, `DestroyAssetFromSlotPayload` in `libs/atlas-saga/payloads.go`.
- [x] **A2** Added `ForfeitQuest` saga action + `ForfeitQuestPayload { CharacterId, WorldId, QuestId, ShowEffect }` in `libs/atlas-saga/{model,payloads,unmarshal}.go`.
- [x] **A3** Added `CharacterStatusMessageOperationDropLossItemBody` + clientbound `StatusMessageDropLossStackableItem`/`StatusMessageDropLossUnStackableItem` encoders. §9.1 resolution: reused `DROP_PICK_UP` opcode with stackable variant writing negated `int32` quantity, unstackable variant matching the existing pickup shape (mode byte 2). Stackable/unstackable derived via `inventory.TypeFromItemId`. Round-trip unit tests added.

## Phase B — Conversation operation surface

- [x] **B1** Defaulted `ShowEffect: true` on the five reward step constructors in `operation_executor.go`.
- [x] **B2** Added `resolveSilent` helper; `silent: true` flips `ShowEffect: false` on each of the five ops.
- [x] **B3** Added `forfeit_quest` operation that builds a `ForfeitQuest` saga step with `ShowEffect: true` default.
- [x] **B4** Implemented `suppressAwardAssetByCompleteQuest` planner with reward-coverage tracking; table-driven tests cover: no `complete_quest`, matching reward, partial-quantity mismatch, already-silent, post-completion (no suppression), and double-claim of a single reward. Open Q §9.4 confirmed: post-`complete_quest` `award_item` is NOT suppressed.

## Phase C — Saga orchestrator & atlas-quest

- [x] **C1** Atlas-saga-orchestrator's asset consumer (`kafka/consumer/asset/consumer.go`) emits one `conversation_reward_notice` Kafka message per successful asset CREATED/DELETED/QUANTITY_CHANGED step when the saga's pending step is AwardAsset/DestroyAsset/DestroyAssetFromSlot with `ShowEffect == true`. New topic `EVENT_TOPIC_CONVERSATION_REWARD_NOTICE`. Body `{ characterId, kind, itemId, quantity }`. Tenant header rides via the standard producer.
- [x] **C2** Registered `ForfeitQuest` in orchestrator dispatch (`saga/handler.go`), routed to atlas-quest's existing `RequestForfeitQuest`. Added `handleQuestForfeitedEvent` consumer to mark the saga step completed.
- [x] **C3** atlas-quest already had a complete `handleForfeitQuestCommand` consumer + `Forfeit` processor + `EmitQuestForfeited` event emitter; verified end-to-end and reused without modification.

## Phase D — Character notices

- [x] **D1** Added `ShowEffect` to `AwardExperienceCommandBody` in both atlas-saga-orchestrator and atlas-character. Orchestrator passes the flag through; atlas-character's `AwardExperience` appends `White` + `Chat` distribution entries to the emitted `ExperienceChangedStatusEventBody` when `ShowEffect == true`, leaving the existing distribution shape untouched otherwise.
- [x] **D2** §9.2 audit: orchestrator consumes `MesoChangedStatusEventBody` for saga-step completion (`kafka/consumer/character/consumer.go`), so source-side suppression at atlas-character would break saga progression. Switched to channel-side suppression: added `ShowEffect bool` to `MesoChangedStatusEventBody` (atlas-character source, atlas-channel consumer); atlas-channel skips writing the chat line when `ShowEffect == false`. Existing non-conversation callers (storage operations, admin "give meso", quest meso rewards) updated to set `ShowEffect: true` to preserve current behavior.

## Phase E — Channel consumer

- [x] **E1** Added `services/atlas-channel/.../kafka/consumer/conversation_reward_notice/` consumer following `InitConsumers(l)(cmf)(groupId)`. `item_gain` → `CharacterEffectWriter` + `CharacterQuestEffectBody`. `item_loss` → `CharacterStatusMessageWriter` + new `CharacterStatusMessageOperationDropLossItemBody`. Session-miss path logs info and no-ops. Registered in `main.go`.

## Phase F — Documentation & verification

- [x] **F1** Updated `docs/npc_conversation_conversion_spec.md`: documented `silent: bool` on the six reward operations, added `forfeit_quest` entry, cross-referenced `award_item` + `complete_quest` suppression behavior.
- [x] **F2** `go build ./...` and `go test ./...` clean across `libs/atlas-saga`, `libs/atlas-packet`, `services/atlas-npc-conversations`, `services/atlas-saga-orchestrator`, `services/atlas-character`, `services/atlas-quest`, `services/atlas-channel`, and `services/atlas-messages` (touched for meso annotation).
- [ ] **F3** PRD §10 acceptance walk pending end-to-end manual verification on a connected v83 client.

## PRD §10 Acceptance Criteria Tracking

- [ ] Roger apple → item-gain effect + apple in inventory.
- [ ] NPC `destroy_item` → item-loss chat line + slot cleared.
- [ ] NPC `award_exp` → `You have gained N exp. (+N)` chat line + EXP updates.
- [ ] NPC `award_mesos` → existing meso chat line + meso updates.
- [ ] `silent: true` on each reward op → state change, no chat/effect.
- [ ] `forfeit_quest` operation → quest removed from journal + `ForfeitQuestRecord` packet fires.
- [ ] `award_item` → `complete_quest` (overlapping rewards) → reward effect renders exactly once.
- [ ] `award_item` followed by non-matching `complete_quest` → both effects render.
- [ ] Existing non-conversation callers unchanged (existing service tests still pass).
- [ ] All affected Go services build cleanly; existing tests pass.
- [ ] New unit tests pass: item-loss packet body, suppression planner.
- [ ] `docs/npc_conversation_conversion_spec.md` lists `silent` + documents `forfeit_quest`.

## Open Questions to Resolve During Implementation

- [x] §9.1 Item-loss packet opcode — Decision: reuse existing `DROP_PICK_UP` opcode with two new clientbound bodies (stackable writes negated `int32` quantity; unstackable mirrors the pickup mode-byte 2 shape). Stackable/unstackable derived via `inventory.TypeFromItemId`.
- [x] §9.2 Meso silent source (audit) — Decision: channel-side suppression. Source-side breaks orchestrator's saga-step completion (atlas-saga-orchestrator's `handleCharacterMesoChangedEvent` advances the saga on this event). Added `ShowEffect` to `MesoChangedStatusEventBody`; atlas-channel skips the chat line when false.
- [x] §9.3 `forfeit_quest` item cleanup — confirmed out of scope. Follow-up task id (if any): not filed; revisit if a script needs it.
- [x] §9.4 Suppression of post-`complete_quest` `award_item` — confirmed: not suppressed. Implemented as documented; tests cover the case.

## Notes / Decisions Log

_(Append dated entries as decisions are made during implementation.)_
