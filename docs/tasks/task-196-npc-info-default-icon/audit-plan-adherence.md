# Plan Audit — task-196-npc-info-default-icon

**Plan Path:** docs/tasks/task-196-npc-info-default-icon/plan.md
**Audit Date:** 2026-08-06
**Branch:** task-196-npc-info-default-icon
**Base:** 31c7a664f (main tip at branch start)
**Diff Scope:** `31c7a664f..HEAD` (6 commits)

## Executive Summary

All four plan tasks were faithfully executed. The change surface is exactly
what Task 4 Step 6 asserts — `libs/atlas-wz/icons/{extract.go, fixture_test.go,
npc_default_test.go, npc_default_edge_test.go}` plus the three task docs,
nothing under `services/`, no `go.mod`/`go.sum` touched. Every code block in
the plan (fixture helpers, the two Task 2 tests, `findNpcCanvas`, the
`ExtractNpcIcon` rewrite, the three Task 3 edge tests) landed byte-for-byte,
including the documented `f.Close()` compile-fix deviation. `go build`,
`go vet`, and `go test ./...` are clean in `libs/atlas-wz`; all 8 `icons`
package tests pass. `tools/redis-key-guard.sh` and `tools/goroutine-guard.sh`
both exit 0. The only cosmetic gap: the plan.md checkboxes were never flipped
to `[x]` even though the corresponding commits exist — a doc-hygiene miss,
not a functional one.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Fixture scaffolding (`fixture_test.go`) | DONE | Commit `0353ec45a`. File matches plan verbatim except `t.Cleanup(func() { f.Close() })` at fixture_test.go:66 vs. plan's `_ = f.Close()` — expected deviation, since `wz.File.Close()` returns no value and `_ =` would not compile. `markDefault/markStand/markLink/markTopLevel`, `payloadFor`, `pixelPayload`, `openFixture`, `newArchive`, `pixelAt` all present as specified (fixture_test.go:22-85). |
| 2 | `findNpcCanvas` fix | DONE | Commit `2ce0359af`. `extract.go:239-243` rewrites `ExtractNpcIcon` to call `findNpcCanvas`; `extract.go:330-343` adds `findNpcCanvas` immediately above `findStandCanvas` (extract.go:346), matching the plan's placement and doc comment word-for-word. `npc_default_test.go` (commit) contains `TestNpcPrefersInfoDefault` and `TestNpcFallsBackToStand`, identical to the plan's Step 1 code. Both tests pass (verified below). `ExtractMobIcon` (extract.go:100) and `ExtractReactorIcon` (extract.go:106) are untouched — confirmed via `git diff` showing only the `ExtractNpcIcon` doc/body and the new `findNpcCanvas` function changed. |
| 3 | Edge-case guard tests | DONE | Commit `abd3d0ae8`. `npc_default_edge_test.go` contains all four tests specified: `TestMobIgnoresInfoDefault`, `TestNpcIgnoresTopLevelDefaultDir`, `TestNpcInfoDefaultBeatsLink`, `TestNpcFollowsLinkToInfoDefault` — byte-identical to the plan's code block. All pass. |
| 4 | Full verification sweep | DONE (verification-only, no commit expected) | Re-ran independently: `cd libs/atlas-wz && go vet ./... && go build ./... && go test ./...` all exit 0; `go test ./icons/... -v` reports 8/8 passed (TestPublicSurfaceExists, TestNormalizeId, TestNpcPrefersInfoDefault, TestNpcFallsBackToStand, TestMobIgnoresInfoDefault, TestNpcIgnoresTopLevelDefaultDir, TestNpcInfoDefaultBeatsLink, TestNpcFollowsLinkToInfoDefault). `tools/redis-key-guard.sh` exit 0, `tools/goroutine-guard.sh` exit 0. `git diff --name-only main...HEAD -- '*go.mod' '*go.sum'` empty (Step 5 holds). `git diff --stat main...HEAD` shows exactly the 4 icons files + 3 task docs, nothing under `services/` (Step 6 holds, see below). No commit exists for Task 4, matching the plan's own description of it as "verification only, no commit." |

**Completion Rate:** 4/4 tasks (100%) — functionally. Note: plan.md's own
`- [ ]` checkboxes (19 step-level items) were never edited to `- [x]` in any
of the 6 commits, so the plan document itself does not visibly self-report
completion, even though every step's artifact exists in the tree/history.
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All four tasks have direct commit evidence and matching file content.
The plan's "Post-merge: the deploy step" section is explicitly out of scope
for this branch (WZ re-ingest is an operational step, not a code task) and is
correctly not represented by any commit.

## Change-Surface Verification (Plan Task 4, Step 6)

```
$ git diff --stat main...HEAD
 docs/tasks/task-196-npc-info-default-icon/context.md |  107 +++
 docs/tasks/task-196-npc-info-default-icon/design.md  |  194 +++++
 docs/tasks/task-196-npc-info-default-icon/plan.md    |  500 +++++++++
 libs/atlas-wz/icons/extract.go                       |   20 +-
 libs/atlas-wz/icons/fixture_test.go                  |   85 ++
 libs/atlas-wz/icons/npc_default_edge_test.go         |   91 ++
 libs/atlas-wz/icons/npc_default_test.go              |   45 ++
 7 files changed, 1039 insertions(+), 3 deletions(-)
```

Exactly matches the plan's assertion: the 3 task docs, plus exactly the four
named files under `libs/atlas-wz/icons/`, nothing under `services/`.
`git diff --name-only main...HEAD -- '*go.mod' '*go.sum'` returned empty, so
Step 5's mandatory-bake trigger is correctly not activated.

## Build & Test Results

| Module | Build | Vet | Test | Notes |
|--------|-------|-----|------|-------|
| libs/atlas-wz | PASS | PASS | PASS | `go build ./...`, `go vet ./...`, `go test ./...` all exit 0. `icons` package: 8/8 tests pass (`go test ./icons/... -v -count=1`). |
| repo-root guards | — | — | PASS | `tools/redis-key-guard.sh` exit 0; `tools/goroutine-guard.sh` exit 0 (both scan the whole tree, unaffected by this change as the plan expected). |

Per the task instructions, the full `-race` suite and `services/atlas-data`
were already verified in a prior pass and were not re-run here.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

1. (Cosmetic, non-blocking) Flip plan.md's `- [ ]` checkboxes to `- [x]` for
   the 19 completed steps, or note in the plan that checkbox-tracking was
   skipped in favor of commit-per-task — the plan document currently
   understates its own completion when read in isolation.
