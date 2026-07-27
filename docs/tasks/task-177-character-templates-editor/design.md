# Character Templates Editor — Design

Task: task-177-character-templates-editor
Inputs: [prd.md](prd.md) (approved), [prototype.html](prototype.html) (visual source of truth)
Scope: `services/atlas-ui` only. Zero backend changes, zero layout/shell changes.

---

## 1. Architecture at a glance

One shared editor component owns the entire "Character Templates" section. Both
page components render it inside their existing layouts and supply a small
adapter that binds it to the right data context:

```
TenantsCharacterTemplatesPage ─┐
                               ├─> CharacterTemplatesEditor (shared)
TemplatesCharacterTemplatesPage┘        │
                                        ├─ TemplateSelector        (segmented control + "+ New", ?tpl= sync)
                                        ├─ TemplateActionsMenu     (kebab: Duplicate / Remove-with-confirm)
                                        ├─ IdentitySection         (class, gender, MapPicker)
                                        ├─ AppearancePoolSection ×4 (faces, hairs, hairColors, skinColors)
                                        │    ├─ AppearanceThumb    (cropped composite, select/remove)
                                        │    └─ AppearanceBrowserDialog (paginated visual add flow)
                                        ├─ EquipmentPoolSection ×4 (tops, bottoms, shoes, weapons)
                                        │    └─ ItemSearchCombobox (server search + manual-id fallback)
                                        ├─ StartingKitSection      (items rows + skills rows)
                                        ├─ PreviewCard             (sticky live composite + worn-icon strip)
                                        └─ SaveBar                 (dirty indicator, Discard, Save)
```

### File map

New (all under `src/components/features/characters/templates/`):

| File | Responsibility |
|---|---|
| `CharacterTemplatesEditor.tsx` | Top-level: adapter prop, editor state hook, two-column grid, empty state, loading/error states |
| `editorState.ts` | Pure reducer + action creators + selectors for the whole form state (unit-testable without React) |
| `jobNames.ts` | jobIndex→world-name map (mirrors `JobFromIndex`), template label derivation incl. ordinal suffixes |
| `TemplateSelector.tsx` | Segmented control, `+ New`, `role="tablist"` semantics |
| `TemplateActionsMenu.tsx` | Kebab `DropdownMenu` + `AlertDialog` remove confirm |
| `IdentitySection.tsx` | Class select + Advanced numeric jobIndex/subJobIndex, gender select |
| `MapPicker.tsx` | Starting-map picker: name search via maps service, manual numeric entry, unresolvable-id hint |
| `AppearancePoolSection.tsx` | Generic pool group (thumbnail row, empty-pool warning, add button) parameterized per dimension |
| `AppearanceThumb.tsx` | One cropped composite thumbnail (button + accent ring + hover ×, id caption) |
| `AppearanceBrowserDialog.tsx` | Paginated visual browser for faces/hairs; digit variants for hairColors/skinColors |
| `EquipmentPoolSection.tsx` | Icon+name+id rows, `<n> options · player picks one` header, remove × |
| `ItemSearchCombobox.tsx` | Popover search input → result rows → "Use id <n>" fallback |
| `StartingKitSection.tsx` | Items rows (same row treatment) + skills rows (numeric add with name/icon lookup) |
| `PreviewCard.tsx` | Sticky live preview via `useCharacterImage`, worn-equipment icon strip, caption |
| `SaveBar.tsx` | Sticky save bar: dirty text, Discard (confirm), Save |
| `__tests__/` | Component + reducer tests (Vitest + Testing Library) |

New data layer:

| File | Responsibility |
|---|---|
| `src/services/api/cosmetics.service.ts` | `getAllFaceIds()` / `getAllHairIds()` — enumerate `/api/data/cosmetics/faces|hairs` via the existing `fetchAll` util (`services/api/pagination.ts`) |
| `src/lib/hooks/api/useCosmetics.ts` | `useFaceIds()` / `useHairIds()` React Query hooks, long `staleTime` (WZ data is static per tenant version) |
| `src/lib/hooks/api/useItemNames.ts` | `useItemNames(ids)` — batched per-id name resolution via `useQueries`, sharing the query key of the existing `useItemName` (`useItemStrings.ts`) so caches merge |

