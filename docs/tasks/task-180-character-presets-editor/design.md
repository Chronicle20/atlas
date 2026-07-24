# Character Presets Editor — Design

Version: v1
Status: Approved
Created: 2026-07-20
Consumes: [`prd.md`](prd.md) (approved), [`prototype.html`](prototype.html) (visual source of truth, "A · Card library → editor" tab)

---

## 1. Summary

Replace the two wholesale-duplicated raw-ID preset forms
(`tenants-character-presets-form.tsx`, `templates-character-presets-form.tsx`)
with **one shared, adapter-parameterized editor** under
`src/components/features/characters/presets/`, structurally mirroring the
task-177 templates editor but with a **card-library landing view** in front of
the focused per-preset editor. Zero backend changes, zero shell changes.

The task-177 dependency (PRD Open Q2) is **resolved**: PR #1028
(`feat(task-177): shared character-templates visual editor`) has landed on
`main` and is an ancestor of this branch's `HEAD`. Every shared module the PRD
names already exists at
`src/components/features/characters/templates/`. This design **builds directly
on those modules** and, where reuse requires a small, additive, backward-
compatible change to a task-177 file, says so explicitly (§7, §8).

## 2. Guiding constraints (from the PRD, made concrete)

- **State via `useReducer` + baseline snapshot, NOT react-hook-form** — the
  task-177 "D1" pattern (`editorState.ts`). One working array for the whole
  `presets` list; dirty = deep compare vs. baseline; free switching between
  presets never loses edits.
- **One PATCH of the full configuration** through the context's mutation,
  spreading `attributes.characters` so the sibling `templates` array survives.
- **Save/Discard surfaced through the existing shared detail action bar**
  (`useRegisterDetailActionBar` → `SaveBar`, rendered once by
  `TenantDetailLayout` / `TemplateDetailLayout`). This *is* the "sticky save
  bar" FR-9.3 describes; reusing it is what keeps "zero shell changes" true.
- **Unresolvable ids degrade, never block** — placeholder icon + numeric id,
  still editable/removable. The backend is the validator of record.
- **Every request carries tenant/region/version headers** via the existing API
  client; light + dark tokens only; sprites `image-rendering: pixelated`.

## 3. Resolved open questions

| PRD Q | Decision | Rationale |
|---|---|---|
| Q1 Skin range | **0–9** with rendered previews | Matches task-177's `AppearanceBrowserDialog` `SKIN_IDS` exactly; "keep both consistent" per the PRD. Bad skins render the default body — non-blocking. |
| Q2 task-177 ordering | **Build on landed #1028** | Verified ancestor of `HEAD`; all shared modules present. No extraction/duplication needed. |
| Q3 Applying a dirty preset | **(a) warn "applies last saved" + offer save-first** | Apply reads persisted config server-side; (a) is the PRD default and avoids a hard gate. A preset with no persisted `id` cannot be applied (disabled with a hint). |
| Q4 jobId map coverage | **Curated seed-data ids + common advances; everything else → `Job <id>`** | Backend authoritative. Map isolated in `presetJobs.ts` for easy extension. |

## 4. Architecture overview

A new sibling folder `presets/` next to `templates/`. The editor renders **two
views inside the same content area**, switched by the presence of a resolvable
`?preset=` param:

```
PresetsEditorAdapter (from page wrapper)
        │
CharacterPresetsEditor          ← owns useReducer state + URL sync + action-bar registration
        │
        ├── (no ?preset=)  PresetLibrary
        │                     ├── LibraryToolbar   (search, tag filter, + New)
        │                     ├── PresetCard[]      (render, badges, tags, dirty-dot, hover quick-actions)
        │                     └── NewPresetCard
        │
        └── (?preset=<id|key>)  PresetEditor         ← two-column: sections | sticky PresetPreviewCard
                                   ├── PresetActionsMenu  (kebab: Duplicate / Apply… / Remove)
                                   ├── IdentitySection
                                   ├── ClassAppearanceSection   (named job picker + single-select appearance)
                                   ├── SpawnProgressionSection  (MapPicker + level/GM/meso)
                                   ├── BaseStatsSection
                                   ├── EquipmentSection         (avg-stats toggle rows)
                                   ├── InventorySection         (quantity rows)
                                   ├── SkillsSection            (numeric-id add + lookup)
                                   └── PresetPreviewCard
        │
        └── Apply flow: AccountPickerDialog → ApplyPresetDialog (initialPresetId)
```

