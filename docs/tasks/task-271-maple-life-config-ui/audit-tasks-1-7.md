# Plan Audit — task-271-maple-life-config-ui (Tasks 1-7)

**Plan Path:** docs/tasks/task-271-maple-life-config-ui/plan.md
**Audit Date:** 2026-08-27
**Branch:** task-271-maple-life-config-ui
**Base Branch:** main (merge base 4358e586f, HEAD b9c859bc9)
**Scope:** Tasks 1-7 only (Tasks 8-13 audited by a separate shard)

## Executive Summary

All seven audited tasks were implemented as specified, with each task's
implementer report, TDD (RED/GREEN) evidence, and per-unit `task-reviewer`
verdict confirmed against the current code. Task 1 had one initial review
finding (an invented `SEED_ML` fixture instead of the canonical shipped-seed
row) which was fixed and re-verified (commit `602afa07b`); Task 4 had one
lint/coverage finding (a discarded reducer return value masking a real
coverage gap) which was fixed and re-reviewed as ADDRESSED (commit
`3227e7285`). Every other task was APPROVED on first review with no blocking
findings. All targeted `vitest` runs for the files these seven tasks touch
(194 tests across 18 files) pass on the current worktree HEAD. Two
non-blocking notes carried forward from earlier reviews (Task 4's lenient
`parseSpPool`, Task 6's variant-loadout test-coverage gap that was
subsequently fixed) are recorded below as already-ruled, per the shard brief.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Domain types, entry-type rename, PATCH round-trip preservation tests | DONE | `services/atlas-ui/src/types/models/template.ts:32,37,73,81,90,107,163` — `EquipmentEntry`/`InventoryEntry` renamed in place (no aliases), six Maple Life interfaces added, `TemplateAttributes.mapleLife?` added. `tenants.service.ts:9,133` — `mapleLife?: MapleLifeConfig` added via single `import type`. New `tenants.service.test.ts` + amended `templates-update.test.ts` prove PATCH preservation (commit `218a048a5`). Initial review flagged an invented fixture; fixed to the canonical shipped-seed row in `602afa07b`, cross-checked against `services/atlas-configurations/seed-data/templates/template_gms_83_1.json` (`task-1-report.md` fix-round-1 section); `task-1-rereview.md:5` — "Open finding verdict: ADDRESSED". |
| 2 | `DetailActionBarConfig.blockingIssues` | DONE | `DetailActionBarContext.tsx:26,71,82,87,102` — field added, threaded through `register()`, added to the effect's dependency list (the exact bug the brief's regression test targets), forwarded to `<SaveBar>`. `SaveBar.tsx:19,27,30-32,39-46,63` — Save disabled and status text swapped when `blockingIssues > 0`; Discard left untouched. New `SaveBar.test.tsx` (6 cases) + 2 new `DetailActionBarContext.test.tsx` cases. `task-2-review.md` — "No blocking or non-blocking findings", full spec compliance confirmed at file:line. |
| 3 | Generalise `AppearancePoolSection` off `CharacterTemplate` | DONE | `AppearancePoolSection.tsx:9-27` — new caller-supplied `pool`/`selectedIndex`/`variantLoadout`/`onPick(index)`/`description` props, all three prior `CharacterTemplate` couplings removed. `CharacterTemplatesEditor.tsx` call site reconstructs identical rendered output (per `task-3-report.md`). Test file rewritten to new props including the two-vs-one-argument `onPick` fix the design doc got wrong. `task-3-review.md` — "APPROVED. No blocking findings, no non-blocking findings." |
| 4 | `mapleLifeEditorState.ts` — draft shape, reducer, projection | DONE | `mapleLifeEditorState.ts` — full **Produces** contract present (`ORDINAL_COUNT`, `GENDER_COUNT`, `MapleLifeClassDraft`, `MapleLifeEditorState`, `MapleLifeAction`, `initialMapleLifeState`, `mapleLifeReducer`, `projectForSave`, `isDirty`, `isEmptyConfig`, etc.), all 6 key design decisions honoured (baseline-as-projection, Go JSON-tag field order, `spSkillId` omission not zeroing, `sp` verbatim-vs-parsed serialisation, single materialisation site, `seedFromTemplate` deep clone). 30/30 tests pass. Lint finding (discarded reducer return, masking a real coverage gap in the "deep-clones the donor" test) fixed in commit `3227e7285` by asserting both halves of the deep-clone property; `task-4-rereview.md:14,41` — "Open finding verdict: ADDRESSED... genuinely exercises the addEquipment mutation". |
| 5 | `maple-life.schema.ts` — hard validation rules | DONE | `src/lib/schemas/maple-life.schema.ts` — all rules (FR-11.1/2/3/4/5/6/7) placed exactly per the brief's table, `MSG` verbatim, `parseSpPool` reused from Task 4 rather than re-derived, `validateMapleLife`/`IssueMap` present. 24/24 tests pass. One documented, correctly-scoped deviation (`.transform` to satisfy `exactOptionalPropertyTypes` for `spSkillId` omission) verified by the reviewer via direct source inspection. `task-5-review.md:4` — "Verdict: APPROVED"; "Findings: None blocking. None non-blocking beyond what's noted". |
| 6 | Soft warnings and hair composition (`mapleLifeWarnings.ts`, `mapleLifeLoadout.ts`) | DONE | Both new modules present with verbatim `WARN`/`KNOWN_SP_SKILL_IDS`/`SP_SKILL_LABELS` text, `composeHair` using `Math.floor`, `buildMapleLifeLoadout`/`buildMapleLifeVariantLoadout`/`combinationCount`. 53/53 tests pass (combined with Task 4's suite). Reviewer's one non-blocking coverage-gap finding (variant-substitution tests pinned only the changed field, not the "all other fields unchanged" property) was fixed with a falsification-proven `it.each` rewrite (verified: deliberately broke the production code, watched 4 tests fail, restored, watched them pass — `task-6-report.md` fix section) — already-ruled per shard brief, noted here as resolved, not re-raised. |
| 7 | `ClassSelector` and `IdentitySection` | DONE | `ClassSelector.tsx` — two roving-tabindex `tablist`s (Gender, Class ordinal), live `jobNameById` lookup for ordinal labels (never a static table, FR-5.2), `derived`/`unconfirmed` badges. `IdentitySection.tsx` — job `Select`/`Input` fallback on `isPending`/`isError`, non-clamping `level` input via `useSyncedNumberInput` (FR-5.5), persistent `role="note"` provenance block for ordinals 2-4 with no dismiss affordance (verified by an explicit absent-button test). 75/75 tests pass across the `maple-life` directory. `task-7-review.md` — "No blocking defects found... Scoped test and lint runs are green." |

**Completion Rate:** 7/7 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All seven tasks in this shard's range are fully implemented, tested,
and reviewed as APPROVED (with two findings raised and fixed in-branch: Task
1's fixture and Task 4's lint/coverage gap, both re-reviewed as ADDRESSED).

## Already-Ruled Items (not re-raised)

Per the shard brief, the following prior findings are not re-raised as new
findings:

- Task 4's `parseSpPool` using lenient `Number.parseInt` — spec-directed per
  the brief's own wording (`task-4-review.md` non-blocking note).
- `isDirty`'s order-sensitive `JSON.stringify` comparison — verified to hold
  because `projectForSave`/`projectClass` write fields via explicit object
  literals, never rest-spread (`task-4-review.md` non-blocking note).
- Task 6's variant-loadout test-coverage gap — proven fixed by deliberate
  production-code breakage and restoration (`task-6-report.md` fix section).

## Build & Test Results

Full `tools/verify.sh` was not run per the shard brief (already run and
recorded as passing over commit `e36cbc0bf` in `progress.md`). Targeted
`vitest` runs covering every file touched by Tasks 1-7:

| Scope | Command | Result |
|---|---|---|
| `maple-life/` dir + schema + AppearancePoolSection + SaveBar + DetailActionBarContext + tenants.service + templates-update tests | `npx vitest run <18 files>` from `services/atlas-ui` | 18 test files / 194 tests, all PASS |

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-ui | Not re-run (reported PASS per implementer/reviewer reports; concurrent full-suite gate already passed at `e36cbc0bf`) | PASS (194/194 targeted) | No Go files touched; frontend-only branch. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (for this task range; final merge
  readiness also depends on the Tasks 8-13 shard's findings)

## Action Items

None. No fixes required for Tasks 1-7.
