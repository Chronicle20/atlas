---
name: Character Creation Error Cascade — Task Checklist
description: Progress checklist for closing the character-creation failure loop across orchestrator, factory, character, skill, and login services.
type: tasks
task: task-002-character-creation-error-cascade
---

# Tasks — Character Creation Error Cascade

Last Updated: 2026-04-17

Legend: effort = S (≤0.5d) / M (0.5–2d) / L (2–5d) / XL (>5d). Phases are sequentially load-bearing unless noted.

## Phase 0 — Safety rails (S)

- [x] **0.1** Create feature branch off `main` for the task. *(effort: S)* — **Skipped per user direction; work continues on `deploy-reorg`.**
- [x] **0.2** Baseline build: `go build ./...` in `atlas-saga-orchestrator`, `atlas-character-factory`, `atlas-character`, `atlas-skills`, `atlas-login`. *(effort: S)* — All green. Note: service dir is `atlas-skills` (plural), not `atlas-skill`.
- [x] **0.3** Baseline test: `go test ./...` for the same five services; pass list captured in `baseline.md`. *(effort: S)*
- [x] **0.4** Confirm no in-flight edits to `saga/`, `compensator.go`, `producer.go`, or the three error-swallow files. *(effort: S)* — Confirmed.

**Acceptance:** Branch exists; all five services build and test green at baseline. ✅

## Phase 1 — Wire-format extensions, no behavior change (S)

- [x] **1.1** Add `AccountId uint32` to `StatusEventFailedBody`. *(effort: S)*
- [x] **1.2** Add `ErrorCodeSagaTimeout = "SAGA_TIMEOUT"` constant. *(effort: S)*
- [x] **1.3** Extend `Saga` with `timeout time.Duration`; marshal as ms in JSON; decode defaults to 30s via `DefaultSagaTimeout`. Builder gets `SetTimeout`; `Saga.Timeout()` returns the effective value. *(effort: S)*
- [x] **1.4** Add `StatusEventTypeFailed` + `FailedStatusEventBody` to factory seed kafka.go. *(effort: S)*
- [x] **1.5** Add `FailedEventStatusProvider(accountId, reason)` to factory seed producer.go. *(effort: S)*
- [x] **1.6** `go build ./...` for orchestrator + factory — green. `go test ./saga/... ./kafka/...` still green. *(effort: S)*

**Acceptance:** All five new fields/constants compile. Existing tests still pass. Nothing consumes the new shapes; nothing emits them. ✅

## Phase 2 — Orchestrator terminal-state guard (M)

- [ ] **2.1** Add a terminal-state field to the saga cache entry (`Pending → Compensating → Failed` / `Pending → Completed`). Decide: mutex on the entry vs. atomic status field — match the cache's existing concurrency style. *(effort: M)*
- [ ] **2.2** Expose an atomic check-and-mark helper (`cache.TryTransition(txId, from, to) (ok bool)`) consumed by `StepCompleted`, the forthcoming timer, and `CompensateFailedStep`. *(effort: S)*
- [ ] **2.3** Update `StepCompleted` so it takes the guard before acting: `ok := TryTransition(Pending, Compensating)` on failure, or completion transition on success. On `!ok`, log "saga already terminal, late completion ignored" and return `nil`. *(effort: M)*
- [ ] **2.4** Unit test: concurrent double-transition attempts — exactly one wins; the loser is a no-op. *(effort: S)*

**Acceptance:** Saga cache exposes a state machine; `StepCompleted` respects it; unit test for concurrent transitions passes; existing tests still green.

## Phase 3 — Guaranteed Failed emission on all error paths (M)

