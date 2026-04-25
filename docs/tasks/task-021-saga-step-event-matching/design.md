# Saga Step–Event Action Matching — Design

Status: Draft
Created: 2026-04-24
Companion PRD: [`prd.md`](./prd.md)

---

## 1. Purpose and scope

This document records the architecture for the fix described in `prd.md`. Requirements,
motivation, acceptance criteria, and the §9.1 evidence log live in the PRD; this design
commits to *how* the fix is shaped in code. The only service touched is
`services/atlas-saga-orchestrator`.

## 2. Architectural choices

Four decisions taken during brainstorming, with the rejected alternatives recorded so
the plan phase knows what not to revisit.

### 2.1 Acceptance decision on the saga processor

**Chosen:** a new gate method on `saga.Processor`:

```go
// AcceptEvent is the single point at which a saga-tagged Kafka event is matched
// against the saga's pending step. It loads the saga, finds the current step,
// consults the acceptance table, and returns the step for payload-specific work
// on success. On any skip path, it debug-logs with a structured reason field
// and returns ok=false.
func (p *ProcessorImpl) AcceptEvent(transactionId uuid.UUID, kind EventKind) (AcceptDecision, bool)

type AcceptDecision struct {
    Saga Saga
    Step Step[any]
}
```

Consumer handlers become: check `e.Type`, call `AcceptEvent`, bail on `!ok`, do
handler-specific work (templateId guard, reward-notice pre-emit, CreateAndEquipAsset
follow-up step) using `decision.Step`, then call `StepCompleted` /
`StepCompletedWithResult`.

**Rejected alternatives:**

- *Plain predicate* (`saga.StepAcceptsEvent(action, kind) bool` per PRD §4.1
  literal): every handler would re-implement the saga-lookup + pending-step
  + structured-log boilerplate. The §4.5 `reason` tags (`action_mismatch`,
  `no_pending_step`, `saga_not_found`, `template_id_mismatch`) would drift across
  files. Rejected for log-discipline reasons.
- *Full wrapper* (`p.CompleteStepIfMatches(tx, kind, result)`): collapses everything
  into one call, but the asset handler emits reward notices *before*
  `StepCompleted` (so the pending step payload is still observable), which would
  force a callback or pre-hook into the wrapper. Rejected for being awkward on the
  most important handler.

Note: the PRD's `StepAcceptsEvent(action, kind) bool` predicate is still implemented
— it is the core of the acceptance table — but consumer handlers call `AcceptEvent`
rather than `StepAcceptsEvent` directly. `StepAcceptsEvent` remains exported so the
table is testable in isolation (§5.2).

### 2.2 Declarative acceptance whitelist as a map literal

**Chosen:** `map[Action][]EventKind`, one entry per `Action`, in
`saga/event_acceptance.go`:

```go
var acceptanceTable = map[Action][]EventKind{
    RebalanceAP:           {EventKindCharacterStatChanged},
    ChangeJob:             {EventKindCharacterJobChanged},
    AwardAsset:            {EventKindAssetCreated, EventKindAssetQuantityChanged},
    DestroyAsset:          {EventKindAssetDeleted, EventKindAssetQuantityChanged},
    DestroyAssetFromSlot:  {EventKindAssetDeleted, EventKindAssetQuantityChanged},
    CreateAndEquipAsset:   {EventKindAssetCreated},
    CreateCharacter:       {EventKindCharacterCreated, EventKindCharacterCreationFailed},
    AwardMesos:            {EventKindCharacterMesoChanged, EventKindCharacterMesoError},
    // ... full entry per Action constant ...
    WarpToPortal:          {},   // self-completing, no event accepts
    UiLock:                {},   // self-completing
    // ...
}

func StepAcceptsEvent(action Action, kind EventKind) bool {
    kinds, ok := acceptanceTable[action]
    if !ok {
        return false // default-deny for unknown actions
    }
    for _, k := range kinds {
        if k == kind {
            return true
        }
    }
    return false
}
```

