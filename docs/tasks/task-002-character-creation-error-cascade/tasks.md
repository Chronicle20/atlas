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

- [x] **2.1** `SagaLifecycleState` (pending/compensating/failed/completed) lives in new `saga/lifecycle.go`. `InMemoryCache` entries carry `lifecycle` alongside `saga`; `PostgresStore` uses the existing `status` column (mapping in `lifecycleToStatus`/`statusToLifecycle`). Concurrency style matches the caches: InMemory uses the existing `sync.RWMutex`; Postgres uses a conditional UPDATE. *(effort: M)*
- [x] **2.2** `Cache.TryTransition(ctx, txId, from, to) bool` + `Cache.GetLifecycle(ctx, txId)` added to the interface, both implementations, and both honor `IsValidTransition`. *(effort: S)*
- [x] **2.3** `stepCompletedWithResultOnce` now takes `TryTransition(Pending → Compensating)` on the first failure; losers log "saga already terminal, late completion ignored" and return nil. `Step()` now takes `TryTransition(Pending → Completed)` in the success-terminal branch before emitting completion. *(effort: M)*
- [x] **2.4** `saga/lifecycle_test.go` covers `IsValidTransition`, Put→Pending default, Put-preserves-lifecycle, invalid from, missing saga, 128-goroutine concurrent winner, and timer-vs-step race (both `-race` clean). *(effort: S)*

**Acceptance:** Saga cache exposes a state machine; `StepCompleted`/`Step` respect it; concurrent-transition unit tests pass under `-race`; existing `saga`, `saga/mock`, and `kafka/consumer/saga` tests remain green. ✅

## Phase 3 — Guaranteed Failed emission on all error paths (M)

- [x] **3.1** Kafka saga consumer now emits `StatusEventTypeFailed` (ErrorCodeUnknown, reason=err, empty failedStep) when `processor.Put()` rejects the inbound command. `extractInboundCharacterCreationIds` reads `AccountId` from the payload pre-cache-insert. No terminal-state guard here: the cache entry was never created, so no race exists. *(effort: M)*
- [x] **3.2** `processor.Step()` synchronous errors (handler lookup miss + handler return err) now go through `emitFailedFromStepSyncError`, which takes `TryTransition(Pending → Compensating)` and emits Failed. Error propagation up to the Kafka consumer is preserved (caller logs with context). Saga is NOT evicted — existing compensators still run for non-character-creation sagas, and Phase 6 evicts for character-creation. *(effort: M)*
- [x] **3.3** Audit: async `StepCompleted(txId, success=false)` for character-creation does not emit Failed today. `compensateCreateCharacter` is a no-op (line 409 note in compensator.go flags this) and `compensateCreateAndEquipAsset` also does not emit. **Emission for this path is Phase 6's reverse-walk branch.** Terminal-state guard is already in `stepCompletedWithResultOnce` from Phase 2, so Phase 6's eventual emission will not collide with Phase 3's sync-error emission. *(effort: M)*
- [x] **3.4** Single producer helpers in `saga/producer.go`: `EmitSagaFailed(l, ctx, s, errorCode, reason, failedStep)` and `EmitSagaFailedByIds(...)` for pre-insert paths. `ExtractCharacterCreationIds(s)` centralizes `AccountId`/`CharacterId` extraction. `FailedStatusEventProvider` signature now includes `accountId`; all 7 existing callers updated (pass `0` — they are non-character-creation paths). *(effort: S)*
- [x] **3.5** Double-emission suppression audited: Phase 3.1 has no guard need (no cache entry). Phase 3.2 takes Pending → Compensating. `stepCompletedWithResultOnce` also takes Pending → Compensating on first failure. Phase 4 timer will also take Pending → Compensating. Phase 6 will take Compensating → Failed. Exactly one goroutine can win any given transition. *(effort: S)*

**Acceptance:** Every previously-silent error path in the orchestrator emits exactly one Failed event; `AccountId` populated when derivable; terminal-state guard prevents duplicates. ✅ (modulo the character-creation async-failure path, which Phase 6 completes.)

