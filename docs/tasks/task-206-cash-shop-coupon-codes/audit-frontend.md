# Frontend Audit — task-206-cash-shop-coupon-codes

- **Audit Scope:** New atlas-ui coupon-codes feature (branch `task-206-cash-shop-coupon-codes` vs merge-base `1e0a321b8`): `src/services/api/coupons.service.ts`, `src/lib/hooks/api/useCoupons.ts`, `src/pages/CouponsPage.tsx`, `coupons-columns.tsx`, `CouponDetailPage.tsx`, `src/components/features/coupons/**`, `src/lib/schemas/coupons.schema.ts`, `src/lib/utils/coupons.ts`, `src/lib/utils/coupon-dates.ts`, `src/components/common/skeletons/CouponPageSkeleton.tsx`, `src/App.tsx`, `src/components/app-sidebar-items.ts`, `src/services/api/index.ts`, plus each area's `__tests__`.
- **Guidelines Source:** frontend-dev-guidelines skill (anti-patterns, architecture, react-query, forms-validation, service-layer, types, multitenancy, styling, components, testing), corrected against this repo's actual Vite/React-Router layout per the auditor brief (not the stale Next.js `app/*/page.tsx` table).
- **Date:** 2026-08-09
- **Build:** PASS
- **Tests:** 2001 passed, 2 failed (245 of 246 files passed) — the 2 failures are in `src/pages/__tests__/TenantsPage.test.tsx`, pre-existing and unrelated to this branch (TenantsPage.tsx/test not touched by this diff; last modified by commit `e0321f319`, task-171, long before this branch).
- **Overall:** PASS

## Build & Test Results

`npm run build` (node v22.22.2 via nvm): `tsc -b && vite build` completed clean, `✓ built in 1.88s`, only a non-blocking chunk-size advisory (`ConversationEditorPanel` > 500kB, pre-existing).

`npm test` (vitest run): `Test Files 1 failed | 245 passed (246)`, `Tests 2 failed | 2001 passed (2003)`. The failing file is `src/pages/__tests__/TenantsPage.test.tsx` (`toHaveBeenCalledWith` assertion on `updateTenantMock`), which is outside this branch's changed-file set and was last touched by an unrelated commit predating this task. No coupon test failed.

## File Inventory

