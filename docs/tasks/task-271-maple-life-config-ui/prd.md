# Maple Life Configuration Editor — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-27
---

## 1. Overview

Task-246 shipped in-game character creation via the `Cash/0543` "Maple Life"
item family. A player uses the item in the field, the client opens
`CUICharacterSaleDlg`, picks a class and an appearance, and
`atlas-character-factory` seeds a real character onto their account. The
server's authority over that dialog lives entirely in one configuration block —
`mapleLife` — carried on both tenant and template configurations
(`services/atlas-configurations/atlas.com/configurations/tenants/rest.go:23`,
`services/atlas-configurations/atlas.com/configurations/templates/rest.go:23`).

That block is currently invisible to operators. `grep -ri maplelife
services/atlas-ui/src` returns nothing: it is absent from
`TenantConfigAttributes` (`services/atlas-ui/src/services/api/tenants.service.ts:75-131`)
and `TemplateAttributes` (`services/atlas-ui/src/types/models/template.ts:73-125`),
has no page, no route, no nav entry, and no breadcrumb. The only way to read or
change it is `curl` against `/api/configurations/{tenants,templates}`.

The cost of that gap is already on record. Task-246's own
`docs/tasks/task-246-maple-life-character-creation/bug-maple-life-seed-never-reaches-live-db.md`
documents a live tenant whose `mapleLife` block had drifted from the corrected
seed file — an operator saw a Warrior created with 114 AP instead of 123 and
pre-fix starting equipment, and diagnosing it required reading raw JSON out of
two API endpoints by hand. There is also nowhere to answer the question
task-246's design left open: `design.md` §A6 rules that the class-ordinal→job
mapping for ordinals 2/3/4 is *unconfirmed data*, to be pinned by observation
and then corrected as configuration. Without a UI, "corrected as configuration"
means hand-editing JSON in a database.

This task adds a dedicated **Maple Life** sub-page beside the existing Character
Templates and Character Presets editors, in both the tenant and template
contexts, following the shape those two established: a detail-layout rail entry,
a thin page per context passing a context adapter into one shared editor, a
reducer-held working copy, a URL-deep-linked selection, and a registered save
bar.

## 2. Goals

Primary goals:

- An operator can read and edit a tenant's or template's entire `mapleLife`
  block from atlas-ui, with the same affordances (item search, appearance
  browser, map picker, live character render) the sibling editors already
  provide.
- The three semantics an operator cannot infer from the raw numbers are surfaced
  in the UI: which ordinals are derived versus unconfirmed, that `spSkillId` is
  inert for ordinals ≥ 2, and that an appearance pool is an allow-list validated
  against the player's submission.
- Every hard rule enforced by `atlas-character-factory`'s validator is enforced
  client-side before Save, so a configuration that would reject live character
  creation cannot be written from this page.
- A tenant whose `mapleLife` block is empty has a first-class path to populate
  it from a matching template, rather than a blank page.
- `mapleLife` is declared in the frontend's configuration types and provably
  survives a save issued from any configuration sub-page.

Non-goals:

- Any change to `atlas-configurations`, `atlas-character-factory`, or
  `atlas-channel`. The REST models already carry the block; this is a
  presentation task.
- Cash-shop commodity or pricing authoring for the `5430xxx`/`5431xxx`/`5432xxx`
  item ids. That is separate seed data, explicitly out of scope in task-246's
  PRD §2 and still out of scope here.
- Changing the wire protocol, the dispatch arm, the saga, or any packet codec.
- Option B (a tab inside the Character Presets editor) and Option C (a
  schema-driven raw-table panel) from `prototype.html`. Both were evaluated and
  rejected; only Option A is built.
- A drift indicator comparing a tenant's block against its source template.
  Considered and deliberately deferred.
- Adding, removing, or renumbering class ordinals. See FR-4.

## 3. User Stories