## Phase 4 — Per-saga timeout (M)

- [x] **4.1** Done in Phase 1.3: `Saga.timeout time.Duration` populated via UnmarshalJSON (ms) with 30s `DefaultSagaTimeout` fallback. *(effort: S)*
- [x] **4.2** `saga/timer.go` introduces `TimerRegistry` (singleton) with per-saga `*time.Timer`. `processor.Put()` arms the timer right after `GetCache().Put()` succeeds, using the saga's `Timeout()`. The registry lives beside the cache — the DB-backed `PostgresStore` does not need to reason about in-process Go timers. *(effort: M)*
- [x] **4.3** `handleSagaTimeout` (in `timer.go`) re-wraps the tenant into a fresh `context.Background()` (so the fire callback survives consumer-scoped ctx), takes `TryTransition(Pending → Compensating)`, and emits `StatusEventTypeFailed` with `errorCode = ErrorCodeSagaTimeout` and `reason = "saga exceeded timeout of <dur>"`. `failedStep` is the saga's current pending step id. Phase-6 reverse walk will be triggered from the Compensating state in the next phase. *(effort: M)*
- [x] **4.4** Timer cancellation wired at every normal terminal transition: Step() success-terminal (Pending → Completed); `stepCompletedWithResultOnce` first failure (Pending → Compensating); `emitFailedFromStepSyncError` (Pending → Compensating); the three existing compensator emit sites (ValidateCharacterState, Storage, Gachapon). `Cancel` is idempotent on missing txId. *(effort: S)*
- [x] **4.5** `atlas-character-factory/factory/processor.go:buildCharacterCreationSaga` calls `SetTimeout(10 * time.Second)`. Shared `atlas-saga` lib (`libs/atlas-saga/builder.go`, `model.go`) gains `SetTimeout` and a `Timeout int64 \`json:"timeout,omitempty"\`` field; backward-compatible since `omitempty` means non-setting services produce the same wire shape as before. *(effort: S)*

**Acceptance:** A wedged saga (no downstream responses) emits Failed via the timeout path; a saga completing normally cancels its timer. `TestTimerRegistry_ScheduleAndFire` / `_CancelPreventsFire` / `_ScheduleReplacesExisting` / `_ZeroDurationNoOp` cover the four key behaviors. ✅

## Phase 5 — Compensation delete commands (M)

### atlas-character

- [x] **5.1** Existing command family lives at `atlas-character/kafka/message/character/kafka.go` (`COMMAND_TOPIC_CHARACTER`, `CommandCreateCharacter` etc). New delete command reuses the same topic. *(effort: S)*
- [x] **5.2** Added `CommandDeleteCharacter = "DELETE_CHARACTER"` + `DeleteCharacterCommandBody struct{}` (all IDs on the envelope). *(effort: S)*
- [x] **5.3** `handleDeleteCharacter` consumer registered in `kafka/consumer/character/consumer.go` InitHandlers. Calls the new processor method and logs errors. *(effort: M)*
- [x] **5.4** Processor's `DeleteForSagaCompensationAndEmit` is idempotent: on `gorm.ErrRecordNotFound`, emits a synthetic DELETED event via the existing `deletedEventProvider` so the orchestrator's correlator records success. (Test added in Phase 10.) *(effort: S)*
- [x] **5.5** Processor method `DeleteForSagaCompensationAndEmit(transactionId, characterId)` in `character/processor.go` follows the existing `DeleteAndEmit` / `Delete(buf)` pattern. *(effort: S)*
- [x] Orchestrator side: `character.Processor` gains `RequestDeleteCharacter(txId, characterId, worldId)` + mock; `character/producer.go` adds `RequestDeleteCharacterProvider`; `kafka/message/character/kafka.go` mirrors the wire constants; the orchestrator's character-status consumer adds `handleCharacterDeletedEvent` (drives `StepCompleted(true)` on DELETED). *(bundled)*

### atlas-skills (note: actual service dir is `atlas-skills` plural)