- [ ] **3.1** At `atlas-saga-orchestrator/.../kafka/consumer/saga/consumer.go:49`: on `processor.Put()` error, emit `StatusEventTypeFailed` with `errorCode = ErrorCodeUnknown`, `reason = err.Error()`, empty `failedStep`. Keep the error log. Use a helper to extract `AccountId` from the inbound command (it has not entered step execution yet). *(effort: M)*
- [ ] **3.2** In `processor.Step()` sync error path: instead of returning the error to the consumer (where it is currently logged and dropped), take the terminal-state guard and emit `StatusEventTypeFailed` with `failedStep = <stepId>`, the action-handler-supplied `reason`, derived `errorCode` (default `ErrorCodeUnknown`). *(effort: M)*
- [ ] **3.3** Audit `StepCompleted(transactionId, success=false)` for character-creation: confirm/ensure it emits `StatusEventTypeFailed` with `failedStep = <currentStepId>`, `errorCode` derived from upstream domain event where available, `reason` captured from upstream body. *(effort: M)*
- [ ] **3.4** All three emission sites go through a single producer helper so the `AccountId`/`CharacterId` extraction from `CharacterCreatePayload` (re-use the `extractCharacterCreationResults` pattern at `saga/producer.go:34`) is not duplicated. *(effort: S)*
- [ ] **3.5** Verify double-emission is suppressed by the Phase-2 guard across all three paths. *(effort: S)*

**Acceptance:** Every previously-silent error path in the orchestrator emits exactly one Failed event; `AccountId` populated when derivable; terminal-state guard prevents duplicates.

## Phase 4 — Per-saga timeout (M)

- [ ] **4.1** Add `timeout time.Duration` to `Saga` model; populated from the inbound command body (defaulted to 30s in Phase 1.3). *(effort: S)*
- [ ] **4.2** Schedule a per-saga `time.AfterFunc` at saga acceptance (inside `processor.Put()`). Store the timer handle on the cache entry. *(effort: M)*
- [ ] **4.3** On timer fire: take the terminal-state guard; if still `Pending`, transition to `Compensating`; mark the current step `Failed` with reason `"saga timed out"`; drive Phase-6 compensation; emit `StatusEventTypeFailed` with `errorCode = ErrorCodeSagaTimeout` and `reason = "saga exceeded timeout of <N>s"`. *(effort: M)*
- [ ] **4.4** Cancel the timer on normal terminal transitions (both Completed and Failed paths). Guard against `nil` timer if cache cleanup already happened. *(effort: S)*
- [ ] **4.5** `atlas-character-factory` passes `timeout: 10 * time.Second` in the outbound saga-creation command (move alongside Phase 7 once the wire field is known-good — same PR, just ordered here for clarity). *(effort: S)*

**Acceptance:** A wedged saga (no downstream responses) emits Failed at exactly `timeout` + <1s compensation start; a saga completing normally cancels its timer; no leaked timers observable.

## Phase 5 — Compensation delete commands (M)

### atlas-character

- [ ] **5.1** Identify the existing saga-correlated command family in `atlas-character` (same topic as `CreateCharacter`). Cite the file path in the commit message. *(effort: S)*
- [ ] **5.2** Add `RequestDeleteCharacter(transactionId, characterId)` command body + command topic constant (or reuse existing topic if conventions allow). *(effort: S)*
- [ ] **5.3** Add consumer handler: delete the character row and cascade rows; emit saga-correlated status event on success. *(effort: M)*
- [ ] **5.4** Idempotent-on-missing: if the character row does not exist, treat as success (no error, success status event). Add a unit test for this path. *(effort: S)*
- [ ] **5.5** Add processor method (`character/processor.go`) wrapping the delete logic, consistent with existing processor patterns. *(effort: S)*

### atlas-skill

- [ ] **5.6** Identify the existing saga-correlated command family in `atlas-skill`. Cite the file path. *(effort: S)*
- [ ] **5.7** Add `RequestDeleteSkill(transactionId, characterId, skillId)` command body + consumer. *(effort: S)*
- [ ] **5.8** Handler: delete the skill row; emit saga-correlated status event. *(effort: S)*
- [ ] **5.9** Idempotent-on-missing with unit test coverage. *(effort: S)*
- [ ] **5.10** Add processor method wrapping the delete. *(effort: S)*

