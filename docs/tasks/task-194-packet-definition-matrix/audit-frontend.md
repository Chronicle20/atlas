# Frontend Audit — task-194-packet-definition-matrix (atlas-ui)

- **Audit Scope:** `services/atlas-ui` changes between merge-base `31c7a664f975e8fadcd2e0e4e893427bddc340d9` and HEAD `a52adecc6` (new `src/lib/socket/`, `src/components/features/socket/**`, `src/pages/PacketMatrixPage.tsx`, `/packet-matrix` route, modified `templates.service.ts`/`tenants.service.ts`/`App.tsx`/`app-sidebar-items.ts`/`deployment-routes.ts`/the four per-object pages/`template.ts`, and the 6 deleted legacy form pages).
- **Guidelines Source:** `.claude/skills/frontend-dev-guidelines/*` + `services/atlas-ui/CLAUDE.md` (repo-specific; supersedes the skill's stale Next.js/Jest references — this app is Vite + React Router + Vitest).
- **Date:** 2026-08-06
- **Build:** PASS (`npm run build`, independently not re-run — `verification.md` §7 records `tsc -b` clean, exit 0; re-ran the new test suite instead, see below)
- **Tests:** re-ran the socket-feature subset independently — `npx vitest run src/lib/socket src/components/features/socket src/pages/__tests__/PacketMatrixPage.test.tsx src/lib/hooks/api/__tests__/useSocketObjects.test.tsx src/services/api/__tests__/socket-matrix.test.ts src/services/api/__tests__/templates-update.test.ts` → **16 files / 270 tests passed**, 0 failed. Full-suite result from `verification.md` (208 files / 1659 tests) not independently re-run — no reason found to doubt it.
- **Overall:** PASS (no blocking FE-* violation found; two pre-existing/deferred minors carried forward, both already recorded in `.superpowers/sdd/plan/progress.md`)

## Build & Test Results

Independent re-run:
```
$ npx vitest run src/lib/socket src/components/features/socket \
    src/pages/__tests__/PacketMatrixPage.test.tsx \
    src/lib/hooks/api/__tests__/useSocketObjects.test.tsx \
    src/services/api/__tests__/socket-matrix.test.ts \
    src/services/api/__tests__/templates-update.test.ts
Test Files  16 passed (16)
     Tests  270 passed (270)
```
Consistent with `verification.md`'s full-suite 208/1659 pass and clean `tsc -b`/`vite build`.

## File Inventory

- **Pages:** `src/pages/PacketMatrixPage.tsx` (new), `TemplatesHandlersPage.tsx`, `TemplatesWritersPage.tsx`, `TenantsHandlersPage.tsx`, `TenantsWritersPage.tsx` (all swapped onto `DefinitionGridPage`), `TemplatesCharacterPresetsPage.tsx`, `TemplatesCharacterTemplatesPage.tsx` (touched by the `useUpdateTemplate` signature change), `templates-worlds-form.tsx`, `templates-properties-form.tsx` (same).
- **Components (features/socket):** `DefinitionGridPage.tsx`, `GridToolbar.tsx`, `PacketGrid.tsx`, `PacketGridRow.tsx`, `PacketGridCell.tsx`, `DefinitionDrawer.tsx`, `DefinitionActionDialogs.tsx`, `OptionsMatrix.tsx`, `CopyFromAncestorFlow.tsx`, `FillMissingValidatorsDialog.tsx`, `dialogs/{AddDefinitionDialog,EditDefinitionDialog,CopyDefinitionDialog,DeleteDefinitionDialog,MarkUnsupportedDialog,ResetToAncestorDialog,fields,dialog-base}.tsx`.
- **Hook:** `src/lib/hooks/api/useSocketObjects.ts` (new); `useTemplates.ts` (signature hardened, `Partial<TemplateAttributes>` → `TemplateAttributes`).
- **Services:** `templates.service.ts`, `tenants.service.ts` (added `getSocketMatrix` sparse reads; doc comments on the write-path hazard).
- **Schema:** `src/lib/schemas/socket-definition.ts` (new).
- **Types:** `src/types/models/socket.ts` (new, shared `SocketConfig` shape), `template.ts` (now imports it).
- **Pure domain lib:** `src/lib/socket/{model,opcode,normalize,matrix,options,ancestry,mutate,routes}.ts` (new).
- **Tests:** colocated under `__tests__/` throughout all of the above; `dialogs/__tests__/dialogs.test.tsx` covers all six dialogs together.

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -rn ": any\|as any"` over every new/changed socket file returns zero matches. |
| FE-02 | No manual class concatenation | PASS | `grep -rn 'className={".*"\s*+'` returns zero matches; all conditional classes route through `cn()` (e.g. `PacketGridCell.tsx:43-48`). |
| FE-03 | No direct API client calls in components | PASS | `grep -rn '@/lib/api/client'` under `components/features/socket` and `pages/PacketMatrixPage.tsx` returns zero matches; every mutation goes through `useSocketMutation` (`src/lib/hooks/api/useSocketObjects.ts:94-133`). |
| FE-04 | No inline Zod schemas in components | PASS | The only `z.object` in the branch's socket surface is `src/lib/schemas/socket-definition.ts:18`; all six dialogs import it (`AddDefinitionDialog.tsx:22`, etc.). |
| FE-05 | No spinners for content loading | WARN (pre-existing pattern, not a new violation) | `DefinitionGridPage.tsx:205` and `PacketMatrixPage.tsx:179` render `<LoadingSpinner />` (an `animate-spin` `Loader2`, `src/components/common/LoadingSpinner.tsx:18-24`) for the page's initial full-document load, not a submit button. This is a genuine anti-pattern-doc violation, but `LoadingSpinner`/`PageLoader` predate this branch and are already used by 14 other existing pages vs. 13 using `Skeleton` — the branch followed an existing (inconsistent) repo convention rather than introducing a new one. Not blocking, but worth a follow-up to swap both new pages onto a skeleton. |
| FE-06 | No hardcoded colors | PASS | `grep -rnE "bg-(white|black|gray-[0-9]+|red-[0-9]+|...)"` over every new socket `.tsx` file returns zero matches; all state styling uses semantic classes (`bg-primary/5`, `bg-muted/40`, `text-muted-foreground`, `ring-primary`). |
| FE-07 | No state mutation | PASS | The only `.push`/`.splice`/`.sort` calls in new component code operate on freshly-built local arrays, never on React state in place: `ResetToAncestorDialog.tsx:45-46` (`parts.push` on a local array built that render), `GridToolbar.tsx:175,182` (`Array.from(new Set(...)).sort()` on a `useMemo`-derived array), `DefinitionDrawer.tsx:199` (`Array.from(new Set(...)).sort()`). `lib/socket/mutate.ts`'s `entries.push`/`entries.splice` (lines 201, 260) operate on `collectionOf`'s `cloneJson()` output, never the input `cfg` — every mutate.ts function is documented and tested as pure/non-aliasing (`mutate.ts:36-44`). |
| FE-08 | No default exports for components | PASS | `grep -rln "export default"` over every new file in `components/features/socket` and `pages/PacketMatrixPage.tsx` returns zero matches; all named exports. |
| FE-09 | Tenant guard in hooks | PASS | `useSocketMatrixTenants`/`useSocketMatrixTemplates` are tenant-agnostic (org-wide config reads, correctly Pattern-C — no `tenant` param needed since they list every template/tenant). `useSocketMutation`'s `mutationFn` re-derives per-target tenant/template id from its argument, not context. No `enabled` gap found for a hook that requires a tenant. |
| FE-10 | Tenant ID in query keys | N/A (not a per-tenant-scoped query) | `socketKeys` (`useSocketObjects.ts:37-41`) intentionally has no tenant dimension — it lists across ALL templates/tenants (a cross-tenant admin matrix by design, not a single active-tenant read), so `tenant?.id` inclusion does not apply here. Existing per-tenant hooks (`tenantKeys`, `templateKeys`) used elsewhere in the same files are unaffected by this branch. |
| FE-11 | Error handling with `createErrorFromUnknown` | PASS | Every dialog's submit catch uses it: `AddDefinitionDialog.tsx:82`, `CopyFromAncestorFlow.tsx:166`, `ResetToAncestorDialog.tsx:125`, and the rest of the six dialogs (grep confirms `createErrorFromUnknown` imported and used in all of `AddDefinitionDialog`, `EditDefinitionDialog`, `CopyDefinitionDialog`, `DeleteDefinitionDialog`, `MarkUnsupportedDialog`, `ResetToAncestorDialog`, `FillMissingValidatorsDialog`, `CopyFromAncestorFlow`), surfaced via `toast.error(...)`. |

## The sparse-cache write hazard (adversarial focus item)

**Could not construct a violating call path.** Findings:

- `templatesService.getSocketMatrix()` / `tenantsService.getSocketMatrix()` (sparse, `region/majorVersion/minorVersion/socket` only) are read exclusively by `useSocketMatrixTemplates`/`useSocketMatrixTenants` under `socketKeys.matrix()`/`socketKeys.tenantMatrix()` (`useSocketObjects.ts:37-67`).
- The single write path, `useSocketMutation` (`useSocketObjects.ts:94-133`), **never reads `socketKeys.*`**. Its `mutationFn` always calls `templatesService.getById(target.id)` or `tenantsService.getTenantConfigurationById(target.id)` — both full-document reads — immediately before applying the splice and PATCHing. Verified this holds for every one of the 8 mutation call sites (`grep -rn "useSocketMutation" src/components/features/socket` → `AddDefinitionDialog.tsx:52`, `EditDefinitionDialog.tsx:69`, `CopyDefinitionDialog.tsx:91`, `DeleteDefinitionDialog.tsx:56`, `MarkUnsupportedDialog.tsx:40`, `ResetToAncestorDialog.tsx:91`, `FillMissingValidatorsDialog.tsx:67`, `CopyFromAncestorFlow.tsx:108`).
- Sparse `SocketObject`s (from `fromTemplate`/`fromTenantConfig` over the matrix read) are used only as **read-only sources of binding values** to splice into the freshly re-fetched document (`CopyFromAncestorFlow.tsx:141-154`, `ResetToAncestorDialog.tsx:100-119`) — never as the mutation's base document.
- The general-purpose (non-socket) `useUpdateTemplate`/`useUpdateTenantConfiguration` hooks were independently hardened on this branch: `useTemplates.ts`'s `useUpdateTemplate` signature changed from `{ updates: Partial<TemplateAttributes> }` to `{ updates: TemplateAttributes }` (whole document required), and all 6 non-test call sites (`TemplatesCharacterTemplatesPage.tsx:26-28`, `templates-worlds-form.tsx:56`, `templates-properties-form.tsx:72-74`, `TemplatesCharacterPresetsPage.tsx:26-28`, plus the two Tenants-side equivalents) verified to spread `template.attributes`/`tenant.attributes` from the corresponding **full-document** `useTemplate(id)`/`useTenantConfiguration(id)` read, never from a sparse source.

This is a genuine, verified PASS — the design note in the code (`useSocketObjects.ts:27-41`, `templates.service.ts:247-259`, `tenants.service.ts:266-276`) matches the actual call graph.

## Zod schema vs. Go `validate.go` drift check

Compared `src/lib/schemas/socket-definition.ts` against `services/atlas-configurations/atlas.com/configurations/socket/validate.go`:

| Server rule (`validate.go`) | Client enforcement | Match |
|---|---|---|
| Name non-blank, trimmed (`validateCollection:94-99`) | `z.string().trim().min(1, ...)` (`socket-definition.ts:19`) | Yes |
| Opcode `0[xX][0-9A-Fa-f]{1,4}` (`validate.go:23`) | `OPCODE_PATTERN` — identical regex (`socket-definition.ts:11`) | Yes, byte-for-byte |
| Handler validator required, non-blank (`validate.go:122-127`) | `definitionFormSchemaFor("handler")` `superRefine` (`socket-definition.ts:45-55`) | Yes |
| Services ⊆ `{login, channel}` (`validate.go:129-136`, sourced from `libs/atlas-opcodes/config.go:6-7`) | `z.array(z.enum(KNOWN_SERVICES))`, `KNOWN_SERVICES = ["login","channel"]` (`socket-definition.ts:14`) | Yes — confirmed `ServiceLogin`/`ServiceChannel` constants equal `"login"`/`"channel"` in `libs/atlas-opcodes/config.go:6-7` |
| Duplicate `(name, opCode)` within a collection (`validate.go:108-119`) | Not a Zod field-level rule (can't be, it's a whole-document check) — instead enforced structurally by `lib/socket/mutate.ts`'s `addBinding`/`editBinding` collision checks (`mutate.ts:192-199`, `228-240`), which throw `MutationError` before any write. | Enforced, different layer |
| `unsupported` entry non-blank / not duplicated / not also-defined (`validate.go:150-168`) | No Zod rule (unsupported names are never free-typed — they come from an existing grid row name via `markUnsupported`/`clearUnsupported`, which is structurally single-add and auto-drops from the opposite list, `mutate.ts:203-206, 281-286`) | Enforced, different layer, unreachable-by-construction |

No drift found. The two rules not expressed as Zod are whole-document invariants the UI's mutation functions make structurally unreachable rather than field-validate — an acceptable, deliberate design choice, not a gap that lets a user submit something the server 400s on.

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | `Template`/`TenantConfig`/`TenantBasic` all `{ id: string, attributes: {...} }` (`tenants.service.ts:20-23, 125-128`, `template.ts`). `SocketConfig` (`types/models/socket.ts`) is a nested attribute shape, not a top-level resource — correctly not `{id,attributes}`. |
| FE-13 | Service pattern | PASS | `templatesService`/`tenantsService` use the documented direct-object pattern (not class-based `BaseService`, consistent with the rest of the codebase's non-`BaseService` services) — `services/api/templates.service.ts:195`, `tenants.service.ts:188`. |
| FE-14 | Query key factory uses `as const` | PASS | `socketKeys` (`useSocketObjects.ts:37-41`) — every branch `as const`. |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | PASS | `AddDefinitionDialog.tsx:54-57` (`useForm({ resolver: zodResolver(schema) })`); identical pattern confirmed in `EditDefinitionDialog.tsx`, `CopyDefinitionDialog.tsx`. |
| FE-16 | Schema in `lib/schemas/` with inferred type | PASS | `socket-definition.ts:18-30` — `definitionFormSchema` + `export type DefinitionFormValues = z.infer<typeof definitionFormSchema>`. |
| Rule 1 (task-specific) | `lib/socket/` has no React/RQ/service **value** import | PASS | Full import sweep of all 8 files in `src/lib/socket/` (`ancestry.ts, options.ts, opcode.ts, model.ts, matrix.ts, mutate.ts, normalize.ts, routes.ts`): the only cross-boundary import is `normalize.ts:2` — `import type { TenantConfig } from "@/services/api/tenants.service"` — a type-only import, erased under `verbatimModuleSyntax: true` (`tsconfig.app.json:12`). No React, no `@tanstack/react-query`, no value import from `services/api` anywhere in the directory. |
| Rule 2 (task-specific) | `exactOptionalPropertyTypes: true` handled correctly | PASS | Confirmed genuinely enabled at `tsconfig.app.json:27` (the `services/atlas-ui/CLAUDE.md` note claiming it's off is stale, as the task brief said). No cast found silencing it — every optional-field construction in the new code uses the conditional-spread idiom (`...(input.fname !== undefined ? { fname: input.fname } : {})`, `mutate.ts:83-89, 99-106`; same pattern repeated in every dialog's submit handler and in `ancestry.ts`/`ResetToAncestorDialog.tsx:112-118`). |
| Rule 3 (task-specific) | Absent-vs-disabled split preserved | PASS | Page-level controls (mode switch, column picker, baseline selector) are conditionally **not rendered** via optional handler props (`GridToolbar.tsx:398,416,424` — `{onKindChange && (...)}` etc.), matching FR-7.3/8.2. Drawer-level Open/Edit/Delete are **disabled** (not absent) via `disabled={editDeleteDisabled}` with a `title` reason (`DefinitionDrawer.tsx:342-366`), matching FR-5.4. Both confirmed correct per the task's ruling. |
| Rule 4 (task-specific) | No new dependency | PASS | `git diff ...package.json package-lock.json` empty (also independently confirmed in `verification.md` §10). |
| Rule 5 (task-specific) | `select.tsx` reuse / native checkbox justification | PASS | `GridToolbar.tsx:14-19` imports the existing `@/components/ui/select` (Radix) for the region/version pickers; `ColumnPicker`'s multi-select list (`GridToolbar.tsx:259-265`) and `CopyFromAncestorFlow.tsx:193-219`'s candidate list use native `<input type="checkbox">`, justified by the confirmed absence of `checkbox.tsx` in the component library. |

## Accessibility on the grid

| Check | Status | Evidence |
|---|---|---|
| `role="grid"` | PASS | `PacketGrid.tsx:73` — `<table role="grid" ...>`. |
| `aria-rowindex`/`aria-colindex` | PASS | Header row `aria-rowindex={1}` (`PacketGrid.tsx:75`), body rows `aria-rowindex={i + 2}` (`PacketGrid.tsx:125`), every `<th>`/`<td>` carries `aria-colindex` (`PacketGrid.tsx:78,86,96`, `PacketGridRow.tsx:44,60`, `PacketGridCell.tsx:42`). |
| State never color-only | PASS | Unsupported renders literal text `"n/a"` (`PacketGridCell.tsx:60`), duplicate-opcode and options-missing render labelled glyphs with `aria-label`/`title` (`PacketGridCell.tsx:71-85`), not color alone. |
| Keyboard navigation | WARN (partial) | `PacketGrid.tsx:48-61`'s `onKeyDown` only moves focus among the leftmost row-header `<button>`s (`tr > th button`) on ArrowUp/ArrowDown — there is no Left/Right cell-to-cell arrow navigation across a `role="grid"`'s data cells, which the ARIA grid pattern conventionally expects. Every cell button remains reachable via Tab and activatable via native Enter/Space, so the grid is not keyboard-inaccessible, but the arrow-key contract is incomplete relative to the `role="grid"` semantics it declares. Non-blocking (documented as a deferred minor in `progress.md` Task 14: "no test exercises Enter-key activation directly" — related but not identical gap; this specific left/right gap was not previously flagged, noting it here for the first time). |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS | Every new component/hook/lib file has a colocated `__tests__/*.test.ts(x)` sibling (confirmed via diff stat: 8 `lib/socket/__tests__/*`, `components/features/socket/__tests__/*` + `dialogs/__tests__/dialogs.test.tsx`, `useSocketObjects.test.tsx`, `PacketMatrixPage.test.tsx`, `socket-matrix.test.ts`, `templates-update.test.ts`). |
| FE-18 | Mocks updated when services changed | PASS (N/A formal `__mocks__/`) | No repo-wide `__mocks__/` directory convention exists (`find src -iname __mocks__` → empty); tests use inline `vi.mock()`. New service methods (`getSocketMatrix`) are covered by their own new test files (`socket-matrix.test.ts`) rather than requiring a shared mock update. |
| Vitest globals, not `jest.*` | PASS | `grep -rln "jest\."` over every new socket test file returns zero matches. |
| Assert behavior, not implementation | WARN (pre-existing, documented) | `progress.md` Task 17 already records: "tests pull the `apply` closure out of mocked `mutateAsync` calls and invoke it directly, so they prove 'the right splice was composed' rather than 'the right thing happened on screen'" — this pattern is real (spot-checked in `dialogs/__tests__/dialogs.test.tsx`) but was already flagged during the SDD review as matching the brief's own template, not a new finding. |

## Independently found, not previously flagged

- **`src/lib/socket/matrix.ts:209-224` `sortRows` tie-break bug (real, but is the exact sibling of an already-deferred bug).** When `key === "opcode"` and both rows' `baselineOpCodeValue` are `null`, `cmp = a.name.localeCompare(b.name)` is computed, but the function's final line (`return cmp === 0 ? a.name.localeCompare(b.name) : cmp * sign`) then multiplies that name-comparison result by `sign` a second time when the names differ — so a descending sort (`dir === "desc"`) reverses the alphabetical tie-break for opcode-less rows, directly contradicting the function's own doc comment at lines 200-202 ("A tie ... always breaks by ascending name, unaffected by `dir`"). This is the identical bug class `progress.md` Task 9 already fixed in the `"state"` branch and explicitly deferred for the `"opcode"` branch ("minor (deferred, FOR FINAL REVIEW)... Same class as the tie-break bug just fixed in the 'state' branch... 1-line fix"). Confirmed still present at HEAD. Non-blocking (cosmetic sort-order edge case limited to rows with an unparseable/absent baseline opcode sorted descending), but flagging per this audit's file:line-or-fail standard since it contradicts a doc comment that asserts the opposite behavior.

## Summary

### Blocking (must fix)
- None found.

### Non-Blocking (should fix)
- FE-05: `DefinitionGridPage.tsx:205`, `PacketMatrixPage.tsx:179` use `LoadingSpinner` (animate-spin) for full-page content loading instead of a skeleton — matches an existing (inconsistent) repo-wide pattern, not newly introduced, but still a documented anti-pattern.
- Accessibility: `PacketGrid.tsx:48-61` keyboard navigation is Up/Down-only across row headers; no Left/Right cell-to-cell arrow navigation for a `role="grid"`.
- `src/lib/socket/matrix.ts:209-224`: `sortRows`'s opcode-tie-break-by-name gets double-multiplied by `sign` when both compared opcodes are null, contradicting its own doc comment. Sibling of an already-fixed bug in the same function; already on record as deferred in `progress.md` Task 9, now independently reconfirmed still present at HEAD.
- Testing: dialog tests invoke the mocked `apply` closure directly rather than asserting purely on rendered DOM state (already flagged in `progress.md` Task 17; reconfirmed, not new).
