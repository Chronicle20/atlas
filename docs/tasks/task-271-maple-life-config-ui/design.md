# Maple Life Configuration Editor — Design

Task: `task-271-maple-life-config-ui`
Phase: 2 (design)
Input: `docs/tasks/task-271-maple-life-config-ui/prd.md` (approved)
Status: Draft for review
Created: 2026-08-27

---

## 1. Scope and posture

Frontend-only. `atlas-configurations` already serves and accepts the block on
both configuration documents:

```
services/atlas-configurations/atlas.com/configurations/tenants/rest.go:23
    MapleLife    maplelife.RestModel  `json:"mapleLife"`
services/atlas-configurations/atlas.com/configurations/templates/rest.go:23
    MapleLife    maplelife.RestModel  `json:"mapleLife"`
```

No Go file changes. Every rule the UI enforces is a mirror of
`services/atlas-character-factory/atlas.com/character-factory/factory/processor.go`
`resolveMapleLifePreset` or a value domain from task-246 `design.md` §A3; the
backend stays the validator of record (FR-11.13).

The design follows the two sibling editors verbatim wherever they already solve
a problem — one shared editor plus a per-context adapter, a reducer-held working
copy seeded once behind a `loaded` guard, a URL-deep-linked selection written
with `{ replace: true }`, and a save bar registered through
`useRegisterDetailActionBar`. Deviations from that shape are enumerated in §9
with their reasons; there are five, and each is forced by something the siblings
do not have to do.

---

## 2. Module layout

```
src/types/models/template.ts                  (M) domain types + TemplateAttributes.mapleLife
src/services/api/tenants.service.ts           (M) TenantConfigAttributes.mapleLife
src/App.tsx                                   (M) two lazy routes
src/lib/breadcrumbs/routes.ts                 (M) two patterns + two route constants
src/components/features/tenants/TenantDetailLayout.tsx      (M) rail entry
src/components/features/templates/TemplateDetailLayout.tsx  (M) rail entry
src/components/DetailActionBarContext.tsx     (M) optional blocking-issue field
src/components/features/characters/templates/SaveBar.tsx    (M) render that field
src/components/features/characters/templates/AppearancePoolSection.tsx (M) generalise

src/lib/schemas/maple-life.schema.ts          (N) Zod hard rules
src/pages/TenantsMapleLifePage.tsx            (N)
src/pages/TemplatesMapleLifePage.tsx          (N)
src/components/features/characters/maple-life/
    MapleLifeEditor.tsx        (N) adapter contract, load/deep-link/save-bar wiring
    mapleLifeEditorState.ts    (N) draft shape, reducer, projectForSave, isDirty
    mapleLifeWarnings.ts       (N) soft (non-blocking) rules
    mapleLifeLoadout.ts        (N) §A3 hair composition + render loadout
    ClassSelector.tsx          (N) gender + ordinal segmented controls
    IdentitySection.tsx        (N) jobId / level / mapId + provenance notice
    AppearancePoolsSection.tsx (N) four pools for the selected gender
    ProgressionSection.tsx     (N) stats / ap / meso / ten SP books
    SpSkillSection.tsx         (N) spSkillId offer, ordinal >= 2 disabling
    StartingKitSection.tsx     (N) presets EquipmentSection + InventorySection
    MapleLifePreviewCard.tsx   (N) sticky render + combination count
    SeedFromTemplateDialog.tsx (N) FR-12
    EmptyState.tsx             (N) tenant vs template empty copy
```

Each unit is independently testable: the reducer and the two rule modules are
pure and get `.test.ts` files with no React; every section component takes a
plain entry/handler pair and gets a `.test.tsx`.

---

## 3. Types

### 3.1 Domain types

Added to `src/types/models/template.ts`, beside `CharacterTemplate` /
`CharacterPreset`, mirroring `maplelife/rest.go` field-for-field:

