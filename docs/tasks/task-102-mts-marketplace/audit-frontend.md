# Frontend Audit — task-102-mts-marketplace (MTS marketplace UI)

- **Audit Scope:** atlas-ui TypeScript/React changes on `task-102-mts-marketplace` (`main..HEAD`, BASE `eed47d480`, HEAD `87dfa758a`)
- **Guidelines Source:** frontend-dev-guidelines skill (FE-*) + atlas-ui CLAUDE.md
- **Date:** 2026-06-18
- **Build:** PASS (`npm run build`, exit 0, `built in 1.25s`)
- **Tests:** 813 passed, 1 failed (`Pager.test.tsx` timeout flake — passes 8/8 in isolation; NOT an MTS file, NOT a regression)
- **Lint (MTS files only):** PASS — `eslint` exit 0, zero output across all 8 MTS files (no new lint errors)
- **Overall:** PASS (no FE-* FAIL; one Minor data-contract observation)

## Build & Test Results

- `npm run build` → exit 0. `MarketplacePage-B5GlcUf3.js` chunk emitted; full `tsc -b` (which type-checks `.test.ts` too) clean.
- `npm test` (vitest run) → `Test Files 1 failed | 93 passed (94)`, `Tests 1 failed | 813 passed (814)`. The single failure is `src/components/common/__tests__/Pager.test.tsx > disables First and Prev on page 1` — `Test timed out in 5000ms` under full-suite load. Re-run isolated: `Test Files 1 passed (1) / Tests 8 passed (8)`. Pre-existing flake (Pager is a reused component, untouched by this branch's diff), not an MTS regression. (Note: the task prompt named `TenantsPage.test.tsx` as the known flake; the observed flake this run was `Pager.test.tsx`. Both are load-dependent timeouts, neither is an MTS file.)
- Lint: ran `./node_modules/.bin/eslint` on all 8 MTS source files — exit 0, no diagnostics. No new errors introduced.

## File Inventory

- `src/lib/schemas/mts-config.schema.ts` — Schema (Zod)
- `src/services/api/mts-config.service.ts` — Service
- `src/services/api/mts-listings.service.ts` — Service
- `src/services/api/__tests__/mts-config.service.test.ts` — Test
- `src/services/api/__tests__/mts-listings.service.test.ts` — Test
- `src/lib/hooks/api/useMtsConfig.ts` — Hook (query + mutation)
- `src/lib/hooks/api/useMtsListings.ts` — Hook (query)
- `src/pages/MarketplacePage.tsx` — Page (read-only listings browser)
- `src/pages/TenantsMtsConfigPage.tsx` — Page (wrapper)
- `src/pages/tenants-mts-config-form.tsx` — Component (config form, colocated)
- `src/App.tsx` — Other (route wiring; +2 lazy imports, +2 routes)
- `src/components/app-sidebar.tsx` — Other (nav entry)
- `src/components/features/tenants/TenantDetailLayout.tsx` — Other (tab entry)

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | Grep of all 8 MTS files for `: any`/`as any`/`null as any` → zero matches. Test casts use `ReturnType<typeof vi.fn>`, not `any` (mts-config.service.test.ts:33, mts-listings.service.test.ts:41). |
| FE-02 | No manual class concat | PASS | All `className` are static string literals (e.g. MarketplacePage.tsx:114,242; tenants-mts-config-form.tsx:81). No `+`/template concatenation in `className`. |
| FE-03 | No direct API client in components | PASS | Pages import hooks/services only — MarketplacePage.tsx:3-5 (`useTenantConfiguration`, `useMtsListings`), tenants-mts-config-form.tsx:8 (`useMtsConfig`). No `@/lib/api/client` import in any page/component. Client is imported only in the service layer (mts-config.service.ts:1, mts-listings.service.ts:1) — the documented pattern. |
| FE-04 | No inline Zod in components | PASS | `z.object` lives only in lib/schemas/mts-config.schema.ts:26; form imports it (tenants-mts-config-form.tsx:9). No `z.` in any component. The `.refine()` cross-field rule is in the schema file (mts-config.schema.ts:65), the allowed location. |
| FE-05 | No spinners for content loading | PASS w/ NOTE | Content loading uses text placeholders, not `animate-spin` (MarketplacePage.tsx:235-236). `animate-spin` appears only on the Search submit button (MarketplacePage.tsx:207) — allowed. NOTE: content loading states use plain text divs ("Loading listings...", tenants-mts-config-form.tsx:68) rather than `<Skeleton>`; guideline prefers skeletons but bans spinners — no spinner is used, so this passes the rule as written. |
| FE-06 | No hardcoded colors | PASS | Grep for `bg-(white\|black\|gray-\|...)` / `text-(white\|black\|gray-)` in both pages → zero. Semantic tokens used: `text-muted-foreground` (MarketplacePage.tsx:222,230,236), `text-destructive` (232), `bg-background` (244). |
| FE-07 | No state mutation | PASS | All `setState` use fresh objects/values — `setApplied({...})` (MarketplacePage.tsx:89,105), `setPage(1)` (96). Hook optimistic update spreads immutably (useMtsConfig.ts:60-63). Service merge spreads (mts-config.service.ts:62,67). No `.push/.splice/.sort` into setState. |
| FE-08 | No default component exports | PASS | Named exports: `MarketplacePage` (MarketplacePage.tsx:40), `TenantsMtsConfigPage` (TenantsMtsConfigPage.tsx:4), `MtsConfigForm` (tenants-mts-config-form.tsx:36). Grep `export default` → zero in MTS files. (atlas-ui is Vite/React-Router, not Next; named exports are the convention.) |
| FE-09 | Tenant guard in hooks | PASS | `useMtsConfig` takes explicit `tenantId` and gates `enabled: !!tenantId` (useMtsConfig.ts:39). `useMtsListings` takes an explicit `enabled` flag (useMtsListings.ts:24,30); MarketplacePage passes `!!activeTenant` (MarketplacePage.tsx:74). The listings hook is world-scoped, not tenant-scoped, and tenant context is supplied by `api.setTenant` headers + the `enabled` guard. |
| FE-10 | Tenant ID in query keys | PASS | `mtsConfigKeys.detail(tenantId)` includes tenantId (useMtsConfig.ts:26) — `enabled` only fires with a truthy tenantId so a 'no-tenant' sentinel is unnecessary; this matches the existing `tenantKeys.configDetail(id)` precedent (useTenants.ts:176). `mtsListingsKeys.browse(worldId, filter)` keys on worldId + filter (useMtsListings.ts:14); listings are world-scoped (not tenant-scoped) and the React Query cache is fully cleared on tenant switch via `TenantProvider` (`queryClient.clear()` per atlas-ui CLAUDE.md), so cross-tenant cache bleed is prevented. |
| FE-11 | Error handling | PASS w/ NOTE | Mutation surfaces errors via toast at the call site (tenants-mts-config-form.tsx:63 `toast.error`). Listings errors surfaced in UI via `listingsQuery.error.message` (MarketplacePage.tsx:231-233). NOTE: `useMtsConfig.ts:71` and the mutation logs via `console.error` in `onError` in addition to the toast — the guideline discourages `console.log`-for-errors, but here the user-facing path is the toast and the console line is diagnostic, mirroring the existing `useTenants.ts:147` pattern. No use of `createErrorFromUnknown()`, but no raw `.catch` swallowing either; React Query owns rejection handling. Acceptable. |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | `MtsConfig` = `{ id: string, attributes: MtsConfigAttributes }` (mts-config.service.ts:30-33); `MtsListing` = `{ id, attributes }` (mts-listings.service.ts:59-62). |
| FE-13 | Service pattern | PASS | Both services use the documented thin-adapter-over-`api` pattern (mts-config.service.ts:48-49,66; mts-listings.service.ts:104) rather than `BaseService` — consistent with `tenants.service.ts`. Atlas-ui CLAUDE.md sanctions "thin adapters over lib/api/client". |
| FE-14 | Query key factory `as const` | PASS | `mtsConfigKeys` all/details/detail use `as const` (useMtsConfig.ts:24-26); `mtsListingsKeys` all/browse use `as const` (useMtsListings.ts:13-15). |
| FE-15 | Forms use RHF + zodResolver | PASS | `useForm({ resolver: zodResolver(mtsConfigSchema), ... })` (tenants-mts-config-form.tsx:45-48); fields via `<FormField control={form.control} ...>` (84-105); submit `form.handleSubmit(onSubmit)` (81). |
| FE-16 | Schema + inferred type | PASS | `export type MtsConfigFormData = z.infer<typeof mtsConfigSchema>` paired with the schema (mts-config.schema.ts:74). |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests for changed code | PASS w/ NOTE | Both services tested: `mts-config.service.test.ts` (path, JSON:API envelope on PATCH, schema validity/refine/non-integer), `mts-listings.service.test.ts` (query-builder edge cases, endpoint, return passthrough). NOTE: no test for the hooks (`useMtsConfig`/`useMtsListings`) nor the `MarketplacePage`/`MtsConfigForm` components. Per atlas-ui CLAUDE.md component/hook test coverage is sparse repo-wide (Phase-5 backlog); the schema + service logic — the load-bearing, JSON:API-envelope-correctness surface — is covered. Recommend (non-blocking) adding a `MtsConfigForm` render+submit test and a `MarketplacePage` pager-boundary test. |
| FE-18 | Mocks updated for service changes | PASS | New services mock `@/lib/api/client` inline per test (mts-config.service.test.ts:6-8, mts-listings.service.test.ts:5-7) matching the methods used (`getOne`/`patch`, `getList`). No shared `__mocks__` interface to drift. |

## Verified PASS Highlights (load-bearing)

- **JSON:API write envelope (the #1 risk flagged):** PATCH body is `{data:{id,type:"mts-configs",attributes}}` (mts-config.service.ts:63-66), asserted byte-for-byte in mts-config.service.test.ts:53-67. Bare-body 400 avoided.
- **Listings browse query wiring:** flat (non-bracketed) params, `itemId` dropped when 0, omitted when undefined (mts-listings.service.ts:80-91); asserted in mts-listings.service.test.ts:9-35.
- **Pager with no total metadata:** `hasNextPage = listings.length === LISTINGS_PAGE_SIZE`, `lastPage = hasNextPage ? page+1 : page` (MarketplacePage.tsx:80-81) — a correct next/prev pager given the backend returns no `lastPage`/total. Page reset to 1 on world change (133), filter apply (96), and clear (106).
- **exactOptionalPropertyTypes:** OFF in this project (atlas-ui CLAUDE.md). The optional-undefined filter fields (`mts-listings.service.ts:65-72` use `string | undefined`) and MarketplacePage's `|| undefined` coercion (MarketplacePage.tsx:67-71) build clean under the actual tsconfig. No EOPT violation under the real config.

## Summary

### Blocking (must fix)
- None. Build PASS, lint clean on MTS files, all MTS-relevant tests pass; every FE-* check passes with file:line evidence.

### Non-Blocking (should fix)
- **FE-17 — hook/component test gap:** no tests for `useMtsConfig`/`useMtsListings` hooks or the `MarketplacePage`/`MtsConfigForm` components. Add a form render+submit test and a pager-boundary test. Consistent with repo Phase-5 backlog, so not blocking.
- **FE-05 / loading polish:** content-loading states are plain text divs (MarketplacePage.tsx:235-236, tenants-mts-config-form.tsx:68) rather than `<Skeleton>`. Rule (no spinners) is satisfied; skeletons would be the preferred polish.
- **Data-contract observation (not an FE-* rule):** `MarketplacePage` derives `worldId` from the **array index** of `tenantConfig.attributes.worlds` (MarketplacePage.tsx:141 `value={String(index)}`) and sends it to `/api/worlds/{index}/listings`. The `worlds` config object has no explicit world-id field (tenants.service.ts:111-121), so this assumes the array position equals the atlas-mts world id. Verify that contract against atlas-mts; if worlds can be sparse/reordered, the browse will target the wrong world. Worth confirming before release.
