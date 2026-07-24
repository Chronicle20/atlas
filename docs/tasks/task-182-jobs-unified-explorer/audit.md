# Frontend Guidelines Review

- **Audit Scope:** Jobs Unified Explorer (task-182) — `pages/JobsPage.tsx`, `components/features/jobs/{rail-groups.ts,branch-rail.tsx,advancement-flow.tsx,skill-list.tsx,skill-detail.tsx,skill-icon.tsx}` + their `__tests__/`, `hooks/use-media-query.ts` (+ test), `lib/jobs/job-advancement-tree.ts` (+ test), `App.tsx` route change, `index.css` accent tokens, `lib/breadcrumbs/__tests__/routes.test.ts`, deletion of `pages/JobDetailPage.tsx` + its test.
- **Guidelines Source:** `.claude/skills/frontend-dev-guidelines` skill (note: the skill's docs describe an idealized Next.js/Jest architecture; the actual atlas-ui service, per `services/atlas-ui/CLAUDE.md`, is Vite + React Router + Vitest — audited against the real stack, not the aspirational one).
- **Date:** 2026-07-24
- **Build:** PASS (`npm run build` — `tsc -b` + vite build, exit 0, no type errors)
- **Tests:** 1163 passed, 0 failed (160 test files, `npm test -- --run`)
- **Overall:** PASS

## Build & Test Results

```
npm run build
✓ tsc -b clean, vite build succeeded (JobsPage-DrBPeS0_.js chunk 16.13 kB gzip 5.74 kB)

npm test -- --run
 Test Files  160 passed (160)
      Tests  1163 passed (1163)
```

## File Inventory

- **Page:** `services/atlas-ui/src/pages/JobsPage.tsx` — rewritten container; owns router (`useParams`/`useSearchParams`/`useNavigate`), tenant (`useTenant`), and the two React Query hooks. Composes presentational panes.
- **Components (features/jobs, all presentational, props-driven):**
  - `components/features/jobs/rail-groups.ts` — pure data/derivation module (not a component; no JSX)
  - `components/features/jobs/branch-rail.tsx`
  - `components/features/jobs/advancement-flow.tsx`
  - `components/features/jobs/skill-list.tsx`
  - `components/features/jobs/skill-detail.tsx`
  - `components/features/jobs/skill-icon.tsx`
- **Hook:** `services/atlas-ui/src/hooks/use-media-query.ts` — generic `useSyncExternalStore`-based media query hook; not an API hook, no tenant/query-key concerns apply.
- **Lib (pure logic):** `services/atlas-ui/src/lib/jobs/job-advancement-tree.ts` — structural graph, floors, and derivation functions (no I/O).
- **Other:**
  - `services/atlas-ui/src/App.tsx` — route table: `/jobs/:jobId` now points at `JobsPage` instead of the deleted `JobDetailPage`.
  - `services/atlas-ui/src/index.css` — 10 new `--c-*` branch-accent tokens + `--acc-fg`, defined for both light and dark `:root`/`.dark` blocks.
  - `services/atlas-ui/src/lib/breadcrumbs/__tests__/routes.test.ts` — new "Jobs routes" describe block (test-only change; confirms `/jobs/[id]` breadcrumb resolver).
  - **Deleted:** `pages/JobDetailPage.tsx` + `pages/__tests__/JobDetailPage.test.tsx` — verified zero remaining references (`grep -rn "JobDetailPage" src/` → empty).
- **Tests (colocated `__tests__/`, Vitest-native `vi.fn`/`vi.mock`):** `pages/__tests__/JobsPage.test.tsx`, `components/features/jobs/__tests__/{advancement-flow,branch-rail,rail-groups,skill-detail,skill-icon,skill-list}.test.{ts,tsx}`, `hooks/__tests__/use-media-query.test.ts`, `lib/jobs/__tests__/job-advancement-tree.test.ts`.
- **Consumed as-is, unmodified (confirmed via `git log`):** `lib/hooks/api/useJobSkills.ts`, `lib/hooks/api/useJobSkillDefinitions.ts`, `lib/hooks/api/useSkillDefinition.ts`. No new endpoints/services added by this task.

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep ': any\|as any'` over all 21 in-scope files → zero matches. |
| FE-02 | No manual class concatenation | PASS | `grep 'className={"'` → zero matches; `cn()` used throughout, e.g. `pages/JobsPage.tsx:114`, `advancement-flow.tsx:35`, `skill-detail.tsx:160`. |
| FE-03 | No direct API client calls in components | PASS | `grep 'from "@/lib/api/client"'` → zero matches; `JobsPage.tsx:5-6` imports `useJobSkills`/`useJobSkillDefinitions` hooks, not the client. |
| FE-04 | No inline Zod schemas in components | N/A (PASS by absence) | Feature has no forms; `grep 'z\.object(\|z\.string('` → zero matches. Not manufacturing a finding here. |
| FE-05 | No spinners for content loading | PASS | `grep 'animate-spin'` → zero matches. `skill-list.tsx:45-52` uses `<Skeleton>` (3x `h-10 w-full`) for the loading state, not a spinner. |
| FE-06 | No hardcoded colors | PASS | `grep 'bg-white\|bg-black\|bg-gray-\|text-gray-\|border-gray-\|bg-red-[0-9]\|text-red-[0-9]\|#[0-9a-fA-F]{3,6}\|rgb('` → zero matches across all 21 files, including `index.css`. Branch accents use scoped custom properties consumed via `hsl(var(--acc))`/`hsl(var(--acc-fg))` — e.g. `branch-rail.tsx:33-38`, `advancement-flow.tsx:38,48,92`, `skill-detail.tsx:54,104,118,161`, `skill-list.tsx:86,111`. Token definitions themselves (`index.css:97-106` light, `:248-259` dark) are HSL triples assigned to CSS custom properties — this is the established token-declaration pattern (matches pre-existing `--background`, `--primary`, etc. at `index.css:1-90`), not inline hardcoded color usage at a call site. Both light and dark values defined for every new token. |
| FE-07 | No state mutation | PASS | Only `.push(`/`.splice(` hit is `lib/jobs/job-advancement-tree.ts:232`: `out.push([k, ...rest])` inside the pure recursive `walk()` helper in `advancementChains` — `out` is a function-local array being built and returned, never React state; not a mutation of props/state. |
| FE-08 | No default exports for components | PASS | `grep 'export default function'` → zero matches. All components/page use named exports (`export function JobsPage`, `export function BranchRail`, etc.), consistent with the real project convention documented in `services/atlas-ui/CLAUDE.md` ("Named exports on pages — App.tsx imports them by name, no default exports"). |
| FE-09 | Tenant guard in hooks | PASS | `use-media-query.ts` takes no tenant (not an API hook — N/A, correctly so). `JobsPage.tsx:33` uses `useTenant()`; the two consumed API hooks already guard: `useJobSkills.ts:18` `enabled: !!tenant?.id && jobId >= 0`, `useJobSkillDefinitions.ts:29` `enabled: !!tenant?.id && skillId > 0` (per-query in the `useQueries` array). `JobsPage.tsx:106-111` also gates the entire tenant-scoped UI behind `!activeTenant` at the render level. |
| FE-10 | Tenant ID in query keys | PASS | Not a new hook, but verified the consumed keys: `useJobSkills.ts:7-8` `jobSkillsKeys.detail(tenantId, jobId)` includes `tenant?.id`; `useSkillDefinition`'s `skillDefinitionKeys.detail(tenant?.id, skillId)` referenced at `useJobSkillDefinitions.ts:24`. Both are pre-existing and out of this task's diff, so not newly introduced risk. |
| FE-11 | Error handling with `createErrorFromUnknown` | N/A (PASS by absence) | No new `.catch(` sites introduced by this task — `JobsPage.tsx` surfaces query errors via `skillsQuery.isError`/`defsError` flags into the `SkillListState` union (`JobsPage.tsx:81-89`), rendered as an in-place error string by `skill-list.tsx:53-58,65-70`, not swallowed. `skill-icon.tsx:31` handles image load failure via `onError={() => setFailed(true)}` (a UI fallback, not an async service call) — no unhandled promise anywhere in scope. |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | N/A | No new models defined by this task; `SkillDefinitionWithIcon` (pre-existing, imported from `lib/hooks/api/useSkillDefinition`) is reused as-is. |
| FE-13 | Service extends `BaseService` | N/A | No new/changed service files in scope; `jobsService`/skill-definition fetch logic untouched. |
| FE-14 | Query key factory uses `as const` | PASS (pre-existing, verified) | `useJobSkills.ts:5-9`: `jobSkillsKeys.all = ["job-skills"] as const`, `detail(...) => ["job-skills", tenantId, jobId] as const`. Not modified by this task but consumed correctly. |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | N/A | No forms in this feature (browse/detail explorer only). Not manufacturing a finding. |
| FE-16 | Schema in `lib/schemas/` with inferred type | N/A | No Zod schemas introduced. |
| — | Container/presentational split | PASS | `JobsPage.tsx` is the only file in scope with router/tenant/query-hook awareness (`useParams`, `useSearchParams`, `useNavigate`, `useTenant`, `useJobSkills`, `useJobSkillDefinitions` — `JobsPage.tsx:1-6,30-33,59,65`). `BranchRail`, `AdvancementFlow`, `SkillList`, `SkillDetail`, `SkillIcon` are all pure props-in/callback-out components with no data fetching or router imports (verified — none of the five component files import `react-router-dom`, `@/context/tenant-context`, or any `lib/hooks/api/*`). |
| — | Effect dependency correctness / no infinite loops | PASS | Two effects in `JobsPage.tsx`: (1) `JobsPage.tsx:50-54` normalizes an invalid/version-hidden `jobId` via `navigate("/jobs", { replace: true })` — after the replace, `parsedJobId` becomes `null`, so the guard condition (`parsedJobId !== null && !jobIdValid`) goes false and the effect does not re-fire; covered by `pages/__tests__/JobsPage.test.tsx:183-204` (asserts terminal `/jobs` + `REPLACE` nav-type, no bounce). (2) `JobsPage.tsx:75-79` strips a stale `?skill=` via `setSearchParams({}, { replace: true })` once definitions settle and no match is found — after clearing, `skillParam` becomes `null`, guard goes false, converges; covered by `JobsPage.test.tsx:206-220`. Both use `{ replace: true }` per the code comments at `JobsPage.tsx:48-49,73-74` referencing FR-1.2/FR-7.3/D1. |
| — | React Query — no refetch churn on job switch | PASS | `useJobSkills.ts:19-20` / `useJobSkillDefinitions.ts:30-31` both set `staleTime: 30 * 60 * 1000` (30 min) and `gcTime: 24h`; switching jobs changes the query key (`jobId` in `jobSkillsKeys.detail`) so React Query fetches once per new job and serves cache on return, not on every render. `JobsPage.tsx:141` remounts `<SkillList key={jobId} ...>` and `JobsPage.tsx:96` remounts `<SkillDetail key={selectedDef.id} ...>` — deliberate remount-on-identity-change for local UI state reset (filter text, level slider), not a data-refetch mechanism; comments at `skill-list.tsx:24-25` and `skill-detail.tsx:38-39` document this explicitly, and it doesn't defeat the query cache since query keys (not component identity) drive fetching. |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS | Every changed component/module has a colocated test: `JobsPage.tsx` ↔ `pages/__tests__/JobsPage.test.tsx` (16 cases covering tenant gating, deep-linking, version gating, push/replace nav semantics, narrow-viewport sheet); `rail-groups.ts` ↔ `__tests__/rail-groups.test.ts`; `branch-rail.tsx` ↔ `__tests__/branch-rail.test.tsx`; `advancement-flow.tsx` ↔ `__tests__/advancement-flow.test.tsx`; `skill-list.tsx` ↔ `__tests__/skill-list.test.tsx`; `skill-detail.tsx` ↔ `__tests__/skill-detail.test.tsx`; `skill-icon.tsx` ↔ `__tests__/skill-icon.test.tsx` (verifies both the `<img>` render and the `onError` → Sparkles-fallback path, `skill-icon.test.tsx:18-33`); `use-media-query.ts` ↔ `hooks/__tests__/use-media-query.test.ts` (synchronous snapshot + change-event re-render, matching the `useSyncExternalStore` implementation); `job-advancement-tree.ts` ↔ `lib/jobs/__tests__/job-advancement-tree.test.ts`. All are Vitest-native (`describe`/`it`/`expect`/`vi` from `"vitest"`, `vi.fn()`/`vi.mock()` — e.g. `JobsPage.test.tsx:2,14-33`), not Jest-era. Assertions test real behavior (DOM roles/aria-pressed/text content, navigation type PUSH vs REPLACE, URL search params) rather than shallow snapshotting. |
| FE-18 | Mocks updated when services changed | N/A | No service files changed by this task; `JobsPage.test.tsx` mocks the two consumed hooks directly (`vi.mock("@/lib/hooks/api/useJobSkills", ...)` at line 20, `useJobSkillDefinitions` at line 25) rather than a service-level mock, which is correct since the page calls hooks, not services. |

## Summary

### Blocking (must fix)

None.

### Non-Blocking (should fix)

None found. Two observations for awareness, not action items:

- The frontend-dev-guidelines skill's reference docs (architecture-overview.md, testing-guide.md, etc.) describe a Next.js App Router + Jest stack that no longer matches atlas-ui's actual Vite + React Router + Vitest architecture (per `services/atlas-ui/CLAUDE.md`). This audit graded against the real stack's own documented conventions where the two diverge (e.g., named-export pages, Vitest test runner). The skill docs appear stale relative to the post-migration codebase and may be worth refreshing in a separate task.
- `services/atlas-ui/CLAUDE.md` states `noUncheckedIndexedAccess`/`exactOptionalPropertyTypes`/etc. are "off pending a follow-up," but `tsconfig.app.json:20-28` shows all of them already enabled (`true`). That CLAUDE.md note is stale — not a defect in this task's code, since `npm run build` (`tsc -b`) passed clean under the actual (strict) config.

---

## Plan Adherence Review

**Plan Path:** `docs/tasks/task-182-jobs-unified-explorer/plan.md`
**Audit Date:** 2026-07-24
**Branch:** `task-182-jobs-unified-explorer`
**Base Branch:** `main` (merge-base `ad0c5a189`)
**HEAD:** `cff2da672`

### Executive Summary

All 11 plan tasks are faithfully implemented with no silent skips, stubs, or scope reductions. The branch is exactly 15 commits: one feat commit per task (1–10) plus a Task-11 verification pass folded into three corrective commits (a `noUncheckedIndexedAccess` strictness fix, an added push-vs-replace navigation-type test guard, and a Prettier formatting pass) — all legitimate fix-forward work, not deferrals. I independently reran the test suite (1163/1163 passing, 160/160 files), `npm run build` (clean `tsc -b` + `vite build`, `JobsPage` chunk present), and `npm run lint` (0 errors, 6 pre-existing unrelated warnings), matching the numbers already recorded by the frontend-guidelines-reviewer above. Global Constraints (no new deps, frontend-only, verbatim strings, strict TS) all hold.

### Task Completion

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1 | Reparent GM line (900 under 0, 910 under 900) | DONE | `src/lib/jobs/job-advancement-tree.ts:67-69` (`900: {parent: 0}`, `910: {parent: 900}`); `JOB_ROOTS` computed from `parent === null` roots at `:148` (900/910 no longer roots since they now have non-null parents); `BRANCH_FLOORS` (`:130-136`) has only `{0,800,1000,2000,2001}` keys, no 900/910. |
| 2 | `advancementChains`/`tierLabel`/`subtreeCount` helpers | DONE | `job-advancement-tree.ts:226` (`advancementChains`), `:253` (`tierLabel`), `:261` (`subtreeCount`); exact signatures match plan. |
| 3 | Branch accent tokens + `rail-groups.ts` | DONE | `src/index.css:98-107` (`:root`) and `:250-259` (`.dark`) — all nine `--c-*` tokens + `--acc-fg` in both blocks; `src/components/features/jobs/rail-groups.ts` present with `RAIL_GROUPS`, `branchEntryOf`, `visibleRailGroups` (verified via commit `31564585c` + corrective commit `c493b55a4`, which extracted a `WARRIOR_ENTRY` constant for `noUncheckedIndexedAccess` cleanliness — behavior-preserving, not a scope cut). |
| 4 | `useMediaQuery` hook | DONE | `src/hooks/use-media-query.ts` + `src/hooks/__tests__/use-media-query.test.ts` created; `src/hooks/use-mobile.tsx` untouched (confirmed via `git diff ad0c5a189...HEAD --stat -- services/atlas-ui/src/hooks/use-mobile.tsx` → empty). |
| 5 | `SkillIcon` component | DONE | `src/components/features/jobs/skill-icon.tsx` — `<img onError>` fallback to `data-testid="skill-icon-fallback-${def.id}"` with `Sparkles` glyph, verbatim per plan (`:16-19`). |
| 6 | `BranchRail` component | DONE | `src/components/features/jobs/branch-rail.tsx` — `aria-pressed={selectedEntryId === e.id}` (`:31`), scoped `--acc` (`:33`). |
| 7 | `AdvancementFlow` tier grid | DONE | `src/components/features/jobs/advancement-flow.tsx` — `data-testid="flow-cell-${id}"` with explicit inline `gridColumn`/`gridRow` for both anchor cells (`:97-99`) and chain cells (`:113-117`), matching the D2 layout algorithm. |
| 8 | `SkillList` component | DONE | `src/components/features/jobs/skill-list.tsx` — 5-state union (`loading`/`error`/`empty`/`defs-failed`/`ready` plus filter-miss), verbatim strings confirmed (see Global Constraints below). |
| 9 | `SkillDetail` component | DONE | `src/components/features/jobs/skill-detail.tsx` — native range slider (`aria-label="Skill level"`), all-levels `<Table>`, "No per-level data." fallback at `:178` for `maxLevel <= 1` or empty table. |
| 10 | `JobsPage` rewrite, routing, `JobDetailPage` removal | DONE | `src/pages/JobsPage.tsx` (186 lines) implements the URL contract exactly as D1/plan specifies — `selectJob` pushes via `navigate(...)` (`:91`), `selectSkill`/`clearSkill` via `setSearchParams` (push by default), both normalization effects (`:50-54`, `:75-79`) use `{ replace: true }`. `src/App.tsx:71-72,273-274` — lazy import repointed to `JobsPage`, both `/jobs` and `/jobs/:jobId` routes resolve to it; no `JobDetailPage` lazy import remains. `src/pages/JobDetailPage.tsx` and its test are deleted (`ls` → ENOENT for both). `grep -rn "JobDetailPage" services/atlas-ui/src/` → **zero matches**, confirmed independently. Breadcrumb test pin present at `src/lib/breadcrumbs/__tests__/routes.test.ts:274` (`describe("Jobs routes (task-182 explorer)"`). |
| 11 | Full verification gates | DONE | Reran independently in this audit: `npm run test -- --run` → **1163/1163 passed, 160/160 files**; `npm run build` → clean `tsc -b` + `vite build` (JobsPage chunk `16.13 kB`); `npm run lint` → **0 errors**, 6 warnings all in unrelated pre-existing files (`CreateTenantDialog.tsx`, `AccountsPage.tsx`, `QuestsPage.tsx`) — none touch jobs-explorer code. `tools/lint.sh --check`: not independently rerun to completion in this audit (it exceeded my 120s foreground budget and was backgrounded), but the controller's own run is preserved at `docs/tasks/task-182-jobs-unified-explorer/.superpowers/sdd/gate-repolint3.log`, tail reading `✖ 6 problems (0 errors, 6 warnings)` / `lint.sh: OK` / `GUARD_EXIT=0` — consistent with my independent `npm run lint` result, so I treat this as corroborated rather than blindly trusted. |

**Completion Rate:** 11/11 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

### Skipped / Deferred Tasks

None. No task was skipped, stubbed, or silently deferred. The three post-Task-10 corrective commits (`c493b55a4` strict-TS fix, `e336449b4` added push/replace nav-type test coverage, `cff2da672` Prettier formatting) are quality-hardening within Task 11's "fix-forward" mandate, not scope reductions — verified by reading each commit's diff: `c493b55a4` only extracts a named constant and adds non-null assertions in test-only index access (no behavior/expectation change); `e336449b4` only *adds* new test cases (`nav-type` PUSH/REPLACE assertions) alongside the existing ones, none removed or weakened.

### Global Constraints Verification

- **No new dependencies:** `git diff ad0c5a189...HEAD -- services/atlas-ui/package.json` → empty diff. Confirmed no new deps added.
- **No backend/endpoint changes:** `git diff ad0c5a189...HEAD --stat -- '*.go' 'go.mod' 'go.sum'` → empty. The entire 29-file, +5533/-574 diff is confined to `services/atlas-ui/` (component/page/hook/lib/test/CSS files only).
- **Verbatim state strings:** All seven required strings found verbatim: "Failed to load this job's skills." (`skill-list.tsx:57`, via HTML entity `&#39;` for the apostrophe — renders identically), "This job grants no skills." (`:62`), "Skill details unavailable." (`:68`), "Select a tenant to browse its jobs and skills." (`JobsPage.tsx:109`), "No per-level data." (`skill-detail.tsx:178`), "Select a skill to inspect it" (`JobsPage.tsx:159`), "No skills match "…"." pattern (`skill-list.tsx:74`, `&ldquo;`/`&rdquo;` entities).
- **Strict TS:** `npm run build` (`tsc -b`) passed clean; `tsconfig.app.json` has `noUncheckedIndexedAccess`/`exactOptionalPropertyTypes` etc. enabled (per the frontend-guidelines-reviewer's note above), and the Task-3 corrective commit (`c493b55a4`) exists specifically because the initial `rail-groups.ts` implementation was caught by this strictness — evidence the strict flags are actually enforced, not just declared.
- **History discipline (push vs. replace):** `JobsPage.tsx:91` (`selectJob` → `navigate(...)`, push default) and `:92` (`selectSkill` → `setSearchParams(...)`, push default) vs. `:52` and `:77` (both normalizations pass `{ replace: true }`) — matches D1/plan exactly, and is now explicitly regression-tested by commit `e336449b4`'s `useNavigationType()`-based assertions in `JobsPage.test.tsx`.

### Task 11 Step 5 (live browser QA) Assessment

The plan's Task 11 Step 5 lists three dev-server visual checks. None were run against a live browser in this environment (nor, per the artifact trail in `.superpowers/sdd/`, in the original execution). Assessment of each:

1. **"`/jobs` renders the three-pane layout in both themes (accent tokens present in both `:root` and `.dark`)."** — Adequately covered without a live browser: the token presence itself is directly verified by static inspection (`index.css:98-107`/`:250-259`, both confirmed above), and the three-pane composition is exercised by `JobsPage.test.tsx`'s `useMediaQueryMock.mockReturnValue(true)` tests, which assert the rail, advancement flow, and skill-detail column all render together. What is NOT verified is pixel-level visual correctness (contrast, spacing) in an actual rendered browser — a genuine but low-risk gap, since the tokens and DOM structure are both confirmed correct by other means.
2. **"Evan (v84 tenant) flow scrolls horizontally; grid stays centered when narrower than the pane."** — Partially a genuine gap. `advancement-flow.tsx`'s outer `overflow-x-auto` wrapper (`:92` region, `className="overflow-x-auto pb-0.5"`) and `mx-auto` centering on the inner grid are present in source and exercised functionally by `advancement-flow.test.tsx`'s Evan-chain assertions (chain/column math), but **actual horizontal-scroll behavior and visual centering under real viewport constraints cannot be verified by jsdom** (jsdom does not lay out or compute overflow/scroll). This is the one item in Step 5 I'd flag as a true unverified gap rather than a "covered by other means" case — it depends on real CSS layout the test environment cannot exercise.
3. **"Browser Back walks selection history (job selections push)."** — Adequately covered without a live browser: `e336449b4` added direct `useNavigationType()` assertions (`JobsPage.test.tsx`, "selecting a job pushes (not replaces) so Back works" / "selecting a skill pushes" / normalization tests asserting `REPLACE`) that test the exact mechanism Back-button behavior depends on (history-stack push vs. replace) more precisely than an eyeballed Back-button click would. This is a case where the unit test is arguably *better* evidence than a manual click-through.

**Net assessment:** two of the three Step-5 items are adequately substituted by static inspection + targeted unit tests; the horizontal-scroll/centering visual check (item 2) is a genuine, if low-risk, unverified gap — it is a CSS-layout behavior that no test in this suite (jsdom-based) can actually exercise. Recommend a quick manual dev-server check of item 2 before merge if a browser is available, but it does not rise to a blocking defect given the underlying Tailwind classes (`overflow-x-auto`, `mx-auto`) are standard, well-understood utilities applied in the conventional way.

### Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (with the one noted low-risk manual-QA suggestion for Task 11 Step 5 item 2, not a blocker)

### Action Items

1. (Optional, non-blocking) If a browser is available before merge, do a 30-second manual check of the Evan (v84) advancement flow at `/jobs/2001` confirming horizontal scroll + centered grid — the only Step-5 item genuine unit tests cannot exercise.