- As an operator, I want to see a tenant's Maple Life classes and appearance
  pools in the admin UI, so that diagnosing a wrong-stats report does not
  require reading raw configuration JSON.
- As an operator, I want to correct a class's AP, SP pool, stats, or starting
  equipment and save it, so that a seed-drift bug is a two-minute fix rather
  than a database edit.
- As an operator, I want the page to tell me that ordinals 2/3/4 have an
  unconfirmed job order, so that I know to verify against a live client before
  trusting them.
- As an operator, I want the SP-skill control to be unavailable on ordinals
  where the client never renders that step, so that I cannot author a setting
  that can never take effect.
- As an operator, I want to be stopped from saving an SP pool that cannot cover
  the skill level plus its prerequisite, so that the failure surfaces here
  rather than as a rejected player creation.
- As an operator, I want to see the character a given class and appearance
  combination actually produces, so that an appearance allow-list that diverges
  from the client's own options is visible before a player hits it.
- As an operator setting up a new tenant, I want to seed its Maple Life block
  from a template of the same region and version, so that I do not have to
  author ten class rows by hand.

## 4. Functional Requirements

### FR-1 Configuration types and round-trip preservation

- **FR-1.1** — `mapleLife` MUST be declared on both `TenantConfigAttributes`
  (`services/atlas-ui/src/services/api/tenants.service.ts`) and
  `TemplateAttributes` (`services/atlas-ui/src/types/models/template.ts`), as an
  optional property, mirroring the existing optional `cashShop` precedent. A
  tenant or template with no block decodes to `undefined`, which is a legitimate
  state (§6) and not an error.
- **FR-1.2** — The domain types MUST live in
  `services/atlas-ui/src/types/models/template.ts` beside the existing
  `CharacterTemplate` / `CharacterPreset` types, matching the Go field names and
  JSON tags exactly (§6).
- **FR-1.3** — Both save paths spread the previously fetched attributes object
  (`tenants.service.ts:312` for tenants, `TemplatesCharacterTemplatesPage.tsx`'s
  `...template.attributes` for templates), so an undeclared `mapleLife` *should*
  already survive a save issued from a sibling page. This is **unverified**. The
  task MUST prove it with a regression test that fetches a configuration
  carrying a `mapleLife` block, saves an unrelated section (e.g. character
  presets), and asserts the block is present and byte-identical in the PATCH
  body. If it does not survive, fixing that is in scope.
- **FR-1.4** — Saving from the Maple Life page MUST NOT disturb any other
  section of the configuration. The same regression shape applies in reverse:
  save `mapleLife`, assert `characters`, `npcs`, `worlds`, `socket`, and
  `cashShop` are unchanged in the PATCH body.

### FR-2 Routing, navigation, breadcrumbs

- **FR-2.1** — Two new routes, registered in
  `services/atlas-ui/src/App.tsx` alongside the existing
  `/character/templates` and `/character/presets` entries, both lazily loaded
  via `lazyWithReload`:
  - `/tenants/:id/character/maple-life` → `TenantsMapleLifePage`
  - `/templates/:id/character/maple-life` → `TemplatesMapleLifePage`
- **FR-2.2** — A "Maple Life" entry MUST be added to both detail-layout rails,
  after "Character Presets":
  `services/atlas-ui/src/components/features/tenants/TenantDetailLayout.tsx` and
  `services/atlas-ui/src/components/features/templates/TemplateDetailLayout.tsx`.
- **FR-2.3** — Both patterns MUST be registered in
  `services/atlas-ui/src/lib/breadcrumbs/routes.ts` with label `Maple Life` and
  parent `/{tenants,templates}/[id]/character` (the existing non-navigable
  grouping node), and added to the route-constant map at the bottom of that file
  as `TENANT_CHARACTER_MAPLE_LIFE` / `TEMPLATE_CHARACTER_MAPLE_LIFE`.

### FR-3 Shared editor and per-context adapters