Modified:

- `src/pages/TenantsCharacterTemplatesPage.tsx`, `TemplatesCharacterTemplatesPage.tsx` — thin wrappers building the adapter and rendering the editor.
- `src/components/ui/popover.tsx` — new shadcn primitive (see D5).
- `package.json` — add `@radix-ui/react-popover`.

Deleted:

- `src/pages/tenants-character-templates-form.tsx`
- `src/pages/templates-character-templates-form.tsx` (removes the inline `CharacterTemplate` redeclaration — FR-1.3)

---

## 2. Key decisions

### D1 — Form state: plain reducer, not react-hook-form

**Decision: a `useReducer`-based state module (`editorState.ts`), not RHF.**

The PRD allows "react-hook-form + `useFieldArray` or equivalent controlled
state". Every mutation in this editor is a programmatic array operation
(add/remove/duplicate template, add/remove pool entry, set identity field) —
there are almost no free-typed validated inputs, which is the case RHF is built
for. The old forms already show the cost of forcing RHF here: `Path<T>` /
`PathValue<T>` generic gymnastics for what is `setValue(name, array)`
(`tenants-character-templates-form.tsx:252-277`).

State shape:

```ts
interface EditorState {
  templates: CharacterTemplate[];      // working copy
  baseline: CharacterTemplate[];       // last loaded/saved snapshot
  selectedIndex: number;               // mirrored to ?tpl=
  previewPicks: Record<number, PreviewPicks>; // per template index, UI-only
}
interface PreviewPicks { faceIdx: number; hairIdx: number; hairColorIdx: number; skinIdx: number; }
```

Actions: `load`, `select`, `addTemplate`, `duplicateTemplate`,
`removeTemplate`, `setIdentity`, `addPoolEntry`, `removePoolEntry`,
`setPreviewPick`, `discard`, `savedOk`. Dirty is derived:
`JSON.stringify(templates) !== JSON.stringify(baseline)` (arrays are tiny —
seed templates are ~10 numbers per pool — so deep compare per render is fine;
memoized with `useMemo`).

`removeTemplate`/`duplicateTemplate` also remap the `previewPicks` keys and
clamp `selectedIndex` (remove selects the nearest remaining index, duplicate
selects the copy — FR-3.1). `removePoolEntry` clamps the corresponding preview
pick index so the preview never points past the end of a pool.

Rejected alternative — RHF + `useFieldArray`: keeps repo-typical form
convention, but adds subscription/path indirection for zero validation
benefit, and the single-save-bar / free-switching model (FR-9.1) is exactly
"one array in state with a baseline snapshot", which a reducer expresses
directly. Numeric text entry (Advanced class fields, manual-id fallbacks) is
validated locally at the input site before dispatching, same as the old
`handleAdd` did.

### D2 — Dual-context wiring: a data adapter built by each page wrapper

**Decision: the editor takes one `adapter` prop; each page builds it from its
existing hooks.**

```ts
interface TemplatesEditorAdapter {
  templates: CharacterTemplate[] | undefined; // undefined while loading
  isLoading: boolean;
  error: Error | null;
  save: (templates: CharacterTemplate[]) => void; // fire-and-forget; result via callbacks
  isSaving: boolean;
}
```

- Tenant page: from `useTenantConfiguration(id)` /
  `useUpdateTenantConfiguration()`; `save` re-wraps as
  `{ characters: { ...tenant.attributes.characters, templates } }` — the exact
  PATCH shape the old form used (`tenants-character-templates-form.tsx:78-96`),
  preserving presets untouched.
- Template page: from `useTemplate(id)` / `useUpdateTemplate()`, spreading
  `attributes.characters` the same way.

`save` handles its own toasts inside the wrapper (`onSuccess`/`onError` on the
mutate call, as today) and the editor observes `isSaving` plus an
`onSaveSuccess` callback to reset the baseline (`savedOk` action).

