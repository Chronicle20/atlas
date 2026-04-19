# Task 013 — Tenant Rename Plan

**Last Updated:** 2026-04-19
**Source PRD:** `docs/tasks/task-013-tenant-rename/prd.md`
**Scope:** `services/atlas-ui` (frontend only). No backend changes.

## Executive Summary

Operators currently have no UI affordance to rename a tenant. The clone-from-template flow
auto-generates the unhelpful label `` `Tenant from Template ${templateId}` `` in
`services/atlas-ui/src/pages/TemplatesPage.tsx:154`, and the `TenantsPage` row-actions
dropdown only exposes **Delete**. The backend already supports the edit path
(`PATCH /api/tenants/{tenantId}` at `services/atlas-tenants/atlas.com/tenants/tenant/resource.go:93`)
and `tenantsService.updateTenant` already merges attributes
(`services/atlas-ui/src/services/api/tenants.service.ts:151`), so this is an atlas-ui-only task.

Deliverables:
1. Add a **Rename** row action on the tenants list that opens a shadcn `Dialog` with a prefilled, validated name field and submits a `PATCH`.
2. Replace the hardcoded name in the "Create Tenant from Template" dialog with a required `Name` field governed by the same validation.
3. Fix `TenantProvider.refreshTenants` so renaming the **active** tenant immediately updates the selector without a page reload.
4. Vitest coverage for the new affordances, validation rules, and the `refreshTenants` rehydration behavior.

## Current State Analysis

### Frontend
- `src/pages/TenantsPage.tsx` — list + delete flow; no edit surface. Uses `useTenant()` + `refreshTenants()` instead of direct React Query hooks.
- `src/pages/tenants-columns.tsx` — single-column row-actions dropdown (lines 44–70) exposes only **Delete**; `ColumnProps` only declares `onDelete`.
- `src/pages/TemplatesPage.tsx` — "Create Tenant from Template" dialog body is empty (lines 323–347); `handleCreateTenantFromTemplate` assembles the name inline from the template id at line 154.
- `src/context/tenant-context.tsx` — `refreshTenants` (line 73) only reselects the active tenant when it was **deleted** (line 81). If the active tenant is still present but its attributes changed (e.g., rename), `activeTenant` keeps pointing at the stale object, so the selector label does not update. The id-compare effect at line 35 means re-setting the same-id tenant will **not** clobber the query cache.
- `src/services/api/tenants.service.ts:151` — `updateTenant` already merges `{ name }` with the existing `tenant.attributes`, so the call site only needs to pass the new name.

### Backend
- `services/atlas-tenants/atlas.com/tenants/tenant/resource.go:93` — `UpdateTenantHandler` calls `processor.UpdateAndEmit(tenantId, name, region, majorVersion, minorVersion)`. A rename re-emits a tenant-updated Kafka event; this is acceptable per PRD §5.

## Proposed Future State

- Operators can rename any tenant from the list via a non-destructive `Dialog`. The active tenant's selector label updates without a reload.
- The template onboarding dialog requires a friendly name up front; the hardcoded `` `Tenant from Template ${id}` `` string is removed.
- `refreshTenants` re-hydrates the active tenant from the refreshed list when it is still present, so downstream consumers of `useTenant()` always see fresh attributes.
- Vitest suites cover the new menu item, rename happy/failure paths, trimmed-name edge cases, and the `refreshTenants` rehydration.

## Implementation Phases

### Phase 1 — Shared Form Schema & Context Fix (S)

Land the foundational pieces first so the two UI surfaces in Phase 2 can reuse them without churn.

1. Extract a Zod schema for the tenant-name field into a shared spot (see `src/lib/schemas/` per atlas-ui conventions) with the rules from PRD §4.2:
   - string, trimmed, min 1 (after trim), max 100, no character-set restriction.
   - Export a type alias for the form values.
2. Update `TenantProvider.refreshTenants` at `src/context/tenant-context.tsx:73` so that when `activeTenant` is present in the refreshed list, it is re-set to the fresh object. The existing deletion branch stays untouched. The id-compare effect at line 35 means the same-id re-set does not clear the query cache — verify this in a test.
3. Add/extend Vitest coverage in `src/context/__tests__/tenant-context.test.tsx` for:
   - Active tenant attributes change → `activeTenant` is re-hydrated; query cache is **not** cleared.
   - Active tenant removed → existing reselect behavior still fires.

### Phase 2 — UI Surfaces (M)

4. `src/pages/tenants-columns.tsx`
   - Extend `ColumnProps` with `onRename?: (id: string) => void`.
   - Add a `Rename` `DropdownMenuItem` placed **above** `Delete` inside the existing `DropdownMenu`. Non-destructive styling (no `text-destructive`).
   - Guard the item on `onRename` presence, mirroring the `onDelete` guard.
