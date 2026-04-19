# Tenant Rename — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-19
---

## 1. Overview

Atlas UI lets operators create tenants by cloning a template. Today the clone flow on `TemplatesPage` uses a hardcoded display name — `` `Tenant from Template ${templateId}` `` (see `services/atlas-ui/src/pages/TemplatesPage.tsx:154`) — which produces an unfriendly label that users cannot change later from the UI. The `TenantsPage` row actions currently only expose **Delete**; there is no way to edit a tenant's metadata once created.

This feature adds a **Rename** affordance to the tenants list so operators can give an existing tenant a friendly name, and updates the clone-from-template flow to collect a name up front rather than auto-generating one. Both the backend `PATCH /api/tenants/{tenantId}` handler and the frontend `tenantsService.updateTenant` helper already exist — this is primarily a UI change plus a small tweak to the onboarding dialog.

## 2. Goals

Primary goals:
- Operators can rename an existing tenant from the tenants list without visiting another page or editing config directly.
- When creating a tenant from a template, operators can specify a friendly name as part of the onboarding dialog instead of accepting an auto-generated one.
- Changes are reflected immediately in the list and in the tenant selector dropdown (both are driven by `TenantProvider`).

Non-goals:
- Editing `region`, `majorVersion`, or `minorVersion` in the same dialog. Those fields participate in tenant routing/versioning semantics and are out of scope.
- A bulk-rename flow.
- Rename from the tenant properties detail page.
- Enforcing uniqueness of tenant names (backend does not today; adding a client-side check would be a false guarantee).
- An audit log of name changes.
- Role-based permissioning (there is no auth layer in atlas-ui today — see `services/atlas-ui/CLAUDE.md`).

## 3. User Stories

- As an operator, I want to rename a tenant from the tenants list so I can replace the default `"Tenant from Template …"` label with something meaningful.
- As an operator creating a tenant from a template, I want to choose the tenant's name in the creation dialog so I don't have to immediately rename it afterwards.
- As an operator, I want feedback (success/failure toast) after renaming so I know the change was applied.
- As an operator, I want the renamed tenant to show up with its new name in the tenant selector dropdown without refreshing the page.

## 4. Functional Requirements

### 4.1 Rename action on the tenants list

- `services/atlas-ui/src/pages/tenants-columns.tsx` row-actions dropdown gets a new **Rename** item, placed above **Delete**. The dropdown stays a single `DropdownMenu` — no new column.
- Clicking **Rename** opens a modal dialog (shadcn `Dialog`, not `AlertDialog` — rename is non-destructive).
- The dialog contains a single text field `Name` prefilled with the tenant's current `attributes.name`.
- Submit button label: `Save`. Cancel button label: `Cancel`.
- Submit is disabled while the input is invalid, while the mutation is pending (label becomes `Saving…`), or when the trimmed name is unchanged from the tenant's current name (prevents no-op PATCH / tenant-updated event).
- On success: close the dialog, show a `toast.success("Tenant renamed")`, and ensure both the tenants list and the tenant selector reflect the new name.
- On failure: keep the dialog open, show `toast.error("Failed to rename tenant")`, re-enable submit.

### 4.2 Name validation

Zod schema in the dialog form:
- `name` is a string, trimmed.
- Minimum length: 1 (non-empty after trim).
- Maximum length: 100 characters.
- No uniqueness check.
- No character-set restriction beyond what the input allows.

Validation mode: `onChange` (matches `TemplatesPage` conventions).

### 4.3 Clone-from-template: collect name up front

- `services/atlas-ui/src/pages/TemplatesPage.tsx` — the "Create Tenant from Template" dialog (lines 323–347) currently has no form body. Add a single `Name` text field using `react-hook-form` + `zod`, subject to the same validation as §4.2.
- Default value: empty string. The operator must explicitly type a name (no auto-prefill from the template id).
- The create button stays disabled until the form is valid.
- The hardcoded `` `Tenant from Template ${templateForTenant.id}` `` (line 154) is removed — the name comes from the form.
- All other behavior of `onboardingService.onboardTenant` is preserved (atlas-tenants create + atlas-configurations create, same `ConfigurationCreationError` handling).

### 4.4 Data refresh / cache invalidation

- `TenantsPage` currently uses `useTenant()` + `refreshTenants()` rather than React Query hooks directly (see `services/atlas-ui/src/context/tenant-context.tsx` via `TenantProvider`). After a successful rename, call `refreshTenants()` so the table updates.
- `TenantProvider.refreshTenants` at `services/atlas-ui/src/context/tenant-context.tsx:73` today only re-selects the active tenant **if it was deleted** (line 81). It does not re-hydrate the active tenant's own attributes when it is still in the refreshed list, so a rename of the currently-active tenant leaves `activeTenant` pointing at a stale object and the selector keeps showing the old name. **This must be fixed** as part of this task: extend `refreshTenants` so that when the active tenant is still present in the refreshed list, `activeTenant` is re-set to the fresh object from that list.
  - The existing id-comparison effect at `tenant-context.tsx:35` will not trigger a cache clear when re-setting the same-id tenant (it compares by `id`), so this is safe.
  - `localStorage` does not need to change — the id is unchanged.
- No optimistic update. Wait for server response, then refresh. (Rename is rare; rollback/toast dance not worth it.)
- After refresh, the renamed tenant's new name must be visible in both the tenants table and the tenant selector (if it is the active tenant) without requiring a page reload or re-selection.

### 4.5 Error handling

- Network / 5xx failure → keep dialog open, show failure toast, log the error via existing `console.error` pattern used in `TenantsPage.handleDeleteTenant`.
- Form-level validation errors render inline under the field via `FormMessage`.
- The delete flow's existing error handling is not changed.

