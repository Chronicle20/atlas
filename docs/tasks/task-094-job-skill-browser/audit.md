# Frontend Audit — task-094 Jobs & Skills Browser

- **Audit Scope:** Diff `e7ffed475..7ae39a1c1` over `services/atlas-ui` (frontend-only)
- **Guidelines Source:** frontend-dev-guidelines skill (FE-* checklist)
- **Date:** 2026-06-13
- **Build:** PASS
- **Tests:** 783 passed, 0 failed (90 files)
- **Overall:** PASS

## Build & Test Results

- `npm run build` — exit 0, `built in 1.70s`. Only the pre-existing chunk-size advisory (ConversationEditorPanel/index) — unrelated to task-094.
- `npm test` (`vitest run`) — `Test Files 90 passed (90)`, `Tests 783 passed (783)`.

## File Inventory

- `src/services/api/skills.service.ts` — Service (direct-client adapter; broadened `SkillEffect`, optional `maxLevel`)
- `src/lib/hooks/api/useSkillDefinition.ts` — Hook (query + shared fetcher/retry)
- `src/lib/hooks/api/useJobSkillDefinitions.ts` — Hook (`useQueries` fan-out)
- `src/lib/hooks/api/useJobSkills.ts` — Hook (pre-existing, referenced; unchanged in this diff range)
- `src/lib/skills/skill-type.ts` — Other (pure derivation)
- `src/lib/skills/level-table.ts` — Other (pure derivation)
- `src/lib/jobs-hierarchy.ts` — Other (static data + pure filter)
- `src/lib/utils/skill-effect-format.ts` — Other (renamed `LABELS`→`STATUP_LABELS`, exported)
- `src/pages/JobsPage.tsx` — Page
- `src/pages/JobDetailPage.tsx` — Page
- `src/App.tsx`, `src/components/app-sidebar.tsx`, `src/lib/breadcrumbs/routes.ts` — Other (wiring)
- Tests: `lib/__tests__/jobs-hierarchy.test.ts`, `lib/hooks/api/__tests__/useJobSkillDefinitions.test.tsx`, `lib/skills/__tests__/{level-table,skill-type}.test.ts`, `pages/__tests__/{JobsPage,JobDetailPage}.test.tsx`, `services/api/__tests__/skills.service.test.ts`

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | Grep `: any`/`as any`/`<any` over all task files: zero matches. Dynamic data uses `unknown` (e.g. `level-table.ts:71` casts `e[field] as number \| undefined`, narrowing a known-numeric union — not `any`) |
| FE-02 | No manual class concat | PASS | All `className` props are string literals; grep for `className={\`` / `className={"…"+` : zero matches |
| FE-03 | No direct API client in components | PASS | `@/lib/api/client` is imported only in `skills.service.ts:1`; pages import via hooks/services. Components: zero matches |
| FE-04 | No inline Zod in components | PASS | Read-only feature, no Zod anywhere in scope |
| FE-05 | No spinners for content loading | PASS | `JobDetailPage.tsx:133-138` uses `<Skeleton>` for loading; grep `animate-spin`: zero matches |
| FE-06 | No hardcoded colors | PASS | Semantic tokens only (`text-muted-foreground`, `text-destructive`, `text-primary`, `bg-background`). Grep for `bg-white/black/gray-N/red-N/...`: zero matches |
| FE-07 | No state mutation | PASS | `.sort()` at `useJobSkillDefinitions.ts:40` runs on a fresh array from `.map().filter()`; `.push()` at `level-table.ts:72,80,84` targets locally-constructed `columns`/`statupTypes`. No mutation of state/props |
| FE-08 | No default exports for components | PASS | `JobsPage.tsx:9` and `JobDetailPage.tsx:99` use named `export function`. (atlas-ui is React Router, not Next.js — named page exports are the documented convention) |
| FE-09 | Tenant guard in hooks | PASS | `useSkillDefinition.ts:52` `enabled: !!tenant?.id && skillId > 0`; `useJobSkillDefinitions.ts:29` same per-query guard; both take explicit `tenant` param |
| FE-10 | Tenant ID in query keys | PASS | `skillDefinitionKeys.detail(tenant?.id, skillId)` (`useSkillDefinition.ts:12-13`) — tenant id is the first key segment used by both the single and batch hooks, keeping the cache tenant-isolated |
| FE-11 | Error handling via React Query | PASS | Read-only surface: no `.catch` anywhere (grep zero). Errors surface through query error state — `JobDetailPage.tsx:139,143` render `text-destructive` messages on `isError`. `createErrorFromUnknown` is for imperative `.catch` blocks, none of which exist here |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | Wire type `SkillResource` (`skills.service.ts:61-73`) is `{ id, type, attributes }`; service maps it to the flat domain `SkillDefinition` (`:83-95`) — the documented adapter pattern, mirroring atlas-data effect.RestModel JSON tags (casing preserved: `MPConsume`, `hpR`, `MHPRRate`) |
| FE-13 | Service pattern | PASS | `skillsService` is a singleton object using the direct-`api`-client pattern (`skills.service.ts:77`), consistent with the existing `jobsService`/`charactersService` simple-resource style |
| FE-14 | Query key factory `as const` | PASS | `skillDefinitionKeys` (`useSkillDefinition.ts:10-14`) uses `as const` on both `all` and `detail` |
| FE-15 | Forms use RHF + zodResolver | N/A | No forms — read-only browser. Confirmed: no `useForm`/`zodResolver`/`z.object` in scope |
| FE-16 | Schema in lib/schemas with inferred type | N/A | No Zod schemas in scope (no write/validation surface) |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS | Both pages (`JobsPage.test.tsx`, `JobDetailPage.test.tsx`), both hooks of substance (`useJobSkillDefinitions.test.tsx`), service (`skills.service.test.ts`), and pure libs (`jobs-hierarchy`, `level-table`, `skill-type`) covered. Assertions test real behavior — version filtering, skeleton/empty/error states, icon fallback, table expansion, parallel fetch, column derivation |
| FE-18 | Mocks updated when services changed | PASS | `skills.service.test.ts:4-6` mocks `@/lib/api/client`; hook/page tests mock the service/hooks they depend on. New `getSkillById`/`maxLevel`/`effects` shape is exercised |

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- (Informational, not a guideline violation) `skillDefinitionKeys.all` (`useSkillDefinition.ts:11`) is `["skill-definition"]` with no tenant segment. The actively-used `detail` key carries the tenant id, so there is no cross-tenant leakage; `all` is only a hierarchical prefix. No change required for compliance.
- (Informational) `useJobSkillDefinitions` `useMemo` deps on the `results` array (`useJobSkillDefinitions.ts:46`); React Query returns a fresh array each render so the memo recomputes each render. Harmless given small fan-out; not a guideline rule.