5. `src/pages/TenantsPage.tsx`
   - Add local state for the rename dialog (open flag + target tenant object, not just id — we need current attributes to prefill and submit merged payload).
   - Add the `Dialog` markup with `DialogTitle`, `DialogDescription`, a `react-hook-form` form bound to the Phase 1 schema, a single `FormField` for `Name` (prefilled with `tenant.attributes.name`), and `Cancel` / `Save` buttons.
   - Submit handler: call `tenantsService.updateTenant(tenant, { name })`, await, then `refreshTenants()`, close dialog, `toast.success("Tenant renamed")`.
   - Failure handler: keep dialog open, `toast.error("Failed to rename tenant")`, `console.error(err)` — mirror `handleDeleteTenant`'s pattern.
   - Submit button disabled when: form invalid, mutation pending (label swap to `Saving…`), or trimmed name equals current `tenant.attributes.name` (no-op guard per PRD §4.1).
   - Validation mode `onChange` (matches `TemplatesPage` conventions per PRD §4.2).
6. `src/pages/TemplatesPage.tsx`
   - Wire a `react-hook-form` instance with the Phase 1 schema into the "Create Tenant from Template" dialog body (currently empty, lines 323–347).
   - Disable the create button until the form is valid.
   - Thread the form's `name` value into `handleCreateTenantFromTemplate` and on into `onboardingService.onboardTenant`. Preserve existing `ConfigurationCreationError` handling untouched.
   - Remove the hardcoded `` `Tenant from Template ${templateForTenant.id}` `` at line 154.
   - Reset the form when the dialog closes so stale input does not leak across openings.

### Phase 3 — Tests (S)

7. `src/pages/__tests__/tenants-columns.test.tsx` (create or extend):
   - Renders `Rename` menu item when `onRename` is provided; invokes callback with the correct id.
   - Does not render `Rename` when `onRename` is omitted.
8. `src/pages/__tests__/TenantsPage.test.tsx` (create or extend):
   - Opens dialog prefilled with current name.
   - Submits valid new name → `updateTenant` called with merged attributes, dialog closes, success toast fires, `refreshTenants` called.
   - Empty / whitespace-only name rejected with inline error; no network call.
   - Name > 100 chars rejected with inline error.
   - Submit disabled when trimmed input equals current name.
   - PATCH failure keeps dialog open, shows error toast, logs to `console.error`.
9. `src/pages/__tests__/TemplatesPage.test.tsx` (create or extend):
   - Create button disabled until name is valid.
   - On submit, `onboardingService.onboardTenant` receives the user-entered name (not the hardcoded fallback).

### Phase 4 — Verification (S)

10. `cd services/atlas-ui && npm run lint && npm run test && npm run build` — all green.
11. Manual smoke test per PRD §10 acceptance criteria (rename happy path, active-tenant rename reflected in selector, clone-from-template name requirement).
12. Confirm `services/atlas-tenants` has no diffs and its existing tests are untouched.

## Risk Assessment & Mitigation

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Re-hydrating `activeTenant` with a same-id-different-attrs object triggers the id-compare effect and clears the query cache | Low | High (UX stutter, re-fetch storm) | The id-compare effect at `tenant-context.tsx:35` compares by `id`; same-id re-set won't fire. Lock this in with an explicit test in Phase 1 (step 3). |
| No-op PATCH emits a redundant tenant-updated Kafka event | Low | Low (downstream consumers already tolerate idempotent updates) | Disable submit when trimmed name == current name (PRD §4.1, Phase 2 step 5). |
| Rename dialog state leaks across rows if the dialog is reused | Medium | Low (prefill shows wrong name) | Store the **tenant object**, not the id, in dialog state; clear it on close. Reset the form in the dialog's open/close effect. |
| Form state from a previous template onboarding leaks into the next dialog open | Medium | Low | `form.reset()` on dialog close in `TemplatesPage` (Phase 2 step 6). |
| Legacy Jest-style tests in atlas-ui are excluded from `tsc -b` (per atlas-ui CLAUDE.md) | Medium | Medium | Write new tests against Vitest (`vi.*`, not `jest.*`). Do not touch the legacy Jest-era files. |

## Success Metrics

- All PRD §10 acceptance checkboxes satisfied.
- `npm run lint`, `npm run test`, `npm run build` pass in `services/atlas-ui`.
- Manual: renaming the active tenant updates the selector label without a page reload.
- No diffs in `services/atlas-tenants`.

## Dependencies & Resources

- Existing backend endpoint `PATCH /api/tenants/{tenantId}` (no change needed).
- Existing `tenantsService.updateTenant` (no change needed).
- shadcn `Dialog`, `DropdownMenu`, `Form`, `Input`, `Button` components.
- `react-hook-form`, `zod`, `sonner` (toast) — already in atlas-ui deps.

## Timeline Estimates

- Phase 1 (schema + context fix + its tests): **S** (~0.5 day)
- Phase 2 (three UI files): **M** (~1 day)
- Phase 3 (tests): **S** (~0.5 day)
- Phase 4 (verification): **S** (~0.25 day)
- **Total:** ~2 days of focused work; expect one fix-and-rebuild cycle on lint/test.

## Out of Scope (per PRD §2)

- Editing region / majorVersion / minorVersion in the rename dialog.
- Bulk rename.
- Rename from tenant properties detail page.
- Client-side uniqueness enforcement.
- Rename audit log.
- Role-based permissioning.