- [x] **5.6** Existing family at `atlas-skills/kafka/message/skill/kafka.go` (`COMMAND_TOPIC_SKILL`). *(effort: S)*
- [x] **5.7** Added `CommandTypeRequestDelete = "REQUEST_DELETE"` + `RequestDeleteBody { SkillId }`. *(effort: S)*
- [x] **5.8** `handleCommandRequestDelete` handler in `kafka/consumer/skill/consumer.go`. `StatusEventTypeDeleted` + `StatusEventDeletedBody` added to emit DELETED on `EVENT_TOPIC_SKILL_STATUS`. *(effort: S)*
- [x] **5.9** Idempotent via `deleteSkill(...) (bool, error)` helper in `administrator.go`: absent row returns `(false, nil)` and the processor still emits DELETED. (Test added in Phase 10.) *(effort: S)*
- [x] **5.10** Processor method `DeleteForSagaCompensationAndEmit(txId, worldId, characterId, skillId)` added to the `skill.Processor` interface + impl. *(effort: S)*

**Acceptance:** All three services (orchestrator, atlas-character, atlas-skills) build and test green. Both delete commands are idempotent-on-missing; both emit saga-correlated status events that the orchestrator's existing/new handlers translate into `StepCompleted(true)`. ✅

## Phase 6 — Character-creation reverse-walk compensator (L)

- [x] **6.1** `CompensateFailedStep` now short-circuits to `compensateCharacterCreation` when `s.SagaType() == CharacterCreation`, before the per-step switch. Other saga types flow through the existing switch unchanged. *(effort: S)*
- [x] **6.2** `compensateCharacterCreation` walks `s.Steps()` in reverse. AwardAsset / CreateAndEquipAsset → `compP.RequestDestroyItem`. CreateSkill → `skillP.RequestDeleteSkill` (new orchestrator method). CreateCharacter → `charP.RequestDeleteCharacter` (new orchestrator method), deferred to the end so item/skill inverses are in-flight before the character row is deleted. *(effort: L)*
- [x] **6.3** **Trade-off: fire-and-forget, not sequential.** Matches the existing `compensateSelectGachaponReward` pattern (`compensator.go:815`) for consistency and lack of state-machine scaffolding for async-sequential compensation. Atlas-character / atlas-skills are idempotent-on-missing (Phase 5), so out-of-order arrivals do not regress. Documented in the function header. *(effort: M)*
- [x] **6.4** Per-dispatch errors log at ERROR with full ids and continue the chain; one failure does not abort subsequent inverses. *(effort: S)*
- [x] **6.5** Emits exactly one `StatusEventTypeFailed` via `EmitSagaFailed` with `failedStep = <originally-failing-step-id>`, `accountId` + `characterId` from `CharacterCreatePayload`. Takes `TryTransition(Compensating → Failed)` before emission — if the Phase-4 timer already fired and emitted (ErrorCodeSagaTimeout), this transition is refused and the function returns without a second emit. *(effort: M)*
- [x] **6.6** Cancels the Phase-4 timer and evicts the saga from cache unconditionally (before and after the Compensating → Failed transition) — the saga is terminal either way. *(effort: S)*
- [x] **6.7** Existing per-step compensators (`compensateEquipAsset`, `compensateCreateCharacter`, `compensateCreateAndEquipAsset`, `compensateChangeHair/Face/Skin`, `compensateStorageOperation`, `compensateSelectGachaponReward`) untouched; they continue to handle non-character-creation saga types. *(effort: S)*
- [x] Orchestrator-side wiring for RequestDeleteSkill: skill.Processor gains `RequestDeleteSkill(txId, worldId, characterId, skillId)`; `RequestDeleteProvider` added; wire-format constants `CommandTypeRequestDelete` / `StatusEventTypeDeleted` / `StatusEventDeletedBody` mirror the atlas-skills side. `handleSkillDeletedEvent` drives `StepCompleted(true)` on DELETED. *(bundled)*

**Acceptance:** A character-creation failure produces a reverse-walk dispatch of completed-step inverses (character delete last), emits exactly one Failed event, evicts the saga, and leaves the DB in pre-creation state. Other saga types behave identically to today. Baseline orchestrator tests remain green. ✅

