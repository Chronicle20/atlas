# Character Presets Editor — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-20
---

## 1. Overview

Character **presets** are the per-tenant library of ready-made characters an
operator can stamp onto an account: "Fresh Beginner", "Test Warrior", "CPQ
Party Member", "GM Admin". Unlike character *templates* (task-177) — which
define the pools of choices offered at the create screen for a
`(jobIndex, subJobIndex, gender)` key — a preset is a single concrete
character: one face, one hair, one hair color, one skin tone, one worn
loadout, a verbatim stat block, granted inventory, and granted skills. Presets
are consumed by `ApplyPresetDialog`
(`src/components/features/characters/ApplyPresetDialog.tsx`), which creates a
character from the chosen preset on a selected account.

The current admin UI for this data is two wholesale-duplicated form files
(`src/pages/tenants-character-presets-form.tsx` and
`src/pages/templates-character-presets-form.tsx`) that render every preset as a
collapsible accordion row full of raw numeric IDs — no icons, no names, no
character preview, and no way to tell one preset from another without expanding
it. This is the presets counterpart of the problem task-177 solved for
templates, and it is the last consumer of the `character-presets.schema.ts`
free-text-uint32 form that the task-037 follow-up TODO flagged for an
`<ItemPicker>`/`<SkillPicker>` replacement.

This task replaces both files with one shared **Option A** editor, following
the interactive prototype iterated with the user (committed alongside this PRD
as [`prototype.html`](prototype.html) — open it, "A · Card library → editor"
tab; it is the visual source of truth). The editor has two views inside the
existing detail-page shell with **zero layout and zero backend changes**:

1. A **card-library landing view** — presets as a searchable, tag-filterable
   gallery of rendered character cards.
2. A **focused per-preset editor** — drilled into from a card, with a sticky
   live preview and a sectioned form (Identity, Class & appearance, Spawn &
   progression, Base stats, Equipment, Inventory, Skills).

It reuses the task-177 building blocks: the cosmetics browser (in single-select
mode), the atlas-renders preview flow, `getAssetIconUrl` item/skill icons, and
item-strings/maps name resolution.

## 2. Goals

Primary goals:
- One shared editor component used by both the Tenant Details and Template
  Details "Character Presets" sections; delete the two duplicated forms.
- Make every ID visual and named: item icons + names for equipment/inventory,
  skill icons + names, rendered thumbnails for appearance picks, map name for
  the starting map, a job name for the class, and a live composited preview.
- A card-library landing view that reads as a catalog — each card answers
  "what character is this, and what is it for?" at a glance.
- Add the missing preset lifecycle affordances: create (+ New), duplicate,
  remove, and a card-level "apply to account" entry point.
- Deep-linkable preset selection via URL param (`?preset=<id>`).
- Fit entirely inside the existing `TenantDetailLayout` /
  `TemplateDetailLayout` shell — zero shell changes.
- Zero backend changes — every data need is served by existing endpoints.

Non-goals:
- Character Templates section (task-177, separate feature and pages).
- Any change to `DetailSidebar`, the detail layouts, routing shell, or nav.
- Backend/API changes in any Go service.
- Changes to seed preset JSON
  (`services/atlas-configurations/seed-data/templates/*.json`).
- The Option B (split workbench) and Option C (character sheet) prototype
  directions — explicitly dropped in favor of Option A.
- A skill *browser* (numeric-id-with-lookup add flow only, matching task-177
  FR-7).

## 3. User Stories

- As a tenant operator, I want to see each preset as a named, visual card
  (rendered character, class, level, tags, purpose) so that I can find the
  right preset without expanding raw-ID accordions.
- As a tenant operator, I want to search and tag-filter the preset library so
  that it stays navigable as it grows past a handful of entries.
- As a tenant operator, I want to click a card and edit that one preset in a
  focused view with a live preview, so that I see what a created character will
  look like before an operator applies it.
