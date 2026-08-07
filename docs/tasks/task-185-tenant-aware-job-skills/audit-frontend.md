# Frontend Audit — task-185-tenant-aware-job-skills

- **Audit Scope:** atlas-ui TS/React surface for tenant-aware job/skill visibility — `types/api/responses.ts`, `lib/api/client.ts`, `services/api/jobs.service.ts`, `lib/hooks/api/useJobs.ts`, `lib/jobs/job-advancement-tree.ts`, `components/features/jobs/{rail-groups.ts,advancement-flow.tsx,branch-rail.tsx}`, `pages/JobsPage.tsx`, and their `__tests__`.
- **Guidelines Source:** frontend-dev-guidelines skill (anti-patterns, react-query, multi-tenancy, service-layer, types, styling, components, testing, architecture-overview, forms-validation)
- **Date:** 2026-07-27
- **Build:** PASS (evidenced in `.superpowers/sdd/plan/task-11-report.md` Step 7 — `tsc -b` + `vite build`, `✓ built in 1.28s`, no code changed since; not independently re-run per task instruction)
- **Tests:** 1340 passed, 0 failed (188 files) — same evidence source, Step 7
- **Overall:** NEEDS-WORK

## Build & Test Results

Verbatim from `.superpowers/sdd/plan/task-11-report.md` (Step 7, this exact worktree/branch):
```
npm run test
 Test Files  188 passed (188)
      Tests  1340 passed (1340)

npm run build
tsc -b + vite build completed, ending "✓ built in 1.28s"
```
Not re-run in this audit (no specific doubt required re-verification); trusted per task brief.

## File Inventory

