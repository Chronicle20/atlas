---
name: Character Creation Error Cascade — Implementation Plan
description: Phased plan to close the character-creation failure loop end-to-end (orchestrator emission + timeout, compensation, factory bridge, login write, and the three error-swallow fixes).
type: plan
task: task-002-character-creation-error-cascade
---

# Character Creation Error Cascade — Implementation Plan

Last Updated: 2026-04-17
Companion docs: `prd.md`, `tasks.md`, `context.md`

## Executive Summary

When character creation fails — most visibly when `atlas-data` is unreachable during `award_item` / `create_and_equip_asset` — no failure ever reaches the client: the socket waits forever for a `CREATED` event that never arrives. Three concrete error-swallow sites are responsible: the saga command consumer drops `processor.Put()` errors (`atlas-saga-orchestrator/.../saga/consumer.go:49`), the `atlas-character` create consumer discards `CreateAndEmit` with `_, _` (`atlas-character/.../character/consumer.go:352`), and the `atlas-login` seed consumer subscribes only to `CREATED` (`atlas-login/.../seed/consumer.go:43-71`). The orchestrator's compensator is also materially incomplete: `compensateCreateCharacter` does no rollback, `AwardAsset`/`CreateSkill` have no compensator entries, and `CompensateFailedStep` only compensates the failed step — not the chain of already-completed prior steps.

This task closes the loop. Every failure mode emits exactly one `StatusEventTypeFailed` saga event, compensation walks the completed steps in reverse to restore pre-creation state, the factory bridge re-emits failure to `EVENT_TOPIC_SEED_STATUS`, and `atlas-login` writes `AddCharacterEntryWriter(AddCharacterCodeUnknownError)` to the session. A caller-supplied per-saga timeout (10s for character creation; 30s default) is added as a backstop so wedged sagas fail closed. The work spans five services: `atlas-saga-orchestrator`, `atlas-character-factory`, `atlas-character`, `atlas-skill`, and `atlas-login`.

**No REST contract, Kafka topic, or DB schema is introduced.** `EVENT_TOPIC_SAGA_STATUS`, `COMMAND_TOPIC_SAGA`, and `EVENT_TOPIC_SEED_STATUS` are extended with new event types / optional body fields (backward compatible). Two new saga-correlated compensation commands are added to `atlas-character` and `atlas-skill`.

## Current State Analysis

**Fan-out today.** Client TCP socket → `atlas-login` → REST 202 to `atlas-character-factory` → `atlas-saga-orchestrator` saga command (`COMMAND_TOPIC_SAGA`) → per-step fan-out to `atlas-character`, `atlas-inventory`, `atlas-skill`, with several of those relying on `atlas-data` for template/wz lookups. Success path emits `StatusEventTypeCompleted` on `EVENT_TOPIC_SAGA_STATUS`; the factory's existing saga-status bridge re-emits `CREATED` on `EVENT_TOPIC_SEED_STATUS`; the login seed consumer hands the session back to the client.

**Three confirmed error-swallow sites.**
1. `atlas-saga-orchestrator/.../kafka/consumer/saga/consumer.go:49` — logs and drops `processor.Put()` errors.
2. `atlas-character/.../kafka/consumer/character/consumer.go:352` — `_, _ = ...CreateAndEmit(...)` discards both return values.
3. `atlas-login/.../kafka/consumer/seed/consumer.go:43-71` — subscribes only to `StatusEventTypeCreated`, no failure handler, no timeout backstop.

**Compensator gaps (verified).**
- `compensateCreateCharacter` (`compensator.go:392-444`) is a no-op; the comment at line 409 flags the missing delete command.
- `AwardAsset` and `CreateSkill` have no compensator entries (switch at `compensator.go:205-225`).
- `CompensateFailedStep` only compensates the failing step; only `compensateSelectGachaponReward` (line 795) walks completed steps in reverse — ad-hoc, not a reusable pattern.
- Per-step compensators (`compensateEquipAsset`, `compensateCreateCharacter`, `compensateCreateAndEquipAsset`, `compensateChangeHair/Face/Skin`) mark the step Pending and continue; they do not emit Failed. Only `ValidateCharacterState` (line 191), `compensateStorageOperation` (line 768), and `compensateSelectGachaponReward` (line 850) emit Failed today.

