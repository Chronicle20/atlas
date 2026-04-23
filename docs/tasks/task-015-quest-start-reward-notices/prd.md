# Quest Start Reward Notices — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-20
---

## 1. Overview

When a quest is **started**, Atlas may grant items, experience, and mesos from the quest's WZ `StartActions` (e.g., a warrior-advancement quest that hands out a starter weapon, a tutorial quest that prepays a small exp bonus). Today, EXP and mesos fire their normal client-side chat notices (atlas-quest's saga builder hardcodes `ShowEffect: true` on `AddAwardExperience` and `AddAwardMesos`), but **items are silently granted**: `AddAwardItem` leaves `ShowEffect` at its zero value, and `QuestStartedEventBody` carries no item list, so no `CharacterQuestEffectBody` packet is written on the channel side. The player's inventory gains the item with no on-screen acknowledgement, which feels broken — particularly for job-advancement flows where the starter item is the visible payoff.

Task-014 (`docs/tasks/task-014-conversation-reward-notices/`) established the analogous pattern for quest **completion** and for direct conversation rewards: conversation-sibling `award_item` steps feed `CompleteQuestPayload.Rewards`; atlas-saga-orchestrator funnels that list through `CompleteCommandBody.Rewards`; atlas-quest's `Complete()` accepts them as `externalRewards` and reports them on `QuestCompletedEventBody.Items`; atlas-channel's `announceQuestCompleted` renders `CharacterQuestEffectBody` for the items; and the conversation planner's `suppressAwardAssetByCompleteQuest` zeros duplicate sibling `ShowEffect`s so the effect fires exactly once.

This task closes the symmetric gap on the start side. It extends `QuestStartedEventBody` with an `Items` list, threads a `Rewards` field through `StartQuestPayload` → `StartCommandBody` → `Start()` / `StartChained()` as `externalRewards`, renders items on the channel side via the existing `CharacterQuestEffectBody` packet, and adds `suppressAwardAssetByStartQuest` to the conversation planner. No WZ data leaks into atlas-saga-orchestrator — the source of truth for conversation-driven starts is the sibling `award_item` steps, exactly as it is for completion.

## 2. Goals

Primary goals:
- Every quest start that grants items produces a client-visible `CharacterQuestEffectBody` effect listing those items, whether the quest was started via an NPC conversation, a map-entry auto-start, or a chained follow-up from a completed quest.
- Conversation-driven starts let the conversation's sibling `award_item` steps override the WZ `StartActions` list that would otherwise be reported (matching the override semantic completion already has via `externalRewards`).
- A conversation that declares both `award_item` and `start_quest` renders the item-gain effect **exactly once** — not once from the sibling `award_item` and again from the `QuestStartedEventBody.Items` render path.
- Non-conversation callers (auto-start, chain follow-ups, admin tools) keep their current EXP/meso notice behavior and additionally gain the new item notice sourced from WZ `StartActions`.
- Existing JSON conversation scripts require **no changes** to benefit — the new behavior is transparent.

Non-goals:
- No change to EXP/meso notice paths. Those already fire on start via atlas-quest's builder and are working as intended.
- No `silent` opt-out on `start_quest` (parity with `complete_quest`, which also has none).
- No changes to the `QuestCompleted` flow — task-014 already covers it.
- No new Kafka topics. This reuses the existing `EVENT_TOPIC_QUEST_STATUS` topic with an extended event body.
- No client packet additions — reuses the existing `CharacterQuestEffectBody`.
- No new UI in atlas-ui.
- No change to the quest data model, the quest HTTP resource, or quest progress tracking.
- No feature flags or backwards-compatibility shims. The event body gains `Items []ItemReward` with an `omitempty` tag; when empty, older consumers see the same shape they see today.

## 3. User Stories

