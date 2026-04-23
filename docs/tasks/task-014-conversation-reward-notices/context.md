# Task 014 — Context: Key Files, Decisions, Dependencies

Last Updated: 2026-04-19

---

## Key Files

### Saga library (`libs/atlas-saga/`)
- `model.go` — payload structs (`AwardItemActionPayload`, `AwardExperiencePayload`, `AwardMesosPayload`, `DestroyAssetPayload`, slot variant). Add `ShowEffect bool`. Define `ForfeitQuestPayload`.
- `payloads.go`, `builder.go`, `unmarshal.go`, `validation.go` — touch as needed for the new field + new action.

### Packet library (`libs/atlas-packet/character/`)
- `status_message_body.go` — add `CharacterStatusMessageOperationDropLossItemBody`.
- `clientbound/status_message.go` — encoder modeled on `StatusMessageDropPickUpStackableItem` / `Unstackable` (lines 77–147). Derive stackable vs unstackable from `itemId` range.
- New companion test file under `libs/atlas-packet/character/`.

### atlas-npc-conversations (`services/atlas-npc-conversations/atlas.com/npc/`)
- `conversation/operation_executor.go` — primary edit surface.
  - `complete_quest` branch ~line 233.
  - `award_item` branch ~lines 848–894.
  - `award_mesos` branch ~lines 896–940.
  - `award_exp` branch ~lines 941–988.
  - `destroy_item` branch ~lines 1211–1251.
  - `destroy_item_from_slot` branch ~lines 1253–1293.
  - `createStepForOperation` planner around line 842 — host of new suppression rule.
  - Add new `forfeit_quest` branch.

### atlas-saga-orchestrator (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/`)
- `saga/` — step dispatch + payload handling. Read `ShowEffect` after step success.
- `kafka/` — register new producer for `conversation_reward_notice`; register new `ForfeitQuest` action consumer.
- `quest/` — outbound message construction for `ForfeitQuest`.

### atlas-character (`services/atlas-character/atlas.com/character/`)
- `kafka/consumer/character/consumer.go:172-186` — `AwardExperience` handler.
- `AwardExperienceAndEmit` implementation (in same package or processor) — append `White` + `Chat` distributions when `ShowEffect == true`.
- Meso update path — suppress `MesoChangedStatusEventBody` when `ShowEffect == false` (after audit).

### atlas-quest (`services/atlas-quest/atlas.com/quest/kafka/`)
- `consumer/` — add handler for new `ForfeitQuest` saga step.
- `producer/` — already emits `QuestForfeitedEventBody`; reuse.

### atlas-channel (`services/atlas-channel/atlas.com/channel/kafka/consumer/`)
- `character/consumer.go:249-307` — existing `announceExperienceGain` (no change, but verify `inChat` logic).
- `character/consumer.go:385` — existing `IncreaseMesoBody` writer (no change).
- `quest/consumer.go:117` — pattern reference for `CharacterQuestEffectBody` emission.
- `quest/consumer.go:132-157` — existing `handleQuestForfeited` (no change).
- New consumer package (e.g., `conversation_reward_notice/`) with `InitConsumers(l)(cmf)(groupId)`.

### Documentation
- `docs/npc_conversation_conversion_spec.md` — document `silent: bool` and `forfeit_quest`.
- `docs/tasks/task-014-conversation-reward-notices/prd.md` — source of truth for requirements.

## Key Decisions

1. **`ShowEffect` is opt-in** — zero value (`false`) preserves silent behavior for non-conversation callers. Conversation operation builders default to `true`; scripts can opt out with `silent: true`.
2. **One Kafka topic for ad-hoc item events** — `conversation_reward_notice`, scoped narrowly to item gain/loss. EXP and meso continue riding their existing status-event topics with source-side suppression.
3. **Suppression is planner-side, not orchestrator-side** — the conversation operation_executor knows the full step list and can flip `ShowEffect: false` on `award_item` steps before emission. The orchestrator emits according to the flag.
4. **Item-loss encoder derives stackable/unstackable from `itemId`** — caller passes only `(itemId, quantity)`; encoder mirrors the existing asset consumer's classification helper.
5. **`forfeit_quest` only updates journal + fires the existing forfeit packet** — item cleanup is explicitly out of scope (PRD §9.3).
6. **Suppression is `(itemId, quantity)` exact-tuple match preceding `complete_quest`** — quantity > coverage stays visible; post-`complete_quest` `award_item` is not suppressed (per current assumption).
7. **Use `docs/tasks/task-NNN-slug/` location** — per project memory, this convention superseded the old `docs/tasks/legacy-<feature-name>/` pattern on 2026-04-16.

## Dependencies

### Inter-task
- A1 (saga payload field) gates B1, C1, D1, D2.
- A2 (`ForfeitQuest` payload) gates B3, C2, C3.
- A3 (item-loss packet) gates E1.
- B1, B2, B3, B4 gate F1.
- C1 + A3 gate E1.
- All phases gate F2 (full build/test sweep) and F3 (acceptance walkthrough).

### External / pre-implementation
- Open Question §9.1 (item-loss opcode) — resolve before A3 finalization. Source: v83 client capture or compatible server reference.
- Open Question §9.2 (meso event consumer audit) — `grep` all consumers of `MesoChangedStatusEventBody` before D2.
- Open Question §9.4 (post-`complete_quest` suppression) — confirm "no" assumption with task owner during B4.

### Build verification scope
Per `CLAUDE.md`: any service whose shared lib (`libs/atlas-saga`, `libs/atlas-packet`) changes must have its Docker build verified. Affected services:
- atlas-npc-conversations
- atlas-saga-orchestrator
- atlas-character
- atlas-quest
- atlas-channel

## Conventions to Honor

- **Immutable models + builder pattern** for any new domain types.
- **Processor pattern**: `NewProcessor(l, ctx)` with pure `Method(mb)` and side-effecting `MethodAndEmit()`.
- **Consumer registration**: `InitConsumers(l)(cmf)(groupId)`.
- **Multi-tenancy**: `tenant.MustFromContext(ctx)` on emit; consumer parses tenant header.
- **Logging**: ctx-scoped structured logger; debug on success, info on session-not-connected (not error).
- **JSON:API**: not applicable here — no HTTP additions.
- **Tests**: table-driven where shape varies; fixture-based byte assertions for new packet bodies.
