# Task 013 — Tenant Rename Tasks

**Last Updated:** 2026-04-19

Checklist tracks progress against the plan in `task-013-tenant-rename-plan.md`. Effort: S (<2h), M (2–6h), L (1 day), XL (>1 day).

## Phase 1 — Shared Schema & Context Fix

- [ ] **1.1** Add shared tenant-name Zod schema (S)
  - File: `services/atlas-ui/src/lib/schemas/tenant.ts` (create if missing; otherwise extend)
  - Rules: string, trimmed, `min(1)` after trim, `max(100)`; export inferred type.
  - Depends on: —
  - Acceptance: Schema is importable; unit-level round-trip of `{ name: "  foo  " }` → `{ name: "foo" }` validates; `""`, `"   "`, and 101-char strings fail.

- [ ] **1.2** Extend `refreshTenants` to rehydrate active tenant (S)
  - File: `services/atlas-ui/src/context/tenant-context.tsx` (around line 73)
  - Change: when the active tenant's id is still present in the refreshed list, set `activeTenant` to that fresh object. Preserve the existing deletion reselect path (line 81).
  - Depends on: —
  - Acceptance: Active tenant attribute changes are reflected without a reload; deletion path untouched.

- [ ] **1.3** Tests for `refreshTenants` rehydration (S)
  - File: `services/atlas-ui/src/context/__tests__/tenant-context.test.tsx`
  - Cases:
    - Active tenant still present with changed `attributes.name` → `activeTenant.attributes.name` updates; `queryClient.clear` is **not** called again (id unchanged, per effect at line 35).
    - Active tenant removed → existing reselect behavior still fires.
  - Depends on: 1.2
  - Acceptance: `npm run test` green for the context suite.

## Phase 2 — UI Surfaces

- [ ] **2.1** Add `onRename` to tenants-columns (S)
  - File: `services/atlas-ui/src/pages/tenants-columns.tsx`
  - Change: extend `ColumnProps` with `onRename?: (id: string) => void`; add `Rename` `DropdownMenuItem` above `Delete` inside the existing `DropdownMenu`. Non-destructive styling.
  - Depends on: —
  - Acceptance: Column renders with both items when callbacks provided; only `Delete` when `onRename` absent.

- [ ] **2.2** Rename dialog state + handler in `TenantsPage` (M)
  - File: `services/atlas-ui/src/pages/TenantsPage.tsx`
  - Store the **target tenant object** (not just id) in state so the dialog can prefill and submit the merged payload.
  - Submit handler calls `tenantsService.updateTenant(tenant, { name })`, awaits, then `refreshTenants()`, then closes dialog and toasts success.
  - Failure: keep dialog open, `toast.error(...)`, `console.error(...)` — mirror `handleDeleteTenant`.
  - Depends on: 1.1, 2.1
  - Acceptance: Rename happy path and failure path work in a manual smoke test.

- [ ] **2.3** Rename dialog markup (M)
  - File: `services/atlas-ui/src/pages/TenantsPage.tsx`
  - Use shadcn `Dialog` (not `AlertDialog`) with `DialogTitle` + `DialogDescription`.
  - `react-hook-form` + schema from 1.1, `mode: "onChange"`, prefilled with `tenant.attributes.name`.
  - Buttons: `Cancel` / `Save` (label → `Saving…` while pending).
  - Submit disabled when: form invalid, mutation pending, or `trim(name) === trim(current)`.
  - Reset form on dialog close.
  - Depends on: 2.2
  - Acceptance: All PRD §4.1 behaviors satisfied.

- [ ] **2.4** Template onboarding: name field in dialog (M)
  - File: `services/atlas-ui/src/pages/TemplatesPage.tsx`
  - Wire `react-hook-form` + schema from 1.1 into the currently-empty dialog body (lines 323–347).
  - Create button disabled until valid.
  - Thread form `name` into `handleCreateTenantFromTemplate` → `onboardingService.onboardTenant`.
  - Remove hardcoded `` `Tenant from Template ${templateForTenant.id}` `` at line 154.
  - Reset form on dialog close.
  - Preserve existing `ConfigurationCreationError` handling verbatim.
  - Depends on: 1.1
  - Acceptance: Operator must type a name; hardcoded string no longer appears in the codebase (`grep -rn "Tenant from Template " services/atlas-ui/src` returns zero).

## Phase 3 — Tests

- [ ] **3.1** `tenants-columns` tests (S)
  - File: `services/atlas-ui/src/pages/__tests__/tenants-columns.test.tsx`
  - Cases: renders `Rename` when `onRename` provided and invokes with correct id; omits it otherwise.
  - Depends on: 2.1
  - Acceptance: Suite green.

- [ ] **3.2** `TenantsPage` rename tests (M)
  - File: `services/atlas-ui/src/pages/__tests__/TenantsPage.test.tsx`
  - Cases:
    - Dialog opens with current name prefilled.
    - Valid new name → `updateTenant` called with merged attrs; dialog closes; success toast; `refreshTenants` called.
    - Empty / whitespace-only → inline error; no network call.
    - > 100 chars → inline error.
    - Submit disabled when trimmed input == current name.
    - PATCH failure → dialog stays open; error toast; `console.error` called.
  - Depends on: 2.2, 2.3
  - Acceptance: Suite green.

- [ ] **3.3** `TemplatesPage` name-required tests (S)
  - File: `services/atlas-ui/src/pages/__tests__/TemplatesPage.test.tsx`
  - Cases: Create disabled until valid; on submit `onboardTenant` receives user-entered name.
  - Depends on: 2.4
  - Acceptance: Suite green.

## Phase 4 — Verification

- [ ] **4.1** Lint + tests + build (S)
  - Commands: `cd services/atlas-ui && npm run lint && npm run test && npm run build`
  - Depends on: all prior phases
  - Acceptance: All three exit 0.

- [ ] **4.2** Manual smoke test (S)
  - Checklist mirrors PRD §10:
    - [ ] Rename from list → dialog prefills, saves, toast, list updates.
    - [ ] Renaming the **active** tenant updates the selector label without a reload.
    - [ ] Empty / whitespace / >100 chars rejected inline.
    - [ ] Submit disabled when trimmed name is unchanged.
    - [ ] PATCH failure keeps dialog open with error toast.
    - [ ] Template onboarding requires a name; no more "Tenant from Template …" placeholder.
  - Depends on: 4.1
  - Acceptance: All boxes ticked.

- [ ] **4.3** Confirm backend diff is empty (S)
  - Command: `git status services/atlas-tenants` should show no modifications.
  - Acceptance: Zero diffs in `services/atlas-tenants`.
