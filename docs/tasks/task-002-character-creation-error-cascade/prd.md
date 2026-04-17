# Character Creation Error Cascade — Product Requirements Document

Version: v2
Status: Draft
Created: 2026-04-17
Updated: 2026-04-17 — replaced §9 open questions with confirmed findings; expanded §4.3 compensation requirements; revised §4.4 to extend the existing factory bridge rather than introduce a new topic; added §4.7 (timeout/StepCompleted race), §4.8 (delete-command idempotency), §4.9 (orphan-leak acceptance), and §8 testing call-out.

---

## 1. Overview

When a player attempts to create a character through the login flow, the request fans out from `atlas-login` (TCP socket) → `atlas-character-factory` (REST 202) → `atlas-saga-orchestrator` (multi-step saga) → discrete services (`atlas-character`, `atlas-inventory`, `atlas-skill`, etc.), several of which depend on `atlas-data` to resolve wz lookups for starting items, equipment, and skills. Today, when any step in that fan-out fails — most acutely when `atlas-data` is unavailable and the `award_item` / `create_and_equip_asset` steps cannot resolve template data — no failure ever surfaces back to the client. The socket sits indefinitely waiting for a `character_status: CREATED` event that will never arrive, and the user perceives the game as hung.

Investigation identified three concrete error-swallow sites: the saga command consumer logs and discards errors from `processor.Put()` (`kafka/consumer/saga/consumer.go:49`); the `atlas-character` create consumer explicitly discards the error from `CreateAndEmit` with `_, _` (`kafka/consumer/character/consumer.go:352`); and `atlas-login`'s seed consumer subscribes only to `StatusEventTypeCreated`, with no handler for failure events and no timeout backstop (`kafka/consumer/seed/consumer.go:43-71`). The saga infrastructure already publishes a `StatusEventTypeFailed` event on `EVENT_TOPIC_SAGA_STATUS` with a structured body (`reason`, `failedStep`, `sagaType`, `errorCode`, `characterId`), so the wire format and emit path partly exist — they are simply not wired through end-to-end for the character-creation flow.

This task closes the loop. It guarantees a Failed saga event is emitted on every error path of the character-creation saga, ensures the orchestrator runs full compensation to roll the system back to a pre-creation state, surfaces that failure to the player as a generic "creation failed" client write via `atlas-character-factory`, and adds a caller-supplied saga timeout (with a sensible default) as a backstop so a wedged saga eventually fails closed instead of hanging open.

## 2. Goals

### Primary goals

- Every failure mode in the character-creation saga (`atlas-data` unavailable, downstream service error, step handler error, saga-cache write error, timeout) results in exactly one `StatusEventTypeFailed` event published to `EVENT_TOPIC_SAGA_STATUS` with the originating `transactionId`.
- The character-creation saga performs full compensation on failure: any character row, items, skills, or assets created by completed steps prior to the failing step are rolled back, leaving the system in the state it was in before the seed REST call was issued.
- `atlas-character-factory` consumes saga status events, filters to `SagaType == CharacterCreation` it owns (by transactionId tracking), and re-emits a domain-specific character-creation status event consumed by `atlas-login`.
- `atlas-login` writes `AddCharacterEntryWriter(AddCharacterCodeUnknownError)` to the originating client session on any failure, instead of waiting indefinitely.
- A saga timeout, supplied by the caller at saga creation time (with a 30s default fallback), causes the orchestrator to emit a Failed event and trigger compensation if the saga has not reached a terminal state by the deadline. `atlas-character-factory` will pass `10s` for character-creation sagas.
- The three identified error-swallow sites are fixed: the saga consumer emits Failed on `Put()` error; the `atlas-character` create consumer no longer discards the `CreateAndEmit` return value and ensures a creation-failed event is always emitted on the character status topic; the login seed consumer handles the new failure event.

### Non-goals