- **FR-3.1** — One shared `MapleLifeEditor` component holds all behaviour. Each
  page is a thin wrapper that supplies a `MapleLifeEditorAdapter`, exactly
  mirroring `TemplatesEditorAdapter` and `PresetsEditorAdapter`:

  ```ts
  export interface MapleLifeEditorAdapter {
    mapleLife: MapleLifeConfig | undefined;
    isLoading: boolean;
    error: Error | null;
    /** Fire the context's PATCH; call onSuccess only when it lands. */
    save: (config: MapleLifeConfig, onSuccess: () => void) => void;
    isSaving: boolean;
    /** Present only in tenant context; absent on the template page hides FR-12's action. */
    seedFrom?: { region: string; majorVersion: number; minorVersion: number };
  }
  ```

- **FR-3.2** — The tenant page uses `useTenantConfiguration` /
  `useUpdateTenantConfiguration`; the template page uses `useTemplate` /
  `useUpdateTemplate`. Both toast on success and on error, matching the sibling
  pages verbatim.
- **FR-3.3** — The editor MUST seed its reducer exactly once, guarded by a
  `loaded` flag, so a post-save invalidation refetch cannot clobber an
  in-progress working copy. This is the same guard and the same failure mode
  documented in `CharacterTemplatesEditor.tsx`.
- **FR-3.4** — Loading renders `FormSkeleton`; a query error renders
  `ErrorDisplay`. Both from `@/components/common`, as the siblings do.

### FR-4 Class grid: fixed 5 × 2

- **FR-4.1** — The editor presents a **fixed** grid of ten class rows: ordinals
  `0..4` × genders `0..1`. The ordinal is a label, never an editable field, and
  there is no add or remove action. This mirrors the client, whose
  `OnButtonClicked` cycles `m_nCurrentClass = (m_nCurrentClass + 1) % 5`
  (`design.md` §A2), and makes `ErrClassOrdinalUnknown` unreachable from a
  configuration authored on this page.
- **FR-4.2** — Selection is two controls: a gender segmented control (Male /
  Female) and a five-segment ordinal control labelled `<ordinal> · <job name>`.
  Both `looks` and `classes` are gender-split, so one gender toggle governs the
  whole page.
- **FR-4.3** — A loaded configuration missing one of the ten `(ordinal, gender)`
  rows MUST still render that cell, populated with an explicit "not configured"
  empty state and a warning that a player selecting it is rejected. Selecting it
  and editing any field materialises the row. A row is never silently invented
  on load — only an operator edit creates it.
- **FR-4.4** — The ordinal control MUST show a per-ordinal status marker:
  ordinals 0 and 1 a "derived" badge, ordinals 2/3/4 an "unconfirmed" badge
  (FR-5.3).
- **FR-4.5** — The selected ordinal MUST be deep-linked as a URL search param
  (`?ordinal=`, with `?gender=`), clamped to a valid value on load, written back
  with `{ replace: true }`, following `CharacterTemplatesEditor`'s `?tpl=`
  handling including its documented reasons for running the apply-effect on load
  only.

### FR-5 Identity section

- **FR-5.1** — Editable fields: `jobId`, `level`, `mapId`.
- **FR-5.2** — `jobId` uses the tenant's real job options via
  `usePresetJobOptions` (backed by `/api/data/job-availability`), which supplies
  version-correct display names. Per that hook's contract, `isPending` and
  `isError` mean *unknown* and MUST be rendered as distinct affordances — never
  as an empty list, and never falling back to a static job table.
- **FR-5.3** — Ordinals 2/3/4 MUST carry a persistent, non-dismissible notice
  stating that the ordinal→job order is not derived from the client and must be
  pinned against a live channel log before being trusted, citing task-246
  `design.md` §A6. Ordinals 0 and 1 carry a "derived from the client's own
  step-skip" note instead. The job field remains fully editable in both cases.
