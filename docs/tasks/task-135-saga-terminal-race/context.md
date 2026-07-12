# Saga Terminal-State Race — Implementation Context

Task: task-135-saga-terminal-race
Companion to `plan.md`. Everything an implementer needs beyond the per-task steps.

## Key files (module root: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/`)

| File | Role in this task |
|---|---|
| `saga/processor.go` | `AcceptEvent` (:362) gains the terminal gate; `stepCompletedWithResultOnce` (:424) gains the commit-time gate; new `absorbLateTerminalEvent`/`absorbLateTerminal`; new `WithCashshopProcessor`. `isVersionConflict`/`maxConflictRetries` (:81-91) are reused by the claim loop. |
| `saga/event_acceptance.go` | `acceptanceTable`/`StepAcceptsEvent`; new `EventOutcome` + `outcomeTable` + `EventOutcomeOf`; new `SkipReasonSagaTerminal`. |
| `saga/model.go` | `Step[T]` gains `lateCompensated` (JSON key `lateCompensated`); `Saga.WithStepLateCompensated`; `WithStepStatus`/`WithStepResult` literals must copy the new field. |
| `saga/store.go` | `TryTransition` (:290) bumps `version`; `Put` (:100) and `Remove` (:193) become terminal-preserving via SQL `CASE`. |
| `saga/cache.go` | Interface unchanged. `GetLifecycle` already race-safe on both impls. `InMemoryCache` needs **no change** (single mutex + commit-time gate cover it; its `Remove` hard-deletes, which is why the integration test uses the sqlite-backed store). |
| `saga/compensator.go` | New `CompensateLateStep` + `claimLateCompensation` + `dispatchLateInverse` + `lateCompensableActions`; new `csP cashshop.Processor` field threaded through every `With*` copy. Reverse-walk idioms to mirror: `DispatchCharacterCreationRollbacks` (:982), `DispatchPetEvolutionRollbacks` (:1105), `compensateEquipAsset` (:298). |
| `saga/timer.go` | `handleSagaTimeout` (:88) — unchanged; invoked directly by the integration test. |
| `saga/lifecycle.go` | Ordering-invariant doc comment (PRD acceptance criterion). |
| `saga/producer.go` / `saga/producer_testseam.go` | `EmitSagaFailedByIds` refactored behind swappable `emitSagaFailedByIdsFn`; `SetEmitSagaFailedForTest` seam (`//go:build test`). |
| `cashshop/mock/processor.go` | New mock (interface at `cashshop/processor.go:14`). |
| `saga/mock/processor.go` | Must gain `WithCashshopProcessorFunc` — interface change breaks it otherwise. |
| `main.go` | **No change.** Reaper (`reapTimedOutSagas` :241) drives compensation via `MarkEarliestPendingStep(Failed)` + `Step`; terminal-preserving `Put` still allows active→compensating. `recoverSagas` (:176) re-drives `Step` on restart — which is why `CompensateLateStep` never mutates step *status* (only the marker). |

## Decisions locked at design time (do not re-litigate)

- Approach B (design §2): layered guards — read gate + commit gate + version bump + terminal-preserving writes. Not handler-level changes (~30 consumer call sites stay untouched), not an actor model.
- Late-success routing decided **inside** `AcceptEvent` via EventKind outcome classification; `stepCompletedWithResultOnce` is the second (authoritative) gate.
- Hard deadline, no grace window; no dead-letter topic; structured log + span-metric only.
- At-most-once rollback (claim-then-dispatch): negation inverses are not idempotent downstream. A crash between claim and dispatch loses the rollback but is auditable via log+span.
- Lifecycle is never re-transitioned by the absorb path; no `Failed` emission from it (the timer already emitted exactly one).
- Metric = Tempo span-metrics (task-040 pipeline), span `saga.late_event_absorbed`, attributes `tenant.id`, `saga.type`, `saga.lifecycle_state`, `late.outcome`, `late.compensated`. `transaction.id` is forbidden as a span attribute — log line only.
- v1 compensable set = value-transfer class: AwardAsset, CreateAndEquipAsset, CreateSkill, CreateCharacter, AwaitCharacterCreated, DestroyAsset, AwardMesos, AwardCurrency, AwardExperience, DeductExperience, AwardFame, EquipAsset, UnequipAsset. Everything else absorb-only + `late_effect_unrecoverable` WARN.

## Deviations from design.md (with rationale)

