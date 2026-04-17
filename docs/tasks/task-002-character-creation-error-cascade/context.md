---
name: Character Creation Error Cascade — Context
description: Key files, decisions, dependencies, and gotchas for closing the character-creation failure loop across orchestrator, factory, character, skill, and login services.
type: context
task: task-002-character-creation-error-cascade
---

# Context — Character Creation Error Cascade

Last Updated: 2026-04-17

## Key Files (current repo, to be touched)

### `atlas-saga-orchestrator`

- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/saga/kafka.go`
  - `StatusEventFailedBody` (~line 36) — **adds** `AccountId uint32 \`json:"accountId"\``.
  - **Adds** `ErrorCodeSagaTimeout = "SAGA_TIMEOUT"` constant.
  - Saga-creation command body — **adds** optional `timeout` field (int milliseconds).
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/saga/consumer.go`
  - Line 49 — `processor.Put()` error currently logged and discarded. **Emit Failed before return.**
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go` (or wherever `Step()` lives)
  - `processor.Step()` sync error path — **emit Failed** instead of returning to the consumer where it is dropped.
  - `processor.Put()` — **schedule per-saga timer** at saga acceptance; attach handle to the cache entry.
  - `StepCompleted(txId, success)` — take the new terminal-state guard; emit Failed on `success == false` for character-creation.
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go`
  - Line 205 — `CompensateFailedStep` switch. **Add new branch** `if s.SagaType() == CharacterCreation { reverseWalkCompensate(...) }` taking precedence over per-step cases.
  - Line 392-444 — `compensateCreateCharacter` (currently a no-op; comment at 409 notes missing delete command). Superseded by the new reverse-walk branch.
  - Line 502 — existing `CreateAndEquipAsset` destroy logic. **Reuse** inside the new branch.
  - Line 795 — `compensateSelectGachaponReward` (only extant reverse-walk). Pattern reference; do not touch.
  - Lines 205-225 — per-step compensator switch (no entries for `AwardAsset`, `CreateSkill` today). Left as-is; new branch handles character-creation path.
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/producer.go`
  - Line 34 — `extractCharacterCreationResults` pattern. **Reuse** for `AccountId`/`CharacterId` extraction from `CharacterCreatePayload` when emitting Failed.
  - **Add Failed-event helper** so all three emission sites go through one place.
- Saga cache (wherever in-memory saga state is tracked)
  - **Add** `timeout time.Duration` and timer handle to entry.
  - **Add** terminal-state machine (`Pending → Compensating → Failed` / `Pending → Completed`) with atomic `TryTransition(from, to)`.
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/integration_test.go` — **update** for new failure semantics.
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/createandequip_integration_test.go` — **update** for new compensation branch.
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/mock/processor.go` — **update** mock surface for timeout/terminal-state fields.
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/README.md` — **update** Kafka table (new Failed emission paths, `AccountId` on failed body, `ErrorCodeSagaTimeout`, timeout field on inbound command).

### `atlas-character-factory`

- `services/atlas-character-factory/atlas.com/character-factory/kafka/message/seed/kafka.go`
  - **Add** `StatusEventTypeFailed = "FAILED"`.
  - **Add** `FailedStatusEventBody` (optional `Reason string` for log correlation).
- `services/atlas-character-factory/atlas.com/character-factory/kafka/producer/seed/producer.go`
  - **Add** `FailedEventStatusProvider(accountId uint32, reason string)` mirroring existing `CreatedEventStatusProvider`.
- `services/atlas-character-factory/atlas.com/character-factory/kafka/consumer/saga/consumer.go` (lines 40-73)
  - **Add** `handleSagaFailedEvent` alongside existing `handleSagaCompletedEvent`. Register both via `AdaptHandler`. Filter by `SagaType == CharacterCreation`.
- REST handler that creates the saga (likely in `rest/` or `character/` handler tree)
  - **Pass** `timeout: 10 * time.Second` on the outbound saga-creation command body.
- `services/atlas-character-factory/atlas.com/character-factory/README.md` — **update** Kafka table (new `FAILED` emit, `timeout: 10s` on outbound command).

### `atlas-login`

- `services/atlas-login/atlas.com/login/kafka/consumer/seed/consumer.go` (lines 43-71)
  - **Add** `StatusEventTypeFailed` handler alongside existing `StatusEventTypeCreated` handler on the same `EVENT_TOPIC_SEED_STATUS` subscription.
  - On FAILED: resolve session by envelope top-level `AccountId`; write `AddCharacterEntryWriter(AddCharacterCodeUnknownError)`; tolerate disconnected session; clear in-flight creation state.
- `services/atlas-login/atlas.com/login/README.md` — **update** Kafka table (new `FAILED` consumption).

### `atlas-character`

