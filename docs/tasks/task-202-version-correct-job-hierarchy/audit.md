## Frontend guidelines review

- **Scope:** `git diff b748113ec HEAD -- 'services/atlas-ui/**'` (40 files: see prompt file list — job-graph rewrite + all consumers)
- **Guidelines source:** `.claude/skills/frontend-dev-guidelines/` (SKILL.md, anti-patterns, architecture-overview, patterns-react-query, patterns-forms-validation, patterns-service-layer, patterns-types, patterns-multitenancy, patterns-styling, patterns-components, testing-guide)
- **Build:** PASS — `npm run build` (`tsc -b && vite build`) clean, no type errors.
- **Tests:** PASS — `npm test` → 235 files / 1930 tests passed, 0 failed.
- **Overall: PASS** — no FE-* violations found in the branch's own changes.

### File inventory (branch diff only)

- Service: `services/api/availability.service.ts` — `JobAvailabilityResource`/`JobAvailabilityEntry` now carry `parent`/`identity`.
- Type/domain lib (new): `lib/jobs/job-graph.ts` — `JobNode`/`JobGraph` + pure graph helpers.
- Hook (new): `lib/hooks/api/useJobGraph.ts` — `useJobGraph()`, `useJobNameLookup()`.
- Hook (modified): `lib/hooks/api/useJobAvailability.ts`, `lib/hooks/usePresetJobOptions.ts`, `lib/hooks/useBreadcrumbs.ts`.
- Component: `components/features/jobs/{rail-groups.ts,advancement-flow.tsx}`, `components/features/characters/SkillsSection.tsx`, `components/features/characters/presets/{JobCombobox,JobSkillsAddButton,PresetCard,PresetEditor}.tsx`, `components/features/rankings/LeaderboardRow.tsx`.
- Page: `pages/{JobsPage,CharactersPage,GuildDetailPage}.tsx`, `pages/characters-columns.tsx`.
- Other: `lib/breadcrumbs/routes.ts` (non-component resolver-context plumbing).
- Deleted: `lib/jobs.ts`, `lib/jobs/job-advancement-tree.ts` (the v83-keyed static tables) and their tests.
- Test-only fixture (new): `lib/jobs/__tests__/job-graph-fixtures.ts`.

