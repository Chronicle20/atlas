# Saga Step–Event Action Matching — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-24
---

## 1. Overview

Atlas uses a saga pattern (implemented in `services/atlas-saga-orchestrator`) to coordinate multi-step, cross-service flows such as NPC first-job advancement, quest completion, inventory grants, storage moves, and character creation. Each saga is a linearly ordered list of steps, each with an `Action` (e.g., `award_asset`, `change_job`, `rebalance_ap`, `complete_quest`). The orchestrator emits commands to downstream services for the currently pending step and waits for a corresponding domain event on a status topic (e.g., `EVENT_TOPIC_CHARACTER_STATUS`, `EVENT_TOPIC_ASSET_STATUS`, `EVENT_TOPIC_QUEST_STATUS`). The consumer for each status topic calls `Processor.StepCompleted(transactionId, success)` to advance the saga.

**The bug:** every consumer handler blindly calls `StepCompleted` whenever it sees an event tagged with a saga's `transactionId`, regardless of whether the event type is semantically related to the currently pending step's action. Domain operations frequently emit more than one event per logical action — for example, `change_job` on atlas-character triggers a `JOB_CHANGED` event followed by one or more `STAT_CHANGED` ripple events (to reflect AVAILABLE_AP, AVAILABLE_SP, HP/MAX_HP, MP/MAX_MP changes that the job change cascades). Each of those ripple events matches the saga's transaction id, so each one advances the saga by one step. The next legitimate steps (typically `award_asset` steps waiting on `ASSET_CREATED`) get silently marked complete by the ripples, and their real events later arrive to a saga with no pending steps ("No current step found for asset created event").

**The user-visible regression that motivated this task:** during Thief first-job advancement, a new character receives three reward items (throwing stars, claw, dagger). The expected client behaviour is three `conversation_reward_notice` events, one per item, which the v83 client renders as three "You have obtained …" chat lines. What actually happens today:

- Two of the three reward notices are silently dropped because the corresponding `award_asset` steps were already marked complete by change_job's ripple `STAT_CHANGED` events.
- The one notice that does fire carries the templateId of the *last* award_asset step's payload, not the templateId of the first asset actually created — so the player sees a chat line for the wrong item (the dagger, which they didn't get first).

Kubernetes log evidence is captured in §9.1. The same bug affects any saga whose earlier step emits multiple downstream events (confirmed for `change_job`; suspected for any action that cascades through HP/MP/level recomputation). This task eliminates the class of bug, not just the specific symptom.

## 2. Goals

Primary goals:

- Every event consumer in `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/*` refuses to mark a pending saga step complete unless the event semantically matches the step's `Action`. Unrelated ripple events are silently ignored (debug-logged).
- `award_asset` / `create_and_equip_asset` step completion additionally requires the incoming `ASSET_CREATED` (or `ASSET_QUANTITY_CHANGED`) event's `templateId` to match the step payload's `TemplateId`. Same requirement on `destroy_asset` for the matching destroy events.
- `conversation_reward_notice` emissions use the **incoming event's** `templateId` and `quantity`, not the current step's payload, so the player-visible chat line always reflects the item actually created/destroyed.
- All failure-path handlers (e.g., `handleCharacterCreationFailedEvent`, `handleCharacterMesoErrorEvent`, asset-created-validation failures) also gate their `StepCompleted(..., false)` call on the pending step's action, so a stray failure signal from one subsystem can't compensate an unrelated saga step.
- A shared helper in the `saga` package (e.g., `StepAcceptsEvent(step Step[any], eventKind EventKind) bool`) centralises the action→event-kind whitelist so the mapping is declared in one place and enforced uniformly across all 11 consumer packages.
- The existing Thief advancement scenario (and by extension all five Explorer NPC advancements and all five Cygnus quest advancements from task-020) produces exactly three `conversation_reward_notice` events with the correct templateIds when advancing the Thief flow. Verified by an integration test that replays the Kafka event sequence.
- No script, payload, or Kafka message schema changes. Producers emit the same events they always have; the fix lives entirely inside atlas-saga-orchestrator.

Non-goals:

- No step-ID correlation on Kafka events. We continue to correlate by `transactionId` only; the new layer is per-action acceptance filtering.
- No collapsing of atlas-character's multiple STAT_CHANGED ripples into a single bundled event. That is a separate concern in the character service and is not necessary once the orchestrator correctly ignores unrelated events.
- No new Kafka topics, message bodies, or JSON-API endpoints.
- No changes to NPC conversation scripts, quest scripts, or `libs/atlas-saga` payload shapes.
- No changes to saga persistence, the cache, `SagaTimers`, or the compensation algorithm.
- No new UI in atlas-ui.
- No new observability platform work (no Prometheus, Grafana, or OpenTelemetry integration). Debug/info-level logs only.
- No change to the task-020 operation ordering (`rebalance_ap` → `change_job` → `award_item*`). That ordering is correct and remains.