**Key rules:**

- **One entry per `Action` constant** declared in `libs/atlas-saga`. Self-completing
  actions (fire-and-forget, like `WarpToPortal`, `UiLock`, `ShowInfo`,
  `PlayPortalSound`, `SendMessage`) have explicit empty entries.
  This is enforced by a coverage test (§5.2) that iterates the full Action set and
  asserts a table entry exists.
- **Success and failure kinds share an entry.** An action like `CreateCharacter`
  accepts both `EventKindCharacterCreated` (the success handler's kind) and
  `EventKindCharacterCreationFailed` (the failure handler's kind). The handler
  itself decides whether to call `StepCompleted(true)` or `StepCompleted(false)` —
  the gate only decides whether to call it at all.
- **Default-deny** on unknown `(action, kind)` pairs. A new action added without a
  table entry completes nothing until the mapping is declared.

**Rejected alternatives:**

- Nested map (`map[Action]map[EventKind]struct{}`): O(1) lookup instead of slice
  scan, but at this scale (≤5 kinds per action) the constant-factor difference is
  noise. Rejected for readability — the slice form is easier to diff in a PR.
- Generated switch statement: zero-allocation but hard to iterate for the
  every-action-represented test. Rejected for the iteration story alone.

### 2.3 Hardcoded `EventKind` per consumer handler function

**Chosen:** each handler hardcodes one `saga.EventKind` constant. No runtime
classification layer. The handler function's identity is the classification.

```go
func handleCharacterStatChangedEvent(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[...]) {
    if e.Type != character2.StatusEventTypeStatChanged {
        return
    }
    p := saga.NewProcessor(l, ctx)
    decision, ok := p.AcceptEvent(e.TransactionId, saga.EventKindCharacterStatChanged)
    if !ok {
        return
    }
    _ = decision // handler-specific work if any
    _ = p.StepCompleted(e.TransactionId, true)
}
```

This matches the existing 1:1 shape — each consumer already has one handler
function per wire event type (the `if e.Type != ...` guard at the top of each
handler proves this).

**Rejected alternative:** a `classify(e StatusEvent[...]) EventKind` function per
consumer package. Not harmful, but the 1:1 shape doesn't need it; an extra function
call per event is pure boilerplate.

### 2.4 In-process handler-level integration test

**Chosen:** replay the §9.1 event sequence by calling each
`kafka/consumer/*.handle*Event` function *directly* in the order from the evidence
log, against a real saga in the cache. The events are Go structs, not Kafka wire
bytes. Reward-notice emissions are captured via a test-installable stub.

**Rejected alternatives:**

- *Real Kafka with a testcontainer:* end-to-end fidelity, but none of the bug class
  this task fixes is in the wire layer. Rejected for being disproportionate to the
  bug.
- *Unit-tests-only, no dedicated integration test:* the bug signature is
  specifically an ordering issue across multiple handlers touching the same saga.
  Unit tests don't catch re-ordering if someone flips a switch elsewhere. Rejected
  for losing the whole-scenario regression guard that §10.5 requires.

## 3. Components and files

### 3.1 New files

**`saga/event_acceptance.go`**

- `type EventKind string` — a compact tag for the semantic class of an event.
- `const EventKind* = "..."` — one constant per event-class the consumers emit.
  Grouped by subsystem (character, asset, quest, compartment, skill, guild, invite,
  buddy, cashshop, consumable, pet, storage, saga). Minimum set per PRD §4.1;
  full list enumerated during implementation.
- `var acceptanceTable map[Action][]EventKind` — the whitelist; one entry per
  Action.
- `func StepAcceptsEvent(action Action, kind EventKind) bool` — the predicate.
  Exported for test use; called in production only via `AcceptEvent`.
- Unexported `skipReason*` constants: `action_mismatch`, `template_id_mismatch`,
  `no_pending_step`, `saga_not_found`. These are the `reason` field values in
  structured skip logs. Centralised so §4.5 log-tag discipline cannot drift.
- Unexported `logSkip(l logrus.FieldLogger, fields logrus.Fields, reason string)`
  helper that debug-logs with a `reason` field set — used by `AcceptEvent` and by
  the asset consumer's templateId-mismatch branch.

**`saga/event_acceptance_test.go`**

See §5.2.

**`saga/step_event_matching_integration_test.go`**

See §5.3.

### 3.2 Modified files

**`saga/processor.go`** — add `AcceptEvent` method and the `AcceptDecision` struct.
`AcceptEvent` is added to the `Processor` interface. Implementation:

```go
func (p *ProcessorImpl) AcceptEvent(transactionId uuid.UUID, kind EventKind) (AcceptDecision, bool) {
    s, err := p.GetById(transactionId)
    if err != nil {
        logSkip(p.l, logrus.Fields{
            "transaction_id": transactionId.String(),
            "event_kind":     kind,
        }, skipReasonSagaNotFound)
        return AcceptDecision{}, false
    }
    step, ok := s.GetCurrentStep()
    if !ok {
        logSkip(p.l, logrus.Fields{
            "transaction_id": transactionId.String(),
            "event_kind":     kind,
        }, skipReasonNoPendingStep)
        // also warn-once-per-saga if no pending step in the saga accepts this kind
        p.maybeWarnUnmatchedEvent(s, kind)
        return AcceptDecision{}, false
    }
    if !StepAcceptsEvent(step.Action(), kind) {
        logSkip(p.l, logrus.Fields{
            "transaction_id": transactionId.String(),
            "step_id":        step.StepId(),
            "step_action":    step.Action(),
            "event_kind":     kind,
        }, skipReasonActionMismatch)
        p.maybeWarnUnmatchedEvent(s, kind)
        return AcceptDecision{}, false
    }
    return AcceptDecision{Saga: s, Step: step}, true
}
```

`maybeWarnUnmatchedEvent` scans pending steps and warns once per
`(transactionId, kind)` if no pending step accepts this kind — see §4.4.

**`saga/producer.go`** — introduce a test seam for `EmitConversationRewardNotice`.
Plan-phase picks the exact shape between two options (both are workable; the
decision is style not correctness):

- **Option 1 — package-level function variable:**
  ```go
  var emitConversationRewardNotice = emitConversationRewardNoticeImpl

  func EmitConversationRewardNotice(l, ctx, characterId, kind, itemId, quantity) error {
      return emitConversationRewardNotice(l, ctx, characterId, kind, itemId, quantity)
  }
  // tests overwrite `emitConversationRewardNotice` in setup/teardown.
  ```
- **Option 2 — `NoticeEmitter` interface** with a real impl and a test stub,
  threaded through the asset consumer via a small constructor.

Option 1 is the minimum-delta choice; Option 2 is cleaner if more call sites are
added later. Either satisfies the integration-test requirement.

**`kafka/consumer/asset/consumer.go`** — all four handlers gated. Key changes:

- `handleAssetCreatedEvent`: call `AcceptEvent(..., EventKindAssetCreated)`; on
  success, for `AwardAsset` / `CreateAndEquipAsset` branches verify
  `e.TemplateId == payload.Item.TemplateId` (debug-log `template_id_mismatch` and
  return on mismatch); keep the existing CreateAndEquipAsset follow-up-step logic;
  call `emitRewardNoticeForCurrentStep` passing `e.TemplateId` and `e.Body.Quantity`
  (see §3.4); then `StepCompletedWithResult`.
- `handleAssetDeletedEvent`: gate on `EventKindAssetDeleted`; templateId guard on
  DestroyAsset / DestroyAssetFromSlot; emit notice with event's templateId and
  event's quantity; `StepCompleted`.
- `handleAssetQuantityUpdatedEvent`: gate on `EventKindAssetQuantityChanged`;
  templateId guard; emit notice with event's templateId and event's delta
  quantity; `StepCompletedWithResult`.
- `handleAssetMovedEvent`: gate on `EventKindAssetMoved`; no templateId guard
  (asset-move steps don't carry a templateId in the sense the guard checks);
  `StepCompleted`.
- **Removed fallback:** the current `consumer.go:115` path that calls
  `StepCompletedWithResult(..., true, ...)` when the saga is not found is removed.
  `AcceptEvent` handles the saga-not-found case as a debug-log skip; there is
  nothing correct to complete when the saga doesn't exist.

**`kafka/consumer/character/consumer.go`** — all ten handlers gated. Each handler
picks its constant: `EventKindCharacterMapChanged`, `EventKindCharacterExperienceChanged`,
`EventKindCharacterLevelChanged`, `EventKindCharacterMesoChanged`,
`EventKindCharacterJobChanged`, `EventKindCharacterCreated`,
`EventKindCharacterCreationFailed`, `EventKindCharacterMesoError`,
`EventKindCharacterStatChanged`, `EventKindCharacterDeleted`. Failure-path
handlers (`handleCharacterCreationFailedEvent`, `handleCharacterMesoErrorEvent`)
use the same gate; a failure event from an unrelated subsystem is now safely
ignored.

**`kafka/consumer/compartment/consumer.go`** — six handlers gated.
`handleCompartmentErrorEvent` and `handleCompartmentCreationFailedEvent` are the
failure-path gates; the rest are success-path.

**`kafka/consumer/buddylist/consumer.go`**,
**`kafka/consumer/cashshop/consumer.go`**,
**`kafka/consumer/consumable/consumer.go`**,
**`kafka/consumer/guild/consumer.go`** (5 handlers),
**`kafka/consumer/invite/consumer.go`** (3 handlers),
**`kafka/consumer/pet/consumer.go`**,
**`kafka/consumer/quest/consumer.go`** (3 handlers),
**`kafka/consumer/skill/consumer.go`**,
**`kafka/consumer/storage/consumer.go`** — same mechanical rewrite. Each handler
adds an `AcceptEvent` call with its hardcoded `EventKind` constant before the
existing `StepCompleted` / `StepCompletedWithResult` call.

**`kafka/consumer/saga/consumer.go`** — review for `StepCompleted` call sites
(it has some); gate whichever apply. The grep shows call sites; the plan phase
confirms.

**Per-consumer test files** (`kafka/consumer/*/consumer_test.go`) — extend or
create per the strategy in §5.1.

### 3.3 Unchanged

- `libs/atlas-saga` — no payload shape changes, no new action constants.
- `saga/processor.go:316–330` (`stepCompletedWithResultOnce` idempotency guards)
  — unchanged. `AcceptEvent` runs before this code; it does not replace it.
- `saga/builder.go`, `saga/cache.go`, `saga/store.go`, `saga/compensator.go`,
  `saga/lifecycle.go`, `saga/timer.go`, `saga/resource.go`, `saga/rest.go` — no
  changes.
- All other services. The fix is fully internal to atlas-saga-orchestrator.

### 3.4 Reward-notice helper: templateId from event

`emitRewardNoticeForCurrentStep` in `kafka/consumer/asset/consumer.go:30` today
reads `templateId` and `quantity` from the current pending step's payload. It is
rewritten to:

- **Always** use the caller-provided `templateId` argument (which comes from the
  incoming event).
- Accept an additional `quantity uint32` argument, also sourced from the event
  body. For `ASSET_CREATED` and `ASSET_DELETED` that's a direct read from
  `e.Body.Quantity`. For `ASSET_QUANTITY_CHANGED` it is the delta from
  `e.Body.<field>` (exact field name determined at plan time by reading the
  message-body type).
- Exception: `DestroyAssetFromSlot` still reads `quantity` from the step payload,
  because the slot-based destroy's asset event does not carry a meaningful delta.
  This matches today's behaviour and is explicitly documented in PRD §4.4. Signal
  this exception with a handler branch (the `DestroyAssetFromSlot` case ignores
  the `quantity` arg).
- `ShowEffect` and `CharacterId` still come from the step payload (per PRD §4.4).

## 4. Runtime behaviour

### 4.1 Success path (§9.1 Thief scenario, post-fix)

See [§5.3](#53-integration-test-thief-advancement) for the full event-by-event
trace. Summary: all five steps complete in order, three
`conversation_reward_notice` emissions fire with templateIds
`[2070015, 1472061, 1332063]`, two STAT_CHANGED ripples land as debug
`action_mismatch` skips, no warn-level log fires, zero "no pending step" debug
logs.

### 4.2 Skip paths

`AcceptEvent` can return `(_, false)` for three structured reasons, each carrying
a `reason` field in the debug log:

| Reason | Condition | Preserved from today? |
|---|---|---|
| `saga_not_found` | `GetById` returned an error | Yes — today's handlers treat this as a no-op (except the asset-consumer's stale fallback, which is removed). |
| `no_pending_step` | Saga exists, `GetCurrentStep()` returns `(_, false)` | Yes — existing debug-log-and-return behaviour. |
| `action_mismatch` | Step exists, step's Action does not accept this kind | **New** — this is the bug fix. |

Additionally the asset handler can emit a `template_id_mismatch` debug log after
`AcceptEvent` succeeds — the gate passed but the event's templateId doesn't match
the step payload.

### 4.3 Failure-path gating

Handlers that call `StepCompleted(..., false)` (failure signals) go through the
same `AcceptEvent` gate. Their `EventKind` constants (e.g.,
`EventKindCharacterCreationFailed`, `EventKindCharacterMesoError`,
`EventKindCompartmentCreationFailed`, `EventKindCompartmentError`) are included
in the acceptance table entries of the actions they can legitimately fail
(e.g., `CreateCharacter` accepts both `EventKindCharacterCreated` and
`EventKindCharacterCreationFailed`). A failure event for a subsystem unrelated
to the current step is a `action_mismatch` skip, not a silent compensation.

### 4.4 Warn-once-per-saga for unmatched events

Implementation: a small `sync.Map` on `ProcessorImpl` keyed by
`(transactionId, kind)`. When `AcceptEvent` enters the skip path, call
`maybeWarnUnmatchedEvent(saga, kind)`:

1. Iterate `saga.GetPendingSteps()` (or equivalent — pick the existing method at
   plan time).
2. If any pending step's `Action` accepts this `kind`, return (the event will be
   consumed by a later saga state; not a warn-worthy skip).
3. If no pending step accepts this kind, check the dedup map. If already logged
   for this `(transactionId, kind)`, return.
4. Otherwise, warn with fields `transaction_id`, `tenant_id`, `event_kind`, and
   `reason="unmatched_event"`. Insert into the dedup map.

Entries for terminal sagas are fine to leave in the map — the `transactionId` is
never reused. If unbounded growth ever becomes a concern, clean up in the
terminal-state transition; not a concern for v1.

### 4.5 Concurrency and idempotency

`AcceptEvent` runs before `StepCompleted`, which is the only place version
conflicts happen. The existing `stepCompletedWithResultOnce` retry loop
(`saga/processor.go:316–330`) is unchanged. If two concurrent events pass the
gate simultaneously (rare — they would need to arrive on the same saga in the
same millisecond with compatible `EventKind`s for the same step), the existing
retry loop resolves the race exactly as it does today.

The net effect is: fewer racing completions reach `StepCompleted`, so version
conflict pressure drops.

## 5. Testing

### 5.1 Per-consumer unit tests

Each consumer package gets a small table-driven test covering three cases per
handler:

- **Match:** saga's current step accepts this `EventKind` → `StepCompleted` is
  called.
- **No match:** current step does not accept this kind → `StepCompleted` is NOT
  called, one debug log with `reason=action_mismatch`.
- **No saga / no pending step:** existing behaviour (debug log, no crash).

Asset consumer adds two more cases:

- **TemplateId match:** `ASSET_CREATED` with matching templateId → notice
  emitted (captured via the test seam from §3.2), step completed.
- **TemplateId mismatch:** `ASSET_CREATED` with non-matching templateId → debug
  log `reason=template_id_mismatch`, no notice, no completion.

Harness: construct a saga via `NewBuilder()`, `GetCache().Put(tctx, saga)`
(matches `saga/integration_test.go:TestCreateAndEquipAsset_CompleteIntegrationFlow`),
call handler directly with a fabricated `StatusEvent[...]`, assert on
`s.GetSteps()[n].Status()` and `logrus/hooks/test` log entries. No processor
mocking — the test exercises the real `AcceptEvent`.

### 5.2 Acceptance-table coverage test

`saga/event_acceptance_test.go`:

- **Coverage:** for every `saga.Action` constant, assert
  `_, ok := acceptanceTable[action]; ok`. Empty-slice entries pass this test;
  missing entries fail. Plan-phase task confirms the iteration is driven by a
  test-only list that matches the exported set in `libs/atlas-saga`.
- **Bug-class anti-matches:** spot tests that `AwardAsset`,
  `CreateAndEquipAsset`, and `ChangeJob` all return `false` for
  `EventKindCharacterStatChanged` (the STAT_CHANGED ripple in §9.1).
- **Success-kind rules:** each action's documented success kind returns `true`.
- **Failure-kind rules:** `CreateCharacter` accepts
  `EventKindCharacterCreationFailed`, `AwardMesos` accepts
  `EventKindCharacterMesoError`, `DestroyAsset` accepts `EventKindAssetDeleted`
  and `EventKindAssetQuantityChanged`, and so on.

### 5.3 Integration test — Thief advancement

`saga/step_event_matching_integration_test.go`:

1. Install a notice-emitter stub (via the seam from §3.2) that records
   `(characterId, kind, templateId, quantity)` into a slice.
2. Build the 5-step saga: `RebalanceAP` → `ChangeJob` → three `AwardAsset`
   steps with templateIds `2070015`, `1472061`, `1332063` (matching §9.1).
3. `GetCache().Put(tctx, saga)`.
4. Replay the event sequence by calling the matching handler functions directly:

| Call | Handler | Event | Expected saga state after | Expected log |
|---|---|---|---|---|
| 1 | `handleCharacterStatChangedEvent` | STAT_CHANGED (rebalance) | step 1 Completed | none special |
| 2 | `handleCharacterJobChangedEvent` | JOB_CHANGED | steps 1–2 Completed | none special |
| 3 | `handleCharacterStatChangedEvent` | STAT_CHANGED (`[JOB]`) | unchanged (3–5 still Pending) | 1× `action_mismatch` debug |
| 4 | `handleCharacterStatChangedEvent` | STAT_CHANGED (AP/SP/HP/MP) | unchanged | 2nd `action_mismatch` debug |
| 5 | `handleAssetCreatedEvent` | ASSET_CREATED (2070015) | step 3 Completed | none special |
| 6 | `handleAssetCreatedEvent` | ASSET_CREATED (1472061) | step 4 Completed | none special |
| 7 | `handleAssetCreatedEvent` | ASSET_CREATED (1332063) | all Completed | none special |

5. Assert: recorded notices are exactly
   `[{_, ItemGain, 2070015, q}, {_, ItemGain, 1472061, q}, {_, ItemGain, 1332063, q}]`
   in that order. Zero "no pending step" logs. Zero warn-level
   `unmatched_event` logs.

This test fails with a clear diff if any of the three fixes (gate, templateId
guard, notice templateId source) regresses.

### 5.4 Out of scope for tests

- No Kafka broker or testcontainer — the bug is not in the wire layer.
- No atlas-channel-side assertion that chat lines render — PRD §10.8 lists this
  as a manual kubectl inspection.

## 6. Out of scope (re-affirmed)

- No step-id correlation on Kafka events.
- No collapsing atlas-character's STAT_CHANGED ripples into one event.
- No new Kafka topics, bodies, or JSON-API endpoints.
- No changes to NPC scripts, quest scripts, `libs/atlas-saga` payloads, or
  atlas-ui.
- No changes to saga persistence, compensator algorithm, or SagaTimers.
- No OpenTelemetry / Prometheus / Grafana work.
- No rolling-deploy compatibility shims — in-flight sagas resume on the new pod
  with strict matching and simply wait longer for their correct event if an
  earlier ripple event today would have falsely completed them.

## 7. Risks and mitigations

| Risk | Mitigation |
|---|---|
| A consumer handler is missed during the rewrite; ripple bug persists on that path. | Grep-based acceptance criterion (`grep -rn "StepCompleted\|StepCompletedWithResult" services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/`) enumerated in PRD §10.2 is re-run in the plan's verification task and again in the audit phase. Every match must be preceded by an `AcceptEvent` call. |
| An action's table entry omits a legitimately-accepting event kind; a real saga stalls waiting for an event that's been silently skipped. | Per-consumer unit tests cover success-kind rules. The warn-once-per-saga log fires in production for genuinely unmatched events, providing a backstop for missed mappings. Worst case a saga reaches its timeout and compensates — same failure mode as today if a downstream service silently drops an event. |
| Test seam for reward-notice emission is added clumsily and leaks real Kafka writes from tests. | Plan phase picks between Option 1 (package-level `var`) and Option 2 (interface) and writes an explicit test that asserts no real producer is invoked when the stub is installed. |
| The warn-once dedup map grows unbounded for a misbehaving producer targeting a long-lived saga. | The cardinality is `pending kinds × sagas` — small in practice. Entries tied to terminal sagas could be cleaned up at terminal-state transition; not required for v1. Called out as a follow-up only if it becomes observable. |
| `AwardAsset` and `CreateAndEquipAsset` both accept `EventKindAssetCreated`; a saga that mixes them could in principle match the wrong step. | In practice the two are never sequenced against the same ASSET_CREATED event because the orchestrator issues one command at a time. The templateId guard is an additional filter. No additional mitigation needed. |

## 8. Plan-phase inputs

The plan phase (`/plan-task task-021-saga-step-event-matching`) should produce a
TDD-ordered task list roughly matching:

1. Introduce `EventKind` type, constants, `acceptanceTable`, and
   `StepAcceptsEvent`. Write the coverage and anti-match tests first.
2. Add `AcceptDecision` + `AcceptEvent` to `Processor` interface and
   `ProcessorImpl`. Write `AcceptEvent` unit tests (every skip path) first.
3. Introduce the reward-notice test seam. Decide between Option 1 and Option 2
   at this point; the design is indifferent.
4. Update `emitRewardNoticeForCurrentStep` to source templateId and quantity
   from the caller (i.e., from the event). Unit-test in isolation.
5. Rewrite `kafka/consumer/asset/consumer.go` handlers in order: each handler
   gets a failing unit test first, then the gate + templateId guard, then tests
   pass. Include the removal of the saga-not-found fallback at `consumer.go:115`.
6. Rewrite each of `character`, `compartment`, `buddylist`, `cashshop`,
   `consumable`, `guild`, `invite`, `pet`, `quest`, `skill`, `storage` consumer
   packages. One package per plan-task step; each step is TDD-driven.
7. Add the warn-once-per-saga logic to `AcceptEvent`. Unit-test.
8. Write the Thief-scenario integration test.
9. Run `go build ./...` + `go test ./...` at atlas-saga-orchestrator root; run
   `grep -rn "StepCompleted\|StepCompletedWithResult" ...` and verify every
   match is gated.
10. Manual kubectl log inspection during a live Thief advancement (PRD §10.8).

The plan phase should not introduce new architectural choices. If any arise,
return to `/design-task` to document the additional decision.