Rejected alternative — the editor calls the hooks itself behind a
`context: "tenant" | "template"` prop: couples the shared component to both
data sources, makes it untestable without mocking two hook families, and
violates the "one clear interface" boundary. The adapter is 20 lines per page.

### D3 — Cosmetics data layer: enumerate ids once, resolve names per visible id

Grounded in the PRD's live verification: the item-strings **search index does
not cover faces/hairs**, so the browser must enumerate
`/api/data/cosmetics/faces` (536 rows) and `/hairs` (1520 rows) and resolve
names by id.

- `cosmetics.service.ts` uses `fetchAll` (`pagination.ts:67`) to pull the full
  id list in pages; the hook caches it with `staleTime: Infinity`-ish (WZ data
  changes only with re-ingest; tenant switch already clears all caches via
  `TenantProvider`). 1520 ids as numbers is trivially small.
- Gender filtering `(Math.floor(id/1000)) % 10 === 1 ⇒ female` reuses the
  documented convention from `resolveGender`
  (`characterRender.service.ts:58-62`) — extracted as a tiny
  `isFemaleCosmeticId(id)` helper next to it rather than duplicated.
- Names: `useItemNames(ids)` runs `useQueries` over only the **current
  browser page** (≤24 ids) with the same query key shape as `useItemName`, so
  individual lookups cache-share across the browser, pool rows, and future
  visits. In-browser name search filters the already-fetched id list
  client-side only after names for the current page resolve — matching the
  PRD's "filters the already-fetched page set".

### D4 — Equipment search: server single-subcategory when possible, client subcategory-set filter otherwise

`parseFilters` accepts exactly **one** subcategory token — no comma lists
(`services/atlas-data/atlas.com/data/item/filter.go:112-131`). Pool filter
configs therefore come in two flavors:

| Pool | Strategy |
|---|---|
| bottoms | server `filter[subcategory]=bottom` |
| shoes | server `filter[subcategory]=shoes` |
| tops | server `filter[compartment]=equipment` + search, client-filter `subcategory ∈ {top, overall}` (Aran's 1042167 is an overall) |
| weapons | server `filter[compartment]=equipment` + search, client-filter subcategory ∈ the 16 weapon tokens listed in `filter.go:53-60` (`one-handed-sword` … `gun`, excluding `pet-equip`) |
| kit items | no compartment filter — search all compartments |

The client-side set filter works because `searchItems` result rows already
carry `subcategory` (`items.service.ts` `ItemSearchResult`). To keep filtered
pages from looking empty, the combobox requests `pageSize: 50` and exposes
"load more" (next page) rather than numbered pagination. Exact subcategory
token values are re-verified against `filter.go` at implementation time, in a
single `poolSearchConfig.ts` constant.

Rejected alternative — N parallel queries, one per weapon subcategory:
16 requests per keystroke; strictly worse.

### D5 — Combobox primitive: add shadcn Popover, skip cmdk

The UI kit has no `popover.tsx` or `command.tsx` and no `cmdk`/
`@radix-ui/react-popover` deps. The add-flow combobox needs an anchored
floating panel with an input and a result list.

**Decision: add the standard shadcn `popover.tsx` (one new dep,
`@radix-ui/react-popover`) and build `ItemSearchCombobox` as
Popover + `Input` + plain listbox rows.** Filtering is server-side (debounced
`searchItems`), so cmdk's client-side matcher adds nothing; keyboard handling
(arrow/enter/escape) is small enough to own with `role="listbox"` semantics.

Rejected alternatives: (a) full shadcn Command/cmdk — extra dep whose core
feature is unused; (b) Dialog-based search — heavier interaction for the
"type a name, click a row" flow the PRD calls a combobox; Dialog stays the
right tool for the *visual browser* (D6) where a large grid needs room.

`MapPicker` reuses the same popover-combobox shell with `useMapsByName` /
`useMaps` as the backend and `<name> · <streetName> · <id>` rows; manual
numeric entry is the same fallback row pattern ("Use id <n>"). Unresolvable
map ids render `Map <id>` + warning hint via a `useMap(id)` lookup that
treats 404 as a hint state, not an error (FR-4.3).

### D6 — Thumbnails and the visual browser

Every appearance thumbnail is a **full character composite cropped to the head
region with CSS** — no new render modes:

- URL from `generateCharacterUrl` (`characterRender.service.ts:98`) with the
  template's current preview picks, one dimension substituted (face thumb
  varies face, etc.), `stance: "stand1"`, `resize: 2` → 192×256 PNG.