- As a player completing a warrior first-job advancement, when the warrior trainer starts the quest and hands me a sword, I want to see the quest-style item-gain effect and a chat line listing the sword — matching what I see when I hand in a quest.
- As a player auto-starting a quest on map entry, I want the same item effect to fire if that quest grants a start item, so I understand why an item just appeared in my inventory.
- As a player clearing a chained quest (finishing one, which auto-starts the next), I want both the completion and the subsequent start to render their respective item effects independently. Seeing two effects in close succession is acceptable — that is the correct reflection of what happened.
- As a conversation script author, I want the reward-notice behavior for a sibling `award_item` + `start_quest` pair to work identically to the existing `award_item` + `complete_quest` pair, so I can author start-flow scripts without learning a new rule.
- As a developer on a non-conversation code path (system auto-grant, admin tool), I want the existing `Start(...)` behavior preserved: I don't need to supply rewards, and WZ `StartActions` automatically become the reported items.

## 4. Functional Requirements

### 4.1 `StartQuestPayload.Rewards`

Add a `Rewards []QuestRewardItem` field to `StartQuestPayload` in `libs/atlas-saga/payloads.go` (around line 252). The `QuestRewardItem` type already exists for the completion path (line 247).

Zero-value semantics: `len(Rewards) == 0` means "no conversation-supplied override — atlas-quest should fall back to WZ `StartActions` for reporting." This matches how `CompleteQuestPayload.Rewards` behaves today.

### 4.2 `StartCommandBody.Rewards`

Add a `Rewards []ItemReward` field (JSON: `rewards,omitempty`) to:
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/quest/kafka.go` (`StartCommandBody`, ~line 37)
- `services/atlas-quest/atlas.com/quest/kafka/message/quest/kafka.go` (`StartCommandBody`, ~line 37)

`ItemReward` already exists in both packages. These types must stay shape-compatible on the wire — a single emit from the orchestrator is consumed by atlas-quest.

### 4.3 Orchestrator forwards rewards

`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` `handleStartQuest` (~line 1532) extracts `payload.Rewards`, converts to `[]questmessage.ItemReward`, and passes to a new `RequestStartQuest(..., rewards)` signature. The pattern mirrors `handleCompleteQuest` (lines 1510-1528) exactly.

`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/quest/producer.go` `StartQuestCommandProvider` (line 13) gains a `rewards []quest.ItemReward` parameter and sets `Body.Rewards`.

`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/quest/processor.go`'s `RequestStartQuest` accepts and forwards the rewards slice. The symmetric helper `RequestCompleteQuest` already accepts `rewards` — follow the same signature pattern.

### 4.4 `Start()` and `StartChained()` accept `externalRewards`

`services/atlas-quest/atlas.com/quest/quest/processor.go`:

- `Start(transactionId, characterId, questId, field, skipValidation, externalRewards []questmessage.ItemReward) (Model, nil, nil)` — new final parameter.
- `StartChained(transactionId, characterId, questId, field, externalRewards []questmessage.ItemReward) (Model, error)` — new final parameter.

Inside `Start()` (after line 189's call to `processStartActions`) and `StartChained()` (after line 327's call), introduce the same override shape used by `Complete()`:

```go
reportedItems := awardedItems   // returned by processStartActions
if len(externalRewards) > 0 {
    reportedItems = externalRewards
}
p.eventEmitter.EmitQuestStarted(transactionId, characterId, f.WorldId(), questId, updated.ProgressString(), reportedItems)
```

### 4.5 `processStartActions` returns awarded items

`processStartActions` currently returns `error`. Change its signature to `([]questmessage.ItemReward, error)` and have it emit an `ItemReward{ItemId, Amount: int32(Count)}` entry for each `AddAwardItem` it appends to the saga builder — both the randomly-selected pool winner and each unconditional positive-count item. Items with `Count < 0` (consumed start-requirement items) are not reported; they are not rewards.

This mirrors `processEndActions` (processor.go:796), which already returns `([]questmessage.ItemReward, error)`.

### 4.6 `QuestStartedEventBody.Items`

Add `Items []ItemReward \`json:"items,omitempty"\`` to:
- `services/atlas-quest/atlas.com/quest/kafka/message/quest/kafka.go` `QuestStartedEventBody` (line 100)
- `services/atlas-channel/atlas.com/channel/kafka/message/quest/kafka.go` `QuestStartedEventBody` (line 94)

