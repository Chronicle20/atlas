# Task 014 — Conversation Reward Notices: Implementation Plan

Last Updated: 2026-04-19
Status: Draft (ready for implementation)
PRD: `prd.md`

---

## 1. Executive Summary

Today, conversation-driven rewards (item, EXP, mesos) and item losses are silently applied to a character's state with no client-visible feedback. The PRD requires that every conversation-sourced reward step produces an effect/chat notice while leaving non-conversation callers (admin, loot, system grants) unchanged.

The strategy is to add an additive `ShowEffect bool` field to the existing saga reward payloads, default it to `true` in the conversation operation builders (with a `silent: true` script opt-out), introduce a new `forfeit_quest` saga action plus operation, and route conversation-sourced item gain/loss through a new `conversation_reward_notice` Kafka topic that atlas-channel consumes to write the appropriate v83 packets. EXP and meso ride their existing status-event paths with source-side suppression toggled by `ShowEffect`. A planner-side suppression rule prevents double-rendering when an `award_item` step is followed by a `complete_quest` whose reward list already covers it.

Scope spans seven services/libs and one docs file, with no DB migrations and full backwards compatibility (zero-value `ShowEffect == false` preserves current silent behavior).

## 2. Current State Analysis

- **Mesos already render**: `services/atlas-channel/.../character/consumer.go:385` writes `IncreaseMesoBody` from `MesoChangedStatusEventBody`.
- **Quest completion already renders rewards**: `services/atlas-channel/.../quest/consumer.go:117` emits `CharacterQuestEffectBody` on `QuestCompletedEventBody`.
- **Quest forfeit packet already exists** but only fires from `handleQuestForfeited` (`.../quest/consumer.go:132-157`) — no script path triggers it because there's no `forfeit_quest` saga action.
- **EXP gain has no chat notice path** for saga-sourced gains — `ExperienceChangedStatusEventBody` distributions don't include `White`/`Chat` entries, so atlas-channel's `announceExperienceGain` (`character/consumer.go:249-307`) doesn't set `inChat: true`.
- **`award_item` and `destroy_item` are silent end-to-end** — saga payloads have no `ShowEffect` flag, atlas-channel has no consumer for ad-hoc conversation item events, and there is no item-loss client packet writer in `libs/atlas-packet`.
- **Operation executor branches** in `services/atlas-npc-conversations/.../conversation/operation_executor.go` (around lines 233, 842–1293) build saga steps without any visibility flag.

## 3. Proposed Future State

- `libs/atlas-saga` payloads carry an additive `ShowEffect bool`; new `ForfeitQuest` action + payload added.
- `libs/atlas-packet` exposes a new item-loss status-message body parameterized by `(itemId, quantity)` with stackable/unstackable derivation matching the existing asset consumer.
- `services/atlas-npc-conversations` builders default `ShowEffect: true` for all reward ops, accept a `silent: bool` script param, implement the `(itemId, quantity)`-tuple suppression rule for `award_item` preceding `complete_quest`, and add a `forfeit_quest` operation.
- `services/atlas-saga-orchestrator` reads `ShowEffect` on reward steps and emits one `conversation_reward_notice` per item gain/loss step on a new Kafka topic. Registers + dispatches `ForfeitQuest`.
- `services/atlas-character` appends `White` + `Chat` distributions to `ExperienceChangedStatusEventBody` when `ShowEffect == true`; suppresses `MesoChangedStatusEventBody` emission when `ShowEffect == false`.
- `services/atlas-quest` consumes `ForfeitQuest`, updates journal, emits `QuestForfeitedEventBody`.
- `services/atlas-channel` registers a new consumer for `conversation_reward_notice`; writes `CharacterQuestEffectBody` for gains and the new item-loss body for losses.
- `docs/npc_conversation_conversion_spec.md` documents `silent` and `forfeit_quest`.

## 4. Implementation Phases

The implementation is decomposed into phases ordered to minimize cross-service breakage. Each phase ends with a build verification.

### Phase A — Foundational types & packet (libs)
Lay down additive saga payload changes and the new item-loss packet body. No consumer changes yet.

### Phase B — Conversation operation surface
Wire the `silent` param, default `ShowEffect: true`, add the `forfeit_quest` operation, and implement the `complete_quest` suppression rule. No event emission yet.

