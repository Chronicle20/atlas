# Plan Audit — task-097-jobs-skills-ux

**Plan Path:** docs/tasks/task-097-jobs-skills-ux/plan.md
**Audit Date:** 2026-06-14
**Branch:** task-097-jobs-skills-ux
**Base Branch:** main (branch base commit `8fe0073f1`)

## Executive Summary

All 8 plan tasks were faithfully implemented; nothing was silently skipped, stubbed, or deferred. The eight UX defects (FR-1…FR-8) are each covered by code and tests. The verification gate is green: `npm run test` passes 798/798 tests across 91 files, and `npm run build` (tsc -b + vite build) is clean with no type errors. The only deviation from the plan's verbatim code is a deliberate, correctness-driven bug fix in the Task 4 tokenizer (an `active`-region flag) that makes the implementation pass the plan's own test case `#cred#then plain` — the test is preserved verbatim, so this is a legitimate DONE-with-note, not a divergence in behavior.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Consolidated job advancement tree (`lib/jobs/job-advancement-tree.ts`) | DONE | `src/lib/jobs/job-advancement-tree.ts` — `JOB_GRAPH` (lines 11-104), `BRANCH_FLOORS` (118-120: `{0:83,800:83,900:83,910:83,1000:83,2000:80,2001:84}`), `JOB_ROOTS`/`childrenOf`/`rootOf`/`floorOf`/`visibleRoots`/`jobTreePath` (122-167). Corrected floors (Cygnus→83, Aran→80) present. Test `src/lib/jobs/__tests__/job-advancement-tree.test.ts` passes. Commit `a987fbd38`. |
| 2 | Re-home `jobTreePath`, remove `lib/utils/job-tree.ts` | DONE | `SkillsSection.tsx:5` imports from `@/lib/jobs/job-advancement-tree`. `src/lib/utils/job-tree.ts` + its test deleted (confirmed absent). Commit `244d34311`. |
| 3 | Beginner skill name fallback (`lib/skills/beginner-skill-names.ts`) | DONE | `src/lib/skills/beginner-skill-names.ts` — 13-entry map 1000-1012 (lines 5-19), `resolveSkillName` blank-name fallback with whitespace-trim + `Skill <id>` default (26-31). Test passes. Commit `372547319`. |
| 4 | Skill description markup parser (`lib/skills/format-skill-description.ts`) | DONE (with note) | `src/lib/skills/format-skill-description.ts` — `FormattedDescription`/`DescSegment` types, master-level capture+suppression (14, 66-74), `#c…#`→`highlight`, `#x…#` strip, bare-`#` reset (31-57). **Note:** implementer added an `active` region flag (lines 22, 38-49) to fix a bug in the plan's verbatim tokenizer — the original would mis-parse `#cred#then` (treating `#t` as a new opener). The plan's test (including the `#cred#then plain` case at test line 35) is unchanged; the fix makes it pass. Test passes. Commit `e0ce6a102`. |
| 5 | `JobsPage` advancement tree + expand affordance + scroll | DONE | `src/pages/JobsPage.tsx` matches plan verbatim — recursive `JobTreeNode`, `Collapsible`+`CollapsibleTrigger` with `aria-label="Toggle {name}"` + rotating chevron + `cursor-pointer` + focus ring (31-36), job-name `Link` to `/jobs/{id}` (37-42), outer `overflow-y-auto` (62), `visibleRoots(major)` over `useMemo` (56-59). Test passes. Commit `e885b9255`. |
| 6 | Remove superseded `jobs-hierarchy` module | DONE | `src/lib/jobs-hierarchy.ts` + `src/lib/__tests__/jobs-hierarchy.test.ts` deleted (confirmed absent). No code references remain (only a descriptive comment in the new module). Commit `ae3d42d03`. |
| 7 | `JobDetailPage` — fallback name, Master Lv label, copyable id, scroll, formatted description | DONE | `src/pages/JobDetailPage.tsx` matches plan verbatim — `resolveSkillName` (94), `Master Lv` label + tooltip (109-120), copyable id via `TooltipTrigger asChild` focusable span + `TooltipContent copyable` (154-168), outer `overflow-y-auto` (148), `formatSkillDescription` rendered via `SkillDescription` (46-61, 95, 123). Test passes. Commit `d317e8779`. |
| 8 | Full verification gate | DONE | `npm run test` 798/798 pass; `npm run build` clean. Stray-reference grep clean (only a comment match). See Build & Test Results. (Lint not separately re-run; baseline is known-broken per project ref and no new code patterns introduced.) Follow-up typecheck fix commit `8309eca34` annotated `rootOf` locals so `tsc -b` passes — build confirmed green. |