`EmitQuestStarted` (both the event-emitter interface at processor.go:18 and the Kafka producer at `kafka/producer/quest/producer.go:100`) accepts an additional `items []questmessage.ItemReward` parameter and writes it to the event body.

### 4.7 atlas-channel renders items on quest start

`services/atlas-channel/atlas.com/channel/kafka/consumer/quest/consumer.go`:

- `handleQuestStarted` (line 57) extracts `e.Body.Items` and passes it into the existing `announceQuestStarted` operator.
- `announceQuestStarted` (line 74) is extended to accept `items []quest.ItemReward`. After writing the existing `CharacterStatusMessageOperationUpdateQuestRecordBody` (line 78), if `len(items) > 0` it additionally writes `CharacterQuestEffectBody("", rewards, 0)` — exactly the form used by `announceQuestCompleted` at line 117.

No other packets change. No foreign-broadcast variant is introduced — completion's foreign broadcast is for the `CharacterQuestCompleteEffectForeignBody` packet specifically (a completion animation other players see), and start has no analog today.

### 4.8 Conversation operation_executor changes

`services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go`:

- In `createSagaForOperations` (around line 795), after the existing `CompleteQuest` rewards-collection pass (lines 812-836), add a symmetric pass that writes the same `[]saga.QuestRewardItem` list into any `StartQuest` step payload's new `Rewards` field.
- Add `suppressAwardAssetByStartQuest` next to `suppressAwardAssetByCompleteQuest` (line 874). It follows the same walk: for each `StartQuest` step, inspect `payload.Rewards`; for each preceding `AwardAsset` step whose `(itemId, quantity)` is fully covered by a remaining reward entry, flip `ShowEffect` to `false` and subtract the covered quantity. Post-`StartQuest` `AwardAsset` steps and over-quantity `AwardAsset` steps remain visible.
- Call `suppressAwardAssetByStartQuest(built)` at line 844 alongside the existing `suppressAwardAssetByCompleteQuest(built)` call.

The two suppression helpers should be independent — a conversation batch with both `start_quest` and `complete_quest` should have each pass look only at its own action type.

### 4.9 No `silent` param on `start_quest`

`start_quest` does not read a `silent` operation parameter. This is a deliberate parity decision with `complete_quest` (which also has no `silent`). The conversation author controls visibility by choosing whether to issue sibling `award_item` steps and whether the quest's WZ `StartActions` contain reward items — not via a per-operation flag.

### 4.10 Idempotency on already-started quests

`Start()`'s `startCore` already returns `ErrQuestAlreadyStarted` when the quest is in `StateStarted` and skips `processStartActions` / `EmitQuestStarted` in that case. No change needed — an "already started" early-return path never emits an event with items, so no re-rendering occurs when a player re-triggers an NPC that would start an in-progress quest.

## 5. API Surface

No HTTP/JSON:API changes.

Kafka additions (existing topics, additive body fields):
- `EVENT_TOPIC_QUEST_STATUS` — `QuestStartedEventBody` gains `items []ItemReward` (omitempty).
- `COMMAND_TOPIC_QUEST` — `StartCommandBody` gains `rewards []ItemReward` (omitempty).

Saga payload addition:
- `StartQuestPayload` gains `Rewards []QuestRewardItem` (omitempty).

Conversation script schema: unchanged. `start_quest` continues to accept `questId` and `npcId`. No new params.

## 6. Data Model

No database migrations. All state flows via Kafka events and saga payloads in memory.