- As a tenant operator, I want to add a new preset and duplicate an existing
  one so that I can build variants without re-entering every field.
- As a tenant operator, I want to browse faces/hairs visually and pick items
  and skills by name so that I don't paste wrong IDs.
- As a tenant operator, I want to apply a preset to an account directly from
  its card so that I don't have to navigate to the account first.
- As a developer, I want one shared editor component so that fixes land in both
  the tenant and template contexts at once, and the two duplicated form files
  and the free-text-uint32 form are gone.

## 4. Functional Requirements

### FR-1 Shared component & adapter

1. New shared component (suggested:
   `src/components/features/characters/presets/CharacterPresetsEditor.tsx`
   plus colocated subcomponents) owns the entire section UI. It is
   parameterized by a small adapter interface supplied by each page wrapper —
   mirroring the task-177 `TemplatesEditorAdapter` shape:
   `{ presets, isLoading, error, save(presets, onSuccess), isSaving }`.
   - Tenant page: `useTenantConfiguration` / `useUpdateTenantConfiguration`,
     PATCHing `{ characters: { ...existing, presets } }` so the sibling
     `templates` array survives.
   - Template page: `useTemplate` / `useUpdateTemplate`, likewise spreading
     `attributes.characters` so `templates` survives.
2. `src/pages/tenants-character-presets-form.tsx` and
   `src/pages/templates-character-presets-form.tsx` are **deleted**; the two
   page components (`TenantsCharacterPresetsPage`,
   `TemplatesCharacterPresetsPage`) render the shared editor inside their
   existing layouts.
3. The editor works on the `CharacterPreset` shape imported from
   `@/types/models/template` (`CharacterPresetAttributes` and nested types);
   no inline re-declaration of the shape survives the refactor.
4. Loading uses skeleton components; errors use the shared `ErrorDisplay`; both
   contexts get identical loading/error states (today they may differ).

### FR-2 Card-library landing view

1. The default view (no `?preset=` param) renders the preset array as a
   responsive grid of cards. Each card shows:
   - A rendered character composite (atlas-renders, stand1) on a tinted stage.
   - The preset `name`.
   - A job badge (class name — FR-4.1) and `Lv <level>` (with `· GM <gm>`
     when `gm > 0`).
   - The `description`, clamped to two lines.
   - The preset's `tags` as chips.
   - A **dirty-dot** indicator when that preset has unsaved edits.
2. **Hover/focus quick-actions** on each card (do not require entering the
   editor):
   - **Duplicate** — deep-copies the preset, appends it, and (per FR-9) marks
     the array dirty. The copy is selectable in the library immediately.
   - **Apply to account** — opens the apply flow (FR-3).
3. **Toolbar** above the grid: a text search box (matches `name`,
   `description`, and `tags`, case-insensitively), a tag-filter chip row
   (`All` + one chip per distinct tag in the library; single-select),
   and a `+ New preset` button.
4. A **`+ New` card** at the end of the grid is an equivalent affordance to the
   toolbar button: appends a blank preset (FR-9.4 defaults) and navigates into
   its editor.
5. **Card order** is the persisted array order — no client reordering or
   sorting. Duplicated/new presets append to the end.
6. **Empty state** (no presets): explanatory copy + a prominent "Add preset"
   button (shared `EmptyState`).

### FR-3 Apply-to-account flow

1. The card "Apply to account" action opens an **account-picker dialog**: a
   name search backed by `useAccountSearch(tenant, namePattern)` (returns
   `Account[]`), rendered as a selectable result list. Debounced query;
   empty/error/loading states.
2. On selecting an account, the flow hands off to the existing
   `ApplyPresetDialog` (`{ tenant, accountId, open, onOpenChange }`),
   pre-scoped to the chosen preset. `ApplyPresetDialog` today selects a preset
   from its own internal list; this task adds an **optional `initialPresetId`
   prop** so it opens on the card's preset. This is an atlas-ui-only change to
   an existing component — no backend change.
