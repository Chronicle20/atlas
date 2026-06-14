# task-097 Jobs & Skills Browser UX Overhaul — Context

Companion to `plan.md`. Frontend-only (`services/atlas-ui`). No Go / atlas-data /
API changes. Read this before executing; it captures the verified findings so the
executor does not re-litigate them.

## Worktree

- Worktree: `.worktrees/task-097-jobs-skills-ux`, branch `task-097-jobs-skills-ux`.
- All paths below are relative to `services/atlas-ui/` unless noted.

## Toolchain (atlas-ui)

- `npm` is a broken Windows install under WSL — **source nvm 22 first**:
  `source ~/.nvm/nvm.sh && nvm use 22` before any `npm`/`npx`.
- Test runner: Vitest. Single file: `npx vitest run <path>`. Full suite: `npm run test`.
- Build + typecheck: `npm run build` (tsc -b + vite build). It type-checks the new
  Vitest test files, so test code must be type-correct (ref:
  `reference_atlas_ui_build_typechecks_tests`).
- Lint baseline is **pre-existing-broken** (~48 errors). Gate on *no new* lint
  errors, not a clean lint (ref: `reference_atlas_ui_npm_nvm_and_lint_baseline`).
- `noUnusedLocals`/`noUncheckedIndexedAccess` are **off** — build won't flag dead
  imports or unchecked index access; clean them up by hand.

## Verified findings (do not re-derive)

1. **Beginner skills exist and have blank names.** Live probe (atlas-data,
   atlas-main, GMS v83 tenant `ec876921-…`, 2026-06-14):
   `GET /api/data/jobs/0/skills` → `{skills:[1000…1012]}` (13 skills);
   `GET /api/data/skills/1000` → `name:"", description:""`. The id→name map
   (1000–1012) is sourced from `libs/atlas-constants/skill/constants.go`
   (`Beginner*Id` block, lines ~2903–2915) — verified, not from memory.
2. **`/jobs/{id}/skills` is NOT version-gated** (ref:
   `reference_atlas_data_jobs_skills_not_version_gated`). Version floors are a
   UI display-curation choice, not a data gate.
3. **No id-0 short-circuit.** `useJobSkills.ts:18` enables on `jobId >= 0`;
   `useJobSkillDefinitions.ts` enables each skill on `skillId > 0` (Beginner ids
   1000–1012 all qualify). FR-3.1 is satisfied by deleting the stale "job 0
   returns 0" comment — there is no runtime fix.
4. **Scroll root cause** = `src/app-shell.tsx:27-28` wraps `<Outlet/>` in
   `overflow-hidden`. The working pattern is page-local `overflow-y-auto`
   (`MapDetailPage.tsx:44`). Both Jobs pages already have the identical outer
   class string **minus** `overflow-y-auto`. FR-6 adds that one token to each —
   **do not touch `app-shell.tsx`**.
5. **Corrected version floors** (FR-8): `CYGNUS 92→83` (ref:
   `reference_maplestory_version_timeline` — KoC exist in v83), `ARAN 88→80`
   (product owner, PRD FR-8.1). `EVAN` stays 84. Adventurer/Special/Admin stay 83.
6. **Copyable id pattern** = `Tooltip` + `TooltipTrigger asChild` (focusable
   element) + `TooltipContent copyable` (carries the id). See
   `components/features/monsters/MonsterHeader.tsx`. The `copyable` prop on
   `TooltipContent` (`components/ui/tooltip.tsx:41`) renders the copy button.
   `CopyableIdHeader` is **not** reused — its layout makes the title the copy
   target and has no back-button slot.

## Source-of-truth consolidation (FR-2/FR-8)

Two structures encode the hierarchy and **drift is the defect**:
- `lib/utils/job-tree.ts` `JOB_TREE` — has parent edges + names, no floors.
  `jobTreePath` is consumed by `components/features/characters/SkillsSection.tsx:5`
  (and `lib/utils/__tests__/job-tree.test.ts`) — so it must be re-homed, not deleted.
- `lib/jobs-hierarchy.ts` `JOB_HIERARCHY` — has floors, no edges. Only
  `JobsPage.tsx` + `lib/__tests__/jobs-hierarchy.test.ts` consume it.

**Decision:** new `lib/jobs/job-advancement-tree.ts` becomes the single source:
`JOB_GRAPH` (parent edges + names, ported verbatim from `JOB_TREE`) +
`BRANCH_FLOORS` (per-root floors) + helpers (`rootOf`, `floorOf`, `visibleRoots`,
`childrenOf`, `JOB_ROOTS`, `jobTreePath`). `jobs-hierarchy.ts` and
`lib/utils/job-tree.ts` are both **removed**; their tests fold into the new
module's test.

## Files

New:
- `src/lib/jobs/job-advancement-tree.ts` (+ `__tests__/job-advancement-tree.test.ts`)
- `src/lib/skills/beginner-skill-names.ts` (+ `__tests__/beginner-skill-names.test.ts`)
- `src/lib/skills/format-skill-description.ts` (+ `__tests__/format-skill-description.test.ts`)

Modified:
- `src/pages/JobsPage.tsx` — recursive indented tree, chevron affordance,
  `overflow-y-auto` (+ rewrite `src/pages/__tests__/JobsPage.test.tsx`).
- `src/pages/JobDetailPage.tsx` — name fallback, `Master Lv` label + tooltip,
  copyable id, `overflow-y-auto`, formatted description (+ update
  `src/pages/__tests__/JobDetailPage.test.tsx`).
- `src/components/features/characters/SkillsSection.tsx` — re-point `jobTreePath`
  import to the new module.

Removed:
- `src/lib/utils/job-tree.ts` + `src/lib/utils/__tests__/job-tree.test.ts`
- `src/lib/jobs-hierarchy.ts` + `src/lib/__tests__/jobs-hierarchy.test.ts`

## Key types (existing, unchanged)

- `SkillDefinition` (`src/services/api/skills.service.ts:50`): `id:number`,
  `name:string`, `description:string` (`""` when not upgraded), `action:boolean`,
  `element:string`, `maxLevel?:number`.
- `SkillDefinitionWithIcon` extends it with `iconUrl:string`.
- `deriveSkillType(def)` and `buildLevelTable(def.effects)` stay as-is.

## Open questions — resolved in design

- OQ-1 keep version gating (corrected). OQ-2 strip `#c…#` to text for v1 (parser
  keeps a `color` marker for later). OQ-3 Beginner names verified against
  constants.go. OQ-4 suppress the `[Master Level : N]` header (captured, not shown).

## Verification gate

`source ~/.nvm/nvm.sh && nvm use 22` then, from `services/atlas-ui/`:
1. `npm run test` — full suite green.
2. `npm run build` — clean (tsc -b + vite build).
3. `npm run lint` — no *new* errors vs the pre-existing baseline.

No Go build / `docker buildx bake` / `go.work` / k8s steps — frontend-only.