## 3. User Stories

- As a player completing first-job advancement, when the NPC awards me multiple items, I want to see one "You have obtained …" chat line per item with the correct item name, so the UI reflects what actually landed in my inventory.
- As a conversation script author, I want to trust that adding an `award_item` step produces exactly one user-visible notice when the item is created, even if earlier steps in the same saga emit ripple events on unrelated topics.
- As a developer extending the saga system with a new action, I want one place to declare which event types can complete that action, rather than hunting through eleven consumer packages for the right handler to update.
- As an operator reading saga-orchestrator logs, I want ripple/ignored events to be visible at `debug` level so I can confirm they were intentionally skipped, and I want a `warn` log if an event arrives that doesn't match *any* action in the saga (i.e., a true unexpected event, as distinct from a well-understood ripple).
- As a developer investigating a saga that hung mid-flow, I want the per-consumer action whitelist to fail safely — an event that should have completed the step either completes it or logs a clear skip; it never silently drops the saga into an inconsistent state.

## 4. Functional Requirements

### 4.1 Shared acceptance helper

A new function lives in the `saga` package (existing Go package at `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/`):

```go
// EventKind is a compact tag for the semantic class of an event received on
// a status topic. It is not the Kafka event body type — it is a per-handler
// classification chosen by the consumer package that received the event.
type EventKind string

// StepAcceptsEvent reports whether a saga step's Action can be legitimately
// completed (or failed) by an event of the given EventKind. Unknown actions
// or kinds return false so new actions default to a safe "no match" posture
// until the mapping is added.
func StepAcceptsEvent(action Action, kind EventKind) bool
```

The function is backed by a single declarative whitelist (e.g., a `map[Action]map[EventKind]bool` or a generated switch). The whitelist is the source of truth for which events legitimately complete which steps.

Mandatory EventKind constants introduced by this task (at minimum — the full list is expanded in §7 per consumer):

- `EventKindCharacterMapChanged`, `EventKindCharacterExperienceChanged`, `EventKindCharacterLevelChanged`, `EventKindCharacterMesoChanged`, `EventKindCharacterJobChanged`, `EventKindCharacterCreated`, `EventKindCharacterCreationFailed`, `EventKindCharacterStatChanged`, `EventKindCharacterMesoError`, `EventKindCharacterDeleted`.
- `EventKindAssetCreated`, `EventKindAssetDeleted`, `EventKindAssetQuantityChanged`, `EventKindAssetMoved`.
- `EventKindQuestStarted`, `EventKindQuestCompleted`, `EventKindQuestForfeited`, `EventKindQuestProgressSet`.
- `EventKindCompartment*` for the compartment consumer's existing sub-event types.
- `EventKindSkill*`, `EventKindGuild*`, `EventKindStorage*`, `EventKindBuddy*`, `EventKindCashShop*`, `EventKindConsumable*`, `EventKindPet*`, `EventKindInvite*` for their respective consumers.

### 4.2 Per-consumer gating

Every consumer handler in `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/*` that today unconditionally calls `StepCompleted` (or `StepCompletedWithResult`) MUST be rewritten to:

1. Load the saga via `saga.NewProcessor(l, ctx).GetById(transactionId)`. If the saga is not found, keep the current debug-log + silent return behaviour (this path already exists; do not regress it).
2. Fetch the current pending step via `s.GetCurrentStep()`. If there is no pending step, the event is a late/duplicate signal — debug-log and return (existing behaviour, preserve it).
3. Classify the incoming event as an `EventKind` (per §4.1). Call `saga.StepAcceptsEvent(currentStep.Action(), kind)`.
4. If `StepAcceptsEvent` returns `false`:
   - Debug-log the skipped event with fields: `transaction_id`, `tenant_id`, `step_id`, `step_action`, `event_kind`, and any handler-specific payload summary (e.g., templateId for asset events).
   - Return without calling `StepCompleted`.
5. If `StepAcceptsEvent` returns `true`, proceed with the existing `StepCompleted` / `StepCompletedWithResult` call.

Failure-path handlers (any consumer that calls `StepCompleted(..., false)`) apply the same gate: the failure only compensates the current step if the event kind is in the step's whitelist. A failure event that doesn't match is likewise debug-logged and skipped.

### 4.3 Asset templateId guard

