# Plan Audit — task-259-mob-skill-aoe-targeting

**Plan Path:** docs/tasks/task-259-mob-skill-aoe-targeting/plan.md
**Audit Date:** 2026-08-21
**Branch:** task-259-mob-skill-aoe-targeting
**Base Branch:** main (implementation range 33e4cc7..HEAD)

## Executive Summary

All 4 plan tasks were implemented essentially verbatim against the plan's prescribed code, including exact file lists, function signatures, and test names. The two accepted controller rulings (Ruling 6: `diseaseTargetTenant()` test helper; Ruling 8: `routine.Go` instead of a bare `go` statement) are both present and correctly scoped. `go build ./...` and `go test ./... -count=1 -race` pass cleanly for the full `atlas-monsters` module, including the new `character/position`, `monster/disease_targets*`, and `monster/disease_callers_test.go` packages. Final-verification greps confirm `rand.Shuffle` is gone (only `rand.Intn` remains), no literal `128` was introduced for seduce, and `golang.org/x/sync` stays an indirect dependency (no `errgroup`).

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | character/position REST client | DONE | `character/position/{rest,requests,processor,processor_test,mock/processor}.go` all present and match plan code verbatim; commit `0ff80b1`; `go test ./character/position/...` passes. |
| 2 | pure box-scoped target selector | DONE | `monster/disease_targets.go` `selectDiseaseTargets`/`positionedCharacter`; `mobskill/builder.go` `SetCount` added and wired into `Build()`; `monster/disease_targets_test.go` contains all 10 planned table cases plus `TestSelectDiseaseTargets_IsDeterministic`; commit `08f8567`. |
| 3 | I/O shell and positionFn seam | DONE | `ProcessorImpl.positionFn` field + wiring in `NewProcessor` (`monster/processor.go:103,128-130`); old `getDiseaseTargets`/`rand.Shuffle` deleted; `skillId` threaded through `executeDebuff`/`executeBanish`/`executeDispel`; `resolvePositions` + new `getDiseaseTargets` in `disease_targets.go`; all 8 planned `TestGetDiseaseTargets_*` shell tests present in `disease_targets_shell_test.go`; commit `51c5242`. Ruling-8 deviation: fan-out uses `routine.Go(p.l, p.ctx, func(_ context.Context) {...})` instead of a bare `go` statement — accepted per controller ruling, WaitGroup+buffered-channel semaphore semantics unchanged. |
| 4 | caller inheritance — banish/dispel | DONE | `information/builder.go` `SetBanish` added; `executeDebuff`/`executeBanish`/`executeDispel` emissions routed through `p.emit(...)` instead of `producer.ProviderImpl(...)`; `testInformationLookup` guard added to `executeBanish`; all 4 planned tests present in `disease_callers_test.go`; commit `9b9c0b9`. |

**Completion Rate:** 4/4 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. The one PRD acceptance criterion excluded from this branch (updating `docs/research/missing-features/monsters-and-bosses.md`) is covered by controller Ruling 2 and is explicitly marked out-of-scope in the plan's Global Constraints section, not a silent skip.

Two additional commits beyond the plan's four (`f5fa66c` and `056e151`) address controller-ruling fix-ups (Ruling 6 and Ruling 8 respectively) and are consistent with — not divergent from — the plan's task 3 content; they refine test/implementation details already scoped to task 3 rather than introducing new, unplanned work.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-monsters (services/atlas-monsters/atlas.com/monsters) | PASS | PASS | `go build ./...` clean; `go test ./... -count=1 -race` — all packages `ok`, including `character/position` (1.3s), `monster` (38.2s, includes new disease-target tests), `monster/information` (17.0s); no race reports. |

## Final-Verification Grep Checks (plan's own gate)

- `rand.Shuffle` removed; only `rand.Intn` (damage formula, `monster/processor.go:712,714`) remains — confirmed.
- No literal `128` for seduce; the only `SkillTypeSeduce` hit outside tests is `monster/disease_targets.go:43`, using `monster2.SkillTypeSeduce` — confirmed.
- `golang.org/x/sync v0.22.0 // indirect` in `go.mod`; no `errgroup` import anywhere in the module — confirmed.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None. No blocking findings.