### Phase C — Saga orchestrator & atlas-quest
Have the orchestrator emit `conversation_reward_notice` for item gain/loss steps with `ShowEffect`, and register/dispatch the new `ForfeitQuest` action to atlas-quest, which performs the journal update and emits the existing forfeited event.

### Phase D — Character notices (EXP & mesos)
Update atlas-character to honor `ShowEffect` on EXP (adds `White`/`Chat` distributions) and meso (suppresses status event emission when false).

### Phase E — Channel consumer
Register the new `conversation_reward_notice` consumer and route to `CharacterQuestEffectBody` (gain) or the new item-loss body (loss).

### Phase F — Documentation & verification
Update `docs/npc_conversation_conversion_spec.md`. Build and test all affected services. Manual / scripted verification of the acceptance criteria.

## 5. Detailed Tasks

Effort key: **S** ≈ <½ day, **M** ≈ ½–1 day, **L** ≈ 1–2 days, **XL** ≈ multi-day.

### Phase A — Foundational types & packet

**A1. Add `ShowEffect` to reward payloads** — Effort: S
- Add `ShowEffect bool \`json:"showEffect"\`` to `AwardItemActionPayload`, `AwardExperiencePayload`, `AwardMesosPayload`, `DestroyAssetPayload`, and the slot variant if separate, in `libs/atlas-saga/model.go` (and `payloads.go`/`builder.go`/`unmarshal.go` as needed).
- Acceptance: `go build ./...` and existing tests in `libs/atlas-saga` pass; zero-value semantics preserved.
- Dependencies: none.

**A2. Add `ForfeitQuest` saga action + payload** — Effort: S
- Define `ForfeitQuestPayload { CharacterId uint32; QuestId uint32; ShowEffect bool }` in `libs/atlas-saga/model.go`.
- Register the new action type in builder/unmarshal/validation alongside existing quest actions.
- Acceptance: builder constructs the new step; round-trip JSON encode/decode works in unit tests.
- Dependencies: A1 layout conventions.