- **Service:** `src/services/api/coupons.service.ts` — CRUD + batch-generate + redemption reads over `/api/coupons`, `/api/coupon-batches`, `/api/coupon-redemptions`.
- **Hook:** `src/lib/hooks/api/useCoupons.ts` — query key factory + 5 query hooks + 4 mutation hooks.
- **Page:** `src/pages/CouponsPage.tsx` — list/filter/status-toggle/delete container.
- **Page:** `src/pages/CouponDetailPage.tsx` — single-coupon detail + redemption history.
- **Other (columns):** `src/pages/coupons-columns.tsx` — `DataTableColumnDef` factory (presentational, no data fetching).
- **Component (feature):** `src/components/features/coupons/CreateCouponDialog.tsx`, `GenerateCouponBatchDialog.tsx`, `RewardRowsField.tsx`, `reward-errors.ts`.
- **Schema:** `src/lib/schemas/coupons.schema.ts` — `couponFormSchema`, `couponBatchFormSchema`, `couponRewardSchema`, `rewardRowSchema`.
- **Other (utils):** `src/lib/utils/coupons.ts` (formatting + CSV export), `src/lib/utils/coupon-dates.ts` (local↔ISO datetime conversion).
- **Other (skeleton):** `src/components/common/skeletons/CouponPageSkeleton.tsx`.
- **Other (routing/nav):** `src/App.tsx` (+13 lines: lazy imports + 2 routes), `src/components/app-sidebar-items.ts` (+1 sidebar entry).
- **Other (barrel):** `src/services/api/index.ts` (+22 lines: coupon re-exports).
- **Tests:** `useCoupons.test.tsx`, `CouponsPage.test.tsx`, `CouponDetailPage.test.tsx`, `coupons.service.test.ts`.

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -rn ': any\|as any' services/api/coupons.service.ts lib/hooks/api/useCoupons.ts lib/schemas/coupons.schema.ts lib/utils/coupons.ts lib/utils/coupon-dates.ts pages/CouponsPage.tsx pages/CouponDetailPage.tsx pages/coupons-columns.tsx components/features/coupons/*` — zero matches. |
| FE-02 | No manual class concatenation | PASS | `cn()` used where conditional classes appear, e.g. `components/features/coupons/RewardRowsField.tsx:149` `className={cn("text-sm", "text-destructive")}`. No `className={"..." + ...}` pattern anywhere in scope. |
| FE-03 | No direct API client calls in components | PASS | `grep -rn 'from "@/lib/api/client"' pages/Coupon* components/features/coupons` — zero matches; all reads/writes go through `couponsService` via `useCoupons.ts` hooks. |
| FE-04 | No inline Zod schemas in components | PASS | `grep -rn 'z\.object(\|z\.string(\|z\.number(' pages/Coupon* components/features/coupons` — zero matches. All schemas live in `lib/schemas/coupons.schema.ts:71-192`. |
| FE-05 | No spinners for content loading | PASS | List/detail loading states use `CouponPageSkeleton` (`pages/CouponsPage.tsx:166-168`, `pages/CouponDetailPage.tsx:52`) and `TableSkeleton` (`pages/CouponDetailPage.tsx:125`). `grep -rln animate-spin` over the coupon feature files returns zero; submit buttons use plain `disabled={...isPending}` (`CreateCouponDialog.tsx:182`, `GenerateCouponBatchDialog.tsx:229`) with no spinner icon at all — stricter than the guideline, not a violation. |
| FE-06 | No hardcoded colors | PASS | `grep -rnE 'bg-(white|black|gray|red|blue|green)-[0-9]'` over the coupon feature files — zero matches. Only semantic classes (`text-destructive`, `text-muted-foreground`, `text-primary`) used, e.g. `pages/coupons-columns.tsx:42`. |
| FE-07 | No state mutation | PASS | `RewardRowsField.tsx:42-46` `patch()` uses `rows.map(...)` returning a new array with spread `{ ...row, ...changes }`; add/remove use `[...rows, emptyRewardRow()]` (`:56`) and `rows.filter(...)` (`:93`). No `.push`/`.splice`/`.sort` found in scope. |
| FE-08 | No default exports for components | PASS | `grep -rn 'export default' pages/Coupon* components/features/coupons lib/hooks/api/useCoupons.ts lib/schemas/coupons.schema.ts services/api/coupons.service.ts` — zero matches; every page/component/hook is a named export. |
| FE-09 | Tenant guard in hooks | PASS | All 5 query hooks in `lib/hooks/api/useCoupons.ts` call `useTenant()` and set `enabled: !!activeTenant` (`:72`, `:86` ANDed with `!!id`, `:104`, `:118`, `:132` ANDed with `!!id`). Regression-tested at `lib/hooks/api/__tests__/useCoupons.test.tsx:57-90` (fetch does not fire with `activeTenant === null`). |
| FE-10 | Tenant ID in query keys | PASS (by established sibling precedent, not the literal guideline text) | `couponKeys` (`lib/hooks/api/useCoupons.ts:38-60`) does not embed `tenant?.id` in any key. This diverges from the anti-patterns doc's literal example but exactly matches this codebase's actual convention for `useTenant()`-pattern hooks — `rewardPoolKeys` in `lib/hooks/api/useRewardPools.ts:22-30` likewise carries no tenant id — because `TenantProvider` calls `queryClient.clear()` on every tenant switch (per `services/atlas-ui/CLAUDE.md` "Tenant contract" section), which already prevents cross-tenant cache bleed for the whole app. This was already reviewed and accepted in a prior fix round (see `.superpowers/sdd/plan/progress.md`, Task 26 note) and is not re-litigated here as a fresh finding. |
| FE-11 | Error handling with `createErrorFromUnknown` | PASS | Every mutation catch in scope calls `createErrorFromUnknown` and surfaces via `toast.error`: `pages/CouponsPage.tsx:129` (toggle), `:156` (delete, non-conflict path), `components/features/coupons/CreateCouponDialog.tsx:86`, `components/features/coupons/GenerateCouponBatchDialog.tsx:102`. The delete path additionally special-cases `CouponConflictError` into an inline `<Alert>` (`CouponsPage.tsx:150-155`) rather than a toast — a deliberate richer UX for a 409 race, not a missed catch. |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | `Coupon`, `CouponBatch`, `CouponRedemption` all follow `{ id: string; attributes: {...} }` (`services/api/coupons.service.ts:64-67`, `:84-87`, `:99-102`). |
| FE-13 | Service uses documented pattern | PASS | `couponsService` (`services/api/coupons.service.ts:209-316`) follows the "direct API client pattern" documented for simple resources — plain object with methods calling `api.getOne/post/patch/delete` plus the shared `fetchPaged` helper, matching `accounts.service.ts`'s sibling style referenced in the brief. No unjustified deviation from `BaseService`. |
| FE-14 | Query key factory uses `as const` | PASS | Every branch of `couponKeys` (`lib/hooks/api/useCoupons.ts:38-60`) ends in `as const`. |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | PASS | `CreateCouponDialog.tsx:56-59` and `GenerateCouponBatchDialog.tsx:65-68` both call `useForm({ resolver: zodResolver(...) , defaultValues: ... })`. |
| FE-16 | Schema in `lib/schemas/` with inferred type | PASS | `lib/schemas/coupons.schema.ts` defines `couponFormSchema`/`couponBatchFormSchema`/`couponRewardSchema`/`rewardRowSchema`, each paired with an inferred type (`CouponFormInput`/`CouponFormOutput` at `:150-151`, `CouponBatchFormInput`/`CouponBatchFormOutput` at `:179-180`). The three deviations from the "obvious" schema shape (`.transform().pipe()` instead of `z.coerce.number()`, `currency` as refined `z.number()` instead of a literal union, the `-1` blank-input sentinel) are documented in-file (`:43-59`, `:61-70`) and were verified in a prior review round not to weaken the discriminated union's exclusivity; re-inspected here and still true — `couponRewardSchema` (`:71-84`) is a real `z.discriminatedUnion`, and `positiveInt` (`:55-59`) rejects `n <= 0`, so `-1` can never validate. |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS | `lib/hooks/api/__tests__/useCoupons.test.tsx`, `pages/__tests__/CouponsPage.test.tsx`, `pages/__tests__/CouponDetailPage.test.tsx`, `services/api/__tests__/coupons.service.test.ts` cover the hook layer, both pages, and the service layer. `CreateCouponDialog` and `GenerateCouponBatchDialog` have no standalone test files, but both are exercised through `CouponsPage.test.tsx` (`pages/__tests__/CouponsPage.test.tsx:157-232` opens each dialog, drives the reward-row fields, and asserts on the resulting mutation calls), which covers their behavior including the reward-row validation path (`:157-179`). `RewardRowsField.tsx` and `reward-errors.ts` have no direct unit test but are transitively exercised the same way. Not a blocking gap. |
| FE-18 | Mocks updated when services changed | PASS | This is a new service with no pre-existing mock to go stale; `services/api/__tests__/coupons.service.test.ts` mocks `@/lib/api/client` directly and asserts the exact JSON:API envelope for create (`:58-64`), update (`:83-86`), and null-clearing update (`:114-117`); `useCoupons.test.tsx:19-25` and `CouponsPage.test.tsx` mock `couponsService` per-test with the methods each test needs. |

## Multi-Tenancy Deep-Dive (per task emphasis)

- No tenant-scoped call escapes the gate: all 5 read hooks in `useCoupons.ts` are gated on `!!activeTenant` (see FE-09); the 4 mutation hooks are intentionally ungated (user-triggered post-mount, not auto-firing on mount), matching the sibling convention noted in `.superpowers/sdd/plan/progress.md` Task 26.
- **Active-toggle PATCH hazard:** `pages/CouponsPage.tsx:118-133` `handleToggleActive` sends `patch: { active: next }` and nothing else (`:122-124`). Verified directly in source (not taken on faith from `progress.md`) — the object literal has exactly one key. Regression-tested at `pages/__tests__/CouponsPage.test.tsx:281-309`: `expect(call[1]).toEqual({ active: false })` and `expect(Object.keys(call[1])).toEqual(["active"])`.
- **Other mutation call sites checked for the same hazard:**
  - `useCreateCoupon` → `couponsService.create` is a POST, not a PATCH; `CreateCouponDialog.tsx:67-79` builds `input` field-by-field from the full form (not a spread of an existing object), so there is no partial-overwrite risk — POST always carries a complete, deliberately-composed create body.
  - `useGenerateCouponBatch` → also POST, same reasoning; `GenerateCouponBatchDialog.tsx:86-96` builds `input` the same field-by-field way.
  - `useDeleteCoupon` → DELETE, no body.
  - Only one call site in the entire branch invokes `useUpdateCoupon`/`couponsService.update` — the toggle above (`grep -rn "useUpdateCoupon\|couponsService.update"` returns exactly `useCoupons.ts:151/159` and `CouponsPage.tsx:35/85/121`). There is no separate "edit coupon" dialog that could accidentally full-object-PATCH; if one is added later, it must reuse the same one-field-at-a-time discipline as `handleToggleActive`.
- `UpdateCouponInput` (`services/api/coupons.service.ts:141-148`) documents the server's exact partial-PATCH semantics (omitted key preserves, explicit `null` clears) in a comment directly above the type, matching the task's stated hazard description verbatim.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- `lib/utils/coupon-dates.ts:14` `isoToLocalInput` is exported but has zero call sites anywhere in `src` outside its own definition (`grep -rn "isoToLocalInput" src` matches only the declaration) — dead code, presumably scaffolded for a future "edit coupon" pre-fill dialog that doesn't exist yet in this branch. Harmless, but worth removing or wiring up.
- `RewardRowsField.tsx` and `reward-errors.ts` have no dedicated unit test file; coverage is currently indirect via `CouponsPage.test.tsx`'s dialog interactions. Sufficient for FE-17 but a direct unit test would pin `firstMessage`'s depth-first-search behavior independent of any one dialog's RHF wiring.