- Per-tenant configuration of timeout values. Timeouts are per-saga, supplied by the caller; tenant-level config is out of scope.
- Distinguishing failure causes to the end user. The client receives a single generic error code; reason/errorCode/failedStep are for server-side logs only.
- Auditing or fixing the same swallowed-error pattern in adjacent login sagas (delete character, rename, etc.). Those will be filed as follow-up tasks.
- Integration tests across multiple services. Unit tests covering the new emit/consume paths and compensation logic are sufficient.
- Adding new error codes beyond what is needed to disambiguate character-creation failures in logs (`ErrorCodeUnknown` and one or two specific codes if naturally introduced — e.g., `ErrorCodeSagaTimeout`).
- Changing the `atlas-character-factory` REST contract or the `POST /characters/seed` 202-Accepted semantics. The REST response remains a transactionId; the failure surfaces over the socket, not as an HTTP error.
- Refactoring how `atlas-data` reports unavailability. Downstream services already convert atlas-data errors into per-domain failure events; this task simply ensures those failure events propagate up the saga.

## 3. User Stories

- As a player, when I submit character creation and the server cannot complete it, I want to see a creation-failed message in the client within 10 seconds, so I can retry rather than restart the client.
- As an operator, when a character creation fails, I want a single `StatusEventTypeFailed` event in Kafka with `failedStep`, `reason`, and `errorCode`, so I can grep logs and isolate which downstream service or dependency caused the failure.
- As an operator, I want the system left in a clean pre-creation state after a failure, so a retry by the same player succeeds without name collisions, orphaned items, or partial database rows.
- As a developer extending the saga orchestrator, I want a single explicit place where step errors are converted to Failed events, so I cannot accidentally introduce a new swallowed-error path.
- As a developer of an upstream caller (today: `atlas-character-factory`; tomorrow: `atlas-npc-conversations`, etc.), I want to specify a per-saga timeout so my service's UX contract isn't held hostage to a downstream service that never responds.

## 4. Functional Requirements

### 4.1 Saga timeout

- The `Saga` model gains a `timeout` field (type `time.Duration`).
- The saga creation Kafka command (`COMMAND_TOPIC_SAGA`) and the corresponding command body are extended to carry a `timeout` value (serialized as integer milliseconds or duration string — implementation choice).
- If the field is absent or zero in the inbound command, the orchestrator applies a default of **30 seconds**.
- `atlas-character-factory` populates the field with **10 seconds** when emitting the character-creation saga command.
- The orchestrator schedules a per-saga timer at saga acceptance. If the saga has not reached `Completed` or `Failed` terminal status when the timer fires, the orchestrator:
  1. Marks the current pending step as `Failed` with reason `"saga timed out"`.
  2. Triggers full compensation (see §4.3).
  3. Emits a `StatusEventTypeFailed` event with `errorCode = "SAGA_TIMEOUT"` (new constant in `kafka/message/saga/kafka.go`), `failedStep = <currentStepId>`, and `reason = "saga exceeded timeout of <N>s"`.
- Timers are cancelled when the saga reaches a terminal state through the normal path.

### 4.2 Guaranteed Failed event emission

The orchestrator must emit exactly one `StatusEventTypeFailed` for each saga that does not reach `Completed`. Specifically:

- **Saga consumer (`kafka/consumer/saga/consumer.go`)**: when `processor.Put()` returns an error, the consumer must emit a Failed event with `errorCode = ErrorCodeUnknown`, `reason = err.Error()`, and `failedStep = ""` (saga did not enter step execution) before returning. The error must still be logged.
- **Step handlers (`saga/handler.go`)**: when a handler returns an error (sync path), the calling site that already exists in `processor.Step()` must emit a Failed event with the failed step's `stepId` as `failedStep`, the action-handler-supplied reason, and an `errorCode` derived from the error (default `ErrorCodeUnknown`). Today `processor.Step()` returns the error to the consumer where it is logged and dropped; this code path must instead emit Failed.
- **Async step completion (`StepCompleted(transactionId, success=false)`)**: callers (e.g., `compartment/consumer.go` handling `StatusEventTypeCreationFailed`) already exist. When `StepCompleted` is invoked with `success=false`, the orchestrator must emit a Failed event with `failedStep = <currentStepId>`, `errorCode` derived from the upstream domain event when available, and a reason captured from the upstream event body. Confirm this path emits today; if not, add it.
- The Failed event's body must populate `sagaType`, `transactionId`, `characterId` (if extractable from saga steps — see existing `extractCharacterCreationResults` pattern in `saga/producer.go:34`), and `failedStep`.
- A saga must not emit more than one Failed event. The orchestrator must guard against double-emission (e.g., timer fires while compensation is mid-flight).

