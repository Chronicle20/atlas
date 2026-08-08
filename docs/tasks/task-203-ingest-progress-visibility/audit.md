# Plan Audit — task-203-ingest-progress-visibility

**Plan Path:** docs/tasks/task-203-ingest-progress-visibility/plan.md
**Audit Date:** 2026-08-08
**Branch:** task-203-ingest-progress-visibility
**Base Branch:** main (merge-base `2f1fa9c87`)
**Head:** `e4da7b9b9`

## Executive Summary

All 12 plan tasks were implemented with file:line evidence; the two mid-flight review findings (Task 7's `Init` bypassing the superseded-pod guard, Task 10's reason-gate/interrupted-worker narrowing) were genuinely fixed in follow-up commits (`deeafa65b`, `35523c3ad`), not just claimed fixed. `go build`, `go vet`, and `go test -race -count=1` are clean for `libs/atlas-redis` and `services/atlas-data/atlas.com/data` (re-run directly, not taken on faith), and `tools/redis-key-guard.sh` / `tools/goroutine-guard.sh` both exit 0. No `// TODO`/stub/501 or literal `/home/<user>/…` path exists in any changed file. One real gap remains: the "Job list errors → serve `running`" branch in `corroborateRunning` (`runtime/rest/resource.go:158-161`) has no test exercising a `List` error — see Action Items. The two PRD acceptance criteria correctly labeled runtime-only (live-poll monotonicity, Job garbage-collection) are honestly characterized as structural-only coverage, not overclaimed.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `Registry.UpdateWithTTL` in `libs/atlas-redis` | DONE | `libs/atlas-redis/registry.go` — `UpdateWithTTL` added after `Update`; commit `64626ff32`. Tests `TestRegistry_UpdateWithTTL_RetainsTTL`, `TestRegistry_Update_ClearsTTL`, `TestRegistry_UpdateWithTTL_NotFound` present in `registry_test.go` and pass (`go test -race ./...` re-run clean). `Update` itself untouched (plan-mandated duplication, not a defect — see design deviation note below). |
| 2 | `ingestrun` leaf package | DONE | `services/atlas-data/atlas.com/data/ingestrun/ingestrun.go` created with `Phase`, `WorkerState`, `WorkerEntry`, `Record`, `KeySuffix`, `NewJobRegistry`, `NewRunRegistry`, `NewRecord`, and pure `With*` mutators exactly as specified; commit `3ab1ea3bc`. Imports only stdlib, `go-redis`, `libs/atlas-redis` (verified via import block, no `atlas-data/data` or `atlas-data/runtime` dependency). `go test ./ingestrun/...` passes (re-run). |
| 3 | `workers.RegisteredNames()` | DONE | `services/atlas-data/atlas.com/data/data/workers/registry.go` — `RegisteredNames()` appended, iterates `Registered` in order; commit `eef5dc239`. Test `TestRegisteredNamesMatchesRegistered` in `data/workers/registry_test.go` passes. |
| 4 | Fix `ingestJobKeySuffixFromLabels`, inject `INGEST_RUN_ID`, initialise run record | DONE | `runtime/rest/jobs.go` — `ingestJobKeySuffixFromLabels` rebuilds raw scope from the unsanitized `tenant` label (F-2 fix); `IngestRegistries`/`NewIngestRegistries` added; `JobCreator.RunRegistry` field added; `Create` mints `runId := uuid.NewString()`, passes to `renderJob`, writes `ingestrun.NewRecord(...)` via `PutWithTTL(..., RecordTTL)`; commit `42ab15c2b`. Tests `TestIngestJobKeySuffixFromLabelsRoundTrips`, `TestRenderJobInjectsRunId`, `TestJobCreatorCreateInitialisesRunRecord` present and passing. |
| 5 | Watchdog writes phase `stuck` | DONE | `runtime/rest/watchdog.go` — `DefaultWatchdogTimeoutSecs = 7200`, `Watchdog.runRegistry()`, `deleteStuckJob` writes `WithPhase(PhaseStuck, now, reason)` via `UpdateWithTTL`, preserves per-worker states (design Q1 — closure never touches `rec.Workers`); commit `902ab68ce`. Tests `TestDeleteStuckJobWritesStuckRecord` (asserts worker states preserved exactly) and `TestDeleteStuckJobWithNoRecordIsQuiet` pass. |
| 6 | `ProgressSink` / `runWithProgress` in `data` package | DONE | `services/atlas-data/atlas.com/data/data/progress.go` — `ProgressSink`, `noopSink`, `RunOption`, `WithProgress`, `newRunConfig`, `runWithProgress`; `runwz.go`'s `RunWorkers` gained variadic `opts ...RunOption`, `runOne` routed through it; commit `37715702f`. Test-local type rename (`event`→`progressEvent`) to avoid collision with existing generic `event[E any]` in `data/kafka.go` — cosmetic, does not change the produced interface (confirmed: `ProgressSink`/`RunOption`/`WithProgress` spelled per brief). Tests pass. |
| 7 | Redis-backed `redisSink` in `runtime/ingest` | DONE (post-fix) | `runtime/ingest/progress.go` — `redisSink`, `Init`, `Finish`, `WorkerStarted`/`WorkerFinished`, `writeCtx` (`context.WithoutCancel` + 5s timeout), superseded-pod guard; commit `242307db7`. **Review finding fixed in `deeafa65b`**: `Init`'s closure originally bypassed the guard `apply()` enforced elsewhere; extracted into shared `guardedUpdate`, now used by both `Init` and `apply`, guard re-evaluated on every optimistic-lock retry. Verified directly in `progress.go:100-118` (`Init` calls `s.guardedUpdate`, same as `apply`). Tests incl. `TestSinkInitDropsWritesForASupersededRun` (added in the fix commit) pass. |
| 8 | Rewrite `GET /api/data/process` | DONE, with one untested branch | `runtime/rest/ingestrun_model.go` (new) + `runtime/rest/resource.go` (rewritten) — `InitResource(jc, regs)`, `processCreate` on `wzinput.ResolveScope`, `processStatus`/`corroborateRunning` implementing the §4.4-over-§7 deviation exactly (fresh heartbeat → running w/o k8s call; no heartbeat + `jc==nil` → unknown; no heartbeat + list error → running; no heartbeat + list succeeds, no live Job → unknown); commit `176427361`. 12 tests in `resource_test.go` cover the FR-4 matrix, **except** the "Job list errors → running" branch (`resource.go:158-161`) — no test constructs a fake clientset with a `List` reactor returning an error. See Action Items. |
| 9 | UI types/fetchers/hooks | DONE | `seed.service.ts` (`IngestRun` types, `getIngestRun`, `getCanonicalIngestRun`), `useSeed.ts` (`useIngestRun`), `useCanonicalData.ts` (`useCanonicalIngestRun`) — all present with `staleTime: 0, refetchInterval: 5000, retry: false` per FR-5.1/5.7; commit `9f455e53c`. |
| 10 | `IngestProgressPanel` + `ingest-progress.ts` | DONE (post-fix) | `components/features/setup/ingest-progress.ts` (`INGEST_BLOCKING_PHASES`, `ingestPublishBlockReason`, `ingestElapsedMs`, `formatDuration`) and `IngestProgressPanel.tsx`; commit `835194be2`. **Review finding fixed in `35523c3ad`**: an intermediate version narrowed the reason paragraph to render only when no worker had its own error, and narrowed the "interrupted" label — both dodges around ambiguous-match test failures, not real requirements. Fix restored unconditional rendering (`IngestProgressPanel.tsx:88-89` renders `run.reason` whenever truthy, independent of worker errors) and scoped the test queries instead; added a regression test covering a run with both a run-level reason and a distinct worker error. Verified directly in current `IngestProgressPanel.tsx`. |
| 11 | Mount panel + gate publish | DONE | `SetupPage.tsx` — `useIngestRun()` + `<IngestProgressPanel>` mounted as its own row directly beneath "Process Data" (verified at file, ~line 401), before the conditional restore row (Q3: document-count badge left untouched, separate row). `BaselinesPage.tsx` — `useCanonicalIngestRun(sel)`, `publishDisabled` extended with `ingestPublishBlockReason(...) !== null`, panel mounted, `warning` slot renders the block reason; commit `35523c3ad`/`e4da7b9b9`. `BaselinesPage.test.tsx` has real assertions for `running`/`succeeded`/`none` gating (`pages/__tests__/BaselinesPage.test.tsx:225-268`). `SetupPage.test.tsx` mocks `useIngestRun` (`data: undefined, isError: false`, line 57) but has no assertion the panel receives/renders those values — see Skipped/Deferred section. |
| 12 | Full verification sweep | DONE | `.superpowers/sdd/plan/task-12-report.md` — full PRD §10 walk with per-row test citations; two rows (6, 8) explicitly and correctly flagged as structural-only, not live-verified. Independently re-run in this audit: `go build`, `go vet`, `go test -race -count=1` clean for `libs/atlas-redis` and `services/atlas-data/atlas.com/data`; `tools/redis-key-guard.sh` and `tools/goroutine-guard.sh` both exit 0. `tools/lint.sh --check` was not re-run in this audit (timed out after 2 minutes in this session, a known false-fail-without-nvm risk per project memory — not re-attempted with nvm loaded); task-12's own report claims it clean. |

