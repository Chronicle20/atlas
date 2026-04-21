# Task 015 — Context: Key Files, Decisions, Dependencies

Last Updated: 2026-04-20

---

## Key Files

### Saga library (`libs/atlas-saga/`)
- `payloads.go` — `QuestRewardItem` already defined at ~line 247. Add `Rewards []QuestRewardItem \`json:"rewards,omitempty"\`` to `StartQuestPayload` at ~line 252.
- `builder.go` / `unmarshal.go` — touch only if JSON round-trip coverage requires it for the new field (zero-value when absent).

### atlas-saga-orchestrator (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/`)
- `kafka/message/quest/kafka.go:~37` — `StartCommandBody`. Add `Rewards []ItemReward \`json:"rewards,omitempty"\``.
- `saga/handler.go:~1532` — `handleStartQuest`. Model after `handleCompleteQuest` at 1510–1528: extract `payload.Rewards`, convert to `[]questmessage.ItemReward`, forward to `RequestStartQuest`.
- `quest/producer.go:13` — `StartQuestCommandProvider`. Add `rewards []quest.ItemReward` parameter; set `Body.Rewards`.
- `quest/processor.go` — `RequestStartQuest`. Add `rewards []quest.ItemReward` parameter; forward to provider. Mirror `RequestCompleteQuest`.

### atlas-quest (`services/atlas-quest/atlas.com/quest/`)
- `kafka/message/quest/kafka.go:~37` — `StartCommandBody`. Add `Rewards []ItemReward` field (shape-compatible with orchestrator).
- `kafka/message/quest/kafka.go:100` — `QuestStartedEventBody`. Add `Items []ItemReward \`json:"items,omitempty"\``.
- `kafka/producer/quest/producer.go:100` — `EmitQuestStarted`. Add `items []questmessage.ItemReward` parameter; write to body.
- `quest/processor.go:18` — `EventEmitter` interface. Widen `EmitQuestStarted` signature.
- Mock event emitter — update to capture `items` and expose for test assertions.
- `quest/processor.go:189` — `Start()`. Add `externalRewards []questmessage.ItemReward` final parameter; apply override; pass `reportedItems` to `EmitQuestStarted`.
- `quest/processor.go:327` — `StartChained()`. Same shape as `Start()`.
- `quest/processor.go` — `processStartActions`. Change return from `error` to `([]questmessage.ItemReward, error)`; emit an `ItemReward` per positive-count `AddAwardItem`. Mirror `processEndActions` at ~line 796.
- `quest/resource.go` — HTTP handler calling `Start(...)`. Pass `nil` for `externalRewards`.
- Kafka command consumer for `StartCommandBody` — pass `c.Body.Rewards` into `Start(...)` / `StartChained(...)`.

### atlas-channel (`services/atlas-channel/atlas.com/channel/`)
- `kafka/message/quest/kafka.go:94` — `QuestStartedEventBody`. Add `Items []ItemReward` (shape-compatible with atlas-quest).
- `kafka/consumer/quest/consumer.go:57` — `handleQuestStarted`. Extract `e.Body.Items`; pass to `announceQuestStarted`.
- `kafka/consumer/quest/consumer.go:74` — `announceQuestStarted`. Add `items []quest.ItemReward` parameter. Write existing status-message packet; then if `len(items) > 0` write `CharacterQuestEffectBody("", rewards, 0)` — same call shape as `announceQuestCompleted` at line 117.
- No foreign-broadcast variant. Completion's foreign packet is for the completion animation only; start has no analog.

### atlas-npc-conversations (`services/atlas-npc-conversations/atlas.com/npc/`)
- `conversation/operation_executor.go:~795` — `createSagaForOperations`. After the existing `CompleteQuest` rewards-collection pass (lines 812–836), add a symmetric pass writing sibling rewards into `StartQuestPayload.Rewards`.
- `conversation/operation_executor.go:844` — add `suppressAwardAssetByStartQuest(built)` call alongside the existing completion suppressor.
- `conversation/operation_executor.go:874` — add `suppressAwardAssetByStartQuest` next to `suppressAwardAssetByCompleteQuest`. The two passes are independent — each inspects only its own action type.