```ts
export interface MapleLifeLookOptions {
  gender: number;
  faces: number[];
  hairs: number[];
  hairColors: number[];
  skinColors: number[];
}

export interface MapleLifeStatBlock {
  str: number; dex: number; int: number; luk: number; hp: number; mp: number;
}

export interface MapleLifeClassEntry {
  ordinal: number;
  gender: number;
  jobId: number;
  level: number;
  mapId: number;
  stats: MapleLifeStatBlock;
  ap: number;
  sp: string;
  spSkillId?: number;   // `json:"spSkillId,omitempty"` — absent means no SP step
  meso: number;
  equipment: EquipmentEntry[];
  inventory: InventoryEntry[];
}

export interface MapleLifeConfig {
  looks: MapleLifeLookOptions[];
  classes: MapleLifeClassEntry[];
}
```

`spSkillId` stays optional to match the Go `omitempty`: a class with no SP step
must round-trip as an absent key, not as `0`, so a save from this page does not
add a field the seed file does not carry.

### 3.2 Shared entry types (FR-9.3)

`EquipmentEntry` / `InventoryEntry` are `CharacterPresetEquipmentEntry` /
`CharacterPresetInventoryEntry` **renamed in place**, not aliased. Renaming is
mechanical and small — the whole non-test surface is four references:

```
src/types/models/template.ts:32,37,63,64
src/components/features/characters/presets/EquipmentSection.tsx:2,16
src/components/features/characters/presets/InventorySection.tsx:2,8
```

(`services/api/tenants.service.ts:34,39` declares its own private copies; see
§3.3.) The old names are deleted, not re-exported — CLAUDE.md's "prefer
straightforward moves over re-exported type aliases".

**Rejected:** having `MapleLifeClassEntry.equipment` typed
`CharacterPresetEquipmentEntry[]`. Structurally correct, but it names a Maple
Life field after presets and would mislead every later reader.

### 3.3 Configuration attributes

`TemplateAttributes` gains `mapleLife?: MapleLifeConfig`.

`TenantConfigAttributes` in `tenants.service.ts` gains the same optional
property. That file today re-declares `characters.templates` and the preset
entry types inline rather than importing from `types/models/template.ts`. That
duplication is pre-existing and **out of scope**; `mapleLife` is added as a
single `import type { MapleLifeConfig }` rather than a fifth inline copy, so the
new field has exactly one definition.

Optionality mirrors the existing `cashShop?` precedent: a configuration with no
block decodes to `undefined`, which §6 of the PRD rules a legitimate state.

---

## 4. Round-trip preservation (FR-1.3 / FR-1.4)

Both write paths are whole-attributes merges, so an *undeclared* key already
survives at runtime — TypeScript types do not strip at runtime:

- Tenant: `tenantsService.updateTenantConfiguration` builds
  `attributes: { ...tenant.attributes, ...updatedAttributes }`
  (`tenants.service.ts:305`).
- Template: `TemplatesCharacterTemplatesPage` passes
  `updates: { ...template.attributes, characters: {...} }`, and
  `templatesService.update` sends that document whole
  (`templates.service.ts:325-336`).

Two things on the template path were checked and are benign:
`validateTemplate` (`templates.service.ts:96`) only *adds* field errors and
never rewrites the document, and `sortTemplate` (`:61`) spreads
`...template.attributes` and touches only `npcs` / `worlds` / `socket`.

So the expected outcome of FR-1.3 is "already preserved". The PRD is right that
this is unverified, and the tests are written to *prove* it rather than assume
it:

- `tenants.service.test.ts` — seed a `TenantConfig` whose attributes carry a
  `mapleLife` block, call `updateTenantConfiguration` with a
  `characters`-only update, assert the captured PATCH body's
  `data.attributes.mapleLife` deep-equals the seeded block.
- `templates.service.test.ts` — same shape through `update`.
- `MapleLifeEditor.test.tsx` — save from this page and assert the adapter
  receives only a `mapleLife` update, and that the page-level merge leaves
  `characters`, `npcs`, `worlds`, `socket`, `cashShop` untouched in the PATCH
  body.

If a run shows the block does *not* survive, the fix lands here (PRD FR-1.3
says so explicitly) and the open question about already-damaged tenants gets
recorded in `progress.md` for the operator, not silently.

---

## 5. Editor architecture

### 5.1 Adapter

```ts
export interface MapleLifeEditorAdapter {
  mapleLife: MapleLifeConfig | undefined;
  isLoading: boolean;
  error: Error | null;
  save: (config: MapleLifeConfig, onSuccess: () => void) => void;
  isSaving: boolean;
  /** Tenant context only; absence hides the seed-from-template action. */
  seedFrom?: { region: string; majorVersion: number; minorVersion: number };
}
```

