# Frontend Audit — task-102-mts-marketplace (whole branch)

- **Audit Scope:** 13 changed TS/TSX files, `6c6f52ab..e3696ccf`, `services/atlas-ui/src`
- **Guidelines Source:** frontend-dev-guidelines skill (FE-* checklist)
- **Date:** 2026-07-10
- **Build:** PASS (`tsc -b && vite build` — "✓ built")
- **Tests:** 879 passed, 0 failed (102 files; MTS-specific: 12 passed)
- **Overall:** NEEDS-WORK (build + tests green; 1 Critical + 2 Important FE violations)

Note: the objective gate was run with the nvm Linux node (`v22.22.2`); the default `/mnt/c` Windows npm in this WSL env errors `ERR_INVALID_URL` and cannot run the scripts.

## File Inventory

- `src/App.tsx` — Other (route wiring: `/marketplace`, `/tenants/:id/mts-config`)
- `src/components/app-sidebar.tsx` — Component (nav entry)
- `src/components/features/tenants/TenantDetailLayout.tsx` — Component (nav item)
- `src/lib/hooks/api/useMtsConfig.ts` — Hook
- `src/lib/hooks/api/useMtsListings.ts` — Hook
- `src/lib/schemas/mts-config.schema.ts` — Schema
- `src/pages/MarketplacePage.tsx` — Page
- `src/pages/TenantsMtsConfigPage.tsx` — Page (wrapper)
- `src/pages/tenants-mts-config-form.tsx` — Page/Feature form
- `src/services/api/mts-config.service.ts` — Service
- `src/services/api/mts-listings.service.ts` — Service
- `src/services/api/__tests__/mts-config.service.test.ts` — Test
- `src/services/api/__tests__/mts-listings.service.test.ts` — Test