**Acceptance:** Both services build and test green; both delete commands are idempotent on missing rows; both emit saga-correlated completion status events consumable by the orchestrator's existing correlator.

## Phase 6 — Character-creation reverse-walk compensator (L)

- [ ] **6.1** In `atlas-saga-orchestrator/.../saga/compensator.go:205`, add a new branch in `CompensateFailedStep` keyed on `s.SagaType() == CharacterCreation`, taking precedence over the per-step switch. *(effort: S)*
- [ ] **6.2** Implementation: walk `s.Steps()` in reverse; for each `Status() == Completed` step, dispatch inverse:
  - `AwardAsset` / `AwardItem` → `compP.RequestDestroyItem(transactionId, characterId, templateId, quantity, removeAll=false)`.
  - `CreateAndEquipAsset` → reuse existing destroy logic (`compensator.go:502` path).
  - `CreateSkill` → new `skillP.RequestDeleteSkill(transactionId, characterId, skillId)` from Phase 5.
  - `CreateCharacter` → new `charP.RequestDeleteCharacter(transactionId, characterId)` from Phase 5. Must be last. *(effort: L)*
- [ ] **6.3** Await each compensation step's success-or-failure event before dispatching the next. Preserve causal ordering. *(effort: M)*
- [ ] **6.4** Log compensation-step failures at ERROR with full ids; do NOT abort the chain — next reverse step still runs. *(effort: S)*
- [ ] **6.5** Emit exactly one `StatusEventTypeFailed` at the end of the chain, with `failedStep = <originally-failing-step-id>`, populated `characterId` and `accountId` from `CharacterCreatePayload`. Take the Phase-2 terminal-state guard. *(effort: M)*
- [ ] **6.6** Evict the saga from cache after emission. Cancel any pending Phase-4 timer first. *(effort: S)*
- [ ] **6.7** Preserve existing per-step compensators (`compensateEquipAsset`, `compensateInventoryTransaction`, `compensateStorageOperation`, `compensateSelectGachaponReward`, etc.) — they continue to serve their non-character-creation saga types. *(effort: S)*

**Acceptance:** A character-creation failure produces an ordered reverse-walk of completed steps, emits exactly one Failed event, evicts the saga, leaves the DB in pre-creation state. Other saga types behave identically to today.

## Phase 7 — Factory bridge: failure handler + 10s timeout (S)

- [ ] **7.1** In `atlas-character-factory/.../kafka/consumer/saga/consumer.go`, add `handleSagaFailedEvent` alongside the existing `handleSagaCompletedEvent`. Register via `AdaptHandler`. *(effort: S)*
- [ ] **7.2** Filter: `StatusEventType == Failed && SagaType == CharacterCreation`. Log and drop otherwise. *(effort: S)*
- [ ] **7.3** Extract `AccountId` from `StatusEventFailedBody` (Phase 1.1). Call `FailedEventStatusProvider(accountId, reason)` (Phase 1.5) to emit `FAILED` on `EVENT_TOPIC_SEED_STATUS`. *(effort: S)*
- [ ] **7.4** In the factory's REST handler that creates the saga, pass `timeout: 10 * time.Second` on the outbound command. *(effort: S)*
- [ ] **7.5** Verify no in-flight tracking map is needed — sagaType filter is sufficient (confirmed in PRD §4.4). *(effort: S)*

**Acceptance:** Failure events on `EVENT_TOPIC_SAGA_STATUS` for character-creation sagas are re-emitted as `FAILED` on `EVENT_TOPIC_SEED_STATUS` with correct `accountId`; factory command includes `timeout: 10s`.

## Phase 8 — atlas-login failure handler (S)

