# Plan Audit — task-115-safe-goroutine-helper

**Plan Path:** docs/tasks/task-115-safe-goroutine-helper/plan.md
**Audit Date:** 2026-07-02
**Branch:** task-115-safe-goroutine-helper
**Base Branch:** main (merge-base 38d4d0ba2)

## Executive Summary

All 13 plan tasks are fully implemented with file:line evidence. The `libs/atlas-routine`
helper, the `goroutineguard` AST analyzer, the `tools/goroutine-guard.sh` wrapper + CI job,
and the full migration of every non-test bare `go` statement (166 migrated, 1 allowlisted =
167 audit rows) all landed. The completion oracle `./tools/goroutine-guard.sh` was re-run
during this audit and exits 0 with zero findings. The single intentional deferral (Task 12
Step 4, RR-6 doc marking) correctly took the documented ABSENT path — no fabricated RR-6
section — and is a rebase-time follow-up, not a skip. **Plan adherence: FULL.**

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `libs/atlas-routine` helper + tests + go.work + 3 Dockerfile edits | DONE | `libs/atlas-routine/routine.go:15-26` matches design shape byte-for-byte; 4 tests in `routine_test.go:28,43,61,83`; `go.work:16`; Dockerfile edits at `Dockerfile:44` (mod COPY), `:74` (source COPY), `:95` (`for L in` loop with `atlas-routine`); go.sum present |
| 2 | `tools/goroutineguard` analyzer + fixtures + cmd | DONE | `tools/goroutineguard/analyzer.go:19-76` (both diagnostic strings present :59,:63); `analyzer_test.go`; fixtures `testdata/src/{bad,good,github.com/.../atlas-routine}`; `cmd/goroutineguard/main.go`; correctly absent from `go.work` |
| 3 | `goroutine-guard.sh` + CI job + audit skeleton (baseline) | DONE | `tools/goroutine-guard.sh` (mode 755); CI job `.github/workflows/pr-validation.yml:109-123`, `needs:` list :505, `GOROUTINE_GUARD_RESULT` :522, summary row :533, failure `if` :539; baseline recorded as **167** (within plan's 160-170 tolerance, plan estimated ~165) in `migration-audit.md:3` |
| 4 | atlas-lock (completed-flag, no hand-rolled recover) | DONE | `leader.go:156` fn goroutine with `completed` flag (:158,:160,:166); `:169` renewer via `routine.Go`; grep confirms zero hand-rolled `recover()` outside `_test.go`; audit rows 166-167 |
| 5 | atlas-kafka (3 sites; safeHandle untouched) | DONE | `manager.go:146,525,560` migrated; `safeHandle` at `:579` remains a plain synchronous func (not a spawn), untouched per §6.4; audit rows 155-157 |
| 6 | atlas-model (6 sites + testutil allowlist) | DONE | `model/processor.go` + `async/processor.go` migrated with `logrus.StandardLogger()`; `go.mod:7` adds direct logrus require; sole allow marker at `testutil/helpers.go:189`; audit rows 159-165 (accepted-consequence notes on rows 159/161/163/164 per §6.1) |
| 7 | atlas-socket/rest/seeder (7 sites) | DONE | audit rows 147-150 (socket), 153-154 (rest), 158 (seeder); guard clean for all three modules |
| 8 | atlas-channel (~60 sites) | DONE | audit rows 18-82 (65 sites across map/movement/party/session/messenger/asset/monster/pet/drop consumers + socket/init, tasks, main.go:318,327); guard clean |
| 9 | 8 multi-site services | DONE | atlas-maps/login/monsters/monster-death/buffs/data/world/pets all represented in audit rows; PRD motivating site `atlas-monsters .../monster/processor.go:700` = row 129 |
| 10 | 24 long-tail services + 3 extra libs | DONE | remaining services migrated; guard-discovered extras beyond plan's enumerated list (atlas-redis rows 145-146, atlas-outbox 151-152, atlas-service 144) migrated — plan explicitly said "trust the guard, not this list" |
| 11 | Zero-findings gate + audit completion | DONE | `migration-audit.md` = 167 fully-dispositioned rows (166 migrated + 1 allowlisted); completion note `:6`; guard re-run this audit exits 0, zero findings; exactly 1 in-code allow marker |
| 12 | DOM-25 + skill + CLAUDE.md item 6 (RR-6 conditional) | DONE | DOM-25 `backend-guidelines-reviewer.md:102`; `SKILL.md:39`; `anti-patterns.md:36`; `CLAUDE.md:28` item 6; RR-6 correctly ABSENT (see deferral below) |
| 13 | Branch-wide verification gate | DONE | Controller-confirmed: 44 changed modules build + `go test -race` clean; `docker buildx bake all-go-services` 58/58; both guards exit 0. Spot-checked here: guard exit 0. |

**Completion Rate:** 13/13 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None skipped. One intentional, plan-sanctioned deferral:

**Task 12 Step 4 (RR-6 doc marking) — correctly deferred via the ABSENT path.**
`grep 'RR-6' docs/architectural-improvements.md` returns no match on this branch. The plan's
Step 4 is explicitly conditional: the RR-6 section exists only in main's *uncommitted* doc
rewrite, so the ABSENT branch instructs recording a rebase-time follow-up rather than
inventing the section. The implementation correctly did NOT fabricate an RR-6 section and
correctly dropped `docs/architectural-improvements.md` from the Task 12 commit. **Follow-up:**
at pre-PR rebase (once the reliability-review rewrite lands on main), add the
"Resolved by task-115" line under the RR-6 heading. This is a documented follow-up, not a gap.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| all 44 changed modules | PASS | PASS | Controller-confirmed `go build`/`go test -race` clean; `docker buildx bake all-go-services` 58/58 |
| goroutine-guard oracle | PASS | PASS | Re-run during this audit: `./tools/goroutine-guard.sh` exit 0, self-test + full services/+libs/ sweep, zero findings |
| pre-existing atlas-socket vet | N/A | N/A | `stdmethods` warning on request/reader.go/response/writer.go is unchanged on main — not a branch defect |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending the RR-6 rebase-time doc follow-up in Task 12 Step 4)

## Action Items

1. At pre-PR rebase, re-run Task 12 Step 4: if the reliability-review rewrite has landed on
   main, add the "Resolved by task-115" line under the RR-6 heading in
   `docs/architectural-improvements.md`. (Non-blocking; explicitly conditional in the plan.)
