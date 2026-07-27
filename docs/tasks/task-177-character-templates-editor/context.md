# Character Templates Editor — Context

Task: task-177-character-templates-editor
Companion to [plan.md](plan.md). Read this first when picking up any task.

## What this is

Replace `src/pages/tenants-character-templates-form.tsx` and
`src/pages/templates-character-templates-form.tsx` (two wholesale-duplicated
raw-ID forms in atlas-ui) with one shared visual editor under
`src/components/features/characters/templates/`. Both page components keep
their names/routes and render the shared editor inside their existing layouts.
**Zero backend changes. Zero layout/shell changes. atlas-ui only.**

## Key decisions (from design.md — do not relitigate)

- **D1** Plain `useReducer` state module (`editorState.ts`), NOT react-hook-form.
  One working array + baseline snapshot; dirty = deep compare. Free template
  switching never loses edits; one Save/Discard bar for the whole array.
- **D2** Editor takes one `TemplatesEditorAdapter` prop
  (`{ templates, isLoading, error, save(templates, onSuccess), isSaving }`);
  each page builds it from its existing hooks. Tenant page PATCHes
  `{ characters: { ...existing, templates } }` via
  `useUpdateTenantConfiguration`; template page via `useUpdateTemplate` —
  spreading `characters` preserves sibling `presets`.