Identical in spirit to `TemplatesEditorAdapter` and `PresetsEditorAdapter`,
including the optional context-only capability (`PresetsEditorAdapter.apply` is
the precedent for `seedFrom`).

Pages are thin. The tenant page reads `useTenantConfiguration` and mutates via
`useUpdateTenantConfiguration({ tenant, updates: { mapleLife } })`; the template
page reads `useTemplate` and mutates via
`useUpdateTemplate({ id, updates: { ...template.attributes, mapleLife } })`.
Both toast on success and error, copying the sibling pages' wording shape.

Neither page needs the presets pages' post-save refetch: nothing in `mapleLife`
is server-assigned (no ids), so the request echo is the truth.

### 5.2 Working-copy shape

The reducer does **not** hold `MapleLifeConfig` directly. Two fields need a
UI-shaped draft:

```ts
export interface MapleLifeClassDraft
  extends Omit<MapleLifeClassEntry, "sp"> {
  /** Ten book values, parsed from `sp`. */
  spBooks: number[];
  /** The `sp` string exactly as loaded. Kept so an unparseable or
   *  non-ten-book value is preserved verbatim rather than rewritten. */
  spRaw: string;
  /** False when the (ordinal, gender) row is absent from the loaded
   *  configuration. Never sent; see materialisation below. */
  present: boolean;
}

export interface MapleLifeEditorState {
  drafts: MapleLifeClassDraft[];   // always exactly 10, ordinal-major
  looks: MapleLifeLookOptions[];   // one per configured gender
  baseline: MapleLifeConfig;       // last loaded/saved, for dirty + discard
  ordinal: number;                 // 0..4
  gender: number;                  // 0 | 1
  picks: Record<string, PreviewPicks>;  // key `${gender}` — looks are gender-split
  loaded: boolean;
}
```

**The fixed grid is a view over a sparse array.** `load` expands the loaded
`classes` into all ten `(ordinal, gender)` slots; a slot with no source row gets
`present: false` and neutral zero values. `projectForSave` emits only slots with
`present: true`, in ordinal-major order, so:

- FR-4.3 holds — an absent row renders with its "not configured" empty state and
  is not invented on load.
- Any edit to an absent slot sets `present: true`, which is exactly
  "selecting it and editing any field materialises the row".
- A round-trip with no edits emits precisely the rows that were loaded.

`isDirty` is `JSON.stringify(projectForSave(state)) !== JSON.stringify(baseline)`
— the same deep-compare-by-serialisation the siblings use, on the projected
(wire) shape so a draft-only field can never read as dirty. `looks` participates
because it is part of the projection.

**SP round-trip.** `spBooks` is `spRaw.split(",")` parsed to numbers. Serialising
back is `spBooks.join(",")`. When `spRaw` does not parse to exactly ten integers,
`spBooks` is left empty and the slot is flagged blocking (FR-11.5); `spRaw` is
emitted unchanged on save so the page never rewrites a value it does not
understand.

> Note for the record: the backend's `parseSPPool`
> (`factory/maple_life.go:142`) accepts *any* comma count and only reads
> `pool[0]`. "Exactly ten" is a UI convention matching what atlas-character
> persists, stricter than the server. It is enforced as a hard rule per FR-11.5,
> and the message says so.

### 5.3 Reducer actions

```
load(config)            select(ordinal, gender)
setIdentity(field, v)   setStat(stat, v)        setScalar(field, v)   // ap, meso, level
setSpBook(index, v)     setSpSkillId(v | undefined)
addLookEntry(dim, id)   removeLookEntry(dim, idx)
addEquipment(id)        removeEquipment(idx)     setEquipmentAvg(idx, v)
addInventory(id)        removeInventory(idx)     setInventoryQty(idx, v)
setPreviewPick(pick, idx)
seedFromTemplate(config)
discard()               savedOk()
```

Every mutating action funnels through an `updateSelected` helper that clones the
selected draft and sets `present: true` — one place owns materialisation.

`seedFromTemplate` replaces `drafts` + `looks` wholesale from a donor
`MapleLifeConfig` but leaves `baseline` alone, so the result reads dirty
(FR-12.4).

### 5.4 Selection and deep link