- [ ] **8.1** In `atlas-login/.../kafka/consumer/seed/consumer.go`, add a handler for `StatusEventTypeFailed` on the existing `EVENT_TOPIC_SEED_STATUS` subscription. *(effort: S)*
- [ ] **8.2** Resolve session by the envelope's top-level `AccountId`. *(effort: S)*
- [ ] **8.3** Write `AddCharacterEntryWriter(AddCharacterCodeUnknownError)` to the session. *(effort: S)*
- [ ] **8.4** Tolerate disconnected session — log at INFO (`accountId`, `transactionId`) and drop, no panic. *(effort: S)*
- [ ] **8.5** Clear any in-flight creation transaction state held for the session. *(effort: S)*

**Acceptance:** A `FAILED` seed event triggers the client write within the orchestrator's latency budget (11s worst case); a disconnected session is safely dropped.

## Phase 9 — Fix atlas-character error-discard and audit `CreateAndEmit` (M)

- [ ] **9.1** `atlas-character/.../kafka/consumer/character/consumer.go:352` — replace `_, _ = ...CreateAndEmit(...)` with captured error; log at ERROR with `transactionId`, `accountId`, and error. *(effort: S)*
- [ ] **9.2** Audit `character/processor.go` `CreateAndEmit`: enumerate all error return paths. Confirm each path emits a `creationFailedEventProvider` with `transactionId`, `accountId`, and a meaningful `reason`. Line 223 already emits on one path — ensure the others do too. *(effort: M)*
- [ ] **9.3** Where a path returns an error without emitting, add the emit. Target: the saga orchestrator's existing character-status consumer drives `StepCompleted(txId, false)` with a meaningful reason in every case. *(effort: M)*
- [ ] **9.4** Unit test: force each error path in `CreateAndEmit`, assert a `creationFailedEventProvider` is emitted with the expected fields. *(effort: M)*

**Acceptance:** Every `CreateAndEmit` error path emits a correlated character-status failure event; the `consumer.go:352` error is no longer discarded.

## Phase 10 — Tests (L)

### Existing-test updates

- [ ] **10.1** Update `atlas-saga-orchestrator/.../saga/integration_test.go` for new Failed emission and terminal-state semantics. *(effort: M)*
- [ ] **10.2** Update `atlas-saga-orchestrator/.../saga/createandequip_integration_test.go` for the new compensation branch. *(effort: M)*
- [ ] **10.3** Update `saga/mock/processor.go` for timeout-related fields and terminal-state methods. *(effort: S)*
- [ ] **10.4** Sweep per-step processor tests for mocks and assertions affected by new emission semantics. *(effort: M)*

### New unit tests

- [ ] **10.5** Orchestrator: timer fires → Failed emitted with `ErrorCodeSagaTimeout`, compensation runs, single emission. *(effort: M)*
- [ ] **10.6** Orchestrator: concurrent timer fire + `StepCompleted(false)` — exactly one Failed emitted. *(effort: M)*
- [ ] **10.7** Orchestrator: saga consumer `Put()` error → Failed emitted. *(effort: S)*
- [ ] **10.8** Orchestrator: step handler sync error → Failed emitted with correct step id and derived error code. *(effort: S)*
- [ ] **10.9** Orchestrator: async `StepCompleted(false)` for character-creation → Failed emitted. *(effort: S)*
- [ ] **10.10** Orchestrator: reverse-walk compensator dispatches inverses in reverse order; `CreateCharacter` delete is last. *(effort: M)*
- [ ] **10.11** Orchestrator: compensation step failure mid-chain → chain continues; Failed still emitted at end. *(effort: M)*
- [ ] **10.12** `atlas-character`: `RequestDeleteCharacter` idempotent on missing row. *(effort: S)*
- [ ] **10.13** `atlas-skill`: `RequestDeleteSkill` idempotent on missing row. *(effort: S)*
- [ ] **10.14** Factory: `handleSagaFailedEvent` filters by sagaType; re-emits with `accountId`; drops non-CharacterCreation failures. *(effort: S)*
- [ ] **10.15** Login: `FAILED` handler writes `AddCharacterCodeUnknownError` to the session resolved by `accountId`. *(effort: S)*
- [ ] **10.16** Login: `FAILED` for a disconnected session is logged and dropped safely. *(effort: S)*
- [ ] **10.17** `atlas-character`: every `CreateAndEmit` error path emits a `creationFailedEventProvider` with expected fields. *(effort: M)*

