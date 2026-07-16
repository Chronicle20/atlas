# Plan Audit — task-136-consumer-fetch-wedge

**Plan Path:** docs/tasks/task-136-consumer-fetch-wedge/plan.md
**Audit Date:** 2026-07-08
**Branch:** task-136-consumer-fetch-wedge
**Base Branch:** main (merge-base 4ab0c492ee)
**Head:** 86fa2571ac

## Executive Summary

All 7 plan tasks were faithfully implemented. Every plan step maps to concrete
code/doc evidence, and each of the 9 implementation commits (c3a913904d..86fa2571ac)
maps to a plan task or a pre-disclosed reviewer/hardening deviation. The changed
module `libs/atlas-kafka` builds clean, `go vet ./...` and `go vet -tags integration`
are clean, and `go test -race -count=1 ./consumer/` passes (6.055s). No `// TODO`,
stubs, or committed absolute paths were found. The three noted deviations (p99
ceiling fix, S2/S4 SetMaxWait + baseline-delta recalibration, maxWait>=fetchTimeout
startup warn) are present and justified, not silent gaps.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Phase-timing instrumentation on Consumer/Snapshot | DONE | manager.go: fields (262-273), Snapshot fields (295-300), recorders recordFetchDuration/recordHandlerDuration/recordBackoff (396-418), onReaderCreated stamp (338-339), recordFetch TTFF (355-358), both loops wrapped (543-546, 563-565, 637-640, 668-670), backoff capture (511). debug.go attrs (73-78) + map (100-105). timing_test.go present. Commit c3a913904d |
| 2 | Dwell reproduction & attribution harness (integration) | DONE | dwell_integration_test.go present (410 lines, `//go:build integration`), S1-S5 + helpers startDwellKafka/createDwellTopics/latencyRecorder/publishStamped/dumpSnapshots/totalRecreates. Commit ee80c2c1a2. Reviewer deviation: p99 now uses `math.Ceil` nearest-rank (line 111) — commit 30ac51339a. Justified. |
| 3 | Pre-fix baseline capture → findings.md draft | DONE | findings.md "Pre-fix baseline (commit 30ac51339a)" table with actual numbers (S1 p99 22ms/0 recreates, S2 FAIL 75 recreates, S4 FAIL 4, S5 648 vs 4), phase attribution + hypothesis verdicts. Commit 83dc59f172 |
| 4 | Idle-vs-stuck tick classification (the fix) | DONE | manager.go: StatsProvider (55-57), readerMadeProgress (63-70), recordIdleTick/recordNoProgressTick (364-384), handleFetchDeadline (424-439), Snapshot fields IdleTicks/etc (291-294), both loops' DeadlineExceeded branches (552-557, 646-653), updated doc comment (529-535). debug.go (68-71, 95-98). idle_stuck_test.go present. Commit d6087234a3 |
| 5 | Config default changes | DONE | config.go: maxWait 10s (34), fetchTimeout 1m (36), rationale comment (10-25). config_test.go asserts new defaults (17-18, 30-31). Commit d0e71ba1bd |
| 6 | Post-fix harness run, IdleTicks assertion, findings completion | DONE | dwell S2 IdleTicks proof `tickedIdle>=10` (275-282); findings.md post-fix results, config table, follow-up decision, final verdicts all completed with quoted numbers. Commit cb434b9b1e. Deviations (SetMaxWait 200ms on S2/S4, S2 baseline-delta recreate assertion) present at lines 244/271-272/364 with calibration rationale documented (findings §"Scenario calibration"). Justified. |
| 7 | Full verification sweep (shared-lib bump discipline) | DONE (partial verification by auditor) | Verification-only task, no artifact. Auditor confirmed: `go test -race -count=1 ./consumer/` PASS; `go vet ./...` and `go vet -tags integration ./consumer/` clean. `docker buildx bake all-go-services` and `redis-key-guard.sh` NOT run by this auditor (out of read-only unit scope) — no redis usage added, no go.mod/Dockerfile wiring change, so low risk. |

**Completion Rate:** 7/7 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Deviations From Plan (all pre-disclosed and justified)

1. **p99 ceiling nearest-rank (Task 2, commit 30ac51339a).** Plan used
   `int(float64(len)*0.99)-1` (truncation, drops worst sample for non-100-multiple
   sizes). Implementation uses `int(math.Ceil(len*0.99))-1` (line 111). Reviewer-driven
   correctness fix; strengthens the assertion. Justified.
2. **S2/S4 SetMaxWait(200ms) + S2 baseline-delta recreate assertion (Task 6).**
   Plan's S2 asserted raw `require.Zero(totalRecreates)`. Implementation adds
   `SetMaxWait(200ms)` to both idle scenarios (lines 244, 364) and replaces the raw-zero
   with a post-join baseline-delta `require.Equal(baselineRecreates, ...)` (lines 255-262,
   271-272). Reason documented in findings §"Scenario calibration": the fix's Stats-based
   idle signal needs fetchTimeout ≫ maxWait AND ≫ group-join time; the compressed test
   used the 10s maxWait default against a 2s fetchTimeout (a ratio inversion) and a
   16-member join transient. Production defaults (10s ≪ 1m) satisfy the invariant. The
   test still proves the fix (0 NEW steady-state recreates + ≥10 idle-ticking consumers).
   Justified.
3. **maxWait>=fetchTimeout startup Warn (hardening, commit 6529987dc4).** Added a
   one-time registration Warn in AddConsumer (manager.go:152-155), never a clamp, plus
   doc notes on SetFetchTimeout/SetMaxWait (config.go:62-73, 83-95) and two new tests in
   manager_test.go (append-only, existing tests unmodified). Guards the calibration
   invariant surfaced in Task 6. Justified.

The final commit (86fa2571ac) is a docs reconcile of findings.md line refs / commit
labels — no code impact.

## Build & Test Results

| Module | Build/Vet | Tests | Notes |
|---------|-------|-------|-------|
| libs/atlas-kafka | PASS | PASS | `go vet ./...` clean; `go vet -tags integration ./consumer/` clean; `go test -race -count=1 ./consumer/` = ok 6.055s. Integration tests (`//go:build integration`) verified green out-of-band per task instructions; not re-run here. |

Constraint check — "existing unit tests pass unmodified": confirmed. The only
manager_test.go change is +82 appended lines (two new hardening tests); no existing
test body was altered. config_test.go changes are the intended default-value assertions
from Task 5.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending the auditor-unrun `docker buildx bake
  all-go-services` and `tools/redis-key-guard.sh` from Task 7 Step 2-3, which the plan
  mandates for a shared-lib bump; risk is low given no redis/go.mod/Dockerfile changes).

## Action Items

None blocking. Optional before PR: run `docker buildx bake all-go-services` and
`tools/redis-key-guard.sh` from the worktree root to close Task 7's remaining
CLAUDE.md build-discipline steps (not exercised by this read-only auditor).