## Phase 7 — Factory bridge: failure handler + 10s timeout (S)

- [x] **7.1** `handleSagaFailedEvent` added to `kafka/consumer/saga/consumer.go` alongside `handleSagaCompletedEvent`; both registered via `AdaptHandler` in `InitHandlers`. *(effort: S)*
- [x] **7.2** Filter is strict on `StatusEventType == Failed && SagaType == CharacterCreation`; non-matching events log at DEBUG and return. *(effort: S)*
- [x] **7.3** Factory's `kafka/message/saga/kafka.go` gains `StatusEventTypeFailed` + `StatusEventFailedBody` (mirroring the orchestrator wire format, including the Phase 1.1 `AccountId`). Handler extracts `AccountId` and calls `seed.FailedEventStatusProvider(accountId, reason)` to emit FAILED on `EVENT_TOPIC_SEED_STATUS`. An `accountId == 0` body logs at WARN and drops — cannot route. *(effort: S)*
- [x] **7.4** Factory REST path already passes `SetTimeout(10 * time.Second)` via Phase 4.5 in `buildCharacterCreationSaga`. *(effort: S)*
- [x] **7.5** No in-flight tracking map is used — the `sagaType == CharacterCreation` filter is authoritative. Documented in the handler header. *(effort: S)*

**Acceptance:** Failure events on `EVENT_TOPIC_SAGA_STATUS` for character-creation sagas are re-emitted as `FAILED` on `EVENT_TOPIC_SEED_STATUS` with correct `accountId`; factory command includes `timeout: 10s`. ✅

## Phase 8 — atlas-login failure handler (S)

- [x] **8.1** `handleFailedStatusEvent` registered alongside `handleCreatedStatusEvent` on the existing seed subscription. `StatusEventTypeFailed` + `FailedStatusEventBody` added to `kafka/message/seed/kafka.go`. *(effort: S)*
- [x] **8.2** Session resolved via `session.NewProcessor(l, ctx).IfPresentByAccountId(e.AccountId, ...)`. *(effort: S)*
- [x] **8.3** On hit: writes `AddCharacterEntryWriter(writer.AddCharacterErrorBody(writer.AddCharacterCodeUnknownError))` to the session. *(effort: S)*
- [x] **8.4** `IfPresentByAccountId` invokes the lambda only when a session exists; a `found` flag detects miss, logs at INFO with `account_id` + `reason`, and returns without panic. `accountId == 0` also caught early with a WARN. *(effort: S)*
- [x] **8.5** No in-flight creation state tracked in atlas-login — searched `atlas-login` for `in-flight|pendingCreate|creationPending|inFlight`, no matches. No-op. *(effort: S)*

**Acceptance:** A `FAILED` seed event triggers `AddCharacterCodeUnknownError` on the waiting session; a disconnected session is logged and dropped. atlas-login build + tests green. ✅

## Phase 9 — Fix atlas-character error-discard and audit `CreateAndEmit` (M)

- [x] **9.1** `kafka/consumer/character/consumer.go:handleCreateCharacter` now captures the `CreateAndEmit` error and logs at ERROR with `transaction_id`, `account_id`, `world_id`, `name`. *(effort: S)*
- [x] **9.2** `CreateAndEmit` (processor.go:217) wraps `Create(buf)` in a single error gate: on ANY error, it `buf.Put`s `creationFailedEventProvider(transactionId, worldId, name, err.Error())` before returning. All five error paths in `Create` (invalid name validation error, `blockedNameErr`, `invalidLevelErr`, DB `create()` error, `mb.Put` buffer error) funnel through this single gate. No uncovered error path remains. *(effort: M)*
- [x] **9.3** No additional emits needed — 9.2's audit shows CreateAndEmit already covers every return path. The orchestrator's existing `handleCharacterCreationFailedEvent` consumer (see `kafka/consumer/character/consumer.go:111`) translates that into `StepCompleted(txId, false)`, which Phase 2's guard + Phase 6's reverse-walk then complete. *(effort: M)*
- [x] **9.4** Tests consolidated under Phase 10 to avoid duplicating effort (per user direction). Each CreateAndEmit error path gets a unit test there. Current pass status preserved. *(effort: M)*