- **FR-5.4** — `mapId` uses the existing `MapPicker`
  (`components/features/characters/templates/MapPicker.tsx`), unchanged.
- **FR-5.5** — `level` is a bounded number input (1–200).

### FR-6 Appearance pools

- **FR-6.1** — Four pool sections per gender — Faces, Hairs, Hair colours, Skin
  tones — rendered from `looks[gender]`. Each shows the option count and the
  statement that the player picks exactly one.
- **FR-6.2** — Each entry renders as a real character sprite via
  `generateCharacterUrl`, as `AppearancePoolSection` / `AppearanceThumb` already
  do for character templates. Clicking an entry sets it as the previewed pick
  for that dimension; it does not modify the configuration.
- **FR-6.3** — Adding an entry uses the existing `AppearanceBrowserDialog`,
  which already covers exactly these four dimensions via `useFaceIds` /
  `useHairIds` and its hair-colour and skin enumerations. `AppearancePoolSection`
  is currently typed against `CharacterTemplate` and builds its preview loadout
  from that shape; it MUST be generalised (pool array plus a loadout-builder
  callback) rather than duplicated, and the character-templates editor MUST
  continue to pass its existing behaviour through unchanged.
- **FR-6.4** — Each entry has a remove affordance. An emptied pool renders the
  blocking validation error in FR-11.
- **FR-6.5** — The section MUST state the value domain for each dimension, since
  the raw numbers are not self-describing (§6): faces are full item ids, hairs
  are normalised style ids, hair colours are bare digits `0..9`, skin tones are
  bare byte ordinals.
- **FR-6.6** — Because the client sources its own carousel from WZ and the
  server only validates against this list, the section MUST carry a one-line
  statement that these are allow-lists and that a list diverging from the
  client's options produces player-visible rejections.

### FR-7 Progression section

- **FR-7.1** — Editable: `stats.str`, `stats.dex`, `stats.int`, `stats.luk`,
  `stats.hp`, `stats.mp`, `ap`, `meso`. Reuse `BaseStatsSection`'s
  `useSyncedNumberInput` draft-echo pattern for every number input; do not
  re-solve the mid-edit clobber bug that pattern exists to fix.
- **FR-7.2** — The stats fields MUST be labelled as the **skill-excluded**
  midpoint: the SP skill's own `29 × effectX` HP/MP contribution is added at
  creation time by `factory/maple_life.go`, not stored here. Without this label
  the numbers read as wrong.
- **FR-7.3** — `ap` and `sp` are what remains **unspent** at `level`; `stats` is
  what was already spent to meet the job requirement. Both MUST be labelled to
  that effect.
- **FR-7.4** — `sp` is the ten-book pool string atlas-character persists
  (`"61,0,0,0,0,0,0,0,0,0"`). It MUST be edited as ten discrete per-book number
  inputs and serialised back to the comma string on save — not as free text.
  Book 0 MUST be visually distinguished as the only book Maple Life reads or
  spends.

### FR-8 SP skill offer

- **FR-8.1** — A `spSkillId` selector offering: None, Improved Max HP Increase
  (Warrior), Improved Max MP Increase (Magician). These two are the only ids
  with a coded prerequisite in `factory/maple_life.go` `prerequisiteFor`.
- **FR-8.2** — For ordinals ≥ 2 the control MUST be **disabled**, with the
  reason stated inline: the client skips step 4 for `m_nCurrentClass >= 2`, so a
  value set there is unreachable, and a player submitting `sp != 0` is rejected
  outright rather than clamped.
- **FR-8.3** — When a skill is selected, the section MUST state which
  prerequisite is granted automatically at level 5, and MUST show the effective
  player cap: `min(10, spPool[0] - 5)`.
- **FR-8.4** — A loaded configuration carrying a `spSkillId` outside the two
  known ids MUST render a non-blocking warning that no prerequisite will be
  granted for it, and MUST preserve the value rather than silently rewriting it.