**Factory bridge already exists for success.** `atlas-character-factory/.../kafka/consumer/saga/consumer.go:40-73` consumes `EVENT_TOPIC_SAGA_STATUS`, filters `SagaType == CharacterCreation`, and emits `CREATED` to `EVENT_TOPIC_SEED_STATUS` via `CreatedEventStatusProvider`. No failure handler exists.

**Timeout absent.** The `Saga` model has no timeout field; the orchestrator has no per-saga timer; `COMMAND_TOPIC_SAGA` has no timeout on its command body; the factory passes none.

**`accountId` missing from failed body.** `StatusEventFailedBody` (`kafka/message/saga/kafka.go:36`) carries `characterId` but no `accountId`. Login resolves sessions by `accountId`, and failures may occur before a `characterId` is ever known.

## Proposed Future State

### Wire-format deltas (all backward compatible)

- `StatusEventFailedBody` gains `AccountId uint32 \`json:"accountId"\`` — zero for non-character-creation sagas.
- `ErrorCodeSagaTimeout = "SAGA_TIMEOUT"` constant added next to existing error codes in `kafka/message/saga/kafka.go`.
- `COMMAND_TOPIC_SAGA` saga-creation body gains optional `timeout` field (integer milliseconds). Missing/zero → default 30s.
- `EVENT_TOPIC_SEED_STATUS` (owned by factory, consumed by login) gains `StatusEventTypeFailed = "FAILED"` and a `FailedStatusEventBody` (with optional `Reason string` for log correlation).

### New Kafka commands (compensation)

- **`atlas-character`**: saga-correlated `RequestDeleteCharacter(transactionId, characterId)` command. Consumer deletes the character row and cascade rows, emits a status event the orchestrator correlates via `StepCompleted`. **Idempotent on missing rows** — treat absent row as success, no error.
- **`atlas-skill`**: saga-correlated `RequestDeleteSkill(transactionId, characterId, skillId)` command. Same contract, same idempotency guarantee.
- `atlas-inventory.RequestDestroyItem` is reused as-is for `AwardAsset` / `AwardItem` rollback — already exercised by `compensateCreateAndEquipAsset` (line 502) and `compensateSelectGachaponReward` (line 834).

### Orchestrator model & behavior deltas

- `Saga` model: new `timeout time.Duration` field. Default-populated to 30s at command-consumption time if absent/zero.
- Per-saga timer scheduled at acceptance, cancelled at terminal state.
- **Saga terminal-state guard** (`Pending → Compensating → Failed` or `Pending → Completed`) on the cache entry. Both the timer and `StepCompleted` atomically check-and-mark; a loser logs "already terminal, no-op".
- **Guaranteed Failed emission** on all four previously-silent paths: consumer `Put()` error, step handler sync error, async `StepCompleted(false)`, timeout.
- **`CharacterCreation`-specific compensation branch** in `CompensateFailedStep` (`compensator.go:205`) that walks `s.Steps()` in reverse, inverts each completed step (`AwardAsset`/`AwardItem` → `RequestDestroyItem`, `CreateAndEquipAsset` → reuse existing destroy logic, `CreateSkill` → `RequestDeleteSkill`, `CreateCharacter` → `RequestDeleteCharacter` last), awaits each, emits exactly one Failed at the end, and evicts the saga from cache.

### Factory, login, and character-service deltas

- **`atlas-character-factory`** extends its existing saga-status consumer with a `handleSagaFailedEvent` handler, filters `SagaType == CharacterCreation`, calls a new `FailedEventStatusProvider(accountId, reason)` that emits `FAILED` on `EVENT_TOPIC_SEED_STATUS`. Passes `timeout: 10s` on saga creation.
- **`atlas-login`** adds a `StatusEventTypeFailed` handler on the existing `EVENT_TOPIC_SEED_STATUS` subscription; resolves the session by `accountId`; writes `AddCharacterEntryWriter(AddCharacterCodeUnknownError)`; tolerates already-disconnected sessions.
- **`atlas-character`** stops discarding the `CreateAndEmit` error at `consumer.go:352`; `CreateAndEmit` is audited so every error path emits a `creationFailedEventProvider` with `transactionId`, `accountId`, and `reason`.

### Key invariants

- Exactly one `StatusEventTypeFailed` per non-completing saga. Terminal-state guard enforces this under timer/handler races.
- Compensation runs to completion even if an individual compensation step fails; failures are logged (ERROR) but do not abort the chain.
- Late-success events after a timed-out saga is evicted are an accepted v1 leak (§4.9 of PRD) — no orphan sweeper in this task.
- Multi-tenancy: all new events carry the standard tenant header. No tenant-scoped in-flight map is required on the factory side since the sagaType filter is sufficient (no other service emits `CharacterCreation` today).

