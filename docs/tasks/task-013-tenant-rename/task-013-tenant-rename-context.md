# Task 013 — Tenant Rename Context

**Last Updated:** 2026-04-19

## Source Material

- PRD: `docs/tasks/task-013-tenant-rename/prd.md` (authoritative).
- Frontend architecture: `services/atlas-ui/CLAUDE.md`.

## Key Files (to modify)

| File | Purpose | Notes |
|---|---|---|
| `services/atlas-ui/src/pages/tenants-columns.tsx` | Row-actions column | Add `onRename` to `ColumnProps` and a `Rename` `DropdownMenuItem` above `Delete`. |
| `services/atlas-ui/src/pages/TenantsPage.tsx` | Tenants list page | Add rename dialog state, submit handler, `Dialog` markup, wire `onRename` through `getColumns`. |
| `services/atlas-ui/src/pages/TemplatesPage.tsx` | Template onboarding | Replace hardcoded name at line 154 with form-driven `Name` field; wire `react-hook-form` into the dialog body (currently empty at lines 323–347). |
| `services/atlas-ui/src/context/tenant-context.tsx` | Tenant provider | Extend `refreshTenants` (line 73) to re-hydrate `activeTenant` when still present in refreshed list. |
| `services/atlas-ui/src/lib/schemas/` | Zod schemas | Add a shared tenant-name schema (trim, min 1, max 100). |

## Key Files (to test)

| File | Coverage |
|---|---|
| `services/atlas-ui/src/pages/__tests__/tenants-columns.test.tsx` | New `Rename` menu item visibility + callback. |
| `services/atlas-ui/src/pages/__tests__/TenantsPage.test.tsx` | Rename happy path, validation, disabled-when-unchanged, failure path. |
| `services/atlas-ui/src/pages/__tests__/TemplatesPage.test.tsx` | Name required in template onboarding; passes user-entered name to `onboardTenant`. |
| `services/atlas-ui/src/context/__tests__/tenant-context.test.tsx` | `refreshTenants` rehydration of active tenant without cache clear. |

## Key Files (read-only reference)

| File | Why it matters |
|---|---|
| `services/atlas-ui/src/services/api/tenants.service.ts:151` | `updateTenant` already merges `{ name }` with existing attributes. Call site passes `(tenant, { name })`. |
| `services/atlas-tenants/atlas.com/tenants/tenant/resource.go:93` | `UpdateTenantHandler`; confirms backend accepts rename-only PATCH. |
| `services/atlas-tenants/atlas.com/tenants/tenant/model.go:10` | `tenant.Model` — unchanged. |
| `services/atlas-ui/src/context/tenant-context.tsx:35` | Id-compare effect; confirms that re-setting `activeTenant` with the same id does **not** clear the query cache. |

## Decisions (from PRD + supporting analysis)

1. **Dialog, not AlertDialog** — rename is non-destructive (PRD §4.1).
2. **Rename above Delete** in the dropdown (PRD §4.1).
3. **Submit disabled when trimmed name == current name** — avoids a no-op PATCH and the redundant tenant-updated Kafka event (PRD §4.1, §9).
4. **No optimistic update** — rename is rare; wait for server, then `refreshTenants` (PRD §4.4).
5. **Extend `refreshTenants` to rehydrate the active tenant** so selector label updates without a reload (PRD §4.4, §9).
6. **Validation:** `onChange` mode; trim; min 1; max 100; no character-set or uniqueness rules (PRD §4.2).
7. **Template onboarding name is required with no auto-prefill** — operator must type one (PRD §4.3).
8. **No backend changes.** `services/atlas-tenants` diff must be empty (PRD §7).

## External Dependencies

- shadcn/ui: `Dialog`, `DropdownMenu`, `Form`, `Input`, `Button`, `Label` (all already vendored under `src/components/ui/`).
- `react-hook-form`, `zod`, `@hookform/resolvers/zod`, `sonner` — already in atlas-ui `package.json`.

## Kafka / Cross-Service Side Effects

- `PATCH /api/tenants/{id}` publishes a tenant-updated Kafka event via `processor.UpdateAndEmit` regardless of which field changed. This is acceptable per PRD §5; no new consumer work is required.

## Test & Verification Commands

```bash
cd services/atlas-ui
npm run lint
npm run test
npm run build
```

All three must pass before the task is complete.

## Non-Goals

See PRD §2 — out of scope: editing region/version fields, bulk rename, rename from detail page,
client-side uniqueness, audit log, RBAC.