- **FR-8.5** — A loaded configuration carrying a non-zero `spSkillId` on an
  ordinal ≥ 2 MUST render the blocking error in FR-11 rather than hiding the
  value behind the disabled control.

### FR-9 Starting kit

- **FR-9.1** — Equipment reuses the presets `EquipmentSection` unchanged: its
  props are already `{ templateId, useAverageStats }[]` with add / remove /
  set-average handlers, structurally identical to `maplelife.EquipmentEntry`.
- **FR-9.2** — Inventory reuses the presets `InventorySection` unchanged: its
  props are already `{ templateId, quantity }[]`, structurally identical to
  `maplelife.InventoryEntry`, including its `QuantityInput` draft handling.
- **FR-9.3** — If reuse requires the shared TS entry types to be referenced from
  both places, they MUST be shared by a straightforward move to a common
  location, not by re-exported aliases.

### FR-10 Live preview

- **FR-10.1** — A sticky preview card renders the character the selected class
  and the currently previewed appearance picks produce, using
  `generateCharacterUrl` with: `face` = the picked face, `hair` = picked hair
  style + picked hair colour, `skin` = picked skin tone, `gender` = the selected
  gender (passed explicitly, never inferred), and `equipment` = the row's
  equipment template ids.
- **FR-10.2** — The hair value MUST be composed the way the client composes it —
  `anHairEquip[0] = hairColor + 10 * (hairStyle / 10)` (`design.md` §A3) — so the
  preview matches what a player sees rather than a plausible approximation.
- **FR-10.3** — The card MUST show the combination count the pools currently
  offer (`faces × hairs × hairColors × skinColors`), which is the only readout
  that makes an over- or under-populated pool obvious at a glance.

### FR-11 Validation

Hard rules **block Save**. Each is a mirror of
`services/atlas-character-factory/atlas.com/character-factory/factory/processor.go:388-437`
or a value domain from task-246 `design.md` §A3, and each MUST cite its own rule
in the message shown:

- **FR-11.1** — A hair style id not divisible by 10.
- **FR-11.2** — A hair colour outside `0..9`.
- **FR-11.3** — Any empty appearance pool for a gender that has at least one
  configured class row.
- **FR-11.4** — A non-zero `spSkillId` on an ordinal ≥ 2.
- **FR-11.5** — An SP pool that is not ten integers.
- **FR-11.6** — A class with an `spSkillId` whose `spPool[0]` cannot cover the
  maximum offerable level plus the prerequisite's 5 — i.e. `spPool[0] < 6`, which
  makes even a level-1 investment unsatisfiable.
- **FR-11.7** — A gender with configured class rows but no `looks` row at all.

Soft rules **warn without blocking**:

- **FR-11.8** — An ordinal in `{2,3,4}` (unconfirmed job order — always present,
  by FR-5.3).
- **FR-11.9** — A `spSkillId` outside the two ids with a coded prerequisite.
- **FR-11.10** — A `(ordinal, gender)` row absent from the configuration.

Requirements on the mechanism:

- **FR-11.11** — Validation MUST be expressed as a Zod schema under
  `services/atlas-ui/src/lib/schemas/`, following the `character-presets.schema.ts`
  precedent.
- **FR-11.12** — Save MUST be disabled while any hard rule fails, and the save
  bar MUST report the count of blocking errors. Every failing field MUST be
  individually marked, not just summarised.
- **FR-11.13** — Client-side validation is an operator affordance, not the
  authority. The backend remains the validator of record; the page MUST NOT
  silently rewrite a value to satisfy a rule.

### FR-12 Empty state and seeding from a template

- **FR-12.1** — A tenant whose `mapleLife` is absent or has no classes renders a
  purposeful empty state, not a blank ten-row grid: it states that Maple Life is
  disabled for this tenant, and that using a `Cash/0543` item will fail with
  `ErrMapleLifeNotConfigured`.