3. The apply flow is available only from the library card (not a required
   affordance inside the focused editor's save bar). Applying does not depend
   on unsaved edits being saved; if the selected preset has unsaved edits, warn
   that apply uses the **last saved** version (the backend applies persisted
   config), and offer to save first. (Alternatively require a clean preset to
   apply — see Open Questions Q3.)

### FR-4 Focused editor — Identity section

1. Entered by clicking a card (or +New). A "← Preset library" backlink returns
   to the landing view. The editor header shows the preset name, job badge, and
   level, with a `⋯` kebab menu (Duplicate, Apply to an account…, Remove with
   confirm) anchored top-right, aligned with the first section — matching the
   task-177 kebab placement.
2. **Identity** fields: `name` (required, ≤64), `defaultName` (the name
   stamped on created characters; empty = prompt on apply), `description`
   (≤512), and `tags` (chip add/remove, free-form strings).

### FR-5 Focused editor — Class & appearance section

1. **Class**: a **named picker** over `jobId`. Because atlas-data exposes no
   job-name endpoint (`jobs.service` only serves `/jobs/{id}/skills`), names
   come from a **curated client-side jobId→name map** (e.g. 0 Beginner, 100
   Warrior, 110 Fighter, 130 Page, 200 Magician, 300 Bowman, 400 Thief, 500
   Pirate, 900 GM, 910 SuperGM, plus the common advances present in seed
   data). The control is a searchable select of the mapped ids **plus an
   "Advanced" numeric entry** that accepts any `jobId`. Unknown ids render as
   `Job <id>` — never blocked; the backend is the validator of record. The map
   lives in a single `presetJobs.ts` module so it is easy to extend.
2. **Gender**: Male/Female select (0/1).
3. **Appearance picks** — single-value, not pools. Each of face, hair, hair
   color, and skin tone renders as a row of rendered thumbnails (the current
   loadout with the one dimension varied) with the current pick ring-selected;
   clicking a thumbnail sets that value and re-renders the live preview. A
   trailing `+` opens the **task-177 cosmetics browser in single-select mode**:
   - Faces: `/api/data/cosmetics/faces` (paginated), gender-filtered via the
     v83 id convention `(id/1000)%10===1 ⇒ female`, names per-id from
     `/api/data/item-strings/{id}`.
   - Hairs: `/api/data/cosmetics/hairs` (paginated), same treatment.
   - Hair color: digits 0–7 composited on the current base hair (rendered id =
     base hair + color digit).
   - Skin: small numeric range with rendered previews (see Open Questions Q1).
   Selecting a value in the browser sets the single field (replaces, does not
   append) and closes the browser.

### FR-6 Focused editor — Spawn & progression section

1. **Starting map**: a name-searchable picker backed by `/api/data/maps`
   (list + `/api/data/maps/{id}`); closed control shows `<name> · <id>`;
   storage remains numeric `mapId`. Manual numeric entry allowed; unresolvable
   ids show `Map <id>` with a non-blocking warning hint (atlas-data coverage
   varies by version). Reuses the task-177 MapPicker.
2. **Level** (1–250), **GM level** (≥0), **Meso** (≥0) — numeric controls
   (stepper for level/GM, formatted numeric input for meso).

### FR-7 Focused editor — Base stats section

1. `str`, `dex`, `int`, `luk`, `hp`, `mp` — each a nonnegative numeric control
   with steppers. Copy notes these are written **verbatim** to the created
   character (not derived from level), matching backend behavior.

### FR-8 Focused editor — Equipment / Inventory / Skills sections

1. **Equipment** (`equipment[]`, each `{ templateId, useAverageStats }`):
   icon + name + id rows with a remove ×, an inline **"avg stats"** toggle per
   row, and an add flow using the item-strings search combobox
   (`itemsService.searchItems`) with per-slot subcategory filters (reuse
   task-177 `poolSearchConfig` for tops/bottoms/shoes/weapons) plus a
   manual-id fallback. Unlike templates (four separate single-select pools),
   presets store one flat worn list — the editor presents one "Worn items"
   group; slot filtering applies to the *add* combobox, not to grouping.
2. **Inventory** (`inventory[]`, each `{ templateId, quantity }`): icon + name
   + id rows with a quantity stepper (min 1) and remove ×; add combobox
   searches all compartments; empty-state copy "No granted items."
3. **Skills** (`skills[]`, each `{ skillId, level }`): skill icon
   (`getAssetIconUrl(..., "skill", id)`) + name (via `useSkillData` / skills
   service) + id rows with a level stepper (min 1) and remove ×; add flow is a
   numeric id input with name/icon lookup on entry; empty-state copy "This
   preset grants no skills." No skill browser in v1.
4. Rows for ids with no icon/name render a placeholder icon + `Unknown
   item`/`Unknown skill` + the id and remain editable/removable.

### FR-9 Focused editor — live preview, state & save

1. **Sticky live preview** card in the right column of a two-column grid inside
   the existing content width (mirrors task-177 FR-8): a stand1 composite of
   the preset's exact single-value loadout (skin, hair = base+color, face,
   worn equipment on slots top -5 / bottom -6 / shoes -7 / weapon -11, gender),
   a worn-equipment icon strip, and one caption line. Uses the existing
   `useCharacterImage` machinery (skeleton/error/retry). Empty loadout falls
   back to skin 0 / hair 30030 / face 20000. Re-renders on any appearance pick,
   equipment change, or gender change. On narrow (`< ~900px`) the preview
   stacks above the editor sections.