**Completion Rate:** 12/12 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0 (two DONE items required a fix-round that was completed and verified; classified as DONE not PARTIAL since the final state is fully correct)

## Skipped / Deferred Tasks

None skipped. Two items called out by the task brief as known-adjudicated gaps, confirmed accurate on inspection:

1. **Task 8 — untested Job-list-error branch (`corroborateRunning`, `resource.go:158-161`).** The branch `list, err := jc.K8s...List(...); if err != nil { return ingestrun.PhaseRunning }` has no direct test. All 12 tests in `resource_test.go` exercise: no record, cross-tenant isolation, shared-scope operator gate, bogus scope, terminal record without k8s, stuck record, fresh-heartbeat running, stale-heartbeat+no-job→unknown, no-k8s-client→unknown, selector narrowing, live-job→running, no-registry→none — but none injects a `List` error via a fake-clientset reactor (`cs.PrependReactor("list", "jobs", func(...) (bool, runtime.Object, error) { return true, nil, errors.New(...) })`), which `client-go`'s `fake.Clientset` supports directly and would be a small, self-contained addition. **Impact if unfixed:** the branch is short (4 lines) and its logic is simple enough to read-verify, but it is the one path in the whole feature where a transient k8s API hiccup could (if the logic were ever inverted by a future edit) silently flip a live run to `unknown` and block the baseline-publish gate — exactly the failure mode FR-4.3/§7 was designed to avoid. An untested branch on the publish-blocking code path is a materially higher-value test than most already-covered branches.
2. **Task 11 — `SetupPage.test.tsx` mocks but doesn't assert.** `useIngestRun` is mocked to return `{ data: undefined, isError: false }` (line 57) so existing `SetupPage` tests don't crash on the new hook, but no test in that file asserts the panel is mounted or renders differently for a populated/error `IngestRun`. This is a real coverage gap for FR-5.1's "Setup page shows the tenant-scope panel" criterion specifically at the `SetupPage` integration level — though the panel's own rendering logic is separately and thoroughly covered by `IngestProgressPanel.test.tsx` (9 cases) and the mount-site wiring is visible by direct code read (`SetupPage.tsx` lines ~401-405).
3. **Task 1 — `UpdateWithTTL` duplicates ~40 lines of `Update`'s body.** Confirmed intentional per plan.md:77 ("`Update` itself is left alone — changing its semantics would touch live monster-store state") — not an oversight; `TestRegistry_Update_ClearsTTL` exists specifically to pin `Update`'s unchanged behavior as a regression trip-wire.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| libs/atlas-redis | PASS | PASS | `go build`/`go vet`/`go test -race -count=1 ./...` re-run directly in this audit; clean. |
| services/atlas-data (atlas.com/data) | PASS | PASS | `go build ./...`, `go vet ./...` clean; `go test -race -count=1 ./ingestrun/... ./data/... ./runtime/...` — all `ok` (ingestrun, data, data/workers, data/wztoxml, runtime/ingest, runtime/rest); `data/mock` has no test files (pre-existing, unrelated). |
| services/atlas-ui | PASS (per task-12 report, not independently re-run) | PASS (per task-12 report) | `npm run build` and vitest reported clean in `.superpowers/sdd/plan/task-12-report.md` Step 4; not re-run in this audit session (no nvm22 invoked). Test file counts confirmed substantive: `IngestProgressPanel.test.tsx` (9 cases), `ingest-progress.test.ts` (7 cases), `BaselinesPage.test.tsx` publish-gate block (3 cases, real assertions). |