### 4.3 Compensation for character creation

The existing compensator (`saga/compensator.go`) is materially insufficient for this saga: `compensateCreateCharacter` (lines 392-444) explicitly does no rollback (comment at line 409-411 notes no character delete command exists), `AwardAsset` and `CreateSkill` have no entries in the compensator switch (lines 205-225), and the generic `CompensateFailedStep` only compensates the **failed step** — it does not walk back through completed prior steps. Only `compensateSelectGachaponReward` (line 795) does the reverse-walk pattern, ad-hoc.

This task introduces a `CharacterCreation`-specific compensation path that walks back through completed steps in reverse order and inverts each. Specifically:

#### 4.3.1 New cross-service commands

- **`atlas-character`** — add a saga-aware character-deletion request, e.g. `RequestDeleteCharacter(transactionId, characterId)`. Implementation: a Kafka command consumed by `atlas-character`, which deletes the character row and cascade rows, then emits a status event the orchestrator correlates back via the standard step-completed mechanism. Include guard against deleting characters not created within an in-flight saga (compare against the saga's `CharacterCreatePayload`).
- **`atlas-skill`** — add an analogous `RequestDeleteSkill(transactionId, characterId, skillId)` command and consumer. Same contract.
- `atlas-inventory` already exposes `RequestDestroyItem(transactionId, characterId, templateId, quantity, removeAll)` (used today by `compensateCreateAndEquipAsset` at `compensator.go:502` and `compensateSelectGachaponReward` at `:834`). Reuse for `AwardAsset`/`AwardItem` rollback.

#### 4.3.2 Character-creation rollback path

- Add a new branch in `CompensateFailedStep` (`compensator.go:205`) keyed on `s.SagaType() == CharacterCreation` that takes precedence over the per-step switch. The branch:
  1. Walks `s.Steps()` in reverse, examining each step with `Status() == Completed`.
  2. For each completed step, dispatches the inverse:
     - `AwardAsset` / `AwardItem` → `compP.RequestDestroyItem(...)` using the original payload's `templateId` and `quantity`.
     - `CreateAndEquipAsset` → already-implemented destroy logic (see `compensator.go:502`); reuse.
     - `CreateSkill` → new `skillP.RequestDeleteSkill(...)`.
     - `CreateCharacter` → new `charP.RequestDeleteCharacter(...)`. This must run **last** (deepest reverse step) so item/skill rows referencing the character are removed first.
  3. Awaits each compensation step's success-or-failure event before proceeding to the next, preserving causal ordering.
  4. Emits `StatusEventTypeFailed` (single emission) once the rollback chain terminates, with `failedStep` = the originally-failing step's id.
  5. Removes the saga from the cache.
- Compensation step failures are logged but do not block the chain: the next reverse step still runs, and the Failed event is still emitted at the end. A compensation failure is logged at ERROR with all relevant ids for operator follow-up.
- The existing per-step compensators (`compensateEquipAsset`, etc.) are not removed in v1 — they continue to serve `InventoryTransaction`, `StorageOperation`, etc. The new branch is character-creation-specific.

#### 4.3.3 Failed event payload — `accountId` required

- `StatusEventFailedBody` (`kafka/message/saga/kafka.go:36`) currently carries `characterId` but no `accountId`. Login resolves sessions by `accountId`, and a character-creation failure may occur before `characterId` is known (e.g., atlas-data unreachable on the first item lookup with `CreateCharacter` not yet run, or `CreateCharacter` itself failing).
- Add `AccountId uint32 \`json:"accountId"\`` to `StatusEventFailedBody`. The orchestrator populates it by extracting from the saga's `CharacterCreatePayload` (use the same helper pattern as `extractCharacterCreationResults` in `producer.go:34`). For non-character-creation sagas it remains zero; this is a backward-compatible addition.

### 4.4 `atlas-character-factory` saga status bridge

The factory **already** consumes `EVENT_TOPIC_SAGA_STATUS` filtered by `SagaType == CharacterCreation` (`kafka/consumer/saga/consumer.go:40-73`) and re-emits the success path to existing topic `EVENT_TOPIC_SEED_STATUS` (`kafka/message/seed/kafka.go:4`). This task extends that bridge to also handle the failure path on the same topic. No new topic is introduced.

- Add `StatusEventTypeFailed = "FAILED"` constant to `kafka/message/seed/kafka.go`.
- Add `FailedStatusEventBody struct{}` (or with a `Reason string` field for diagnostic logging) to `kafka/message/seed/kafka.go`. Body content is for log correlation only; the client-side write is generic.
- Add `FailedEventStatusProvider(accountId uint32, reason string)` to `kafka/producer/seed/producer.go` mirroring the existing `CreatedEventStatusProvider`.
- Extend `handleSagaCompletedEvent` (or split into `handleSagaCompletedEvent` / `handleSagaFailedEvent`, registered as separate handlers on the same topic via `AdaptHandler`): on `StatusEventTypeFailed` with `SagaType == CharacterCreation`, extract `accountId` from the new `StatusEventFailedBody.AccountId` field (see §4.3.3), and call the new producer.
- Filter is by `SagaType` only — the factory does not need an in-flight tracking map. No other service emits `CharacterCreation` sagas today; if that changes, the bridge stays correct because it cares only about saga type, not initiator.
- The existing post-`sagaProcessor.Create(...)` flow in the factory's REST handler is unchanged. The 202-Accepted response continues to return immediately with a transactionId; success/failure travel via Kafka through the bridge as today.

### 4.5 `atlas-login` failure handling

- `atlas-login`'s seed consumer (`kafka/consumer/seed/consumer.go`) already subscribes to `EVENT_TOPIC_SEED_STATUS` and handles `StatusEventTypeCreated`. Topic stays the same; add a new handler registration for `StatusEventTypeFailed`.
- On `FAILED`: resolve the session by `accountId` (top-level field on `seed.StatusEvent`), write `AddCharacterEntryWriter` with `AddCharacterCodeUnknownError`, and clear any in-flight transaction state for that session.
- Sessions must tolerate the case where the client has already disconnected by the time the failure event arrives — log and drop, do not panic.

### 4.6 Fix `atlas-character` discarded error

- `kafka/consumer/character/consumer.go:352` currently reads `_, _ = character.NewProcessor(l, ctx, db).CreateAndEmit(c.TransactionId, model)`. The error must be captured and logged.
- `character/processor.go` must be audited to ensure every error path inside `CreateAndEmit` results in a `creationFailedEventProvider` being placed on the character status topic. Today line 223 emits on one error path; add coverage for any path that returns without emitting.
- The `creationFailedEventProvider` must include enough context (transactionId, accountId, reason) for the saga orchestrator's existing character status consumer to correlate it with the in-flight saga and drive `StepCompleted(txId, false)` with a meaningful reason.

### 4.7 Saga terminal-state guard (timeout / late-completion race)

A timer firing at the deadline can race with a downstream `StepCompleted(txId, ...)` call. The orchestrator must treat the saga's terminal state (`Failed` or `Completed`) as a one-shot transition guarded by the cache:

- The timer's first action is an atomic check-and-mark: if the saga is still pending in the cache, transition it to a terminal "compensating" or "failed" state and proceed; otherwise log "saga already terminal, timer no-op" and return.
- Symmetrically, `StepCompleted(txId, success)` must check terminal state before acting. If the saga has already moved to terminal due to a prior timeout (or duplicate fire), the call logs "saga already terminal, late completion ignored" and returns nil.
- The same guard prevents double-emission of `StatusEventTypeFailed` and double-emission of `StatusEventTypeCompleted`.
- Implementation note: this likely requires a small state machine on the cache entry (e.g., `Pending → Compensating → Failed` or `Pending → Completed`) rather than relying on step-level `Pending/Completed/Failed` alone.

### 4.8 Idempotency of compensation delete commands

The new `atlas-character` `RequestDeleteCharacter` and `atlas-skill` `RequestDeleteSkill` commands must be idempotent against missing rows: if the target row does not exist (e.g., `CreateCharacter` was attempted but no row was actually written before failure), the consumer treats this as a successful no-op, not an error. The orchestrator must not pre-check existence — the delete command is the source of truth for "is this gone now". This keeps the orchestrator's compensation chain simple and avoids TOCTOU races.

### 4.9 Late-success leak (accepted limitation)

If a downstream service is slow but eventually succeeds *after* the saga has timed out and compensation has run, side effects produced after the timeout (e.g., an item row inserted by a delayed atlas-inventory response) will not be rolled back. The orchestrator's saga is gone from the cache, so the late success event lands on nothing. This is an accepted limitation for v1: the user can retry, and operators may need a separate sweep for orphaned inventory rows over time. A follow-up task may add a "saga-aware" check in downstream services that bails before persisting if the originating transaction is already terminal — out of scope here.

## 5. API Surface

### 5.1 Kafka topics

**Modified — `EVENT_TOPIC_SAGA_STATUS`**
- Behavior: orchestrator now emits `FAILED` on every previously-swallowed error path (saga consumer error, step handler error, async `StepCompleted(false)` for character-creation, timeout).
- Schema: `StatusEventFailedBody` (`kafka/message/saga/kafka.go:36`) gains `AccountId uint32 \`json:"accountId"\``. Backward compatible (zero-valued for non-character-creation sagas).
- New constant `ErrorCodeSagaTimeout = "SAGA_TIMEOUT"` added to `kafka/message/saga/kafka.go`.

**Modified — `COMMAND_TOPIC_SAGA`**
- The saga creation command body gains an optional `timeout` field (integer milliseconds). Marshal/unmarshal must default to 30s when missing.

**Modified — `EVENT_TOPIC_SEED_STATUS`** (owned by `atlas-character-factory`, consumed by `atlas-login`)
- Add `StatusEventTypeFailed = "FAILED"` to `services/atlas-character-factory/atlas.com/character-factory/kafka/message/seed/kafka.go`.
- Add `FailedStatusEventBody` (with diagnostic `Reason string` field, optional) alongside the existing `CreatedStatusEventBody`.
- Existing `StatusEvent[E]` envelope keeps `AccountId` at top level — already correct for both Created and Failed.

**New — Kafka commands for compensation**
- `atlas-character`: command consumer for `RequestDeleteCharacter(transactionId, characterId)`. Topic and body schema TBD during implementation; follow existing `atlas-character` saga-correlated command conventions (e.g., the topic that handles `CreateCharacter` today).
- `atlas-skill`: command consumer for `RequestDeleteSkill(transactionId, characterId, skillId)`. Same pattern.

### 5.2 REST

No REST endpoint changes. `POST /characters/seed` continues to return `202 Accepted` with `{ "transactionId": "..." }`.

## 6. Data Model

No schema changes. The orchestrator's saga cache (in-memory) gains a per-saga timer reference and a `timeout` field; the cache is not persisted.

`atlas-character-factory` adds an in-memory map of in-flight transactionIds → originating accountId/worldId for the saga status bridge; not persisted.

## 7. Service Impact

| Service | Changes |
|---|---|
| `atlas-saga-orchestrator` | Add `timeout` to `Saga` model and command body; per-saga timer scheduling and cancellation with double-emission guard; new `ErrorCodeSagaTimeout` constant; add `AccountId` to `StatusEventFailedBody`. New `CharacterCreation`-specific compensation branch in `CompensateFailedStep` (`compensator.go:205`) that walks completed steps in reverse and dispatches inverse actions, then emits a single Failed event and removes the saga from cache. New compensation calls into `atlas-character` (delete character) and `atlas-skill` (delete skill) processors. Guarantee `StatusEventTypeFailed` emission on every error path (saga consumer `Put()` error at `kafka/consumer/saga/consumer.go:49`; step handler errors in `processor.Step()`; async `StepCompleted(false)` for character-creation; timeout). |
| `atlas-character-factory` | Pass `timeout: 10s` when creating character-creation sagas. Extend the **existing** `kafka/consumer/saga/consumer.go` to add a `handleSagaFailedEvent` handler on the same `EVENT_TOPIC_SAGA_STATUS` topic, filtered by `SagaType == CharacterCreation`. Extend `kafka/message/seed/kafka.go` with `StatusEventTypeFailed` and `FailedStatusEventBody`. Add `FailedEventStatusProvider` to `kafka/producer/seed/producer.go`. |
| `atlas-login` | Add a `StatusEventTypeFailed` handler on the existing `EVENT_TOPIC_SEED_STATUS` subscription in `kafka/consumer/seed/consumer.go`. On FAILED, resolve session by `accountId` and write `AddCharacterEntryWriter(AddCharacterCodeUnknownError)`. |
| `atlas-character` | Stop discarding the error returned by `CreateAndEmit` at `kafka/consumer/character/consumer.go:352`. Audit `character/processor.go` to ensure every error path in `CreateAndEmit` emits a `creationFailedEventProvider`. **New**: add a saga-correlated character-deletion command (consumer + processor method) for compensation use. |
| `atlas-skill` | **New**: add a saga-correlated skill-deletion command (consumer + processor method) for compensation use. |
| `atlas-inventory`, `atlas-data` | No code changes expected. `atlas-inventory.RequestDestroyItem` already exists and is reused for `AwardAsset` rollback. |

## 8. Non-Functional Requirements

- **Latency budget**: a failed character-creation saga must surface to the client within 11s in the worst case (10s saga timeout + 1s for compensation start + Failed event emission + factory bridge + login write).
- **Idempotency**: receiving the saga timeout firing after the saga has already reached terminal status is a no-op. The double-emission guard must be safe under concurrent step completion and timer firing.
- **Multi-tenancy**: all new Kafka events carry the standard tenant header. `atlas-character-factory`'s in-flight tracking map must be tenant-scoped (or include tenant in the key) so two tenants cannot collide on transactionId.
- **Logging**: every Failed event emission logs at WARN level (or ERROR for unexpected internal failures) with `transactionId`, `sagaType`, `failedStep`, `errorCode`, and `reason`. The login-side failure write logs at INFO with `accountId` and `transactionId`.
- **Observability**: no new metrics required for v1; existing Kafka consumer metrics will surface volumes. A follow-up task may add counters for `saga_failed_total{saga_type, error_code}`.
- **Backwards compatibility**: the `timeout` field on the saga command is optional with a 30s server-side default. Any existing saga that legitimately takes longer than 30s to complete is, by this PRD's stance, a bug that should surface — we are not preserving the indefinite-wait behavior for any caller. If audits during implementation find a saga that genuinely needs more than 30s, the caller must explicitly supply a higher value rather than relying on the default being raised.
- **Test impact**: the existing orchestrator tests (`saga/integration_test.go`, `saga/createandequip_integration_test.go`) and per-step processor tests are likely to fail under the new failure semantics (Failed event emission where there was none, terminal-state guard, new compensation branch). Audit and update these alongside the new tests; do not treat as follow-up. Mock changes are likely required for `saga/mock/processor.go` to add timeout-related fields.

## 9. Confirmed Findings (formerly Open Questions)

All three open questions from v1 were resolved before implementation by reading the orchestrator and factory code directly:

1. **Compensator coverage**: materially incomplete. `compensateCreateCharacter` (`compensator.go:392-444`) has no rollback (comment at line 409 cites missing character-delete command). `AwardAsset` and `CreateSkill` have no compensator entries (switch at lines 205-225). `CompensateFailedStep` only handles the failed step itself; only `compensateSelectGachaponReward` (line 795) walks completed steps in reverse, ad-hoc. → §4.3 expanded to introduce a `CharacterCreation`-specific reverse-walk compensator and new delete commands in `atlas-character` and `atlas-skill`.

2. **`StepCompleted(false)` Failed emission**: inconsistent. Per-step compensators (`compensateEquipAsset`, `compensateCreateCharacter`, `compensateCreateAndEquipAsset`, `compensateChangeHair/Face/Skin`) only mark the step as Pending and continue — no Failed emission. Only `ValidateCharacterState` branch (line 191), `compensateStorageOperation` (line 768), and `compensateSelectGachaponReward` (line 850) emit Failed today. → §4.2 hard requirement: the new character-creation compensation branch always emits exactly one Failed event at end of rollback chain.

3. **Factory bridge**: already exists for the success path. `atlas-character-factory/kafka/consumer/saga/consumer.go:40-73` consumes `EVENT_TOPIC_SAGA_STATUS`, filters `SagaType == CharacterCreation`, and emits to existing topic `EVENT_TOPIC_SEED_STATUS`. → §4.4 revised to extend this existing consumer, not introduce a new topic. No in-flight tracking map needed; sagaType filter is sufficient.

**Additional finding (introduced during the verification pass):** `StatusEventFailedBody` carries `characterId` but no `accountId`. Failures may occur before `characterId` is established, and login resolves sessions by `accountId`. → §4.3.3 adds `AccountId` to the body schema, populated from the saga's `CharacterCreatePayload`.

## 10. Acceptance Criteria

- [ ] `Saga` model has a `timeout time.Duration` field, populated from the inbound `COMMAND_TOPIC_SAGA` command body or defaulted to 30s.
- [ ] `atlas-character-factory` emits character-creation saga commands with `timeout = 10s`.
- [ ] When a character-creation saga's `award_item` or `create_and_equip_asset` step fails because `atlas-data` is unreachable, exactly one `StatusEventTypeFailed` event is published on `EVENT_TOPIC_SAGA_STATUS` with `sagaType = "character_creation"`, the failing step's id, and a non-empty reason.
- [ ] Compensation runs after a character-creation failure: the partially-created character row is deleted, awarded items are destroyed, equipped items are unequipped and destroyed, and created skills are deleted. A retry of the same character creation succeeds without name-collision or orphan errors.
- [ ] `atlas-character-factory`'s existing `EVENT_TOPIC_SAGA_STATUS` consumer additionally handles `StatusEventTypeFailed` for `SagaType == CharacterCreation` and re-emits to `EVENT_TOPIC_SEED_STATUS` with the new `StatusEventTypeFailed`, carrying `accountId`.
- [ ] `atlas-login`'s existing seed consumer on `EVENT_TOPIC_SEED_STATUS` handles the new `StatusEventTypeFailed` and writes `AddCharacterEntryWriter` with `AddCharacterCodeUnknownError` to the session resolved by `accountId`.
- [ ] When the saga is artificially wedged (no downstream response), the 10s timeout fires, a Failed event is emitted with `errorCode = "SAGA_TIMEOUT"`, compensation runs, and the client receives the failure write.
- [ ] The saga command consumer at `kafka/consumer/saga/consumer.go:49`, the character create consumer at `kafka/consumer/character/consumer.go:352`, and the seed consumer at `kafka/consumer/seed/consumer.go:43-71` no longer swallow errors silently; each path is covered by a unit test.
- [ ] Unit tests cover: timeout firing emits Failed; double-emission is suppressed; saga consumer `Put()` error emits Failed; step handler error emits Failed; async `StepCompleted(false)` emits Failed; factory bridge filters by sagaType and transactionId membership; login failure write resolves the correct session.
- [ ] `go test ./...` passes for `atlas-saga-orchestrator`, `atlas-character-factory`, `atlas-login`, and `atlas-character`. `go build` passes for all four.
- [ ] No new ingress routes; no README updates required (no public REST contract changed). Service-internal Kafka tables in each service's `README.md` updated to reflect the new event topic and modified emit/consume sites.