### Anti-Pattern checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` | PASS | `grep -rnE ': any\b|as any\b'` over all in-scope files: zero matches. |
| FE-02 | No manual class concatenation | PASS | All conditional classes route through `cn()` — e.g. `components/features/jobs/advancement-flow.tsx:38`, `components/features/characters/presets/JobCombobox.tsx:94`, `pages/JobsPage.tsx:150`. |
| FE-03 | No direct API client in components | PASS | No in-scope component/page imports `@/lib/api/client`; `availability.service.ts:1` is the only `client` import and it is the service module itself. |
| FE-04 | No inline Zod in components | PASS | No `z.object(`/`z.string(` in any in-scope component/page (none of this branch's work touches forms). |
| FE-05 | No spinner for content loading | PASS | `animate-spin` appears only on a submit/pending affordance — `JobSkillsAddButton.tsx:103,129` (per-row "adding skills" indicator inside a popover list, not a page/content loading state); `SkillsSection.tsx:26-34` and `pages/JobsPage.tsx` use `Skeleton`/text states instead. |
| FE-06 | No hardcoded colors | PASS | Job/rail styling goes through CSS custom properties (`--acc`, `--c-warrior`, etc., `rail-groups.ts:26-57`, `advancement-flow.tsx:41-51`) and semantic tokens (`bg-muted`, `text-muted-foreground`); `LeaderboardRow.tsx:21` uses `text-green-600` but that predates this branch's diff (pre-existing convention, not introduced here) — flagged non-blocking below. |
| FE-07 | No state mutation | PASS | `job-graph.ts:137,142` use `.push` on a **freshly-constructed local array** (`out`) inside a pure recursive builder, immediately returned — not a mutation of external/component state. No `.sort`/`.splice` on state anywhere in scope. |
| FE-08 | No default export components | PASS | All in-scope components/pages/hooks use named exports (`JobsPage`, `SkillsSection`, `useJobGraph`, etc.). |
| FE-09 | Tenant guard in hooks | PASS | `useJobAvailability.ts:37` `enabled: !!tenant?.id`; `useJobGraph.ts:36-38` sources `activeTenant` via `useTenant()` and composes two tenant-guarded queries; `usePresetJobOptions.ts:39-40` same. |
| FE-10 | Tenant ID in query keys | PASS | `useJobAvailability.ts:12-17` (`jobAvailabilityKeys.list(tenantId ?? "no-tenant")`); `useJobGraph`'s two composed queries (`useJobAvailability`, `useJobs`) are both tenant-keyed, so the derived graph is transitively tenant-isolated with no separate cache of its own — no cross-tenant bleed possible. |
| FE-11 | Error handling via `createErrorFromUnknown` | PASS (N/A) | No new `.catch(` sites in scope; `availability.service.ts`'s pagination guard throws synchronously (`Error`, line 62) and is surfaced through React Query's `isError`, the established pattern for query-layer failures in this codebase (matches `useJobAvailability`/`useJobGraph`'s own `isError` contract). |

### Architecture checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | `availability.service.ts:17-21` `JobAvailabilityResource { id, type, attributes }`. |
| FE-13 | Service extends BaseService / documented direct pattern | PASS | `availability.service.ts` uses the documented direct-`api` object pattern (matches other `services/api/*.service.ts` in this codebase — not BaseService-based even pre-branch). |
| FE-14 | Query key factory `as const` | PASS | `useJobAvailability.ts:12-17` `jobAvailabilityKeys` all branches `as const`. |
| FE-15 | Forms use react-hook-form + zodResolver | N/A | No forms touched by this branch. |
| FE-16 | Schema in `lib/schemas/` + inferred type | N/A | No Zod schemas touched by this branch. |

### Testing checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS | Every changed component/hook/page has a corresponding `__tests__` file in the diff (see stat list in the audit scope); `useJobGraph.test.tsx` covers pending/error/success/name-lookup; `job-graph.test.ts` covers the pure graph helpers directly. |
| FE-18 | Mocks updated when services changed | PASS | `availability.service.ts`'s new `parent`/`identity` fields are exercised end-to-end via `job-graph-fixtures.ts` (`FIXTURE_JOB_TREE`/`FIXTURE_JOBS_SORTED`), consumed only under `__tests__` (confirmed by `grep -rln job-graph-fixtures` — 11 hits, all `__tests__/*.test.tsx`). |

### Task-specific invariants (per the audit brief)

- **`identity` vs wire `id` discipline:** held everywhere checked. Curation keys on `identity` (`rail-groups.ts:15,26-57,61-66`, doc comment explains why); selection/routing/testids key on wire `id` (`JobsPage.tsx:74,117,159`, `advancement-flow.tsx:100,117` `data-testid=flow-cell-${id}`, `JobCombobox.tsx:35,83-103`).
- **`parent: null` vs `parent: 0`:** no coercion found. `grep -rnE '\.parent\s*(\?\?|\|\|)\s*0'` — zero matches. `job-graph.ts:46` correctly tests `e.parent !== null && keptIds.has(e.parent)`, never falsy-coerces.
- **Pending = unknown, not empty:** `SkillsSection.tsx:26-42` and `usePresetJobOptions.ts:43,49-53` (and its two callers `JobCombobox.tsx:29`, `JobSkillsAddButton.tsx:34-37`) all gate on `isPending`/`isSuccess`/`isError`, never on `graph.size`/`options.length`, matching the fixed regression described in the brief. Swept every remaining `useJobGraph()`/`useJobNameLookup()`/`usePresetJobOptions()` call site in scope (`JobsPage.tsx`, `LeaderboardRow.tsx`, `PresetCard.tsx`, `PresetEditor.tsx`, `useBreadcrumbs.ts`, `CharactersPage.tsx`, `GuildDetailPage.tsx`, `characters-columns.tsx`) — none render a definitive "not found"/"no X" statement off an unguarded empty graph. `JobsPage.tsx:51-63` explicitly retains the current jobId while pending rather than redirecting, with a comment citing design D10.
- **Non-component call sites take a resolver, don't call hooks:** `routes.ts:8-18,41-44,188` (`labelResolver(params, ctx)` / `BreadcrumbResolverContext`), fed by `useBreadcrumbs.ts:137-138`; `characters-columns.tsx:44,54` (`jobName: JobNameResolver` prop) fed by `CharactersPage.tsx:19,65`; `GuildDetailPage.tsx:23,133` calls the hook directly and passes a plain function into its own local `getMemberColumns`. All correct per the stated design.
- **`job-graph-fixtures.ts` production-import check:** PASS — all 11 importers found by `grep -rln job-graph-fixtures src` are under `__tests__/`; zero production-code imports.
- **FR-4.7 (no numeric version-literal comparison for job semantics):** PASS within scope. `grep -rnE 'major[vV]ersion\s*[<>=!]|minor[vV]ersion\s*[<>=!]'` across `components/`, `lib/`, `pages/`, `services/` returns hits only in unrelated subsystems this branch didn't touch (template/tenant/baseline version *selection* UI: `TemplatesPage.tsx`, `BaselinesPage.tsx`, `PacketMatrixPage.tsx`, `templates.service.ts`, `tenants.service.ts`, `socket/ancestry.ts`, etc.) — none gate job naming, parenting, or visibility.
- **React Query key/staleTime/enabled discipline for `useJobGraph`:** correct and safe. It composes `useJobAvailability` (own tenant-scoped key, 30 min stale / 24 h gc) and `useJobs` (`lib/hooks/api/useJobs.ts:25-29`, same stale/gc, tenant-keyed) — no separate cache/key of its own, so mounting it from `useBreadcrumbs` (app-wide, inside `AppShell`) and per-row in `LeaderboardRow`/`PresetCard` does **not** multiply network calls: React Query dedupes on the identical composed query keys, and 30 min staleTime means only the first mount per tenant session actually fetches. Hoisting to a shared context is a plausible micro-optimization but not required by any FE-* rule, and centralizing it would reintroduce prop-drilling this design deliberately avoided (see "non-component call sites" invariant above) — **not a blocking finding**.

### Summary

#### Blocking (must fix)
- None.

#### Non-Blocking (should fix / observations)
- `components/features/rankings/LeaderboardRow.tsx:21` hardcodes `text-green-600` for the "moved up" arrow (FE-06 hardcoded-color anti-pattern) instead of a semantic token. Pre-existing (not introduced by this branch's diff — the line is unchanged context around the branch's `useJobNameLookup` migration), but worth a follow-up since it sits directly beside code this task touched.
- `useJobGraph()` is mounted independently in `useBreadcrumbs` (global) and per-row in `LeaderboardRow`/`PresetCard`. Confirmed safe today (shared query keys, generous staleTime — see invariant note above), but if either underlying query's `staleTime` is ever lowered, per-row mounting on a large leaderboard/preset list would start firing many simultaneous background refetches. No action required now; worth a comment or a shared-selector hoist if that stale time ever changes.

---

## Plan adherence review

**Plan Path:** docs/tasks/task-202-version-correct-job-hierarchy/plan.md
**Audit Date:** 2026-08-08
**Branch:** task-202-version-correct-job-hierarchy
**Base Branch:** main (origin/main = b748113ec, already merged into this branch)

### Executive Summary

All 10 plan tasks were faithfully implemented; git evidence (68 files changed, +6323/-897) matches every workstream A–E described in the plan's File Structure table, and all five explicitly pre-approved deviations were verified present exactly as described. `libs/atlas-constants`, `libs/atlas-constants/gen`, and `services/atlas-data/atlas.com/data` all build, vet, and test clean; `services/atlas-ui` builds (type-checks) clean and its full Vitest suite passes (235 files / 1930 tests). The repo-root guards relevant to this branch (redis-key, goroutine, skill-job-id) all pass. No code changes were made during this audit. The plan.md checkboxes were never marked `[x]` (all 79 remain `- [ ]`), which is a process/paperwork gap only — every step has direct code/test evidence of completion.

### Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | JOB reader — absent `skill` node yields no model | DONE | `services/atlas-data/atlas.com/data/job/reader.go:46-58` branches on `ChildByName("skill")` error, returns empty model with FR-1.1 comment; `reader_test.go:102` `TestRead_MissingSkillNode_ProducesNoModel`; `processor_test.go:138` `TestRegisterJob_DragonImageCannotBlankRealDocument` (both walk orders) |
| 2 | SKILL worker — images seen vs documents written | DONE | `data/workers/skill.go:119-158` (`jobStats` struct, `Wrap`, `Log`), wired at `skill.go:69-71`; `skill_test.go:16-71` has all four `TestJobStats_*` tests specified in the plan |
| 3 | Split Cygnus 4th job into its own release class | DONE | `gen/availability.go:74-87` adds the `CygnusStage4` case ahead of the `1000..1599` range; `gen/availability_test.go:117,132` pins it plus the tier-1–3 no-regression guard; CSV has exactly 11 `job,CygnusStage4,false` rows (verified by grep, one per required version key incl. the trailing `gms,12,1` block); `job/availability_test.go` (107 lines) created with all 4 specified tests |
| 4 | FR-2.3 audit of remaining release classes | DONE | `availability-audit.md` (349 lines) covers GM/SuperGM/Pirate/Aran/Evan/Cygnus with per-version verdict tables, Step 1 command+output reproduced, live-query evidence cited; the unanticipated over-claim (jms 185 SuperGM) is fixed in the CSV (`released=false`) and pinned by `TestResolveWire_SuperGmNotBoundAtJms185` (`job/availability_test.go:67`); `docs/TODO.md:405` amended with the struck Cygnus half exactly as specified |
| 5 | Version-blind advancement parent relation | DONE | `libs/atlas-constants/job/parents.go` (161 lines) — `parents` map, `ParentIdentity`, `Set.ParentWire` all match the plan's verbatim code, including the FR-3.2/D7 doc comments; `parents_test.go` (121 lines) has all 5 specified tests |
| 6 | `job-availability` exposes `parent` and `identity` | DONE | `jobavailability/rest.go:10-24` — `RestModel.Parent *uint16`, `.Identity uint16`; `processor.go:44-49` populates both via `set.Job.ParentWire`; `resource_test.go` has all 5 new tests (`RootMarshalsNullParent`, `IdentityIsCanonicalNotWire`, `V72IdentityMatchesWireForPirate`, `NoParentPointsOutsideTheResponse`, plus the pre-existing pagination/V48/V72 tests retained) |
| 7 | atlas-ui — job graph plumbing (additive) | DONE | `services/api/availability.service.ts` widened (`JobAvailabilityEntry` with parent/identity, `fetchAllResources`); `lib/jobs/job-graph.ts` (167 lines) has all specified exports (`JobNode`, `JobGraph`, `buildJobGraph`, `childrenOf`, `rootOf`, `jobTreePath`, `tierLabel`, `advancementChains`, `subtreeCount`, `jobNodeName`); `lib/hooks/api/useJobGraph.ts` (68 lines) has `useJobGraph`/`useJobNameLookup`; `job-graph.test.ts` / `useJobGraph.test.tsx` created |
| 8 | Jobs page cluster onto the API graph | DONE | `rail-groups.ts` rewritten on identity keys (`RailEntry.identity`, `wireIdOf`, `branchEntryOf`, `visibleRailGroups`); `advancement-flow.tsx` takes `graph: JobGraph` prop, `available` prop removed; `JobsPage.tsx` uses `useJobGraph()`, `jobIdValid` gated on `isSuccess && graph.has(...)`; `job-advancement-tree.ts` retained (Task 9 deletes it) |
| 9 | Retire the static name tables | DONE | `git rm` confirmed: `lib/jobs.ts` and `lib/jobs/job-advancement-tree.ts` (+ its test) absent from the tree; every listed consumer (`LeaderboardRow`, `PresetEditor`, `PresetCard`, `JobCombobox`, `SkillsSection`, `characters-columns.tsx`, `CharactersPage.tsx`, `GuildDetailPage.tsx`, `usePresetJobOptions.ts`, `breadcrumbs/routes.ts`, `useBreadcrumbs.ts`) migrated to `useJobNameLookup`/`JobNameResolver`/`BreadcrumbResolverContext`; `grep -rn "JOB_GRAPH\|jobNameMap\|getJobNameById\|JOB_LIST\|job-advancement-tree" src` returns nothing (verified); the `major(Version)? (op) N` grep hits only `templates.service.ts`/`useTemplates.ts` guard checks (`majorVersion >= 0`), unrelated to job naming/parenting/visibility — FR-4.7 intact |
| 10 | Full verification and pre-PR review | DONE | All Go gates re-run clean in this audit (see Build & Test table); repo-root guards (redis-key, goroutine, skill-job-id) all exit 0; `npm run build` and `npm test` clean; no `go.mod` changed on this branch (`git diff b748113ec HEAD --name-only -- '**/go.mod'` empty) so the docker-bake step is correctly and non-silently skippable; `audit.md` exists with a frontend-guidelines-reviewer section already appended (this section adds plan-adherence); Step 8's `git push` deliberately not run per the human partner's call |

**Completion Rate:** 10/10 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

### Pre-Approved Deviations — Verified As Described

1. **Task 3** — `TestClassOf_KnownBoundaries` row for token 1512 updated `Cygnus → CygnusStage4` (`gen/availability_test.go:79`, comment: `// ThunderBreakerStage4 (task-202: split out of Cygnus)`); CSV `meymink` cell uses RFC-4180 doubled-quote escaping (`{""skills"":[]}`) rather than the plan's invalid backslash form — confirmed in the CSV grep output above. No factual content changed. **Confirmed as described.**
2. **Task 4** — `jms,185,1,job,SuperGM` flipped `released=true → false` in the CSV, with a full explanation row citing the missing 910 image and a 404 live query; pinned by `TestResolveWire_SuperGmNotBoundAtJms185` in `libs/atlas-constants/job/availability_test.go:67`. **Confirmed as described.**
3. **Task 7** — `rootOf`, `jobTreePath`, `advancementChains`, `subtreeCount` in `job-graph.ts` all carry per-path `visited`/`ancestors` cycle guards not present in the plan's verbatim code (lines 71-91 for the first two; explicit `ancestors: ReadonlySet<number>` params + skip-on-repeat logic in the latter two). **Confirmed present in all four helpers.**
4. **Task 9** — deleting `job-advancement-tree.ts` orphaned six fixture-consuming test files; `src/lib/jobs/__tests__/job-graph-fixtures.ts` was created as a test-only replacement, and every importer found (`PresetCard.test.tsx`, `LeaderboardRow.test.tsx`, `PresetEditor.test.tsx`, `JobCombobox.test.tsx`, `advancement-flow.test.tsx`, `rail-groups.test.ts`, `branch-rail.test.tsx`, `ClassAppearanceSection.test.tsx`, `JobSkillsAddButton.test.tsx`, `SkillsSection.test.tsx`, `JobsPage.test.tsx`) is under `__tests__`. **Confirmed.**
5. **Post-review fix wave (commit f15288860)** — `SkillsSection.tsx:17-40` now gates on `isPending`/`isError` from `useJobGraph()` before its empty-path branch; `usePresetJobOptions.ts` returns `{ options, isPending, isError }` (widened from a bare array), consumed with loading/error affordances in `JobCombobox.tsx:29` and `JobSkillsAddButton.tsx:35-37`. **Confirmed.**
6. **Task 10 Step 8** (`git push`) — not executed; branch has no upstream push in this audit's evidence, consistent with the human partner's call. Not reported as incomplete.

### Build & Test Results

| Module | Build | Vet | Tests | Notes |
|---|---|---|---|---|
| `libs/atlas-constants` | PASS | PASS | PASS | `go test ./... -count=1` — all packages ok (job, constants, field, item, map, merchant, monster, skill, summon) |
| `libs/atlas-constants/gen` (separate module) | PASS | PASS | PASS | `go run . -check` also exits 0 — "OK: ... up to date" (no generated-file drift) |
| `services/atlas-data/atlas.com/data` | PASS | — | PASS | `go build ./...` clean; `go test ./... -count=1` — all packages ok incl. `job`, `jobavailability`, `data/workers` |
| `services/atlas-ui` | PASS | n/a | PASS | `npm run build` (tsc -b && vite build) clean; `npm test` — 235 files / 1930 tests passed, 0 failed |

Repo-root guards run and confirmed exit 0: `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/skill-job-id-guard.sh` (14 divergent consts checked, clean).

### Skipped / Deferred Tasks

None. No task was skipped, silently dropped, or found only partially implemented.

### Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending Step 8's `git push`, which is explicitly the human partner's call, and any outstanding action items below)

### Action Items

1. (Cosmetic only) `plan.md`'s 79 step checkboxes were never flipped to `[x]` despite every step having verifiable evidence of completion. Not a functional gap — no action required unless the team wants the plan document itself to reflect status for future readers.
2. Confirm with the human partner whether `git push -u origin task-202-version-correct-job-hierarchy` (plan Task 10 Step 8) should now be run, since all verification gates pass.