### Reference (task-014)
- `docs/tasks/task-014-conversation-reward-notices/` — PRD/plan/context/tasks for the completion-side analog. Every symbol this task adds has a companion in the completion path; read those before implementing.

## Key Decisions

1. **Override semantic mirrors completion.** When the conversation supplies sibling `award_item` steps, the `Rewards` payload is non-nil and is authoritative for `QuestStartedEventBody.Items`. atlas-quest's own `processStartActions` saga still places WZ start items into the inventory; they just don't render. If an author wants both to render, they must declare `award_item` for each item explicitly.
2. **No `silent` opt-out on `start_quest`.** Parity with `complete_quest`. Author controls visibility through sibling ops and WZ data, not a per-operation flag.
3. **No feature flag.** New fields are `omitempty` / zero-value-safe. Old consumers tolerate new producers; new consumers fall back to WZ-sourced items when the field is empty.
4. **Two suppression helpers stay independent.** `suppressAwardAssetByStartQuest` and `suppressAwardAssetByCompleteQuest` each look only at their own action type. A conversation with both operations gets each pass applied once.
5. **Quantity-mismatch siblings stay visible.** Same rule as completion suppressor: only fully-covered `(itemId, quantity)` tuples are silenced.
6. **`processStartActions` omits negative-count items from the returned list.** `Count < 0` entries are consumed start-requirement items, not rewards. Mirror `processEndActions`.
7. **`Start()` already short-circuits on `StateStarted`.** No emission occurs on re-triggers; no change needed for idempotency (PRD §4.10).
8. **Chain follow-ups render both effects.** Completing quest A then auto-starting quest B fires the completion effect for A and the start effect for B. Both are intentional.
9. **Use `docs/tasks/task-NNN-slug/` location** — per project memory, this convention superseded `dev/active/<feature-name>/` on 2026-04-16.

## Dependencies

### Inter-task
- A1 (saga payload field) gates C4, E1.
- B1 (atlas-quest `StartCommandBody.Rewards`) gates C1 (wire compat).
- B2 (`EmitQuestStarted` widening) gates B4 and D1 (wire compat).
- B3 (`processStartActions` return type) gates B4.
- B4 (`Start`/`StartChained` signatures) gates B5 and all internal caller updates.
- C1, C2, C3 serially gate C4.
- D1 gates D2.
- E1 (sibling collection) gates E2 (suppression).
- All phases gate F1 (build sweep) and F2 (acceptance walk).

### External / pre-implementation
- None. All open questions in PRD §9 are resolved at spec time.

### Build verification scope
Per `CLAUDE.md`: `libs/atlas-saga` is touched, so any service consuming it must have its Docker build verified. Known consumers affected by this task:
- atlas-saga-orchestrator
- atlas-quest
- atlas-channel
- atlas-npc-conversations

Re-run `grep -r "atlas-saga" services/*/go.mod` before the sweep to catch any consumer not yet listed.

## Conventions to Honor

- **Immutable models + builder pattern** for any new domain types (not expected here — all additions are plain Kafka/payload structs).
- **Processor pattern**: `NewProcessor(l, ctx)` with pure `Method(mb)` and side-effecting `MethodAndEmit()`.
- **Multi-tenancy**: `tenant.MustFromContext(ctx)` on emit; consumer parses tenant header. No change to tenancy handling.
- **Consumer registration**: `InitConsumers(l)(cmf)(groupId)`. No new consumers in this task — existing consumers stay.
- **Logging**: ctx-scoped structured logger. Log `len(items)` at debug on emission to match the completion side's telemetry (verify pattern during implementation).
- **JSON:API**: not applicable — no HTTP additions.
- **Tests**: table-driven where shape varies; mock event emitter must capture new `items` slice for assertions.
