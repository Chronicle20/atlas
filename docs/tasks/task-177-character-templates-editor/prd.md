# Character Templates Editor — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-18
---

## 1. Overview

Character templates are the per-tenant **character-creation contract**: for each
(jobIndex, subJobIndex, gender) key they define the pools of faces, hairs, hair
colors, skin tones, and starting equipment a player may choose at the create
screen, plus the starting map and the items/skills granted on creation.
`atlas-character-factory` validates every creation request against these pools
(`factory/processor.go` `Create`, which rejects any face/hair/color/skin/top/
bottom/shoes/weapon not present in the matching template and falls back to the
template's `mapId`). The arrays are therefore simultaneously *what the create
screen offers* and *what the server will accept*.

The current admin UI for this data is two wholesale-duplicated form files
(`src/pages/tenants-character-templates-form.tsx` and
`src/pages/templates-character-templates-form.tsx`) that render each template
as a stack of bordered boxes full of raw numeric IDs — no icons, no names, no
character preview, inconsistent loading states between the two copies, and no
way to add a template at all (`useFieldArray` destructures only `remove`).

This task replaces both with one shared **compact in-page editor**, following
the interactive prototype iterated with the user (committed alongside this PRD
as [`prototype.html`](prototype.html) — open it in a browser; it is the visual
source of truth). The editor fits inside the existing detail-page shell with
**zero layout or backend changes**: segmented template selector, sectioned
editor with real names and icons, and a sticky live character preview
composited by atlas-renders.

## 2. Goals

Primary goals:
- One shared editor component used by both the Tenant Details and Template
  Details "Character Templates" sections; delete the two duplicated forms.
- Make every ID visual and named: item icons + names for equipment pools,
  rendered thumbnails for appearance pools, map name for the starting map,
  a live composited preview of the selected template.
- Add the missing template lifecycle: create (+ New), duplicate, remove.
- Deep-linkable template selection via URL param.
- Fit entirely inside the existing `TenantDetailLayout` / `TemplateDetailLayout`
  shell (`DetailSidebar` + `lg:max-w-4xl` content column) — zero shell changes.
- Zero backend changes — every data need is served by existing endpoints
  (verified against the live cluster; see §5).

Non-goals:
- Character Presets section (separate feature, separate pages).
- Any change to `DetailSidebar`, the detail layouts, routing shell, or nav.
- Backend/API changes in any Go service.
- The "Creator Preview" player-view mode (prototype option C) — dropped.
- "Edit as JSON" escape hatch — explicitly dropped from the kebab menu (user
  decision; bulk edits wait for a real import/export feature).
- Changes to seed template JSON (`services/atlas-configurations/seed-data/`).

## 3. User Stories

- As a tenant operator, I want to see each character template as a named,
  visual record (class, gender, spawn map, rendered character) so that I can
  audit creation options without memorizing WZ IDs.
- As a tenant operator, I want to add a new template so that a new class/gender
  combination can be offered at character creation (currently impossible in
  the UI).
- As a tenant operator, I want to duplicate an existing template so that I can
  create the counterpart gender or a variant class without re-entering pools.
- As a tenant operator, I want to browse valid faces/hairs visually and add
  them to a pool so that I don't paste wrong IDs that the factory will reject
  or the client will render broken.
- As a tenant operator, I want to search equipment by name when adding to a
  pool so that I don't need to look up item IDs externally.
- As a tenant operator, I want a live preview of what a created character will
  look like so that I catch bad combinations before players do.
- As a developer, I want a single shared form component so that fixes land in
  both the tenant and template contexts at once.

## 4. Functional Requirements

### FR-1 Shared component

1. New shared component (suggested: `src/components/features/characters/`
   `templates/CharacterTemplatesEditor.tsx` plus colocated subcomponents)
   that owns the entire section UI. It is parameterized by a small adapter
   interface — `{ data, isLoading, error, save(templates), isSaving }` —
   supplied by each page wrapper:
   - Tenant page: `useTenantConfiguration` / `useUpdateTenantConfiguration`.
   - Template page: `useTemplate` / `useUpdateTemplate`.
2. `src/pages/tenants-character-templates-form.tsx` and
   `src/pages/templates-character-templates-form.tsx` are deleted; the two
   page components (`TenantsCharacterTemplatesPage`,
   `TemplatesCharacterTemplatesPage`) render the shared editor inside their
   existing layouts.
3. The editor works on `CharacterTemplate[]` imported from
   `@/types/models/template` — the inline re-declaration of the shape in the
   template-side form must not survive the refactor.
4. Loading uses skeleton components; errors use the shared `ErrorDisplay`;
   both contexts get identical states (today they differ).

### FR-2 Template selector (segmented control)

1. Templates render as a segmented control (recessed track, flat text
   segments) labeled `<World> · <M|F>`, where World derives from jobIndex:
   0 → "Cygnus Knights", 1 → "Adventurer", 2 → "Aran", 3 → "Evan"
   (mapping mirrors `atlas-character-factory` `job/model.go` `JobFromIndex`).
   Unknown indexes render as `Job N`. If two templates share the same label,
   suffix an ordinal (`Adventurer · M (2)`).
2. No thumbnails/imagery in the selector (user decision): character imagery
   appears only in the preview card and appearance-pool thumbnails, so a
   sprite always means "rendered output", never navigation.
3. A `+ New` segment appends a template and selects it. The new template
   starts with sensible empty defaults (`jobIndex 0, subJobIndex 0, gender 0,
   mapId 0`, empty pools) and the operator fills Identity in the editor
   (Identity is fully editable — FR-4).
4. Selection syncs to the URL as `?tpl=<index>` via `useSearchParams`;
   out-of-range or non-numeric values clamp to 0. Deep links and refresh
   restore the selection. Switching templates never resets unsaved edits.
5. Empty state (no templates in the configuration): explanatory copy + a
   prominent "Add template" button.

### FR-3 Per-template actions (kebab menu)

1. A `⋯` icon button anchored at the top-right of the editor content, aligned
   with the Identity section header (NOT floating in the selector row — see
   prototype). Opens a shadcn `DropdownMenu` with:
   - **Duplicate template** — deep-copies the selected template, appends it,
     selects the copy.
   - **Remove template** — confirm dialog (template label + "players can no
     longer create this class/gender until re-added"), then removes from the
     array and selects the nearest remaining index.
2. No "Edit as JSON" item (user decision).

### FR-4 Identity section (fully editable)

1. **Class**: select of known (jobIndex, subJobIndex) pairs — Cygnus Knights
   (0.0), Adventurer (1.0), Aran (2.0), Evan (3.0) — displayed as
   `<World> (<jobIndex>.<subJobIndex>)`, plus an "Advanced" affordance that
   accepts arbitrary numeric jobIndex/subJobIndex (forward-compat for e.g.
   Dual Blade 1.1 which the factory does not yet map — do NOT hardcode-block
   unknown values, the backend is the validator of record).
2. **Gender**: Male/Female select (0/1).
3. **Starting map**: picker with name search backed by the maps service
   (`/api/data/maps` list + `/api/data/maps/{id}`); the closed control shows
   `<name> · <streetName> · <id>`; storage remains the numeric `mapId`.
   Manual numeric entry allowed; unresolvable IDs show `Map <id>` with a
   warning hint, not an error (atlas-data coverage varies by version).

### FR-5 Appearance pools (faces, hairs, hairColors, skinColors)

1. Each pool renders as a row of square thumbnails (~76px) showing the head
   region of an atlas-renders composite: the current template's selected
   appearance picks with the one dimension varied (face thumbnails vary face,
   hair thumbnails vary hair, etc.). Thumbnail = CSS crop of the standard
   render (the prototype uses `background-position: -74px -70px` on the
   192×256 stand1 render at `resize=2` — tune during implementation). Each
   thumbnail carries its numeric ID as a small caption.
2. Clicking a thumbnail marks it selected (accent ring) and re-renders the
   live preview (FR-8). Selection is UI-only preview state — never persisted.
3. Each thumbnail exposes a remove affordance (hover/focus ×) with the same
   array-edit semantics as equipment rows.
4. **Add flow — full visual browser** (user decision): a dialog with a
   paginated thumbnail grid of all valid IDs:
   - Faces: enumerate `/api/data/cosmetics/faces` (verified live: 536 rows,
     JSON:API paginated, attributes `{cash}`).
   - Hairs: enumerate `/api/data/cosmetics/hairs` (verified live: 1520 rows).
   - Filter the grid by the template's gender using the v83 id convention
     `(id/1000)%10 === 1 ⇒ female` (same rule as `resolveGender` in
     `characterRender.service.ts`), with a toggle to show all.
   - Names resolve per-id from `/api/data/item-strings/{id}` (verified live:
     faces and hairs have real names — e.g. 20000 "Male 1 (Black)", 30030
     "Black Buzz"). NOTE (verified): the item-strings **search** index does
     NOT cover faces/hairs (`?search=Buzz` returns 0 rows), so the browser
     must enumerate via the cosmetics endpoints and resolve names by id —
     batched and cached with React Query; name search within the browser
     filters the already-fetched page set client-side.
   - Grid thumbnails are on-demand atlas-renders composites (current
     character with the candidate id applied), rendered lazily per page
     (page size ≤ 24) — atlas-renders persists by loadout hash so repeat
     opens are cache hits.
   - Already-in-pool ids are marked and cannot be double-added.
5. **Hair colors**: pool of color digits; thumbnails composite `hair + color`
   (v83 convention: rendered hair id = base hair id + color digit). Add flow
   offers digits 0–7 rendered on the current base hair; digits already in
   the pool are marked.
6. **Skin tones**: pool of skin ids rendered on the current character. Add
   flow offers a small numeric range (no enumeration endpoint exists; seed
   data uses 0–3 — offer 0–9 with rendered previews and let the operator
   judge; see Open Questions).

### FR-6 Equipment pools (tops, bottoms, shoes, weapons)

1. Each pool renders as rows: item icon (`getAssetIconUrl(..., "item", id)`),
   display name (item-strings), numeric ID (mono), remove ×. Group header
   shows `<n> options · player picks one`.
2. **Add flow — combobox**: inline search using the existing item-strings
   search (`itemsService.searchItems`) with per-pool filters:
   - Tops: `filter[subcategory]` top + overall (Aran's 1042167 is an overall
     — the tops pool legitimately contains overalls).
   - Bottoms: bottom. Shoes: shoes. Weapons: the weapon subcategory set
     already enumerated in atlas-data's `item/filter.go`.
   Manual numeric entry is always available as a fallback row in the
   combobox ("Use id <n>").
3. Rows for ids with no icon/name (bad data) render a placeholder icon +
   `Unknown item` + the id, and remain removable.

### FR-7 Starting kit (items, skills)

1. Items: same row treatment as FR-6, header `<n> granted` (granted on
   creation, not picked). Add combobox searches all compartments.
2. Skills: numeric id rows with name resolution via the existing skill data
   hooks (`useSkillData`/skills service); empty state copy: "This class
   starts with no granted skills." Add flow: numeric input with name/icon
   lookup on entry (`getAssetIconUrl(..., "skill", id)`); no skill browser
   in v1.

### FR-8 Live preview (sticky card)

1. Right column of a two-column grid inside the existing content width
   (~`minmax(0,1fr) 252px`); sticky. Card contains, top to bottom:
   - "Live preview" label.
   - 154px-wide stage (accent-tinted gradient + floor shadow, per prototype)
     showing the composited character.
   - A strip of the four worn equipment icons (first of each pool),
     name + id on hover.
   - One short caption line. **No id/value table** (user decision — the
     selected ids are already visible as highlighted thumbnails and rows).
2. The image is the existing `generateCharacterUrl` →
   `/api/assets/.../character/{hash}.png` flow (as used by
   `CharacterRenderer`): skin = selected skin tone, hair = selected base
   hair + selected color digit, face = selected face, items = first entry of
   each equipment pool, gender = template gender, stance `stand1`, resize 2.
   Use the existing `useCharacterImage` machinery (skeleton/error/retry)
   rather than a bare `<img>`.
3. Preview re-renders on: template switch, any appearance pick, any edit
   that changes the first entry of an equipment pool, identity gender
   change.
4. On mobile/narrow (`< ~900px`) the preview stacks below the selector,
   above the editor sections.

### FR-9 Form state & save

1. One form state for the whole `templates` array (react-hook-form +
   `useFieldArray` or equivalent controlled state). **Free switching** (user
   decision): changing the selected template never loses edits; there is one
   Save/Discard bar for the whole array.
2. Sticky save bar at the bottom of the section: dirty indicator text
   ("No unsaved changes" / "Unsaved changes"), Discard (revert to last
   loaded data, confirm if dirty), Save (single PATCH of the full
   configuration via the context's mutation, exactly as today).
3. On save success: toast + reset dirty state; on failure: toast with the
   API error, state preserved.
4. Surface `validateTemplateConsistency`-style warnings (empty faces/hairs/
   hairColors/skinColors pools) inline on the relevant group headers as
   non-blocking warnings — the factory rejects creations against empty
   pools, so operators should see it before players do.

### FR-10 Visual/system integration

1. All styling via Tailwind + shadcn primitives consistent with the app
   theme (light + dark); the prototype's exact palette is illustrative — map
   it onto the app's existing tokens.
2. Pixel-art rendering: `image-rendering: pixelated` on all sprite imagery.
3. Interactive semantics: selector is `role="tablist"`, kebab is a real menu
   with keyboard focus handling (shadcn defaults cover this), thumbnails are
   buttons with focus-visible rings.

## 5. API Surface

No new or modified endpoints. Consumed (all existing; live-verified 2026-07-18
against tenant `06357ee8…` GMS 83.1):

| Need | Endpoint / helper | Verified |
|---|---|---|
| Configuration read/write (tenant) | `useTenantConfiguration` / `useUpdateTenantConfiguration` | existing UI hooks |
| Configuration read/write (template) | `useTemplate` / `useUpdateTemplate` (`/api/configurations/templates`) | existing UI hooks |
| Face enumeration | `GET /api/data/cosmetics/faces` (paginated) | ✅ 536 rows |
| Hair enumeration | `GET /api/data/cosmetics/hairs` (paginated) | ✅ 1520 rows |
| Face/hair/item names | `GET /api/data/item-strings/{id}` | ✅ incl. faces ("Male 1 (Black)") and hairs ("Black Buzz") |
| Item search (equipment/kit add) | `GET /api/data/item-strings?search=&filter[...]` via `itemsService.searchItems` | ✅ (note: search index excludes faces/hairs — do not use search for cosmetics) |
| Map names/search | `GET /api/data/maps`, `GET /api/data/maps/{id}` | ✅ ("Mushroom Town" / "Maple Road") |
| Item/skill icons | `getAssetIconUrl(tenant, region, ver, "item"\|"skill", id)` | existing |
| Character composite | `generateCharacterUrl` → `/api/assets/.../character/{hash}.png` (atlas-renders `/api/wz/character/render/...`) | ✅ used to build the prototype's renders |

Error cases: unresolvable ids degrade to placeholder + numeric id (never
block editing); render failures use `useCharacterImage`'s retry/error state.

## 6. Data Model

Unchanged. `CharacterTemplate` (`src/types/models/template.ts:4-19`) remains
the persisted shape:

```ts
{ jobIndex, subJobIndex, gender, mapId,
  faces[], hairs[], hairColors[], skinColors[],
  tops[], bottoms[], shoes[], weapons[], items[], skills[] }
```

UI-only state (never persisted): per-template preview picks
`{ faceIdx, hairIdx, hairColorIdx, skinIdx }`, selected template index
(mirrored to `?tpl=`), dirty flag.

Type cleanup: the template-side form's inline redeclaration of this shape is
removed with the file (FR-1.3).

## 7. Service Impact

`atlas-ui` only. Touched areas:

- `src/pages/TenantsCharacterTemplatesPage.tsx`, `TemplatesCharacterTemplatesPage.tsx` — thin wrappers over the shared editor.
- `src/pages/tenants-character-templates-form.tsx`, `templates-character-templates-form.tsx` — **deleted**.
- New `src/components/features/characters/templates/**` — editor, selector, pool groups, browsers, preview card.
- New/extended service + hooks for cosmetics enumeration (`/api/data/cosmetics/faces|hairs`) with React Query caching.
- Possibly small additions to `src/lib/hooks/useCharacterImage.ts` call sites (no behavioral change).

No Go service changes. No layout/shell changes (`TenantDetailLayout`,
`TemplateDetailLayout`, `DetailSidebar`, `App.tsx` routes untouched).

## 8. Non-Functional Requirements

- **Render volume**: browser grids request ≤ 24 composites per page, lazily
  (`loading="lazy"`), and only for the open page. atlas-renders caches by
  loadout hash (bucket-persisted), so repeated opens are cache hits; an
  admin browsing warms the cache for the real login flow. Cap concurrent
  render fetches (existing `useCharacterImage` preload utilities).
- **Name resolution volume**: per-id item-strings lookups are batched and
  cached with React Query (`staleTime` generous — WZ strings are static per
  version); cache keys include tenant.
- **Multi-tenancy**: all requests go through the existing API client which
  injects `TENANT_ID`/`REGION`/`MAJOR_VERSION`/`MINOR_VERSION`; switching
  tenants clears caches via the existing `TenantProvider` behavior.
- **Theming**: light and dark, tokens only.
- **Accessibility**: keyboard operability for selector/menu/thumbnails;
  `prefers-reduced-motion` respected (no new animations beyond hover/focus).
- **Testing**: component tests for selector URL-param sync, add/duplicate/
  remove flows, pool edit semantics, and the adapter wiring of both page
  contexts (Vitest + Testing Library, per repo conventions).

## 9. Open Questions

1. **Skin tone enumeration** — no endpoint enumerates valid skins; seed data
   uses 0–3. v1 offers 0–9 with rendered previews (a bad id just renders the
   default body). Acceptable, or should the range be locked to 0–3 for v83?
2. **Dual Blade (1.1)** — `JobFromIndex` has a TODO for subJobIndex 1;
   Identity's Advanced input permits entering it anyway. Confirm that's the
   desired posture (UI permissive, backend authoritative).
3. **Hair-color rendering gaps** — not every base hair has all color
   variants in WZ; a missing variant renders as a failed composite. v1 shows
   the render error state in the picker; is that sufficient signal?

## 10. Acceptance Criteria

- [ ] Both `/tenants/:id/character/templates` and
      `/templates/:id/character/templates` render the shared editor; the two
      duplicated form files are deleted; no inline `CharacterTemplate`
      redeclaration remains.
- [ ] Segmented selector shows `<World> · <M|F>` labels (no thumbnails),
      supports `?tpl=N` deep links, and clamps invalid values.
- [ ] `+ New` appends and selects a blank template; Identity (class, gender,
      starting map with name search) is fully editable.
- [ ] Kebab menu (Duplicate / Remove with confirm) is anchored top-right of
      the editor content, aligned with the Identity header; no Edit-as-JSON.
- [ ] Appearance pools render as clickable rendered thumbnails with remove
      affordances; the add dialog is a paginated visual browser backed by
      `/api/data/cosmetics/faces|hairs`, gender-filtered by the id
      convention, with per-id name resolution.
- [ ] Equipment/kit pools render icon + name + id rows with a filtered
      search combobox add flow and manual-id fallback.
- [ ] Sticky live preview composites via the existing atlas-renders flow,
      shows the worn-equipment icon strip, and contains no id/value table;
      it updates on picks, template switch, and relevant edits.
- [ ] One PATCH-the-array save flow with free template switching, dirty
      indicator, discard-with-confirm, success/error toasts, and inline
      empty-pool warnings.
- [ ] Loading skeletons and `ErrorDisplay` are identical in both contexts.
- [ ] Light + dark themes; sprites pixelated; selector/menu keyboard
      operable.
- [ ] `npm run test`, `npm run lint`, `npm run build` clean in
      `services/atlas-ui` (nvm node 22; no new lint errors per the repo's
      baseline rule), and `tools/lint.sh --check` clean at repo root.
- [ ] Component tests cover selector URL sync, add/duplicate/remove, pool
      edits, and both page adapters.