- **FR-12.2** — The tenant empty state offers a **Seed from template** action.
  No source-template link is stored on a tenant configuration (verified: nothing
  in `TenantConfigAttributes` records one), so the action MUST resolve candidate
  templates by matching `region` + `majorVersion` + `minorVersion` via
  `useTemplatesByRegionAndVersion`.
- **FR-12.3** — Exactly one match seeds directly after confirmation. Multiple
  matches present a picker. Zero matches, or a matched template whose own
  `mapleLife` is empty, states that plainly and offers no action.
- **FR-12.4** — Seeding populates the working copy only. Nothing is persisted
  until the operator saves, and the save bar MUST show the change as dirty.
- **FR-12.5** — The action is absent on the template page (no `seedFrom` in the
  adapter), which instead offers a plain "add the ten class rows" empty state.

### FR-13 Form state and save

- **FR-13.1** — All edits go to a reducer-held working copy. Save sends the
  whole `mapleLife` block; Discard restores the last loaded value.
- **FR-13.2** — The save bar is registered through
  `useRegisterDetailActionBar`, matching both siblings, and reports dirty state.
- **FR-13.3** — Save is disabled when not dirty, while `isSaving`, or while any
  hard validation rule fails.
- **FR-13.4** — On success, toast and clear dirty state. On failure, toast the
  error and keep the working copy intact.

## 5. API Surface

No new or modified endpoints. The block is already served and accepted by
existing routes:

| Method | Path | Use |
|---|---|---|
| `GET` | `/api/configurations/tenants/{id}` | Read `attributes.mapleLife` |
| `PATCH` | `/api/configurations/tenants/{id}` | Write the merged attributes |
| `GET` | `/api/configurations/templates/{id}` | Read `attributes.mapleLife` |
| `PATCH` | `/api/configurations/templates/{id}` | Write the merged attributes |

Supporting reads, all already in use by the sibling editors:
`/api/data/job-availability` (job names, via `usePresetJobOptions`),
`/api/data/maps` (via `MapPicker`), `/api/data/cosmetics` (via `useFaceIds` /
`useHairIds`), `/api/data/item-strings` (item names), and `/api/assets/...`
(character renders and item icons).

Both PATCHes are whole-attributes merges, not sparse patches — the client sends
`{ ...attributes, ...updates }`. FR-1.3 and FR-1.4 exist because that shape
makes every configuration sub-page a potential eraser of any section it does not
know about.

Error cases are the standard ones already handled by the shared API client;
this task introduces no new response shape.

## 6. Data Model

No database change. The frontend types mirror
`services/atlas-configurations/atlas.com/configurations/tenants/maplelife/rest.go`
(the template package is identical):

```ts
export interface MapleLifeLookOptions {
  gender: number;          // 0 male, 1 female
  faces: number[];         // full item ids, e.g. 20000 / 21000
  hairs: number[];         // normalized style ids, (v/10)*10, e.g. 30030
  hairColors: number[];    // bare digit 0..9
  skinColors: number[];    // bare byte ordinal
}

export interface MapleLifeStatBlock {
  str: number; dex: number; int: number; luk: number; hp: number; mp: number;
}

export interface MapleLifeClassEntry {
  ordinal: number;         // client m_nCurrentClass, 0..4
  gender: number;          // 0 | 1
  jobId: number;
  level: number;
  mapId: number;
  stats: MapleLifeStatBlock;   // AP already spent; skill-EXCLUDED hp/mp
  ap: number;                  // unspent at `level`
  sp: string;                  // ten-book pool, "61,0,0,0,0,0,0,0,0,0"
  spSkillId?: number;          // absent/0 ⇒ no SP step (ordinals >= 2)
  meso: number;
  equipment: { templateId: number; useAverageStats: boolean }[];
  inventory: { templateId: number; quantity: number }[];
}

export interface MapleLifeConfig {
  looks: MapleLifeLookOptions[];
  classes: MapleLifeClassEntry[];
}
```