- `services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go` (line 352)
  - **Replace** `_, _ = character.NewProcessor(l, ctx, db).CreateAndEmit(c.TransactionId, model)` with captured error; log at ERROR with `transactionId`, `accountId`, error.
- `services/atlas-character/atlas.com/character/character/processor.go`
  - **Audit** `CreateAndEmit`: every error return path must emit `creationFailedEventProvider`. Line 223 already emits on one path — add coverage for others.
- Saga-correlated command family for this service (same topic as `CreateCharacter`)
  - **Add** `RequestDeleteCharacter(transactionId, characterId)` command body.
  - **Add** consumer handler: delete character row + cascade rows; emit saga-correlated status event.
  - **Idempotent on missing row** (accepted success).
  - **Add** processor method wrapping the delete.
- `services/atlas-character/atlas.com/character/README.md` — **update** Kafka table (new `RequestDeleteCharacter` consumption; fixed `CreateAndEmit` error propagation).

### `atlas-skill`

- Saga-correlated command family for this service (same topic as `CreateSkill`)
  - **Add** `RequestDeleteSkill(transactionId, characterId, skillId)` command body.
  - **Add** consumer handler: delete skill row; emit saga-correlated status event.
  - **Idempotent on missing row**.
  - **Add** processor method wrapping the delete.
- `services/atlas-skill/atlas.com/skill/README.md` — **update** Kafka table (new `RequestDeleteSkill` consumption).

### Unchanged but imported / referenced

- `atlas-inventory`: `RequestDestroyItem(transactionId, characterId, templateId, quantity, removeAll)` is reused as-is for `AwardAsset`/`AwardItem` rollback. Already exercised at `saga/compensator.go:502` (`compensateCreateAndEquipAsset`) and `:834` (`compensateSelectGachaponReward`). **No code change.**
- `atlas-data`: no change. Downstream services already convert atlas-data errors into per-domain failure events; this task propagates those failure events up the saga.

## Key Decisions

1. **Backward-compatible wire changes only.** `AccountId` on `StatusEventFailedBody`, `timeout` on saga-creation command body, `StatusEventTypeFailed` + `FailedStatusEventBody` on seed topic are all additive. No topic-rename, no breaking change.
2. **30s default timeout, 10s for character creation.** Default lives in the orchestrator's command decoder. Factory explicitly passes 10s. Any saga legitimately needing >30s must set its own value — we are not preserving indefinite waits.
3. **Reverse-walk compensator is character-creation-specific.** Other saga types continue to use the existing per-step compensators. This avoids destabilizing `InventoryTransaction`, `StorageOperation`, `SelectGachaponReward`, etc.
4. **Delete commands are idempotent on missing rows.** Target absent → success, not error. This sidesteps TOCTOU between orchestrator compensation dispatch and the in-flight original step, and simplifies the orchestrator's chain logic.
5. **Terminal-state guard at the cache level, not per-step.** Step-level `Pending/Completed/Failed` is not sufficient — the timer + `StepCompleted` race happens above the step level. A cache-entry state machine (`Pending → Compensating → Failed`) is the enforcement point. Both transitions and emissions take this guard.
6. **Single generic client error code.** PRD §2 non-goal: distinguishing failure causes to the user. Server-side logs carry `reason` / `errorCode` / `failedStep`; the client gets `AddCharacterCodeUnknownError`.
7. **`accountId` populated from `CharacterCreatePayload`, not from step results.** Failures may occur before `CreateCharacter` has run; `accountId` must be derivable at saga acceptance time. The payload is known before any step runs.
8. **No in-flight map on the factory.** `SagaType == CharacterCreation` filter is sufficient — no other service emits this saga type today. If that changes later, the bridge remains correct because it keys on saga type, not initiator identity.
9. **Late-success leak is accepted for v1 (PRD §4.9).** A downstream service that eventually succeeds after saga timeout will produce orphan side effects. Operator can sweep separately; follow-up task may add saga-aware "bail before persisting" in downstream services.

## Dependencies Between Phases

```
Phase 0 (safety)
   │
   ▼
Phase 1 (wire-format extensions) ──┬──► Phase 7 (factory bridge) ──► Phase 8 (login handler)
   │                               │
   ▼                               │
Phase 2 (terminal-state guard) ────┤
   │                               │
   ▼                               │
Phase 3 (guaranteed Failed emit) ──┤
   │                               │
   ▼                               │
Phase 4 (per-saga timeout) ────────┤
   │                               │
   ▼                               │
Phase 5 (delete commands) ─────────┼──► Phase 9 (atlas-character fixes — parallel with 5/6/7/8)
   │                               │
   ▼                               │
Phase 6 (reverse-walk compensator)─┘
   │
   ▼
Phase 10 (tests — continuous from Phase 3 onward)
   │
   ▼
Phase 11 (build/verify sweep)
```

