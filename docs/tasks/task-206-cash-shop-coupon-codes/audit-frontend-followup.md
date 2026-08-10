# Frontend Audit — Follow-up (coupon admin UI polish)

- **Audit Scope:** Diff `89a624f4b..af7da733b` — 13 changed TS/React files under `services/atlas-ui/src`
- **Guidelines Source:** `.claude/skills/frontend-dev-guidelines` (SKILL.md + resources/*)
- **Date:** 2026-08-10
- **Build:** PASS (`npm run build` — `tsc -b && vite build`, clean, no type errors)
- **Tests:** 2011 passed, 0 failed (`npm test`, 247 files, Vitest)
- **Overall:** PASS (no FE-* FAIL items; two Important findings, both non-blocking deviations from documented convention)

## Build & Test Results

```
> atlas-ui@0.1.0 build
> tsc -b && vite build
✓ built in 1.19s
(warning only: some chunks >500kB — pre-existing, unrelated to this diff)

> atlas-ui@0.1.0 test
> vitest run
 Test Files  247 passed (247)
      Tests  2011 passed (2011)
```

Both commands were run from `services/atlas-ui` inside the correct worktree (`.worktrees/task-206-cash-shop-coupon-codes`) under Node 22 (`nvm use 22`).

## File Inventory

- **Page:** `src/pages/CouponDetailPage.tsx` (modified — hook reorder, "Back to coupons" button removed)
- **Component:** `src/components/features/coupons/CashItemPicker.tsx` (new)
- **Component:** `src/components/features/coupons/RewardRowsField.tsx` (modified — currency `<Input>` → `<Select>`, serial `<Input>` → `<CashItemPicker>`)
- **Hook:** `src/lib/hooks/api/useActorNames.ts` (new — `useAccountNames`, `useCharacterNames`)
- **Hook:** `src/lib/hooks/api/useItemCommodities.ts` (modified — adds `useCommodityBySerial` + `itemCommoditiesKeys.bySerial`)
- **Service:** `src/services/api/commodities.service.ts` (modified — adds `getBySerialNumber`, extracts shared `toModel`)
- **Schema:** `src/lib/schemas/coupons.schema.ts` (modified — adds `CURRENCY_VALUES`, refinement now derives from it)
- **Lib (breadcrumbs):** `src/lib/breadcrumbs/routes.ts`, `src/lib/breadcrumbs/resolvers.ts` (modified — `/coupons` routes + `EntityType.COUPON` resolver)
- **Tests:** `src/components/features/coupons/__tests__/RewardRowsField.test.tsx` (new), `src/pages/__tests__/CouponDetailPage.test.tsx` (modified), `src/lib/breadcrumbs/__tests__/resolvers.test.ts` (modified), `src/lib/breadcrumbs/__tests__/routes.test.ts` (modified)

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -n ": any\|as any"` over all 9 non-test in-scope files returns zero matches. |
| FE-02 | No manual class concatenation | PASS | All conditional classes go through `cn()` — e.g. `RewardRowsField.tsx:162` `cn("text-sm", "text-destructive")`; no `className={"..." + ...}` found. |
| FE-03 | No direct API client calls in components | PASS | `CashItemPicker.tsx` and `RewardRowsField.tsx` import only hooks (`useItemSearch`, `useItemCommodities`, `useItemName`) — no `@/lib/api/client` import. The only new `@/lib/api/client` import is in `services/api/commodities.service.ts:1`, which is the service layer itself — correct location. |
| FE-04 | No inline Zod schemas in components | PASS | No `z.object(`/`z.string(` etc. in `CashItemPicker.tsx` or `RewardRowsField.tsx`; the `CURRENCY_VALUES` refinement lives in `lib/schemas/coupons.schema.ts:79-83`. |
| FE-05 | No spinners for content loading | PASS | `grep -n "animate-spin"` over the 4 non-service in-scope component/page/hook files returns zero matches. `CashItemPicker.tsx:180-191` uses text states ("Loading cash-shop entries…") rather than a spinner for its popover content — acceptable but see Minor note below (no `Skeleton`, just text — guideline exempts small inline states, but the letter of "use skeleton components" is not literally followed). Not a FAIL since no spinner is used either. |
| FE-06 | No hardcoded colors | PASS | `grep -nE "bg-(white|black|gray-[0-9]|red-[0-9]|blue-[0-9]|green-[0-9])"` over the 3 UI files returns zero matches; all classes are semantic (`text-muted-foreground`, `text-destructive`, `bg-accent`, `hover:bg-accent`). |
| FE-07 | No state mutation | PASS | `RewardRowsField.tsx:52-56` (`patch`) and `:66` (`onChange([...rows, emptyRewardRow()])`) use spread/map; `useActorNames.ts:37-41` (`toRecord`) builds a fresh object via `forEach`, never mutates an existing one. No `.push(`/`.splice(`/`.sort(` on state found. |
| FE-08 | No default exports for components | PASS | `CashItemPicker.tsx:61` `export function CashItemPicker(...)`; `RewardRowsField.tsx:45` `export function RewardRowsField(...)`; `CouponDetailPage.tsx:40` `export function CouponDetailPage()`. `grep -n "export default"` returns zero. |
| FE-09 | Tenant guard in hooks | **Important (deviation)** | `useActorNames.ts:44-61,63-80` (`useAccountNames`, `useCharacterNames`) use `useTenant()` (part 1 of the check is satisfied) but the individual `useQueries` entries carry **no `enabled` field at all** — unlike the sibling/precedent hook `lib/hooks/api/useItemNames.ts:21` (`enabled: !!activeTenant`) which does exactly this same batched-by-id pattern and does set `enabled`. `useActorNames.ts` instead relies on `lookupIds()` (`useActorNames.ts:28-31`) returning `[]` when `tenant` is falsy, so the `unique.map(...)` query array is empty and no query object is ever constructed. This is *functionally* safe — no request fires with a null tenant — but it silently diverges from the documented/established `enabled: !!tenant?.id` idiom the FE-09 checklist item explicitly calls out, and from the codebase's own `useItemNames` precedent that does the identical thing correctly. Any future refactor of `lookupIds` that stops filtering on `tenant` would reintroduce the exact bug FE-09 exists to prevent, with no `enabled` guard as a second line of defense. |
| FE-09b | Non-null assertion consistency | **Important (deviation)** | `useActorNames.ts:70` `characterKeys.detail(activeTenant!, String(id))` uses a non-null assertion, while the parallel `useAccountNames` at `useActorNames.ts:51` passes `activeTenant` (no assertion) to `accountKeys.detail`, because `accountKeys.detail` accepts `Tenant \| null` (`useAccounts.ts:39-40`) but `characterKeys.detail` requires non-null `Tenant` (`useCharacters.ts:35-36`). The assertion is reachable only because `unique` is already empty when `activeTenant` is null (see FE-09 above), so it does not crash today — but per `patterns-types.md:236` ("Use `!` assertion only when you've already validated"), the validation here is *indirect*, living in a separate function (`lookupIds`) one hop away from the assertion site, not a local guard the reader can verify at the assertion itself. This is a real inconsistency between the two co-located hooks in the same file and is worth tightening (e.g. mirror `useAccountNames`'s pattern, or use `!!activeTenant` + `enabled` per FE-09). |
| FE-10 | Tenant ID in query keys | PASS | `useAccountNames` reuses `accountKeys.detail(activeTenant, String(id))` (`useAccounts.ts:39-40`, folds `tenant?.id \|\| "no-tenant"`); `useCharacterNames` reuses `characterKeys.detail(activeTenant!, String(id))` (`useCharacters.ts:35-36`, folds `tenant?.id`). `useItemCommodities.ts:10-11` `bySerial: (serialNumber, tenantId) => ["commodities", "serial", serialNumber, tenantId ?? ""] as const` mirrors the existing `byItem` key's tenant-suffix convention (`useItemCommodities.ts:8-9`). No cross-tenant cache collision risk. |
| FE-11 | Error handling with `createErrorFromUnknown` | PASS (no new `.catch()` sites) | No new `.catch(` calls were introduced. `useCommodityBySerial` (`useItemCommodities.ts:33-45`) and `useAccountNames`/`useCharacterNames` rely on React Query's built-in `isError`/`data` surfaces rather than manual `.catch()`, which is the established pattern for these query hooks elsewhere in the codebase (e.g. `useItemName`, `useItemNames`) — consistent, not a new gap. `resolvers.ts:544-553` (`EntityType.COUPON` resolver) follows the exact `try/catch` + `console.warn` + `throw new ResolverError(...)` shape already used by every other resolver in the file (e.g. `EntityType.BAN` at `resolvers.ts:437-449`) — consistent with established precedent, so not flagged as a fresh deviation even though it's `console.warn` rather than `createErrorFromUnknown`/toast (that's how every resolver in this file already behaves; resolvers are not user-facing forms/mutations). |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS (pre-existing type, unchanged) | `commodities.service.ts`'s new `toModel()` (`commodities.service.ts:18-29`) and `getBySerialNumber` (`commodities.service.ts:38-45`) both target `types/models/npc.ts:100-109`'s `ItemCashShopCommodity`, which is a flattened `{ id, itemId, count, ... }` shape rather than `{ id, attributes: {...} }`. This is a pre-existing type not touched by this diff (the diff only extracts the existing inline mapping into a shared `toModel` helper), so it is not a new violation, but flagged for visibility since new consumer code (`CashItemPicker.tsx`) now depends on it more heavily. |
| FE-13 | Service extends `BaseService` (when applicable) | PASS (pattern unchanged) | `commoditiesService` (`commodities.service.ts:31`) is a plain object literal, not a `BaseService` subclass or a class using the "Direct API Client Pattern" — this predates the diff (the object-literal shape was already there); the diff only adds `getBySerialNumber` inside the existing shape. Not a new deviation. |
| FE-14 | Query key factory uses `as const` | PASS | `itemCommoditiesKeys.bySerial` (`useItemCommodities.ts:10-11`) ends in `as const`, matching `byItem` (`useItemCommodities.ts:8-9`). No new key factory in `useActorNames.ts` — it reuses `accountKeys`/`characterKeys`, both already `as const`. |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | PASS (unchanged) | `RewardRowsField` remains a fully controlled, presentational field array (`RewardRowsField.tsx:1-11` doc comment, confirmed by props-only `rows`/`onChange`) — it is driven by the parent dialogs' `react-hook-form` `Controller`, which is out of this diff's scope and unchanged. `CashItemPicker` follows the same controlled `value`/`onChange` contract (`CashItemPicker.tsx:45-51`). |
| FE-16 | Schema in `lib/schemas/` with inferred type | PASS | `CURRENCY_VALUES` (`coupons.schema.ts:27`) is a plain `as const` array, not itself a schema, correctly placed alongside `couponRewardSchema` in `lib/schemas/coupons.schema.ts`; `couponRewardSchema`'s refinement at `coupons.schema.ts:79-83` consumes it. The file already pairs schemas with inferred/derived types (`_RewardMatchesService` at `coupons.schema.ts:96-98`, `RewardRowInput` interface at `coupons.schema.ts:30-36`). No inline duplicate definition anywhere else. |

## Component/Accessibility Notes (informational, non-blocking)

- `CashItemPicker.tsx:194-220` — `<ul role="listbox">` / `<li role="option" tabIndex={0}>` with manual `onKeyDown` for Enter/Space is a reasonable manual combobox-list pattern, and all interactive elements have explicit `type="button"` (`CashItemPicker.tsx:102-104,131-134,153-156,171-174`). No missing `aria-label` on icon-only affordances relevant here. Not a guideline violation; noted only because the pattern is hand-rolled rather than using an existing shadcn combobox primitive, which is worth a maintainer's eye but is not itself an FE-* checklist item.
- `resolvers.ts:548` `coupon.attributes?.code || \`Coupon ${entityId}\`` uses `||`, so a coupon whose `code` is an empty string (not just `null`/`undefined`) would silently fall back to `Coupon ${entityId}`. Every other resolver in the file (`ban`, `merchant`) uses the same `||` idiom for their own name-ish field, so this matches established precedent, not a new deviation — but genuinely blank `code` is a plausible enough coupon state that a `??` (only falling back on nullish) might be intended. Flagged as Minor, not blocking.

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS | `RewardRowsField.test.tsx` (new, 139 lines) exercises the currency `<Select>` (`RewardRowsField.test.tsx:377-388`), the `CashItemPicker` item→commodity flow (`:390-432`), and its manual-serial-entry path (`:434-446`) — i.e. `CashItemPicker` is covered end-to-end through its host even without a standalone `CashItemPicker.test.tsx`. `CouponDetailPage.test.tsx` gained a dedicated test for the new name-resolution behavior (`:913-935`) plus an assertion that the removed "Back to coupons" link is gone (`:907-911`). `resolvers.test.ts` and `routes.test.ts` both gained coverage for the new `EntityType.COUPON` resolver and `/coupons` routes. |
| FE-18 | Mocks updated when services changed | PASS | `RewardRowsField.test.tsx:331-333` mocks the new `commoditiesService.getBySerialNumber`; `CouponDetailPage.test.tsx:873-879` adds mocks for `accountsService.getAccountById` / `charactersService.getById` (consumed by the new `useActorNames` hooks); `resolvers.test.ts:456-460` adds a `couponsService` mock. All three mock additions match the actual new service/hook call surfaces introduced by this diff. |

## Summary

### Blocking (must fix)
- None. Build is clean, all 2011 tests pass, and no FE-* check produced a hard FAIL.

### Non-Blocking (should fix)
- **FE-09 (Important):** `useActorNames.ts` (`useAccountNames` at lines 44-61, `useCharacterNames` at lines 63-80) omits the `enabled: !!tenant?.id` guard on its `useQueries` entries that the FE-09 checklist item and the codebase's own precedent (`useItemNames.ts:21`) both require, relying instead on an indirect empty-array gate in `lookupIds()` (lines 28-31). Functionally safe today; add the explicit `enabled` flag (or at least assert the invariant with a comment/test that ties `lookupIds`'s behavior to the guard) so a future edit to `lookupIds` can't silently reintroduce an unguarded tenant-null fetch.
- **FE-09b (Important):** `useCharacterNames` (`useActorNames.ts:70`) uses `activeTenant!` while `useAccountNames` (`useActorNames.ts:51`) does not need to, purely because `characterKeys.detail` requires non-null `Tenant` while `accountKeys.detail` accepts `Tenant | null`. Recommend either widening `characterKeys.detail`'s signature to `Tenant | null` (matching `accountKeys.detail` and the majority precedent in `patterns-react-query.md`/`patterns-multitenancy.md`), or adding a local, visibly-scoped guard at the assertion site rather than relying on the cross-function invariant from `lookupIds`.
- **Minor:** `resolvers.ts:548`'s `||` fallback treats an empty-string coupon `code` the same as a missing one; matches existing precedent in the file but worth a follow-up if genuinely blank codes are possible.
- **Minor:** `CashItemPicker.tsx:180-191` uses inline loading/error text instead of a `Skeleton`, which is a minor deviation from "use skeleton components for loading" — acceptable given the small, inline nature of the content (a popover list), but noted for completeness since it wasn't a strict skeleton usage either.
