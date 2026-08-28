# Plan Audit — task-279-carnival-mob-skills

**Plan Path:** docs/tasks/task-279-carnival-mob-skills/plan.md
**Audit Date:** 2026-08-28
**Branch:** task-279-carnival-mob-skills
**Base Branch:** main (merge base bda6566f3)

## Executive Summary

All 7 plan tasks were faithfully executed, with production code matching the plan's prescribed
diffs almost verbatim (identical case-arm text, identical helper function, identical comments).
Both controller rulings (N1: fresh monster per skill id in the carnival warning-suppression
loop; N2: SEAL_SKILL rejection log assertion CONTAINS "SEAL_SKILL", SEAL rejection assertion
CONTAINS "SEAL" but NOT "SEAL_SKILL") are present in the code exactly as ruled. Both touched
modules (`libs/atlas-constants`, `services/atlas-monsters/atlas.com/monsters`) build and test
clean (`go build ./...` and `go test ./... -count=1`, all packages `ok`), and the flagless
`tools/verify.sh` gate log (`.superpowers/sdd/plan/gates/gate-final-2.log`) confirms "All checks
passed." across all 91 modules. No skipped or partial tasks were found.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Map/classify/name carnival mob skills 150-157 in `libs/atlas-constants` | DONE | `libs/atlas-constants/monster/skill.go` diff (commit ef0cb9b7c) matches plan Step 3-5 verbatim: `SkillTypeToStatusName` gains `SkillTypeCarnivalPAD/MAD/PDR/MDR` folded into existing arms, plus new arms for `CarnivalACC→Accuracy`, `CarnivalEVA→Avoidability`, `CarnivalSealSkill→SealSkill`; `IsAoeSkill` gains the 8 carnival types plus FR-5.3 divergence comment verbatim; `skillNameMap` gains all 8 `CARNIVAL_*` entries. Tests: `skill_test.go` has all 8 plan-named tests (`TestSkillTypeToStatusName_Carnival`, `TestSkillTypeToStatusName_SharedStatArms`, `TestIsAoeSkill_CarnivalAndRegressions`, `TestSkillNameToId_Carnival`, `TestSkillTypeNames_IncludesCarnival`, `TestSkillCategory_Carnival`). `go build`/`go test` pass. |
| 2 | Dispatch `CARNIVAL_BUFF` through `executeStatBuff` in `UseSkill`/`UseSkillGM` | DONE | `processor.go` diff (commit 9612616a7) adds `monster2.SkillCategoryCarnivalBuf` to both switch case arms at the plan-specified sites, adds the `testMobSkillLookup` seam to `UseSkillGM` (processor.go:1022-1027), and updates the doc comment at :90. New file `carnival_skill_test.go` contains `TestUseSkill_Carnival_AppliesMappedStatus`, `TestUseSkillGM_Carnival_AppliesMappedStatus`, `TestUseSkill_Carnival_NoUnknownCategoryWarning`, `TestExecuteStatBuff_Carnival_NoOppositeImmunityPrecancel_NotReflect` — all matching plan's table-driven design. **Ruling N1 confirmed**: `TestUseSkill_Carnival_NoUnknownCategoryWarning` creates a fresh monster per skill id via `newCarnivalTestProcessor` inside the `for _, id := range carnivalIds` loop (carnival_skill_test.go:163-197), with a doc comment explicitly stating why (SEAL_SKILL from an earlier subtest must not block a later cast in the same loop). |
| 3 | Pin buff value, duration, recast, and AoE behavior in tests (no production change) | DONE | New file `carnival_value_test.go` (commit 685e1f607) contains exactly the 5 plan-specified tests: `TestExecuteStatBuff_Carnival_NegativeXSurvives` (FR-4.3, -990 survives), `TestExecuteStatBuff_Carnival_DurationIsMilliseconds` (FR-4.4), `TestExecuteStatBuff_Carnival_RecastRefreshesValueAndExpiry` (FR-4.1), `TestExecuteStatBuff_Carnival_NoBoundingBox_CasterOnly` (FR-5.2), `TestExecuteStatBuff_Carnival_WithBoundingBox_InBoxOnly`. No production diff in this commit (verified: `git diff bda6566f3..23b2f2b6a -- ...processor.go` shows only Task 2/5 hunks, none introduced in 685e1f607). |
| 4 | `skillSuppressingStatus` helper + picker gate on SEAL_SKILL | DONE | `picker.go` diff (commit b5eceb779) adds `skillSuppressingStatus(m Model) monster2.TemporaryStatType` immediately after `isPickerRelevantStatus`, with the exact doc comment and SEAL_SKILL-checked-first logic from the plan; replaces the bare `"SEAL"` check at the sentinel gate with `if st := skillSuppressingStatus(m); st != ""`. `picker_test.go` gains all 5 plan-named tests: `TestPicker_SealSkillMonster_ReturnsSentinel`, `TestPicker_SealMonster_StillReturnsSentinel`, `TestPicker_SealSkillAndSeal_ReturnsSentinel`, `TestPicker_HsalfSkill156NoProp_ReturnsSentinel`, `TestSkillSuppressingStatus` (5-case table matching plan exactly, including "both → SealSkill wins"). |
| 5 | `UseSkill` executor gate on SEAL_SKILL, named blocking status | DONE | `processor.go` diff (commits 558d822b2 + fix 23b2f2b6a) replaces the bare `"SEAL"` check at :866 with `if st := skillSuppressingStatus(m); st != ""` and the exact plan comment. New file `seal_skill_test.go` contains `TestUseSkill_SealSkill_RejectsAndLogsDistinctly`, `TestUseSkill_Seal_StillRejects`, `TestUseBasicAttack_SealSkill_StillSucceeds` (FR-6.6), `TestUseSkill_Skill157ThenAnySkill_RejectedEndToEnd`. **Ruling N2 confirmed**: `TestUseSkill_SealSkill_RejectsAndLogsDistinctly` asserts a logged message `strings.Contains(e.Message, "SEAL_SKILL")` (seal_skill_test.go:75-83); `TestUseSkill_Seal_StillRejects` asserts a message containing `"SEAL"` (line ~156) AND explicitly asserts `!strings.Contains(matched, "SEAL_SKILL")` (line ~172-174) on that same matched entry — this is precisely the fix landed in 23b2f2b6a per progress.md. Scope fact confirmed: `UseSkillGM` (processor.go:1015-1044) has no seal/SEAL_SKILL gate — intentional, matches plan's "add no gate to UseSkillGM" instruction. |
| 6 | Update `atlas-monsters/docs/domain.md` | DONE | Diff (commit f1beab939) updates the `UseSkill` numbered step 1 to describe rejection on `SEAL` or `SEAL_SKILL` with distinct logging and pre-animation-delay timing; updates step 7 to add `CARNIVAL_BUFF` (mob skill types 150-157) to the stat-buff dispatch group; updates the Skill Picker section item 1 ("sealed" → "carries `SEAL` or `SEAL_SKILL`") and item 4 (adds the FR-6.5 clause that SEAL_SKILL now gates the picker itself, not just triggering an inert re-pick). All repo-relative, no absolute paths, no invented claims. **Scope fact confirmed**: `docs/research/missing-features/monsters-and-bosses.md` does not exist in the repo (`git ls-files \| grep -i monsters-and-bosses` returns nothing) — correctly not created, matching the plan's explicit "criterion dropped" scope note. |
| 7 | Full verification gate + code review before PR | DONE | `.superpowers/sdd/plan/gates/gate-final-2.log` ends "All checks passed." across all 91 modules (flagless `tools/verify.sh`, full merge-base scope bda6566f3..23b2f2b6a), per `progress.md`'s "Branch-end gate, attempt 2 — PASS" entry. `progress.md` records Task 5's fix commit 23b2f2b6a was itself re-reviewed and APPROVED (0 blocking/non-blocking/not-evaluable). Independently re-verified in this audit: both modules build and test clean (see Build & Test Results below). |