Two params, `?ordinal=` (0..4) and `?gender=` (0|1), applied **once** on load
with `deps: [state.loaded]`, clamped to a valid value, and written back with
`{ replace: true }` — the pattern and the reasoning are lifted from
`CharacterTemplatesEditor`'s `?tpl=` effect (which documents why a
length-watching effect races the router). Here the grid is fixed at ten, so
there is no length to watch at all and the on-load-only rule is unconditional;
every later selection change writes the URL directly from the click handler.

`ClassSelector` renders two `role="tablist"` segmented controls with the roving
tabindex, arrow-key, and Home/End handling copied from `TemplateSelector`
(NFR: accessibility). Ordinal segment labels are `<ordinal> · <job name>` with
the derived / unconfirmed badge (FR-4.4); the job name comes from
`usePresetJobOptions`, falling back to the raw `jobId` while it is pending.

---

## 6. Validation

### 6.1 Split: Zod for hard, a pure function for soft

Zod is pass/fail; a soft rule that must render as a visible warning without
failing cannot be a Zod issue. So:

- `src/lib/schemas/maple-life.schema.ts` — `mapleLifeSchema`, a Zod schema over
  the **projected wire shape**, carrying every hard rule. Follows the
  `character-presets.schema.ts` precedent (plain `z.object`, `safeParse` at the
  call site).
- `src/components/features/characters/maple-life/mapleLifeWarnings.ts` — a pure
  `mapleLifeWarnings(state): Warning[]` for the soft rules.

Both consume the projection, not the draft, so what is validated is what is sent.

### 6.2 Hard rules → Zod placement

| Rule | Where | Message cites |
|---|---|---|
| FR-11.1 hair style not divisible by 10 | `looks.hairs` element refine | task-246 `design.md` §A3 |
| FR-11.2 hair colour outside 0..9 | `looks.hairColors` element `min(0).max(9)` | §A3 |
| FR-11.5 `sp` not ten integers | `classes` element refine on `sp` | atlas-character SP pool shape |
| FR-11.4 non-zero `spSkillId` on ordinal ≥ 2 | `classes` element refine | `processor.go:424-427`, §A4 |
| FR-11.6 `spPool[0] < 6` when `spSkillId` set | `classes` element refine | `processor.go:428-437` |
| FR-11.3 empty pool for a gender with class rows | top-level `superRefine` | `processor.go:405-422` |
| FR-11.7 gender with class rows but no `looks` row | top-level `superRefine` | `processor.go:397-405` |

The last two are cross-entity and cannot be expressed inside an element schema —
`superRefine` on the root is the only correct home, and it emits issues at
`["looks"]` / `["classes", i]` paths so the UI can still mark fields.

FR-11.6's threshold is derived, not guessed: the server computes
`needed = in.SP + 5` when the skill has a coded prerequisite and rejects
`needed > pool[0]`. The smallest non-zero investment is `in.SP = 1`, so
`pool[0] < 6` makes *every* investment unsatisfiable. The message states that
derivation.

### 6.3 Field marking

`safeParse` issues carry a `path`. `MapleLifeEditor` converts them once into
`Map<string, string[]>` keyed by a dotted path and passes the relevant slice
down to each section, which renders per-field messages. This satisfies FR-11.12's
"every failing field individually marked" without any section knowing about Zod.

Warnings are passed the same way, in a separate map, and render in the
`warning-foreground` treatment `AppearancePoolSection` already uses.

### 6.4 Save bar (FR-11.12)

`DetailActionBarConfig` today is `{ dirty, isSaving, onSave, onDiscard }` — no
way to express "dirty but not saveable". Add one optional field:

```ts
export interface DetailActionBarConfig {
  dirty: boolean;
  isSaving: boolean;
  onSave: () => void;
  onDiscard: () => void;
  /** Count of blocking validation errors. > 0 disables Save and is
   *  reported in the bar. Omitted means "no validation gate". */
  blockingIssues?: number;
}
```

`SaveBar` disables Save when `(blockingIssues ?? 0) > 0` and shows
`"N blocking errors"` in place of the "Unsaved changes" text; Discard stays
enabled (an operator must be able to back out of an invalid edit).
`useRegisterDetailActionBar` adds `blockingIssues` to the primitive dependency
list it already computes for `dirty` / `isSaving`, so the re-push behaviour is
unchanged. Both existing callers omit the field and are behaviourally identical.