Constraints, with their source of record:

| Constraint | Source |
|---|---|
| `(ordinal, gender)` is the lookup key; a missing pair ⇒ `ErrClassOrdinalUnknown` | `factory/maple_life.go:20-26` |
| Ordinal 0 = Warrior, 1 = Magician are derived; 2/3/4 order is unconfirmed | `task-246/design.md` §A6 |
| `spSkillId` only reachable for ordinals 0/1 | `task-246/design.md` §A4, `processor.go:424-427` |
| Only `WarriorImprovedMaxHpIncreaseId` / `MagicianImprovedMaxMpIncreaseId` carry a coded prerequisite | `factory/maple_life.go:33-42` |
| `spPool[0]` must cover the chosen level + 5; player `sp` ≤ 10 | `processor.go:428-437` |
| `stats.hp/mp` exclude the SP skill's `29 × effectX` contribution | `factory/maple_life.go:93-105` |
| Empty `classes`, or a gender with no `looks` row ⇒ `ErrMapleLifeNotConfigured` | `processor.go:390-405` |
| A submitted look value not in the allow-list ⇒ `ErrLookInvalid` | `processor.go:405-422` |
| Value domains: face = item id, hair = `(v/10)*10`, hairColor = `0..9`, skin = byte | `task-246/design.md` §A3 |

Reference data ships in four GMS templates today —
`services/atlas-configurations/seed-data/templates/template_gms_{83,87,92,95}_1.json` —
each with 2 look rows and 10 class rows (jobs 100/200/300/400/500 at level 30).

## 7. Service Impact

**`atlas-ui`** — the whole change:

- New: `pages/TenantsMapleLifePage.tsx`, `pages/TemplatesMapleLifePage.tsx`, and
  `components/features/characters/maple-life/*` (editor, reducer state, class
  selector, identity / appearance / progression / SP / kit sections, preview,
  seed-from-template dialog).
- New: `lib/schemas/maple-life.schema.ts`.
- Modified: `App.tsx` (two routes), `TenantDetailLayout.tsx`,
  `TemplateDetailLayout.tsx`, `lib/breadcrumbs/routes.ts`,
  `types/models/template.ts`, `services/api/tenants.service.ts`.
- Modified for reuse: `AppearancePoolSection` generalised per FR-6.3; possibly a
  move of the shared equipment/inventory entry types per FR-9.3.

**`atlas-configurations`** — no change. Read/write dependency only; both REST
models already carry the block.

**`atlas-character-factory`** — no change. Source of the validation rules the
page mirrors.

**`atlas-channel`** — no change.

## 8. Non-Functional Requirements

- **Multi-tenancy** — All data reads go through the existing tenant-scoped API
  client, which injects `TENANT_ID` / `REGION` / `MAJOR_VERSION` /
  `MINOR_VERSION`. The template page edits a template, but its supporting
  lookups (jobs, cosmetics, items, maps, renders) are still tenant-scoped
  through the active tenant, exactly as the character-templates page already
  handles this split. No new header path.
- **Rendering cost** — Each appearance entry is one character render request.
  With the shipped seed (3 faces + 3 hairs + 4 hair colours + 4 skin tones = 14
  per gender) this is bounded and comparable to the character-templates page.
  Thumbs MUST use `loading="lazy"` and the existing `AppearanceThumb` failure
  handling; a failed render degrades to the id, never to a broken image.
- **Accessibility** — The gender and ordinal segmented controls MUST implement
  the same roving-tabindex keyboard handling as `TemplateSelector`
  (arrow keys, Home/End). Disabled controls MUST carry an accessible
  explanation, not just visual dimming.
- **Observability** — No new backend telemetry. Save failures surface as toasts
  with the server's message, as the siblings do.
- **Security** — No new privilege boundary; the page is inside the existing
  configuration admin surface. No secret is displayed or logged.
