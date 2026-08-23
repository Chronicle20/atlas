# Frontend Audit — task-258-refreshable-empty-states

- **Audit Scope:** `git diff 1461bfc96..HEAD -- services/atlas-ui` (28 files: EmptyState, DataTableWrapper, useGridRefresh, tenant-context, and 16 pages + their tests)
- **Guidelines Source:** frontend-dev-guidelines skill
- **Date:** 2026-08-21
- **Build:** PASS
- **Tests:** 2131 passed, 0 failed (259 test files)
- **Overall:** PASS

## Build & Test Results

Run in an isolated scratch worktree at `e209ae745` (per instructions, not the shared tree):

```
$ npm run build
✓ built in 2.58s   (tsc -b && vite build, no errors)

$ npm test
 Test Files  259 passed (259)
      Tests  2131 passed (2131)
   Duration  63.83s
```

No failures. `tools/verify.sh` was not re-run per instructions (already green on this branch, confirmed by the branch's own final ledger commit).

## File Inventory

- **Component:** `src/components/common/EmptyState.tsx` — adds `onRefresh`/`isRefreshing`/`lastUpdatedAt` props, action row, caption
- **Component:** `src/components/common/DataTableWrapper.tsx` — threads the three new props to `EmptyState` on the empty branch only
- **Hook:** `src/lib/hooks/useGridRefresh.ts` — `RefreshableQuery` structural interface (D1), `lastUpdatedAt` derivation (D2)
- **Other:** `src/context/tenant-context.tsx` — `tenantsUpdatedAt`, widened `refreshTenants(): Promise<TenantRefreshResult>`
- **Page:** `AccountsPage.tsx`, `BansPage.tsx`, `CharactersPage.tsx`, `CouponsPage.tsx`, `EventDefinitionsPage.tsx`, `EventOccurrencesPage.tsx`, `GuildDetailPage.tsx`, `GuildsPage.tsx`, `MapsPage.tsx`, `MerchantsPage.tsx`, `QuestsPage.tsx`, `ReportsPage.tsx`, `RewardPoolsPage.tsx`, `ServicesPage.tsx`, `TemplatesPage.tsx`, `TenantsPage.tsx`, `TransportsPage.tsx` — wire `lastUpdatedAt`/`onRefresh`/`isRefreshing` through to their grid (`DataTableWrapper` for all but `TransportsPage`'s scheduled tab and `EventOccurrencesPage`'s populated table, which render `EmptyState` directly against a raw `<Table>`)
- **Tests:** `EmptyState.test.tsx`, `DataTableWrapper.test.tsx`, `useGridRefresh.test.ts` (new), plus additions to `TenantsPage.test.tsx`, `EventOccurrencesPage.test.tsx`, `RewardPoolsPage.test.tsx`, `TransportsPage.test.tsx`

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `git diff ... \| grep -E '^\+.*(: any\|as any)'` — zero matches in the diff |
| FE-02 | No manual class concatenation | PASS | `git diff ... \| grep -E '^\+.*className=\{".*\+'` — zero matches; `cn()` used throughout, e.g. `src/components/common/EmptyState.tsx:57` |
| FE-03 | No direct API client calls in components | PASS | `git diff ... \| grep -E 'from "@/lib/api/client"'` — zero matches added |
| FE-04 | No inline Zod schemas in components | PASS | No `z.object(`/`z.string(` added in this diff |
| FE-05 | No spinners for content loading | PASS | Only new `animate-spin` usage is `src/components/common/EmptyState.tsx:57`, on the refresh **button** icon — identical convention to the pre-existing header refresh button in `src/components/data-table.tsx:88`, which this diff does not change. No spinner substitutes for a page/content loading state. |
| FE-06 | No hardcoded colors | PASS | `git diff ... \| grep -E 'bg-white\|bg-black\|bg-gray-\|text-gray-\|border-gray-\|bg-red-\|text-red-'` — zero matches |
| FE-07 | No state mutation | PASS | `git diff ... \| grep -E '\.(push\|splice\|sort)\('` — zero matches added (`RewardPoolsPage.tsx` comment even calls out replacing a mutation-adjacent bug with `.reduce`, but that's pre-existing logic, not new mutation) |
| FE-08 | No default exports for components | PASS | `git diff ... \| grep -E 'export default function'` — zero matches added |
| FE-09 | Tenant guard in hooks | PASS (n/a) | `useGridRefresh` is not itself a data-fetching hook — it composes already-tenant-guarded `UseQueryResult`s (or the `TenantsPage` context shim) passed in by the caller; it performs no fetch of its own, so no `enabled: !!tenant?.id` guard applies to it. `TenantsPage.tsx:48-64` sources its `RefreshableQuery` from `useTenant()`'s own (pre-existing, tenant-agnostic) `refreshTenants`. |
| FE-10 | Tenant ID in query keys | PASS (n/a) | No new query key factories added in this diff; all wired pages reuse pre-existing hooks/query keys unchanged. |
| FE-11 | Error handling with `createErrorFromUnknown` | PASS | `useGridRefresh.ts:47-50` (pre-existing, unmodified logic) routes failures through `toast.error(failed.error, ...)`, and `src/lib/utils/toast.ts`'s `error()` internally calls `transformError`/`transformApiError`/`transformValidationError` (the app's sanctioned error-transform layer) plus `logError` — this is the established error-surfacing path, not a raw `console.error`. `TenantsPage.tsx`'s `tenantsSource.refetch` (new in this diff, lines 51-64) correctly forwards `result.error` (itself a `createErrorFromUnknown`-shaped object from `tenant-context.tsx:141`) into that same path rather than swallowing it. |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS (n/a) | No new models in this diff |
| FE-13 | Service extends `BaseService` | PASS (n/a) | No new/changed service classes in this diff |
| FE-14 | Query key factory uses `as const` | PASS (n/a) | No new query key factories in this diff |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | PASS (n/a) | `TenantsPage.tsx`'s rename form (`useForm({ resolver: zodResolver(tenantNameSchema) })`, line ~76) is pre-existing and untouched by this diff's refresh feature |
| FE-16 | Schema in `lib/schemas/` with inferred type | PASS (n/a) | No new schemas in this diff |
| — | D1 structural `RefreshableQuery` vs `Pick<UseQueryResult, …>` | PASS | `src/lib/hooks/useGridRefresh.ts:6-13` — `UseQueryResult` genuinely has `dataUpdatedAt: number` (verified in `@tanstack/query-core`'s `.d.ts`), so real query results satisfy the interface structurally; the hand-rolled interface is required because `TenantsPage`'s tenant-context-backed source (`TenantsPage.tsx:49-64`) is not a `UseQueryResult` at all and could not be expressed via `Pick<UseQueryResult, …>`. `refetch`'s return-type is a structural subset (`{ isError, error }`) of the real `QueryObserverResult`, which TS accepts contravariantly since the real return type is a superset of required fields. Sound. |
| — | Non-memoized `onRefresh`/derived values at call sites | PASS | `useGridRefresh` is a plain function (no internal `useState`/`useEffect`), so it recomputes `isRefreshing`/`onRefresh`/`lastUpdatedAt` every render without violating hook rules; none of the 17 call sites pass `onRefresh` into a `useEffect`/`useMemo` dependency array in this diff (confirmed by reading each page's changed hunk), so the lack of `useCallback` wrapping has no observed correctness impact — it is a micro-perf non-issue, not a bug. |
| — | `TenantsPage.tsx` `tenantsSource` object literal every render | PASS | Same reasoning as above — `useGridRefresh` doesn't memoize its input, so a fresh object each render is harmless; `isRefreshing`/`onRefresh`/`lastUpdatedAt` are recomputed from the same source values (`isRefreshingTenants`, `tenantsUpdatedAt`) each render regardless of object identity. |
| — | `TenantsPage.tsx` `if (loading && !isRefreshingTenants)` guard | PASS | `tenant-context.tsx`'s `refreshTenants()` sets the shared `loading` flag true for the duration of the refetch (line ~112-141, pre-existing), which would otherwise re-trigger `TenantPageSkeleton` on every refresh click. Gating on `!isRefreshingTenants` (set by the new local state at the start of `tenantsSource.refetch`, `TenantsPage.tsx:53`) keeps the populated grid on screen during a refresh instead of skeleton-flashing it away, matching the feature's own goal (refresh in place). Covered by the new test `"shows the grid, not the skeleton, while refreshing"` (`TenantsPage.test.tsx`). |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS | `EmptyState.test.tsx` (new, 20 cases covering refresh button, action precedence, caption), `DataTableWrapper.test.tsx` (new, 8 cases), `useGridRefresh.test.ts` (new, `lastUpdatedAt` derivation + refresh/toast behavior); page-level refresh-from-empty-state coverage added to `TenantsPage.test.tsx`, `EventOccurrencesPage.test.tsx`, `RewardPoolsPage.test.tsx`, `TransportsPage.test.tsx` |
| FE-18 | Mocks updated when services changed | PASS (n/a) | No service-layer interface changed by this diff; `tenant-context.tsx`'s widened `refreshTenants` return type is additive (old callers awaiting `Promise<void>` still compile against `Promise<TenantRefreshResult>`), and the two untouched consumers (`app-tenant-switcher.tsx`, `CreateTenantDialog.tsx`) both just `await refreshTenants()` and discard the result — confirmed via `grep -rl refreshTenants src` |

## Not evaluable from the diff

None. Every checklist item was resolvable from the diff plus targeted lookups (React Query's `UseQueryResult` type declaration, the two untouched `refreshTenants` call sites, and the pre-existing `data-table.tsx`/`lib/utils/toast.ts` conventions the diff builds on).

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None found. `useGridRefresh`'s inputs/outputs are not memoized (`RefreshableQuery[]` array literals, `onRefresh` closure recreated every render), but no call site in this diff relies on referential stability for correctness (none pass these into a dependency array), so this is a style note rather than a defect worth blocking on.