All additions are `omitempty`-tagged / zero-value-safe. Existing non-conversation callers (auto-start, chain follow-ups, admin tools that do not supply rewards) see identical behavior on the wire after the change as before — the new fields are absent/empty and atlas-quest falls back to WZ `StartActions` for reporting.

## 7. Service Impact

| Service | Change |
|---|---|
| `libs/atlas-saga` | Add `Rewards []QuestRewardItem` to `StartQuestPayload`. Update any unmarshaller coverage to include the field (zero-value when absent). |
| `services/atlas-saga-orchestrator` | Add `Rewards` to `StartCommandBody`. `StartQuestCommandProvider` accepts `rewards`. `handleStartQuest` extracts `payload.Rewards`, converts to `[]questmessage.ItemReward`, and forwards via `RequestStartQuest`. |
| `services/atlas-quest` | Add `Rewards` to `StartCommandBody`. `handleStartQuestCommand` passes `c.Body.Rewards` into `Start()`. Add `externalRewards` parameter to `Start()` and `StartChained()`. Change `processStartActions` return type to `([]questmessage.ItemReward, error)` and populate items from awarded `AddAwardItem` calls. Extend `EmitQuestStarted` to accept and send an `items` slice. Extend `QuestStartedEventBody` with `Items []ItemReward`. Update internal callers of `Start` (`resource.go`, any in-process callers) to pass `nil` for `externalRewards`. Update `EventEmitter` interface + mock. |
| `services/atlas-channel` | Extend `QuestStartedEventBody` with `Items []ItemReward`. `handleQuestStarted` forwards items to `announceQuestStarted`, which writes `CharacterQuestEffectBody` when `len(items) > 0`. |
| `services/atlas-npc-conversations` | In `createSagaForOperations`, collect sibling `AwardAsset` items into each `StartQuest` step's `Rewards` payload (mirror of the existing completion pass). Add `suppressAwardAssetByStartQuest` and invoke alongside `suppressAwardAssetByCompleteQuest`. |

No changes to atlas-character, atlas-ui, or any other service.

## 8. Non-Functional Requirements

**Performance:** One additional `CharacterQuestEffectBody` packet per quest start that has items, routed only to the starting character's session. Quest starts are interactive and low-throughput. No additional DB reads. No extra Kafka round-trips — the existing `QuestStartedEventBody` already fires; this task only widens its body.

**Backwards compatibility:**
- All new wire fields are `omitempty`/zero-value-safe. A consumer running the old schema against a new producer sees an extra JSON field it ignores; a new consumer against an old producer sees an empty `Items` slice and takes the WZ fallback path.
- Non-conversation callers supply `nil` for `externalRewards` and observe the same behavior they do today (WZ `StartActions` drive the reported items).
- Existing conversation scripts get the new behavior transparently; no script-author action is required.

**Multi-tenancy:** No change to tenancy handling. All new Kafka fields ride inside existing tenant-scoped events and commands.

**Observability:** Existing logging in `handleQuestStarted`, `Start()`, and `processStartActions` is preserved. Log the `len(items)` at debug level on emission for parity with how completion is logged today (if completion logs the reward count — check during implementation and match).

**Security:** No new external surface. All new fields are on internal Kafka topics.

**Testing:** See §10.

## 9. Open Questions

None outstanding at spec time — the four open items from the design conversation (auto-start/chained render behavior, `silent` parity, event body shape, `Start()` signature change) are resolved in §4.1–§4.9. Carry forward any implementation-time questions into `plan.md` once `/dev-docs` is run.

## 10. Acceptance Criteria

### Behavioral