- `AppearanceThumb` renders a fixed ~76px square `overflow-hidden` div with
  the image absolutely positioned at the prototype's offset
  (`background-position: -74px -70px` equivalent; final offset tuned during
  implementation), `image-rendering: pixelated`, `loading="lazy"`.
- Because the loadout differs only in one id, atlas-renders' loadout-hash
  cache makes repeat views cheap; grid pages are capped at 24 (FR-5.4,
  NFR render volume). Grid images use plain lazy `<img>` (per UI CLAUDE.md
  image guidance); only the big preview uses `useCharacterImage` (D8).
- Missing composites (e.g. hair-color variant absent from WZ) show the
  image `onerror` placeholder state inside the thumb — the PRD's accepted
  signal for open question 3.

`AppearanceBrowserDialog` covers all four dimensions with one component and a
per-dimension source:

| Dimension | Candidate ids | Names |
|---|---|---|
| faces | `useFaceIds()` (gender-filtered, toggle to show all) | `useItemNames` per page |
| hairs | `useHairIds()` (same) | same |
| hairColors | digits 0–7 composited as `baseHair + digit` (v83 convention) | none — digit caption |
| skinColors | 0–9 rendered on current character (PRD open question 1 posture: offer with previews, operator judges) | none — id caption |

Already-in-pool ids render marked + disabled (FR-5.4).

### D7 — URL sync (`?tpl=`)

`CharacterTemplatesEditor` owns the sync via `useSearchParams` (pattern
already used by `GuildsPage`/`CharactersPage`):

- On mount / templates load: parse `tpl`; non-numeric or out-of-range clamps
  to 0 and is written back with `{ replace: true }`.
- On select/add/duplicate/remove: dispatch first, then
  `setSearchParams({ tpl: String(newIndex) }, { replace: true })` — replace
  avoids history spam so Back leaves the page rather than stepping through
  selections.
- Selection changes never touch `templates`, so switching can't lose edits
  (FR-2.4).

### D8 — Live preview loadout

`PreviewCard` builds a `MapleStoryCharacterData` for `useCharacterImage`
(skeleton/error/retry for free, per FR-8.2):

- skin = pool[skinIdx], hair = pool[hairIdx] + pool[hairColorIdx] digit,
  face = pool[faceIdx], gender = template gender.
- equipment record uses the verified render slot ids
  (`EquipmentPanel.tsx:26-50`): top → `-5`, bottom → `-6`, shoes → `-7`,
  weapon → `-11`, each the **first entry** of its pool; slots with empty pools
  are omitted.
- Empty appearance pools fall back to render defaults (skin 0, hair 30030,
  face 20000 — the same "default clothing" posture task-078 established) so a
  fresh `+ New` template still previews a body; the empty-pool warning
  (FR-9.4) is the operator's signal, not a broken image.
- Re-render triggers are automatic: the loadout is derived state, so any
  qualifying edit (FR-8.3) changes the hash → new query key.

The worn-equipment icon strip uses `getAssetIconUrl(..., "item", id)`
(`asset-url.ts:5`) for the four first-of-pool ids, with name+id tooltip via
the existing `Tooltip` primitive.

