# Frontend Audit — task-289-tenant-template-drift-reset

- **Audit Scope:** `git diff 9613e7259..0f1b3c75e -- services/atlas-ui` (25 files: tenant template-drift attributes, `TenantResetButton`, `useResetTenantConfiguration`, drift badges on tenants list + detail header, six per-section reset mounts, mapleLife copy-on-clone in `onboarding.service.ts`)
- **Guidelines Source:** frontend-dev-guidelines skill (note: several skill examples assume Next.js App Router; this repo's actual stack per `services/atlas-ui/CLAUDE.md` is Vite + react-router-dom v7, Vitest not Jest — evaluated against the repo's real conventions where they differ from the generic skill text, e.g. no `page.tsx`/`layout.tsx` default-export exception applies here)
- **Date:** 2026-09-02
- **Build:** PASS (`npm run build` — `tsc -b && vite build`, clean, only a pre-existing chunk-size warning on `ConversationEditorPanel`, unrelated to this diff)
- **Tests:** 2399 passed, 0 failed (283 test files, `vitest run`)
- **Overall:** PASS

## Build & Test Results

```
$ npm run build
✓ built in 1.37s
(chunk-size warning only, unrelated to this diff)

$ npm test
 Test Files  283 passed (283)
      Tests  2399 passed (2399)
```

Per the task's tooling warning, the type-safety claim above is based on `npm run build` (`tsc -b && vite build`), not `npx tsc --noEmit`.

## File Inventory

