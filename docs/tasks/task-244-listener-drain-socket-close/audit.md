# Plan Audit — task-244-listener-drain-socket-close

**Plan Path:** docs/tasks/task-244-listener-drain-socket-close/plan.md
**Audit Date:** 2026-08-20
**Branch:** task-244-listener-drain-socket-close
**Base Branch:** main (merge-base `0b6917b6e`)

## Executive Summary

All 5 plan tasks were implemented and committed exactly as specified, including both documented deliberate deviations (Task 3's `SessionsForHandle`/`KickSession` placement in `socket/drain.go` instead of `main.go`, and Task 4's absence of a dedicated test). Two post-task fix rounds (`901b64606`, `1e1042a10`) closed real gate findings — a bare `go` statement needing `routine.Go`, a data race in two Task 2 tests, and errcheck violations on unchecked `Close()` returns — none of which reopened or contradicted a plan task. `libs/atlas-socket`, `atlas-channel`, and `atlas-login` all build and pass `go test ./... -race` (module-local) cleanly at HEAD. Working tree is clean apart from untracked review artifacts. No skipped, partial, or deferred plan tasks were found.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `libs/atlas-socket` — `Bind`/`Serve` split, `WaitGrouper`, accept-loop `net.ErrClosed` exit | DONE | `23cc82c5c`; `server.go` matches plan's exact code blocks (WaitGrouper, buildConfig, Bind, Serve, Run, run signature change); `bind_serve_test.go` has all 4 named tests (`TestBindSucceedsOnEphemeralPort`, `TestBindFailsWhenPortAlreadyBound`, `TestServeReturnsWhenListenerClosedWhileContextLive`, `TestServeIncrementsSessionWaitGroupAtAcceptSite`). `atlas-login` builds with zero diff (confirmed `git status --porcelain services/atlas-login` empty). |
| 2 | `atlas-channel/listener` — handle hooks, drain reordering, `ErrDraining`, concurrent `DrainAll` | DONE | `a2fe09919`; `handle.go` `Handle` struct gains `CloseListener`/`Sessions`/`Kick` verbatim; `registry.go` trims `Dependencies` to 2 fields, adds `ErrDraining`, `Add` refuses draining, `Drain` phase 1 closes listener / phase 2 uses `h.Sessions`/`h.Kick`, `DrainAll` concurrent; `main.go` 3-field stub deletion. All matches plan's literal code. |
| 3 | `atlas-channel/socket` — bind-first `CreateSocketService`, `dualWaitGroup`, drain-phase-2 helpers | DONE | `dbb7fcb1e`; `init.go` rewritten to bind first via `socket.Bind`, add `dualWaitGroup`, drop `SetPort`; new `socket/drain.go` with `SessionsForHandle`/`KickSession` exported (confirmed by direct read) — this is the documented deviation from design §4.5, and the plan's own "Placement note" text is reproduced verbatim as a code comment is not required, but the file/function shape matches exactly what the plan specifies. `drain_test.go` has `TestKickSessionRejectsANonSessionModel`. |
| 4 | `atlas-channel/main.go` — propagate the bind error and populate the handle | DONE | `345593a61`; `buildListener` tail matches plan's exact replacement block: `lis, err := socket.CreateSocketService(...)`, `if err != nil { return nil, err }`, `h.CloseListener = lis.Close`, `h.Sessions = socket.SessionsForHandle(...)`, `h.Kick = socket.KickSession(...)`. No test of its own, as the plan explicitly calls for (`main.go` is untestable by construction) — not a gap. |
| 5 | `atlas-channel/configuration/projection` — thread `h.Ctx` and retry failed adds | DONE | `92ae4c480`; `loop.go` threads `h.Ctx` into `AddBody` (Step 3), adds `pending`/`retries` maps and the retry-first tick body (Step 4), `execute` returns `error` with `ErrDraining` handled distinctly. `projection_test.go` gains 195 lines covering the three named test functions. |

**Completion Rate:** 5/5 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. No task was skipped, partially implemented, or silently deferred.

### Post-task fix rounds (not plan tasks, but part of "done means verified")

- **Fix round 1** (`901b64606`): converted `DrainAll`'s bare `go` to `routine.Go` (goroutineguard finding), fixed a genuine `-race` data race in two of Task 2's own tests (`TestRegistry_AddRefusesADrainingHandle`, `TestRegistry_DrainClosesTheBoundSocket`) by adding `Registry.State(key)` and a notifying-WaitGrouper synchronization point, and fixed 2 errcheck violations in `libs/atlas-socket/bind_serve_test.go`. Reviewed and APPROVED (`review-fix-1.md`), with the reviewer explicitly verifying the race fix removed the unsynchronized access rather than masking it.
- **Fix round 2** (`1e1042a10`): fixed the remaining 2 errcheck violations (`registry_test.go:338`, `init_test.go:52`) surfaced once the round-1 fixes stopped shadowing the module in `tools/lint.sh`'s per-module ordering. Report confirms a branch-diff sweep for the same violation class found nothing further.

Both fix rounds are in-scope corrections to code the plan tasks themselves introduced, not scope creep and not evidence of a skipped task — the plan's own two documented deviations (Task 3 placement, Task 4 no-test) are separately confirmed present and unchanged.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| `libs/atlas-socket` | PASS | PASS | `go test ./... -count=1`: all packages ok (`atlas-socket`, `.../crypto`; others have no test files). |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS | `go test ./... -count=1`: full module green. Targeted `go test ./listener/... ./socket/... ./configuration/... -race -count=1`: all green, including `atlas-channel/listener`, `atlas-channel/socket`, `atlas-channel/configuration/projection`. |
| `services/atlas-login/atlas.com/login` | PASS | PASS | `git status --porcelain services/atlas-login` empty (zero edits, per plan Global Constraint); `go build ./...` and `go test ./... -count=1` both exit 0. |

Repo-wide `tools/verify.sh` was intentionally not run in this audit per instructions (a run is in progress against this tree); the ledger records the last gate (`bahurghyo`, `--quick --base cb51b28ed`) as covering Tasks 1-5 + fix round 1 with build/vet/goroutineguard all PASS and only the (now-fixed by round 2) errcheck findings failing. Fix round 2's report shows a clean `gofmt -l .` and green module-local tests, consistent with that finding being resolved, but the flagless gate result was not independently re-run here.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** NEEDS_REVIEW — task-level implementation is complete and faithful; the outstanding gate items per the ledger are the final flagless `tools/verify.sh`, the whole-branch code review, and `superpowers:requesting-code-review`/`finishing-a-development-branch`, none of which are this audit's job and none of which indicate a task was skipped.

## Action Items

1. Re-run `tools/verify.sh` flagless (not `--quick`) once the in-progress run referenced in the dispatch instructions completes, per CLAUDE.md's "only the flagless invocation counts."
2. Complete the remaining ledger steps: whole-branch code review over `0b6917b6e..HEAD` (most capable model per `code-reviewer.md`), then `superpowers:requesting-code-review` and `superpowers:finishing-a-development-branch`. These are process steps already tracked in `.superpowers/sdd/plan/progress.md`, not defects found by this audit.