**A3. Add item-loss status-message body** — Effort: M
- New writer body in `libs/atlas-packet/character/status_message_body.go`: `CharacterStatusMessageOperationDropLossItemBody(itemId uint32, quantity uint32)`.
- New encoder in `libs/atlas-packet/character/clientbound/status_message.go` modeled on `StatusMessageDropPickUpStackableItem` / `Unstackable` (lines 77–147), deriving stackable vs unstackable from `itemId` range (mirror the asset consumer's existing logic — locate that helper before duplicating).
- Resolve Open Question §9.1 (signed-quantity wire format vs mode byte) before encoding.
- Unit tests in `libs/atlas-packet/character/` covering both stackable and unstackable item ids.
- Acceptance: tests pass; `go build ./...` clean.
- Dependencies: none.

### Phase B — Conversation operation surface

**B1. Default `ShowEffect: true` in reward step builders** — Effort: M
- In `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go`, set `ShowEffect: true` in the saga-step constructors for: `award_item` (~848–894), `award_mesos` (~896–940), `award_exp` (~941–988), `destroy_item` (~1211–1251), `destroy_item_from_slot` (~1253–1293).
- Acceptance: existing operation-executor tests still pass; new test cases assert `ShowEffect == true` by default on each constructed step.
- Dependencies: A1.

**B2. Add `silent: bool` script param** — Effort: S
- Parse `silent` from the operation params in each branch above; if `true`, override `ShowEffect: false`.
- Acceptance: table-driven test verifies `silent: true` flips the flag.
- Dependencies: B1.

**B3. Add `forfeit_quest` operation** — Effort: M
- New branch in `createStepForOperation` mirroring `complete_quest` / `start_quest`: required `questId: uint32`, optional `silent: bool`.
- Builds a `ForfeitQuest` saga step with `ShowEffect: true` default.
- Acceptance: operation parses, validates required `questId`, errors on missing param; constructed step references new action.
- Dependencies: A2.

**B4. Implement `complete_quest` reward suppression** — Effort: L
- In `createStepForOperation` (`operation_executor.go:842`) when planning a saga that contains `complete_quest`, walk **preceding** steps and set `ShowEffect: false` on any `award_item` whose `(itemId, quantity)` exactly matches an entry in the `complete_quest` reward list.
- Quantity > reward coverage → not suppressed (extras are intentional).
- `award_exp` and `award_mesos` are not suppressed.
- Confirm Open Question §9.4: post-`complete_quest` `award_item` is **not** suppressed (current assumption).
- Table-driven unit tests covering: no `complete_quest` → no suppression; matching reward → suppression; partial-quantity mismatch → no suppression; `silent: true` already false → unchanged.
- Acceptance: tests pass; suppression observable in the constructed step list.
- Dependencies: B1, B2.

### Phase C — Saga orchestrator & atlas-quest

**C1. Emit `conversation_reward_notice` from orchestrator** — Effort: M
- In atlas-saga-orchestrator, on successful completion of an `AwardItem` or `DestroyAsset` (and slot variant) step where `ShowEffect == true`, emit one `conversation_reward_notice` Kafka message.
- Body: `{ characterId, kind: "item_gain" | "item_loss", itemId, quantity }` (positive `quantity`; sign in `kind`).
- Tenant header per project convention.
- Topic name registered alongside other orchestrator topics.
- Acceptance: integration-style test (or focused unit) confirms emission count and payload shape; no emission when `ShowEffect == false`.
- Dependencies: A1.

**C2. Register `ForfeitQuest` action handler in orchestrator** — Effort: S
- Route the new action to atlas-quest via the existing saga-step dispatch pattern.
- Acceptance: dispatch picks up `ForfeitQuest` and produces the consumer-bound message.
- Dependencies: A2.

**C3. atlas-quest `ForfeitQuest` consumer** — Effort: M
- Add a consumer for the new `ForfeitQuest` saga step in `services/atlas-quest/atlas.com/quest/kafka/consumer/`.
- Update the quest journal (remove from active) and emit the existing `QuestForfeitedEventBody`.
- Out of scope per Open Question §9.3: item cleanup on forfeit.
- Acceptance: emitting a `ForfeitQuest` step results in journal removal + forfeited event; atlas-channel's existing `handleQuestForfeited` writes the client packet unchanged.
- Dependencies: A2, C2.

### Phase D — Character notices

**D1. EXP `White` + `Chat` distributions** — Effort: M
- In `services/atlas-character/.../character/consumer.go:172-186` (and `AwardExperienceAndEmit`), when payload `ShowEffect == true`, append `White` and `Chat` distribution entries to `ExperienceChangedStatusEventBody`.
- When `ShowEffect == false`, leave distributions as today.
- Acceptance: integration verifies chat line appears with `silent: false` and is absent with `silent: true`. EXP totals always update.
- Dependencies: A1.

**D2. Suppress `MesoChangedStatusEventBody` when `ShowEffect == false`** — Effort: M
- In atlas-character's meso update path, audit the consumers of `MesoChangedStatusEventBody` (Open Question §9.2) before suppressing. If safe, source-side suppress when `ShowEffect == false` on `AwardMesosPayload`. Otherwise, switch to channel-side suppression and document.
- Acceptance: `silent: true` on `award_mesos` produces no chat line; balance still updates.
- Dependencies: A1; resolve §9.2 audit.

### Phase E — Channel consumer

**E1. Register `conversation_reward_notice` consumer** — Effort: M
- New consumer package under `services/atlas-channel/atlas.com/channel/kafka/consumer/` (e.g., `conversation_reward_notice/`) following `InitConsumers(l)(cmf)(groupId)`.
- `kind == "item_gain"` → `CharacterEffectWriter` writes `CharacterQuestEffectBody("", []QuestReward{{ItemId, Amount: int32(quantity)}}, 0)`.
- `kind == "item_loss"` → `CharacterStatusMessageWriter` writes the new item-loss body from A3.
- Session lookup miss → log info + no-op.
- Wire registration in the channel's consumer init.
- Acceptance: consumer registers; with a connected session, both packet types are dispatched correctly; missing session is a no-op without error.
- Dependencies: A3, C1.

### Phase F — Documentation & verification

**F1. Update `docs/npc_conversation_conversion_spec.md`** — Effort: S
- Document `silent: bool` as accepted on `award_item`, `award_exp`, `award_mesos`, `destroy_item`, `destroy_item_from_slot`, `forfeit_quest`.
- Add the `forfeit_quest` operation entry under §Operations.
- Acceptance: spec includes both additions; cross-reference suppression rule for `award_item` + `complete_quest`.
- Dependencies: B2, B3, B4.

**F2. Cross-service build & test sweep** — Effort: M
- `go build ./...` and `go test ./...` in: `libs/atlas-saga`, `libs/atlas-packet`, `services/atlas-npc-conversations`, `services/atlas-saga-orchestrator`, `services/atlas-character`, `services/atlas-quest`, `services/atlas-channel`.
- Verify Docker builds for any service whose shared lib touched (per CLAUDE.md).
- Acceptance: all builds and tests green.
- Dependencies: all prior phases.

**F3. Acceptance-criteria walkthrough** — Effort: M
- Walk each PRD §10 bullet against the implementation; record evidence (test names, log/screenshot references) in `tasks.md`.
- Acceptance: every checkbox in PRD §10 is verified or has an explicit follow-up.
- Dependencies: F2.

## 6. Risk Assessment & Mitigation

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Item-loss packet opcode wrong (Open Q §9.1) — no chat line or client desync | Medium | High | Confirm against v83 client capture or compatible server before E1; gate behind unit test fixtures with byte-level assertions. |
| Suppressing `MesoChangedStatusEventBody` breaks an unrelated downstream (Open Q §9.2) | Medium | Medium | Grep all consumers of the event before D2; if any non-chat consumer depends on it, switch to channel-side suppression. |
| Suppression planner misses tuple match due to ordering or quantity edge case | Medium | Medium | Table-driven tests in B4 covering all enumerated cases; assert step list shape, not just side effects. |
| Non-conversation callers accidentally start firing notices | Low | High | Zero-value `ShowEffect == false` preserves silent path; add a test that constructs each payload via existing non-conversation builder and asserts `ShowEffect == false`. |
| Adding distributions to `ExperienceChangedStatusEventBody` changes packet shape for other consumers | Low | Medium | Read all consumers of the event in atlas-channel + others; the `Distributions` slice is already iterated, so appending entries is additive — verify with a focused test. |
| `forfeit_quest` without item cleanup leaves orphan items (Open Q §9.3) | Low | Low | Out of scope; document as known limitation; follow-up task if a script needs it. |
| Cross-service refactor cycle longer than estimated | Medium | Medium | Phase ordering minimizes cross-breakage; each phase has its own build gate. Expect 2–3 fix-and-rebuild cycles per CLAUDE.md guidance. |

## 7. Success Metrics

- All PRD §10 acceptance criteria pass.
- Roger's apple conversation produces an item-gain effect end-to-end on a v83 client.
- Zero regressions in existing service test suites across the seven affected packages.
- Suppression unit tests cover all four enumerated cases in B4.
- Item-loss packet unit tests cover stackable + unstackable.

## 8. Required Resources & Dependencies

- Local Go toolchain across all affected services.
- Access to a v83-compatible client capture or reference encoder for resolving §9.1 (item-loss opcode). If unavailable, accept the v83-convention default (negated quantity in stackable body) and verify in a manual session.
- Existing producer/consumer plumbing for new Kafka topic — no infra changes required (Kafka topic is auto-created by convention).
- Familiarity with: saga payload conventions in `libs/atlas-saga`, the operation-executor planner pattern, `CharacterEffectWriter` / `CharacterStatusMessageWriter` in atlas-channel.

## 9. Timeline Estimate

- **Phase A:** 1 day (S+S+M). Independent, can parallel.
- **Phase B:** 1.5–2 days (M+S+M+L). Sequential within phase.
- **Phase C:** 1.5 days (M+S+M).
- **Phase D:** 1 day (M+M); D2 gated on §9.2 audit.
- **Phase E:** 1 day (M).
- **Phase F:** 0.5–1 day (S+M+M).

**Total:** ~6–7 working days for one engineer; ~4–5 with parallelization across libs vs services.

## 10. Open Questions Recap (from PRD §9)

These must be resolved during implementation, not deferred:
1. Item-loss packet opcode — gate before A3 finalization.
2. Meso silent source — audit before D2.
3. `forfeit_quest` item cleanup — explicitly out of scope.
4. Post-`complete_quest` `award_item` suppression — confirm "no" assumption during B4.