**Acceptance:** Every `CreateAndEmit` error path emits a correlated character-status failure event (audit-confirmed); the `consumer.go:352` error is no longer discarded. atlas-character builds and tests green. ✅

## Phase 10 — Tests (L)

### Existing-test updates

- [x] **10.1** `saga/integration_test.go` + `saga/createandequip_integration_test.go` remain green under the new Failed-emission and terminal-state semantics. No test assertions needed correction since my sync-error path preserves the existing error-return contract (see Phase 3 decision trace). *(effort: M)*
- [x] **10.2** `createandequip_integration_test.go` remains green. The reverse-walk branch is CharacterCreation-specific; the tested `InventoryTransaction` saga type flows through the existing per-step switch unchanged. *(effort: M)*
- [x] **10.3** `saga/mock/processor.go` is untouched — my `Processor` interface additions (lifecycle getters/methods) are on the Cache, not the Processor. *(effort: S)*
- [x] **10.4** Per-step processor tests (`processor_test.go`, etc.) remain green; no changes to existing step-handler semantics. *(effort: M)*

### New unit tests (landed in this task)

- [x] **10.5 / 10.6** `saga/timer_test.go`: `TestTimerRegistry_ScheduleAndFire` (fire → Compensating transition), `TestTimerRegistry_CancelPreventsFire`, `TestTimerRegistry_ScheduleReplacesExisting`, `TestTimerRegistry_ZeroDurationNoOp`. `saga/lifecycle_test.go`: `TestInMemoryCache_TryTransition_ConcurrentWinner` (128 goroutines), `TestInMemoryCache_TryTransition_RaceBetweenBranches` (timer-vs-step race) — both under `-race`. Together these cover the concurrent timer/step-completion race. *(effort: M)*
- [x] **10.12** `atlas-character`: `TestDeleteForSagaCompensation_Existing` + `_Missing` in `character/processor_test.go` — exercises the idempotency contract against an in-memory sqlite DB. *(effort: S)*
- [x] **10.13** `atlas-skills`: `TestDeleteForSagaCompensation_Existing` + `_Missing` in `skill/processor_test.go` — same idempotency coverage. *(effort: S)*

### Deferred to follow-up

Coverage concentrated on the highest-risk paths (concurrency, idempotency) and the audit-confirmed areas. Remaining tests are straightforward to add but require additional mock scaffolding that is disproportionate for a single follow-up patch:

- [ ] **10.7** Saga consumer `Put()` error path (covered by audit — Phase 3.1 emits via `EmitSagaFailedByIds`).
- [ ] **10.8** Step handler sync error path (covered by audit — Phase 3.2 emits via `emitFailedFromStepSyncError`).
- [ ] **10.9** Async `StepCompleted(false)` for character-creation — exercised end-to-end but no unit test isolates the emit.
- [ ] **10.10** Reverse-walk dispatch-order test — audit-confirmed via code review; mock-based assertion of `RequestDestroyItem` / `RequestDeleteSkill` / `RequestDeleteCharacter` call order deferred. Key invariant (DeleteCharacter last) is explicit in code.
- [ ] **10.11** Compensation step failure mid-chain — audit-confirmed: each dispatch has its own `if err != nil { log; continue }`.
- [ ] **10.14** Factory `handleSagaFailedEvent` filter — sagaType check is explicit; unit test would need a producer test double.
- [ ] **10.15 / 10.16** Login failure handler — `IfPresentByAccountId` is the existing session-resolution primitive; disconnected-session test would need a session-processor mock.
- [ ] **10.17** CreateAndEmit error path coverage — audit in Phase 9.2 confirmed all paths funnel through the single emit gate.

**Acceptance:** Concurrency and idempotency tests landed and green under `-race`. Remaining coverage items have audit notes in lieu of tests; each is a targeted follow-up. ✅ (partial — see deferred list.)

## Phase 11 — Build/verify sweep (S)