2. **One form state for the whole `presets` array** (plain `useReducer` state
   module + baseline snapshot; dirty = deep compare — the D1 pattern from
   task-177, **not** react-hook-form). **Free switching**: entering/leaving a
   preset editor never loses edits; there is one Save/Discard bar for the whole
   array. The library dirty-dots derive from per-preset diff vs. baseline.
3. **Sticky save bar** at the bottom of the section: dirty text ("No unsaved
   changes" / "Unsaved changes"), **Discard** (revert to last loaded, confirm
   if dirty; resets appearance/preview state), **Save** (single PATCH of the
   full configuration via the context's mutation). Success → toast + reset
   dirty; failure → toast with the API error, state preserved. Field-level API
   validation errors map back to the offending preset/field where the error
   `meta.path` allows (as the current form already does).
4. **+New / Duplicate defaults**: a new preset starts with sensible valid
   defaults (name "New preset", `jobId 0`, `gender 0`, `face 20000`, `hair
   30030`, `hairColor 0`, `skinColor 0`, `mapId 0`, `level 1`, `stats
   {4,4,4,4,50,5}`, empty equipment/inventory/skills). Duplicate deep-copies
   the source preset (new stable id assigned on save per existing semantics).

### FR-10 URL sync & deep-linking

1. Preset selection syncs to the URL as `?preset=<id>` via `useSearchParams`
   (`{ replace: true }`). Presence of a resolvable `?preset=` opens the focused
   editor for that preset; absence shows the library. Unresolvable/stale ids
   fall back to the library (do not error). New/duplicated presets that lack a
   persisted `id` until save use a stable client key for `?preset=` within the
   session.

### FR-11 Visual/system integration

1. All styling via Tailwind + shadcn primitives consistent with the app theme
   (light + dark); the prototype palette is illustrative — map onto existing
   tokens.
2. Pixel-art rendering: `image-rendering: pixelated` on all sprite imagery.
3. Interactive semantics: cards are keyboard-activatable, the kebab is a real
   shadcn menu, thumbnails/toggles are buttons with focus-visible rings, the
   tag filter and search are keyboard operable.

## 5. API Surface

No new or modified backend endpoints. Consumed (all existing; the task-177
cosmetics/maps/item-strings/render surface was live-verified 2026-07-18):

| Need | Endpoint / helper | Source |
|---|---|---|
| Configuration read/write (tenant) | `useTenantConfiguration` / `useUpdateTenantConfiguration` | existing |
| Configuration read/write (template) | `useTemplate` / `useUpdateTemplate` | existing |
| Account search (apply flow) | `useAccountSearch(tenant, namePattern)` → `Account[]` | existing (`useAccounts.ts:129`) |
| Apply preset to account | `ApplyPresetDialog` (+ new optional `initialPresetId` prop) | existing component, UI-only extension |
| Face / hair enumeration | `GET /api/data/cosmetics/faces` / `/hairs` (paginated) | existing (task-177) |
| Face/hair/item names | `GET /api/data/item-strings/{id}` | existing |
| Item search (equipment/inventory add) | `itemsService.searchItems(filters)` | existing |
| Map names/search | `GET /api/data/maps`, `/api/data/maps/{id}` | existing |
| Skill name/icon | `useSkillData` / skills service; `getAssetIconUrl(..., "skill", id)` | existing |
| Item icons | `getAssetIconUrl(..., "item", id)` | existing |
| Character composite | `generateCharacterUrl` → `/api/assets/.../character/{hash}.png` | existing |

Note: there is **no** job-name endpoint — job names are a curated client map
(FR-5.1). Item-strings **search** does not cover faces/hairs — cosmetics are
enumerated and named per-id (same constraint as task-177).

Error cases: unresolvable ids degrade to placeholder + numeric id (never block
editing); render failures use `useCharacterImage`'s retry/error state.

## 6. Data Model

Unchanged and persisted verbatim. The `CharacterPreset` shape
(`src/types/models/template.ts`, mirrored by `character-presets.schema.ts`):

```ts
{
  id?: string,               // stable per-preset id, assigned by backend
  attributes: {
    name, description, tags[],
    jobId, gender (0|1),
    face, hair, hairColor, skinColor,
    mapId, level, meso, gm,
    stats: { str, dex, int, luk, hp, mp },
    defaultName,
    equipment: { templateId, useAverageStats }[],
    inventory: { templateId, quantity }[],
    skills:    { skillId, level }[],
  }
}
```

UI-only state (never persisted): selected preset (mirrored to `?preset=`),
the working array + baseline snapshot, per-preset dirty flags, and the
live-preview thumbnail/override state. `character-presets.schema.ts` is reused
for validation; the deleted forms' inline `DEFAULT_PRESET_ATTRIBUTES` moves
into the shared editor.

## 7. Service Impact

`atlas-ui` only. Touched areas:

- `src/pages/TenantsCharacterPresetsPage.tsx`,
  `TemplatesCharacterPresetsPage.tsx` — thin wrappers over the shared editor.
- `src/pages/tenants-character-presets-form.tsx`,
  `templates-character-presets-form.tsx` — **deleted**.
- New `src/components/features/characters/presets/**` — editor, card library,
  focused editor, section subcomponents, account-picker dialog, `presetJobs.ts`.
- `ApplyPresetDialog.tsx` — new optional `initialPresetId` prop (UI-only).
- Reuse of task-177 modules (cosmetics browser, MapPicker, combobox/popover,
  `poolSearchConfig`, `isFemaleCosmeticId`, `useCharacterImage` wiring). If
  task-177 has not landed on `main` when this task starts, those shared modules
  become a dependency to reconcile (see Open Questions Q2).

No Go service changes. No layout/shell changes.

## 8. Non-Functional Requirements

- **Render volume**: library cards each request one composite (cached by
  loadout hash in atlas-renders); cosmetics browser grids page at ≤24, lazy.
  Cap concurrent render fetches via existing `useCharacterImage` utilities.
- **Name resolution volume**: per-id item-strings/map/skill lookups batched and
  cached with React Query (generous `staleTime`; keys include tenant).
- **Multi-tenancy**: all requests go through the existing API client injecting
  tenant/region/version headers; switching tenants clears caches via the
  existing `TenantProvider`.
- **Theming**: light + dark, tokens only.
- **Accessibility**: keyboard operability for cards/menu/thumbnails/search/tag
  filter; `prefers-reduced-motion` respected.
- **Testing**: Vitest + Testing Library component tests for: library
  search/tag-filter, card quick-actions (duplicate, apply-open),
  `?preset=<id>` URL sync + library/editor toggling, add/remove/duplicate
  lifecycle, each section's edit semantics, the account-picker → ApplyPreset
  handoff, and both page adapters (asserting `templates` sibling survival on
  save). New test files must type-check under `tsc -b`.

## 9. Open Questions

1. **Skin tone range** — no endpoint enumerates valid skins; seed data uses
   0–3. Offer 0–9 with rendered previews (bad id renders default body) or lock
   to 0–3 for v83? (task-177 left this open too; keep both consistent.)
2. **task-177 dependency ordering** — this task reuses task-177's cosmetics
   browser, MapPicker, combobox/popover, and render wiring. Preferred:
   task-177 lands on `main` first and this branch builds on it. If they run
   concurrently, the design phase must decide whether to extract the shared
   modules or temporarily duplicate. Confirm sequencing before planning.
3. **Applying a dirty preset** — apply-to-account uses persisted config. When
   the card's preset has unsaved edits, do we (a) warn "applies last saved" and
   offer save-first, or (b) require a clean preset to enable the apply action?
   Default assumption: (a).
4. **`jobId` curated map coverage** — confirm the initial jobId→name map only
   needs to cover ids present in seed data + common advances, with everything
   else falling back to `Job <id>` (backend authoritative). Assumed yes.

## 10. Acceptance Criteria

- [ ] Both `/tenants/:id/character/presets` and
      `/templates/:id/character/presets` render the shared editor; the two
      duplicated form files are deleted; no inline `CharacterPreset`
      redeclaration remains.
- [ ] Library view shows preset cards (rendered sprite, job-name badge, level,
      tags, description, dirty-dot) with search + single-select tag filter and
      a `+ New` affordance; card order is the persisted array order.
- [ ] Card hover quick-actions Duplicate and Apply-to-account work; Apply opens
      an account-search dialog that hands off to `ApplyPresetDialog` scoped to
      the card's preset.
- [ ] Clicking a card opens the focused editor with a backlink, top-right kebab
      (Duplicate / Apply / Remove-with-confirm), and all sections: Identity,
      Class & appearance (named job picker + advanced numeric; single-select
      appearance thumbnails + cosmetics browser), Spawn & progression (map name
      search, level/GM/meso), Base stats, Equipment (avg-stats toggle),
      Inventory (quantity), Skills (numeric-id add with lookup).
- [ ] `?preset=<id>` deep-links the focused editor; absent/unresolvable shows
      the library without error.
- [ ] Sticky live preview composites via the existing atlas-renders flow, shows
      the worn-equipment strip, and updates on picks/edits/gender change.
- [ ] One PATCH-the-array save flow with free switching, dirty indicator +
      per-card dirty-dots, discard-with-confirm, success/error toasts, and
      field-level error mapping; save preserves the sibling `templates` array
      in both contexts.
- [ ] Loading skeletons and `ErrorDisplay` are identical in both contexts.
- [ ] Light + dark themes; sprites pixelated; cards/menu/search/thumbnails
      keyboard operable.
- [ ] `npm run test`, `npm run lint` (no new errors vs baseline), and
      `npm run build` clean in `services/atlas-ui` (nvm node 22); and
      `tools/lint.sh --check` clean at repo root.
- [ ] Component tests cover library search/filter, card actions, URL sync,
      lifecycle (add/duplicate/remove), section edits, the apply handoff, and
      both page adapters.