Layout: two-column grid `minmax(0,1fr) 252px` inside the existing content
column; preview column `sticky top-<offset>`; below ~900px (`lg` breakpoint
approximation consistent with the shell's `lg:max-w-4xl`) it stacks between
selector and editor sections (FR-8.4).

---

## 3. Data flow

**Load** — adapter query resolves → `load` action seeds `templates` +
`baseline` (normalizing missing arrays to `[]`, exactly as the old `reset`
mapping did) → `?tpl=` parsed/clamped → preview picks default to index 0 per
dimension.

**Edit** — components dispatch actions; dirty derives from
templates≠baseline; empty-pool warnings derive per group
(mirroring the checks in `templatesService.validateTemplateConsistency`,
`templates.service.ts:438-453`, without calling it — it's template-context
only and re-fetches).

**Save** — SaveBar → `adapter.save(state.templates)` → wrapper PATCHes the
full configuration (single mutation, both contexts identical semantics) →
`onSaveSuccess` → `savedOk` resets baseline; failure keeps state, toast shows
the API error (FR-9.3).

**Discard** — confirm (AlertDialog) when dirty → `discard` restores
`templates = baseline` (preview picks reset too — they may index removed
entries).

---

## 4. Error handling

| Failure | Behavior |
|---|---|
| Config/template load error | Shared `ErrorDisplay` (identical in both contexts — FR-1.4); loading uses `FormSkeleton`/`CardSkeleton` from `components/common` |
| Item/skill id with no name or icon | Placeholder icon + `Unknown item`/`Unknown skill` + mono id; row stays removable (FR-6.3) |
| Map id unresolvable | `Map <id>` + warning hint, non-blocking (FR-4.3) |
| Composite render failure | Thumb: onerror placeholder tile; Preview: `useCharacterImage` error + retry state |
| Save failure | Error toast with API message; form state preserved |
| Cosmetics enumeration failure | Browser dialog shows `ErrorDisplay` with retry; pools themselves stay editable (manual flows unaffected) |

---

## 5. Testing

Reducer (`editorState.test.ts`, pure): add/duplicate/remove index & preview-pick
remapping, pool add/remove with pick clamping, dirty derivation, discard,
savedOk.

Component tests (Vitest + Testing Library, colocated `__tests__/`, `vi.*`
mocks per current conventions — not the legacy Jest style):

- `TemplateSelector`: labels from jobIndex/gender incl. ordinal suffix and
  `Job N` fallback; `?tpl=` deep link restore + clamp (render inside
  `MemoryRouter`); `+ New` appends and selects.
- `TemplateActionsMenu`: duplicate deep-copies (mutating the copy leaves the
  original untouched); remove confirms and selects nearest index.
- `EquipmentPoolSection` + `ItemSearchCombobox`: search rows add ids,
  manual "Use id" fallback, subcategory client-filter for tops/weapons,
  double-add prevention.
- `AppearanceBrowserDialog`: gender filter + show-all toggle, in-pool
  marking, page cap.
- Both page wrappers: adapter wiring smoke tests (mock hooks; assert PATCH
  payload preserves sibling `characters` keys, e.g. presets).

Gates (per PRD acceptance): `npm run test`, `npm run lint`, `npm run build`
in `services/atlas-ui` (nvm node 22), `tools/lint.sh --check` at repo root.
Note `tsc -b` type-checks new `*.test.ts(x)` files (repo memory: build
type-checks tests) — call sites and tests must land together.

---

## 6. Risks & notes

- **Render burst on first browser open** (24 fresh composites): bounded by
  page cap + lazy loading; atlas-renders persists by hash so it is one-time
  per tenant/loadout. No preloader needed for v1; `useCharacterImagePreloader`
  exists if grid scroll feels janky.
- **`fetchAll` on 1520 hairs** = ~16 pages of 100 sequentially on dialog
  open; acceptable (ids only), cached indefinitely afterward. If it feels
  slow, raising page size for the enumeration call is a one-line tune.
- **Crop offset** for 76px head thumbs is visual-tuning work against real
  renders; the prototype's `-74px/-70px @ resize=2` is the starting value.
- **Open questions** land as the PRD states them: skins 0–9 with previews,
  Advanced identity permits unmapped (jobIndex, subJobIndex) (backend is the
  validator of record), render-error state is the signal for missing
  hair-color variants.