- **D3** Cosmetics: enumerate `/api/data/cosmetics/faces|hairs` with `fetchAll`
  (item-strings **search does not cover faces/hairs** — resolve names per id
  via `/api/data/item-strings/{id}`, batched with `useQueries` sharing
  `useItemName`'s cache keys). Gender filter = v83 convention
  `(id/1000)%10===1 ⇒ female`, extracted as `isFemaleCosmeticId` in
  `characterRender.service.ts`.
- **D4** Equipment search: server `filter[subcategory]` for bottoms/shoes
  (single token only — atlas-data `parseFilters` rejects comma lists);
  client-side subcategory-set filter for tops (`top`+`overall`) and weapons
  (16 tokens from `filter.go:55-60`, no `pet-equip`). Configs live in
  `poolSearchConfig.ts`.
- **D5** New shadcn `popover.tsx` + dep `@radix-ui/react-popover` (the ONLY new
  dep). No cmdk. Combobox = Popover + Input + `role="listbox"` rows with a
  "Use id N" manual fallback. MapPicker reuses the same shell.
- **D6** Appearance thumbnails = full stand1/resize=2 composite CSS-cropped to
  ~76px head region (start offsets `-74/-70`, tune at the end). Browser dialog
  pages at 24; grid images are plain lazy `<img>`; only the big preview uses
  `useCharacterImage`.
- **D7** `?tpl=<index>` URL sync via `useSearchParams` with `{replace:true}`;
  clamp invalid values to 0 and write back.
- **D8** Preview loadout: skin/hair/face from picked pool entries (hair =
  base hair + color digit), equipment = first entry of each pool on slots
  top `-5`, bottom `-6`, shoes `-7`, weapon `-11`; empty pools fall back to
  skin 0 / hair 30030 / face 20000.

Dropped by user decision (do NOT implement): selector thumbnails, Edit-as-JSON,
preview id/value table, Creator-Preview mode, skill browser.

## Key existing files (verified)

| File | Why it matters |
|---|---|
| `src/types/models/template.ts:4-19` | `CharacterTemplate` — the persisted shape, unchanged |
| `src/services/api/characterRender.service.ts` | `generateCharacterUrl`, `resolveGender` (line 58), `CharacterLoadout`; gains `isFemaleCosmeticId` |
| `src/lib/hooks/useCharacterImage.ts` | Big-preview machinery (skeleton/error/retry); takes `MapleStoryCharacterData` |
| `src/types/models/maplestory.ts:98` | `MapleStoryCharacterData` shape |
| `src/services/api/pagination.ts:67` | `fetchAll` — cosmetics enumeration |
| `src/services/api/items.service.ts` | `itemsService.searchItems(filters)` + `ItemSearchFilters`/`ItemSearchPage` |
| `src/services/api/item-strings.service.ts` | `itemStringsService.getItemString(id)` |
| `src/lib/hooks/api/useItemStrings.ts:5-8` | `itemStringKeys.byId` — cache key shape `useItemNames` must share |
| `src/lib/hooks/api/useMaps.ts` | `useMap`, `useMapsByName`; `MapData {id, attributes:{name, streetName}}` |
| `src/lib/hooks/useSkillData.ts` | Skill name/icon for kit skill rows (`{name, iconUrl}`) |
| `src/lib/utils/asset-url.ts:5` | `getAssetIconUrl(tenant, region, major, minor, "item"\|"skill", id)` |
| `src/components/features/characters/EquipmentPanel.tsx:26-50` | Canonical render slot ids (top -5, pants -6, shoes -7, weapon -11) |
| `src/lib/hooks/api/useTenants.ts` | `useTenantConfiguration(id)`, `useUpdateTenantConfiguration().mutate({tenant, updates}, {onSuccess,onError})` |
| `src/lib/hooks/api/useTemplates.ts` | `useTemplate(id)`, `useUpdateTemplate().mutate({id, updates}, ...)` |
| `src/context/tenant-context.tsx` | `useTenant().activeTenant = {id, attributes:{name, region, majorVersion, minorVersion}} \| null` |
| `src/components/common/` | `ErrorDisplay` (data-testid="error-display"), `FormSkeleton`, `EmptyState` (data-testid="empty-state") |
| `src/components/ui/` | Existing primitives: dialog, dropdown-menu, alert-dialog, select, switch, tooltip, skeleton — popover is NEW |
| `services/atlas-data/atlas.com/data/item/filter.go` | Subcategory token source of truth (re-verify tokens if it changed) |
| `services/atlas-configurations/seed-data/` | Seed template JSON — reference only, do not modify |

## Dependencies & environment

- Node 22 via nvm: `source ~/.nvm/nvm.sh && nvm use 22` before ANY npm command;
  run from `services/atlas-ui`.
- New npm dep: `@radix-ui/react-popover` (Task 6). Nothing else.
- Tests: Vitest + Testing Library, `vi.*` mocks only (61 legacy Jest-style
  files exist but are excluded from `tsc -b`; new tests are NOT excluded —
  they must type-check). House mocking style:
  `src/components/features/characters/__tests__/ApplyPresetDialog.test.tsx`.
- Gates: `npm run test`, `npm run lint` (no NEW errors vs baseline),
  `npm run build` in atlas-ui; `tools/lint.sh --check` at repo root
  (run fix mode `tools/lint.sh` first).

## Task order & coupling

Tasks 1–5 are pure/data-layer foundations (parallelizable). Task 6 (popover +
combobox) unblocks 7 and 13. Task 11 unblocks 12. Task 15 assembles everything;
Task 16 wires pages + deletes old forms; Task 17 is gates + visual tuning +
code review. Every task = red test → implement → green → commit.

## Gotchas

- `tsc -b` type-checks new test files — never commit a test that doesn't compile.
- Radix Select/DropdownMenu under jsdom may need
  `Element.prototype.hasPointerCapture`/`scrollIntoView` stubs (Task 10 note).
- `DropdownMenuItem` `variant` prop and `FormSkeleton` testid passthrough:
  check the local component files first; fallbacks are spelled out in the plan.
- The editor seeds its reducer once (`loaded` flag) — post-save invalidation
  refetches must not clobber the working copy.
- Preview picks are UI-only; `removePoolEntry` clamps them; `discard` resets
  them (they may index removed entries).
- Unresolvable map/item/skill ids degrade to placeholder + id — never block
  editing (atlas-data coverage varies by version).
- PATCH must spread `...attributes.characters` (presets survival is asserted
  by the Task 16 wrapper tests).