- **0 → 1 → 2 → 3 → 4** are sequentially load-bearing.
- **5 → 6** (compensator needs the delete commands).
- **7, 8, 9** can parallelize once Phase 1 is in (factory/login consume the new wire shapes; `atlas-character` fixes are independent of orchestrator internals).
- **10** runs continuously from Phase 3 and lands alongside 11.

## Relevant PRD Sections

- §4.1 — Saga timeout (→ Phase 4).
- §4.2 — Guaranteed Failed event emission (→ Phase 3).
- §4.3 — Compensation for character creation (→ Phases 5, 6).
  - §4.3.1 — New cross-service commands (→ Phase 5).
  - §4.3.2 — Character-creation rollback path (→ Phase 6).
  - §4.3.3 — `AccountId` on failed body (→ Phase 1).
- §4.4 — `atlas-character-factory` saga status bridge (→ Phases 1, 7).
- §4.5 — `atlas-login` failure handling (→ Phase 8).
- §4.6 — Fix `atlas-character` discarded error (→ Phase 9).
- §4.7 — Saga terminal-state guard (→ Phase 2, enforced in 3/4/6).
- §4.8 — Idempotency of compensation delete commands (→ Phase 5).
- §4.9 — Late-success leak accepted (→ non-goal).
- §5 — API Surface deltas (→ Phase 1).
- §7 — Service Impact matrix (→ Phases 1, 3, 4, 5, 6, 7, 8, 9).
- §8 — Non-Functional Requirements (→ Phases 10, 11).
- §10 — Acceptance Criteria (→ Phase 10 test coverage + cross-phase checklist in `tasks.md`).

## Gotchas

- **Module names vs. service dir names.** Per CLAUDE.md: module names are short (e.g., `atlas-transports`, not the full path). When adding imports between services or shared-lib consumers, use the `go.mod` module name, not the directory path.
- **`world.Id` is `byte`, `channel.Id` is `byte`, `_map.Id` is `uint32`.** `tenant.Model.Region()` returns `string`, not `world.Id`. Don't conflate when populating `AccountId` / `WorldId` on saga bodies.
- **Read files before editing.** Tool requirement; applies across the five services touched here.
- **Test files reference internal functions.** Renaming handler names (e.g., `handleSagaCompletedEvent` → `handleSagaCompletedAndFailedEvent`) breaks tests. Prefer adding `handleSagaFailedEvent` as a sibling.
- **Shared-lib changes require Docker rebuilds.** Per CLAUDE.md: always verify Docker builds when changing shared libraries. `kafka/message/saga/kafka.go` and `kafka/message/seed/kafka.go` are shared-in-spirit — the new field `AccountId`, new constant `ErrorCodeSagaTimeout`, new event type `FAILED`, and new body `FailedStatusEventBody` ripple through the consumers. Phase 11 covers.
- **Existing `saga/integration_test.go` and per-step tests likely break.** PRD §8 explicit: do not defer. Phase 10 covers.
- **Mock regeneration.** `saga/mock/processor.go` needs timeout + terminal-state additions. Interface-change in the processor means the mock must track.
- **Order of delete dispatch.** In Phase 6, `CreateCharacter` delete MUST be last (deepest reverse step) so item/skill rows referencing the character are removed first — otherwise FK constraints or orphan-check logic in the character service may reject the delete.
- **Tenant scoping.** All new Kafka events carry the standard tenant header. Multi-tenancy is intact today; preserve when adding `handleSagaFailedEvent` (use `tenant.MustFromContext(ctx)` as usual).
- **`AddCharacterEntryWriter(AddCharacterCodeUnknownError)` existence.** Confirm the writer constant exists in `atlas-login` before Phase 8. If not, it must be added — but PRD assumes it exists. Worth a 1-minute grep before Phase 8.

## Baseline Verification Commands

```
# Phase 0 baseline
(cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./...)
(cd services/atlas-character-factory/atlas.com/character-factory && go build ./... && go test ./...)
(cd services/atlas-character/atlas.com/character && go build ./... && go test ./...)
(cd services/atlas-skill/atlas.com/skill && go build ./... && go test ./...)
(cd services/atlas-login/atlas.com/login && go build ./... && go test ./...)

# Phase 11 smoke
# 1. docker-compose stop atlas-data
# 2. Attempt character creation via client
# 3. Observe AddCharacterCodeUnknownError within ~11s
# 4. docker-compose start atlas-data
# 5. Retry same name — should succeed (confirms compensation cleaned up)
```

## Follow-Up Work (explicitly out of scope)

- Same error-swallow audit for adjacent login sagas (delete character, rename).
- Metrics: `saga_failed_total{saga_type, error_code}` counters.
- Saga-aware "bail before persisting" in downstream services to eliminate the §4.9 late-success leak.
- Per-tenant timeout configuration.
- Cross-service integration tests (v1 relies on unit tests).