- **Testing** — Vitest coverage MUST include: the reducer's load/edit/discard
  transitions, every hard validation rule from FR-11 (each asserted
  individually), the FR-1.3 and FR-1.4 round-trip preservation tests, the
  ordinal-≥2 SP disabling, the deep-link clamp, and the FR-12 seed-from-template
  resolution including the zero-match and multi-match branches.
- **Guidelines** — The work is subject to the `frontend-dev-guidelines` skill
  and the FE-* checklist; `frontend-guidelines-reviewer` runs before the PR.

## 9. Open Questions

1. **Ordinal→job order for 2/3/4 remains unpinned.** Task-246 `design.md` §A6
   left it to live observation and it has not been done. This task surfaces the
   uncertainty (FR-5.3) but does not resolve it; resolving it is a seed-data
   change, not a UI change.
2. **Does `mapleLife` actually survive the existing PATCH round-trip?** FR-1.3
   requires proving it. If it does not, some tenants may already have lost their
   block through an unrelated configuration save, and a data-repair note may be
   needed alongside the fix.
3. **Skin-tone enumeration.** `AppearanceBrowserDialog` offers 0–9 with a
   comment that no enumeration endpoint exists and seed data uses 0–3. Maple
   Life inherits that uncertainty unchanged; no new answer is proposed here.
4. **Should `level` be constrained below 200?** The shipped seed uses 30 for
   every class and the client's dialog is described as a level-30 offering, but
   nothing in the factory enforces an upper bound. FR-5.5 proposes 1–200; a
   tighter bound would need evidence.

## 10. Acceptance Criteria

- [ ] `mapleLife` is declared on `TenantConfigAttributes` and `TemplateAttributes`,
      and the domain types live beside `CharacterTemplate` / `CharacterPreset`.
- [ ] A test proves a `mapleLife` block survives a save issued from a sibling
      configuration page unchanged, and that a Maple Life save leaves every other
      section unchanged.
- [ ] `/tenants/:id/character/maple-life` and `/templates/:id/character/maple-life`
      route, render, appear in both detail rails, and produce correct breadcrumbs.
- [ ] The editor shows a fixed 5 × 2 class grid with a gender toggle and an ordinal
      selector; there is no way to add, remove, or renumber an ordinal.
- [ ] Ordinals 2/3/4 carry a persistent unconfirmed-order notice; 0 and 1 carry a
      derived badge.
- [ ] The SP-skill control is disabled with a stated reason on ordinals ≥ 2, and a
      loaded non-zero `spSkillId` on such an ordinal is a blocking error.
- [ ] Appearance pools render real character sprites, add via
      `AppearanceBrowserDialog`, remove per entry, and state each dimension's value
      domain; `AppearancePoolSection` is generalised, not duplicated, and the
      character-templates page is unchanged in behaviour.
- [ ] Equipment and inventory reuse the presets `EquipmentSection` /
      `InventorySection` without forking them.
- [ ] The SP pool is edited as ten discrete book inputs and serialised back to the
      comma string; book 0 is visually distinguished.
- [ ] Stats are labelled as the skill-excluded midpoint; `ap`/`sp` are labelled as
      unspent-at-level.
- [ ] Every hard rule in FR-11 blocks Save with a field-level marker and a message
      citing its rule; every soft rule warns without blocking.
- [ ] The live preview composes hair as `hairColor + 10 * (hairStyle / 10)`, passes
      gender explicitly, and reports the offered combination count.
- [ ] A tenant with no `mapleLife` shows the disabled-state message and a
      Seed-from-template action that resolves candidates by region + version, with
      the zero-match and multi-match branches both handled.
- [ ] Save/Discard run through `useRegisterDetailActionBar`; Save is disabled when
      clean, saving, or invalid.
- [ ] `tools/verify.sh` exits 0 with no flags.
- [ ] `frontend-guidelines-reviewer` and `plan-adherence-reviewer` both pass before
      the PR is opened.