- **Component:** `src/components/features/tenants/TenantResetButton.tsx` (new) — confirmation dialog, tooltip, toasts
- **Component:** `src/components/features/tenants/TenantDetailLayout.tsx` — drift badge + whole-document `TenantResetButton` mount in header
- **Hook:** `src/lib/hooks/api/useTenants.ts` — `useResetTenantConfiguration` mutation, `TenantResetSection` re-export
- **Service:** `src/services/api/tenants.service.ts` — `TenantResetSection` type, `reset()` method, computed-key stripping in `updateTenantConfiguration`
- **Service:** `src/services/api/onboarding.service.ts` — mapleLife copy-on-clone
- **Page:** `src/pages/TenantsPage.tsx` — `useTenantConfigurations` wired into `getColumns` for drift lookup
- **Page:** `src/pages/tenants-columns.tsx` — `templateDrift` badge column
- **Page:** `src/pages/tenants-properties-form.tsx`, `TenantsCharacterPresetsPage.tsx`, `TenantsCharacterTemplatesPage.tsx`, `TenantsHandlersPage.tsx`, `TenantsMapleLifePage.tsx`, `TenantsWritersPage.tsx` — six scoped `TenantResetButton` mounts
- **Other (util):** `src/lib/utils/config-export.ts` — strips the five computed drift keys from export payloads
- **Tests:** `TenantResetButton.test.tsx`, `TenantDetailLayout.test.tsx`, `useTenants.reset.test.tsx`, `tenants.service.test.ts` (extended), `onboarding.service.test.ts` (extended), `tenants-columns.test.tsx` (extended), `TenantsPage.test.tsx` (extended), `TenantsSectionReset.test.tsx` (new), `config-export.test.ts` (extended), `TenantsCharacterPresetsPage.test.tsx` / `TenantsCharacterTemplatesPage.test.tsx` (extended)

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -n ': any\|as any'` over all 25 changed files under `services/atlas-ui/src` — zero matches |
| FE-02 | No manual class concatenation | PASS | `grep -n 'className={"'` over all 25 changed files — zero matches; `cn()` used at `TenantResetButton.tsx:175` (`cn(buttonVariants({ variant: "destructive" }))`) |
| FE-03 | No direct API client calls in components/pages | PASS | `grep -n 'from "@/lib/api/client"'` over all changed files — only hit is `tenants.service.ts:1`, the service layer itself; no component/page imports it |
| FE-04 | No inline Zod schemas in components | PASS | Only `z.object(` hit is `tenants-properties-form.tsx:25`, which is pre-existing code (the diff for that file is 6 insertion lines mounting `TenantResetButton`, per `git diff 9613e7259..0f1b3c75e -- services/atlas-ui/src/pages/tenants-properties-form.tsx`); not introduced by this branch |
| FE-05 | No spinners for content loading | PASS | Only `animate-spin` hit is `TenantResetButton.tsx:179`, inside the `AlertDialogAction` confirm button (`isResetting` state), i.e. a submit-button spinner, the documented exception |
| FE-06 | No hardcoded colors | PASS | `grep -nE 'bg-(white|black|gray-\d+|red-\d+|blue-\d+|green-\d+)'` over all changed files — zero matches; badges use `variant="secondary"`/`variant="destructive"` (`tenants-columns.tsx:75`, `TenantDetailLayout.tsx:...`, `TenantResetButton.tsx:175`) |
| FE-07 | No state mutation | PASS | `.sort(`/`.push(`/`.splice(` hits in the diff range are all either pre-existing (`tenants.service.ts:190,209,212` — unchanged by this diff, confirmed via `git diff` hunks) or already-immutable spreads (`config-export.ts:100-101`, `[...array].sort(...)`) — no new mutation introduced |
| FE-08 | No default exports for components | PASS | `grep -n 'export default function'` over all changed files — zero matches; all six new page mounts and `TenantResetButton` use named exports, consistent with `services/atlas-ui/CLAUDE.md` ("Named exports on pages") |
| FE-09 | Tenant guard in hooks | PASS | `useResetTenantConfiguration` (`useTenants.ts`) is a mutation, not a query, so `enabled` does not apply; it takes an explicit `id: string` and delegates to `tenantsService.reset`, consistent with the tenant-management-service exception in `patterns-service-layer.md` ("Tenant-management services (TenantsService) don't take tenant parameter") |
| FE-10 | Tenant ID in query keys | PASS | `tenantKeys.configDetail`/`configLists`/`configList` all descend from `tenantKeys.configs()` under `tenantKeys.all`; existing key factory unchanged by this diff, no new tenant-scoped key added that skips tenant scoping |
| FE-11 | Error handling with `createErrorFromUnknown` | PASS | `TenantResetButton.tsx:91` — `toast.error(createErrorFromUnknown(e, "Reset failed").message)` inside the `onConfirm` catch block; verified by test `TenantResetButton.test.tsx:187-210` ("a failure toasts the server detail and leaves the dialog open") |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | `tenants.service.ts:157-160` — `interface TenantConfig { id: string; attributes: TenantConfigAttributes }`; new drift fields (`baselineTemplateId`, `baselineRevision`, `storedRevision`, `templateDrift`, `sectionDrift`) added as optional members of `TenantConfigAttributes` (`tenants.service.ts:141-152`), not a parallel shape |
| FE-13 | Service extends `BaseService` (when applicable) | PASS | `tenantsService` uses the object-literal / direct-client pattern (no `tenant` param on `reset`), matching the documented exception for tenant-management services in `patterns-service-layer.md` |
| FE-14 | Query key factory uses `as const` | PASS | `tenantKeys` factory (`useTenants.ts:30-45`, unchanged by this diff) — every branch ends `as const`; new hook reuses `tenantKeys.configDetail`/`configLists` rather than defining new untyped keys |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | N/A (not evaluable — no new form in this diff) | `TenantResetButton` is a confirmation dialog, not a form; no new `useForm` call introduced |
| FE-16 | Schema in `lib/schemas/` with inferred type | N/A | No new Zod schema introduced by this diff |
| **JSON:API `.data` unwrap on `api.post`** | `reset()` must unwrap since `api.post` does not | PASS | `tenants.service.ts:376-381` — `const response = await api.post<ApiSingleResponse<TenantConfig>>(...); return sortTenantConfig(response.data);`. Confirmed `api.post<T>` returns the raw processed response body without unwrapping (`src/lib/api/client.ts:238-253`, `return this.processResponse<T>(response);` with no `.data` access), so the explicit `response.data` here is required and correct. Verified further by `tenants.service.test.ts:126-140`, which mocks `post` to resolve `{ data: seededConfig() }` (the real wire envelope, per the test's own comment referencing `server.MarshalResponse[ViewRestModel]`) |
| **Cache invalidation covers the LIST query carrying drift badges** | Reset mutation must invalidate the query the tenants list badge reads from | PASS | Tenants-list drift badge (`tenants-columns.tsx`) reads from `useTenantConfigurations()` (`TenantsPage.tsx:150`), keyed `tenantKeys.configList(options)` = `[...tenantKeys.configLists(), options]` (`useTenants.ts:41-42`). `useResetTenantConfiguration`'s `onSuccess` invalidates `tenantKeys.configLists()` (`useTenants.ts` diff, `onSuccess` block), which is a query-key *prefix* of `configList(options)` — React Query's default `invalidateQueries` matching invalidates all queries whose key starts with the given array, so the list query is covered. Confirmed by test `useTenants.reset.test.tsx:70-86`, which asserts `configLists()` and `configDetail(id)` and `socketKeys.all` are all invalidated on success, and `:88-104`, which asserts nothing is invalidated on failure |
| **Cross-environment reset presented as 404, never 403** | No UI copy may treat 403 as the cross-environment case | PASS | `grep -rn "404\|403"` over `TenantResetButton.tsx`, `useTenants.ts`, `tenants.service.ts` — zero matches; the component has no status-code-specific branching at all, routing every failure through the same generic `createErrorFromUnknown(e, "Reset failed")` path (`TenantResetButton.tsx:91`), so there is no copy anywhere that could mischaracterize a 403 as the cross-environment case |
| **`scoped` boolean hardening covers every branch** | Confirmation copy must agree with what `tenantsService.reset` actually sends for every reachable `sections`/`sectionLabel` combination | PASS | `TenantResetButton.tsx:69` derives `scoped = sections !== undefined && sections.length > 0` once, and every piece of copy (`triggerLabel:97-99`, `titleLabel:100-102`, `confirmLabel:103-105`, dialog description:143-160, toast:79-83) branches on `scoped`, not on `sections` truthiness — including the `sections={[]}` case, which correctly renders whole-document copy. Verified by `TenantResetButton.test.tsx:271-290` ("an empty sections array renders whole-document copy, not scoped copy"), which asserts the whole-document heading, whole-document button label, and whole-document description text all appear. One residual gap: that same test does not additionally assert the *mutation call argument* for the `sections={[]}` case (it only asserts the rendered copy); `onConfirm` spreads `{ sections: [] }` into `reset.mutateAsync` for this case (`TenantResetButton.tsx:75-78`, `sections !== undefined` is true for `[]`), which `tenantsService.reset` then correctly collapses to no request body (`tenants.service.ts:372-375`, `sections && sections.length > 0`) — behavior is correct, but not directly asserted end-to-end from the button in that one test |
| **Caller contract: every mount passes non-empty `sections` + `sectionLabel`, or neither** | No mount may pass only one of the two, or an empty `sections` array | PASS | All six section-page mounts pass both a non-empty `sections` array and a `sectionLabel` (`tenants-properties-form.tsx`, `TenantsCharacterPresetsPage.tsx`, `TenantsCharacterTemplatesPage.tsx`, `TenantsHandlersPage.tsx`, `TenantsMapleLifePage.tsx`, `TenantsWritersPage.tsx` — all diff hunks above); `TenantDetailLayout.tsx` passes neither (`<TenantResetButton id={id} />`). No mount passes an empty array or a `sectionLabel` without `sections`. Verified by `TenantsSectionReset.test.tsx`, which mocks `TenantResetButton` and asserts the `data-sections`/`data-label` attributes for each of the six pages |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS | Every new/changed production file has a corresponding test file in the diff: `TenantResetButton.tsx` ↔ `TenantResetButton.test.tsx` (16 cases), `TenantDetailLayout.tsx` ↔ `TenantDetailLayout.test.tsx` (drift-summary cases at lines 117-162), `useTenants.ts` ↔ `useTenants.reset.test.tsx`, `tenants.service.ts` ↔ `tenants.service.test.ts` (`reset` + computed-key-hygiene describe blocks), `onboarding.service.ts` ↔ `onboarding.service.test.ts` (mapleLife copy cases at lines 122-222), `tenants-columns.tsx` ↔ `tenants-columns.test.tsx` (`templateDrift` describe block), the six section pages ↔ `TenantsSectionReset.test.tsx`, `config-export.ts` ↔ `config-export.test.ts` |
| FE-18 | Mocks updated when services changed | PASS | `TenantsSectionReset.test.tsx:23-35` and `TenantDetailLayout.test.tsx:16-21` both mock `@/components/features/tenants/TenantResetButton` with a prop-capturing stub reflecting the new `id`/`sections`/`sectionLabel` interface; `useTenants.reset.test.tsx:16-20` mocks `tenantsService.reset` directly |

## Not evaluable from the diff

- FE-15 / FE-16 (forms/Zod): no new form or schema is introduced by this diff, so these checklist rows are N/A rather than evaluated — recorded as N/A above rather than PASS/FAIL.
- Whether other, unrelated call sites in the wider codebase have the same `api.post` `.data`-unwrap defect class as the one fixed in `be544d91b`: out of scope per the Scope section — only this diff's new `api.post` call site (`tenants.service.ts:376`) was audited; a full-repo sweep for the same defect class was not performed.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- `TenantResetButton.test.tsx:271-290` verifies the empty-`sections` case renders whole-document copy but does not also assert the mutation is invoked with no meaningful `sections` from that exact render path (a one-line addition — spy on `mutateAsync` and assert the same shape as the `sections={undefined}` case — would close the gap noted above).
- (Already accepted, restated for completeness per task instructions, not new): dialog copy uses plain hyphens where the brief specified em dashes (`TenantResetButton.tsx:149,157`); no test exercises a truthy-but-not-`=== true` `templateDrift` value (e.g. `1` or `"true"`).