(This app is Vite + react-router-dom under `src/`, not Next.js `app/`; pages use named exports wired via `lazy()` in App.tsx. FE-08's Next.js `page.tsx` default-export exception does not apply — named exports are correct here.)

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | grep of all 13 files: zero `: any` / `as any` / `<any>` |
| FE-02 | No manual class concat | PASS | No conditional classNames; no `+`/template concat in `className=` |
| FE-03 | No direct API client in components | PASS | MarketplacePage/form import from `@/services/api/*` + hooks; only the two service files import `@/lib/api/client` |
| FE-04 | No inline Zod in components | PASS | Schema lives in `lib/schemas/mts-config.schema.ts`; the form imports it |
| FE-05 | No spinners for content loading | FAIL (Minor) | Content loading uses plain text, not Skeleton: MarketplacePage.tsx:238-239, tenants-mts-config-form.tsx:67-68. `animate-spin` (MarketplacePage.tsx:210) is on the Search submit button — allowed |
| FE-06 | No hardcoded colors | PASS | Only semantic tokens (`bg-background`, `text-muted-foreground`, `text-destructive`); grep for `bg-white`/`gray-N`/etc. is empty |
| FE-07 | No state mutation | PASS | Optimistic update spreads immutably (useMtsConfig.ts:60-64); MarketplacePage uses `setState` with fresh objects |
| FE-08 | No default exports for components | PASS | All new pages/components use named exports; grep `export default` empty |
| FE-09 | Tenant guard in hooks | PASS | useMtsConfig `enabled: !!tenantId` (useMtsConfig.ts:39); useMtsListings gated by caller `enabled=!!activeTenant` (MarketplacePage.tsx:77) |
| FE-10 | Tenant ID in query keys | FAIL (Critical) | `mtsListingsKeys.browse` omits tenant id (useMtsListings.ts:12-16) — see Critical #1. `mtsConfigKeys.detail(tenantId)` includes it (useMtsConfig.ts:26) — PASS |
| FE-11 | Error handling via `createErrorFromUnknown` | FAIL (Important) | Not used anywhere in the change; useMtsConfig.ts:71 uses `console.error`; form onError toasts a static string (tenants-mts-config-form.tsx:63) — see Important #2 |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | `MtsConfig`/`MtsListing` are `{id, attributes}` (mts-config.service.ts:30-33, mts-listings.service.ts:65-68). Types live in service files rather than `types/models/` — allowed under the per-service re-export convention (Minor) |
| FE-13 | Service extends BaseService (when applicable) | PASS | Both use the documented direct-client pattern (plain singleton object over `api`/`apiClient`); no validation/transform needs. Stylistic note: other direct services use `class X {}` + `new X()`; these are object literals (Minor) |
| FE-14 | Query key factory uses `as const` | PASS | mtsConfigKeys (useMtsConfig.ts:23-27) and mtsListingsKeys (useMtsListings.ts:12-16) both `as const` |
| FE-15 | Forms use react-hook-form + zodResolver | PASS | tenants-mts-config-form.tsx:45-48 `useForm({ resolver: zodResolver(mtsConfigSchema) })` |
| FE-16 | Schema in lib/schemas with inferred type | PASS | mts-config.schema.ts:74 `export type MtsConfigFormData = z.infer<typeof mtsConfigSchema>`; cross-field `.refine()` correctly co-located in schema |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | FAIL (Important) | Service + schema tests present (mts-config.service.test.ts, mts-listings.service.test.ts, 12 tests). NO component/page test for MarketplacePage.tsx or the MtsConfigForm — see Important #3. Repo convention has page tests (ItemsPage/TenantsPage/JobDetailPage.test.tsx) |
| FE-18 | Mocks updated when services changed | PASS (N/A) | New services; tests mock `@/lib/api/client` directly (mts-config.service.test.ts:6-8, mts-listings.service.test.ts:5-8). No shared `__mocks__/` interface to update |

## Summary

### Critical (must fix)

- **[FE-10] `mtsListingsKeys.browse` has no tenant id → cross-tenant cache exposure.**
  `useMtsListings.ts:12-16` keys the query `["mts-listings","browse", worldId, filter]`. Listings are tenant-scoped only via the mutable `apiClient.tenant` header (set globally by tenant-context.tsx:48,52); the tenant is absent from both the URL (`/api/worlds/{worldId}/listings`) and the query key.
  Failure scenario: an admin browses tenant A's world 0, then switches to tenant B. `MarketplacePage` stays mounted, so `worldId` and the applied filter persist and the query key is unchanged; the query sets no `staleTime`, so it inherits the 5-minute default. React Query serves tenant A's cached listings under tenant B for up to 5 minutes with **no refetch triggered by the switch** — one tenant's marketplace data shows under another. This is anti-patterns.md #3 verbatim. Fix: add `activeTenant?.id ?? 'no-tenant'` to the browse key (and thread tenant into the hook).

### Important (should fix)

- **[FE-11] No `createErrorFromUnknown`; mutation error goes to `console.error` + a static toast.**
  `useMtsConfig.ts:71` does `console.error("Failed to update MTS configuration:", error)` (anti-patterns.md #10 — console for errors), and the form's `onError` (`tenants-mts-config-form.tsx:63`) toasts the fixed string `"Failed to update MTS configuration"`, discarding the real detail. Failure scenario: the backend rejects a save (e.g. 400/409 validation, stale id) and the admin sees a generic message with no actionable reason, while the real detail is buried in the console. Fix: surface `createErrorFromUnknown(err, ...).message` in the toast.

- **[FE-17] No component/page tests for MarketplacePage or MtsConfigForm.**
  Only service/schema tests exist. Both components are non-trivial: MarketplacePage owns a pending-vs-applied filter state machine, 1-based↔0-based page conversion (`page - 1`, MarketplacePage.tsx:73), world-index mapping and meta-driven pagination; MtsConfigForm owns `form.reset` on load and the empty-string number coercion. Failure scenario: a regression in the page-offset conversion or the filter-apply logic ships untested and silently returns the wrong page/rows. Repo convention (ItemsPage/TenantsPage/JobDetailPage tests) expects page coverage.

### Non-Blocking (Minor)

- **[FE-05]** Content loading uses text, not Skeleton: MarketplacePage.tsx:238-239, tenants-mts-config-form.tsx:67-68. The config form matches the sibling tenant-config forms (properties/writers/worlds/handlers all use text loading), so it is convention-consistent; the guideline still prefers Skeleton.
- **Number input type smell:** tenants-mts-config-form.tsx:96-97 writes `""` (string) into a `number`-typed RHF field on clear. Works (Zod rejects on submit) but is loose under strict typing.
- **Stale doc comment:** mts-listings.service.ts:82 says `saleType` maps to `BUY_NOW / AUCTION`; the backend enum and the UI values are lowercase `"fixed"`/`"auction"` (verified `services/atlas-mts/atlas.com/mts/listing/model.go:14-15`; MarketplacePage.tsx:162-164 emits `"fixed"`/`"auction"`). Values are correct — comment only is wrong.
- **[FE-12]** JSON:API types live in the service files, not `types/models/` — permitted by the per-service re-export convention.
- **Array index as worldId / React key:** MarketplacePage.tsx:143-144. The `worlds` config has no id field (tenants.service.ts:111-121), so index-as-worldId is the de-facto convention; acceptable.

## Final resolution (post-review fixes)

Applied on the whole-branch finalization pass:

- **[FE-10] Critical — FIXED.** `mtsListingsKeys.browse` now takes `tenantId` as the first key segment and `useMtsListings(tenantId, worldId, filter, enabled)`; `MarketplacePage` passes `activeTenant?.id ?? ""`. Switching tenants now yields a distinct cache entry and refetch (mirrors the `guildKeys` tenant-first pattern). `useMtsListings.ts`, `MarketplacePage.tsx`.
- **[FE-11] Important — FIXED.** The config-save `onError` now surfaces the real backend detail via `createErrorFromUnknown(error, "Failed to update MTS configuration").message` instead of a static string. `tenants-mts-config-form.tsx`.
- **[FE-17] Important — FIXED.** Added `MarketplacePage.test.tsx` (4 tests: default browse, 1-based→0-based page conversion, tenant-gating empty state, filter apply + pager advance) and `tenants-mts-config-form.test.tsx` (3 tests: hydration, empty state, save-submits-tenant-id). 13/13 pass (incl. the pre-existing service test); `npm run build` clean.
- **Stale doc comment — FIXED.** `mts-listings.service.ts` `saleType` comment now reads `"fixed" / "auction"`.
- **FE-05 (text-not-Skeleton loading), number-input `""` coercion, FE-12 (types in service files), index-as-worldId — DEFERRED (Minor, convention-consistent).** Left as-is; they match sibling tenant-config forms and the per-service re-export convention.