Guards (repo root, re-run in this audit): `tools/redis-key-guard.sh` exit 0; `tools/goroutine-guard.sh` exit 0. `tools/lint.sh --check` not completed in this session (timed out at 2 min without nvm loaded); trusted from task-12's report but not independently confirmed here.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** NEEDS_REVIEW (not blocking — see rationale)

Rationale for `NEEDS_REVIEW` rather than a clean `READY_TO_MERGE`: every plan task is genuinely done, both known review-round fixes are verified in the actual code (not just claimed), and the acceptance-criteria walk is honest about its two structural-only rows. The one open item — the untested Job-list-error branch on the publish-blocking read path — is small enough that a reviewer might reasonably decide to accept it as-is (per the task's own framing, "already adjudicated"), but it sits on a code path with above-average consequence if it silently breaks in the future, so a human sign-off on that specific trade-off is warranted before merge rather than auto-passing it through this audit.

## Action Items

1. **Recommended, not blocking:** Add a test for `corroborateRunning`'s Job-list-error branch (`runtime/rest/resource.go:158-161`) using `fake.NewSimpleClientset(...)` with a `PrependReactor("list", "jobs", ...)` returning an error, asserting the result stays `ingestrun.PhaseRunning`. This is mechanically straightforward given the existing test scaffolding in `resource_test.go` (`newRegs`, `doStatus` helpers already exist) and closes the one path in the FR-4.3 corroboration logic without direct coverage.
2. **Optional:** Add a `SetupPage.test.tsx` case asserting the mounted `IngestProgressPanel` reflects a non-empty `useIngestRun` result (e.g., renders the phase text), to cover FR-5.1 at the integration level rather than only at the unit (`IngestProgressPanel.test.tsx`) and manual-code-read levels.
3. **Optional, before final merge sign-off:** Re-run `tools/lint.sh --check` from the repo root with nvm22 loaded to independently confirm the clean result task-12 reported (this audit's own attempt timed out without nvm and was not retried).

---

## Resolution (controller, post-audit)

Recorded after the three reviewer sections above. All surviving findings were
addressed in the final fix wave (`a1da0502b`) and verified by a scoped
re-review:

| Finding | Source | Resolution |
|---|---|---|
| FILE-02 — RestModels/Transform not in `rest.go` | backend | **Fixed.** Human ruled the guideline governs over the plan's mandated filename; `ingestrun_model.go` renamed to `rest.go` via `git mv` (100% similarity, pure rename). Pre-existing RestModels elsewhere in the package were deliberately left alone. |
| Watchdog stuck write lacks a run-id guard | backend | **Fixed.** Human ruled fix-now. `deleteStuckJob` now recovers run identity from the Job's `INGEST_RUN_ID` container env var (stamped by `renderJob`) and applies the same guard used by the ingest-pod write paths, inside the optimistic-lock closure. Covered by `TestDeleteStuckJobDoesNotClobberNewerRun`. |
| "Job list errors → serve `running`" branch untested | plan-adherence | **Fixed.** `TestProcessStatusRunningWithJobListErrorStaysRunning` forces a `List` error via `PrependReactor` and asserts `running`, not `unknown`. |
| `SetupPage.test.tsx` asserts nothing about the panel | frontend (FE-17) | **Fixed.** New test supplies a realistic `useIngestRun` value and asserts the panel's rendered content. |

Deliberately **not** actioned, with rulings:

- **FE-06** raw `text-amber-*` / `text-emerald-*` in `IngestProgressPanel.tsx` —
  continues a pre-existing codebase-wide pattern; no `warning`/`success`
  semantic token exists in `app/globals.css` to redirect to. Fixing it is a
  design-system change, not a task-203 change.
- **FE-10** query-key idiom in the new hooks — matches every sibling key in the
  same files; changing only the new ones would create inconsistency.
- **Task 1** `UpdateWithTTL` duplicating `Update`'s WATCH/retry body — a
  deliberate, documented trade-off to avoid changing `Update`'s semantics for
  live callers including monster-store state.

Known behavioural note for the PR body: the new watchdog run-id guard fails
**open** (the write proceeds) when either side's run id is empty — e.g. a Job
created before run ids existed. This reproduces the pre-fix behaviour for that
case only and is not a regression.