## Implementation Phases

Phase ordering is load-bearing. Each phase leaves the tree in a build-passing state; intermediate phases may add emission without yet being consumed, but nothing is wired in reverse.

### Phase 0 — Safety rails (S)
Feature branch off `main`. Baseline build & test of the five affected services. Confirm no in-flight edits in the saga orchestrator / factory / character / skill / login trees.

### Phase 1 — Wire-format extensions, no behavior change (S)
Add `AccountId` to `StatusEventFailedBody`. Add `ErrorCodeSagaTimeout` constant. Add optional `timeout` to the saga-creation command body with a 30s default at decode time. Add `StatusEventTypeFailed` + `FailedStatusEventBody` to `atlas-character-factory`'s `kafka/message/seed/kafka.go`. Add `FailedEventStatusProvider` in the factory's seed producer. These are pure additions; nothing consumes or emits the new shapes yet.

### Phase 2 — Orchestrator terminal-state guard (M)
Introduce a small state machine on the saga cache entry (`Pending → Compensating → Failed` / `Pending → Completed`). All transitions are atomic (mutex on the cache entry or `sync/atomic` on a status field, whichever fits the existing cache shape). Callers of `StepCompleted` and the forthcoming timer path take this guard. Until Phases 3–5 land, nothing new drives transitions; this phase is pure groundwork and is validated by existing tests plus a small unit test that exercises concurrent double-transition attempts.

### Phase 3 — Guaranteed Failed emission on all error paths (M)
Emit `StatusEventTypeFailed` at the three silent sites in the orchestrator:
1. `kafka/consumer/saga/consumer.go:49` — `Put()` error → emit Failed with `errorCode = ErrorCodeUnknown`, `reason = err.Error()`, empty `failedStep` (saga never entered step execution).
2. `processor.Step()` — step-handler sync errors → emit Failed with the failing step's id, handler-supplied reason, derived `errorCode` (default `ErrorCodeUnknown`).
3. Async `StepCompleted(txId, success=false)` — emit Failed with the current step id, reason from the upstream domain event body, `errorCode` where available. (Confirm whether existing branches already emit; add where missing.)
All three go through the Phase-2 terminal-state guard. `extractCharacterCreationResults` pattern (`producer.go:34`) is reused to populate `characterId` and `accountId` where available.

### Phase 4 — Per-saga timeout (M)
Add `timeout time.Duration` to `Saga`. Orchestrator reads it from the inbound command body (default 30s). Schedule a per-saga `time.AfterFunc` (or equivalent) at acceptance. On fire: take the terminal-state guard; if still `Pending`, mark `Compensating`, set the current step `Failed` with reason `"saga timed out"`, drive compensation (Phase 5), emit Failed with `errorCode = ErrorCodeSagaTimeout` and `reason = "saga exceeded timeout of <N>s"`. Cancel the timer on normal terminal path. The double-emission guard is the terminal-state check from Phase 2.

### Phase 5 — Compensation delete commands (M)
Add `RequestDeleteCharacter(transactionId, characterId)` command to `atlas-character`: Kafka command + consumer + processor method + completion status event. Add `RequestDeleteSkill(transactionId, characterId, skillId)` command to `atlas-skill`: same shape. Both are **idempotent against missing rows** — absent target is success, not error. Both emit a saga-correlated status event that the orchestrator's existing correlator can turn into `StepCompleted(txId, true)`. Follow each service's existing saga-correlated command conventions (same topic family as `CreateCharacter` / `CreateSkill`, whichever already exists).

### Phase 6 — Character-creation reverse-walk compensator (L)
New branch in `CompensateFailedStep` (`compensator.go:205`) keyed on `s.SagaType() == CharacterCreation`, taking precedence over the per-step switch. Walks `s.Steps()` in reverse, dispatches inverse actions (`AwardAsset`/`AwardItem` → `RequestDestroyItem`; `CreateAndEquipAsset` → reuse existing `compensator.go:502` destroy path; `CreateSkill` → new `RequestDeleteSkill`; `CreateCharacter` → new `RequestDeleteCharacter`, always last), awaits each step's completion event before proceeding, logs compensation failures at ERROR without aborting the chain, emits exactly one `StatusEventTypeFailed` at the end (with `failedStep` = originally-failing step), and evicts the saga from cache. Preserve existing per-step compensators — other saga types still use them.