1. **`CompensateLateStep` returns `(bool, error)`**, not `error` (design §3.4 sketch). The `late.compensated` span attribute (design §3.6) needs to know whether an inverse was actually dispatched.
2. **`DestroyAssetFromSlot` moved to the non-compensable set.** Design §3.4's table listed it beside `DestroyAsset`, but `DestroyAssetFromSlotPayload` (libs/atlas-saga/payloads.go:102) carries no `TemplateId`, so the destroyed item cannot be recreated from the step payload — same "payload lacks the prior value" class as ChangeHair.
3. **JSON key is `lateCompensated`** (camelCase, consistent with `stepId`/`createdAt`), not the design's lowercase `latecompensated`.
4. **`TryTransition` does NOT invalidate the local `ver` map** (design §3.3b said "invalidates"). Deleting the entry would route the next `Put` through the unguarded OnConflict insert path; leaving it stale forces `VersionConflictError` → retry → commit-time gate, which is the intended absorption funnel. The stale-entry behavior fails closed.
5. **OnConflict `Put` branch also hardened** (status CASE + monotonic `sagas.version + 1` instead of reset-to-excluded): design §3.3c's "even a Put built on a fresh read cannot resurrect" is unreachable without it, because the insert path bypasses the optimistic version check entirely.

## Test infrastructure notes

- `TestMain` (saga/testmain_test.go) installs `producertest.InstallNoop()` — Kafka writes are no-ops in tests; producertest has **no capture facility**, hence the `SetEmitSagaFailedForTest` seam for counting Failed emissions.
- Seam files and the tests that use them carry `//go:build test`; run both `go test -race ./...` **and** `go test -race -tags=test ./...`.
- `gorm.io/driver/sqlite` is already in the module graph (indirect); `go mod tidy` promotes it and otel when first imported. sqlite handles the `type:jsonb`/`type:uuid` gorm tags (dynamic typing) and supports `excluded` upsert references and `CASE` expressions used by the store guards.
- The integration test must use `NewPostgresStore(sqliteDB)` via `SetCache` (+ `t.Cleanup(ResetCache)`): the production race window depends on soft-delete semantics (`GetById` finds soft-deleted rows, store.go:73); `InMemoryCache.Remove` hard-deletes and would dead-end at `saga_not_found`.
- Existing helpers to reuse: `newAcceptEventTestProcessor`/`putAcceptEventSaga` (accept_event_test.go), Builder pattern everywhere; no `*_testhelpers.go` files.
- Spans in tests hit the global (noop) tracer provider — safe, nothing to assert; do not add a tracer fixture.

## Dependencies between tasks

```
Task 1 (outcomes) ─┬─> Task 5 (CompensateLateStep) ─> Task 7 (AcceptEvent gate) ─> Task 8 (commit gate) ─> Task 9 (integration)
Task 2 (marker)  ──┤                                                                       ^
Task 4 (cashshop plumbing) ─┘                                                              │
Task 3 (store guards) ─────────────────────────────────────────────────────────────────────┤
Task 6 (Failed seam) ──────────────────────────────────────────────────────────────────────┘
Task 10 (verification) last.
```

Tasks 1, 2, 3, 4, 6 are mutually independent and may be executed in any order (or in parallel worktree-free subagents is NOT an option — same files; execute serially).

## Out of scope / follow-through

- **Ops follow-through (not in this repo):** append `saga.type`, `saga.lifecycle_state`, `late.outcome`, `late.compensated` to the Tempo `span_metrics.dimensions` allowlist ConfigMap (same runbook as task-040; Tempo hot-reloads). No Tempo config exists in this repo — flag in the PR description.
- Reconciling sagas corrupted before the fix; timeout-duration tuning; grace windows; widening the compensable set (the WARN + span data will justify it later); `CompensateFailedStep`'s default-case retry semantics (`compensator.go:259`, pre-existing, only reachable pre-terminal).
- `PostgresStore.UpdateStatusFailed` (store.go:240) writes `failed` unconditionally — reachable only from timeout-decision paths, and `failed` is itself terminal; left unchanged.

## Verification (CLAUDE.md)

From module root: `go vet ./...`, `go build ./...`, `go test -race ./...`, `go test -race -tags=test ./...`.
From worktree root: `docker buildx bake atlas-saga-orchestrator` (go.mod touched — mandatory), `tools/redis-key-guard.sh`.
Then `superpowers:requesting-code-review` before any PR.
