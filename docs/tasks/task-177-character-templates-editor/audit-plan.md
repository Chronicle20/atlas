# Plan Audit — task-177-character-templates-editor

**Plan Path:** docs/tasks/task-177-character-templates-editor/plan.md
**Audit Date:** 2026-07-18
**Branch:** task-177-character-templates-editor
**Base Branch:** main (base commit 092069bce, head 391a4f979)

## Executive Summary

All 17 plan tasks are implemented and verifiable in the diff — every planned file exists, both duplicated legacy forms (`tenants-character-templates-form.tsx`, `templates-character-templates-form.tsx`, -647 lines combined) are deleted, and both page wrappers (`TenantsCharacterTemplatesPage.tsx`, `TemplatesCharacterTemplatesPage.tsx`) render the shared `CharacterTemplatesEditor` through a clean `TemplatesEditorAdapter`. All five explicitly-dropped features (selector thumbnails, Edit-as-JSON, id/value table, Creator-Preview mode, skill browser) are confirmed absent by direct source inspection. A spot-check test run of the 16 new/changed template test files (77 tests) passed; the full-suite/build/lint gates were already verified green this session per the task brief and are not re-run here. No stubs, TODOs, or silently-skipped work were found. This branch is ready for PR.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Template labels (`jobNames.ts`) | DONE | `src/components/features/characters/templates/jobNames.ts` (46 lines) + `__tests__/jobNames.test.ts` (62 lines, exact API: `worldNameFromJobIndex`, `genderLabel`, `templateLabels`, `KNOWN_CLASSES`). |
| 2 | Editor state reducer (`editorState.ts`) | DONE | `editorState.ts` (267 lines) + `__tests__/editorState.test.ts` (150 lines). Progress log documents a self-caught `noUncheckedIndexedAccess` build fix (commit `3913ce722`) — build gate enforced, not skipped. |
| 3 | Loadout builders + `isFemaleCosmeticId` | DONE | `previewLoadout.ts` (75 lines); `characterRender.service.ts` diff confirms `isFemaleCosmeticId` extracted and `resolveGender` now calls it (behavior-preserving, verified by diff at services/atlas-ui/src/services/api/characterRender.service.ts:53-59). |
| 4 | Cosmetics enumeration service + hooks | DONE | `src/services/api/cosmetics.service.ts` + `src/lib/hooks/api/useCosmetics.ts` + `__tests__/cosmetics.service.test.ts`. Correctly NOT added to the legacy hooks barrel (noted deliberate in progress.md). |
| 5 | Batched item-name hook (`useItemNames.ts`) | DONE | `src/lib/hooks/api/useItemNames.ts` + `__tests__/useItemNames.test.tsx`, shares `itemStringKeys` cache with `useItemName` as specified. |
| 6 | Popover primitive + pool search configs + `ItemSearchCombobox` | DONE | `src/components/ui/popover.tsx`, `poolSearchConfig.ts`, `ItemSearchCombobox.tsx` (211 lines) + test (248 lines). `package.json` diff shows exactly one new dependency, `@radix-ui/react-popover@^1.1.19` — no `cmdk`, no other additions (global constraint honored). Progress log documents an in-loop debounce regression caught by review and fixed before task close (commits `0d59baf6d`+`0a4230f02`). |
| 7 | `MapPicker` | DONE | `MapPicker.tsx` (135 lines) + test. Warning hint now uses `text-warning-foreground` token (Task 17 cleanup applied, verified at MapPicker.tsx:128 — no hardcoded amber). |
| 8 | `TemplateSelector` | DONE | `TemplateSelector.tsx`: `role="tablist"` (line 26), `role="tab"` + `aria-selected` (lines 34-35), `+ New` button (line 52). No `<img>`/thumbnail markup — confirms FR-2.2 "no thumbnails in the selector" by direct grep. |
| 9 | `TemplateActionsMenu` | DONE | `TemplateActionsMenu.tsx` — Duplicate + Remove-with-confirm via `AlertDialog`, confirm copy exactly matches FR-3.1 wording ("players can no longer create this class/gender..."). No "Edit as JSON" item (verified by reading full file). |
| 10 | `IdentitySection` | DONE | `IdentitySection.tsx` — Class select (`KNOWN_CLASSES`) + Advanced numeric jobIndex/subJobIndex fields (lines 88-119), Gender select, `MapPicker` wired to `onSetIdentity("mapId", ...)` (line 138-141). |
| 11 | `AppearanceThumb` + `AppearancePoolSection` | DONE | `AppearanceThumb.tsx` exports `THUMB_SIZE=76`/`THUMB_OFFSET_X=-74`/`THUMB_OFFSET_Y=-70`; `disabled={marked}` on the thumb button blocks double-add (line 42, satisfies FR-5.4). `AppearancePoolSection.tsx` reuses `THUMB_SIZE` for the Add button (Task 17 cleanup applied, line 90) rather than a hardcoded `h-[76px]`. Warning uses `text-warning-foreground` (line 59). |
| 12 | `AppearanceBrowserDialog` | DONE | `AppearanceBrowserDialog.tsx` (190 lines) — paginated grid (`PAGE_SIZE=24`), gender filter with "Show all genders" `Switch` and opposite-gender-first reorder on show-all (documented bug-fix, lines 79-84), already-in-pool marking via `AppearanceThumb`'s `marked` prop, names resolved via `useItemNames` only for faces/hairs (never search-indexed, per FR-5.4 note). |
| 13 | `ItemRow` + `EquipmentPoolSection` + `StartingKitSection` | DONE | All three files present; `StartingKitSection.tsx` confirmed: items via `ItemSearchCombobox`, skills via numeric input + `useSkillData` name/icon lookup, empty-skills copy exact match ("This class starts with no granted skills."), no skill browser (FR-7.2 honored). |
| 14 | `PreviewCard` | DONE | `PreviewCard.tsx` (135 lines) + test; grep for `table`/`id/value` inside the file returns no match — confirms FR-8.1 "No id/value table" (user decision) is honored. |
| 15 | `SaveBar` + `CharacterTemplatesEditor` assembly | DONE | `SaveBar.tsx` — dirty text, Discard (disabled unless dirty) → confirm `AlertDialog` → `onDiscard`, Save (disabled unless dirty or saving). `CharacterTemplatesEditor.tsx` (294 lines) wires all sections, seed-once guard (`!state.loaded`), `?tpl=` URL sync with clamp-to-0 on bad input, and a dedicated `discardChanges` handler that re-syncs the URL against `state.baseline.length` — this is the documented Important-severity fix from the review loop (commit `0d39bd4e0`), confirmed present in the current file (lines 140-152). |
| 16 | Page wrappers + delete duplicated forms | DONE | `git diff --stat` confirms `src/pages/tenants-character-templates-form.tsx` (-325) and `src/pages/templates-character-templates-form.tsx` (-322) deleted. Both `TenantsCharacterTemplatesPage.tsx` and `TemplatesCharacterTemplatesPage.tsx` build a `TemplatesEditorAdapter` from their existing hooks (`useTenantConfiguration`/`useUpdateTenantConfiguration` and `useTemplate`/`useUpdateTemplate` respectively) and render `<CharacterTemplatesEditor adapter={adapter} />`. `grep -rn` for the deleted filenames anywhere under `src/` returns zero hits — no dangling references. `App.tsx` has an empty diff (routes/nav untouched, per global constraints). |
| 17 | Full verification gates + visual tuning | DONE (with disclosed exception) | Per progress.md and this session's task brief: `npm run test` (1094 tests), `tsc -b` + vite build (0 TS errors), `npm run lint` (no new errors), `tools/lint.sh --check` (task-177-clean) all green. Three roll-up cleanups applied and verified above (warning token, THUMB_SIZE reuse, stale comment fix). Step 3 "visual smoke in a live dev server" was explicitly **not** performed (no browser tool available) — this is honestly disclosed in progress.md line 55 rather than falsely claimed, per the plan's own instruction ("If no live environment is reachable, state that explicitly ... do NOT claim visual verification that didn't happen"). Crop offsets left at prototype defaults (-74/-70) as a flagged manual-QA follow-up, not a silent gap. |

