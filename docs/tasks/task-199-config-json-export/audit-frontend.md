# Frontend Audit — task-199-config-json-export

- **Audit Scope:** TS/React changes, merge base `626161f8b` → head `182554b1b` (see file list below)
- **Guidelines Source:** frontend-dev-guidelines skill
- **Date:** 2026-08-07
- **Build:** PASS (established prior to this audit — not re-run per instructions)
- **Tests:** 234 files / 1890 tests passing (established prior to this audit — not re-run per instructions)
- **Overall:** NEEDS-WORK

## Build & Test Results

Per task instructions, the gate status was already established and not re-run:
`npm run test` 234 files / 1890 tests passing; `npm run build` (tsc + vite) clean;
`tools/lint.sh --check` OK (Prettier clean, ESLint 0 errors, 0 warnings in this branch's files).

## File Inventory

- `services/atlas-ui/src/lib/utils/download-json.ts` — Other (utility, new)
- `services/atlas-ui/src/lib/utils/config-export.ts` — Other (utility, new)
- `services/atlas-ui/src/components/DetailActionBarContext.tsx` — Component (context, modified — appended `useDetailActionBarState`)
- `services/atlas-ui/src/components/features/config/ConfigExportButton.tsx` — Component (new)
- `services/atlas-ui/src/components/features/templates/TemplateDetailLayout.tsx` — Component (modified — header restructure)
- `services/atlas-ui/src/components/features/tenants/TenantDetailLayout.tsx` — Component (modified — header restructure)
- `services/atlas-ui/src/lib/utils/__tests__/download-json.test.ts` — Test
- `services/atlas-ui/src/lib/utils/__tests__/config-export.test.ts` — Test
- `services/atlas-ui/src/components/features/config/__tests__/ConfigExportButton.test.tsx` — Test
- `services/atlas-ui/src/components/features/templates/__tests__/TemplateDetailLayout.test.tsx` — Test
- `services/atlas-ui/src/components/features/tenants/__tests__/TenantDetailLayout.test.tsx` — Test

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -n ': any\|as any'` over all 6 source files returns zero matches. |
| FE-02 | No manual class concatenation | PASS | No `className={"..." +` pattern in any changed file; grep returns zero matches. |
| FE-03 | No direct API client calls in components | PASS | No `@/lib/api/client` import in `ConfigExportButton.tsx`, `TemplateDetailLayout.tsx`, or `TenantDetailLayout.tsx`; `ConfigExportButton.tsx:11-12` imports `useTemplate`/`useTenantConfiguration` from the hooks layer instead. |
| FE-04 | No inline Zod schemas in components | PASS | No `z.object(`/`z.string(` in any changed file; feature adds no new validated form. |
| FE-05 | No spinners for content loading | PASS | No `animate-spin` in any changed file. The button is disabled (not spinner-gated) while the query is loading — `ConfigExportButton.tsx:74` `disabled={!query.data}`. |
| FE-06 | No hardcoded colors | PASS | No `bg-white|bg-black|bg-gray-*|text-gray-*|bg-red-*` etc. in any changed file. |
| FE-07 | No state mutation | PASS | `config-export.ts:71-73,79-83` builds a new `out` record and a new `socket` object rather than mutating `attributes`; `config-export.test.ts:102-109` (`does not mutate the input`) asserts this behaviorally. |
| FE-08 | No default exports for components | PASS | `ConfigExportButton.tsx:34`, `TemplateDetailLayout.tsx:15`, `TenantDetailLayout.tsx:15` all use `export function ComponentName(...)`. `grep -n 'export default'` over all changed files returns zero matches. |
| FE-09 | Tenant guard in hooks | PASS (pre-existing pattern, unmodified) | `ConfigExportButton.tsx:39-42` calls the pre-existing `useTemplate`/`useTenantConfiguration` hooks (`lib/hooks/api/useTemplates.ts:113-124`, `lib/hooks/api/useTenants.ts:228-242`), both Pattern-C (tenant-agnostic, resource looked up by its own `id`) with `enabled: !!id`. This mirrors the pre-existing call site `DefinitionGridPage.tsx:62-64`, which calls both hooks unconditionally with the same `scope === X ? id : ""` idiom — not a new anti-pattern, and not a Rules-of-Hooks violation (both hooks are called unconditionally on every render; only their `enabled` predicate varies). |
| FE-10 | Tenant ID in query keys | PASS (pre-existing, out of scope) | `templateKeys.detail(id)` / `tenantKeys.configDetail(id)` (used at `useTemplates.ts:118`, `useTenants.ts:233`) are keyed by resource `id`, not by `tenant?.id`, because templates and tenant-configuration documents are global resources addressed by their own id (Pattern C in `patterns-react-query.md`), not by the active tenant. `TenantProvider`'s `queryClient.clear()` on every tenant switch (per `services/atlas-ui/CLAUDE.md` "Tenant contract") independently prevents any cross-tenant cache leak, so a tenant switch cannot leak stale data into the export. Neither hook nor its key factory was touched by this branch. |
| FE-11 | Error handling with `createErrorFromUnknown` | **FAIL** | `ConfigExportButton.tsx:62-64` — `catch { toast.error("Export failed"); }` — swallows the error and never calls `createErrorFromUnknown()`. Every other async-catch site in `components/features/**` (e.g. `CopyFromAncestorFlow.tsx`, `PoolFormDialog.tsx`, `DeleteDefinitionDialog.tsx`, `TeleportRockListCard.tsx`, `MonsterBookWidget.tsx`, `AddTeleportRockMapDialog.tsx`, `ApplyPresetDialog.tsx`, `FillMissingValidatorsDialog.tsx`, `ResetToAncestorDialog.tsx`, `DefinitionActionDialogs.tsx`) routes through `createErrorFromUnknown()` before surfacing via toast, per the anti-pattern doc's canonical pattern (`anti-patterns.md` §10). This branch's catch block loses the actual error entirely (no `console.error`, no `errorLogger`, no classified message) — a `downloadJson` throw (e.g. cyclic-object serialization failure, per `download-json.ts:1-7`'s own doc comment) surfaces only the generic string "Export failed" with no diagnostic trail. |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | N/A | No `types/models/` files changed in this branch. |
| FE-13 | Service extends `BaseService` | N/A | No `services/api/` files changed; the feature deliberately adds no new service method (task framing: "Frontend only — no new endpoint, no new service method"). |
| FE-14 | Query key factory uses `as const` | N/A | No query key factory changed. |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | N/A | No form added. |
| FE-16 | Schema in `lib/schemas/` with inferred type | N/A | No Zod schema added. |
| — | Typing soundness of `toConfigExportPayload` | WARN | `config-export.ts:62-87` — `toConfigExportPayload<T extends ExportableConfigAttributes>(attributes: T): T` builds a `Record<string, unknown>` and asserts `as T` on return (line 86). This does not use `any` (FE-01 does not fire), and the file's own comment (lines 65-70) explains why: assigning to a property of a generic `T` is not expressible in TypeScript, so the record is built against the narrower structural interface and asserted back. The cast is not fully sound in the general case — nothing statically ties `out`'s shape back to `T`'s *other*, untouched properties, so a caller could in principle receive an object that satisfies `ExportableConfigAttributes` but not the full `T` if `toConfigExportPayload` ever started dropping or renaming a key it doesn't explicitly handle. In its current form it only ever spreads-then-reassigns three known keys (`npcs`, `worlds`, `socket`), so every other property of `T` survives untouched and the assertion is safe today. A safer formulation exists (e.g. taking `attributes: T & ExportableConfigAttributes` and building the output via `{ ...attributes, npcs: ..., worlds: ..., socket: ... }` typed as `T` directly via a mapped spread, or narrowing the return type to `Omit<T, 'npcs'\|'worlds'\|'socket'> & ExportableConfigAttributes`) but would complicate the call-site ergonomics documented in the comment. Non-blocking: no banned pattern is used, and both `config-export.test.ts` and the `tsc -b` build (which type-checks with `exactOptionalPropertyTypes`/`noUncheckedIndexedAccess` on, confirmed via `tsconfig.app.json:22,27`) pass. Recorded for awareness, not as an FE-* failure. |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS | `download-json.test.ts` (4 tests), `config-export.test.ts` (13 tests), `ConfigExportButton.test.tsx` (8 tests), `TemplateDetailLayout.test.tsx` (1 test), `TenantDetailLayout.test.tsx` (1 test) — one test file per non-trivial changed unit. `DetailActionBarContext.tsx`'s new `useDetailActionBarState` export is exercised indirectly through `ConfigExportButton.test.tsx` (renders inside `DetailActionBarProvider`) but has no dedicated unit test of its own null/registered branches (`DetailActionBarContext.tsx:110-113`). Not scored as a fail since the two call-time branches (`ctx?.config ?? null`) are trivial one-liners and are exercised transitively by the button test's disabled/enabled paths — flagged for completeness only. |
| FE-18 | Mocks updated when services changed | N/A | No `services/api/` files changed; `ConfigExportButton.test.tsx:20-25` mocks `templatesService`/`tenantsService` locally in the test file (both already existed, unmodified). |

## Summary

### Blocking (must fix)

- **FE-11** — `services/atlas-ui/src/components/features/config/ConfigExportButton.tsx:62-64`: the export-failure catch block calls `toast.error("Export failed")` directly without `createErrorFromUnknown()`, diverging from the project's established error-handling convention used at every other async-catch site under `components/features/**`. Fix: wrap with `createErrorFromUnknown(err, "Failed to export configuration")` and surface its `.message` via the toast (and/or log it), consistent with `anti-patterns.md` §10.

### Non-Blocking (should fix)

- `toConfigExportPayload`'s `as T` return cast (`config-export.ts:86`) is sound only because the function currently touches exactly three known keys; a future edit that drops/renames a key without updating this function would fail silently at the type level. Consider a narrower return-type formulation if the function grows more branches.
- `DetailActionBarContext.tsx`'s new `useDetailActionBarState()` hook (lines 110-113) has no dedicated unit test isolating its two branches (no provider / no registered config vs. registered config); currently only exercised transitively via `ConfigExportButton.test.tsx`.