---

# Plan-Adherence Audit — task-094 Jobs & Skills Browser

- **Auditor:** plan-adherence-reviewer
- **Plan:** docs/tasks/task-094-job-skill-browser/plan.md (11 tasks)
- **Branch:** task-094-job-skill-browser
- **Base:** main
- **Date:** 2026-06-13
- **Commits audited:** 0e0f4e940..7ae39a1c1 (10 task commits)
- **Build:** PASS (`npm run build`, built in 1.39s)
- **Full test suite:** PASS — 90 files / 783 tests
- **task-094 test files:** PASS — 8 files / 29 tests
- **Lint:** 48 errors / 7 warnings — all pre-existing; ZERO in any task-094 file (no new lint)
- **Overall verdict:** FULL adherence — READY_TO_MERGE

## Per-Task Verdict

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1 | skills.service.ts maxLevel + broadened SkillEffect | PASS | skills.service.ts:12-73 (broadened SkillEffect, optional maxLevel on SkillDefinition+SkillResource), :92 maps maxLevel; test skills.service.test.ts (2 cases). Additive only — full suite green. |
| 2 | fetchSkillDefinitionWithIcon + skillDefinitionRetry; hook routes through them | PASS | useSkillDefinition.ts:17-40 (both exports), :48 hook queryFn calls fetcher, :55 uses retry. Existing useSkillDefinition.test.tsx still passes. |
| 3 | useJobSkillDefinitions parallel useQueries, reuses detail key, empty-ids zero requests | PASS | useJobSkillDefinitions.ts:22-34 (useQueries over skillDefinitionKeys.detail), :36-46 memo sort-by-id. Test asserts 2 parallel + empty fires no requests. |
| 4 | deriveSkillType Buff/Active/Passive, degrades safely | PASS | skill-type.ts:12-21 (statup/overTime→Buff, action→Active, else Passive). 5-case test incl. missing-field degradation. |
| 5 | buildLevelTable + STATUP_LABELS reused (not a 2nd map) | PASS | skill-effect-format.ts:12 `LABELS`→`export STATUP_LABELS`, :28 formatStatup uses it; level-table.ts:2 imports STATUP_LABELS, :84 reuses it. 7-case test + existing skill-effect-format test green. |
| 6 | static JOB_HIERARCHY + pure filterHierarchy; leaf names via getJobNameById | PASS | jobs-hierarchy.ts:19-21 jobNodeName via getJobNameById, :47-101 hierarchy, :108-120 pure filter (no mutation). 5-case test green. |
| 7 | Ground minMajorVersion against live data; honest comment; test still passes | PASS | jobs-hierarchy.ts:23-40 — comment HONESTLY records the 2026-06-13 atlas-main probe finding (atlas-data does NOT version-gate; v83/v92/v95 return identical populated lists), reframes floors as a UI display-curation choice, not a data gate. Constants unchanged (correct: any floor>83 hides Cygnus/Legend on v83). Hierarchy test passes. Resolution is sound and transparent. |
| 8 | JobsPage version-filtered Collapsible tree, zero network | PASS | JobsPage.tsx:12-15 filterHierarchy memo (no fetch), :37-59 Collapsible tree. DOCUMENTED DEVIATION VERIFIED: inner class Collapsible (:43) has NO defaultOpen (plan had it) — resolves the job-100-"Warrior" == class-"Warrior" getByText collision. Archetype Collapsible keeps defaultOpen (:37). Both test cases pass. Sound. |
| 9 | JobDetailPage skill list + expandable per-level table; icon onError fallback; states | PASS | JobDetailPage.tsx:19-39 SkillIcon onError fallback, :41-68 LevelTable, :70-97 SkillRow (icon/title/badge/maxLevel), :99-157 loading/error/empty/list states. 5-case test green. (Cosmetic-only diff from plan: `Lv` wrapped in nested span :82-84, entity-escaped apostrophes — functionally identical.) |
| 10 | App.tsx + app-sidebar.tsx + breadcrumbs/routes.ts wiring | PASS | App.tsx:28-29 lazy imports, :88-89 routes; app-sidebar.tsx:57-60 Jobs entry; routes.ts:7 import, :147-160 two ROUTE_CONFIGS w/ labelResolver, :476-477 ROUTE_PATTERNS. Adds entityType:'job' + ROUTE_PATTERNS entries beyond plan letter (consistent with existing patterns). Build green. |
| 11 | Final gate: build + full suite + no-new-lint | PASS | build clean; 90 files/783 tests pass (incl. pre-existing flaky TenantsPage which passed); lint = pre-existing baseline (48 err), zero task-094-file errors. No separate commit needed (no lint fixes required) — matches plan. |

**Completion rate:** 11/11 (100%). Skipped without approval: 0. Partial: 0. Deferred: 0.

## Minor Deviations (all justified, none blocking)

1. **Task 1** — maxLevel mapped via conditional spread `...(maxLevel !== undefined && {maxLevel})` (skills.service.ts:92) rather than the plan's unconditional `maxLevel: skill.attributes.maxLevel`. STRICTER and correct: omits the key when absent instead of setting it to `undefined`. Both satisfy the test's `toBeUndefined()` assertion. No impact.
2. **Task 8** — inner class-level `defaultOpen` removed (documented; verified as the real implementation reason — name collision). No impact.
3. **Task 9** — `Lv {maxLevel}` markup wraps the value in a nested `<span>`; apostrophes HTML-entity-escaped. Cosmetic only.
4. **Task 10** — adds `entityType: 'job'` and `ROUTE_PATTERNS.JOBS/JOB_DETAIL` beyond the plan's literal snippet; matches sibling Item/Map route conventions. Improvement, not a gap.

## Action Items

None. The plan was faithfully executed end-to-end; the two pre-documented deviations (Task 7 probe finding, Task 8 collapse) are honestly recorded in-code and soundly resolved. No stubs, no deferrals, no silent skips.