**Acceptance:** Every PRD §10 acceptance-criterion bullet has at least one corresponding test; all pass.

## Phase 11 — Build/verify sweep (S)

- [ ] **11.1** `go build ./...` for `atlas-saga-orchestrator`, `atlas-character-factory`, `atlas-character`, `atlas-skill`, `atlas-login`. *(effort: S)*
- [ ] **11.2** `go test ./...` for the same five services. *(effort: S)*
- [ ] **11.3** Docker build for each of the five services (shared-lib additions ripple through imports — per CLAUDE.md, always verify). *(effort: S)*
- [ ] **11.4** Update Kafka-table section of each service's `README.md` for new emit/consume sites:
  - `atlas-saga-orchestrator`: new Failed emission paths; `AccountId` on `StatusEventFailedBody`; `ErrorCodeSagaTimeout`; timeout field on inbound command.
  - `atlas-character-factory`: new `FAILED` emit on `EVENT_TOPIC_SEED_STATUS`; `timeout: 10s` on outbound saga command.
  - `atlas-character`: new `RequestDeleteCharacter` command consumption; fixed error-propagation on `CreateAndEmit`.
  - `atlas-skill`: new `RequestDeleteSkill` command consumption.
  - `atlas-login`: new `FAILED` consumption on `EVENT_TOPIC_SEED_STATUS`. *(effort: S)*
- [ ] **11.5** Manual smoke test: stop `atlas-data`, create a character, confirm client receives `AddCharacterCodeUnknownError` within ~11s; confirm DB is clean; confirm retry with same name succeeds when `atlas-data` is back. *(effort: S)*

**Acceptance:** All five services build, test, and Docker-build green. All README Kafka tables reflect the changes. Manual smoke test passes end-to-end.

## Cross-phase acceptance checklist (mirrors PRD §10)

- [ ] `Saga` model has a `timeout time.Duration` field, populated from command body or defaulted to 30s.
- [ ] `atlas-character-factory` emits character-creation saga commands with `timeout = 10s`.
- [ ] When `award_item` / `create_and_equip_asset` fails because `atlas-data` is unreachable, exactly one `StatusEventTypeFailed` is emitted on `EVENT_TOPIC_SAGA_STATUS` with sagaType, failing step id, and non-empty reason.
- [ ] Compensation restores pre-creation state: character row deleted, items destroyed, skills deleted. Retry with same name succeeds.
- [ ] Factory's `EVENT_TOPIC_SAGA_STATUS` consumer handles `StatusEventTypeFailed` for `CharacterCreation` and re-emits `FAILED` on `EVENT_TOPIC_SEED_STATUS` with `accountId`.
- [ ] Login's seed consumer handles `StatusEventTypeFailed` and writes `AddCharacterCodeUnknownError` resolved by `accountId`.
- [ ] Wedged saga: 10s timeout fires, Failed emitted with `ErrorCodeSagaTimeout`, compensation runs, client receives failure write.
- [ ] The three error-swallow sites are covered by unit tests and no longer silent.
- [ ] Unit tests for: timeout emission, double-emission suppression, saga `Put()` error, step handler error, async `StepCompleted(false)`, factory bridge filter, login session resolution.
- [ ] `go test ./...` and `go build` pass for `atlas-saga-orchestrator`, `atlas-character-factory`, `atlas-login`, `atlas-character`, `atlas-skill`.
- [ ] Service-internal Kafka tables in each `README.md` reflect the new event topics and emit/consume sites.