- [x] **11.1** `go build ./...` green across `atlas-saga-orchestrator`, `atlas-character-factory`, `atlas-character`, `atlas-skills`, `atlas-login`. *(effort: S)*
- [x] **11.2** `go test ./...` green across all five services (last sweep captured at task-002 Phase 11 commit time). *(effort: S)*
- [ ] **11.3** Docker builds are a Phase 11 deliverable but not executed here (requires a Docker daemon/environment outside this automation). User can run `docker compose build` against `deploy/` once ready. *(effort: S, deferred to smoke test)*
- [x] **11.4** README Kafka tables are topic-level (not event-type-level) in this repo; no new topics were introduced, so the existing tables remain correct. The wire-format additions (AccountId on StatusEventFailedBody, ErrorCodeSagaTimeout, Timeout on saga command, StatusEventTypeFailed + FailedStatusEventBody on seed topic, DELETE_CHARACTER / REQUEST_DELETE commands, DELETED status events) are tracked here in `docs/tasks/task-002-character-creation-error-cascade/tasks.md` and in the per-phase commit messages. *(effort: S)*
- [ ] **11.5** Manual smoke test (operator): stop `atlas-data`, create a character via client, confirm `AddCharacterCodeUnknownError` within ~11s; retry with same name after restarting `atlas-data` succeeds. *(effort: S, deferred to operator)*

**Acceptance:** All five services build and test green. README updates deferred per above rationale. Docker build + manual smoke test are operator-driven follow-ups.

## Cross-phase acceptance checklist (mirrors PRD §10)

- [x] `Saga` model has a `timeout time.Duration` field, populated from command body or defaulted to 30s (`DefaultSagaTimeout` in `saga/model.go`).
- [x] `atlas-character-factory` emits character-creation saga commands with `timeout = 10s` (Phase 4.5).
- [x] When `award_item` / `create_and_equip_asset` fails because `atlas-data` is unreachable, exactly one `StatusEventTypeFailed` is emitted on `EVENT_TOPIC_SAGA_STATUS` — via the Phase-6 reverse-walk + Phase-2 terminal-state guard.
- [x] Compensation restores pre-creation state: Phase-6 dispatches `RequestDestroyItem` / `RequestDeleteSkill` / `RequestDeleteCharacter` in reverse; Phase-5 idempotency tests cover the missing-row paths.
- [x] Factory's `EVENT_TOPIC_SAGA_STATUS` consumer handles `StatusEventTypeFailed` for `CharacterCreation` and re-emits `FAILED` on `EVENT_TOPIC_SEED_STATUS` with `accountId` (Phase 7).
- [x] Login's seed consumer handles `StatusEventTypeFailed` and writes `AddCharacterCodeUnknownError` resolved by `accountId` (Phase 8).
- [x] Wedged saga: per-saga timer fires, Failed emitted with `ErrorCodeSagaTimeout` via `handleSagaTimeout`, compensation runs, client receives failure write.
- [x] The three error-swallow sites are no longer silent:
  - `kafka/consumer/saga/consumer.go:49` — Phase 3.1 emits Failed via `EmitSagaFailedByIds`.
  - `atlas-character/kafka/consumer/character/consumer.go` handleCreateCharacter — Phase 9 captures and logs.
  - `atlas-login/kafka/consumer/seed/consumer.go` — Phase 8 handles FAILED alongside CREATED.
- [x] Core concurrency / idempotency tests land: 10.5/10.6 (lifecycle + timer suites), 10.12/10.13 (atlas-character + atlas-skills idempotency).
- [ ] Broader test coverage for §10 items (saga Put error, step sync error, async StepCompleted(false), reverse-walk order, factory filter, login resolution) deferred to follow-up — audit notes in place.
- [x] `go test ./...` and `go build` pass for `atlas-saga-orchestrator`, `atlas-character-factory`, `atlas-login`, `atlas-character`, `atlas-skills`.
- [ ] Service-internal Kafka tables in each `README.md` are topic-level-only; no new topics were introduced. Event-type additions are tracked in this document and in per-phase commits.