**Completion Rate:** 7/7 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. The one PRD acceptance criterion that was dropped (`docs/research/missing-features/monsters-and-bosses.md` §8 update) was dropped by explicit user decision recorded in the plan's Task 6 scope note, not silently skipped, and is called out as a deliberate non-finding per this audit's brief.

## PRD FR-* Mapping

| FR | Requirement | Where satisfied |
|---|---|---|
| FR-1.1 | Map 150-157 to correct `TemporaryStatType` | `libs/atlas-constants/monster/skill.go` `SkillTypeToStatusName`; `TestSkillTypeToStatusName_Carnival` |
| FR-1.2 | 150/152 share arms with 100/110, 102/112 | `skill.go` — carnival ids folded into existing case arms, not new arms; `TestSkillTypeToStatusName_SharedStatArms` |
| FR-1.3 | Unmapped ids still return `""` | `default: return ""` unchanged; regression covered in `TestSkillTypeToStatusName_Carnival` (id 149) |
| FR-2.1/2.2/2.3 | 8 `CARNIVAL_*` names registered | `skillNameMap` entries; `TestSkillNameToId_Carnival`, `TestSkillTypeNames_IncludesCarnival` |
| FR-3.1/3.2 | `CarnivalBuf` dispatches to `executeStatBuff`, no new executor | `processor.go` UseSkill/UseSkillGM switch arms |
| FR-3.3 | No opposite-immunity precancel, no reflect | `TestExecuteStatBuff_Carnival_NoOppositeImmunityPrecancel_NotReflect` |
| FR-3.4 | `default:` warning arm untouched | verified in diff — no change to default case |
| FR-4.1/4.2 | Recast refreshes, not rejected | `TestExecuteStatBuff_Carnival_RecastRefreshesValueAndExpiry` |
| FR-4.3 | No clamping (-990 survives) | `TestExecuteStatBuff_Carnival_NegativeXSurvives` |
| FR-4.4 | Duration in ms, unchanged conversion | `TestExecuteStatBuff_Carnival_DurationIsMilliseconds` |
| FR-5.1/5.2/5.3 | AoE true for 150-157, box gate unchanged, deviation documented | `IsAoeSkill` diff + comment; `TestIsAoeSkill_CarnivalAndRegressions`; `TestExecuteStatBuff_Carnival_NoBoundingBox_CasterOnly` / `_WithBoundingBox_InBoxOnly` |
| FR-6.1/6.2 | Picker gate rejects SEAL_SKILL | `skillSuppressingStatus` + picker.go gate; `TestPicker_SealSkillMonster_ReturnsSentinel` |
| FR-6.3 | UseSkill gate rejects SEAL_SKILL | processor.go:866 gate; `TestUseSkill_SealSkill_RejectsAndLogsDistinctly`, `TestUseSkill_Skill157ThenAnySkill_RejectedEndToEnd` |
| FR-6.4 | Symbols not string literals | `monster2.TemporaryStatTypeSealSkill`/`TemporaryStatTypeSeal` used throughout, no new bare `"SEAL"`/`"SEAL_SKILL"` literals in production code |
| FR-6.5 | No change needed to `pickerRelevantStatuses` | Confirmed — `picker.go` diff touches only the gate site and adds the helper, not `pickerRelevantStatuses` |
| FR-6.6 | `UseBasicAttack` unaffected | `TestUseBasicAttack_SealSkill_StillSucceeds` |
| FR-7.1 | No "unknown skill category" warning | `TestUseSkill_Carnival_NoUnknownCategoryWarning` |
| FR-7.2 | SEAL_SKILL rejection log distinct from SEAL | `TestUseSkill_SealSkill_RejectsAndLogsDistinctly` + `TestUseSkill_Seal_StillRejects` (Ruling N2) |

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| libs/atlas-constants | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` — all packages `ok`, including `monster` package |
| services/atlas-monsters/atlas.com/monsters | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` — all packages `ok` (`monster` package 25.6s, includes all new carnival/seal_skill/picker tests) |
| Full repo (flagless verify.sh) | PASS | PASS | `.superpowers/sdd/plan/gates/gate-final-2.log`: "All checks passed." across 91 modules, `-race` included, docker bake included per `progress.md` |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None. All 7 plan tasks are DONE with direct file:line/commit evidence, both controller rulings (N1, N2) are correctly implemented, both documented scope facts (no `UseSkillGM` gate, no `monsters-and-bosses.md` file) are correctly honored as intentional non-findings, and independent build/test verification plus the recorded flagless `tools/verify.sh` pass confirm the branch is verified.