### Phase 7 — Factory bridge: failure handler + 10s timeout (S)
Extend the factory's existing `EVENT_TOPIC_SAGA_STATUS` consumer with `handleSagaFailedEvent`, registered alongside `handleSagaCompletedEvent` via `AdaptHandler`. Filter `SagaType == CharacterCreation`. Extract `AccountId` from the Phase-1 `StatusEventFailedBody.AccountId`. Emit `FAILED` on `EVENT_TOPIC_SEED_STATUS` via the Phase-1 `FailedEventStatusProvider`. In the factory's REST saga-creation path, pass `timeout: 10 * time.Second` on the outbound saga command.

### Phase 8 — `atlas-login` failure handler (S)
Add a `StatusEventTypeFailed` handler on the existing seed consumer (`kafka/consumer/seed/consumer.go`). Resolve the session by top-level `AccountId` on `seed.StatusEvent[E]`. Write `AddCharacterEntryWriter(AddCharacterCodeUnknownError)`. Tolerate disconnected session — log and drop, no panic. Clear any in-flight creation state for that session.

### Phase 9 — Fix `atlas-character` error-discard and audit `CreateAndEmit` (M)
Stop discarding at `consumer.go:352` — capture and log the error. Audit `character/processor.go` `CreateAndEmit`: every early-return error path must emit a `creationFailedEventProvider` carrying `transactionId`, `accountId`, and a reason sufficient for the saga orchestrator's existing character-status consumer to drive `StepCompleted(txId, false)`. Line 223 emits on one path today — ensure coverage on the others.

### Phase 10 — Tests (L)
Update existing orchestrator tests (`saga/integration_test.go`, `saga/createandequip_integration_test.go`) and per-step processor tests under the new failure/terminal-state semantics. Expand `saga/mock/processor.go` for timeout/terminal-state fields. Add unit tests covering:
- timer fires → Failed emitted, compensation runs, single emission;
- double-emission suppressed (concurrent timer + late `StepCompleted`);
- saga consumer `Put()` error → Failed emitted;
- step handler sync error → Failed emitted;
- async `StepCompleted(false)` → Failed emitted;
- factory bridge filters by sagaType, re-emits `FAILED` with `accountId`;
- login failure write resolves the correct session, tolerates disconnected session;
- reverse-walk compensator: `AwardAsset`/`CreateSkill`/`CreateAndEquipAsset` inverted in reverse order; `CreateCharacter` deleted last; idempotent delete on missing row.

### Phase 11 — Build/verify sweep (S)
`go build ./...` and `go test ./...` for `atlas-saga-orchestrator`, `atlas-character-factory`, `atlas-character`, `atlas-skill`, `atlas-login`. Docker builds for each (shared-lib changes may be involved — per CLAUDE.md, always verify Docker builds on shared-lib changes). Update each service's internal Kafka table in `README.md` to reflect the new emit/consume sites.

## Risk Assessment and Mitigation

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Double Failed emission under timer/handler race | Medium | Medium — duplicate client-facing error writes; log noise | Phase 2 terminal-state guard; explicit unit test for concurrent transitions (Phase 10). |
| Compensation delete commands race against the still-running original create (late success from downstream) | Medium | Low-Medium — orphan rows, resolved on retry | §4.9 accepted limitation for v1. Idempotent deletes (Phase 5) mean orchestrator does not have to reason about TOCTOU. |
| `AccountId` absent at failure time (CreateCharacter never ran) | High | High — login cannot resolve session | Populate `AccountId` from `CharacterCreatePayload` at saga acceptance, not from a completed step result — payload is known before any step runs. Phase 1 verification. |
| Pre-existing sagas that legitimately take >30s silently break | Low-Medium | High — new "failures" appear in flows that previously worked | §8 PRD stance: indefinite wait is not preserved for any caller. Audit saga types during Phase 4; if any genuinely need >30s, caller must set explicit higher value — default stays 30s. |
| Existing orchestrator integration tests fail under new emission semantics | High | Medium — Phase 10 becomes a long tail | Explicitly scoped into Phase 10 from the outset (§8 PRD non-functional requirement); do not defer. |
| Shared-lib changes (new `ErrorCodeSagaTimeout`, new `AccountId` field, new constants on seed topic) ripple into Docker images | High | Medium — image-drift causes runtime type mismatches | Per CLAUDE.md, rebuild Docker for all five affected services in Phase 11. Verify images against freshly built go modules. |
| New `RequestDeleteCharacter` / `RequestDeleteSkill` commands lack parity with existing saga-correlated command conventions | Medium | Medium — cascading rework after code review | Phase 5 explicitly instructs: follow existing command conventions (same topic family as `CreateCharacter` / `CreateSkill`). Cite specific file(s) during implementation. |
| Compensation step failure cascades into chain abort | Low | High — half-rolled-back saga | §4.3.2 explicit: log and continue; Failed event emits at end regardless. Phase 10 test covers a forced compensation-step failure. |
| Factory-side `handleSagaFailedEvent` filter misses non-CharacterCreation sagas → cross-saga-type noise | Low | Low — dropped events in other saga flows | Filter by `SagaType == CharacterCreation` only; explicit log at DEBUG for dropped events to aid diagnosis. |

