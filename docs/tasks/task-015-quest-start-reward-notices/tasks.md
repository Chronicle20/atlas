# Task 015 — Quest Start Reward Notices: Task Checklist

Last Updated: 2026-04-20 (implementation)

Reference: `plan.md` (full detail), `context.md` (key files/decisions), `prd.md` (requirements).

---

## Phase A — Foundational saga type (libs)

- [x] **A1** Add `Rewards []QuestRewardItem \`json:"rewards,omitempty"\`` to `StartQuestPayload` in `libs/atlas-saga/payloads.go` (~line 252). Reuse existing `QuestRewardItem` type. Update `unmarshal.go` / `builder.go` only if coverage requires.

## Phase B — atlas-quest internals

- [x] **B1** Add `Rewards []ItemReward \`json:"rewards,omitempty"\`` to `StartCommandBody` in `services/atlas-quest/atlas.com/quest/kafka/message/quest/kafka.go` (~line 37).
- [x] **B2** Add `Items []ItemReward \`json:"items,omitempty"\`` to `QuestStartedEventBody` (~line 100). Widen `EmitQuestStarted` in `kafka/producer/quest/producer.go:100` and the `EventEmitter` interface at `quest/processor.go:18`. Update the mock to capture `items` for test assertions.
- [x] **B3** Change `processStartActions` return type from `error` to `([]questmessage.ItemReward, error)`. Emit one `ItemReward{ItemId, Amount: int32(Count)}` per positive-count `AddAwardItem`; skip `Count < 0`. Mirror `processEndActions` at ~line 796.
- [x] **B4** Add `externalRewards []questmessage.ItemReward` as the new final parameter on `Start()` (processor.go:189) and `StartChained()` (processor.go:327). Apply `reportedItems := awardedItems; if len(externalRewards) > 0 { reportedItems = externalRewards }` and pass to `EmitQuestStarted`. Update `resource.go` to pass `nil`. Update any other in-process callers.
- [x] **B5** Wire `c.Body.Rewards` into `Start(...)` / `StartChained(...)` in the atlas-quest command consumer for `StartCommandBody`.

## Phase C — atlas-saga-orchestrator threading

- [x] **C1** Add `Rewards []ItemReward` to `StartCommandBody` in `services/atlas-saga-orchestrator/.../kafka/message/quest/kafka.go` (~line 37). Stay shape-compatible with B1.
- [x] **C2** Add `rewards []quest.ItemReward` parameter to `StartQuestCommandProvider` in `quest/producer.go:13`; set `Body.Rewards`.
- [x] **C3** Add `rewards []quest.ItemReward` parameter to `RequestStartQuest` in `quest/processor.go`; forward to the provider. Mirror `RequestCompleteQuest`.
- [x] **C4** In `saga/handler.go:~1532` `handleStartQuest`, extract `payload.Rewards`, convert to `[]questmessage.ItemReward`, and pass to `RequestStartQuest`. Mirror `handleCompleteQuest` at 1510–1528.

## Phase D — atlas-channel rendering

- [x] **D1** Add `Items []ItemReward` to `QuestStartedEventBody` in `services/atlas-channel/.../kafka/message/quest/kafka.go:94`. Stay shape-compatible with B2.
- [x] **D2** In `kafka/consumer/quest/consumer.go:57` `handleQuestStarted`, extract `e.Body.Items`. Update `announceQuestStarted` (line 74) to accept `items []quest.ItemReward`. After writing the existing status-message packet, write `CharacterQuestEffectBody("", rewards, 0)` when `len(items) > 0` — same call shape as `announceQuestCompleted` at line 117. No foreign-broadcast variant.

## Phase E — atlas-npc-conversations planner