- Type: `services/atlas-ui/src/types/api/responses.ts` — new `JsonApiResource`, `ApiResponse.included?`, `ApiPagedResponse`
- Other (API client): `services/atlas-ui/src/lib/api/client.ts` — new `api.getListDocument`
- Service: `services/atlas-ui/src/services/api/jobs.service.ts` — new `getJobs({includeSkills})`
- Hook: `services/atlas-ui/src/lib/hooks/api/useJobs.ts` — new React Query hook + `jobsKeys`
- Other (domain logic, not a component): `services/atlas-ui/src/lib/jobs/job-advancement-tree.ts` — floors deleted, `available: ReadonlySet<number>` predicates
- Component: `services/atlas-ui/src/components/features/jobs/rail-groups.ts`
- Component: `services/atlas-ui/src/components/features/jobs/advancement-flow.tsx`
- Component: `services/atlas-ui/src/components/features/jobs/branch-rail.tsx`
- Page: `services/atlas-ui/src/pages/JobsPage.tsx`
- Tests: `services/atlas-ui/src/services/api/__tests__/jobs.service.test.ts`, `.../lib/jobs/__tests__/job-advancement-tree.test.ts`, `.../components/features/jobs/__tests__/{rail-groups.test.ts,branch-rail.test.tsx,advancement-flow.test.tsx}`, `.../pages/__tests__/JobsPage.test.tsx`. **No test file exists for `lib/hooks/api/useJobs.ts`** (see FE-17).

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -n ": any\b\|as any\b"` over all 9 in-scope files returned zero matches |
| FE-02 | No manual class concatenation | PASS | `grep -n 'className={"'` over all in-scope `.tsx` files returned zero matches; `advancement-flow.tsx:35-40`, `branch-rail.tsx:56` use `cn()` |
| FE-03 | No direct API client calls in components | PASS | `grep` for `@/lib/api/client` under `components/features/jobs/` and `pages/JobsPage.tsx` returns zero matches; `jobs.service.ts:1` is the only importer of `@/lib/api/client` in scope |
| FE-04 | No inline Zod schemas in components | PASS | `grep -n "z\.object\|z\.string("` over in-scope components/pages returns zero matches |
| FE-05 | No spinners for content loading | PASS | `grep -n "animate-spin"` over in-scope files returns zero matches; `branch-rail.tsx:20-35` renders `<Skeleton>` (imported `branch-rail.tsx:3`) while `isPending` |
| FE-06 | No hardcoded colors | PASS | `grep -nE "bg-(white\|black\|gray-[0-9]\|red-[0-9]\|blue-[0-9]\|green-[0-9])"` returns zero matches; colors are theme tokens (`--c-warrior` etc., `rail-groups.ts:20-46`) or semantic classes (`text-muted-foreground`, `bg-card`, `bg-secondary` in `advancement-flow.tsx:39,46,49`) |
| FE-07 | No state mutation | PASS | No `.push(`/`.splice(`/`.sort(` on component state in scope; `jobs.service.ts:41,48` `jobs.push(...)` mutates a local accumulator array that is never React state, then is returned wholesale — not a mutation of externally-visible state |
| FE-08 | No default exports for components | PASS | `grep -n "export default"` over all in-scope files returns zero matches; e.g. `JobsPage.tsx:30` `export function JobsPage()` |
| FE-09 | Tenant guard in hooks | PASS | `useJobs.ts:20` takes explicit `tenant: Tenant \| null \| undefined` param; `useJobs.ts:25` `enabled: !!tenant?.id` |
| FE-10 | Tenant ID in query keys | **FAIL** | `useJobs.ts:7`: `list: (tenantId: string \| undefined) => ["jobs", tenantId] as const` — no `\|\| 'no-tenant'` / `?? 'no-tenant'` fallback, unlike every other tenant-scoped key factory in the codebase (`useBans.ts:23`, `useAccounts.ts:35`, `useConversations.ts:47`, `useNpcs.ts:36`, `useGuilds.ts:36`, `useQuests.ts:22`, all use `tenant?.id ?? "no-tenant"` or `\|\| "no-tenant"`). Currently masked by the `enabled: !!tenant?.id` guard (a disabled query never populates the `["jobs", undefined]` cache entry), but it is a real deviation from the documented/enforced convention and becomes a live collision risk the moment anything changes the `enabled` condition. |
| FE-11 | Error handling with `createErrorFromUnknown` | PASS (N/A) | Zero `.catch(` in scope (`grep` confirmed); errors surface through React Query state instead — `JobsPage.tsx:142-151` renders a distinct `jobsQuery.isError` card rather than swallowing the error, which is the query-based equivalent of the anti-pattern's intent |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | `jobs.service.ts:4-8` `JobResource { id: string; type: string; attributes: {...} }`; `types/api/responses.ts:5-9` `JsonApiResource { type, id, attributes? }` |
| FE-13 | Service extends `BaseService` (when applicable) | PASS | `jobs.service.ts:23` uses the plain-object direct-client pattern, consistent with the codebase majority (30 of 43 `services/api/*.service.ts` files use this pattern, e.g. `accounts.service.ts`, `bans.service.ts`, `quests.service.ts`) — not a BaseService candidate since it needs no validation/transformation |
| FE-14 | Query key factory uses `as const` | PASS | `useJobs.ts:6,7` both key entries end `as const` |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | N/A | No form added in this branch's scope |
| FE-16 | Schema in `lib/schemas/` with inferred type | N/A | No new Zod schema added in this branch's scope (confirmed via FE-04 grep) |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | **FAIL (partial)** | `lib/hooks/api/useJobs.ts` has no corresponding test file — `find src/lib/hooks/api/__tests__ -iname "*useJobs*"` returns nothing, while sibling hooks in the same feature (`useJobSkills.ts` → `useJobSkills.test.tsx`, `useJobSkillDefinitions.ts` → `useJobSkillDefinitions.test.tsx`) both have direct `renderHook`-based tests exercising `enabled`/data/error states. All other in-scope files have test files (see File Inventory) and those tests are behavior-based and substantive (`job-advancement-tree.test.ts`, `rail-groups.test.ts`, `branch-rail.test.tsx`, `advancement-flow.test.tsx`, `JobsPage.test.tsx`, `jobs.service.test.ts` all assert on rendered output / returned data / call arguments, not on mock-internal plumbing) |
| FE-18 | Mocks updated when services changed | PASS | `jobs.service.test.ts:5-10` mocks `@/lib/api/client`'s `getListDocument` directly and asserts on the URL/pagination/`included`-indexing behavior actually implemented in `jobs.service.ts`; `JobsPage.test.tsx:35-38` mocks `useJobs` at the hook boundary and drives all three query states (`pending`/`success`/`error`) via the `jobsQuery()` helper (lines 309-329) |

## Crux Check: JobsPage async-state handling (per audit brief §A)

| Item | Status | Evidence |
|------|--------|----------|
| `jobIdValid` gated on `isSuccess`, not truthiness | PASS | `JobsPage.tsx:54-59`: `jobIdValid = parsedJobId !== null && Number.isInteger(parsedJobId) && JOB_GRAPH[parsedJobId] !== undefined && jobsQuery.isSuccess && available.has(parsedJobId)` — explicit `.isSuccess`, not `!!jobsQuery.data` |
| Redirect effect gated on `isSuccess` | PASS | `JobsPage.tsx:72-81`: effect condition is `activeTenant && jobsQuery.isSuccess && parsedJobId !== null && !jobIdValid` |
| `BranchRail` shows skeleton while pending, not spinner | PASS | `JobsPage.tsx:161-166` passes `isPending={jobsQuery.isPending}`; `branch-rail.tsx:20-35` renders `<Skeleton>` rows, tested at `JobsPage.test.tsx:354-371` (`renders the rail skeleton while the job set is loading`) |
| `defaultJobId` only consulted once a successful load confirms `available` | PASS (with a caveat) | `groups` (line 47-50) — the source `defaultJobId` (line 51) is derived from — is `[]` unless `jobsQuery.isSuccess`, so `defaultJobId` resolves to the hardcoded fallback `100` until success. `jobId` (line 63-67) only actually *selects* `defaultJobId` as the active job either when `isSuccess` is true and the route param is invalid (line 66), or as a last-resort placeholder when there is no route param at all and the query is still pending/errored (line 67, `parsedJobId ?? defaultJobId`) — i.e., it never overrides a real deep-link while pending (matches the design-D10 comment at lines 60-62), it only fills in a value to render *something* on bare `/jobs` while loading. Not a defect, but worth naming precisely since the literal English of the brief's check ("only consulted once... non-empty") doesn't hold for the bare-`/jobs`-while-pending sub-case. |
| `jobsQuery.isError` renders a distinct error state | PASS | `JobsPage.tsx:142-151`, `data-testid="jobs-load-error"`, distinct copy ("Could not load this tenant's job list..."); tested at `JobsPage.test.tsx:373-393` which also asserts the skeleton testid is *absent* in the error case, proving the two states don't visually collide |

## Pagination loop (`jobs.service.ts:40-55`, per audit brief §C)

| Check | Status | Evidence |
|-------|--------|----------|
| Terminates on a normal/empty final page | PASS | `url = doc.links?.next` (line 54) becomes `undefined` when the server omits `next`, ending the `while (url)` loop; tested at `jobs.service.test.ts:52-70` |
| Accumulates `data` and `included` across pages | PASS | `jobs.push(...(doc.data ?? []))` (line 48) runs every iteration; `skillsById.set(id, inc)` (line 52) runs inside the per-page `for` loop over `doc.included ?? []`, so both accumulate across all pages, not just the first/last |
| **No infinite-loop guard against a self-referential or malformed `links.next`** | **FAIL** | Lines 40-55: `while (url) { ...; url = doc.links?.next; }` has no visited-URL set, no iteration cap, and no check that the new `url` differs from the one just fetched. If the backend (or a future regression in it) ever returns `links.next` pointing at the same page again, or a `next` cursor that never advances, this loop runs forever, growing `jobs`/`skillsById` unbounded, hanging the tab. The backend contract as described (max page size 250, ~82 jobs, "links.next... to follow for subsequent pages") is well-behaved today, but nothing in this client code enforces or defends against a violation of that assumption — this is exactly the kind of trust boundary a client should not fully outsource to server correctness. |

## Summary

### Blocking (must fix)
- FE-10 — `useJobs.ts:7` `jobsKeys.list` omits the `?? 'no-tenant'` fallback every other tenant-scoped key factory in the codebase uses; currently inert only because of the `enabled` guard, but it's a real, citable deviation from the enforced multi-tenancy convention (`useJobs.ts:7` vs. e.g. `useBans.ts:23`).
- Pagination loop (`jobs.service.ts:40-55`) has no defense against a self-referential or non-advancing `links.next`, which would hang the browser tab in an unbounded `while` loop. Add either a visited-URL `Set` or a hard iteration cap (e.g., derived from `meta.page.last`) before advancing `url`.
- FE-17 — `lib/hooks/api/useJobs.ts` has no test file, unlike its two sibling hooks (`useJobSkills`, `useJobSkillDefinitions`) added in the same feature area, both of which do have direct `renderHook` coverage. The hook's `enabled` guard and query-key derivation are never exercised directly — `JobsPage.test.tsx` mocks `useJobs` away entirely, so nothing in the test suite would catch a regression in the hook itself (e.g., the FE-10 issue above).

### Non-Blocking (should fix)
- None beyond the items already listed as blocking; no other FE-* deviations were found across the anti-pattern, architecture, or remaining testing checklist items.