## Success Metrics

- **Correctness.** With `atlas-data` artificially unavailable: exactly one `StatusEventTypeFailed` on `EVENT_TOPIC_SAGA_STATUS`; exactly one `FAILED` on `EVENT_TOPIC_SEED_STATUS`; client receives `AddCharacterCodeUnknownError` within 11s. Retry of same name/slot succeeds without collision.
- **No regressions.** Success path latency unchanged (measured by an existing orchestrator test). Existing non-character-creation sagas are unaffected (manually verified on `InventoryTransaction` / `StorageOperation` / `SelectGachaponReward`).
- **Test coverage.** All acceptance-criteria bullets from PRD §10 have a corresponding unit test or updated integration test.
- **Build hygiene.** `go build ./...` and `go test ./...` pass for all five affected services; Docker images build for the three that changed shared-lib imports.

## Required Resources and Dependencies

### People / time
- One Go backend engineer familiar with the Atlas saga orchestrator and Kafka consumer pattern. ~5–7 working days for Phases 0–11 if no cross-service surprises.

### External dependencies
- None new. `atlas-inventory.RequestDestroyItem` already exists and is reused. No new infra, no new topics, no DB changes.

### Code dependencies (in-repo)
- **Changed services**: `atlas-saga-orchestrator`, `atlas-character-factory`, `atlas-character`, `atlas-skill`, `atlas-login`.
- **Unchanged but imported**: `atlas-inventory` (consumer side — unchanged code path), `atlas-data` (unchanged).
- **Shared libs**: the `kafka/message/saga/kafka.go` constants/bodies live inside `atlas-saga-orchestrator` itself; consumers import it directly. The new `AccountId` field and `ErrorCodeSagaTimeout` constant ripple to `atlas-character-factory` and `atlas-login` via import. Per CLAUDE.md, **verify Docker builds** after any such change.

## Timeline Estimates

Effort shorthand: S (≤0.5d) / M (0.5–2d) / L (2–5d) / XL (>5d).

| Phase | Effort | Running total |
|---|---|---|
| 0 — Safety rails | S | 0.5d |
| 1 — Wire-format extensions | S | 1.0d |
| 2 — Terminal-state guard | M | 2.0d |
| 3 — Guaranteed Failed emission | M | 3.5d |
| 4 — Per-saga timeout | M | 5.0d |
| 5 — Compensation delete commands (×2 services) | M | 6.5d |
| 6 — Reverse-walk compensator | L | 9.5d |
| 7 — Factory bridge failure handler | S | 10.0d |
| 8 — Login failure handler | S | 10.5d |
| 9 — Fix `atlas-character` discard + `CreateAndEmit` audit | M | 12.0d |
| 10 — Tests (existing updates + new coverage) | L | 15.0d |
| 11 — Build/verify sweep | S | 15.5d |

Total: ~15–16 working days of focused work, or ~3 calendar weeks with normal review/CI friction. Phases 0–4 are sequentially load-bearing; Phases 5, 7, 8, 9 can parallelize once Phase 1 is in; Phase 6 depends on 5; Phase 10 is continuous from Phase 3 onward.

## Out of Scope (reminder from PRD §2 non-goals)

- Per-tenant timeout configuration.
- Client-side distinction of failure causes (single generic error code).
- Same error-swallow audit for adjacent login sagas (delete character, rename).
- Cross-service integration tests (unit is sufficient).
- New error codes beyond `ErrorCodeSagaTimeout` and any that naturally fall out.
- Changes to `POST /characters/seed` REST contract or 202-Accepted semantics.
- Refactoring how `atlas-data` reports unavailability.
- Orphan-row sweeper for late-success leaks (§4.9 accepted).