**Completion Rate:** 8/8 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. Every task has code + passing test evidence.

## FR-1…FR-8 Coverage (per the plan's self-review table)

| Requirement | Status | Evidence |
|---|---|---|
| FR-1 expand affordance (chevron, pointer, focus, distinct states) | DONE | `JobsPage.tsx:31-36` — `CollapsibleTrigger` with `aria-label`, `cursor-pointer`, `focus-visible:ring-2`, `group-data-[state=open]:rotate-90`. |
| FR-2 advancement-tree layout, single source, nav targets, indentation | DONE | `job-advancement-tree.ts` single source; `JobsPage.tsx` recursive `JobTreeNode` with `paddingLeft: depth*16` (13) and per-node `Link` to `/jobs/{id}`. |
| FR-3 Beginner render + generic blank-name fallback | DONE | `beginner-skill-names.ts` + `JobDetailPage.tsx:94`. Blank-name driven, not id-0 special-cased (matches context.md finding #3). |
| FR-4 Master Lv label + tooltip + shared term | DONE | `JobDetailPage.tsx:109-120` (`Master Lv` + tooltip) and 127 (`Master Level:` detail). |
| FR-5 copyable job id via existing pattern | DONE | `JobDetailPage.tsx:154-168` reuses the `Tooltip`+`copyable` pattern from MonsterHeader. |
| FR-6 page-local scroll, no app-shell change | DONE | `overflow-y-auto` added to `JobsPage.tsx:62` and `JobDetailPage.tsx:148`; `app-shell.tsx` untouched (not in diff). |
| FR-7 description markup parser, unit-tested, unknown directives stripped | DONE | `format-skill-description.ts` + dedicated test; rendered in `JobDetailPage.tsx`. |
| FR-8 corrected floors (Cygnus 83, Aran 80), documented basis | DONE | `BRANCH_FLOORS` (118-120) + provenance comment (106-117); old floored module removed (Task 6). |

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-ui | PASS | PASS | `npm run build` clean (tsc -b + vite built in 1.45s; the >500 kB chunk warning is a pre-existing advisory on `ConversationEditorPanel`/`index`, unrelated to this task). `npm run test`: 91 files / 798 tests passed. Task-specific files (5) → 32 tests passed. Node v22.22.2. |

No Go services touched — frontend-only change, so `go build` / `go test` / `docker buildx bake` / redis-key-guard do not apply (per plan Task 8 and CLAUDE.md).

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None required. Optional, non-blocking:

1. The Task 4 tokenizer deviation (`active` flag) is correct and well-commented but was not reflected back into `plan.md`'s code block. No action needed for merge; note it in the PR description so reviewers don't flag the plan-vs-code text difference.

---

# Frontend Audit (FE-*) — task-097-jobs-skills-ux

- **Audit Scope:** Changed TS/React files under `services/atlas-ui/src` (branch diff `8fe0073f1` → HEAD)
- **Guidelines Source:** frontend-dev-guidelines skill + `services/atlas-ui/CLAUDE.md` (Vite/React-Router SPA, not Next.js — the skill's Next.js-specific rules are read through this lens)
- **Date:** 2026-06-14
- **Build:** PASS
- **Tests:** 798 passed, 0 failed (91 files)
- **Overall:** PASS

## Build & Test Results

- `npm run build` (tsc -b + vite build): exit 0. `JobDetailPage-EG0UgWo7.js` emitted; built in 1.28s. The >500 kB chunk warning is the pre-existing `ConversationEditorPanel` bundle, unrelated to this change.
- `npm test` (vitest run): `Test Files 91 passed (91)`, `Tests 798 passed (798)`.
- `npx eslint` on the six touched source files: zero errors (clean — no new lint debt against the ~48-error baseline).

## File Inventory

- `lib/jobs/job-advancement-tree.ts` — **Other** (pure data/graph module, ported from deleted `lib/utils/job-tree.ts`)
- `lib/skills/beginner-skill-names.ts` — **Other** (display-hint map + pure resolver)
- `lib/skills/format-skill-description.ts` — **Other** (pure tokenizer/parser)
- `pages/JobsPage.tsx` — **Page** (named export per SPA convention)
- `pages/JobDetailPage.tsx` — **Page** (named export per SPA convention)
- `components/features/characters/SkillsSection.tsx` — **Component** (one import re-pointed only)
- Tests: `job-advancement-tree.test.ts`, `beginner-skill-names.test.ts`, `format-skill-description.test.ts`, `JobsPage.test.tsx`, `JobDetailPage.test.tsx`
- Deleted: `lib/utils/job-tree.ts`, `lib/jobs-hierarchy.ts` (+ tests)

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | grep `: any`/`as any`/`null as any` across all 5 source files: zero matches. (Tests use `as unknown as Tenant`, an unknown-cast, not `any`.) |
| FE-02 | No manual class concatenation | PASS | grep `className={"` + concat: zero matches. classNames are plain string literals; conditional styling uses `data-[state=open]` variants (JobsPage.tsx:35), not JS concat. No `cn()` needed since no conditional class composition exists in scope. |
| FE-03 | No direct API client calls in pages | PASS | grep `@/lib/api/client` in both pages: zero matches. Data flows through `useJobSkills`/`useJobSkillDefinitions` hooks (JobDetailPage.tsx:5-6); JobsPage consumes only the pure graph module. |
| FE-04 | No inline Zod schemas | PASS | grep `z.object(`/`z.string(`/etc.: zero matches. No forms in this change. |
| FE-05 | No spinners for content loading | PASS | grep `animate-spin`: zero matches. Loading uses `<Skeleton>` (JobDetailPage.tsx:184-188); JobsPage has no async loading. |
| FE-06 | No hardcoded colors | PASS | grep for raw palette colors: zero matches. Semantic tokens only: `text-primary`, `text-muted-foreground`, `text-destructive`, `bg-muted`, `bg-background`, `focus-visible:ring-ring`, `text-foreground`. |
| FE-07 | No state mutation | PASS | The `.push` matches are on locally-constructed arrays inside pure functions (`format-skill-description.ts:26,55` local `segments` accumulator; `lib/skills/level-table.ts` out of scope) — never on React state. State setters use replacement (`setFailed(true)`, JobDetailPage.tsx:41). `.sort()` calls operate on freshly `.map()`-ed copies (job-advancement-tree.ts:125,132). |
| FE-08 | No default exports for components | PASS | grep `export default`: zero matches. Pages use named exports (`export function JobsPage`/`JobDetailPage`) per `services/atlas-ui/CLAUDE.md` SPA convention — the Next.js page-default-export exception does not apply here. |
| FE-09 | Tenant guard in hooks | N/A (consumed, not changed) | `useJobSkills` (useJobSkills.ts:18 `enabled: !!tenant?.id && jobId >= 0`) and `useJobSkillDefinitions` (useJobSkillDefinitions.ts:29 `enabled: !!tenant?.id && skillId > 0`, plus throw-guard at :26) both gate on tenant. Not modified in scope; verified correct. |
| FE-10 | Tenant ID in query keys | N/A (consumed, not changed) | `jobSkillsKeys.detail(tenant?.id, jobId)` (useJobSkills.ts:7-8) and `skillDefinitionKeys.detail(tenant?.id, skillId)` both include the tenant id. Not modified in scope. |
| FE-11 | Error handling | PASS | No `.catch`/raw async in scope. Errors surface declaratively via React Query flags: `skillsQuery.isError` → destructive message (JobDetailPage.tsx:189-190); `defsError` → "Skill details unavailable." (:193-194); per-icon load failure → `onError` fallback glyph (:41). Appropriate for query-driven pages. |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | Consumes `SkillDefinitionWithIcon` (flattened view-model from pre-existing `useSkillDefinition`) and `Tenant.attributes.majorVersion` (JobsPage.tsx:57) — tenant follows `{id, attributes}`. No new JSON:API resource types defined; `JobEntry` (job-advancement-tree.ts:1-5) is a UI graph node, not a resource, so the shape rule does not apply. |
| FE-13 | Service extends BaseService | N/A | No service files changed. `jobsService.getSkillsByJobId` consumed via existing hook. |
| FE-14 | Query key factory uses `as const` | N/A (consumed) | `jobSkillsKeys` uses `as const` (useJobSkills.ts:6,8). Not modified. |
| FE-15 | Forms use react-hook-form + zodResolver | N/A | No forms in this change. |
| FE-16 | Schema in lib/schemas with inferred type | N/A | No Zod schemas in this change. |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS | `JobsPage.test.tsx` (5 cases) and `JobDetailPage.test.tsx` (8 cases) cover empty/loading/error/success paths plus accessibility affordances (toggle label, copyable focusable id). Pure modules each have dedicated unit tests (job-advancement-tree: 7 incl. orphan-parent invariant; beginner-skill-names: 6; format-skill-description: 7 incl. bare-`#` reset edge). Queries are by role/text/label, not implementation. |
| FE-18 | Mocks updated when services changed | N/A | No services changed. Page tests mock the hooks and `useTenant` via `vi.mock` (JobDetailPage.test.tsx:11-18), consistent with the project's Vitest pattern. |

## Accessibility Spot-Checks (PRD-specific)

- **Collapsible toggle** (JobsPage.tsx:31-36): `CollapsibleTrigger` has `aria-label={`Toggle ${name}`}`, `cursor-pointer`, visible focus ring (`focus-visible:ring-2 focus-visible:ring-ring`), chevron rotation via `group-data-[state=open]:rotate-90`; keyboard-operable Radix trigger. Verified by JobsPage.test.tsx:51. PASS.
- **Copyable job-id tooltip** (JobDetailPage.tsx:154-168): trigger `<span>` has `tabIndex={0}`, `cursor-help`, focus ring; `<TooltipContent copyable>` uses a real supported prop (`components/ui/tooltip.tsx:41,43,116-117`). Verified by JobDetailPage.test.tsx:83-88. PASS.
- **Master-Lv tooltip** (JobDetailPage.tsx:106-120): same focusable `tabIndex={0}` + ring pattern. PASS.
- **Skill row expander** (JobDetailPage.tsx:100-104): `CollapsibleTrigger asChild` wraps a real `<button>`, reachable by role. PASS.

## Notes / Non-Blocking Observations

- `@/lib/jobs` (JobDetailPage.tsx:8) resolves to the pre-existing `lib/jobs.ts` (exports `getJobNameById` at line 87), **not** the new `lib/jobs/` directory — a deliberate filename-vs-directory coexistence. Compiles and is correct, but the two `jobs` siblings (`lib/jobs.ts` + `lib/jobs/`) are an easy future-reader trap. Cosmetic only; no rule violated.
- Provenance comments (job-advancement-tree.ts:8-10, 106-117; beginner-skill-names.ts:1-4) correctly cite `libs/atlas-constants` and the live-probe finding that `/jobs/{id}/skills` is not version-gated. Good traceability.

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- Consider consolidating `lib/jobs.ts` and `lib/jobs/` to avoid the dual-`jobs` ambiguity (optional, cosmetic).