**Completion Rate:** 17/17 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0 (Task 17's visual-smoke sub-step is an explicitly disclosed, plan-sanctioned exception — not a silent partial)

## Skipped / Deferred Tasks

None skipped. The only unexecuted plan *sub-step* is Task 17 Step 3 (interactive dev-server visual smoke test), which requires a live ingress/browser session unavailable to the agentic workers. The plan itself anticipates this exact scenario and instructs explicit disclosure rather than a false claim; the implementer complied (progress.md line 55: "Visual smoke NOT done [no browser tool]"). Impact: crop-offset tuning (`THUMB_OFFSET_X`/`THUMB_OFFSET_Y` in `AppearanceThumb.tsx`) and general light/dark/appearance-browser visual QA remain a manual follow-up before or shortly after merge; no functional risk, since all interaction logic is covered by the 1094 automated tests.

## FR Coverage Verification (plan's Task 17 self-review claims)

Cross-checked each FR against source, not just the plan's claim:

- **FR-1** (shared component, adapter, deleted duplicated forms, no inline type re-declaration): DONE. `TemplatesEditorAdapter` interface at `CharacterTemplatesEditor.tsx:25-32`; both legacy form files deleted; `grep` for inline `CharacterTemplate` shape re-declarations under the deleted files' former paths returns nothing (the one incidental hit, `EmptySlotTile.tsx`, is a pre-existing, out-of-scope file unrelated to this branch — last touched in commit `3aa70e177`, unrelated PR #738 — and is a derived type alias off `TenantConfigAttributes`, not a duplicate literal shape).
- **FR-2** (segmented selector, no thumbnails, `+New`, `?tpl=` URL sync, empty state): DONE — verified directly in `TemplateSelector.tsx` and `CharacterTemplatesEditor.tsx`.
- **FR-3** (kebab menu, Duplicate/Remove, no Edit-as-JSON): DONE — verified in `TemplateActionsMenu.tsx`.
- **FR-4** (Class/Gender/Map, Advanced numeric fields): DONE — verified in `IdentitySection.tsx` + `MapPicker.tsx`.
- **FR-5** (appearance pools, visual browser, gender filter, already-marked): DONE — verified in `AppearancePoolSection.tsx` + `AppearanceBrowserDialog.tsx` + `AppearanceThumb.tsx`.
- **FR-6** (equipment pools, combobox add, per-pool filters): DONE — verified in `EquipmentPoolSection.tsx` + `poolSearchConfig.ts` + `ItemSearchCombobox.tsx`.
- **FR-7** (starting kit items/skills, no skill browser): DONE — verified in `StartingKitSection.tsx`.
- **FR-8** (sticky preview, no id/value table): DONE — verified in `PreviewCard.tsx`.
- **FR-9** (single form state, free switching, sticky save bar, empty-pool warnings): DONE — verified in `editorState.ts` (`isDirty`, `emptyPoolWarnings`) + `SaveBar.tsx` + `CharacterTemplatesEditor.tsx`.
- **FR-10** (Tailwind/shadcn only, pixelated rendering, a11y semantics): DONE — `[image-rendering:pixelated]` present on all sprite `<img>` tags found (`AppearanceThumb.tsx:62`, `SkillRow` in `StartingKitSection.tsx:31`, etc.); `role="tablist"`/`role="tab"` present; warning-color hardcoding removed in the Task 17 cleanup pass.

## Dropped-Feature Absence Check

`grep -rniE "edit.?as.?json|creator.?preview|skill.?browser|id/value table|selector thumb"` across `src/components/features/characters/templates/` and both page wrappers returned **zero matches**. Combined with the direct file reads above (no `<table>` in `PreviewCard.tsx`, no thumbnail `<img>` in `TemplateSelector.tsx`, no "Edit as JSON" menu item in `TemplateActionsMenu.tsx`, numeric-input-only skill add in `StartingKitSection.tsx`), all five explicitly dropped features are confirmed absent.

## Deviations From Plan (non-cosmetic, all judged acceptable)

1. **Task 2 build-gate fix** (commit `3913ce722`): plan's test code as literally written triggers `noUncheckedIndexedAccess`; implementer added `!` assertions rather than disabling the flag. Correct call — the flag stays on project-wide per atlas-ui CLAUDE.md.
2. **Task 6 debounce regression** (fixed within the task, commits `0d59baf6d`→`0a4230f02`): `setPage(1)` was initially decoupled from the debounced search term, causing an undebounced query on refine-while-paginated. Caught by the in-task review loop and fixed with a regression test before the task was marked complete — this is the review process working as intended, not a shipped defect.
3. **Task 12 gender-reorder fix**: `showAll` toggle originally could show 0 new candidates on page 1 because opposite-gender ids sorted after same-gender ids; fixed to lead with opposite-gender candidates. Legitimate bug fix during implementation, documented inline (`AppearanceBrowserDialog.tsx:79-84`).
4. **Task 15 discard/URL-sync fix** (commit `0d39bd4e0`): initial assembly's `onDiscard` skipped `syncSelection`, so discarding a newly-added-and-selected template could land the URL on tab 0 instead of the nearest remaining tab. Fixed by re-scoping the clamp effect and giving `discardChanges` its own `syncSelection` call (present in current code, `CharacterTemplatesEditor.tsx:140-152`).

None of these deviations reduce plan/PRD coverage; each is a within-task quality improvement caught by the task's own review loop before being marked complete, with before/after evidence in `.superpowers/sdd/progress.md`.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-ui | PASS (per task brief: `tsc -b` 0 errors + vite build; not re-run this session) | PASS | Spot-check run this session: `npm run test -- src/components/features/characters/templates src/pages/__tests__/TenantsCharacterTemplatesPage.test.tsx src/pages/__tests__/TemplatesCharacterTemplatesPage.test.tsx` → **16 test files, 77 tests, all passed** (2.65s). Full-suite claim (1094 tests) from the task brief / progress.md not independently re-run per audit instructions (already verified green this session). |

`tools/lint.sh --check` reported clean for task-177 (only pre-existing environmental noise about a deleted sibling worktree, per task brief — not re-verified independently in this audit pass, taken as given per instructions).

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

1. (Non-blocking, pre-existing disclosed gap) Perform the interactive visual smoke test from Task 17 Step 3 against a live dev server/ingress at least once before or shortly after merge — confirm `AppearanceThumb` crop offsets (`THUMB_OFFSET_X=-74`/`THUMB_OFFSET_Y=-70`) actually center the head region on real renders, and spot-check light/dark theme rendering of the new warning-token color and the appearance browser dialog. No code changes are anticipated to be required; this is a manual-QA confirmation step only.
2. No other action items — all 17 plan tasks are implemented, the two legacy duplicated forms are deleted with no dangling references, all five explicitly-dropped features are confirmed absent, and the branch adheres to every global constraint (atlas-ui-only change, single new dependency, no App.tsx/route changes).
