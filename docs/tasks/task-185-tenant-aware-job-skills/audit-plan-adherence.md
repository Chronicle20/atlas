# Plan Audit — task-185-tenant-aware-job-skills

**Plan Path:** docs/tasks/task-185-tenant-aware-job-skills/plan.md
**Audit Date:** 2026-07-27
**Branch:** task-185-tenant-aware-job-skills
**Base Branch:** main
**Merge base:** 60f6ddaa3 → HEAD bab6d9380 (20 commits)

## Executive Summary

All 11 plan tasks were faithfully implemented; every task's code, tests, and documentation were verified directly against the working tree, not merely against the plan text or implementer self-reports. The five documented deviations (exported `skill.ParseJobId` instead of a duplicate `job.parseJobId`; a `\btype\b`-scoped regex instead of a literal substring check; the corrected "list endpoint never 404s" wording; the Task-9→Task-10 test-coverage restoration; the shared `Skeleton` component instead of a raw pulse div) are all present in the code exactly as the execution ledger (`progress.md`) describes, and none of them weaken the plan's intent. `go build`/`go vet`/`go test ./job/... ./data/workers/... ./baseline/... ./tenantpurge/...` and `libs/atlas-constants/job` tests were re-run during this audit and pass (cached, consistent with Task 11's prior race-enabled run). No `// TODO`, stub, or 501 was introduced by this branch; the only pre-existing `TODO` in `libs/atlas-constants/job/model.go:114` predates this branch and was not touched. Completion rate: 11/11 tasks (100%); 0 skipped without approval; 0 unresolved partials.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `job` reader and registry | DONE | `services/atlas-data/atlas.com/data/job/reader.go`, `registry.go`, `reader_test.go` (7 tests) implement `Read`, `GetModelRegistry`; deliberately calls `skill.ParseJobId` (exported at `skill/reader.go:31`) instead of a duplicate local `parseJobId`, per the ledger's Task-1 human ruling — `job/reader.go:4,40`. |
| 2 | Storage-backed job processor and resource wiring | DONE | `job/processor.go` matches plan verbatim: `NewStorage`, `NewProcessor`, `GetSkillsForJob` (404-collapse at line 51-58), `Register`, `RegisterJob` (int,error) at lines 73-90. `job/resource.go:20-48` wires `InitResource(db)`, `handleGetJobSkills`. `main.go:184` = `job.InitResource(db)(GetServer())`. `job/mock/processor.go` matches interface 1:1. |
| 3 | `GET /data/jobs` compound list endpoint | DONE | `job/list_rest.go` implements `ListRestModel`/`ListFrom`/`ListFromAll`/`WithResolvedSkills` exactly as specified (unconditional linkage via `GetReferencedIDs`, gated `included` via `GetReferencedStructs`). `job/resource.go:26,55-64,66-111` adds the route, `includesSkills`, `handleGetJobsRequest`. `job/rest_test.go:22` (`TestStoredContentCarriesNoRelationships`) and `job/resource_test.go:223-300` (5 list tests) all present and pass. |
| 4 | Ingest — write JOB documents from the SKILL worker | DONE | `data/workers/skill.go:57-67` inserts the JOB pass after `mobskill`, before the icon loop, using `countingRegister`/`logJobDocCount` (lines 113-137) — no new `data.Workers` entry. `data/workers/skill_test.go` has all 4 specified tests. |
| 5 | Verify JOB documents ride baseline publish and tenant purge | DONE (strengthened) | `baseline/dump_test.go:39-60` (`TestCopyOutSQLDocumentsHasNoTypeFilter`) uses a `\btype\b` regex scoped to the WHERE clause (`typeColumnRef` at line 37) rather than the plan's literal `"type ="` substring check — a controller-ruled strengthening (ledger Task-5) because the literal check misses `type IN(...)`/`type='x'`/`type <> 'x'`. `tenantpurge/purge_test.go:49-55` adds the `'JOB'` row alongside `'item'`. No production code changed, matching D9's "verify not assert." |
| 6 | Collapse `libs/atlas-constants/job` | DONE | `job/constants.go` is 185 lines (plan: <200), 0 `Job` value vars (`grep -c "^var [A-Z]... = Job{"` → 0), `Jobs` is a literal map with exactly 23 `fourthJob: true` markers verified ID-for-ID against `constants_test.go`'s `fourthJobIdsUnderTest()`. `model.go:9-20` drops `Skills()`/`Buffs()`, keeps the struct shape `{id, fourthJob}`. `go.mod` diff vs main is empty. `Id`/`Type` constants and `advancement.go` untouched (empty diff). |
| 7 | Documentation — REST reference and rollout runbook | DONE (corrected) | `services/atlas-data/docs/rest.md` documents `GET /api/data/jobs` and corrects the `/{jobId}/skills` 404 wording (line 1137). `docs/runbooks/job-document-backfill.md` states the list endpoint returns 200/empty/`meta.total:0` and only `/skills` 404s (lines 12-13) — the plan's own Task-11-Step-10 text ("both job endpoints 404") is factually wrong per code, and this doc was written correctly from the start (per ledger, no correction was needed here — only the plan's restated claim needed correcting downstream in the PR description). Baseline-publish request body includes the mandatory JSON:API envelope (lines 65-75), a fix-round addition beyond the plan's original snippet. No absolute/home paths found in either file. |
| 8 | atlas-ui compound-document plumbing | DONE | `types/api/responses.ts` adds `JsonApiResource` + `ApiResponse.included?`. `lib/api/client.ts:365-372` adds `getListDocument` beside (not replacing) `getList`. `services/api/jobs.service.ts` implements `getJobs({includeSkills})` with `links.next` following. `lib/hooks/api/useJobs.ts` implements the React Query hook. `services/api/__tests__/jobs.service.test.ts` has all 5 specified test cases. |
| 9 | Retire the version floors from the job tree | DONE | `lib/jobs/job-advancement-tree.ts` — `BRANCH_FLOORS`/`NODE_FLOORS`/`floorOf` fully deleted; `visibleRoots`, `visibleChildrenOf`, `advancementChains`, `subtreeCount` all take `available: ReadonlySet<number>`. `rail-groups.ts` and `advancement-flow.tsx` updated to match. `grep -rn "BRANCH_FLOORS\|NODE_FLOORS\|floorOf" services/atlas-ui/src/` → empty. |
| 10 | Wire `JobsPage` to the tenant job set | DONE | `pages/JobsPage.tsx:37-97` wires `useJobs`, gates `jobIdValid`/redirect/`loading` on `jobsQuery.isSuccess`/`isPending`, adds the `jobs-load-error` branch. `branch-rail.tsx:1-36` adds the `isPending` skeleton using the shared `Skeleton` component (sound deviation from the plan's raw `animate-pulse` div, per ledger). Task 9's dropped assertions (orphan-parent invariant, leaf-returns-`[]`, childless-root `tierLabel`) are restored in `job-advancement-tree.test.ts` (lines 105, 118-122, 125-133) — deliberate extra work beyond Task 10's own scope, done on the same branch per the ledger. |
| 11 | Full verification gates | DONE | `task-11-report.md` documents all 10 steps; independently re-verified in this audit: `go build ./...` clean in both `libs/atlas-constants` and `services/atlas-data/atlas.com/data`; `go test ./job/... ./data/workers/... ./baseline/... ./tenantpurge/...` and `libs/atlas-constants/job` tests pass; mechanical acceptance greps (line count, 0 `Job` vars, no `Skills()`/`Buffs()`, no floor identifiers, no `constJob`, unchanged `go.mod`) all confirmed independently. |

**Completion Rate:** 11/11 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. No task was skipped, stubbed, or left partially implemented. All deviations from the plan's literal text (enumerated in `progress.md` and repeated per-task above) were adjudicated by the human/controller during execution and are recorded as intentional improvements, not gaps:

1. **Task 1** — `skill.ParseJobId` exported and reused instead of a duplicated `job.parseJobId`, per an explicit human ruling that the plan's D1 dependency-direction rationale was a non-sequitur (Task 3 makes `job` import `skill` anyway).
2. **Task 5** — the baseline type-filter guard test was strengthened from a literal substring check to a `\btype\b` WHERE-clause-scoped regex, per controller ruling, because the literal check would have missed `type IN(...)` / `type='x'` / `type <> 'x'` predicates — a stricter, not weaker, test than the plan specified.
3. **Task 7 / Task 11 Step 10** — the plan's own claim that "both job endpoints 404" is factually wrong (confirmed against `job/resource.go:66-111` and `TestGetJobs_EmptyForTenantWithNoJobs`); the runbook and PR description correctly state the list endpoint returns 200 with an empty array / `meta.total:0`, and only `/{jobId}/skills` 404s.
4. **Task 9 → Task 10** — Task 9's rewrite of `job-advancement-tree.test.ts` dropped three pre-existing assertions (orphan-parent invariant, leaf-returns-`[]`, childless-root `tierLabel`); Task 10 restored all three non-vacuously, confirmed present in the current file.
5. **Task 10** — `BranchRail`'s pending state uses the shared `Skeleton` component (`components/ui/skeleton`) rather than the plan's inline `animate-pulse` div — a sound, equivalent substitution.

None of these represent unresolved gaps; each is closed in the shipped code and independently re-confirmed by this audit.

## Build & Test Results

Re-run directly during this audit (not merely re-quoting Task 11's report):

| Service / Module | Build | Tests | Notes |
|---|---|---|---|
| `services/atlas-data/atlas.com/data` | PASS (`go build ./...`) | PASS (`go test ./job/... ./data/workers/... ./baseline/... ./tenantpurge/...`, cached `ok`) | No failures; matches Task 11's `go test -race ./...` full-module run. |
| `libs/atlas-constants` | PASS (`go build ./...`) | PASS (`go test ./job/... -v`, `TestAdvancement` + table tests) | `constants_test.go`'s `TestJobsTableShape`/`TestFourthJobMembership`/`TestFromSkillIdStillResolves` all pass; matches Task 11's `go test -race ./...`. |
| `services/atlas-character`, `services/atlas-skills`, `services/atlas-configurations` (unaffected consumers) | Reported PASS in `task-11-report.md` | Reported PASS in `task-11-report.md` | Not re-run in this audit (slow, and `git diff --stat` over these three trees is empty — no source changes to verify against). Cited per instruction rather than re-run. |
| `services/atlas-ui` | Reported PASS (`npm run build`, `tsc -b` + `vite build`) in `task-11-report.md` | Reported PASS (188 files / 1340 tests) in `task-11-report.md` | Not re-run in this audit per instruction (slow, already evidenced); spot-checked that the specific test files claimed (`jobs.service.test.ts`, `job-advancement-tree.test.ts`'s three restored assertions, `JobsPage.test.tsx`'s new cases) exist with the expected test names/assertions in the current tree. |
| `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check` | N/A | PASS per `task-11-report.md` (one fix-round commit `bab6d9380` for prettier drift in a Task-10-created file, then clean re-run) | Not re-run in this audit; the fix commit is visible in `git log` and its diff (`JobsPage.test.tsx`, 10 ins / 5 del, pure re-wrap) is consistent with a formatting-only fix. |
| `docker buildx bake atlas-data` | N/A — not required | — | `go.mod`/`go.sum` diff vs `main` is empty (`git diff --stat main -- '**/go.mod' '**/go.sum'` → no output), so per CLAUDE.md item 4 the bake gate does not apply. Confirmed independently in this audit. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

Every one of the 11 tasks' produced interfaces, files, and tests exist in the working tree exactly as specified (modulo the five adjudicated, documented, and verified-sound deviations). The Global Constraints all hold: `libs/atlas-constants/go.mod` is unchanged; `Jobs` retains all 82 ids and 23 `fourthJob` markers (verified by both `constants_test.go` and this audit's direct grep); `Job` remains a struct; `Id`/`Type`/`advancement.go` are untouched (empty diffs); the hard cutover has no fallback to `constJob.Jobs[id].Skills()` anywhere (`grep -rn "constJob" services/atlas-data/atlas.com/data/job/` → empty, and `Skills()`/`Buffs()` are fully removed from `libs/atlas-constants/job`); stored `content` carries no `relationships`/`included` (enforced by `TestStoredContentCarriesNoRelationships`); no `// TODO`, stub, or 501 was introduced by this branch's diff; no absolute/home paths appear in any committed runbook or rest.md content.

## Action Items

None required before merge. Minor items already logged as deliberately-deferred in `progress.md` (e.g., no test asserts the paginated `links` envelope on `GET /data/jobs`; `jobsKeys.list` lacks an explicit no-tenant-fallback test; `links.next` loop has no visited-URL guard) are low-risk, pre-existing-pattern-consistent, and were explicitly triaged as out-of-scope-for-this-task rather than silently dropped — no further action is required to consider this plan complete.