## 5. API Surface

No new endpoints. No existing endpoints change.

Existing endpoint used:
- `PATCH /api/tenants/{tenantId}` — `services/atlas-tenants/atlas.com/tenants/tenant/resource.go:93` (`UpdateTenantHandler`).
  - Request body (JSON:API): `{ data: { id, type: "tenants", attributes: { name, region, majorVersion, minorVersion } } }`.
  - Backend handler calls `processor.UpdateAndEmit(tenantId, name, region, majorVersion, minorVersion)` — all four fields participate, so the frontend must send the existing values for region/majorVersion/minorVersion along with the new `name`. `tenantsService.updateTenant` (`services/atlas-ui/src/services/api/tenants.service.ts:151`) already merges with the existing `tenant.attributes`, so passing `{ name }` is sufficient at the call site.
  - Response: `204`-equivalent (`api.patch<void>`); the frontend reconstructs the updated tenant locally.

Side effect to be aware of: `UpdateAndEmit` publishes a tenant-updated Kafka event regardless of which field changed. This is acceptable — a rename is a legitimate tenant change and downstream consumers should handle it.

## 6. Data Model

No schema changes.

Relevant existing types (unchanged):
- `TenantBasicAttributes` — `services/atlas-ui/src/services/api/tenants.service.ts:8`
- `tenant.Model` — `services/atlas-tenants/atlas.com/tenants/tenant/model.go:10`

## 7. Service Impact

### `services/atlas-ui`
- **Modified** `src/pages/TenantsPage.tsx` — rename dialog state, rename handler, dialog markup.
- **Modified** `src/pages/tenants-columns.tsx` — add `onRename` callback to `ColumnProps`, add `Rename` item to the dropdown.
- **Modified** `src/pages/TemplatesPage.tsx` — add `name` field to the "Create Tenant from Template" dialog; remove hardcoded name at line 154; thread form state into `handleCreateTenantFromTemplate`.
- **Modified** `src/context/tenant-context.tsx` — extend `refreshTenants` to re-hydrate `activeTenant` from the refreshed list when it is still present (see §4.4).
- **No changes expected** to `src/services/api/tenants.service.ts` — `updateTenant` is already sufficient.

### `services/atlas-tenants`
- No changes. Endpoint and processor already support name updates.

### Other services
- None.

## 8. Non-Functional Requirements

- **Multi-tenancy**: The rename dialog operates on a specific tenant row; the four tenant headers (`TENANT_ID` etc., per `services/atlas-ui/CLAUDE.md` § Tenant contract) are set by `api.setTenant` at the provider level and carry through the PATCH request as today. No header changes.
- **Performance**: A rename is a single PATCH followed by a list refresh. No perf concerns.
- **Accessibility**: Dialog gets a proper `DialogTitle` and `DialogDescription`. The name input gets a `FormLabel`. The rename dropdown item is keyboard-accessible via the existing shadcn `DropdownMenu` plumbing.
- **i18n**: No existing i18n system in atlas-ui; strings are hardcoded English consistent with the rest of the UI.
- **Tests**: Vitest + Testing Library, colocated under `__tests__/` (per CLAUDE.md § Testing). At minimum cover:
  - `tenants-columns.tsx` — renders Rename menu item when `onRename` is provided; invokes callback with the correct id.
  - `TenantsPage.tsx` — opens dialog with prefilled name, submits rename, shows success toast and calls `refreshTenants`.
  - Form validation — empty name is rejected; trimmed whitespace-only name is rejected; 100-char max; submit is disabled when the trimmed name equals the current name.
  - `tenant-context.tsx` — `refreshTenants` re-hydrates `activeTenant` when its attributes have changed in the refreshed list; still re-selects when the active tenant was removed.
- **Observability**: No new logging. Existing `console.error` on failure mirrors the delete path.

## 9. Open Questions

None at this time. Two prior open questions have been resolved and folded into the requirements:
- Submit is disabled when the trimmed new name equals the current name (§4.1) to avoid a no-op PATCH and the associated tenant-updated Kafka event.
- `TenantProvider.refreshTenants` will be extended to re-hydrate `activeTenant` from the refreshed list so a rename of the currently-active tenant is reflected in the selector without a page reload (§4.4).

## 10. Acceptance Criteria

- [ ] The tenants list row-actions dropdown shows a `Rename` item above `Delete`.
- [ ] Clicking `Rename` opens a dialog prefilled with the tenant's current name.
- [ ] Submitting a valid new name calls `PATCH /api/tenants/{id}` with the merged attributes, closes the dialog, shows a success toast, and refreshes the list so the new name is visible.
- [ ] Submitting an empty or whitespace-only name shows an inline validation error and does not fire a request.
- [ ] Submitting a name longer than 100 characters shows an inline validation error.
- [ ] Submit is disabled when the trimmed input equals the tenant's current name.
- [ ] When the renamed tenant is the active tenant, the tenant-selector label updates without a page reload (via the extended `refreshTenants`).
- [ ] Failure of the PATCH keeps the dialog open, shows an error toast, and logs to `console.error`.
- [ ] The "Create Tenant from Template" dialog requires a name (same validation as rename) and uses that name in `onboardingService.onboardTenant`. The hardcoded `Tenant from Template {id}` string is gone.
- [ ] Unit tests added for: new menu item, rename happy path, rename validation, clone-from-template name requirement.
- [ ] `npm run lint`, `npm run test`, and `npm run build` all pass in `services/atlas-ui`.
- [ ] Backend (`services/atlas-tenants`) has no code changes and its tests still pass.
