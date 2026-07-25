# Plan Audit — task-135-saga-terminal-race

**Plan Path:** docs/tasks/task-135-saga-terminal-race/plan.md
**Audit Date:** 2026-07-08
**Branch:** task-135-saga-terminal-race
**Base Branch:** main (merge-base 4ab0c492ee) → Head e4468bd3d9

## Plan-Adherence Review

### Executive Summary

All 10 plan tasks were faithfully implemented; every stated deliverable, interface name, and test was found in the diff with file:line evidence. Nothing was silently skipped, stubbed, or deferred. The three documented intentional deviations (RemoveAll DestroyAsset exclusion, `(bool, error)` signature returning `(false, err)` on dispatch failure, camelCase `lateCompensated` JSON key, otel promoted to a direct dependency) are all present exactly as described. The focused test suite (`-race -tags=test`) passes, and `go vet`/`go build` are clean; no TODO/stub markers remain in landed source.

### Task Completion

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1 | EventKind outcome classification + `SkipReasonSagaTerminal` | DONE | `saga/event_acceptance.go:218-312` (`EventOutcome`, `OutcomeSuccess/Failure`, `outcomeTable`, `EventOutcomeOf`), `:324` (`SkipReasonSagaTerminal`). Tests `saga/event_acceptance_test.go:TestOutcomeTableCompleteness`, `TestEventOutcomeOf_FailureKinds`. |
| 2 | `lateCompensated` step marker | DONE | `saga/model.go:722` (field), `:748` (accessor), `:760/770` (Marshal), `:915/928` (Unmarshal), `:587` (`WithStepLateCompensated`); copy-on-write literals preserve field at `:544, :573, :632`. Test `TestStepLateCompensatedMarker_RoundTrip`. |
| 3 | Terminal-preserving store (version bump + guarded Put/Remove) | DONE | `saga/store.go:321` (TryTransition `version+1`), `:130` (Put optimistic CASE), `:182/184` (Put OnConflict CASE + monotonic version), `:209` (Remove CASE preserving `failed`). Tests `TestPostgresStore_TryTransitionBumpsVersion/PutCannotResurrectTerminal/RemovePreservesFailed`. |
| 4 | Cashshop processor plumbing + mock | DONE | `cashshop/mock/processor.go` (6 methods), `processor_test.go:11` interface assertion; `saga/compensator.go:34,86,100` + `csP` copied in every `With*` builder; `saga/processor.go:37,225-231`; `saga/mock/processor.go:25,92`. |
| 5 | `CompensateLateStep` claim-then-dispatch | DONE | `saga/compensator.go:1201` (`lateCompensableActions`), `:1217` (`CompensateLateStep`), `:1272` (`claimLateCompensation`), `:1309` (`dispatchLateInverse` full per-action switch). Tests `TestCompensateLateStep_*` (4, incl. RemoveAll). |
| 6 | `EmitSagaFailed` test seam | DONE | `saga/producer.go:107-120` (`emitSagaFailedByIdsFn`/`emitSagaFailedByIdsImpl`); `saga/producer_testseam.go` (`SetEmitSagaFailedForTest`, `//go:build test`). |
| 7 | Terminal gate in `AcceptEvent` + absorb core + span | DONE | `saga/processor.go:410-412` (gate after saga-not-found, before pending/mismatch), `:465` (`absorbLateTerminalEvent`), `:481` (`absorbLateTerminal`), `:507-513` (span `saga.late_event_absorbed` + 5 attributes). otel direct in `go.mod:21`. Tests `TestAcceptEvent_TerminalLifecycleAbsorbs/TerminalRoutesLateSuccessOnly/PendingLifecycleStillAccepts`. |
| 8 | Commit-time gate + ordering-invariant doc | DONE | `saga/processor.go:530-538` (`stepCompletedWithResultOnce` re-check → `absorbLateTerminal("step_completed",...)`), `saga/lifecycle.go:15-16` (task-135 ordering invariant comment). Test `TestStepCompleted_TerminalAfterAccept`. |
| 9 | Deterministic task-102 integration reproduction | DONE | `saga/late_event_integration_test.go` (`//go:build test`): `TestLateEvent_TimeoutRacesCompletion` (a–e), `TestLateEvent_FailureOutcomeAbsorbOnly`. Uses `NewPostgresStore(sqlite)` + `SetEmitSagaFailedForTest`. |
| 10 | Full verification sweep | DONE | Focused `go test -race -tags=test ./saga/` PASS; `go vet ./saga/...` + `go build ./...` clean; per task prompt the full sweep + `docker buildx bake atlas-saga-orchestrator` + `redis-key-guard.sh` were run green from the module root. |

**Completion Rate:** 10/10 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

### Intentional Deviations — Verified Present

