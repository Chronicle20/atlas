# Frontend Audit — task-271-maple-life-config-ui

- **Audit Scope:** `git diff 4358e586f..b9c859bc9` (merge base `4358e586f`, HEAD `b9c859bc9`), atlas-ui only — 47 changed files (types, service layer, Zod schema, reducer/projection modules, warning/loadout derivation, 8 editor section components, 1 shared editor, 2 pages, routing/breadcrumb wiring, and their tests).
- **Guidelines Source:** frontend-dev-guidelines skill (cross-checked against `services/atlas-ui/CLAUDE.md`, which documents the real Vite + React Router + Vitest stack; the skill's Next.js/Jest framing is stale for this repo and was read accordingly).
- **Date:** 2026-08-27
- **Build:** PASS (discharged externally — `tools/verify.sh` green over commit `e36cbc0bf`, per task instructions; not re-run)
- **Tests:** 2316/2316 passed (discharged externally, per task instructions; not re-run)
- **Overall:** NEEDS-WORK (0 blocking FAIL, 2 non-blocking WARN)

## Build & Test Results

Per task instructions, `tools/verify.sh`/`npm run build`/`npm test` were **not** re-run. Verification is already discharged: the flagless `tools/verify.sh` passed over commit `e36cbc0bf` — atlas-ui lint+format guard and atlas-ui test+build gate both green, 2316/2316 vitest, 0 lint errors. The 9 remaining eslint warnings are pre-existing, in files this branch does not touch, and are not re-raised here.

## File Inventory

- **Page:** `services/atlas-ui/src/pages/TenantsMapleLifePage.tsx`, `services/atlas-ui/src/pages/TemplatesMapleLifePage.tsx`
- **Component (feature, new):** `components/features/characters/maple-life/{AppearancePoolsSection,ClassSelector,EmptyState,IdentitySection,MapleLifeEditor,MapleLifePreviewCard,ProgressionSection,SeedFromTemplateDialog,SpSkillSection,StartingKitSection}.tsx`
- **Domain/state module (new, non-component `.ts` under a component dir):** `components/features/characters/maple-life/{mapleLifeEditorState,mapleLifeLoadout,mapleLifeWarnings}.ts`
- **Component (feature, modified):** `components/features/characters/templates/{AppearancePoolSection,CharacterTemplatesEditor,SaveBar}.tsx`, `components/features/characters/presets/{EquipmentSection,InventorySection}.tsx` (type-only rename), `components/features/templates/TemplateDetailLayout.tsx`, `components/features/tenants/TenantDetailLayout.tsx`, `components/DetailActionBarContext.tsx`
- **Schema:** `lib/schemas/maple-life.schema.ts`
- **Type:** `types/models/template.ts` (adds `MapleLifeConfig` and friends)
- **Service:** `services/api/tenants.service.ts` (+2 lines: `mapleLife?: MapleLifeConfig` attribute and import)
- **Other:** `App.tsx` (2 new lazy routes), `lib/breadcrumbs/routes.ts` (2 new route configs), test files for every item above (23 test files, ~2900 LOC)

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -rn ': any\b\|as any\b'` across all in-scope `.ts`/`.tsx` (including tests) — zero matches |
| FE-02 | No manual class concatenation | PASS | `grep -rn 'className={"'` across in-scope components — zero matches; all conditional classes go through `cn()` or a fixed ternary string (e.g. `ClassSelector.tsx:61-66` `tabClassName`) |
| FE-03 | No direct API client calls in components | PASS | `grep -rln '@/lib/api/client'` across `components/features/characters/maple-life`, both pages — zero matches; pages go through `useTenantConfiguration`/`useUpdateTenantConfiguration`/`useTemplate`/`useUpdateTemplate` (`pages/TenantsMapleLifePage.tsx:9-12`, `pages/TemplatesMapleLifePage.tsx:9`) |
| FE-04 | No inline Zod schemas in components | PASS | `grep -rn 'z\.object(\|z\.string(\|z\.number('` across `components/features/characters/maple-life` — zero matches; all validation lives in `lib/schemas/maple-life.schema.ts` |
| FE-05 | No spinners for content loading | PASS | `grep -rln 'animate-spin'` across in-scope files — zero matches; `MapleLifeEditor.tsx:210` uses `<FormSkeleton fields={8} />` for the pre-load window |
| FE-06 | No hardcoded colors | PASS | `grep -rnE 'bg-(white\|black\|gray-[0-9]+\|red-[0-9]+...)'` — zero matches. `IdentitySection.tsx:132-134` uses `border-warning bg-warning/10 text-warning-foreground`, backed by real theme tokens at `services/atlas-ui/src/index.css:33-34,141,143,213,215`. `MapleLifePreviewCard.tsx:35` has an arbitrary `drop-shadow-[0_6px_4px_rgba(0,0,0,0.25)]`, but this is a verbatim copy of pre-existing, unchanged sibling code (`presets/PresetPreviewCard.tsx:107`, `templates/PreviewCard.tsx:118`) — not a new violation introduced by this branch |
| FE-07 | No state mutation | PASS | `.push()` hits in `mapleLifeEditorState.ts:139,153,182` and `mapleLifeWarnings.ts:31,35,42,56` are all on locally-scoped arrays built fresh inside a function body (`buildDrafts`, `buildLooks`, `parseSpPool`, `mapleLifeWarnings`) and returned, never a mutation of reducer state; `mapleLifeReducer` (`mapleLifeEditorState.ts:322-448`) is spread-based throughout, including nested updates (`updateSelected`/`updateSelectedLook` at `mapleLifeEditorState.ts:233-251`) |
| FE-08 | No default exports for components | PASS | `grep -rn 'export default function'` across in-scope files — zero matches; all named exports |
| FE-09 | Tenant guard in hooks | N/A / not evaluable — see below | No hook files (`lib/hooks/api/`) are touched by this diff. `MapleLifePreviewCard.tsx:20,25` (the one component that reads `useTenant()` directly) correctly guards render with `{activeTenant && (...)}` |
| FE-10 | Tenant ID in query keys | Not evaluable from the diff | No query-key factories are touched by this diff — see "Not evaluable" section |
| FE-11 | Error handling with `createErrorFromUnknown` | WARN (non-blocking) | Both page adapters surface mutation failure via `onError: (error) => toast.error(...)` (`pages/TenantsMapleLifePage.tsx:39-42`, `pages/TemplatesMapleLifePage.tsx:29-32|`) — acceptable, react-query's own `onError` is the equivalent of the anti-pattern doc's `.catch()` + `createErrorFromUnknown()` idiom. However, `SeedFromTemplateDialog.tsx:47` calls `useTemplates()` and destructures only `data`, never checking `isError`/`isLoading` — a failed or in-flight templates fetch renders as "No template of this region and version carries a Maple Life block" (`SeedFromTemplateDialog.tsx:104-108`), indistinguishable from a genuine zero-match. This mirrors a pre-existing pattern (`components/features/baselines/BaselineTargetPicker.tsx:31` does the same), so it is not a new anti-pattern, but it is a genuine gap in this component and worth fixing |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | `Template { id: string; attributes: TemplateAttributes }` (`types/models/template.ts:148-151`) unchanged; `MapleLifeConfig`/`MapleLifeClassEntry`/`MapleLifeLookOptions` (`types/models/template.ts:73-114`) are correctly modeled as nested attribute value objects, not top-level resources, so `{id, attributes}` does not apply to them |
| FE-13 | Service extends `BaseService` (when applicable) | PASS | `tenants.service.ts` uses the documented "direct API client" object-literal pattern (pre-existing, not `BaseService`-derived); the 2-line diff (`tenants.service.ts:9,133`) only adds the `mapleLife?: MapleLifeConfig` attribute and its import, doesn't change the pattern |
| FE-14 | Query key factory uses `as const` | Not evaluable from the diff | No `lib/hooks/api/` files are touched — see "Not evaluable" section |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | PASS (established precedent) | `MapleLifeEditor.tsx` uses `useReducer` (`MapleLifeEditor.tsx:74-78`), not RHF. This matches the pre-existing, unchanged `CharacterTemplatesEditor.tsx:1,55` (`useReducer`, no RHF) — the multi-section, cross-field-derived editor pattern in this codebase is deliberately reducer-based, not a new deviation from FE-15 |
| FE-16 | Schema in `lib/schemas/` with inferred type | WARN (non-blocking) | `lib/schemas/maple-life.schema.ts:113` declares `export const mapleLifeSchema: z.ZodType<MapleLifeConfig> = z.object({...})...` — the outer schema and its five nested sub-schemas (`lookOptions`, `statBlock`, `equipmentEntry`, `inventoryEntry`, `classEntry`, `maple-life.schema.ts:25-102`) never export a `z.infer`-derived type; the type instead flows the other way (`MapleLifeConfig` imported from `types/models/template.ts:9` and asserted onto the schema). This is a defensible, in-code-documented tradeoff (matching the server-side domain type exactly, plus the `exactOptionalPropertyTypes` transform at `maple-life.schema.ts:99-104`), but it is a genuine deviation from the documented "schema paired with `export type X = z.infer<typeof schema>`" pattern and was not on the pre-ruled exclusion list |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS | Every new/modified component and module has a matching `__tests__` file: `AppearancePoolsSection`, `ClassSelector`, `IdentitySection`, `MapleLifeEditor`, `MapleLifePreviewCard`, `ProgressionSection`, `SeedFromTemplateDialog`, `SpSkillSection`, `StartingKitSection`, `mapleLifeEditorState`, `mapleLifeLoadout`, `mapleLifeWarnings`, `maple-life.schema`, `AppearancePoolSection` (updated), `SaveBar` (new), `DetailActionBarContext` (new), `routes.test.ts` (breadcrumbs), `templates-update.test.ts` (updated), `tenants.service.test.ts` (new). The two page components (`TenantsMapleLifePage.tsx`, `TemplatesMapleLifePage.tsx`) have no test file — already ruled as a documented, deliberate exclusion and not re-raised |
| FE-18 | Mocks updated when services changed | N/A | No shared `__mocks__/` directory exists in this codebase; every test file uses an inline `vi.mock(...)` (e.g. `StartingKitSection.test.tsx:6-8`, `tenants.service.test.ts:6-15`), consistent with the rest of the repo, so there is nothing to diff against |

## Not evaluable from the diff

- FE-09 (tenant guard) and FE-10 (tenant ID in query keys) for `useTenantConfiguration`, `useUpdateTenantConfiguration`, `useTemplate`, `useUpdateTemplate` — these hooks are consumed by the two new pages but are not themselves in the diff (`lib/hooks/api/useTenants.ts`, `lib/hooks/api/useTemplates.ts` are unchanged). Confirming their `enabled: !!tenant?.id` guards and tenant-scoped query keys would require reading those hook files outside the review surface.
- FE-14 (query key factory `as const`) — same reason; no hook files are touched by this diff.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- FE-11 — `SeedFromTemplateDialog.tsx:47` doesn't surface `useTemplates()` loading/error state, so a failed or in-flight fetch is indistinguishable from "no eligible template" (`SeedFromTemplateDialog.tsx:104-108`). Pre-existing pattern elsewhere in the repo, not unique to this branch, but worth fixing here since the empty-vs-error distinction matters for an admin-facing seed action.
- FE-16 — `lib/schemas/maple-life.schema.ts` types flow from `types/models/template.ts` onto the schema (`z.ZodType<MapleLifeConfig>`) rather than the documented `export type X = z.infer<typeof schema>` pattern; defensible given the need to match the server-side domain type exactly, but a deviation from the documented convention.
