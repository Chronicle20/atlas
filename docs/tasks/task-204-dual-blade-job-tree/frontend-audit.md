# Frontend Audit — task-204-dual-blade-job-tree

- **Audit Scope:** `services/atlas-ui/src/components/features/jobs/__tests__/dual-blade-rail.test.ts` (the only file in `git diff origin/main...HEAD` under `services/atlas-ui/`), plus a no-change-required review of `src/components/features/jobs/rail-groups.ts`, `src/lib/jobs/job-graph.ts`, `src/components/features/jobs/advancement-flow.tsx`, `src/components/features/jobs/branch-rail.tsx`, and `src/pages/JobsPage.tsx`.
- **Guidelines Source:** frontend-dev-guidelines skill
- **Date:** 2026-08-09
- **Build:** PASS
- **Tests:** 1953 passed, 0 failed (240 files)
- **Overall:** PASS

## Build & Test Results

```
$ npm run build   (tsc -b && vite build, under nvm 22)
✓ built in 1.20s   (tsc -b type-checked the new test file; no errors)

$ npm test        (vitest run)
 Test Files  240 passed (240)
      Tests  1953 passed (1953)
```

Node/npm on PATH resolved to a Windows npm shim (`/mnt/c/Program Files/nodejs/npm`) that fails on the WSL UNC worktree path; ran under `nvm use 22` per project memory (`reference_atlas_ui_npm_nvm_and_lint_baseline`) to get a working `tsc`/`vite`/`vitest`.

## File Inventory

- `services/atlas-ui/src/components/features/jobs/__tests__/dual-blade-rail.test.ts` — **Test** (new file; the only TS change in the diff)
- `services/atlas-ui/src/lib/jobs/job-graph.ts` — **Other** (graph/derivation lib; unchanged, reviewed to verify the "no production change needed" claim)
- `services/atlas-ui/src/components/features/jobs/rail-groups.ts` — **Other** (rail curation; unchanged, reviewed for the same reason)
- `services/atlas-ui/src/components/features/jobs/advancement-flow.tsx` — **Component** (unchanged, reviewed for the same reason)
- `services/atlas-ui/src/components/features/jobs/branch-rail.tsx` — **Component** (unchanged, reviewed for the same reason)
- `services/atlas-ui/src/pages/JobsPage.tsx` — **Page** (unchanged, reviewed for the same reason)

## Does the five-deep, third-child Dual Blade line need a production change?

Traced every place depth, width, or subtree shape could plausibly be assumed and found none:

- `job-graph.ts:58-63` `childrenOf` — filters+sorts by `parent === id`; no cap on count. A third child of Rogue (`430`) is returned exactly like the existing two (`410`, `420`).
- `job-graph.ts:84-96` `jobTreePath` — walks `parent` links with a cycle guard (`visited` set), terminates only on `parent === null` or a repeat; no depth ceiling.
- `job-graph.ts:98-103` `ordinal` — has cases for 1st/2nd/3rd and a general `` `${n}th` `` fallback for anything else; not `if (n>3) throw` or similar. `tierLabel` (`:110-115`) is `jobTreePath(...).length - 1`, i.e. plain depth-from-root — confirmed by the test at `dual-blade-rail.test.ts:78-79` (`430` → `"2nd"`, `434` → `"6th"`, matching Assassin/Hermit/Night Lord's existing 2nd/3rd/4th pattern one-for-one).
- `job-graph.ts:123-147` `advancementChains` — DFS to every leaf, no branch-count or depth cap; a leaf yields `[]` regardless of how deep it is (Assassin's chain is 3 deep, Dual Blade's is 5 — same code path, same recursion, just more frames).
- `rail-groups.ts:58-83` `branchEntryOf`/`RAIL_GROUPS` — keyed by canonical `identity` (400 for the whole Rogue/Thief branch), not by wire id, depth, or child count. A node anywhere on the path (`jobTreePath`) that matches `identity: 400` routes the whole branch, including Dual Blade, into the Explorers/Thief entry. No enumeration of expected children.
- `advancement-flow.tsx:76-137` `AdvancementFlow` — builds a CSS grid where `rows = chains.length` and each chip's column is `anchorCols + 1 + k` (`k` = index within its own chain). Column count is derived per-chain from `chain.length`, not fixed; the container is wrapped in `overflow-x-auto` (`:92`) so a 5-wide chain scrolls rather than clipping. No `grid-template-columns` literal, no `slice(0, 3)`, no hardcoded row/column count anywhere in the file (confirmed via grep — zero hits for `slice(0, 3)`, `MAX_DEPTH`, `.length === 3`, `repeat(3`/`repeat(4`).
- `branch-rail.tsx` — renders `e.count` (from `subtreeCount`) and `g.entries` with a plain `.map`; no per-branch child-count assumption.
- `JobsPage.tsx:149-156` — the three-column *page* grid (`grid-cols-[200px_..._...]`) is the rail/advancement/detail layout split, unrelated to per-branch job-tree depth or width; not a hidden constraint on Dual Blade.

**Conclusion: the "no production TS change needed" premise in the test file's doc comment (`dual-blade-rail.test.ts:19-23`) is accurate.** Every helper the Dual Blade line exercises (`childrenOf`, `jobTreePath`, `advancementChains`, `tierLabel`, `subtreeCount`, `branchEntryOf`, `visibleRailGroups`, the `AdvancementFlow` grid) is depth- and width-generic; nothing enumerates "three tiers" or "two children" as a structural limit. This new test file is the only artifact needed to pin that behavior.

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -n ': any\|as any' dual-blade-rail.test.ts` — zero matches |
| FE-02 | No manual class concatenation | PASS | File contains no `className` usage at all (pure logic test, no rendering) |
| FE-03 | No direct API client calls in components | PASS | No import of `@/lib/api/client`; only imports `job-graph`, `availability.service` types, and `rail-groups` (`dual-blade-rail.test.ts:2-12`) |
| FE-04 | No inline Zod schemas in components | PASS | No `z.object`/`z.string` anywhere in the file |
| FE-05 | No spinners for content loading | N/A | No rendering in this file (pure data/logic test) |
| FE-06 | No hardcoded colors | N/A | No className/style usage — nothing to check |
| FE-07 | No state mutation | PASS | `THIEF_BRANCH`/`ALL_IDS`/`graph`/`preDualBlade` are all built once via `const` + pure calls (`:25-48`); no `.push`/`.splice`/`.sort` on component state (there is none) |
| FE-08 | No default exports for components | N/A | File exports nothing (test file); not a component module |
| FE-09 | Tenant guard in hooks | N/A | Not a hook file |
| FE-10 | Tenant ID in query keys | N/A | Not a query-key factory; test data is hand-built `JobAvailabilityEntry[]`, no React Query involved |
| FE-11 | Error handling with `createErrorFromUnknown` | N/A | No async/`.catch` code path in the file |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | N/A | `JobAvailabilityEntry` (`dual-blade-rail.test.ts:8`) is a plain derived-data type, not a JSON:API resource; consistent with its existing use elsewhere in the jobs feature (unchanged in this diff) |
| FE-13 | Service extends `BaseService` | N/A | No service code touched |
| FE-14 | Query key factory uses `as const` | N/A | No query key factory touched |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | N/A | No form code |
| FE-16 | Schema in `lib/schemas/` with inferred type | N/A | No schema added |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS | The change *is* the test; it exercises `buildJobGraph`, `advancementChains`, `subtreeCount`, `tierLabel` (`src/lib/jobs/job-graph.ts`) and `branchEntryOf`, `visibleRailGroups` (`src/components/features/jobs/rail-groups.ts`) against a realistic Rogue/Dual Blade fixture. All 4 `it()` blocks pass (`dual-blade-rail.test.ts:51,57,68,75`; verified via `npx vitest run` — 4 passed) |
| FE-18 | Mocks updated when services changed | N/A | No service module changed; test builds `JobAvailabilityEntry[]` fixtures directly rather than going through a mocked service, so no `__mocks__/` entry applies |

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None. The test file's own doc comment (`:14-23`) makes an explicit, checkable claim ("Nothing in the UI needed a change for this") — verified true above by tracing every consumer of depth/width-shaped data in the jobs feature. No stray edits were made to the worktree during this audit; `git status` is clean apart from this new `frontend-audit.md`.
