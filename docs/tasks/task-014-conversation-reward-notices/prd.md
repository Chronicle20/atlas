# Conversation Reward Notices — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-19
---

## 1. Overview

When an NPC conversation (or a quest completion driven from one) rewards the player with an item, EXP, or mesos — or takes something away — the v83 client expects a short-lived visual effect and/or a chat line ("You have obtained 1 Apple", "You have gained 100 EXP", "-1 Apple"). Atlas today only performs the state mutation: `award_item` lands in the inventory, `award_exp` updates the character sheet, `destroy_item` silently clears the slot. The player sees no acknowledgement beyond the stat/inventory delta, which makes tutorial-style conversations like Roger's apple feel broken.

Two of these paths are already nearly wired — meso gain already writes `IncreaseMesoBody` to chat (`services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go:385`), and quest completion already emits `CharacterQuestEffectBody` with rewards (`services/atlas-channel/atlas.com/channel/kafka/consumer/quest/consumer.go:117`). What's missing is (a) EXP chat notices for saga-sourced gains, (b) any notice at all for ad-hoc item gain/loss from conversations, and (c) the ability for a script to forfeit a quest end-to-end.

This task closes those gaps by threading a `ShowEffect` signal through the saga payloads, emitting a new `conversation_reward_notice` Kafka event from atlas-saga-orchestrator on conversation-sourced reward steps, adding the missing item-loss and forfeit plumbing, and rendering the appropriate client packets from atlas-channel.

## 2. Goals

Primary goals:
- Every conversation-sourced `award_item` causes the client to see a quest-style item-gain effect with the item id and quantity.
- Every conversation-sourced `destroy_item` / `destroy_item_from_slot` causes the client to see an item-loss chat line with the item id and (signed) quantity.
- Every conversation-sourced `award_exp` causes the existing EXP chat notice (`+N exp`) to appear.
- Every conversation-sourced `award_mesos` causes the existing meso chat notice to appear (mostly already works — verify and keep honoring `silent`).
- Scripts can forfeit a quest via a new `forfeit_quest` operation, and the existing quest-forfeited client packet fires.
- Existing JSON conversation scripts require **no changes** to benefit from the new notices. A per-operation opt-out (`silent: true`) is available for the rare hidden-bookkeeping case.
- Non-conversation callers of the saga actions (admin tools, loot flows, system grants) keep their current behavior and do not start double-firing notices.

Non-goals:
- No new UI in atlas-ui.
- No changes to which items/exp/mesos are awarded — only whether a notice is rendered.
- No rework of the monster-drop or field-pickup notice paths (they already have their own flows).
- No cross-channel / party-wide effect broadcast for these notices — they are single-character.
- No new sound packets, animation tweaks, or client-side behavior beyond what the existing effect/status-message writers already produce.
- No redesign of the JSON script schema beyond adding `silent` and the new `forfeit_quest` operation.
- No changes to `complete_quest`'s own reward-rendering behavior (it already calls `CharacterQuestEffectBody` with rewards at `services/atlas-channel/atlas.com/channel/kafka/consumer/quest/consumer.go:117`).

## 3. User Stories

- As a player, when Roger gives me an apple, I want to see the item-gain effect and a chat line confirming the apple so I know the NPC actually did something.
- As a player, when an NPC consumes a quest item from my inventory, I want a chat line telling me what was taken, so it doesn't look like the slot silently vanished.
- As a player, when I hand in a quest via an NPC, I want the EXP chat line to appear alongside the item-reward effect, so I can see what I earned.
- As a player, when an NPC lets me forfeit a quest mid-conversation, I want the quest to actually forfeit (journal updates, client sees the forfeit packet).
- As a conversation script author, I want reward notices to be the default so existing scripts don't need retrofitting, but I want a `silent: true` escape hatch for setup/housekeeping operations that should not be visible.
- As a developer on a non-conversation code path (admin tool, loot system), I want the saga payload changes to be backwards compatible — omitting `ShowEffect` keeps the current silent behavior.

## 4. Functional Requirements

### 4.1 Saga payload field

The following saga payloads in `libs/atlas-saga/model.go` gain one new boolean field:

- `AwardItemActionPayload` — add `ShowEffect bool \`json:"showEffect"\``
- `AwardExperiencePayload` — add `ShowEffect bool \`json:"showEffect"\``
- `AwardMesosPayload` — add `ShowEffect bool \`json:"showEffect"\``
- `DestroyAssetPayload` — add `ShowEffect bool \`json:"showEffect"\``
- Slot variant (if distinct payload exists for `destroy_item_from_slot`) — add `ShowEffect bool \`json:"showEffect"\``