**Rejected:** the presets editor's approach — validate inside `onSave` and toast
the first issue. It is the cheaper change but it fails FR-11.12 twice: Save is
not disabled, and only one issue is ever surfaced.

---

## 7. Reuse decisions

### 7.1 `AppearancePoolSection` generalisation (FR-6.3)

Today it takes `template: CharacterTemplate`, derives `pool = template[dimension]`
and calls `buildVariantLoadout(template, picks, dimension, id)` itself. Both
couplings are to `CharacterTemplate`. The generalisation inverts them:

```ts
interface AppearancePoolSectionProps {
  dimension: AppearancePoolKey;
  title: string;
  pool: number[];
  selectedIndex: number;
  variantLoadout: (dimension: AppearancePoolKey, id: number) => CharacterLoadout;
  onPick: (index: number) => void;
  onRemoveEntry: (index: number) => void;
  renderAddDialog: (open: boolean, onOpenChange: (o: boolean) => void) => ReactNode;
  /** Extra copy under the header — FR-6.5 value domain, FR-6.6 allow-list note. */
  description?: ReactNode;
}
```

`template`, `picks`, and the internal `PICK_KEY_BY_POOL` lookup leave the
component; the caller resolves them. `CharacterTemplatesEditor` becomes the
first caller and keeps identical rendered output — it passes
`pool={template[dimension]}`, `selectedIndex={picks[PICK_KEY_BY_POOL[dimension]]}`,
`variantLoadout={(dim, id) => buildVariantLoadout(template, picks, dim, id)}`,
and maps `onPick(idx)` back onto its `setPreviewPick` action. The existing
`AppearancePoolSection.test.tsx` is updated to the new props; its assertions do
not change.

Note the empty-pool warning currently hard-coded in the component ("character
creation will fail while this pool is empty") stays as-is — it is true for both
callers.

**Rejected:** a generic type parameter over the owning object. It would keep the
callers shorter but pushes `CharacterTemplate`'s indexing shape into Maple
Life's draft type, which does not have it (`looks` are gender-split, not
per-class).

### 7.2 `EquipmentSection` / `InventorySection` (FR-9.1, FR-9.2)

Used unchanged after the §3.2 rename. Their props are already
`{ equipment, onAdd, onRemove, onSetAvg }` and
`{ inventory, onAdd, onRemove, onSetQty }` over exactly the two entry shapes
`maplelife.EquipmentEntry` / `maplelife.InventoryEntry` project onto. No fork,
no wrapper beyond a `StartingKitSection` that composes the two and adds the
section heading.

### 7.3 `BaseStatsSection`

**Not** reused as a component: it is typed against `CharacterPresetAttributes`
and its footer copy ("Written verbatim to the created character") is wrong for
Maple Life, where stats are the *skill-excluded* midpoint (FR-7.2). What is
reused is the mechanism the PRD actually names — `useSyncedNumberInput` — for
every number input on the page (stats, ap, meso, level, the ten SP books). The
mid-edit clobber bug is not re-solved.

### 7.4 `MapPicker`, `AppearanceBrowserDialog`, `AppearanceThumb`, `ItemSearchCombobox`

Used unchanged. `AppearanceBrowserDialog` already covers exactly the four
dimensions and takes `gender` plus a `variantLoadout` callback, which is all
Maple Life needs.

---

## 8. Preview and hair composition

New module `mapleLifeLoadout.ts`, deliberately separate from
`templates/previewLoadout.ts`:

```ts
export function composeHair(hairStyle: number, hairColor: number): number {
  return hairColor + 10 * Math.floor(hairStyle / 10);
}
```

This is the client's own expression, `anHairEquip[0] = hairColor + 10 * (hairStyle / 10)`
(task-246 `design.md` §A3, integer division). The templates module does
`baseHair + colorDigit`, which is the *same value* only when the base is already
a multiple of ten. Maple Life normalises explicitly because FR-11.1 exists
precisely to catch a base that is not — and a preview that silently agreed with
a bad value would hide the error the page is supposed to surface.