`AwardAsset` and `CreateAndEquipAsset` step completion additionally requires:

- The incoming `ASSET_CREATED` event's `TemplateId` to equal the step's payload `Item.TemplateId` (field name per `saga.AwardItemActionPayload` / `saga.CreateAndEquipAssetPayload`).
- Or, for the `ASSET_QUANTITY_CHANGED` completion path used when a stackable item merges into an existing stack, the event's `TemplateId` to equal the step's payload `Item.TemplateId`.

If the kind matches but the templateId does not, the handler debug-logs the mismatch (including expected and actual templateIds) and returns without completing the step. The saga then waits for the correct event.

`DestroyAsset` and `DestroyAssetFromSlot` completion on `ASSET_DELETED` or `ASSET_QUANTITY_CHANGED` applies the symmetric guard against the step's payload `TemplateId`.

### 4.4 Conversation reward notice correction

The helper `emitRewardNoticeForCurrentStep` in `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/asset/consumer.go` currently emits a notice using `payload.Item.TemplateId` / `payload.Item.Quantity` from the current pending step. After this task:

- The templateId used in the notice MUST come from the incoming event (already passed to the helper via the `templateId` parameter), not from the step payload.
- The quantity used in the notice:
  - For `ASSET_CREATED`: comes from the event body (e.g., `e.Body.Quantity`).
  - For `ASSET_QUANTITY_CHANGED`: comes from the event body (the delta, not the post-change absolute).
  - For `ASSET_DELETED`: comes from the event body (the destroyed amount).
  - For `DestroyAssetFromSlot` (where the step payload has the quantity and the event does not carry a distinct delta), the step payload's quantity is still acceptable; document this exception.