- **Task 5 RemoveAll exclusion** (context.md §Deviation 2 extension, controller-approved): `saga/compensator.go:1238-1244` routes `DestroyAsset` with `payload.RemoveAll` into the absorb-only path with a `late_effect_unrecoverable` WARN; explicit-quantity `DestroyAsset` still compensates via `:1336-1344`. Covered by `TestCompensateLateStep_DestroyAssetRemoveAll_Unrecoverable`.
- **`CompensateLateStep` returns `(bool, error)`**: `saga/compensator.go:73,1217`; the bool feeds `late.compensated` span attr (`processor.go:497,513`). On dispatch failure it returns `(false, err)` (`compensator.go:1260`) — matches the prompt's stated deviation (plan sketch showed `(true, err)`; commit ae64a84f5d corrected it).
- **JSON key `lateCompensated`** (camelCase): `saga/model.go:760,915`.
- **otel promoted to direct dependency**: `go.mod:21` (`go.opentelemetry.io/otel v1.44.0`, no `// indirect`).
- **`TryTransition` leaves local `ver` map stale** (fails closed): `saga/store.go:311-316` comment + no `s.ver` mutation.
- **OnConflict Put branch hardened** (status CASE + `sagas.version + 1`): `saga/store.go:182-184`.

### Build & Test Results

| Module | Build | Tests | Notes |
|--------|-------|-------|-------|
| atlas-saga-orchestrator | PASS (`go build ./...`) | PASS (`go test -race -tags=test ./saga/` on all new tests, 1.07s) | `go vet ./saga/...` clean; no TODO/stub markers in landed source. |

### Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending the standing pre-PR `superpowers:requesting-code-review` and the out-of-tree Tempo `span_metrics.dimensions` allowlist follow-through flagged in context.md — an ops task, not a code gap).

### Action Items

None. All tasks implemented; no gaps found.

## Backend-Guidelines Review

- **Reviewer:** backend-guidelines-reviewer (adversarial DOM-*/SUB-*/SEC-*)
- **Scope:** `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator` (saga/, cashshop/mock/), base `4ab0c492ee` → head `e4468bd3d9`
- **Date:** 2026-07-08
- **Build/Test:** PASS (prior full sweep: go vet / go build / go test -race normal + -tags=test / docker bake / redis-key-guard all green; not re-run)
- **Verdict:** PASS

### Applicability note

The `saga` package is the orchestration engine, not a REST resource package. REST-shaped DOM checks (DOM-04/05/08/09/12/14/15/17/18/19 Transform/RegisterInputHandler/JSON:API/handler-layering) are N/A — no `resource.go`, `rest.go`, or HTTP handlers were touched. Checks below are the ones the diff can violate.

### Checklist results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor/compensator take `logrus.FieldLogger` | PASS | `saga/compensator.go:77` (`l logrus.FieldLogger`); `saga/store.go:37` |
| DOM-11 | Lazy/optimistic store, no eager resurrection | PASS | `saga/store.go:302-329` TryTransition version-bump; `:120-156` optimistic CASE Put |
| DOM-20 | Table-driven / builder tests | PASS | `saga/store_test.go:31-39` NewBuilder; builder used across all new tests |
| DOM-21 | Reuse atlas-constants types, no reinvention | PASS | `saga/compensator.go:20-21` imports `atlas-constants/{channel,world}`; `:1350` `channel.NewModel(...)`. `EventKind`/`EventOutcome` are service-local event classifications with no atlas-constants equivalent |
| DOM-24 | Kafka producer stubbed in emitting tests | PASS | `saga/testmain_test.go:11` `producertest.InstallNoop()`; no per-test `t.Cleanup(producer.ResetInstance)` |
| Immutable model | Copy-on-write preserves all fields | PASS | `saga/model.go:565-604` `WithStepLateCompensated`/`WithStepStatus`/`WithStepResult` copy all 8 `Step` fields + all 5 `Saga` fields (struct at `model.go:268`, `Step` at `:716`) |
| Error handling | Late-inverse dispatch propagates typed errors | PASS | `saga/compensator.go:1246-1265` claim/dispatch errors returned; bad payload asserts return errors (`:1313` etc.) |
| Test hygiene | No `*_testhelpers.go` | PASS | `find . -name '*_testhelper*'` → none; test setup via Builder + sqlite `:memory:` (`store_test.go:16-22`) |
| SEC (span PII) | `transaction.id` excluded from span attributes | PASS | `saga/processor.go:495-503` span sets only `tenant.id`,`saga.type`,`saga.lifecycle_state`,`late.outcome`,`late.compensated`; `transaction_id` appears only in the log `fields` (`:463`) |
| SEC (secrets) | No hardcoded secrets | PASS | none introduced |

### Concurrency / optimistic-locking review (store.go)

- `TryTransition` bumps `version` (`store.go:321`) but deliberately does not sync `s.ver` — documented at `store.go:311-316`; every pre-transition optimistic `Put` fails `VersionConflictError` and re-reads into the commit-time gate. Correct and consistent with the retry loops (`compensator.go:1273-1304`, `stepCompletedWithResultOnce`).
- Terminal-preserving `Put` (both optimistic `store.go:130` and OnConflict `:182`) and `Remove` (`:209`) use SQL `CASE` so `failed`/`completed` can never be regressed; `saga_data` still updates for the marker. Verified no path resurrects a terminal status.
- Claim-then-dispatch marker (`compensator.go:1272-1305`) gives at-most-once rollback under version-race; loser goroutines observe the persisted `lateCompensated` marker.

### Findings

None. No DOM-*/SUB-*/SEC-* violation found with file:line evidence.