- [ ] Starting a WZ-defined quest whose `StartActions.Items` is non-empty via an NPC conversation renders the quest-style item-gain effect listing those items.
- [ ] Starting a WZ-defined quest via map-entry auto-start renders the same effect (no conversation context required).
- [ ] Completing a quest that triggers a chained follow-up renders the completion effect for the finished quest AND the start effect for the next quest in the chain. Both fire; neither is suppressed.
- [ ] An NPC conversation batch containing `award_item` (sword, 1) followed by `start_quest` (questId whose WZ `StartActions` also grants sword x1) renders the item-gain effect exactly once — not twice.
- [ ] **Override semantic (mirrors completion).** When the conversation supplies any sibling `award_item`, `externalRewards` is non-nil and is authoritative for `QuestStartedEventBody.Items`. Any items granted by atlas-quest's own `processStartActions` saga in that same call still land in the inventory (the saga runs regardless), but they do not appear in the reported `Items` and therefore render silently. Example: `award_item (shield, 1)` + `start_quest (WZ grants sword x1)` → inventory gets shield and sword; the effect renders shield only. If an author wants both to render, they must declare `award_item (sword, 1)` explicitly as a second sibling.

- [ ] An NPC conversation batch containing `award_item` (potion, 2) followed by `start_quest` (questId whose WZ `StartActions` grants potion x1) suppresses the sibling's `ShowEffect` only if the sibling's quantity is fully covered. Since the sibling wants quantity 2 and the conversation-supplied `Rewards` list also carries (potion, 2) because it was built from the sibling itself, the suppression is tautological — the sibling's `AwardAsset` goes silent, `QuestStartedEventBody.Items` reports (potion, 2), one effect renders. This is the common case.
- [ ] A quest already in `StateStarted` that is re-triggered (e.g., NPC is re-talked-to mid-quest) does not re-render the start effect (no `QuestStartedEventBody` is emitted in that path today).
- [ ] EXP and meso chat notices on quest start continue to fire as they do today (atlas-quest's saga builder unchanged for those steps).

### Non-regression

- [ ] Existing quest-completion reward notice behavior (task-014) is unchanged.
- [ ] Existing non-conversation callers of `Start()` / `StartChained()` compile and run with `nil` `externalRewards`; no behavioral diff.
- [ ] `processEndActions` and `QuestCompletedEventBody` are untouched.
- [ ] All existing service tests pass. The HTTP resource handler in `services/atlas-quest/atlas.com/quest/quest/resource.go` still works (updated for the new signature but behaviorally equivalent).

### Tests

- [ ] Unit test in `atlas-npc-conversations` covering `suppressAwardAssetByStartQuest`: no `start_quest` → no suppression; `start_quest` with matching sibling → suppression; `start_quest` with quantity-mismatch sibling → no suppression; `award_item` following `start_quest` → no suppression; batch with both `start_quest` and `complete_quest` → independent suppression.
- [ ] Unit test in `atlas-npc-conversations` covering the sibling-rewards collection pass writing into `StartQuestPayload.Rewards`.
- [ ] Unit test in `atlas-quest` covering `Start()` with non-empty `externalRewards` → event body `Items` matches override.
- [ ] Unit test in `atlas-quest` covering `Start()` with nil `externalRewards` and non-empty `StartActions.Items` → event body `Items` matches what `processStartActions` awarded.
- [ ] Unit test in `atlas-quest` covering `Start()` with nil `externalRewards` and empty `StartActions.Items` → event body `Items` is empty.
- [ ] Unit test in `atlas-quest` covering `StartChained()` paths (same three cases).
- [ ] Mock event emitter updated to capture `items` on `EmitQuestStarted` and assertions added in existing tests that exercise start.
- [ ] atlas-saga-orchestrator test covering `handleStartQuest` forwarding `payload.Rewards` into the emitted `StartCommandBody.Rewards`.

### Build

- [ ] `libs/atlas-saga`, `services/atlas-saga-orchestrator`, `services/atlas-quest`, `services/atlas-channel`, `services/atlas-npc-conversations` all build cleanly.
- [ ] All affected services' existing unit and integration tests pass.