- The `ShowEffect` boolean is still read from the step payload (this is the author's opt-out per task-014).
- The character id is still read from the step payload.

Once §4.2 and §4.3 are in place, the current step's `TemplateId` will equal the event's `TemplateId` whenever a notice fires. The switch to "read from event" is therefore cosmetically identical in the common case; the point of the change is correctness: if the fix in §4.2/§4.3 ever regresses, the notice payload stays honest.

### 4.5 Observability

All debug/warn logs emitted by this feature use structured fields (per existing `logrus.Fields` usage):

- `transaction_id`, `tenant_id` — always.
- `step_id`, `step_action` — whenever the current step is known.
- `event_kind` — always (so log filters can count per-kind skips).
- `event_template_id`, `expected_template_id` — on asset handlers that skip for templateId mismatch.
- `reason` — a short free-form tag (e.g., `"action_mismatch"`, `"template_id_mismatch"`, `"no_pending_step"`, `"saga_not_found"`) so log analysis can aggregate.

A `warn`-level log fires once per saga transaction if an incoming event's `transactionId` matches a known saga, but the event's kind is not in **any** remaining pending step's acceptance set. This catches the "genuinely unexpected event" case (e.g., a malformed payload, a new producer emitting an unknown event type) without firing on every legitimate ripple.

### 4.6 Existing idempotency guards preserved

The existing guards in `Processor.stepCompletedWithResultOnce` (duplicate-completion ignore when `FindEarliestPendingStepIndex() == -1`, terminal-state guard for compensation) remain unchanged. This task narrows the set of events that reach `StepCompleted`; it does not alter what `StepCompleted` does once called.

## 5. API Surface

No new HTTP endpoints. No new Kafka topics. No new Kafka message bodies.

The only new Go public surface is within the `saga` package:

```go
type EventKind string

const (
    EventKindCharacterMapChanged EventKind = "character.map_changed"
    EventKindCharacterStatChanged EventKind = "character.stat_changed"
    // ...one constant per event kind, organised by subsystem
)

func StepAcceptsEvent(action Action, kind EventKind) bool
```

The helper is called from each consumer package; consumer packages otherwise keep their existing function signatures.

## 6. Data Model

No schema changes. No migrations. No tenant-model changes. The saga persistence layer (Postgres via `SagaStore`) is untouched — only the in-flight step-dispatch logic changes.

## 7. Service Impact

Only **`services/atlas-saga-orchestrator`** changes. Files that must change:

| File | Change |
|---|---|
| `saga/event_acceptance.go` (new) | Declares `EventKind` constants, the acceptance whitelist, and `StepAcceptsEvent`. |
| `saga/event_acceptance_test.go` (new) | Table-driven tests covering every (Action, EventKind) pair in the whitelist, plus a lexicographic "every Action is represented" check. |
| `kafka/consumer/asset/consumer.go` | All four handlers (`handleAssetCreatedEvent`, `handleAssetDeletedEvent`, `handleAssetQuantityUpdatedEvent`, `handleAssetMovedEvent`) gated on `StepAcceptsEvent`. Template-id guard added. `emitRewardNoticeForCurrentStep` updated per §4.4. |
| `kafka/consumer/asset/consumer_test.go` (extend or create) | Unit tests covering each handler × each relevant step action, including the templateId-mismatch case. |
| `kafka/consumer/character/consumer.go` | Ten handlers (`handleCharacterMapChangedEvent`, `handleCharacterExperienceChangedEvent`, `handleCharacterLevelChangedEvent`, `handleCharacterMesoChangedEvent`, `handleCharacterJobChangedEvent`, `handleCharacterCreatedEvent`, `handleCharacterCreationFailedEvent`, `handleCharacterMesoErrorEvent`, `handleCharacterStatChangedEvent`, `handleCharacterDeletedEvent`) gated. |
| `kafka/consumer/character/consumer_test.go` (extend or create) | Unit tests per handler, plus the motivating scenario: a sequence of `JOB_CHANGED` + multiple `STAT_CHANGED` + `ASSET_CREATED*3` against a 5-step saga must complete exactly the right steps. |
| `kafka/consumer/buddylist/consumer.go` | Single handler gated. |
| `kafka/consumer/cashshop/consumer.go` | Single handler gated. |
| `kafka/consumer/compartment/consumer.go` | All handlers gated (lines 58, 75, 82, 97, 119, 140 per the inventory scan). |
| `kafka/consumer/consumable/consumer.go` | Single handler gated. |
| `kafka/consumer/guild/consumer.go` | Five handlers gated. |
| `kafka/consumer/invite/consumer.go` | Three handlers gated. |
| `kafka/consumer/pet/consumer.go` | Single handler gated. |
| `kafka/consumer/quest/consumer.go` | Three handlers gated (`handleQuestStarted`, `handleQuestCompleted`, `handleQuestForfeited` or equivalents — confirm names during implementation). |
| `kafka/consumer/skill/consumer.go` | Handlers gated. |
| `kafka/consumer/storage/consumer.go` | Handlers gated. |
| `kafka/consumer/*/consumer_test.go` | Unit tests per consumer. Small table-driven tests are acceptable — the interesting assertions are "no call when action doesn't match" and "call when it does". |

No changes to `libs/atlas-saga`, any other service, any JSON script, or any client.

The fix is entirely internal to atlas-saga-orchestrator, which means **no rolling-deploy compatibility risk** — sagas in-flight during the deploy resume on the new pod with strict matching. Any saga whose next expected event was in the to-be-gated "blind complete" path still completes when the correct event arrives.

## 8. Non-Functional Requirements

**Performance.** Each event now reads the saga from cache to classify the pending step. `saga.GetById` + `GetCurrentStep` is already called in several handlers; we are extending coverage, not adding fundamentally new work. The additional in-memory map lookup in `StepAcceptsEvent` is O(1). No measurable latency impact expected.

**Correctness under concurrency.** The existing `stepCompletedWithResultOnce` version-conflict retry loop (processor.go:316–330) handles concurrent completion attempts. Our new gate runs *before* `StepCompleted` is invoked, so we don't change concurrent-update semantics — we just reduce the number of racing completion attempts in the first place, which if anything reduces version-conflict retry pressure.

**Observability.** Debug logs are low-volume (saga flows are human-triggered, typically 1–10 events per minute per tenant). The new `warn` for "event matches no pending step action" is deliberately bounded — once per saga transaction — to avoid log-spam on a single misbehaving producer.

**Multi-tenancy.** All existing handlers already use `tenant.MustFromContext(ctx)` indirectly via `saga.NewProcessor`. No changes to tenant scoping; the whitelist is global, not per-tenant.

**Security.** No security surface. No authn/authz changes. No PII touched.

**Backward compatibility.** Sagas persisted in Postgres from before this change resume with the new matching. Since the fix only *tightens* acceptance (an event that previously completed the wrong step will now be ignored), the worst case is that a pre-fix in-flight saga waits longer for its correct event — which is the correct behaviour.

## 9. Open Questions

### 9.1 Evidence summary for the PRD record

The motivating incident, captured from atlas-saga-orchestrator pod `atlas-saga-orchestrator-5bc77c67c-9mlk6` on 2026-04-24T17:07 for transaction `98fa5263-feed-4b64-b30f-455aab6a4327` (character 12, Thief advancement):

1. Saga received with 5 steps: `rebalance_ap-12`, `change_job-12`, `award_item-12` (template 2070015), `award_item-12-3` (template 1472061), `award_item-12-4` (template 1332063).
2. STAT_CHANGED (rebalance) at 17:07:00.350 → `rebalance_ap-12` marked completed ✓.
3. JOB_CHANGED at 17:07:00.488 → `change_job-12` marked completed ✓.
4. STAT_CHANGED (updates: `[JOB]`) at 17:07:00.619 → **`award_item-12` falsely marked completed** ❌.
5. STAT_CHANGED (updates: `[AVAILABLE_AP, AVAILABLE_SP, HP, MAX_HP, MP, MAX_MP]`) at 17:07:00.722 → **`award_item-12-3` falsely marked completed** ❌.
6. ASSET_CREATED (2070015) at 17:07:00.780 → `emitRewardNoticeForCurrentStep` read the *current pending step*'s payload (`award_item-12-4`, templateId 1332063) and emitted a notice with the wrong templateId. Step `award_item-12-4` marked completed ✓ (correct step, wrong reason).
7. ASSET_CREATED (1472061) at 17:07:00.956 → "No current step found for asset created event." — no notice.
8. ASSET_CREATED (1332063) at 17:07:00.967 → "No current step found for asset created event." — no notice.

atlas-channel received exactly one `conversation_reward_notice` for character 12, item 1332063 (at 17:07:00.844), consistent with step 6 above.

### 9.2 Open questions flagged during spec review

None outstanding. All architectural questions were resolved during interactive scoping:

- Audit scope: full sweep of all 11 consumer packages.
- Asset templateId guard: yes (§4.3).
- Unmatched-event behaviour: debug-log per-event, warn-log once-per-saga if the event kind matches no pending step action (§4.5).
- Multi-kind completion: 1:1 by default except for `AwardAsset` / `DestroyAsset` / `DestroyAssetFromSlot` which accept both `ASSET_CREATED`/`ASSET_DELETED` and `ASSET_QUANTITY_CHANGED` (stackable-merge case).
- Failure-path gating: yes, failure events are also gated (§4.2 final paragraph).
- NPC ordering from task-020 stays (rebalance_ap → change_job → award_items).
- Test strategy: unit tests per consumer plus an integration test replaying the Thief scenario.

## 10. Acceptance Criteria

The feature is considered complete when all of the following hold:

1. **Shared helper exists.** `saga.StepAcceptsEvent(action, kind)` is callable from every consumer package and is backed by a declarative whitelist. Every `Action` declared in `libs/atlas-saga/model.go` is represented in the whitelist (covered by a test that iterates over the `Action` constants).

2. **Every consumer handler is gated.** `grep -rn "StepCompleted\|StepCompletedWithResult" services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/` shows that every call site is preceded by a `StepAcceptsEvent`-based gate (or equivalent). No handler calls `StepCompleted` unconditionally on a saga-tagged event.

3. **Asset templateId guard enforced.** A unit test confirms: given a saga whose current pending step is `award_asset { templateId: 2070015 }`, an `ASSET_CREATED { templateId: 1472061 }` event does NOT mark the step complete and instead logs a templateId-mismatch skip.

4. **Conversation reward notice templateId sourced from event.** A unit test confirms that `emitRewardNoticeForCurrentStep` produces a notice whose templateId matches the incoming event's templateId, not the step payload's, even if they differ (which after §4.3 they cannot in practice — the test is a safety net).

5. **Thief advancement integration test.** An integration test (in `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/` or a new integration-level test package) replays the exact event sequence from §9.1 against a 5-step saga and asserts:
   - All five steps complete in order, each by the correct event.
   - No step is completed by a ripple STAT_CHANGED.
   - Exactly three `conversation_reward_notice` emissions are produced, with templateIds `{2070015, 1472061, 1332063}` in that order.
   - No "No current step found for asset created event" debug log is emitted.

6. **No regressions.** `go build ./...` and `go test ./...` pass for atlas-saga-orchestrator. The existing saga/ package tests (processor, builder, handler, model, integration) still pass. Every other service in the monorepo builds and tests unchanged.

7. **No script or payload changes.** `git diff main -- libs/atlas-saga services/atlas-character services/atlas-npc-conversations/conversations` shows no files changed by this task.

8. **Observability sanity check.** A manual kubectl log inspection during a Thief advancement test shows:
   - Exactly two debug "action_mismatch" skips at the orchestrator (for the two ripple STAT_CHANGED events after `change_job`).
   - Three "action_mismatch" or similar skips are absent — every ASSET_CREATED matches.
   - Zero warn-level "event matches no pending step action" logs during the flow.
   - atlas-channel logs three `EVENT_TOPIC_CONVERSATION_REWARD_NOTICE` receipts in the correct order.