`buildMapleLifeLoadout(entry, look, picks)` returns
`{ skin, hair: composeHair(...), face, equipment, gender: <selected gender> }`.
Gender is passed explicitly from the selection, never inferred (FR-10.1).
Equipment is the draft's own `equipment[].templateId` list mapped onto the
canonical render slots; the slot table is reused from
`previewLoadout.EQUIP_SLOT_BY_POOL` where the ids match and otherwise falls back
to letting the render service place them — a Maple Life kit carries seven equips
(seed row: 1040021, 1060016, 1072039, 1302008, 1442001, 1422001, 1312005), more
than the four-slot template pool covers.

The card shows `faces × hairs × hairColors × skinColors` for the selected gender
(FR-10.3), and is `sticky` in the right column at `lg:` and above, ordered first
on narrow screens — the same two-column grid `CharacterTemplatesEditor` uses.

---

## 9. Deviations from the sibling editors

Five, each forced:

1. **Fixed ten-slot grid instead of a variable list.** The client cycles
   `m_nCurrentClass % 5` and there is no add/remove. The draft array is always
   length 10; sparseness is carried by `present`, not by array length.
2. **Draft type ≠ wire type.** `sp` and row presence need UI shapes, so there is
   a `projectForSave`. `CharacterTemplatesEditor` has none (its working copy is
   the wire shape); `CharacterPresetsEditor` already has one, and this follows it.
3. **Validation gates Save.** Requires the `DetailActionBarConfig.blockingIssues`
   extension in §6.4. The siblings validate at click time.
4. **Two deep-link params instead of one.** `looks` and `classes` are both
   gender-split, so gender is a first-class axis, not a field of the selection.
5. **A separate hair-composition module.** §8.

---

## 10. Empty state and seeding (FR-12)

`MapleLifeConfig` is treated as empty when it is `undefined` or
`classes.length === 0` — the same test `resolveMapleLifePreset` applies before
returning `ErrMapleLifeNotConfigured` (`processor.go:390-392`). The empty state
names that error and states that a `Cash/0543` use will fail.

**Candidate resolution.** `useTemplatesByRegionAndVersion` cannot serve FR-12.3:
`templatesService.getByRegionAndVersion` calls `api.getOne` and returns
`[sortTemplate(response)]` (`templates.service.ts:457-474`) — a one-element array
by construction, so a multi-match branch could never be reached through it.

The dialog therefore uses `useTemplates()` (`getAll`, full attributes) and
filters client-side on `region` + `majorVersion` + `minorVersion` from the
adapter's `seedFrom`. Candidates whose own `mapleLife` is empty are listed as
ineligible with the reason shown rather than hidden, so "the template you expect
has no block" is visible rather than looking like a zero-match.

- 1 eligible match → confirm, then `seedFromTemplate`.
- \>1 → a picker listing `id`, region/version, and class/look counts.
- 0 eligible → state it plainly, no action offered.

The seeded value is deep-cloned (`structuredClone`) into the working copy; the
donor template is never mutated and nothing is persisted until Save (FR-12.4).

The template page supplies no `seedFrom`, which hides the action; its empty
state offers "Add the ten class rows", which materialises all ten drafts with
`present: true` at neutral values and marks the page dirty (FR-12.5).

---

## 11. Semantics surfaced in the UI

The three things an operator cannot read off the numbers, and where each lives:

- **Ordinal provenance (FR-4.4, FR-5.3).** Ordinals 0/1 carry a "derived" badge
  in the selector and a short note in Identity ("derived from the client's own
  step-skip"). Ordinals 2/3/4 carry an "unconfirmed" badge and a persistent,
  non-dismissible notice citing task-246 `design.md` §A6. The job field stays
  editable in both cases. The notice text names the fix — pin it against a live
  channel log — because this task does not resolve it.
- **`spSkillId` inertness (FR-8.2).** The control is `disabled` with
  `aria-describedby` pointing at the inline reason, not merely dimmed. A loaded
  non-zero value on such an ordinal is *shown*, not hidden, and is a blocking
  error (FR-8.5, FR-11.4).
- **Allow-list semantics (FR-6.6).** One line per pool section: the client
  sources its carousel from WZ and the server only checks membership, so a list
  that diverges from the client's options produces player-visible `ErrLookInvalid`
  rejections. Each section also states its value domain (FR-6.5): faces are full
  item ids (20000 / 21000), hairs are normalised style ids, hair colours are bare
  digits 0..9, skin tones are bare byte ordinals.

`SpSkillSection` offers exactly three options — None, `WarriorImprovedMaxHpIncreaseId`
(1000001), `MagicianImprovedMaxMpIncreaseId` (2000001) — because those are the only
two with a coded prerequisite in `prerequisiteFor` (`factory/maple_life.go:33-42`);
the ids are confirmed against the shipped seed. When one is selected it states the
prerequisite granted at level 5 and the effective cap `min(10, spBooks[0] - 5)`
(FR-8.3). A loaded `spSkillId` outside the two is preserved and warned about,
never rewritten (FR-8.4, FR-11.9).

---

## 12. Testing

Pure modules (no React):

- `mapleLifeEditorState.test.ts` — load expands 10 slots and marks presence
  correctly from a sparse config; `projectForSave` emits only present rows in
  ordinal-major order; an untouched load round-trips byte-identically; editing an
  absent row materialises it; `discard` restores baseline; `savedOk` rebases;
  `isDirty` is false after load and true after any edit; SP parse/serialise
  round-trips, and an unparseable `sp` is preserved verbatim.
- `maple-life.schema.test.ts` — each hard rule FR-11.1 … FR-11.7 asserted
  individually, both the failing case and the adjacent passing case, plus the
  issue `path` each emits.
- `mapleLifeWarnings.test.ts` — FR-11.8/9/10, each individually.
- `mapleLifeLoadout.test.ts` — `composeHair` against the §A3 expression,
  including a non-multiple-of-ten base (proves it normalises rather than adding).

Component:

- `MapleLifeEditor.test.tsx` — seed-once guard (a second adapter delivery does
  not clobber an edit); deep-link clamp for both params; Save disabled when
  clean / saving / invalid, with the blocking count rendered; save payload
  contains only `mapleLife`.
- `ClassSelector.test.tsx` — roving tabindex, arrows, Home/End; badges per
  ordinal.
- `SpSkillSection.test.tsx` — control disabled with an accessible reason at
  ordinals ≥ 2; loaded non-zero value visible and blocking.
- `AppearancePoolsSection.test.tsx` — thumbs render, add/remove dispatch,
  emptied pool blocks.
- `SeedFromTemplateDialog.test.tsx` — zero / one / multi eligible, and the
  "matched but its block is empty" branch.
- `MapleLifePreviewCard.test.tsx` — hair composition and combination count.
- `AppearancePoolSection.test.tsx` — updated to the new props; existing
  assertions unchanged (proves FR-6.3's "templates page unchanged in behaviour").

Service:

- `tenants.service.test.ts` / `templates.service.test.ts` — the FR-1.3 and
  FR-1.4 preservation assertions described in §4.

Route/nav:

- Breadcrumb tests for both new patterns under `lib/breadcrumbs`, matching the
  existing `/character/templates` coverage.

---

## 13. Explicitly not built

Per PRD §2, and re-affirmed here so the plan does not drift into them: prototype
Options B and C; a tenant-vs-template drift indicator; cash-shop commodity
authoring for `5430xxx`/`5431xxx`/`5432xxx`; any change to
`atlas-configurations`, `atlas-character-factory`, or `atlas-channel`; adding,
removing, or renumbering class ordinals; and resolving the ordinal→job order for
2/3/4 (surfaced, not answered).

---

## 14. Open items carried forward from the PRD

1. **Ordinal→job order for 2/3/4** stays unpinned (PRD OQ-1). Surfaced by
   FR-5.3; resolving it is a seed-data change.
2. **Does `mapleLife` survive the existing PATCH?** §4 argues it does and
   specifies the tests that decide it. If a test fails, the fix is in scope and
   a data-repair note is written to the task folder.
3. **Skin-tone enumeration** — inherited from `AppearanceBrowserDialog`
   unchanged (offers 0–9; seed uses 0–3). No new answer proposed.
4. **`level` upper bound** — FR-5.5's 1–200 is adopted as specified. The backend
   enforces no upper bound and `ClassEntry.Level` is a `byte`, so anything above
   255 is unrepresentable regardless; 200 is a UI convention, and the input is a
   bound, not a silent clamp of a loaded value.