Zero-value semantics: `ShowEffect == false` means **no notice**. This preserves existing behavior for every caller that does not opt in. Non-conversation callers do not need to change.

A new saga action `ForfeitQuest` is added with payload `ForfeitQuestPayload { CharacterId uint32; QuestId uint32; ShowEffect bool }`.

### 4.2 Conversation operation defaults

In `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go`, the saga-step builders for the following operations set `ShowEffect: true` by default:

- `award_item` (`createStepForOperation` branch around line 848–894)
- `award_exp` (branch around line 941–988)
- `award_mesos` (branch around line 896–940)
- `destroy_item` (branch around line 1211–1251)
- `destroy_item_from_slot` (branch around line 1253–1293)
- `forfeit_quest` (new — see §4.3)
- `complete_quest` (branch around line 233) — sets `ShowEffect: false` on any **preceding** `award_item` steps in the same saga whose `(itemId, quantity)` pair is covered by the `complete_quest` reward list (suppression — see §4.6)

Each of these operations also accepts an optional `silent: bool` parameter (default `false`). When `silent == true`, the builder sets `ShowEffect: false`. The JSON schema documentation in `docs/npc_conversation_conversion_spec.md` is updated accordingly.

### 4.3 `forfeit_quest` operation

Add a new operation to `operation_executor.go` mirroring `complete_quest` / `start_quest`:

- Operation type: `forfeit_quest`
- Params: `questId: uint32` (required), `silent: bool` (optional, default `false`)
- Saga action: `ForfeitQuest` (new — see §4.1)
- `ShowEffect` defaults to `true`

Handler side: atlas-quest consumes the `ForfeitQuest` saga step, updates quest journal state, and emits the existing `QuestForfeitedEventBody` Kafka event. Atlas-channel's `handleQuestForfeited` at `services/atlas-channel/atlas.com/channel/kafka/consumer/quest/consumer.go:132-157` already writes `CharacterStatusMessageOperationForfeitQuestRecordBody` — no change needed on the channel side for the forfeit packet itself.

Document the operation in `docs/npc_conversation_conversion_spec.md` §Operations.

### 4.4 EXP notice (atlas-character)

In atlas-character's `AwardExperience` handler (`services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go:172-186` and the `AwardExperienceAndEmit` implementation), when the incoming saga payload has `ShowEffect == true`:

- The emitted `ExperienceChangedStatusEventBody` MUST include `White` and `Chat` entries in its `Distributions` slice for the gained amount. This causes atlas-channel's existing `announceExperienceGain` (`services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go:249-307`) to set `inChat: true` on the packet and render the chat line.
- When `ShowEffect == false`, distributions are emitted as they are today (no white/chat entries), preserving silent behavior for non-conversation paths.

No changes required in atlas-channel for EXP.

### 4.5 Meso notice (verify-only)

Meso gain from `award_mesos` already traverses: operation_executor → `saga.AwardMesos` → atlas-character meso update → `MesoChangedStatusEventBody` → atlas-channel `handleStatusEventMesoChanged` → `CharacterStatusMessageOperationIncreaseMesoBody` (`services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go:385`).

Requirement: when `ShowEffect == false` on `AwardMesosPayload`, atlas-character MUST NOT emit `MesoChangedStatusEventBody` — or atlas-channel MUST suppress the status message. Pick the former (source-side suppression) because it keeps atlas-channel ignorant of the `ShowEffect` signal for meso. Verify with an integration-style test that `silent: true` on `award_mesos` produces no chat line while still crediting the mesos.

### 4.6 Suppression inside quest-completion flows

When a conversation's saga plan contains a `complete_quest` step and one or more preceding `award_item` steps, the planner in `operation_executor.createStepForOperation` (file ref: `operation_executor.go:842`) MUST inspect the `complete_quest` rewards list and set `ShowEffect: false` on each preceding `award_item` step whose `(itemId, quantity)` is covered by those rewards. This prevents double-rendering since `complete_quest`'s existing `CharacterQuestEffectBody` emission already shows the item rewards.

Scope of suppression:
- Match is by exact `(itemId, quantity)` tuple.
- If an `award_item` step grants a quantity larger than the `complete_quest` reward covers, it is **not** suppressed (the conversation intends to grant extra).
- Preceding `award_exp` and `award_mesos` steps are NOT suppressed — `CharacterQuestEffectBody` only renders items. EXP and meso notices still fire independently.