**Why a folder, not a rewrite of `templates/`:** presets and templates share
*building blocks* (renderers, pickers, thumbnails) but not *shape* — templates
store per-dimension **pools** keyed by `(jobIndex, subJobIndex, gender)`;
presets store a single concrete character with a stable backend `id`. Forcing
one component over both shapes would tangle the boundaries the PRD deliberately
separates (task-177 vs task-180). Shared logic is reused at the leaf-module
level, not the container level.

## 5. State model — `presetEditorState.ts`

Mirrors `editorState.ts` (reducer + `initial…` + `isDirty` + clone/normalize
helpers) with three deltas driven by presets having a **stable id** and an
**id-addressable library**:

1. **Stable client key per working preset.** Selection and `?preset=` are
   **id/key-addressed**, not index-addressed (task-177 used `?tpl=<index>`
   because templates have no id). Each working preset carries a UI-only
   `key: string`, assigned on `load` (`p.id ?? "local-<n>"` via a reducer
   counter) and on `add`/`duplicate` (`"local-<n>"`). The `key` is **never
   persisted, never sent in the PATCH, never part of the dirty compare** —
   `isDirty` and `save` project each working preset to `{ id?, attributes }`
   before comparing/emitting. This is the presets analogue of task-177's
   separate `previewPicks` map, but attached per-object so a card library can
   address rows without index bookkeeping.

2. **Selection by key.** `selectedKey: string | null` (null = library view).
   `add`/`duplicate` set `selectedKey` to the new row's key; `remove` clears to
   `null` (back to library) — matching the prototype's flow of dropping to the
   library after deleting the open preset.

