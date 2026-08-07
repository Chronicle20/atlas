# Frontend Audit — commit 7453a5d33 (Task 20: report types/service/hooks)

- **Audit Scope:** `services/atlas-ui/src/types/models/report.ts`, `services/atlas-ui/src/services/api/reports.service.ts`, `services/atlas-ui/src/lib/hooks/api/useReports.ts` (commit `7453a5d33` only)
- **Guidelines Source:** frontend-dev-guidelines skill
- **Date:** 2026-08-05
- **Build:** PASS
- **Tests:** 1389 passed, 0 failed (192 test files)
- **Overall:** NEEDS-WORK

## Build & Test Results

```
$ npm run build
✓ built in 1.08s
(only pre-existing ConversationEditorPanel >500kB chunk warning, unrelated to this change)

$ npm test
 Test Files  192 passed (192)
      Tests  1389 passed (1389)
   Duration  22.96s

$ npx eslint src/types/models/report.ts src/services/api/reports.service.ts src/lib/hooks/api/useReports.ts
(no output — clean)
```

Backend contract (attribute names, resource type `"reports"`, nullable `chatLog`/`serverTranscript`, `TranscriptLine` shape) was independently pre-verified by the controller against `services/atlas-ban/atlas.com/ban/report/{resource.go,model.go}` and confirmed correct — not re-derived here.

## File Inventory