### 4.7 New Kafka event: `conversation_reward_notice`

Atlas-saga-orchestrator, on successful completion of a saga step whose payload has `ShowEffect == true`, emits one message on a new Kafka topic `conversation_reward_notice`. One message per reward step.

Message body (JSON, tenant-scoped via header per the project convention):
```
{
  "characterId": uint32,
  "kind": "item_gain" | "item_loss",
  "itemId": uint32,
  "quantity": uint32  // always positive; kind encodes sign
}
```

EXP and meso do not use this event — they ride their existing status-event paths (§4.4, §4.5).

### 4.8 New atlas-channel consumer for `conversation_reward_notice`

Atlas-channel registers a consumer for the new topic. Handler behavior:

- `kind == "item_gain"` → write `CharacterQuestEffectBody("", []QuestReward{{ItemId: itemId, Amount: int32(quantity)}}, 0)` via `CharacterEffectWriter` to the target character's session (pattern matches `services/atlas-channel/atlas.com/channel/kafka/consumer/quest/consumer.go:117`).
- `kind == "item_loss"` → write a new item-loss status message (see §4.9) via `CharacterStatusMessageWriter`.

If the target character is not connected on this channel, the handler no-ops (standard session-lookup pattern).

### 4.9 New item-loss packet

Add a new writer body in `libs/atlas-packet/character/status_message_body.go` modeled on the existing `StatusMessageDropPickUpStackableItem` / `StatusMessageDropPickUpUnStackableItem` encoders (lines 77–147 in `libs/atlas-packet/character/clientbound/status_message.go`), parameterized for loss:

- `CharacterStatusMessageOperationDropLossItemBody(itemId uint32, quantity uint32) ...`
- Internally uses whichever opcode/mode byte the v83 client interprets as a loss line (implementation detail; the existing DropPickUp stackable body with a negated quantity is the most common v83 convention). The goal is a client chat line of the form `-<qty> <item name>`.
- The encoder MUST NOT branch on stackable vs unstackable from caller input — derive it from `itemId` range the same way the asset consumer does today.

A companion unit test in `libs/atlas-packet/character/` covers both stackable and unstackable item ids.

## 5. API Surface

No HTTP/JSON:API changes.

Kafka additions:
- **New topic:** `conversation_reward_notice`. Producer: atlas-saga-orchestrator. Consumer: atlas-channel.
- **Existing topic, new message type:** `saga` adds `ForfeitQuest` action type. (Name the type per the existing naming convention in `libs/atlas-saga/`.)

Script schema additions (documented in `docs/npc_conversation_conversion_spec.md`):
- `silent: bool` — optional, default `false`, accepted on `award_item`, `award_exp`, `award_mesos`, `destroy_item`, `destroy_item_from_slot`, `forfeit_quest`.
- `forfeit_quest` operation with `questId: uint32` param.

## 6. Data Model

No database migrations. All state flows via Kafka events and saga payloads in memory.

Saga payload changes are additive; `ShowEffect bool` field's zero value preserves existing silent behavior for every caller that does not set it.

## 7. Service Impact

| Service | Change |
|---|---|
| `libs/atlas-saga` | Add `ShowEffect` field to `AwardItemActionPayload`, `AwardExperiencePayload`, `AwardMesosPayload`, `DestroyAssetPayload`, and any `DestroyAssetFromSlotPayload`. Add new `ForfeitQuest` action + `ForfeitQuestPayload`. |
| `libs/atlas-packet` | Add item-loss status-message body + clientbound encoder. Unit tests. |
| `services/atlas-npc-conversations` | Add `silent` param handling across reward ops. Add `forfeit_quest` operation. Implement suppression for `award_item` steps whose reward is covered by a following `complete_quest`. Default `ShowEffect: true` on conversation-sourced reward steps. |
| `services/atlas-saga-orchestrator` | Read `ShowEffect` from reward-step payloads. On success, emit `conversation_reward_notice` for item gain/loss. Register the new `ForfeitQuest` action and route it to atlas-quest. |
| `services/atlas-character` | In `AwardExperience`, append `White` + `Chat` distributions when `ShowEffect == true`. In meso update, suppress `MesoChangedStatusEventBody` emission when `ShowEffect == false`. |
| `services/atlas-quest` | Handle the new `ForfeitQuest` saga step: update journal and emit `QuestForfeitedEventBody`. |
| `services/atlas-channel` | New consumer for `conversation_reward_notice`. Writes `CharacterQuestEffectBody` for gain and the new item-loss status message for loss. No changes to existing EXP, meso, or quest-forfeit consumers. |
| `docs/npc_conversation_conversion_spec.md` | Document `silent` param and `forfeit_quest` operation. |