3. **Preview-override state per preset** (skin/hair/haircolor/face live picks
   before they're written). Because preset appearance fields are **single
   values already on the preset object**, appearance edits write **directly to
   `attributes`** (e.g. `setAppearance("face", id)`), so — unlike task-177 —
   there is **no separate `previewPicks` layer**. The live preview composites
   straight from `attributes`. This removes an entire class of index-remap bugs
   the template reducer carries (`remapPicksAfterRemove`).

Reducer actions: `load`, `select(key|null)`, `addPreset`, `duplicatePreset(key)`,
`removePreset(key)`, `setField(path,value)` (identity/spawn/stats/class/gender/
appearance — a small typed union, not free `any`), `addEquip/removeEquip/
setEquipAvg`, `addInventory/removeInventory/setInventoryQty`, `addSkill/
removeSkill/setSkillLevel`, `addTag/removeTag`, `discard`, `savedOk`.

`DEFAULT_PRESET_ATTRIBUTES` (currently inlined in both deleted forms) moves here
as the single source for +New / normalization defaults (FR-9.4):
`name "New preset", jobId 0, gender 0, face 20000, hair 30030, hairColor 0,
skinColor 0, mapId 0, level 1, meso 0, gm 0, stats {4,4,4,4,50,5}`, empty
equipment/inventory/skills. `normalizePreset` fills missing fields (defends
against partial seed JSON, same as the deleted forms' spread).

## 6. URL sync & deep-linking (`?preset=`)

Follows task-177's proven two-effect structure, re-expressed for keys:

- **Seed-once effect** (`deps: [adapter.presets, state.loaded]`): first time the
  adapter delivers data, `dispatch({type:"load"})`. The `loaded` guard makes the
  reducer authoritative afterward, so a post-save invalidation refetch never
  clobbers the working copy.
- **Deep-link-on-load effect** (`deps: [state.loaded]`): read `?preset=`; resolve
  against `id` **then** `key`; if it matches a working preset, `select(key)`;
  otherwise leave `selectedKey = null` (library). Unresolvable/stale ids fall
  back to the library **without error** (FR-10). Runs once on load only.
- **`syncSelection(key|null)`** helper owns all URL writes with
  `{ replace: true }`: it sets `?preset=<id ?? key>` when a preset is selected
  and **deletes** the param when returning to the library. Every internal
  mutation (`select`, `add`, `duplicate`, `remove`, `discard`) calls it with the
  reducer's own post-mutation selection — the handler owns URL agreement, so
  there is no length-watching effect to race the router (the exact bug the
  task-177 comments warn about).

Freshly-saved new/duplicated presets keep their `local-<n>` key in the URL for
the session (the `loaded` guard means we don't reseed with the backend-assigned
id until a full reload) — acceptable and identical in spirit to task-177.

## 7. Live preview — `presetLoadout.ts` + `PresetPreviewCard.tsx`

`buildPresetLoadout(attrs): CharacterLoadout` — simpler than templates because
every value is singular:

```
skin  = attrs.skinColor
hair  = attrs.hair + attrs.hairColor        (v83 convention: base + color digit)
face  = attrs.face
gender = attrs.gender
equipment = filterEquipment(
             mapEach(attrs.equipment, e => getDefaultSlotForTemplateId(e.templateId)))
```

- **Worn-slot placement is solved by existing code**:
  `getDefaultSlotForTemplateId` (`lib/utils/maplestory.ts`) maps a template id →
  its canonical negative slot for **all** equip classifications (hat, face/eye
  acc, top/overall, bottom, shoes, gloves, cape, shield, weapon 130–159, …),
  and `filterEquipment` (`characterRender.service.ts`) drops cash/pet/mount
  slots. So the preset preview is *richer* than the 4-slot template preview:
  every placeable worn item shows, later same-slot items overwrite earlier
  (map semantics). Items with no canonical slot (use/etc/cash) are skipped for
  the render but still listed in the Equipment section and the worn strip.
- **Empty loadout fallback** → `RENDER_DEFAULT_SKIN/HAIR/FACE` (0 / 30030 /
  20000), reusing the constants already exported by
  `templates/previewLoadout.ts`.
- **Render** via `useCharacterImage(character, {stance:"stand1", resize:2})` —
  identical machinery/skeleton/error/retry to `templates/PreviewCard.tsx`.
  `PresetPreviewCard` is a thin variant of `PreviewCard`: same stage/worn-strip/
  caption, sourcing the loadout from `buildPresetLoadout` and the worn strip
  from the placed slots. It re-renders on any appearance/equipment/gender edit
  because those write straight to `attributes` (§5.3). On narrow (`<~900px`) it
  stacks above the sections (`order-1 lg:order-2`, same grid idiom).

## 8. Appearance browsing — reuse decision

The appearance thumbnails (`AppearanceThumb`) are reused **verbatim**. The
cosmetics **browser** needs single-select ("replace", not "append to pool")
semantics, and today `AppearanceBrowserDialog` is coupled to `CharacterTemplate`
(`template[dimension]` pools + `buildVariantLoadout(template, picks, …)` +
`onAdd`).

**Chosen approach — generalize the shared dialog with additive, backward-
compatible props (Option A).** Introduce a small seam:

- Replace the `template`/`picks`/`onAdd` inputs with:
  `gender: number`, `variantLoadout: (dimension, id) => CharacterLoadout`,
  `selectedId?: number`, `markedIds?: number[]`, `onSelect: (id) => void`,
  and `selectMode: "add" | "replace"` (default `"add"`).
- Templates keep current behavior by passing `template.gender`,
  `(dim,id)=>buildVariantLoadout(template,picks,dim,id)`,
  `markedIds={template[dimension]}`, `onSelect={onAdd}`, `selectMode="add"`.
- Presets pass `attrs.gender`, a `buildPresetVariantLoadout(attrs,dim,id)`
  closure (built on `buildPresetLoadout` with one dimension substituted, same
  v83 hair math), `selectedId={current value}`, `onSelect={setAppearance}`,
  `selectMode="replace"` (thumb shows a selection ring on `selectedId`, closes
  on pick). The 0–9 skin / 0–7 hair-color candidate lists already live in the
  dialog and are shared.

*Rationale & tradeoff:* the PRD explicitly asks to reuse "the task-177 cosmetics
browser **in single-select mode**", the project ethos strongly favors reuse over
duplication ("one shared editor so fixes land in both"), and the change is
purely additive — templates' existing props map onto the new seam with
`selectMode="add"` defaults, and task-177's browser tests guard the regression
surface. **Rejected Option B** (a copy-pasted `PresetCosmeticsBrowser`) — ~150
duplicated lines drift out of sync, contradicting the reason this task exists.
**Rejected Option C** (feed a synthetic `CharacterTemplate` to the untouched
dialog) — abuses the pool/marked semantics (current pick would render as a
disabled "marked" thumb, not a selected one) and leaks a fake shape through the
API. If, during planning, the prop refactor proves more invasive than the
task-177 tests can cheaply cover, fall back to Option B for the browser only —
but Option A is the plan of record.

## 9. Editor sections — reuse map

| Section | Reuse | Preset-specific delta |
|---|---|---|
| **Identity** | tag chip add/remove idiom | `name` (≤64, required), `defaultName`, `description` (≤512), `tags` — plain inputs bound to reducer, `presetSchema` validates on save. |
| **Class & appearance** | `AppearanceThumb`, generalized `AppearanceBrowserDialog` (§8) | **new `presetJobs.ts`**: curated `jobId→name` map + searchable select + "Advanced" numeric entry; unknown → `Job <id>`. Gender M/F select. Face/hair = current thumb + `+`; hair-color 0–7 thumbs; skin 0–9 thumbs. |
| **Spawn & progression** | **`MapPicker({value,onChange})` verbatim** | level (1–250), GM (≥0), meso (≥0) numeric steppers. |
| **Base stats** | numeric stepper idiom | `str/dex/int/luk/hp/mp` — written verbatim to the created character (copy notes this). |
| **Equipment** | `ItemRow` (+ optional `trailing` slot), `ItemSearchCombobox`, `POOL_SEARCH_CONFIGS` | flat "Worn items" list of `{templateId, useAverageStats}`; per-row **avg-stats** `Switch`; add via a subcategory selector (Tops/Bottoms/Shoes/Weapons→pool key, `All`→`items`) feeding `ItemSearchCombobox`, plus a manual-id numeric fallback. |
| **Inventory** | `ItemRow` (+ `trailing`), `ItemSearchCombobox` (`items`) | rows of `{templateId, quantity}`; per-row quantity stepper (min 1); empty copy "No granted items." |
| **Skills** | `useSkillData`, the `SkillRow` icon/name idiom | rows of `{skillId, level}`; per-row level stepper (min 1); add = numeric-id input with name/icon lookup on entry; empty copy "This preset grants no skills." No skill browser (FR-8.3). |

**One additive task-177 change:** `ItemRow` gains an optional
`trailing?: ReactNode` rendered between the id and the remove ×, so the
avg-stats toggle and quantity stepper attach without a second row component.
Backward-compatible (undefined by default); the templates `EquipmentPoolSection`
and `StartingKitSection` are unaffected. If the maintainers prefer to leave
`ItemRow` untouched, the fallback is a `PresetItemRow` that composes the same
`useItemName` + `getAssetIconUrl` lookups — noted, but the additive prop is the
plan of record for the same reuse reason as §8.

## 10. Apply-to-account flow & context gating

**Key structural fact:** `ApplyPresetDialog` requires a live `Tenant`
(`useTenantConfiguration(tenant.id)`, serviceable worlds, `useNameValidity`,
`useCreateCharacterFromPreset`) and `useAccountSearch(tenant, …)` requires the
same. **Templates are configuration templates, not deployed tenants — there is
no `Tenant`, no account service, no worlds.** Therefore **apply-to-account is
available only in the tenant context.**

The adapter carries an **optional apply capability**:

```ts
interface PresetsEditorAdapter {
  presets: CharacterPreset[] | undefined;
  isLoading: boolean;
  error: Error | null;
  save: (presets: CharacterPreset[], onSuccess: () => void) => void;
  isSaving: boolean;
  apply?: { tenant: Tenant };   // present only for the tenant page
}
```

- **Tenant page** supplies `apply: { tenant }`, sourced from `useTenant(id)`
  (verified `export type Tenant = TenantBasic`, so `useTenant` returns a usable
  `Tenant`; its `attributes.region/majorVersion/minorVersion` and `id` are
  exactly what `ApplyPresetDialog` reads). Cards render the **Apply** hover
  quick-action and the kebab **Apply…** item.
- **Template page** omits `apply`; those affordances are hidden. Duplicate,
  remove, edit, live preview all work identically in both contexts.
- **Flow:** card Apply → `AccountPickerDialog` (new): debounced name search via
  `useAccountSearch`, selectable result list with empty/loading/error states →
  on account pick, opens `ApplyPresetDialog` with `accountId` **and a new
  optional `initialPresetId`** (FR-3.2) pre-scoping it to the card's preset.
- **`ApplyPresetDialog` change (UI-only):** add `initialPresetId?: string`;
  when set, seed `defaultValues.presetId` and skip the internal preset grid (or
  pre-select it). It continues to read presets from **saved** config → apply
  uses the last-saved version (FR-3.3). If the card's preset is dirty, warn and
  offer save-first; a preset with no persisted `id` disables Apply with a hint
  ("Save this preset before applying").

## 11. Page wrappers & deletions

- **Delete** `src/pages/tenants-character-presets-form.tsx` and
  `templates-character-presets-form.tsx` (and their inline
  `DEFAULT_PRESET_ATTRIBUTES` / `CharacterPreset`-shaped locals).
- **Rewrite** `TenantsCharacterPresetsPage.tsx` and
  `TemplatesCharacterPresetsPage.tsx` to build the adapter and render
  `<CharacterPresetsEditor adapter={…} />` inside the existing
  `TenantDetailLayout` / `TemplateDetailLayout` — structurally identical to
  `TenantsCharacterTemplatesPage` / `TemplatesCharacterTemplatesPage`:
  - Tenant: `useTenantConfiguration` + `useUpdateTenantConfiguration`, PATCH
    `{ characters: { ...tenant.attributes.characters, presets } }`; `apply`
    from `useTenant(id)`.
  - Template: `useTemplate` + `useUpdateTemplate`, PATCH
    `{ characters: { ...template.attributes.characters, presets } }`; no `apply`.
- **Field-level API error mapping** (the `meta.path` → `presets[<id>].<field>`
  logic in the deleted forms) moves into the editor's `save` error handler,
  routing each error to the offending preset/field where the path resolves.
- Types come **only** from `@/types/models/template`
  (`CharacterPreset`, `CharacterPresetAttributes`, nested entry types); no
  inline preset shape survives. `character-presets.schema.ts` (`presetSchema`)
  is reused for validation.

## 12. File inventory

**New — `src/components/features/characters/presets/`**
`CharacterPresetsEditor.tsx`, `presetEditorState.ts`, `presetJobs.ts`,
`presetLoadout.ts`, `PresetLibrary.tsx`, `PresetCard.tsx`, `LibraryToolbar.tsx`,
`NewPresetCard.tsx`, `PresetEditor.tsx`, `PresetActionsMenu.tsx`,
`IdentitySection.tsx`, `ClassAppearanceSection.tsx`,
`SpawnProgressionSection.tsx`, `BaseStatsSection.tsx`, `EquipmentSection.tsx`,
`InventorySection.tsx`, `SkillsSection.tsx`, `PresetPreviewCard.tsx`,
`AccountPickerDialog.tsx`, plus colocated `__tests__/`.

**Modified (additive, backward-compatible)**
`templates/AppearanceBrowserDialog.tsx` (generalized props, §8),
`templates/ItemRow.tsx` (optional `trailing`, §9),
`characters/ApplyPresetDialog.tsx` (`initialPresetId`, §10),
`pages/TenantsCharacterPresetsPage.tsx`, `pages/TemplatesCharacterPresetsPage.tsx`.

**Deleted**
`pages/tenants-character-presets-form.tsx`,
`pages/templates-character-presets-form.tsx`.

**Reused unchanged**
`templates/AppearanceThumb.tsx`, `templates/MapPicker.tsx`,
`templates/ItemSearchCombobox.tsx`, `templates/poolSearchConfig.ts`,
`lib/utils/maplestory.ts` (`getDefaultSlotForTemplateId`,
`synthesizeEquippedAssetsFromTemplateIds`),
`services/api/characterRender.service.ts` (`filterEquipment`, `generateCharacterUrl`,
`isFemaleCosmeticId`), `lib/hooks/useCharacterImage`, `lib/hooks/useSkillData`,
`lib/hooks/api/useItemStrings` + `useItemNames`, `lib/hooks/api/useCosmetics`,
`lib/hooks/api/useMaps`, `lib/hooks/api/useAccounts` (`useAccountSearch`),
`components/DetailActionBarContext` (`useRegisterDetailActionBar`),
`components/common` (`EmptyState`, `ErrorDisplay`, `FormSkeleton`),
shadcn primitives.

## 13. Testing (Vitest + Testing Library)

Per FR-8/§8 of the PRD, new test files must type-check under `tsc -b`. Coverage:

- **Reducer (`presetEditorState`)** — pure unit tests: load assigns keys, dirty
  ignores `key`, add/duplicate/remove selection + key stability, appearance/
  stat/equip/inventory/skill edits, discard restores baseline, `savedOk`
  rebaselines. (Highest-value, fastest — the state module is the correctness
  core.)
- **Library** — search matches name/description/tags case-insensitively;
  single-select tag filter; dirty-dot derives from per-preset diff; `+ New`
  card and toolbar button both append+navigate; empty state.
- **Card quick-actions** — Duplicate appends a deep copy + marks dirty; Apply
  opens `AccountPickerDialog` (tenant adapter) and is **absent** under the
  template adapter.
- **URL sync** — `?preset=<id>` opens the focused editor; absent/unresolvable →
  library without error; internal mutations keep the param in agreement.
- **Sections** — each section's edit semantics (job named-picker + advanced
  numeric, single-select appearance replace, MapPicker wiring, avg-stats toggle,
  quantity/level steppers, numeric-id skill add + lookup, unresolved-id
  placeholder rows).
- **Apply handoff** — `AccountPickerDialog` search → select → `ApplyPresetDialog`
  opens scoped via `initialPresetId`; dirty-preset warning; unsaved-preset
  Apply disabled.
- **Both page adapters** — save spreads `characters` so the sibling `templates`
  array **survives** the PATCH (assert the mutation payload) in both contexts;
  identical loading-skeleton / `ErrorDisplay` states.

## 14. Risks & mitigations

- **task-177 shared-file edits regress templates.** Mitigation: both edits
  (§8 browser props, §9 `ItemRow.trailing`) are additive with defaults that
  preserve existing call sites; task-177's tests run green as a gate; the
  documented Option-B/`PresetItemRow` fallbacks exist if a refactor proves
  costly.
- **Apply-flow `Tenant` availability.** Verified resolvable via `useTenant(id)`
  (`Tenant = TenantBasic`); no new endpoint.
- **`?preset=` key vs backend-id after save.** Session keeps the `local-<n>`
  key (loaded-guard, no reseed) — same accepted limitation as task-177's
  post-save index; a full reload reconciles to the backend id.
- **Preview render volume.** Library cards each request one composite, cached by
  loadout hash in atlas-renders; `useCharacterImage` bounds concurrency;
  browser grids page at ≤24 lazy — unchanged from task-177's live-verified
  budget.

## 15. Alternatives considered (and rejected)

- **Extend `CharacterTemplatesEditor` to also render presets** — rejected:
  pool-vs-single shapes and the id-vs-index addressing diverge enough that one
  container would tangle both; the PRD separates the features by design.
- **react-hook-form + `useFieldArray`** (what the deleted forms used) — rejected
  by the PRD in favor of the D1 reducer pattern: free preset-to-preset switching
  without losing edits, one array-level dirty/save, and a clean deep-compare
  baseline that RHF's field-array identity churn fights.
- **Index-addressed `?preset=<index>`** (copy task-177's `?tpl=`) — rejected:
  presets have a stable backend id; index links break on reorder/removal and
  aren't shareable. Id/key addressing is the natural fit and erases the
  index-remap bookkeeping.
- **Client-side sort/reorder of the library** — out of scope (FR-2.5); card
  order is persisted array order; new/duplicated append.