- [x] **E1** In `conversation/operation_executor.go:~795` `createSagaForOperations`, after the `CompleteQuest` rewards-collection pass (lines 812–836), add a symmetric pass writing sibling rewards into each `StartQuest` step's `Rewards` payload field.
- [x] **E2** Add `suppressAwardAssetByStartQuest` next to `suppressAwardAssetByCompleteQuest` (~line 874). For each `StartQuest` step, walk preceding `AwardAsset` steps; fully-covered `(itemId, quantity)` tuples flip `ShowEffect: false` and the reward entry's remaining quantity is decremented. Call `suppressAwardAssetByStartQuest(built)` at line 844 alongside the completion suppressor. Table-driven tests: no `start_quest` → no suppression; matching sibling → suppression; partial-quantity sibling → no suppression; post-`start_quest` `award_item` → no suppression; batch with both `start_quest` and `complete_quest` → independent suppression.

## Phase F — Build sweep & acceptance walk

- [x] **F1** `go build ./...` and `go test ./...` clean across `libs/atlas-saga`, `services/atlas-saga-orchestrator`, `services/atlas-quest`, `services/atlas-channel`, `services/atlas-npc-conversations`. Verify Docker builds for every service consuming `libs/atlas-saga` per CLAUDE.md.
- [ ] **F2** PRD §10 acceptance walk: record evidence against each bullet below.

## PRD §10 Acceptance Criteria Tracking

### Behavioral
- [ ] NPC-started WZ quest with `StartActions.Items` non-empty → item-gain effect renders listing those items.
- [ ] Map-entry auto-start → same effect fires without conversation context.
- [ ] Chained complete → auto-start: completion effect + subsequent start effect both fire; neither suppressed.
- [ ] Sibling `award_item (sword, 1)` + `start_quest (WZ grants sword x1)` → item-gain effect renders exactly once.
- [ ] Override semantic: sibling `award_item (shield, 1)` + `start_quest (WZ grants sword x1)` → inventory gets both; effect renders shield only.
- [ ] Sibling `award_item (potion, 2)` + `start_quest (WZ grants potion x1)` → sibling goes silent, `Items` reports (potion, 2), one effect renders.
- [ ] Re-triggering a quest already in `StateStarted` → no re-render (no `QuestStartedEventBody` emitted).
- [ ] EXP / meso notices on start still fire as today.

### Non-regression
- [ ] Existing quest-completion reward notice behavior (task-014) unchanged.
- [ ] Non-conversation callers of `Start()` / `StartChained()` compile with `nil` `externalRewards`; no behavior diff.
- [ ] `processEndActions` and `QuestCompletedEventBody` untouched.
- [ ] Existing service tests pass. HTTP resource handler at `atlas-quest/.../quest/resource.go` behaviorally equivalent.

### Tests
- [ ] Unit test: `suppressAwardAssetByStartQuest` covering all five enumerated cases in E2.
- [ ] Unit test: sibling-rewards collection pass writes into `StartQuestPayload.Rewards`.
- [ ] Unit test: `Start()` with non-empty `externalRewards` → event body `Items` matches override.
- [ ] Unit test: `Start()` with nil `externalRewards` and non-empty `StartActions.Items` → event body `Items` matches `processStartActions` output.
- [ ] Unit test: `Start()` with nil `externalRewards` and empty `StartActions.Items` → `Items` is empty.
- [ ] Unit test: `StartChained()` paths (same three cases).
- [ ] Mock event emitter updated to capture `items`; existing start-exercising tests assert on it.
- [ ] atlas-saga-orchestrator test: `handleStartQuest` forwards `payload.Rewards` into emitted `StartCommandBody.Rewards`.

### Build
- [ ] `libs/atlas-saga`, `services/atlas-saga-orchestrator`, `services/atlas-quest`, `services/atlas-channel`, `services/atlas-npc-conversations` all build cleanly.
- [ ] All affected services' existing unit and integration tests pass.

## Open Questions to Resolve During Implementation

_None outstanding at spec time (PRD §9). Append any implementation-time questions below._

## Notes / Decisions Log

_(Append dated entries as decisions are made during implementation.)_