## 8. Non-Functional Requirements

**Performance:** One extra Kafka message per conversation-sourced item reward step. Conversations are interactive and low-throughput; impact is negligible. No additional DB reads or writes.

**Backwards compatibility:** All saga-payload changes are additive with zero-value-preserves-current-behavior semantics. Existing scripts require no updates. Existing non-conversation callers (admin, loot, system grants) do not start firing notices because they do not set `ShowEffect`.

**Multi-tenancy:** All new Kafka messages carry the tenant header per project convention (`tenant.MustFromContext(ctx)` on emit; consumer parses the header). The new consumer follows the established registration pattern `InitConsumers(l)(cmf)(groupId)`.

**Observability:** All new handlers use the established logger pattern (`ctx`-scoped structured logging) and emit debug-level logs on effect emission. Failures to look up the session are logged at info, not error (session-not-connected is a normal case).

**Security:** No new external surface. All new Kafka topics are internal.

**Testing:** The new item-loss packet body has unit tests for stackable + unstackable. The suppression logic in `createStepForOperation` has table-driven unit tests covering: no `complete_quest` → no suppression; `complete_quest` with matching reward → suppression; `complete_quest` with partial quantity mismatch → no suppression; `silent: true` → `ShowEffect: false` regardless of position. Integration-level verification for EXP and meso silent behavior.

## 9. Open Questions

1. **Item-loss packet opcode.** v83 convention is to reuse the `DropPickUp` stackable/unstackable status-message body with a negative quantity, but the Atlas encoder uses `uint32` for quantity. Resolution options: (a) introduce a signed-quantity wire encoding in a dedicated `DropLossItem` body, or (b) reuse the existing body and flip a mode byte. Confirm against a v83 client capture or compatible server implementation during implementation.
2. **Meso silent source.** §4.5 proposes suppressing `MesoChangedStatusEventBody` at atlas-character when `ShowEffect == false`. If other downstream consumers depend on that event for reasons unrelated to the chat line, suppression may break them. Audit `character_status_event` consumers before implementing and, if needed, switch to the channel-side suppression alternative.
3. **`forfeit_quest` item cleanup.** Some quests hold items that are consumed on forfeit. Out of scope for this task; `forfeit_quest` only updates journal state and fires the forfeit packet. Cleanup behavior can be added as a follow-up task once the operation exists.
4. **Suppression scope.** §4.6 suppresses `award_item` preceding `complete_quest`. Should `award_item` steps that appear **after** `complete_quest` in the same saga be suppressed similarly? Current assumption: no (post-completion grants are intentional extras). Confirm during implementation.

## 10. Acceptance Criteria

- [ ] Talking to Roger and selecting the apple option shows the quest-style item-gain effect and the apple appears in inventory.
- [ ] An NPC `destroy_item` step produces a chat line with the lost item and quantity; the item is removed from inventory.
- [ ] An NPC `award_exp` step produces a chat line of the form `You have gained N exp. (+N)` and the EXP total updates.
- [ ] An NPC `award_mesos` step produces the existing meso chat line and the meso total updates.
- [ ] Setting `silent: true` on any of `award_item`, `award_exp`, `award_mesos`, `destroy_item`, `destroy_item_from_slot`, `forfeit_quest` causes the state change to occur with no chat line or effect.
- [ ] A `forfeit_quest` operation in a conversation removes the quest from the character's active journal and fires the existing `ForfeitQuestRecord` client packet.
- [ ] A conversation that contains `award_item` → `complete_quest` with overlapping rewards renders the reward effect exactly once (via `complete_quest`), not twice.
- [ ] A conversation that contains `award_item` followed by a non-matching `complete_quest` renders the item-gain effect from `award_item` and the quest-complete effect from `complete_quest` independently.
- [ ] Existing non-conversation callers of `AwardAsset`, `AwardExperience`, `AwardMesos`, and `DestroyAsset` continue to function with no behavioral change (verified by running existing service tests unchanged).
- [ ] All affected Go services build cleanly and all existing unit/integration tests pass.
- [ ] New unit tests for the item-loss packet body and for the suppression planner logic pass.
- [ ] `docs/npc_conversation_conversion_spec.md` lists `silent` as a valid param on each reward operation and documents the new `forfeit_quest` operation.