- `services/atlas-ui/src/types/models/report.ts` — **Type** (JSON:API domain model + const-object enums)
- `services/atlas-ui/src/services/api/reports.service.ts` — **Service** (object-literal service, mirrors `bans.service.ts`'s non-class style)
- `services/atlas-ui/src/lib/hooks/api/useReports.ts` — **Hook** (React Query key factory + query/mutation hooks)

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -n ': any\|as any'` across all three files returns zero matches |
| FE-02 | No manual class concatenation | N/A | No JSX/className in any of the three files (types/service/hooks only) |
| FE-03 | No direct API client calls in components | PASS | `@/lib/api/client` is imported only by `reports.service.ts:1`, which *is* the service layer — not a component |
| FE-04 | No inline Zod schemas in components | PASS | `grep -n 'z\.(object\|string\|number)'` returns zero matches in all three files |
| FE-05 | No spinners for content loading | N/A | No JSX in scope |
| FE-06 | No hardcoded colors | N/A | No JSX/className in scope |
| FE-07 | No state mutation | PASS | `reports.service.ts:12` `reports.sort(...)` mutates the array in place, but it is a fresh array just returned by `api.getList` inside the same function (never external/React state) — identical to the pre-existing `sortBans` at `bans.service.ts:50`, which is the established convention for this codebase |
| FE-08 | No default exports for components | PASS | `grep -n 'export default'` returns zero matches across all three files |
| FE-09 | Tenant guard in hooks | PASS | `useReports.ts:37` `enabled: !!tenant?.id`; `useReports.ts:50` `enabled: !!tenant?.id && !!id` — both take explicit `tenant: Tenant \| null` parameters (`useReports.ts:31`, `:43`) |
| FE-10 | Tenant ID in query keys | PASS | `useReports.ts:24,27` — `tenant?.id ?? "no-tenant"` in both `list()` and `detail()` key builders |
| FE-11 | Error handling with `createErrorFromUnknown` | N/A | No `.catch(` blocks in any of the three files; errors propagate through React Query's own error channel (`UseQueryResult.error` / `UseMutationResult.error`), which is the established pattern for this layer — `bans.service.ts`/`useBans.ts` follow the same no-catch convention at this layer (toast/`createErrorFromUnknown` surfacing belongs to the not-yet-built consuming components, out of this commit's scope per the brief) |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | `report.ts:55-59` — `Report { id: string; type: "reports"; attributes: ReportAttributes }` |
| FE-13 | Service extends `BaseService` (when applicable) | PASS (matches codebase convention) | `reports.service.ts:19` uses a plain object-literal export, not a class. This diverges from the skill doc's `patterns-service-layer.md` template (which shows class-based examples) but is byte-for-byte consistent with the actual reference implementation `bans.service.ts:87` (`export const bansService = {...}`), which is itself a plain object, not a `BaseService` subclass. Since the doc is stale relative to the codebase and the task brief explicitly required mirroring `bans.service.ts`, this is not a new violation. |
| FE-14 | Query key factory uses `as const` | PASS | `useReports.ts:21-28` — every branch of `reportKeys` ends in `as const` |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | N/A | No forms in scope |
| FE-16 | Schema in `lib/schemas/` with inferred type | N/A | No Zod schemas in scope |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | **FAIL** | Zero test files exist for any of the three new files: no `services/atlas-ui/src/services/api/__tests__/reports.service.test.ts`, no `.../types/models/__tests__/report.test.ts`, no `.../lib/hooks/api/__tests__/useReports.test.tsx`. The established project convention for this exact layer is `services/atlas-ui/src/services/api/__tests__/bans.service.test.ts`, which unit-tests `bansService`'s pure logic (URL/query-param construction, sort-by-id comparator, type filter) by mocking `@/lib/api/client`. `reports.service.ts` has directly analogous testable logic — `sortReports`'s createdAt-descending comparator (`reports.service.ts:11-17`) and the `?status=` query-param branch in `getAllReports` (`reports.service.ts:20-27`) — none of which is covered. The implementer's own report (`task-20-report.md` → "Verification") states this explicitly: "no test files were added for these three data-layer files themselves (none were requested by the brief)." Hooks (`useReports.ts`) have no direct precedent requiring a dedicated test file — `lib/hooks/api/__tests__/` has no `useBans.test.tsx` either, so hook coverage is expected to arrive indirectly via the page/component tests in Tasks 21-22, which is consistent. The gap is specifically the missing `reports.service.test.ts`. |
| FE-18 | Mocks updated when services changed | N/A | No pre-existing `reports` mocks to update; no `__mocks__/` entries reference this service yet |

## Consistency with Reference Implementation (`bans.service.ts` / `useBans.ts`)

Compared structurally against `services/atlas-ui/src/services/api/bans.service.ts` and `services/atlas-ui/src/lib/hooks/api/useBans.ts`:

- **Matches:** object-literal service export style; no per-call `api.setTenant()` (both rely on `TenantProvider`'s single `api.setTenant(activeTenant)` call, per `services/atlas-ui/CLAUDE.md` → "Tenant contract"); explicit `tenant: Tenant | null` parameter on query hooks (Pattern A); `enabled: !!tenant?.id` guards; `tenant?.id ?? "no-tenant"` key fallback; `gcTime: 5 * 60 * 1000` on both query hooks; mutation hooks invalidate `<resource>Keys.all` on `onSuccess`; `use<Resource>QueryOptions extends QueryOptions` shape (`ReportQueryOptions`/`BanQueryOptions`); `api.patch<T>(url, data, options)` argument order matches `client.ts:389-393`.
- **Backend-driven, non-gratuitous divergence:** `reports.service.ts` uses plain `api.getList`/`api.getOne` (no `fetchAll`/`fetchPaged` pagination draining). Verified this is correct, not an oversight: `services/atlas-ban/atlas.com/ban/report/rest.go:33-62`'s `handleGetReports` returns the full list via `server.MarshalResponse[[]RestModel]` with no `meta`/pagination envelope — the `/reports` endpoint isn't paginated server-side, unlike `/bans` (which needed the task-117 `fetchAll`/`fetchPaged` refactor). `sortReports` sorts by `createdAt` descending rather than bans' sort-by-numeric-id — also correct, since `Report.id` is a UUID (`resource.go`), not a monotonic integer, so numeric-id sort (as bans does) would be meaningless here.
- **Minor, non-blocking divergence:** `useInvalidateReports` (`useReports.ts:70-76`) exposes only `invalidateAll`, whereas the reference `useInvalidateBans` (`useBans.ts:135-143`) exposes both `invalidateAll` and `invalidateLists`. Not a guideline violation (no rule mandates the `invalidateLists` branch), but a gratuitous structural gap that a later task (component work) may need to add back if a list-only invalidation ever becomes necessary. Non-blocking.

## Cache Invalidation Analysis (Focus Area 3)

`useUpdateReportStatus` (`useReports.ts:55-68`) invalidates `queryClient.invalidateQueries({ queryKey: reportKeys.all })` on success (`useReports.ts:65`), where `reportKeys.all = ["reports"] as const` (`useReports.ts:21`).

- Because React Query's `invalidateQueries` defaults to a **prefix match** (not `exact: true`), invalidating `["reports"]` invalidates every key that starts with `"reports"` — both `reportKeys.lists()` (`["reports","list",...]`) and `reportKeys.details()` (`["reports","detail",...]`), across all tenants and all filter/id combinations.
- This exactly matches the reference behavior: `bans.service.ts`'s mutation hooks (`useCreateBan`, `useDeleteBan`, `useExpireBan` at `useBans.ts:99-133`) all invalidate `banKeys.all` the same way, not a narrower `banKeys.detail(tenant, id)`.
- Under-broad risk (stale detail view after a status change) is avoided since `.all` covers details too.
- Over-broad risk (unrelated tenants refetching) is bounded — `invalidateQueries` still only *marks stale + refetches active observers*; queries for tenants with no mounted observer aren't network-fetched, just marked stale, which is the accepted tradeoff this codebase already makes for every other resource's mutations.
- Verdict: correct, and consistent with the reference. Not a finding.

## Summary

### Blocking (must fix)
- **FE-17** — `reports.service.ts` has no unit test, despite the established convention (`bans.service.test.ts`) of testing this exact layer's pure logic (URL building, sort comparator). Add `services/atlas-ui/src/services/api/__tests__/reports.service.test.ts` covering: `sortReports` ordering, the `?status=` query-param branch vs. no-filter branch in `getAllReports`, and the JSON:API PATCH envelope shape in `updateReportStatus`.

### Non-Blocking (should fix)
- `useInvalidateReports` only exposes `invalidateAll`; consider adding `invalidateLists`/`invalidateReport(tenant, id)` to match `useInvalidateBans`'s shape, if/when a future task needs narrower invalidation.
