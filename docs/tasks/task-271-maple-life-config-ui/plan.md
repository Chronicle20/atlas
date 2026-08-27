# Maple Life Configuration Editor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Maple Life configuration sub-page to atlas-ui, in both the tenant
and template contexts, so an operator can read, validate, and edit the
`mapleLife` block that today is reachable only by `curl`.

**Architecture:** Frontend-only. One shared `MapleLifeEditor` component driven by
a per-context `MapleLifeEditorAdapter`, exactly mirroring the two sibling
editors (`CharacterTemplatesEditor`, `CharacterPresetsEditor`): a reducer-held
working copy seeded once behind a `loaded` guard, a URL-deep-linked selection
written with `{ replace: true }`, and a save bar registered through
`useRegisterDetailActionBar`. The class grid is a **fixed** 5 ordinals x 2
genders; sparseness is carried by a `present` flag on the draft, never by array
length, and `projectForSave` emits only present rows. Hard validation is a Zod
schema over the projected wire shape and gates Save via a new
`DetailActionBarConfig.blockingIssues` field; soft rules are a separate pure
function that warns without blocking.

**Tech Stack:** React 19, TypeScript, react-router-dom 7, Zod 4, Vitest 4,
@testing-library/react 16, Tailwind, shadcn/ui.

**Spec:** `docs/tasks/task-271-maple-life-config-ui/design.md`
(PRD: `docs/tasks/task-271-maple-life-config-ui/prd.md`)

## Global Constraints

- **Module root for every command in this plan:** `services/atlas-ui`. Run
  `cd services/atlas-ui` once, then `npx vitest run <path>`, `npm run lint`,
  `npm run build` (`tsc -b && vite build`). Test paths below are relative to
  `services/atlas-ui`.
- **No Go changes.** `atlas-configurations`, `atlas-character-factory`, and
  `atlas-channel` are read-only references. Both REST models already carry the
  block (`tenants/rest.go:23`, `templates/rest.go:23`).
- **Guidelines:** the `frontend-dev-guidelines` skill and the FE-* checklist
  apply to every task. `frontend-guidelines-reviewer` runs before the PR.
- **No placeholders.** Per CLAUDE.md, a landed commit must not contain a
  deferred-work comment, a stubbed handler, or an unimplemented status
  response. No task below specifies one; if a step seems to need one, the step
  is under-specified — raise it rather than landing a placeholder.
- **Never silently rewrite a loaded value** (FR-11.13). An unparseable `sp`, an
  unknown `spSkillId`, an out-of-domain hair colour: all are preserved on the
  wire and surfaced as an error or a warning. The backend is the validator of
  record.
- **Test file placement:** `__tests__/` subdirectory beside the source file, the
  convention used by both sibling directories.
- **Component tests that touch `useSearchParams` need a `MemoryRouter` wrapper**,
  and must `vi.mock("@/components/DetailActionBarContext")` and
  `vi.mock("sonner")` — the shape at
  `src/components/features/characters/presets/__tests__/CharacterPresetsEditor.test.tsx:1-38`.
- **Reference constants, confirmed against the repo, to be used verbatim:**
  - `WarriorImprovedMaxHpIncreaseId = 1000001`
    (`libs/atlas-constants/skill/constants.go:2933`)
  - `MagicianImprovedMaxMpIncreaseId = 2000001`
    (`libs/atlas-constants/skill/constants.go:3023`)
  - Hair composition: `hairColor + 10 * Math.floor(hairStyle / 10)`
    (task-246 `design.md` §A3)
  - `spPool[0] < 6` is unsatisfiable for any non-zero investment: the server
    computes `needed = in.SP + 5` when the skill has a coded prerequisite and
    rejects `needed > pool[0]`
    (`services/atlas-character-factory/atlas.com/character-factory/factory/processor.go:428-437`).
- **Shipped seed row, used as the canonical test fixture** (from
  `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`,
  `attributes.mapleLife`; 2 look rows, 10 class rows):

```json
{
  "looks": [
    { "gender": 0, "faces": [20000, 20001, 20002], "hairs": [30030, 30020, 30000],
      "hairColors": [0, 7, 3, 2], "skinColors": [0, 1, 2, 3] }
  ],
  "classes": [
    { "ordinal": 0, "gender": 0, "jobId": 100, "level": 30, "mapId": 102000000,
      "stats": { "str": 35, "dex": 4, "int": 4, "luk": 4, "hp": 804, "mp": 150 },
      "ap": 123, "sp": "61,0,0,0,0,0,0,0,0,0", "spSkillId": 1000001,
      "meso": 100000,
      "equipment": [ { "templateId": 1040021, "useAverageStats": true } ],
      "inventory": [ { "templateId": 2000002, "quantity": 100 } ] }
  ]
}
```

---

## File Structure

**Modified (9):**

| Path | Responsibility |
|---|---|
| `src/types/models/template.ts` | Maple Life domain types; entry-type rename; `TemplateAttributes.mapleLife` |
| `src/services/api/tenants.service.ts` | `TenantConfigAttributes.mapleLife` |
| `src/components/DetailActionBarContext.tsx` | `DetailActionBarConfig.blockingIssues` |
| `src/components/features/characters/templates/SaveBar.tsx` | render `blockingIssues` |
| `src/components/features/characters/templates/AppearancePoolSection.tsx` | generalise off `CharacterTemplate` |
| `src/components/features/characters/templates/CharacterTemplatesEditor.tsx` | first caller of the generalised section |
| `src/App.tsx` | two lazy routes |
| `src/lib/breadcrumbs/routes.ts` | two patterns + two constants |
| `src/components/features/{tenants/TenantDetailLayout,templates/TemplateDetailLayout}.tsx` | rail entries |

**New (15):** `src/lib/schemas/maple-life.schema.ts`,
`src/pages/{Tenants,Templates}MapleLifePage.tsx`, and under
`src/components/features/characters/maple-life/`: `MapleLifeEditor.tsx`,
`mapleLifeEditorState.ts`, `mapleLifeWarnings.ts`, `mapleLifeLoadout.ts`,
`ClassSelector.tsx`, `IdentitySection.tsx`, `AppearancePoolsSection.tsx`,
`ProgressionSection.tsx`, `SpSkillSection.tsx`, `StartingKitSection.tsx`,
`MapleLifePreviewCard.tsx`, `SeedFromTemplateDialog.tsx`, `EmptyState.tsx`.

---

## Task 1: Domain types, entry-type rename, and round-trip preservation tests

Establishes the type surface every later task consumes, and settles PRD open
question 2 (does `mapleLife` survive the existing PATCH?) with a test rather
than an assumption.

### Files

- `services/atlas-ui/src/types/models/template.ts` — rename the two preset entry
  types in place; add the Maple Life domain types; add
  `TemplateAttributes.mapleLife`
- `services/atlas-ui/src/components/features/characters/presets/EquipmentSection.tsx` — update the renamed import (lines 2, 16)
- `services/atlas-ui/src/components/features/characters/presets/InventorySection.tsx` — update the renamed import (lines 2, 8)
- `services/atlas-ui/src/services/api/tenants.service.ts` — add
  `mapleLife?: MapleLifeConfig` to `TenantConfigAttributes` via a single
  `import type`
- `services/atlas-ui/src/services/api/__tests__/tenants.service.test.ts` — **new file**
- `services/atlas-ui/src/services/api/__tests__/templates-update.test.ts` — add the FR-1.3 preservation case

Patterns to copy: `services/atlas-ui/src/services/api/__tests__/templates-update.test.ts:1-47`
(the `vi.mock("@/lib/api/client")` + captured-`patch` shape).

**Do NOT touch** `tenants.service.ts:34-39,65-66` — that file declares its own
private `CharacterPresetEquipmentEntry` / `CharacterPresetInventoryEntry`
interfaces (4 references: 2 declarations, 2 usages). That duplication is
pre-existing and out of scope per design §3.3; `mapleLife` is added as a single
imported type, not a fifth inline copy.

### Interfaces

**Produces** (every later task consumes these from
`@/types/models/template`):

```ts
export interface EquipmentEntry {
  templateId: number;
  useAverageStats: boolean;
}

export interface InventoryEntry {
  templateId: number;
  quantity: number;
}

export interface MapleLifeLookOptions {
  gender: number;
  faces: number[];
  hairs: number[];
  hairColors: number[];
  skinColors: number[];
}

export interface MapleLifeStatBlock {
  str: number;
  dex: number;
  int: number;
  luk: number;
  hp: number;
  mp: number;
}

export interface MapleLifeClassEntry {
  ordinal: number;
  gender: number;
  jobId: number;
  level: number;
  mapId: number;
  stats: MapleLifeStatBlock;
  ap: number;
  /** Ten-book pool string, e.g. "61,0,0,0,0,0,0,0,0,0". */
  sp: string;
  /** Go `json:"spSkillId,omitempty"` — an absent key means "no SP step". */
  spSkillId?: number;
  meso: number;
  equipment: EquipmentEntry[];
  inventory: InventoryEntry[];
}

export interface MapleLifeConfig {
  looks: MapleLifeLookOptions[];
  classes: MapleLifeClassEntry[];
}
```

`TemplateAttributes` gains `mapleLife?: MapleLifeConfig;` (place it after
`cashShop?`, mirroring the optional-block precedent at `template.ts:112-123`).
`TenantConfigAttributes` gains the identical optional property.

`EquipmentEntry` / `InventoryEntry` are `CharacterPresetEquipmentEntry` /
`CharacterPresetInventoryEntry` **renamed in place**. Delete the old names — do
not re-export aliases (CLAUDE.md: "prefer straightforward moves over
re-exported type aliases"). `CharacterPresetAttributes.equipment` /
`.inventory` (`template.ts:63,64`) now reference the new names. Verified: no
other file in `src/` and no test references either old name.

- [ ] **Step 1: Write the failing preservation tests**

New file `src/services/api/__tests__/tenants.service.test.ts`. Mock setup copied
verbatim from `src/services/api/__tests__/templates-update.test.ts:1-18`
(`patch`/`put`/`getOne` vi.fn()s inside a `vi.mock("@/lib/api/client")`), then:

`describe("tenantsService.updateTenantConfiguration")`:

| subtest | setup | assertion |
|---|---|---|
| `"preserves an undeclared mapleLife block across an unrelated save"` | tenant config whose `attributes` carries `mapleLife: SEED_ML` (the Global Constraints fixture) and `cashShop: { commodities: {} }`; call with `updates = { characters: { templates: [], presets: [] } }` | `patch.mock.calls[0][1].data.attributes.mapleLife` `toEqual(SEED_ML)` |
| `"a mapleLife-only save leaves every other section untouched"` | same seeded config; call with `updates = { mapleLife: SEED_ML }` | `patch.mock.calls[0][1].data.attributes` has `characters`, `npcs`, `worlds`, `socket`, `cashShop` each `toEqual` the seeded values |
| `"PATCHes the configuration path with the tenant id"` | same | `patch.mock.calls[0][0]` `toBe("/api/configurations/tenants/t1")` |

Add to the existing `src/services/api/__tests__/templates-update.test.ts`, inside
`describe("templatesService.update")`:

| subtest | setup | assertion |
|---|---|---|
| `"preserves mapleLife across a characters-only update"` | `fullAttributes()` extended with `mapleLife: SEED_ML`, then `update("t1", { ...attrs, characters: { templates: [], presets: [] } })` | `patch.mock.calls[0][1].data.attributes.mapleLife` `toEqual(SEED_ML)` |

Define `SEED_ML` once per test file as a `MapleLifeConfig` literal built from the
Global Constraints fixture above (both look rows are not needed; one look row
and one class row is sufficient and keeps the literal readable).

- [ ] **Step 2: Run the tests and verify they fail**

```
cd services/atlas-ui && npx vitest run src/services/api/__tests__/tenants.service.test.ts src/services/api/__tests__/templates-update.test.ts
```

Expected: FAIL — `mapleLife` is not assignable to `TenantConfigAttributes` /
`TemplateAttributes` (TypeScript), and `EquipmentEntry` is not exported.

- [ ] **Step 3: Add the types and perform the rename**

In `src/types/models/template.ts`: rename
`CharacterPresetEquipmentEntry` → `EquipmentEntry` and
`CharacterPresetInventoryEntry` → `InventoryEntry` at their declarations
(`:32`, `:37`) and their two usages (`:63`, `:64`); add the six Maple Life
interfaces from the **Produces** block above; add `mapleLife?: MapleLifeConfig;`
to `TemplateAttributes`.

In the two presets sections, update the import specifier and the props type:

```ts
// EquipmentSection.tsx:2
import type { EquipmentEntry } from "@/types/models/template";
// EquipmentSection.tsx:16
  equipment: EquipmentEntry[];
```

```ts
// InventorySection.tsx:2
import type { InventoryEntry } from "@/types/models/template";
// InventorySection.tsx:8
  inventory: InventoryEntry[];
```

In `src/services/api/tenants.service.ts`, add to the existing import block at the
top of the file:

```ts
import type { MapleLifeConfig } from "@/types/models/template";
```

and add to `TenantConfigAttributes` (after the `cashShop?` block that ends at
`:131`):

```ts
  mapleLife?: MapleLifeConfig;
```

- [ ] **Step 4: Run the tests and verify they pass**

```
cd services/atlas-ui && npx vitest run src/services/api/__tests__/tenants.service.test.ts src/services/api/__tests__/templates-update.test.ts
cd services/atlas-ui && npm run build
```

Expected: both test files PASS, `tsc -b` clean. If a preservation subtest FAILS,
the merge is losing the block — fix it in the service (in scope per FR-1.3) and
write a one-paragraph data-repair note to
`docs/tasks/task-271-maple-life-config-ui/progress.md` naming which write path
was lossy.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/types/models/template.ts \
        services/atlas-ui/src/components/features/characters/presets/EquipmentSection.tsx \
        services/atlas-ui/src/components/features/characters/presets/InventorySection.tsx \
        services/atlas-ui/src/services/api/tenants.service.ts \
        services/atlas-ui/src/services/api/__tests__/tenants.service.test.ts \
        services/atlas-ui/src/services/api/__tests__/templates-update.test.ts
git commit -m "feat(atlas-ui): declare mapleLife on both configuration types and prove PATCH round-trip"
```

---

## Task 2: `DetailActionBarConfig.blockingIssues`

The save bar today has no way to express "dirty but not saveable". FR-11.12
requires Save disabled with the blocking count reported. Both existing callers
omit the new field and must remain behaviourally identical.

### Files

- `services/atlas-ui/src/components/DetailActionBarContext.tsx` — add the field, thread it through `register()` and `<SaveBar>`, add it to the effect dependency list
- `services/atlas-ui/src/components/features/characters/templates/SaveBar.tsx` — disable Save and change the status text when `> 0`
- `services/atlas-ui/src/components/__tests__/DetailActionBarContext.test.tsx` — new cases
- `services/atlas-ui/src/components/features/characters/templates/__tests__/SaveBar.test.tsx` — **new file**

Module root: `services/atlas-ui`.

### Interfaces

**Consumes:** nothing from Task 1.

**Produces:**

```ts
export interface DetailActionBarConfig {
  dirty: boolean;
  isSaving: boolean;
  onSave: () => void;
  onDiscard: () => void;
  /** Count of blocking validation errors. > 0 disables Save and is reported
   *  in the bar. Omitted means "no validation gate". */
  blockingIssues?: number;
}
```

- [ ] **Step 1: Write the failing tests**

New file `src/components/features/characters/templates/__tests__/SaveBar.test.tsx`.
`describe("SaveBar")`, rendering `<SaveBar {...props} />` directly (no router,
no context — SaveBar is a leaf; imports are `render`, `screen` from
`@testing-library/react`, `userEvent`, and `describe/it/expect/vi` from
`vitest`).

| case | props | expected |
|---|---|---|
| no gate, dirty | `{ dirty: true, isSaving: false }` | Save button enabled; text `"Unsaved changes"` |
| gate satisfied | `{ dirty: true, isSaving: false, blockingIssues: 0 }` | Save button enabled; text `"Unsaved changes"` |
| one blocking error | `{ dirty: true, isSaving: false, blockingIssues: 1 }` | Save button `toBeDisabled()`; text `"1 blocking error"`; `"Unsaved changes"` NOT in the document |
| three blocking errors | `{ dirty: true, isSaving: false, blockingIssues: 3 }` | text `"3 blocking errors"` |
| blocking but Discard usable | `{ dirty: true, isSaving: false, blockingIssues: 2 }` | Discard button NOT disabled; clicking it then confirming `"Discard changes"` calls `onDiscard` once |
| clean | `{ dirty: false, isSaving: false, blockingIssues: 0 }` | Save disabled; text `"No unsaved changes"` |

Add to `src/components/__tests__/DetailActionBarContext.test.tsx`:

| case | expected |
|---|---|
| `"forwards blockingIssues to the bar"` | register a config with `blockingIssues: 2`; the rendered bar shows `"2 blocking errors"` and Save is disabled |
| `"re-pushes when only blockingIssues changes"` | register with `blockingIssues: 0`, rerender with `blockingIssues: 1`; the bar's Save transitions from enabled to disabled |

The second case is the one that fails without the dependency-list change: the
push effect at `DetailActionBarContext.tsx:82` currently depends on
`[register, present, dirty, isSaving]`, so a `blockingIssues`-only change would
never re-push.

- [ ] **Step 2: Run the tests and verify they fail**

```
cd services/atlas-ui && npx vitest run src/components/features/characters/templates/__tests__/SaveBar.test.tsx src/components/__tests__/DetailActionBarContext.test.tsx
```

Expected: FAIL — `blockingIssues` is not a known prop.

- [ ] **Step 3: Implement**

`DetailActionBarContext.tsx`:

```ts
export interface DetailActionBarConfig {
  dirty: boolean;
  isSaving: boolean;
  onSave: () => void;
  onDiscard: () => void;
  /** Count of blocking validation errors. > 0 disables Save and is reported
   *  in the bar. Omitted means "no validation gate". */
  blockingIssues?: number;
}
```

In `useRegisterDetailActionBar`, add the primitive beside `dirty`/`isSaving`
(after the existing `const isSaving = ...` at `:67`):

```ts
  const blockingIssues = config?.blockingIssues;
```

forward it in the `register({...})` call (currently `:75-80`):

```ts
    register({
      dirty,
      isSaving,
      blockingIssues,
      onSave: () => latest.current?.onSave(),
      onDiscard: () => latest.current?.onDiscard(),
    });
```

and extend the dependency list at `:82`:

```ts
  }, [register, present, dirty, isSaving, blockingIssues]);
```

In `DetailActionBar` (`:93-100`), pass it through:

```tsx
    <SaveBar
      dirty={config.dirty}
      isSaving={config.isSaving}
      blockingIssues={config.blockingIssues}
      onSave={config.onSave}
      onDiscard={config.onDiscard}
    />
```

`SaveBar.tsx` — extend the props (`:14-19`) with
`blockingIssues?: number;`, then inside the component:

```tsx
  const blocked = (blockingIssues ?? 0) > 0;
  const blockingText = `${blockingIssues} blocking ${
    blockingIssues === 1 ? "error" : "errors"
  }`;
```

Replace the status paragraph (`:26-32`) so a blocked bar reports the count
instead of "Unsaved changes", and disable Save (`:41`) with
`disabled={!dirty || isSaving || blocked}`. **Leave Discard's
`disabled={!dirty || isSaving}` unchanged** — an operator must be able to back
out of an invalid edit.

```tsx
      <p
        className={
          blocked
            ? "text-sm font-medium text-destructive"
            : dirty
              ? "text-sm font-medium"
              : "text-sm text-muted-foreground"
        }
      >
        {blocked
          ? blockingText
          : dirty
            ? "Unsaved changes"
            : "No unsaved changes"}
      </p>
```

- [ ] **Step 4: Run the tests and verify they pass**

```
cd services/atlas-ui && npx vitest run src/components/features/characters/templates/__tests__/SaveBar.test.tsx src/components/__tests__/DetailActionBarContext.test.tsx src/components/features/characters/templates/__tests__/CharacterTemplatesEditor.test.tsx src/components/features/characters/presets/__tests__/CharacterPresetsEditor.test.tsx
```

Expected: all PASS. The two sibling editors omit `blockingIssues`, so
`blocked` is `false` and their behaviour is unchanged.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/DetailActionBarContext.tsx \
        services/atlas-ui/src/components/features/characters/templates/SaveBar.tsx \
        services/atlas-ui/src/components/__tests__/DetailActionBarContext.test.tsx \
        services/atlas-ui/src/components/features/characters/templates/__tests__/SaveBar.test.tsx
git commit -m "feat(atlas-ui): let a detail page gate Save on blocking validation errors"
```

---

## Task 3: Generalise `AppearancePoolSection` off `CharacterTemplate`

FR-6.3: reuse, not duplication. Today the component derives
`pool = template[dimension]`, looks up `PICK_KEY_BY_POOL[dimension]`, and calls
`buildVariantLoadout(template, picks, dimension, id)` itself — three couplings
to `CharacterTemplate`. Invert all three onto the caller. The templates editor
becomes the first caller with **identical rendered output**.

### Files

- `services/atlas-ui/src/components/features/characters/templates/AppearancePoolSection.tsx` — new props
- `services/atlas-ui/src/components/features/characters/templates/CharacterTemplatesEditor.tsx` — update the call site at `:219-253`
- `services/atlas-ui/src/components/features/characters/templates/__tests__/AppearancePoolSection.test.tsx` — update to the new props
- `services/atlas-ui/src/components/features/characters/templates/__tests__/CharacterTemplatesEditor.test.tsx` — read-only unless it breaks; it renders the editor, not the section directly

Module root: `services/atlas-ui`.

### Interfaces

**Produces:**

```ts
interface AppearancePoolSectionProps {
  dimension: AppearancePoolKey;
  title: string;
  pool: number[];
  selectedIndex: number;
  variantLoadout: (dimension: AppearancePoolKey, id: number) => CharacterLoadout;
  onPick: (index: number) => void;
  onRemoveEntry: (entryIndex: number) => void;
  /** Editor supplies the AppearanceBrowserDialog here (open state owned locally). */
  renderAddDialog: (
    open: boolean,
    onOpenChange: (open: boolean) => void,
  ) => ReactNode;
  /** Extra copy under the header — FR-6.5 value domain, FR-6.6 allow-list note. */
  description?: ReactNode;
}
```

`CharacterLoadout` is imported from `@/services/api/characterRender.service`
(`characterRender.service.ts:7-13`).

**Note — the design is wrong on one point.** Design §7.1 says the existing
`AppearancePoolSection.test.tsx` assertions "do not change". They must: the test
at `__tests__/AppearancePoolSection.test.tsx:47` asserts
`expect(onPick).toHaveBeenCalledWith("faceIdx", 1)` — two arguments — and the new
`onPick` is single-argument. That assertion becomes
`toHaveBeenCalledWith(1)`. The *behaviour* the tests assert is unchanged; one
assertion's argument list is not.

- [ ] **Step 1: Rewrite the existing test to the new props**

`src/components/features/characters/templates/__tests__/AppearancePoolSection.test.tsx`.
Keep the `vi.mock("@/context/tenant-context")` block at `:5-12` verbatim. Replace
the `renderSection` helper so it supplies the new props, and add one case for
`description`:

```tsx
const POOL = [20000, 21000];

function renderSection(over: Record<string, unknown> = {}) {
  return render(
    <AppearancePoolSection
      dimension="faces"
      title="Faces"
      pool={POOL}
      selectedIndex={0}
      variantLoadout={(_dim, id) => ({
        skin: 0,
        hair: 30030,
        face: id,
        equipment: {},
        gender: 0,
      })}
      onPick={vi.fn()}
      onRemoveEntry={vi.fn()}
      renderAddDialog={() => null}
      {...over}
    />,
  );
}
```

The five existing cases carry over with these adjustments:

| existing case | change |
|---|---|
| `"renders one thumb per pool entry with id captions"` | none |
| `"clicking a thumb sets the preview pick (UI-only)"` | assertion becomes `expect(onPick).toHaveBeenCalledWith(1)` |
| `"the picked thumb is marked pressed"` | override becomes `{ selectedIndex: 1 }` instead of `{ picks: {...} }` |
| `"each thumb has a remove affordance"` | none |
| `"empty pool shows the non-blocking factory warning"` | override becomes `{ pool: [] }` instead of `{ template: normalizeTemplate({}) }` |

Add one new case:

| new case | setup | expected |
|---|---|---|
| `"renders the description when supplied"` | `{ description: <span>Faces are full item ids</span>` } | `screen.getByText("Faces are full item ids")` is in the document |

Delete the now-unused `normalizeTemplate` / `DEFAULT_PICKS` imports at `:14` and
the `const tpl = ...` at `:17`.

- [ ] **Step 2: Run the test and verify it fails**

```
cd services/atlas-ui && npx vitest run src/components/features/characters/templates/__tests__/AppearancePoolSection.test.tsx
```

Expected: FAIL — `pool`, `selectedIndex`, `variantLoadout`, `description` are not
known props, and `onPick` is still called with two arguments.

- [ ] **Step 3: Generalise the component**

`AppearancePoolSection.tsx` — replace the props interface with the **Produces**
block above; delete the `CharacterTemplate`, `PICK_KEY_BY_POOL`, `PreviewPicks`,
and `buildVariantLoadout` imports (`:3`, `:7-12`); add
`import type { CharacterLoadout } from "@/services/api/characterRender.service";`
and keep `import type { AppearancePoolKey } from "./editorState";`. Delete the
`const pickKey = ...` and `const pool = template[dimension];` lines (`:48-49`).
Inside the thumb map, `buildVariantLoadout(template, picks, dimension, id)`
(`:75`) becomes `variantLoadout(dimension, id)`;
`selected={picks[pickKey] === idx}` (`:80`) becomes
`selected={selectedIndex === idx}`; `onSelect={() => onPick(pickKey, idx)}`
(`:81`) becomes `onSelect={() => onPick(idx)}`.

Render `description` under the header row, after the closing `</div>` of the
header at `:64`:

```tsx
      {description && (
        <p className="text-xs text-muted-foreground">{description}</p>
      )}
```

Leave the empty-pool warning at `:58-63` exactly as it is — "character creation
will fail while this pool is empty" is true for both callers.

- [ ] **Step 4: Update the templates editor call site**

`CharacterTemplatesEditor.tsx:219-253`. Replace the four props the component no
longer takes; everything else (`key`, `dimension`, `title`, `onRemoveEntry`,
`renderAddDialog`) is unchanged:

```tsx
                    {APPEARANCE_SECTIONS.map(({ dimension, title }) => (
                      <AppearancePoolSection
                        key={dimension}
                        dimension={dimension}
                        title={title}
                        pool={template[dimension]}
                        selectedIndex={picks[PICK_KEY_BY_POOL[dimension]!]}
                        variantLoadout={(dim, id) =>
                          buildVariantLoadout(template, picks, dim, id)
                        }
                        onPick={(idx) =>
                          dispatch({
                            type: "setPreviewPick",
                            pick: PICK_KEY_BY_POOL[dimension]!,
                            value: idx,
                          })
                        }
                        onRemoveEntry={(entryIndex) =>
                          dispatch({
                            type: "removePoolEntry",
                            pool: dimension,
                            entryIndex,
                          })
                        }
                        renderAddDialog={(open, onOpenChange) => (
                          <AppearanceBrowserDialog
                            dimension={dimension}
                            gender={template.gender}
                            variantLoadout={(dim, id) =>
                              buildVariantLoadout(template, picks, dim, id)
                            }
                            open={open}
                            onOpenChange={onOpenChange}
                            onSelect={(id) =>
                              dispatch({ type: "addPoolEntry", pool: dimension, id })
                            }
                            selectMode="add"
                            markedIds={template[dimension]}
                          />
                        )}
                      />
                    ))}
```

Add `PICK_KEY_BY_POOL` to the existing `./editorState` import at `:5-12` and
drop the now-unused `type PreviewPicks` from it if TypeScript reports it unused.

- [ ] **Step 5: Run the tests and verify they pass**

```
cd services/atlas-ui && npx vitest run src/components/features/characters/templates src/components/features/characters/presets
cd services/atlas-ui && npm run build && npm run lint
```

Expected: all PASS. The templates page renders identically — this is the
FR-6.3 "unchanged in behaviour" guarantee.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/templates/AppearancePoolSection.tsx \
        services/atlas-ui/src/components/features/characters/templates/CharacterTemplatesEditor.tsx \
        services/atlas-ui/src/components/features/characters/templates/__tests__/AppearancePoolSection.test.tsx
git commit -m "refactor(atlas-ui): decouple AppearancePoolSection from CharacterTemplate"
```

---

## Task 4: `mapleLifeEditorState.ts` — draft shape, reducer, projection

The pure core of the editor. No React. Design §5.2/§5.3.

### Files

- `services/atlas-ui/src/components/features/characters/maple-life/mapleLifeEditorState.ts` — **new file**
- `services/atlas-ui/src/components/features/characters/maple-life/__tests__/mapleLifeEditorState.test.ts` — **new file**

Patterns to copy:
`services/atlas-ui/src/components/features/characters/presets/presetEditorState.ts:126-151`
(the `initialX` / `project` / `projectForSave` / `isDirty`-by-serialisation
trio) and
`services/atlas-ui/src/components/features/characters/presets/__tests__/presetEditorState.test.ts:1-21`
(pure-module test setup: `import { describe, it, expect } from "vitest"`, no
render, no wrapper).

Module root: `services/atlas-ui`.

### Interfaces

**Consumes** from Task 1: `MapleLifeConfig`, `MapleLifeClassEntry`,
`MapleLifeLookOptions`, `MapleLifeStatBlock`, `EquipmentEntry`,
`InventoryEntry` — all from `@/types/models/template`.

**Produces:**

```ts
export const ORDINAL_COUNT = 5;
export const GENDER_COUNT = 2;

export type StatKey = "str" | "dex" | "int" | "luk" | "hp" | "mp";
export type ScalarKey = "ap" | "meso" | "level";
export type IdentityKey = "jobId" | "mapId";
export type LookDimension = "faces" | "hairs" | "hairColors" | "skinColors";

export interface PreviewPicks {
  faceIdx: number;
  hairIdx: number;
  hairColorIdx: number;
  skinIdx: number;
}

export interface MapleLifeClassDraft
  extends Omit<MapleLifeClassEntry, "sp"> {
  /** Ten book values parsed from `sp`. Empty when `sp` is unparseable. */
  spBooks: number[];
  /** The `sp` string exactly as loaded, emitted verbatim when spBooks is empty. */
  spRaw: string;
  /** False when the (ordinal, gender) row is absent from the loaded config. */
  present: boolean;
}

export interface MapleLifeLookDraft extends MapleLifeLookOptions {
  present: boolean;
}

export interface MapleLifeEditorState {
  /** Always exactly 10, ordinal-major: index = ordinal * 2 + gender. */
  drafts: MapleLifeClassDraft[];
  /** Always exactly 2, indexed by gender. */
  looks: MapleLifeLookDraft[];
  /** Last loaded/saved value, already run through projectForSave. */
  baseline: MapleLifeConfig;
  ordinal: number;
  gender: number;
  /** Preview picks per gender, key `${gender}` — looks are gender-split. */
  picks: Record<string, PreviewPicks>;
  loaded: boolean;
}

export type MapleLifeAction =
  | { type: "load"; config: MapleLifeConfig | undefined }
  | { type: "select"; ordinal: number; gender: number }
  | { type: "setIdentity"; field: IdentityKey; value: number }
  | { type: "setScalar"; field: ScalarKey; value: number }
  | { type: "setStat"; stat: StatKey; value: number }
  | { type: "setSpBook"; index: number; value: number }
  | { type: "setSpSkillId"; value: number | undefined }
  | { type: "addLookEntry"; dimension: LookDimension; id: number }
  | { type: "removeLookEntry"; dimension: LookDimension; entryIndex: number }
  | { type: "addEquipment"; templateId: number }
  | { type: "removeEquipment"; entryIndex: number }
  | { type: "setEquipmentAvg"; entryIndex: number; value: boolean }
  | { type: "addInventory"; templateId: number }
  | { type: "removeInventory"; entryIndex: number }
  | { type: "setInventoryQty"; entryIndex: number; value: number }
  | { type: "setPreviewPick"; pick: keyof PreviewPicks; value: number }
  | { type: "materialiseAll" }
  | { type: "seedFromTemplate"; config: MapleLifeConfig }
  | { type: "discard" }
  | { type: "savedOk" };

export const DEFAULT_PICKS: PreviewPicks;
export function initialMapleLifeState(): MapleLifeEditorState;
export function mapleLifeReducer(
  state: MapleLifeEditorState,
  action: MapleLifeAction,
): MapleLifeEditorState;
export function draftIndex(ordinal: number, gender: number): number;
export function selectedDraft(state: MapleLifeEditorState): MapleLifeClassDraft;
export function selectedLook(state: MapleLifeEditorState): MapleLifeLookDraft;
export function picksFor(state: MapleLifeEditorState, gender: number): PreviewPicks;
export function parseSpPool(sp: string): number[];
export function projectForSave(state: MapleLifeEditorState): MapleLifeConfig;
export function isDirty(state: MapleLifeEditorState): boolean;
export function isEmptyConfig(config: MapleLifeConfig | undefined): boolean;
```

**Key design decisions the implementer must honour:**

1. **`baseline` is a projection, never the raw loaded config.** `load` sets
   `baseline = projectForSave(<the state it just built>)`, and `savedOk` sets
   `baseline = projectForSave(state)`. Both sides of the `isDirty`
   serialisation therefore pass through the same key order, so an untouched
   load can never read as dirty because the server serialised `meso` before
   `sp`.
2. **`projectForSave` writes fields in the Go JSON-tag order**
   (`tenants/maplelife/rest.go:47-70`): `ordinal, gender, jobId, level, mapId,
   stats, ap, sp, [spSkillId], meso, equipment, inventory`. Write it out
   explicitly; do not rest-spread.
3. **`spSkillId` is omitted, not zeroed,** when the draft's value is
   `undefined` — Go's `omitempty` means a class with no SP step must round-trip
   as an absent key.
4. **`sp` serialisation:** `spBooks.length === 10 ? spBooks.join(",") : spRaw`.
   An unparseable pool is emitted verbatim.
5. **One materialisation site.** Every mutating action goes through an
   `updateSelected(state, fn)` helper that clones the selected draft, applies
   `fn`, and sets `present: true`. `addLookEntry` does the same for the
   selected gender's look draft.
6. **`seedFromTemplate` replaces `drafts` and `looks` wholesale** from a
   `structuredClone` of the donor and leaves `baseline` alone, so the result
   reads dirty (FR-12.4).

Neutral draft for an absent `(ordinal, gender)` row:

```ts
function neutralDraft(ordinal: number, gender: number): MapleLifeClassDraft {
  return {
    ordinal,
    gender,
    jobId: 0,
    level: 1,
    mapId: 0,
    stats: { str: 0, dex: 0, int: 0, luk: 0, hp: 0, mp: 0 },
    ap: 0,
    spBooks: [0, 0, 0, 0, 0, 0, 0, 0, 0, 0],
    spRaw: "0,0,0,0,0,0,0,0,0,0",
    spSkillId: undefined,
    meso: 0,
    equipment: [],
    inventory: [],
    present: false,
  };
}
```

`parseSpPool` returns `[]` unless the string splits on `,` into exactly ten
parts, each of which `Number.parseInt(part, 10)` yields a finite integer for.

`isEmptyConfig(c)` is `c === undefined || c.classes.length === 0` — the same test
`resolveMapleLifePreset` applies before returning `ErrMapleLifeNotConfigured`
(`processor.go:390-392`).

- [ ] **Step 1: Write the failing tests**

New file `src/components/features/characters/maple-life/__tests__/mapleLifeEditorState.test.ts`.
Pure module — imports are `{ describe, it, expect } from "vitest"` plus the
module under test and `MapleLifeConfig` from `@/types/models/template`. Define
a `SEED` fixture at the top of the file: the Global Constraints JSON above,
extended with a **second** class row `{ ordinal: 3, gender: 1, jobId: 400,
level: 30, mapId: 103000000, stats: { str: 4, dex: 25, int: 4, luk: 20, hp: 520,
mp: 130 }, ap: 121, sp: "61,0,0,0,0,0,0,0,0,0", meso: 100000, equipment: [],
inventory: [] }` (no `spSkillId` key — this is the omitempty case) and a second
look row `{ gender: 1, faces: [21000], hairs: [31050], hairColors: [0],
skinColors: [0] }`.

`describe("mapleLifeReducer")`:

| subtest | action(s) | assertion |
|---|---|---|
| `"load expands a sparse config into ten ordinal-major slots"` | `load(SEED)` | `state.drafts` has length 10; `drafts[0].ordinal === 0 && drafts[0].gender === 0`; `drafts[9].ordinal === 4 && drafts[9].gender === 1` |
| `"load marks presence only for rows the config carries"` | `load(SEED)` | `drafts.filter(d => d.present).map(d => [d.ordinal, d.gender])` `toEqual([[0,0],[3,1]])` |
| `"load parses sp into ten books"` | `load(SEED)` | `drafts[0].spBooks` `toEqual([61,0,0,0,0,0,0,0,0,0])`; `drafts[0].spRaw` `toBe("61,0,0,0,0,0,0,0,0,0")` |
| `"load leaves spBooks empty for an unparseable sp"` | `load` a config whose class 0 has `sp: "61,0"` | `drafts[0].spBooks` `toEqual([])`; `drafts[0].spRaw` `toBe("61,0")` |
| `"load with undefined yields ten absent slots and two absent looks"` | `load(undefined)` | every draft `present === false`; both look drafts `present === false`; `state.loaded === true` |
| `"load marks look presence per gender"` | `load(SEED)` | `looks[0].present === true`, `looks[1].present === true`; then load a config with only the gender-0 look row → `looks[1].present === false` |

`describe("projectForSave")`:

| subtest | setup | assertion |
|---|---|---|
| `"emits only present rows, ordinal-major"` | `load(SEED)` | `projectForSave(s).classes.map(c => [c.ordinal, c.gender])` `toEqual([[0,0],[3,1]])` |
| `"an untouched load round-trips to the loaded value"` | `load(SEED)` | `projectForSave(s)` `toEqual(SEED)` (deep equality — key ORDER is normalised by the projection and is deliberately not asserted) |
| `"omits spSkillId entirely when the draft has none"` | `load(SEED)` | `Object.hasOwn(projectForSave(s).classes[1], "spSkillId")` `toBe(false)` |
| `"emits an unparseable sp verbatim"` | `load` the `sp: "61,0"` config | `projectForSave(s).classes[0].sp` `toBe("61,0")` |
| `"re-serialises sp from the books after an edit"` | `load(SEED)`, `select(0,0)`, `setSpBook(0, 75)` | `projectForSave(s).classes[0].sp` `toBe("75,0,0,0,0,0,0,0,0,0")` |
| `"emits only present look rows"` | `load` a gender-0-only-looks config | `projectForSave(s).looks` has length 1 with `gender === 0` |

`describe("materialisation")`:

| subtest | setup | assertion |
|---|---|---|
| `"editing an absent row materialises it"` | `load(SEED)`, `select(2, 0)`, `setIdentity("jobId", 300)` | that draft's `present === true`; `projectForSave(s).classes` has length 3 and includes `{ ordinal: 2, gender: 0 }` with `jobId === 300` |
| `"adding a look entry materialises that gender's look row"` | `load` a gender-0-only-looks config, `select(0, 1)`, `addLookEntry("faces", 21000)` | `looks[1].present === true`; `projectForSave(s).looks` has length 2 |
| `"materialiseAll marks all ten rows and both looks present"` | `load(undefined)`, `materialiseAll` | `projectForSave(s).classes` has length 10; `projectForSave(s).looks` has length 2 |
| `"selecting a row does NOT materialise it"` | `load(SEED)`, `select(2, 0)` | `projectForSave(s).classes` still has length 2 |

`describe("dirty / discard / savedOk")`:

| subtest | setup | assertion |
|---|---|---|
| `"a fresh load is not dirty"` | `load(SEED)` | `isDirty(s)` `toBe(false)` |
| `"any field edit is dirty"` | `load(SEED)`, `select(0,0)`, `setScalar("ap", 124)` | `isDirty(s)` `toBe(true)` |
| `"a selection change alone is not dirty"` | `load(SEED)`, `select(4,1)`, `setPreviewPick("faceIdx", 2)` | `isDirty(s)` `toBe(false)` |
| `"discard restores the baseline"` | `load(SEED)`, edit `ap`, `discard` | `isDirty(s)` `toBe(false)`; `projectForSave(s)` `toEqual(SEED)` |
| `"savedOk rebases so the edit is no longer dirty"` | `load(SEED)`, edit `ap` to 124, `savedOk` | `isDirty(s)` `toBe(false)`; `projectForSave(s).classes[0].ap` `toBe(124)` |
| `"seedFromTemplate replaces the working copy and reads dirty"` | `load(undefined)`, `seedFromTemplate(SEED)` | `isDirty(s)` `toBe(true)`; `projectForSave(s)` `toEqual(SEED)` |
| `"seedFromTemplate deep-clones the donor"` | `load(undefined)`, `seedFromTemplate(SEED)`, then `select(0,0)` and `addEquipment(1302000)` | `SEED.classes[0].equipment` still has its original length |

`describe("helpers")`:

| subtest | assertion |
|---|---|
| `"draftIndex is ordinal-major"` | `draftIndex(0,0) === 0`, `draftIndex(0,1) === 1`, `draftIndex(4,1) === 9` |
| `"parseSpPool accepts exactly ten integers"` | `parseSpPool("61,0,0,0,0,0,0,0,0,0")` `toEqual([61,0,0,0,0,0,0,0,0,0])` |
| `"parseSpPool rejects a nine-book pool"` | `parseSpPool("1,2,3,4,5,6,7,8,9")` `toEqual([])` |
| `"parseSpPool rejects an eleven-book pool"` | `parseSpPool("0,0,0,0,0,0,0,0,0,0,0")` `toEqual([])` |
| `"parseSpPool rejects a non-numeric book"` | `parseSpPool("61,x,0,0,0,0,0,0,0,0")` `toEqual([])` |
| `"isEmptyConfig"` | `isEmptyConfig(undefined) === true`; `isEmptyConfig({ looks: [], classes: [] }) === true`; `isEmptyConfig(SEED) === false` |
| `"picks are gender-split"` | `load(SEED)`, `select(0,0)`, `setPreviewPick("faceIdx", 2)`, `select(0,1)` → `picksFor(s, 1).faceIdx` `toBe(0)`; `picksFor(s, 0).faceIdx` `toBe(2)` |

- [ ] **Step 2: Run the tests and verify they fail**

```
cd services/atlas-ui && npx vitest run src/components/features/characters/maple-life/__tests__/mapleLifeEditorState.test.ts
```

Expected: FAIL — module not found.

- [ ] **Step 3: Implement `mapleLifeEditorState.ts`**

Write the module to the **Produces** contract and the six key decisions above.
`select` sets `ordinal`/`gender` only. `setPreviewPick` writes into
`state.picks[String(state.gender)]`, defaulting from `DEFAULT_PICKS`.
`removeLookEntry` and `removeEquipment`/`removeInventory` splice by index and
mark present. `setSpSkillId(undefined)` deletes the field from the draft (set the
property to `undefined`; `projectForSave` is what omits the key).

- [ ] **Step 4: Run the tests and verify they pass**

```
cd services/atlas-ui && npx vitest run src/components/features/characters/maple-life/__tests__/mapleLifeEditorState.test.ts
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/maple-life/mapleLifeEditorState.ts \
        services/atlas-ui/src/components/features/characters/maple-life/__tests__/mapleLifeEditorState.test.ts
git commit -m "feat(atlas-ui): add the Maple Life editor reducer and wire projection"
```

---

## Task 5: `maple-life.schema.ts` — the hard rules that block Save

FR-11.1 through FR-11.7. A Zod schema over the **projected wire shape**, so what
is validated is exactly what is sent. Every message cites its own rule.

### Files

- `services/atlas-ui/src/lib/schemas/maple-life.schema.ts` — **new file**
- `services/atlas-ui/src/lib/schemas/__tests__/maple-life.schema.test.ts` — **new file**

Patterns to copy: `services/atlas-ui/src/lib/schemas/character-presets.schema.ts:1-49`
(private `z.object` element schemas composed into one exported schema; the call
site uses `safeParse`). Zod is v4 (`services/atlas-ui/package.json:52`).

Module root: `services/atlas-ui`.

### Interfaces

**Consumes** from Task 4: nothing at runtime — the schema validates a
`MapleLifeConfig`, which Task 1 produces.

**Produces:**

```ts
export const mapleLifeSchema: z.ZodType<MapleLifeConfig>;

/** Dotted path -> messages, e.g. "classes.0.spSkillId" -> [...]. */
export type IssueMap = Map<string, string[]>;

/** Runs mapleLifeSchema.safeParse and folds issues into a path-keyed map. */
export function validateMapleLife(config: MapleLifeConfig): IssueMap;
```

`validateMapleLife` joins each issue's `path` with `.` and appends
`issue.message`. `blockingIssues` for the save bar is the total message count:
`[...map.values()].reduce((n, m) => n + m.length, 0)`.

**Rule placement** (design §6.2):

| Rule | Zod placement | Issue path |
|---|---|---|
| FR-11.1 hair style not divisible by 10 | element `.refine` on `looks[].hairs[]` | `looks.<i>.hairs.<j>` |
| FR-11.2 hair colour outside 0..9 | `.min(0).max(9)` on `looks[].hairColors[]` | `looks.<i>.hairColors.<j>` |
| FR-11.5 `sp` not ten integers | element `.refine` on `classes[].sp` | `classes.<i>.sp` |
| FR-11.4 non-zero `spSkillId` on ordinal >= 2 | `.superRefine` on the class element | `classes.<i>.spSkillId` |
| FR-11.6 `spPool[0] < 6` while `spSkillId` is set | `.superRefine` on the class element | `classes.<i>.sp` |
| FR-11.3 empty pool for a gender with class rows | root `.superRefine` | `looks.<i>.<dimension>` |
| FR-11.7 gender with class rows but no `looks` row | root `.superRefine` | `looks` |

FR-11.3 and FR-11.7 are cross-entity and cannot live inside an element schema.

**Exact messages** (each cites its rule; use these strings verbatim so the tests
and the UI agree):

```ts
export const MSG = {
  hairNotNormalised:
    "Hair style ids are normalised to (v/10)*10 (task-246 design.md §A3); this value is not a multiple of 10.",
  hairColorRange:
    "Hair colour is a bare digit 0..9 (task-246 design.md §A3).",
  spNotTenBooks:
    "The SP pool must be exactly ten comma-separated integers, the shape atlas-character persists.",
  spSkillOnHighOrdinal:
    "The client skips the SP step for class ordinal >= 2 (processor.go:424-427), so a player submitting sp != 0 is rejected outright. Clear this value.",
  spPoolTooSmall:
    "Book 0 must be at least 6: the server needs sp + 5 for the prerequisite (processor.go:428-437), so a pool below 6 makes even a level-1 investment unsatisfiable.",
  emptyPool:
    "This pool is empty, so every player submission for this gender is rejected with ErrLookInvalid (processor.go:405-422).",
  missingLookRow:
    "This gender has configured class rows but no looks row, which fails with ErrMapleLifeNotConfigured (processor.go:397-405).",
} as const;
```

- [ ] **Step 1: Write the failing tests**

New file `src/lib/schemas/__tests__/maple-life.schema.test.ts`. Pure module —
`import { describe, it, expect } from "vitest"`. Define a `valid()` helper that
returns a fresh minimal-but-passing `MapleLifeConfig`:

```ts
function valid(): MapleLifeConfig {
  return {
    looks: [
      { gender: 0, faces: [20000], hairs: [30030], hairColors: [0], skinColors: [0] },
    ],
    classes: [
      { ordinal: 0, gender: 0, jobId: 100, level: 30, mapId: 102000000,
        stats: { str: 35, dex: 4, int: 4, luk: 4, hp: 804, mp: 150 },
        ap: 123, sp: "61,0,0,0,0,0,0,0,0,0", spSkillId: 1000001,
        meso: 100000, equipment: [], inventory: [] },
    ],
  };
}
```

Every rule gets both its failing case and the adjacent passing case, and asserts
the issue path:

| rule | failing mutation | expected path | adjacent passing mutation |
|---|---|---|---|
| FR-11.1 | `looks[0].hairs = [30035]` | `"looks.0.hairs.0"` | `looks[0].hairs = [30030]` |
| FR-11.2 | `looks[0].hairColors = [10]` | `"looks.0.hairColors.0"` | `looks[0].hairColors = [9]` |
| FR-11.2 (lower) | `looks[0].hairColors = [-1]` | `"looks.0.hairColors.0"` | `looks[0].hairColors = [0]` |
| FR-11.5 (short) | `classes[0].sp = "61,0"` | `"classes.0.sp"` | `sp = "61,0,0,0,0,0,0,0,0,0"` |
| FR-11.5 (long) | `classes[0].sp = "0,0,0,0,0,0,0,0,0,0,0"` | `"classes.0.sp"` | ditto |
| FR-11.5 (non-numeric) | `classes[0].sp = "61,x,0,0,0,0,0,0,0,0"` | `"classes.0.sp"` | ditto |
| FR-11.4 | `classes[0].ordinal = 2` (keeps `spSkillId: 1000001`); also set `looks[0].gender = 0` unchanged | `"classes.0.spSkillId"` | `ordinal = 2` with `spSkillId` deleted |
| FR-11.6 | `classes[0].sp = "5,0,0,0,0,0,0,0,0,0"` (with `spSkillId` set) | `"classes.0.sp"` | `sp = "6,0,0,0,0,0,0,0,0,0"` |
| FR-11.6 (no skill) | `sp = "5,0,..."` with `spSkillId` deleted | **no** issue at `"classes.0.sp"` | — |
| FR-11.3 | `looks[0].faces = []` | `"looks.0.faces"` | `faces = [20000]` |
| FR-11.3 (all four) | each of `faces`/`hairs`/`hairColors`/`skinColors` emptied in turn | `"looks.0.<dim>"` | — |
| FR-11.3 (no class rows) | `classes = []` and `looks[0].faces = []` | **no** issue — no class row for that gender, so the empty pool is harmless | — |
| FR-11.7 | `classes.push({ ...classes[0], ordinal: 1, gender: 1 })` with no gender-1 look row | `"looks"` | add a gender-1 look row |

Plus:

| subtest | assertion |
|---|---|
| `"the seed fixture validates clean"` | `validateMapleLife(valid()).size` `toBe(0)` |
| `"an empty configuration validates clean"` | `validateMapleLife({ looks: [], classes: [] }).size` `toBe(0)` — an unconfigured tenant is a legitimate state (PRD §6), not an error |
| `"issues accumulate per path"` | `looks[0].hairs = [30035, 30047]` → the map's `"looks.0.hairs.0"` and `"looks.0.hairs.1"` entries each have one message |

Assert failing cases with `expect([...validateMapleLife(cfg).keys()]).toContain(path)`
and passing cases with `expect([...validateMapleLife(cfg).keys()]).not.toContain(path)`.

- [ ] **Step 2: Run the tests and verify they fail**

```
cd services/atlas-ui && npx vitest run src/lib/schemas/__tests__/maple-life.schema.test.ts
```

Expected: FAIL — module not found.

- [ ] **Step 3: Implement the schema**

Compose private element schemas (`lookOptions`, `statBlock`, `equipmentEntry`,
`inventoryEntry`, `classEntry`) into `mapleLifeSchema`, following the
`character-presets.schema.ts` structure. Notes:

- Reuse the ten-book test from Task 4 rather than re-deriving it:
  `import { parseSpPool } from "@/components/features/characters/maple-life/mapleLifeEditorState";`
  and refine with `(sp) => parseSpPool(sp).length === 10`.
- FR-11.6 reads `parseSpPool(sp)[0] < 6` and only fires when `spSkillId` is set
  and truthy AND the pool parsed (a `[]` pool already fired FR-11.5; do not
  double-report).
- The root `superRefine` computes `gendersWithClasses = new Set(classes.map(c => c.gender))`
  and, for each such gender, adds `missingLookRow` at path `["looks"]` when no
  look row has that gender, or `emptyPool` at
  `["looks", <index>, <dimension>]` for each of the four dimensions that is
  empty on the matching look row.
- Element arrays are NOT `.min(1)` — emptiness is only an error when a class row
  for that gender exists, which is what the root refine decides.

- [ ] **Step 4: Run the tests and verify they pass**

```
cd services/atlas-ui && npx vitest run src/lib/schemas/__tests__/maple-life.schema.test.ts
cd services/atlas-ui && npm run build
```

Expected: PASS, `tsc -b` clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/lib/schemas/maple-life.schema.ts \
        services/atlas-ui/src/lib/schemas/__tests__/maple-life.schema.test.ts
git commit -m "feat(atlas-ui): mirror the Maple Life factory validator as a Zod schema"
```

---

## Task 6: soft warnings and hair composition

Two small pure modules. `mapleLifeWarnings.ts` carries the rules that must be
visible without blocking (FR-11.8/9/10); `mapleLifeLoadout.ts` carries the
client's own hair expression (FR-10.2) and the render loadout.

### Files

- `services/atlas-ui/src/components/features/characters/maple-life/mapleLifeWarnings.ts` — **new file**
- `services/atlas-ui/src/components/features/characters/maple-life/mapleLifeLoadout.ts` — **new file**
- `services/atlas-ui/src/components/features/characters/maple-life/__tests__/mapleLifeWarnings.test.ts` — **new file**
- `services/atlas-ui/src/components/features/characters/maple-life/__tests__/mapleLifeLoadout.test.ts` — **new file**

Read-only reference: `services/atlas-ui/src/components/features/characters/templates/previewLoadout.ts:53-73`
(`buildVariantLoadout`'s dimension-substitution shape) — deliberately NOT reused;
see design §8. `CharacterLoadout` comes from
`@/services/api/characterRender.service` (`characterRender.service.ts:7-13`).

Module root: `services/atlas-ui`.

### Interfaces

**Consumes** from Task 4: `MapleLifeEditorState`, `MapleLifeClassDraft`,
`MapleLifeLookDraft`, `PreviewPicks`, `LookDimension`, `draftIndex`.

**Produces:**

```ts
// mapleLifeWarnings.ts
export const KNOWN_SP_SKILL_IDS = [1000001, 2000001] as const;
export const SP_SKILL_LABELS: Record<number, string> = {
  1000001: "Improved Max HP Increase (Warrior)",
  2000001: "Improved Max MP Increase (Magician)",
};

export interface MapleLifeWarning {
  /** Dotted path, same key space as the schema's IssueMap. */
  path: string;
  message: string;
}

export function mapleLifeWarnings(state: MapleLifeEditorState): MapleLifeWarning[];
export function warningMap(state: MapleLifeEditorState): Map<string, string[]>;
```

```ts
// mapleLifeLoadout.ts
/** The client's own expression: anHairEquip[0] = hairColor + 10 * (hairStyle / 10). */
export function composeHair(hairStyle: number, hairColor: number): number;

export function buildMapleLifeLoadout(
  draft: MapleLifeClassDraft,
  look: MapleLifeLookDraft,
  picks: PreviewPicks,
): CharacterLoadout;

export function buildMapleLifeVariantLoadout(
  draft: MapleLifeClassDraft,
  look: MapleLifeLookDraft,
  picks: PreviewPicks,
  dimension: LookDimension,
  candidateId: number,
): CharacterLoadout;

export function combinationCount(look: MapleLifeLookDraft): number;
```

Warning texts (verbatim):

```ts
export const WARN = {
  unconfirmedOrdinal:
    "The ordinal→job order for 2/3/4 is not derived from the client (task-246 design.md §A6). Pin it against a live channel log before trusting it.",
  unknownSpSkill:
    "This skill id has no coded prerequisite in factory/maple_life.go prerequisiteFor, so no prerequisite will be granted for it. The value is preserved as loaded.",
  absentRow:
    "This (ordinal, gender) row is not in the configuration. A player selecting it is rejected with ErrClassOrdinalUnknown. Edit any field to create it.",
} as const;
```

Emission rules:

- FR-11.8 — one `unconfirmedOrdinal` at `classes.<ordinal>.<gender>` for every
  **present** draft whose `ordinal >= 2`.
- FR-11.9 — one `unknownSpSkill` at `classes.<ordinal>.<gender>.spSkillId` for
  every present draft whose `spSkillId` is defined, non-zero, and not in
  `KNOWN_SP_SKILL_IDS`.
- FR-11.10 — one `absentRow` at `classes.<ordinal>.<gender>` for every draft
  with `present === false`.

Note the warning path key space is `<ordinal>.<gender>`, not the schema's
projected array index — warnings are addressed to the fixed grid, which is what
the UI selects by. `MapleLifeEditor` (Task 11) keeps the two maps separate and
never merges their keys.

`buildMapleLifeLoadout` (design §8, FR-10.1): `gender` comes from `look.gender`,
passed explicitly and never inferred; `equipment` maps the draft's
`equipment[].templateId` list onto sequential render slots. Use the four
canonical slots from `previewLoadout.EQUIP_SLOT_BY_POOL` in declaration order
(`"-5"`, `"-6"`, `"-7"`, `"-11"`) for the first four ids and let the render
service place the remainder by omitting them — a Maple Life kit carries seven
equips, more than the four-slot template pool covers, so define:

```ts
const RENDER_SLOTS = ["-5", "-6", "-7", "-11"] as const;
```

and map `draft.equipment.slice(0, RENDER_SLOTS.length)` onto them positionally.

`at(pool, idx)` clamping and the render defaults mirror
`previewLoadout.ts:22-25` and `:8-10` (`RENDER_DEFAULT_SKIN = 0`,
`RENDER_DEFAULT_HAIR = 30030`, `RENDER_DEFAULT_FACE = 20000`) — import those
three constants from `../templates/previewLoadout` rather than redeclaring them.

- [ ] **Step 1: Write the failing tests**

`__tests__/mapleLifeLoadout.test.ts` — `describe("composeHair")`:

| case | inputs | expected | why |
|---|---|---|---|
| already normalised | `(30030, 0)` | `30030` | color 0 adds nothing |
| normalised + colour | `(30030, 7)` | `30037` | seed hairColors carry 7 |
| non-multiple-of-ten base | `(30035, 2)` | `30032` | **normalises**, does not add — this is the case that distinguishes this module from `previewLoadout`'s `baseHair + colorDigit`, which would yield `30037` |
| non-multiple-of-ten, colour 0 | `(30037, 0)` | `30030` | |
| low id | `(0, 3)` | `3` | |

`describe("buildMapleLifeLoadout")`:

| case | assertion |
|---|---|
| `"passes gender explicitly from the look row"` | a gender-1 look row yields `loadout.gender === 1` even though the draft is otherwise identical |
| `"composes hair from the picked style and colour"` | picks `hairIdx: 0`, `hairColorIdx: 1` over `hairs: [30030], hairColors: [0, 7]` → `hair === 30037` |
| `"places the first four equips on the canonical slots"` | draft equipment `[1040021, 1060016, 1072039, 1302008, 1442001]` → `equipment` `toEqual({ "-5": 1040021, "-6": 1060016, "-7": 1072039, "-11": 1302008 })` |
| `"falls back to render defaults for an empty pool"` | look row with all four pools empty → `{ skin: 0, hair: 30030, face: 20000 }` |
| `"clamps an out-of-range pick"` | `faceIdx: 9` over `faces: [20000, 20001]` → `face === 20001` |

`describe("buildMapleLifeVariantLoadout")`:

| case | assertion |
|---|---|
| `"faces substitutes the candidate"` | `dimension: "faces", candidateId: 21000` → `face === 21000` |
| `"hairs composes the candidate with the picked colour"` | picked colour `7`, `candidateId: 30020` → `hair === 30027` |
| `"hairColors composes the picked style with the candidate"` | picked style `30030`, `candidateId: 3` → `hair === 30033` |
| `"skinColors substitutes the candidate"` | `candidateId: 2` → `skin === 2` |

`describe("combinationCount")`:

| case | assertion |
|---|---|
| `"multiplies the four pools"` | the seed gender-0 row (3 faces, 3 hairs, 4 colours, 4 skins) → `144` |
| `"is zero when any pool is empty"` | `faces: []` → `0` |

`__tests__/mapleLifeWarnings.test.ts` — build states with the Task 4 reducer:

| subtest | setup | assertion |
|---|---|---|
| `"warns on every present ordinal >= 2"` (FR-11.8) | load a config with class rows at ordinals 1, 2, and 4 | `warningMap(s)` has keys `"classes.2.0"` and `"classes.4.0"` carrying `WARN.unconfirmedOrdinal`, and NOT `"classes.1.0"` |
| `"does not warn about an unconfirmed ordinal that is absent"` | load a config with only ordinal 0 | no `unconfirmedOrdinal` message anywhere |
| `"warns on an spSkillId outside the two known ids"` (FR-11.9) | class row with `spSkillId: 9999999` | `warningMap(s).get("classes.0.0.spSkillId")` contains `WARN.unknownSpSkill` |
| `"does not warn for either known id"` | `spSkillId: 1000001`, then `2000001` | no `unknownSpSkill` message |
| `"does not warn when spSkillId is absent"` | class row with no `spSkillId` | no `unknownSpSkill` message |
| `"warns for every absent row"` (FR-11.10) | `load(undefined)` | ten `absentRow` messages, one per `"classes.<o>.<g>"` |
| `"stops warning about a row once it is materialised"` | `load(undefined)`, `select(2,0)`, `setIdentity("jobId", 300)` | `warningMap(s).get("classes.2.0")` no longer contains `WARN.absentRow` (it DOES still contain `unconfirmedOrdinal` — ordinal 2 is now present) |

- [ ] **Step 2: Run the tests and verify they fail**

```
cd services/atlas-ui && npx vitest run src/components/features/characters/maple-life/__tests__/mapleLifeLoadout.test.ts src/components/features/characters/maple-life/__tests__/mapleLifeWarnings.test.ts
```

Expected: FAIL — modules not found.

- [ ] **Step 3: Implement both modules**

```ts
export function composeHair(hairStyle: number, hairColor: number): number {
  return hairColor + 10 * Math.floor(hairStyle / 10);
}
```

The rest to the **Produces** contract above.

- [ ] **Step 4: Run the tests and verify they pass**

```
cd services/atlas-ui && npx vitest run src/components/features/characters/maple-life
```

Expected: PASS (Task 4's reducer tests included).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/maple-life/mapleLifeWarnings.ts \
        services/atlas-ui/src/components/features/characters/maple-life/mapleLifeLoadout.ts \
        services/atlas-ui/src/components/features/characters/maple-life/__tests__/mapleLifeWarnings.test.ts \
        services/atlas-ui/src/components/features/characters/maple-life/__tests__/mapleLifeLoadout.test.ts
git commit -m "feat(atlas-ui): add Maple Life soft warnings and client-accurate hair composition"
```

---

## Task 7: `ClassSelector` and `IdentitySection`

The fixed 5x2 selection surface (FR-4.1/4.2/4.4, NFR accessibility) and the
identity fields with their provenance notices (FR-5.1..5.5, FR-11.8 rendering).

### Files

- `services/atlas-ui/src/components/features/characters/maple-life/ClassSelector.tsx` — **new file**
- `services/atlas-ui/src/components/features/characters/maple-life/IdentitySection.tsx` — **new file**
- `services/atlas-ui/src/components/features/characters/maple-life/__tests__/ClassSelector.test.tsx` — **new file**
- `services/atlas-ui/src/components/features/characters/maple-life/__tests__/IdentitySection.test.tsx` — **new file**

Patterns to copy:
- `services/atlas-ui/src/components/features/characters/templates/TemplateSelector.tsx` lines 25-86 — the roving-tabindex `tabRefs` / `handleTabKeyDown` / `role="tablist"` block, copied for BOTH segmented controls.
- `services/atlas-ui/src/components/features/characters/presets/useSyncedNumberInput.ts` lines 13-23 — every number input on the page uses this.

Read-only references: `services/atlas-ui/src/components/features/characters/templates/MapPicker.tsx:12-16`
(`{ value, onChange, debounceMs? }`), `services/atlas-ui/src/lib/hooks/usePresetJobOptions.ts:38-54`
(`{ options, isPending, isError }`).

Module root: `services/atlas-ui`.

### Interfaces

**Consumes** from Task 4: `MapleLifeClassDraft`, `IdentityKey`, `ScalarKey`.
From Task 6: `MapleLifeWarning` paths are not consumed here; the parent passes
resolved message arrays.

**Produces:**

```ts
// ClassSelector.tsx
interface ClassSelectorProps {
  ordinal: number;
  gender: number;
  /** Ten drafts, ordinal-major. Used for the job label and the absent marker. */
  drafts: MapleLifeClassDraft[];
  /** Job display names by id, or null while unknown (isPending/isError). */
  jobNameById: Map<number, string> | null;
  onSelect: (ordinal: number, gender: number) => void;
}
export function ClassSelector(props: ClassSelectorProps): JSX.Element;

// IdentitySection.tsx
interface IdentitySectionProps {
  draft: MapleLifeClassDraft;
  jobs: { options: { id: number; name: string }[]; isPending: boolean; isError: boolean };
  onSetIdentity: (field: IdentityKey, value: number) => void;
  onSetLevel: (value: number) => void;
  /** Blocking messages keyed by field name ("jobId" | "level" | "mapId"). */
  errors?: Record<string, string[]>;
}
export function IdentitySection(props: IdentitySectionProps): JSX.Element;
```

`ClassSelector` renders two `role="tablist"` groups:

- Gender: `aria-label="Gender"`, two tabs labelled `Male` (0) and `Female` (1).
- Ordinal: `aria-label="Class ordinal"`, five tabs labelled
  `` `${ordinal} · ${jobLabel}` `` where `jobLabel` is
  `jobNameById?.get(draft.jobId) ?? String(draft.jobId)` — the raw id is the
  fallback while job names are pending or errored (FR-5.2: never a static job
  table). A draft with `present === false` also renders the text
  `not configured` in the tab.
- Each ordinal tab carries a badge: `derived` for ordinals 0 and 1,
  `unconfirmed` for 2, 3, 4 (FR-4.4).

Both groups use the roving tabindex from `TemplateSelector`: `tabIndex` is `0`
only on the selected tab, `ArrowLeft`/`ArrowRight` wrap modulo the count,
`Home`/`End` jump to the ends, and the handler calls `event.preventDefault()`
then focuses the target ref.

`IdentitySection` renders:

- `jobId` — a `Select` over `jobs.options`. When `jobs.isPending`, render the
  text `Loading job names…` and disable the control; when `jobs.isError`,
  render `Job names unavailable` and still allow a raw numeric entry. Never an
  empty list presented as "no jobs" (FR-5.2).
- `level` — a number `Input` with `min={1} max={200}` through
  `useSyncedNumberInput` (FR-5.5). The bound is an input constraint, not a
  clamp of a loaded value: a loaded `level` outside the range is displayed as
  loaded.
- `mapId` — `<MapPicker value={draft.mapId} onChange={(id) => onSetIdentity("mapId", id)} />` (FR-5.4).
- The provenance notice (FR-5.3), keyed on `draft.ordinal`:
  - ordinals 0, 1: `Derived from the client's own step-skip (task-246 design.md §A6).`
  - ordinals 2, 3, 4: a persistent, non-dismissible block with
    `role="note"` reading
    `The ordinal→job order for 2/3/4 is not derived from the client (task-246 design.md §A6). Pin it against a live channel log before trusting it.`
  The job field stays fully editable in both cases.
- Any `errors[field]` messages under the corresponding control, in
  `text-destructive`.

- [ ] **Step 1: Write the failing tests**

`__tests__/ClassSelector.test.tsx` — imports `render`, `screen` from
`@testing-library/react`, `userEvent`, `describe/it/expect/vi` from `vitest`.
No router, no context mock needed. Build ten drafts with
`mapleLifeReducer(initialMapleLifeState(), { type: "load", config: SEED })`
where `SEED` carries class rows at `(0,0)` and `(2,0)`.

| case | assertion |
|---|---|
| `"renders two tablists"` | `screen.getAllByRole("tablist")` has length 2; their `aria-label`s are `"Gender"` and `"Class ordinal"` |
| `"renders five ordinal tabs and two gender tabs"` | the ordinal tablist has 5 `role="tab"` children; the gender tablist has 2 |
| `"labels an ordinal with its job name"` | `jobNameById = new Map([[100, "Warrior"]])` → a tab with accessible name matching `/0 · Warrior/` |
| `"falls back to the raw jobId when job names are unknown"` | `jobNameById = null` → a tab matching `/0 · 100/` |
| `"badges ordinals 0 and 1 as derived"` | tabs 0 and 1 contain the text `derived` |
| `"badges ordinals 2, 3 and 4 as unconfirmed"` | tabs 2, 3, 4 contain the text `unconfirmed` |
| `"marks an absent row"` | ordinal 1's tab contains `not configured`; ordinal 0's does not |
| `"only the selected tab is in the tab order"` | with `ordinal: 2`, the ordinal tab at index 2 has `tabIndex` `0` and the others `-1` |
| `"ArrowRight moves to the next ordinal"` | focus ordinal tab 0, press `{ArrowRight}` → `onSelect` called with `(1, 0)` |
| `"ArrowLeft wraps from the first to the last"` | focus tab 0, press `{ArrowLeft}` → `onSelect` called with `(4, 0)` |
| `"Home and End jump to the ends"` | from tab 2, `{Home}` → `(0, 0)`; `{End}` → `(4, 0)` |
| `"the gender tablist has its own roving tabindex"` | focus gender tab 0, `{ArrowRight}` → `onSelect` called with `(<current ordinal>, 1)` |
| `"clicking a tab selects it"` | click the Female tab → `onSelect` called with `(<current ordinal>, 1)` |

`__tests__/IdentitySection.test.tsx`:

| case | assertion |
|---|---|
| `"shows the derived note for ordinals 0 and 1"` | draft with `ordinal: 1` → text matching `/derived from the client's own step-skip/i`; the unconfirmed notice is NOT present |
| `"shows a persistent unconfirmed notice for ordinal 2"` | draft with `ordinal: 2` → `screen.getByRole("note")` with text matching `/not derived from the client/i` and `/§A6/`; there is no dismiss button |
| `"the job field stays editable under the unconfirmed notice"` | ordinal 2, `jobs.options` non-empty → the job control is NOT disabled |
| `"reports pending job names distinctly from an empty list"` | `jobs = { options: [], isPending: true, isError: false }` → text `/loading job names/i`; the job control is disabled |
| `"reports a job-name error distinctly"` | `jobs = { options: [], isPending: false, isError: true }` → text `/job names unavailable/i` |
| `"level is bounded 1..200"` | the level input has `min="1"` and `max="200"` |
| `"a loaded out-of-range level is displayed as loaded"` | draft with `level: 240` → the level input's value is `"240"` (not clamped) |
| `"editing the level reports through onSetLevel"` | clear the level input and type `35` → `onSetLevel` last called with `35` |
| `"renders field errors"` | `errors = { jobId: ["bad job"] }` → `screen.getByText("bad job")` |

- [ ] **Step 2: Run the tests and verify they fail**

```
cd services/atlas-ui && npx vitest run src/components/features/characters/maple-life/__tests__/ClassSelector.test.tsx src/components/features/characters/maple-life/__tests__/IdentitySection.test.tsx
```

Expected: FAIL — modules not found.

- [ ] **Step 3: Implement both components**

To the **Produces** contract above. Copy the keyboard block from
`TemplateSelector.tsx:25-57` into a small local helper used by both tablists
rather than duplicating it twice inside `ClassSelector`.

- [ ] **Step 4: Run the tests and verify they pass**

```
cd services/atlas-ui && npx vitest run src/components/features/characters/maple-life
cd services/atlas-ui && npm run lint
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/maple-life/ClassSelector.tsx \
        services/atlas-ui/src/components/features/characters/maple-life/IdentitySection.tsx \
        services/atlas-ui/src/components/features/characters/maple-life/__tests__/ClassSelector.test.tsx \
        services/atlas-ui/src/components/features/characters/maple-life/__tests__/IdentitySection.test.tsx
git commit -m "feat(atlas-ui): add the Maple Life class selector and identity section"
```

---

## Task 8: `AppearancePoolsSection` and `MapleLifePreviewCard`

FR-6 (four pools per gender, with value domains and the allow-list statement)
and FR-10 (sticky preview, explicit gender, combination count).

### Files

- `services/atlas-ui/src/components/features/characters/maple-life/AppearancePoolsSection.tsx` — **new file**
- `services/atlas-ui/src/components/features/characters/maple-life/MapleLifePreviewCard.tsx` — **new file**
- `services/atlas-ui/src/components/features/characters/maple-life/__tests__/AppearancePoolsSection.test.tsx` — **new file**
- `services/atlas-ui/src/components/features/characters/maple-life/__tests__/MapleLifePreviewCard.test.tsx` — **new file**

Patterns to copy:
- `services/atlas-ui/src/components/features/characters/templates/CharacterTemplatesEditor.tsx` lines 219-253 (post-Task-3 form) — how a caller wires the generalised `AppearancePoolSection` plus its `AppearanceBrowserDialog`.
- `services/atlas-ui/src/components/features/characters/templates/__tests__/AppearancePoolSection.test.tsx` lines 5-12 — the `vi.mock("@/context/tenant-context")` block, required by anything that renders a thumb.
- `services/atlas-ui/src/components/features/characters/templates/PreviewCard.tsx` — the sticky render-card shape.

Read-only references: `templates/AppearanceBrowserDialog.tsx:45-61`,
`templates/AppearanceThumb.tsx:11-25`,
`services/api/characterRender.service.ts:105-112` (`generateCharacterUrl`).

Module root: `services/atlas-ui`.

### Interfaces

**Consumes** from Task 3: the generalised `AppearancePoolSection`. From Task 4:
`MapleLifeLookDraft`, `MapleLifeClassDraft`, `PreviewPicks`, `LookDimension`.
From Task 6: `buildMapleLifeLoadout`, `buildMapleLifeVariantLoadout`,
`combinationCount`.

**Produces:**

```ts
// AppearancePoolsSection.tsx
interface AppearancePoolsSectionProps {
  look: MapleLifeLookDraft;
  draft: MapleLifeClassDraft;
  picks: PreviewPicks;
  onPick: (pick: keyof PreviewPicks, index: number) => void;
  onAddEntry: (dimension: LookDimension, id: number) => void;
  onRemoveEntry: (dimension: LookDimension, entryIndex: number) => void;
  /** Blocking messages keyed "faces" | "hairs" | "hairColors" | "skinColors". */
  errors?: Record<string, string[]>;
}
export function AppearancePoolsSection(props: AppearancePoolsSectionProps): JSX.Element;

// MapleLifePreviewCard.tsx
interface MapleLifePreviewCardProps {
  draft: MapleLifeClassDraft;
  look: MapleLifeLookDraft;
  picks: PreviewPicks;
}
export function MapleLifePreviewCard(props: MapleLifePreviewCardProps): JSX.Element;
```

`AppearancePoolsSection` renders four `AppearancePoolSection`s from a local
table, passing the FR-6.5 value domain and the FR-6.6 allow-list line as the
new `description` prop:

```ts
const POOLS: {
  dimension: LookDimension;
  title: string;
  pick: keyof PreviewPicks;
  domain: string;
}[] = [
  { dimension: "faces", title: "Faces", pick: "faceIdx",
    domain: "Full item ids, e.g. 20000 (male) / 21000 (female)." },
  { dimension: "hairs", title: "Hairs", pick: "hairIdx",
    domain: "Normalised style ids — (v/10)*10, e.g. 30030." },
  { dimension: "hairColors", title: "Hair colours", pick: "hairColorIdx",
    domain: "Bare digits 0..9, not full item ids." },
  { dimension: "skinColors", title: "Skin tones", pick: "skinIdx",
    domain: "Bare byte ordinals." },
];

const ALLOW_LIST_NOTE =
  "This is an allow-list: the client sources its own carousel from WZ and the server only checks membership, so a list that diverges from the client's options produces player-visible ErrLookInvalid rejections.";
```

Each section receives `pool={look[dimension]}`,
`selectedIndex={picks[pick]}`,
`variantLoadout={(_dim, id) => buildMapleLifeVariantLoadout(draft, look, picks, dimension, id)}`,
`onPick={(idx) => onPick(pick, idx)}`,
`onRemoveEntry={(idx) => onRemoveEntry(dimension, idx)}`,
`description={<>{domain} {ALLOW_LIST_NOTE}</>}`, and a `renderAddDialog` that
mounts `AppearanceBrowserDialog` with `gender={look.gender}` — explicit, never
inferred (FR-10.1) — `selectMode="add"`, `markedIds={look[dimension]}`, and
`onSelect={(id) => onAddEntry(dimension, id)}`.

Any `errors[dimension]` messages render under that section in
`text-destructive`.

`MapleLifePreviewCard` renders one `<img loading="lazy">` at
`generateCharacterUrl(activeTenant.id, region, majorVersion, minorVersion,
buildMapleLifeLoadout(draft, look, picks), { stance: "stand1", resize: 2 })`,
plus the combination readout:

```
{combinationCount(look)} combinations offered
```

and the four factors beneath it as
`` `${look.faces.length} faces × ${look.hairs.length} hairs × ${look.hairColors.length} hair colours × ${look.skinColors.length} skin tones` ``
(FR-10.3). The card is `sticky top-4` inside the right column.

- [ ] **Step 1: Write the failing tests**

`__tests__/AppearancePoolsSection.test.tsx` — `vi.mock("@/context/tenant-context")`
copied from `templates/__tests__/AppearancePoolSection.test.tsx:5-12`.

| case | assertion |
|---|---|
| `"renders all four pool sections"` | headings `Faces`, `Hairs`, `Hair colours`, `Skin tones` all present |
| `"renders one thumb per entry"` | look row with `faces: [20000, 20001]` → both ids appear as captions |
| `"states each dimension's value domain"` | text `/Normalised style ids/`, `/Bare digits 0\.\.9/`, `/Bare byte ordinals/`, `/Full item ids/` all present |
| `"states the allow-list semantics once per pool"` | `screen.getAllByText(/ErrLookInvalid rejections/)` has length 4 |
| `"clicking a thumb reports the pick for that dimension"` | click `Preview hair 30030` → `onPick` called with `("hairIdx", 0)` |
| `"removing an entry reports the dimension and index"` | click `Remove face 20001` → `onRemoveEntry` called with `("faces", 1)` |
| `"an emptied pool renders its blocking error"` | `look.faces = []`, `errors = { faces: ["This pool is empty…"] }` → that message is in the document |
| `"passes the look row's gender to the browser dialog"` | render with `look.gender = 1`, open the Faces add dialog, assert the dialog received `gender` 1 — spy via `vi.mock("../../templates/AppearanceBrowserDialog")` returning a stub that renders `data-gender={String(props.gender)}` |

`__tests__/MapleLifePreviewCard.test.tsx` — same tenant-context mock.

| case | assertion |
|---|---|
| `"composes hair as color + 10*floor(style/10)"` | `hairs: [30035]`, `hairColors: [2]`, picks at index 0 → the `img` `src` contains `hair=30032` (NOT `30037`) |
| `"passes the look row's gender"` | `look.gender = 1` → the `src` contains `gender=1` |
| `"reports the combination count"` | 3 faces, 3 hairs, 4 colours, 4 skins → text `/144 combinations offered/` |
| `"reports the four factors"` | text `/3 faces × 3 hairs × 4 hair colours × 4 skin tones/` |
| `"a zero-combination pool reads zero"` | `faces: []` → text `/0 combinations offered/` |
| `"the render image is lazy"` | the `img` has `loading="lazy"` |

If `generateCharacterUrl` encodes the loadout as JSON rather than discrete query
params, assert on the decoded value rather than a substring — read
`src/services/api/characterRender.service.ts:105-140` before writing the
assertion and match its actual encoding.

- [ ] **Step 2: Run the tests and verify they fail**

```
cd services/atlas-ui && npx vitest run src/components/features/characters/maple-life/__tests__/AppearancePoolsSection.test.tsx src/components/features/characters/maple-life/__tests__/MapleLifePreviewCard.test.tsx
```

Expected: FAIL — modules not found.

- [ ] **Step 3: Implement both components**

To the **Produces** contract above.

- [ ] **Step 4: Run the tests and verify they pass**

```
cd services/atlas-ui && npx vitest run src/components/features/characters/maple-life
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/maple-life/AppearancePoolsSection.tsx \
        services/atlas-ui/src/components/features/characters/maple-life/MapleLifePreviewCard.tsx \
        services/atlas-ui/src/components/features/characters/maple-life/__tests__/AppearancePoolsSection.test.tsx \
        services/atlas-ui/src/components/features/characters/maple-life/__tests__/MapleLifePreviewCard.test.tsx
git commit -m "feat(atlas-ui): add Maple Life appearance pools and live preview card"
```

---

## Task 9: `ProgressionSection`, `SpSkillSection`, `StartingKitSection`

FR-7 (stats/ap/meso/ten SP books, correctly labelled), FR-8 (the SP skill offer
and its inertness at ordinals >= 2), FR-9 (reuse the presets kit sections
unchanged).

### Files

- `services/atlas-ui/src/components/features/characters/maple-life/ProgressionSection.tsx` — **new file**
- `services/atlas-ui/src/components/features/characters/maple-life/SpSkillSection.tsx` — **new file**
- `services/atlas-ui/src/components/features/characters/maple-life/StartingKitSection.tsx` — **new file**
- `services/atlas-ui/src/components/features/characters/maple-life/__tests__/ProgressionSection.test.tsx` — **new file**
- `services/atlas-ui/src/components/features/characters/maple-life/__tests__/SpSkillSection.test.tsx` — **new file**
- `services/atlas-ui/src/components/features/characters/maple-life/__tests__/StartingKitSection.test.tsx` — **new file**

Patterns to copy: `services/atlas-ui/src/components/features/characters/presets/BaseStatsSection.tsx`
for the six-stat grid layout, and
`services/atlas-ui/src/components/features/characters/presets/useSyncedNumberInput.ts`
lines 13-23 for every number input on the page.

**Do not reuse `BaseStatsSection` as a component**: it is typed against
`CharacterPresetAttributes` and its footer copy ("Written verbatim to the
created character") is wrong for Maple Life (design §7.3). What FR-7.1 actually
names is `useSyncedNumberInput`, and that is what is reused.

Read-only references (used unchanged, NOT edited):
`presets/EquipmentSection.tsx:15-20`, `presets/InventorySection.tsx:7-12`.

This task edits exactly 6 files — three sibling section components that share
the `useSyncedNumberInput` idiom and one progression concern. Splitting them
would give a reviewer three near-identical surfaces to gate separately.

Module root: `services/atlas-ui`.

### Interfaces

**Consumes** from Task 1: `EquipmentEntry`, `InventoryEntry`. From Task 4:
`MapleLifeClassDraft`, `StatKey`, `ScalarKey`. From Task 6:
`KNOWN_SP_SKILL_IDS`, `SP_SKILL_LABELS`.

**Produces:**

```ts
// ProgressionSection.tsx
interface ProgressionSectionProps {
  draft: MapleLifeClassDraft;
  onSetStat: (stat: StatKey, value: number) => void;
  onSetScalar: (field: ScalarKey, value: number) => void;
  onSetSpBook: (index: number, value: number) => void;
  /** Blocking messages keyed "sp" | "ap" | "meso" | a stat name. */
  errors?: Record<string, string[]>;
}
export function ProgressionSection(props: ProgressionSectionProps): JSX.Element;

// SpSkillSection.tsx
interface SpSkillSectionProps {
  draft: MapleLifeClassDraft;
  onSetSpSkillId: (value: number | undefined) => void;
  errors?: Record<string, string[]>;
  warnings?: string[];
}
export function SpSkillSection(props: SpSkillSectionProps): JSX.Element;

// StartingKitSection.tsx
interface StartingKitSectionProps {
  equipment: EquipmentEntry[];
  inventory: InventoryEntry[];
  onAddEquipment: (templateId: number) => void;
  onRemoveEquipment: (index: number) => void;
  onSetEquipmentAvg: (index: number, value: boolean) => void;
  onAddInventory: (templateId: number) => void;
  onRemoveInventory: (index: number) => void;
  onSetInventoryQty: (index: number, value: number) => void;
}
export function StartingKitSection(props: StartingKitSectionProps): JSX.Element;
```

**Required copy, verbatim** (FR-7.2, FR-7.3 — without these labels the numbers
read as wrong):

```ts
const STATS_LABEL =
  "AP already spent to meet the job requirement. HP and MP here EXCLUDE the SP skill's own 29 × effectX contribution, which factory/maple_life.go adds at creation time.";
const UNSPENT_LABEL =
  "AP and SP are what remains UNSPENT at the configured level.";
const BOOK_ZERO_LABEL =
  "Book 0 is the only book Maple Life reads or spends.";
```

`ProgressionSection` renders:
- Six stat inputs (`str`, `dex`, `int`, `luk`, `hp`, `mp`) under `STATS_LABEL`.
- `ap` and `meso` inputs under `UNSPENT_LABEL`.
- Ten SP book inputs labelled `Book 0` … `Book 9`, with Book 0 visually
  distinguished (a `font-semibold` label plus `BOOK_ZERO_LABEL` beneath it)
  (FR-7.4). Values come from `draft.spBooks`.
- When `draft.spBooks.length !== 10` — an unparseable pool — render the ten
  inputs **disabled** and show the raw value with the text
  `` `Unparseable SP pool, preserved as loaded: ${draft.spRaw}` `` plus the
  `errors.sp` messages. The page never rewrites a value it does not understand
  (FR-11.13).
- Every number input goes through `useSyncedNumberInput`.

`SpSkillSection` renders a `Select` with exactly three options (FR-8.1):

| value | label |
|---|---|
| `none` | `None` |
| `1000001` | `Improved Max HP Increase (Warrior)` |
| `2000001` | `Improved Max MP Increase (Magician)` |

- For `draft.ordinal >= 2` the control is `disabled` with
  `aria-describedby` pointing at an inline paragraph reading
  `The client skips step 4 for class ordinal >= 2, so a value set here can never take effect and a player submitting sp != 0 is rejected outright.`
  (FR-8.2). Disabled, not merely dimmed.
- A loaded non-zero `spSkillId` is **shown**, never hidden behind the disabled
  control, and its `errors.spSkillId` messages render (FR-8.5).
- When a known skill is selected, render
  `` `Grants its level-5 prerequisite automatically. Effective player cap: ${Math.min(10, (draft.spBooks[0] ?? 0) - 5)}.` `` (FR-8.3).
- A loaded `spSkillId` outside the two known ids renders its option in the
  select as `` `Unknown skill ${id}` `` so the value is preserved and
  selectable, and the `warnings` messages render in
  `text-warning-foreground` (FR-8.4 / FR-11.9).

`StartingKitSection` composes the two presets components unchanged under a
single heading:

```tsx
<EquipmentSection
  equipment={equipment}
  onAdd={onAddEquipment}
  onRemove={onRemoveEquipment}
  onSetAvg={onSetEquipmentAvg}
/>
<InventorySection
  inventory={inventory}
  onAdd={onAddInventory}
  onRemove={onRemoveInventory}
  onSetQty={onSetInventoryQty}
/>
```

- [ ] **Step 1: Write the failing tests**

`__tests__/ProgressionSection.test.tsx`:

| case | assertion |
|---|---|
| `"labels stats as the skill-excluded midpoint"` | text matching `/EXCLUDE the SP skill's own 29 × effectX/` |
| `"labels ap and sp as unspent at level"` | text matching `/remains UNSPENT at the configured level/i` |
| `"renders ten book inputs"` | ten inputs whose labels match `/^Book [0-9]$/` |
| `"distinguishes book 0"` | text matching `/only book Maple Life reads or spends/i` |
| `"editing a book reports its index and value"` | clear Book 0 and type `75` → `onSetSpBook` last called with `(0, 75)` |
| `"a mid-edit keystroke is not clobbered"` | clear Book 0 entirely → the input's DOM value is `""` (the `useSyncedNumberInput` echo), not snapped back to the canonical value |
| `"an unparseable pool disables the books and shows the raw value"` | draft with `spBooks: []`, `spRaw: "61,0"` → the ten book inputs are disabled and text `/preserved as loaded: 61,0/` is present |
| `"renders the six stat inputs"` | labelled inputs `STR`, `DEX`, `INT`, `LUK`, `HP`, `MP` |
| `"editing a stat reports its key"` | clear `STR` and type `36` → `onSetStat` last called with `("str", 36)` |
| `"editing ap reports the scalar key"` | clear `AP` and type `124` → `onSetScalar` last called with `("ap", 124)` |
| `"renders field errors"` | `errors = { sp: ["bad pool"] }` → `screen.getByText("bad pool")` |

`__tests__/SpSkillSection.test.tsx`:

| case | draft | assertion |
|---|---|---|
| `"offers exactly three options"` | `ordinal: 0`, no `spSkillId` | opening the select shows `None`, `Improved Max HP Increase (Warrior)`, `Improved Max MP Increase (Magician)` and nothing else |
| `"is disabled at ordinal 2"` | `ordinal: 2` | the select `toBeDisabled()` |
| `"is disabled at ordinals 3 and 4"` | `ordinal: 3`, then `4` | disabled in both |
| `"is enabled at ordinals 0 and 1"` | `ordinal: 0`, then `1` | not disabled in either |
| `"the disabled control carries an accessible reason"` | `ordinal: 2` | the select's `aria-describedby` resolves to an element whose text matches `/client skips step 4/i` |
| `"a loaded non-zero value at ordinal >= 2 is visible, not hidden"` | `ordinal: 2, spSkillId: 1000001` | the label `Improved Max HP Increase (Warrior)` is in the document |
| `"a loaded non-zero value at ordinal >= 2 shows its blocking error"` | same, `errors = { spSkillId: ["blocked"] }` | `screen.getByText("blocked")` |
| `"states the prerequisite and the effective cap"` | `ordinal: 0, spSkillId: 1000001, spBooks[0] = 61` | text matching `/level-5 prerequisite/i` and `/Effective player cap: 10/` |
| `"the cap is spBooks[0] - 5 when that is below 10"` | `spBooks[0] = 8` | text `/Effective player cap: 3/` |
| `"an unknown skill id is preserved and warned about"` | `spSkillId: 9999999`, `warnings: ["no prerequisite"]` | the select shows `Unknown skill 9999999`; `screen.getByText("no prerequisite")` |
| `"selecting None clears the value"` | `ordinal: 0, spSkillId: 1000001`, choose `None` | `onSetSpSkillId` called with `undefined` |
| `"selecting a skill sets its id"` | `ordinal: 0`, choose the Magician option | `onSetSpSkillId` called with `2000001` |

`__tests__/StartingKitSection.test.tsx`:

| case | assertion |
|---|---|
| `"renders an equipment row per entry"` | equipment `[{ templateId: 1040021, useAverageStats: true }]` → the id `1040021` is rendered |
| `"renders an inventory row per entry"` | inventory `[{ templateId: 2000002, quantity: 100 }]` → the id `2000002` is rendered |
| `"toggling average stats reports the index"` | toggle the first equipment switch → `onSetEquipmentAvg` called with `(0, false)` |
| `"removing an inventory entry reports the index"` | click the first inventory remove → `onRemoveInventory` called with `(0)` |

Follow the mock setup in
`presets/__tests__/EquipmentSection.test.tsx` and
`presets/__tests__/InventorySection.test.tsx` — read them first and copy
whatever item-name/search mocks they establish, since this task mounts the same
two components.

- [ ] **Step 2: Run the tests and verify they fail**

```
cd services/atlas-ui && npx vitest run src/components/features/characters/maple-life/__tests__/ProgressionSection.test.tsx src/components/features/characters/maple-life/__tests__/SpSkillSection.test.tsx src/components/features/characters/maple-life/__tests__/StartingKitSection.test.tsx
```

Expected: FAIL — modules not found.

- [ ] **Step 3: Implement the three components**

To the **Produces** contract and the required copy above.

- [ ] **Step 4: Run the tests and verify they pass**

```
cd services/atlas-ui && npx vitest run src/components/features/characters/maple-life src/components/features/characters/presets
```

Expected: PASS. The presets suite must stay green — the two kit components are
used unchanged.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/maple-life/ProgressionSection.tsx \
        services/atlas-ui/src/components/features/characters/maple-life/SpSkillSection.tsx \
        services/atlas-ui/src/components/features/characters/maple-life/StartingKitSection.tsx \
        services/atlas-ui/src/components/features/characters/maple-life/__tests__/ProgressionSection.test.tsx \
        services/atlas-ui/src/components/features/characters/maple-life/__tests__/SpSkillSection.test.tsx \
        services/atlas-ui/src/components/features/characters/maple-life/__tests__/StartingKitSection.test.tsx
git commit -m "feat(atlas-ui): add Maple Life progression, SP skill offer, and starting kit"
```

---

## Task 10: Empty state and seed-from-template

FR-12. A tenant with no block gets a purposeful empty state naming the error a
player would hit, plus a Seed-from-template action; the template page gets a
plain "add the ten class rows" instead.

### Files

- `services/atlas-ui/src/components/features/characters/maple-life/EmptyState.tsx` — **new file**
- `services/atlas-ui/src/components/features/characters/maple-life/SeedFromTemplateDialog.tsx` — **new file**
- `services/atlas-ui/src/components/features/characters/maple-life/__tests__/SeedFromTemplateDialog.test.tsx` — **new file**

Patterns to copy: `services/atlas-ui/src/components/common/EmptyState.tsx:6-18`
(the shared `{ title, description, action }` props — this task's
`maple-life/EmptyState.tsx` wraps it with Maple-Life-specific copy, it does not
reimplement it).

Read-only reference: `src/lib/hooks/api/useTemplates.ts:83-90` (`useTemplates()`
returns `UseQueryResult<Template[], Error>`).

Module root: `services/atlas-ui`.

### Interfaces

**Consumes** from Task 1: `MapleLifeConfig`, `Template`/`TemplateAttributes`.
From Task 4: `isEmptyConfig`.

**Produces:**

```ts
// EmptyState.tsx
interface MapleLifeEmptyStateProps {
  /** Tenant context shows the seed action; template context shows "add rows". */
  onSeed?: () => void;
  onAddRows: () => void;
}
export function MapleLifeEmptyState(props: MapleLifeEmptyStateProps): JSX.Element;

// SeedFromTemplateDialog.tsx
export interface SeedCandidate {
  id: string;
  region: string;
  majorVersion: number;
  minorVersion: number;
  lookCount: number;
  classCount: number;
  eligible: boolean;
  mapleLife: MapleLifeConfig | undefined;
}

interface SeedFromTemplateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  seedFrom: { region: string; majorVersion: number; minorVersion: number };
  onSeed: (config: MapleLifeConfig) => void;
}
export function SeedFromTemplateDialog(props: SeedFromTemplateDialogProps): JSX.Element;
```

**Candidate resolution — do NOT use `useTemplatesByRegionAndVersion`.**
`templatesService.getByRegionAndVersion` calls `api.getOne` and returns
`[sortTemplate(response)]` (`templates.service.ts:457-473`) — a one-element array
by construction, so the FR-12.3 multi-match branch could never be reached
through it. The dialog uses `useTemplates()` (`getAll`, full attributes) and
filters client-side on `region === seedFrom.region &&
majorVersion === seedFrom.majorVersion && minorVersion === seedFrom.minorVersion`.

A candidate is `eligible` when `!isEmptyConfig(t.attributes.mapleLife)`.
Ineligible candidates are **listed with their reason shown**, never hidden — "the
template you expect has no block" must be visible rather than looking like a
zero-match.

Branches:

- exactly one eligible → a confirmation naming the template id, then `onSeed`
- more than one eligible → a picker listing `id`, `region v<major>.<minor>`, and
  `` `${classCount} classes · ${lookCount} looks` ``
- zero eligible → the plain statement `No template of this region and version
  carries a Maple Life block.` and no action

`onSeed` always receives `structuredClone(candidate.mapleLife!)` so the donor is
never mutated.

**Required copy, verbatim:**

```ts
const TENANT_EMPTY_TITLE = "Maple Life is disabled for this tenant";
const TENANT_EMPTY_BODY =
  "There is no mapleLife block on this configuration, so a player using a Cash/0543 item will fail with ErrMapleLifeNotConfigured.";
const TEMPLATE_EMPTY_TITLE = "This template has no Maple Life block";
const TEMPLATE_EMPTY_BODY =
  "Add the ten class rows to author one. Nothing is persisted until you save.";
```

`MapleLifeEmptyState` renders `TENANT_*` copy plus a `Seed from template` button
when `onSeed` is supplied, and `TEMPLATE_*` copy otherwise. Both variants render
an `Add the ten class rows` button wired to `onAddRows` (FR-12.5).

- [ ] **Step 1: Write the failing tests**

`__tests__/SeedFromTemplateDialog.test.tsx`. Mock the templates hook:

```ts
const useTemplates = vi.fn();
vi.mock("@/lib/hooks/api/useTemplates", () => ({
  useTemplates: () => useTemplates(),
}));
```

Build `Template` fixtures with the minimum `TemplateAttributes` shape (copy the
`fullAttributes()` helper from
`src/services/api/__tests__/templates-update.test.ts:23-38`) plus a `mapleLife`
block where required. `seedFrom` is always
`{ region: "GMS", majorVersion: 83, minorVersion: 1 }`.

| case | `useTemplates` data | assertion |
|---|---|---|
| `"zero matches states it plainly and offers no action"` | one template at GMS 87.1 | text `/No template of this region and version carries a Maple Life block/`; no button named `/seed/i` |
| `"a matched template with an empty block is listed as ineligible, not hidden"` | one GMS 83.1 template whose `mapleLife` is `{ looks: [], classes: [] }` | its id IS rendered, alongside a reason matching `/no Maple Life block/i`; no seed action for it |
| `"a matched template with no mapleLife key at all is ineligible"` | one GMS 83.1 template with `mapleLife` undefined | same as above |
| `"exactly one eligible match seeds after confirmation"` | one GMS 83.1 template carrying `SEED_ML` | a confirm control naming the template id; clicking it calls `onSeed` once with a value `toEqual(SEED_ML)` |
| `"more than one eligible match presents a picker"` | two GMS 83.1 templates, both carrying blocks with different class counts | both ids rendered; each row shows its `N classes · M looks`; choosing the second calls `onSeed` with the second's block |
| `"a mixed result lists the ineligible alongside the eligible"` | one eligible + one empty-block, both GMS 83.1 | both ids rendered; only the eligible one has a seed action |
| `"the donor template is never mutated"` | one eligible; capture the arg, push onto its `classes` | the fixture's `classes.length` is unchanged |
| `"filters on all three of region, major and minor"` | GMS 83.1 (eligible), GMS 83.2 (eligible), JMS 83.1 (eligible) | only the GMS 83.1 id is rendered |

An `EmptyState.tsx` unit test is not required — its behaviour is asserted from
`MapleLifeEditor.test.tsx` in Task 11, which is where the tenant/template
branch actually diverges.

- [ ] **Step 2: Run the test and verify it fails**

```
cd services/atlas-ui && npx vitest run src/components/features/characters/maple-life/__tests__/SeedFromTemplateDialog.test.tsx
```

Expected: FAIL — modules not found.

- [ ] **Step 3: Implement both components**

To the **Produces** contract and the required copy above.

- [ ] **Step 4: Run the tests and verify they pass**

```
cd services/atlas-ui && npx vitest run src/components/features/characters/maple-life
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/maple-life/EmptyState.tsx \
        services/atlas-ui/src/components/features/characters/maple-life/SeedFromTemplateDialog.tsx \
        services/atlas-ui/src/components/features/characters/maple-life/__tests__/SeedFromTemplateDialog.test.tsx
git commit -m "feat(atlas-ui): add the Maple Life empty state and seed-from-template flow"
```

---

## Task 11: `MapleLifeEditor` — adapter, load, deep link, validation, save bar

The one component that owns behaviour. Everything else it composes already
exists and is tested.

### Files

- `services/atlas-ui/src/components/features/characters/maple-life/MapleLifeEditor.tsx` — **new file**
- `services/atlas-ui/src/components/features/characters/maple-life/__tests__/MapleLifeEditor.test.tsx` — **new file**

Patterns to copy:
- `services/atlas-ui/src/components/features/characters/templates/CharacterTemplatesEditor.tsx` lines 62-70 — the seed-once `loaded` guard, comment and all.
- `services/atlas-ui/src/components/features/characters/templates/CharacterTemplatesEditor.tsx` lines 72-109 — the deep-link apply-on-load-only effect, including the `eslint-disable-next-line react-hooks/exhaustive-deps` and the reason it carries.
- `services/atlas-ui/src/components/features/characters/templates/CharacterTemplatesEditor.tsx` lines 156-178 — the `useRegisterDetailActionBar` call and the pre-load skeleton/error gate.
- `services/atlas-ui/src/components/features/characters/templates/CharacterTemplatesEditor.tsx` lines 194-203 — the `grid gap-6 lg:grid-cols-[minmax(0,1fr)_252px]` two-column layout with `order-2 ... lg:order-1`.
- `services/atlas-ui/src/components/features/characters/presets/__tests__/CharacterPresetsEditor.test.tsx` lines 1-38 — the component-test mock setup.

Module root: `services/atlas-ui`.

### Interfaces

**Consumes** everything from Tasks 1, 2, 4, 5, 6, 7, 8, 9, 10.

**Produces:**

```ts
export interface MapleLifeEditorAdapter {
  mapleLife: MapleLifeConfig | undefined;
  isLoading: boolean;
  error: Error | null;
  /** Fire the context's PATCH; call onSuccess only when it lands. */
  save: (config: MapleLifeConfig, onSuccess: () => void) => void;
  isSaving: boolean;
  /** Tenant context only; absence hides the seed-from-template action. */
  seedFrom?: { region: string; majorVersion: number; minorVersion: number };
}

interface MapleLifeEditorProps {
  adapter: MapleLifeEditorAdapter;
}
export function MapleLifeEditor(props: MapleLifeEditorProps): JSX.Element;
```

**Behaviour contract:**

1. **Seed once.** `useEffect(() => { if (adapter.mapleLife !== undefined && !state.loaded) dispatch({ type: "load", config: adapter.mapleLife }); }, [adapter.mapleLife, state.loaded])`. Also load when the query has settled with `undefined` — gate on `!adapter.isLoading` so an unconfigured tenant reaches the empty state rather than a permanent skeleton.
2. **Pre-load gate.** `if (!state.loaded) { if (adapter.error) return <ErrorDisplay error={adapter.error} />; return <FormSkeleton fields={8} />; }` — the same shape as `CharacterTemplatesEditor.tsx:173-178`, so a transient refetch or save error never blanks an in-progress working copy (FR-3.4).
3. **Deep link.** Two params, applied **once**, `deps: [state.loaded]`. `?ordinal=` parses and clamps to `0..4`; `?gender=` to `0..1`; anything unparseable clamps to `0`. Both are written back in a single `setSearchParams(prev => ..., { replace: true })` when either differs from its raw value. Every later selection change writes the URL directly from the click handler (a `syncSelection(ordinal, gender)` helper), never from an effect — the grid is fixed at ten so there is no length to watch.
4. **Validation.** `const issues = validateMapleLife(projectForSave(state))` and `const warnings = warningMap(state)`, both recomputed per render. `blockingIssues` is the total message count in `issues`.
5. **Save bar.**

```tsx
useRegisterDetailActionBar(
  state.loaded && !isEmptyConfig(projectForSave(state))
    ? {
        dirty,
        isSaving: adapter.isSaving,
        blockingIssues,
        onSave: () =>
          adapter.save(projectForSave(state), () =>
            dispatch({ type: "savedOk" }),
          ),
        onDiscard: discardChanges,
      }
    : null,
);
```

6. **Empty state.** When `isEmptyConfig(projectForSave(state))`, render
   `<MapleLifeEmptyState onSeed={adapter.seedFrom ? () => setSeedOpen(true) : undefined} onAddRows={() => dispatch({ type: "materialiseAll" })} />` plus, when
   `adapter.seedFrom` is present, the `SeedFromTemplateDialog` wired to
   `dispatch({ type: "seedFromTemplate", config })`.
7. **Issue routing.** The schema's paths address the **projected** array
   (`classes.<projIndex>...`); the warnings' paths address the **fixed grid**
   (`classes.<ordinal>.<gender>`). Build a projected-index → `(ordinal, gender)`
   lookup once from `projectForSave(state).classes` and re-key the schema issues
   onto the grid before slicing them per section. Never merge the two maps.
8. **Section props.** Pass only the slice each section needs: `IdentitySection`
   gets `{ jobId, level, mapId }` errors; `ProgressionSection` gets `{ sp, ap,
   meso, str, dex, int, luk, hp, mp }`; `SpSkillSection` gets `{ spSkillId }`
   errors and the `spSkillId` warnings; `AppearancePoolsSection` gets the four
   `looks.<gender>.<dimension>` errors re-keyed to bare dimension names.
9. **No post-save refetch.** Nothing in `mapleLife` is server-assigned (no ids),
   so the request echo is the truth — unlike the presets editor, which refetches
   to pick up server-issued ids.
10. **Job names.** `const jobs = usePresetJobOptions();` and
    `const jobNameById = jobs.isPending || jobs.isError ? null : new Map(jobs.options.map(o => [o.id, o.name]));`

- [ ] **Step 1: Write the failing tests**

`__tests__/MapleLifeEditor.test.tsx`. Setup copied from
`presets/__tests__/CharacterPresetsEditor.test.tsx:1-38`: `MemoryRouter` wrapper
(the editor uses `useSearchParams`), `vi.mock("@/components/DetailActionBarContext")`
capturing the registered config, `vi.mock("sonner")`,
`vi.mock("@/context/tenant-context")`, and shallow `vi.mock`s of the heavy
children (`../AppearancePoolsSection`, `../MapleLifePreviewCard`,
`../StartingKitSection`) so this test asserts wiring, not their internals.

| case | setup | assertion |
|---|---|---|
| `"renders a skeleton before the adapter delivers"` | `isLoading: true`, `mapleLife: undefined` | a skeleton is rendered; no class selector |
| `"renders the error display when the query fails before load"` | `error: new Error("boom")` | text `/boom/` |
| `"seeds the reducer exactly once"` | render with `SEED`, edit `ap`, then rerender with a NEW `SEED` object identity | the edited `ap` survives — the second delivery does not clobber the working copy |
| `"applies ?ordinal and ?gender on load"` | initial entries `["/?ordinal=3&gender=1"]` | the ordinal tab 3 and the Female tab are selected |
| `"clamps an out-of-range ordinal to 0"` | `["/?ordinal=9"]` | ordinal 0 selected; the URL is rewritten to `ordinal=0` |
| `"clamps an unparseable gender to 0"` | `["/?gender=abc"]` | Male selected; the URL is rewritten to `gender=0` |
| `"a selection click writes the URL"` | click the Female tab | the search string contains `gender=1` |
| `"registers a clean bar after load"` | `SEED` | the captured config has `dirty: false`, `blockingIssues: 0` |
| `"reports blocking errors in the bar"` | a config whose class 0 has `sp: "61,0"` | the captured config's `blockingIssues` is `>= 1` |
| `"clears blockingIssues once the error is fixed"` | same, then fix the pool via the ten book inputs | the captured `blockingIssues` returns to `0` |
| `"saves only the mapleLife block"` | `SEED`, edit `ap` to 124, invoke the captured `onSave` | `adapter.save` called once; its first argument `toEqual` a `MapleLifeConfig` — an object with exactly the keys `looks` and `classes` — whose `classes[0].ap` is `124` |
| `"a successful save clears dirty"` | invoke `onSave`, then invoke the `onSuccess` callback the adapter received | the newly captured config has `dirty: false` |
| `"registers no bar for an empty configuration"` | `mapleLife: undefined`, `isLoading: false` | the captured config is `null` |
| `"an empty tenant configuration offers Seed from template"` | `mapleLife: undefined`, `isLoading: false`, `seedFrom: { region: "GMS", majorVersion: 83, minorVersion: 1 }` | a control named `/seed from template/i` is present |
| `"an empty template configuration does not"` | same but no `seedFrom` | no `/seed from template/i` control; text `/Add the ten class rows/i` is present |
| `"adding the ten rows marks the page dirty"` | empty, click `/Add the ten class rows/i` | the captured config has `dirty: true` and ten class rows are selectable |

- [ ] **Step 2: Run the test and verify it fails**

```
cd services/atlas-ui && npx vitest run src/components/features/characters/maple-life/__tests__/MapleLifeEditor.test.tsx
```

Expected: FAIL — module not found.

- [ ] **Step 3: Implement `MapleLifeEditor.tsx`**

To the ten-point behaviour contract above. Layout: `ClassSelector` full-width at
the top, then the two-column grid — sections in the left column
(`IdentitySection`, `AppearancePoolsSection`, `ProgressionSection`,
`SpSkillSection`, `StartingKitSection`), `MapleLifePreviewCard` sticky in the
right column, ordered first on narrow screens via `order-2 ... lg:order-1` on
the left column.

- [ ] **Step 4: Run the tests and verify they pass**

```
cd services/atlas-ui && npx vitest run src/components/features/characters/maple-life
cd services/atlas-ui && npm run build && npm run lint
```

Expected: PASS, `tsc -b` clean, lint clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/maple-life/MapleLifeEditor.tsx \
        services/atlas-ui/src/components/features/characters/maple-life/__tests__/MapleLifeEditor.test.tsx
git commit -m "feat(atlas-ui): add the shared Maple Life editor"
```

---

## Task 12: The two pages and their routes

FR-2.1, FR-3.2. Thin per-context wrappers, exactly the sibling shape.

### Files

- `services/atlas-ui/src/pages/TenantsMapleLifePage.tsx` — **new file**
- `services/atlas-ui/src/pages/TemplatesMapleLifePage.tsx` — **new file**
- `services/atlas-ui/src/App.tsx` — two `lazyWithReload` imports and two `<Route>` entries

Patterns to copy:
- `services/atlas-ui/src/pages/TenantsCharacterTemplatesPage.tsx` lines 1-41 — the tenant page shape (adapter literal, `updateTenantConfig.mutate({ tenant, updates }, { onSuccess, onError })`, toasts).
- `services/atlas-ui/src/pages/TemplatesCharacterTemplatesPage.tsx` lines 16-41 — the template page shape, including the `...template.attributes` spread that FR-1.4 depends on.
- `services/atlas-ui/src/App.tsx` lines 254-263 — the `lazyWithReload` block form.
- `services/atlas-ui/src/App.tsx` lines 414-421 and `:444-451` — the `<Route>` form.

Module root: `services/atlas-ui`.

### Interfaces

**Consumes** from Task 11: `MapleLifeEditor`, `MapleLifeEditorAdapter`.

**Produces:** `TenantsMapleLifePage`, `TemplatesMapleLifePage` — both named
exports, matching the `lazyWithReload(() => import(...).then(m => ({ default: m.X })))`
convention.

Hook import paths (these are NOT in the service files):
`useTenantConfiguration` / `useUpdateTenantConfiguration` /
`useTenant` come from `@/lib/hooks/api/useTenants`;
`useTemplate` / `useUpdateTemplate` come from `@/lib/hooks/api/useTemplates`.

Tenant adapter:

```tsx
const adapter: MapleLifeEditorAdapter = {
  mapleLife: tenant?.attributes.mapleLife,
  isLoading: tenantQuery.isLoading,
  error: tenantQuery.error ?? null,
  isSaving: updateTenantConfig.isPending,
  ...(tenant
    ? {
        seedFrom: {
          region: tenant.attributes.region,
          majorVersion: tenant.attributes.majorVersion,
          minorVersion: tenant.attributes.minorVersion,
        },
      }
    : {}),
  save: (mapleLife, onSuccess) => {
    if (!tenant) return;
    updateTenantConfig.mutate(
      { tenant, updates: { mapleLife } },
      {
        onSuccess: () => {
          toast.success("Successfully saved Maple Life configuration.");
          onSuccess();
        },
        onError: (error) =>
          toast.error(`Failed to update Maple Life configuration: ${error.message}`),
      },
    );
  },
};
```

Template adapter — identical but for `useTemplate` / `useUpdateTemplate`, **no**
`seedFrom` (its absence is what hides the FR-12 action), and the whole-document
update the template PATCH requires:

```tsx
  save: (mapleLife, onSuccess) => {
    if (!template) return;
    updateTemplate.mutate(
      { id: template.id, updates: { ...template.attributes, mapleLife } },
      {
        onSuccess: () => {
          toast.success("Successfully saved Maple Life configuration.");
          onSuccess();
        },
        onError: (error) =>
          toast.error(`Failed to update Maple Life configuration: ${error.message}`),
      },
    );
  },
```

Each page wraps the editor in its detail layout: `<TenantDetailLayout>` /
`<TemplateDetailLayout>`.

- [ ] **Step 1: Write the pages**

Both files to the shapes above.

- [ ] **Step 2: Register the routes in `App.tsx`**

Add beside the existing lazy blocks (after `:225` and after `:263`
respectively):

```tsx
const TemplatesMapleLifePage = lazyWithReload(() =>
  import("@/pages/TemplatesMapleLifePage").then((m) => ({
    default: m.TemplatesMapleLifePage,
  })),
);
```

```tsx
const TenantsMapleLifePage = lazyWithReload(() =>
  import("@/pages/TenantsMapleLifePage").then((m) => ({
    default: m.TenantsMapleLifePage,
  })),
);
```

Add the routes immediately after the existing presets routes (after `:421` and
after `:451`):

```tsx
              <Route
                path="/templates/:id/character/maple-life"
                element={<TemplatesMapleLifePage />}
              />
```

```tsx
              <Route
                path="/tenants/:id/character/maple-life"
                element={<TenantsMapleLifePage />}
              />
```

Match the surrounding indentation exactly.

- [ ] **Step 3: Verify the build and the whole suite**

```
cd services/atlas-ui && npm run build && npm run lint && npm run test
```

Expected: `tsc -b` clean, lint clean, every test PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-ui/src/pages/TenantsMapleLifePage.tsx \
        services/atlas-ui/src/pages/TemplatesMapleLifePage.tsx \
        services/atlas-ui/src/App.tsx
git commit -m "feat(atlas-ui): route the tenant and template Maple Life pages"
```

---

## Task 13: Navigation rails and breadcrumbs

FR-2.2, FR-2.3. The last wiring: the page must be reachable and must produce a
correct trail.

### Files

- `services/atlas-ui/src/lib/breadcrumbs/routes.ts` — two patterns and two route constants
- `services/atlas-ui/src/lib/breadcrumbs/__tests__/routes.test.ts` — coverage for both patterns
- `services/atlas-ui/src/components/features/tenants/TenantDetailLayout.tsx` — rail entry
- `services/atlas-ui/src/components/features/templates/TemplateDetailLayout.tsx` — rail entry

Patterns to copy: the existing `/character/presets` entries —
`routes.ts:392-396` (template pattern), `routes.ts:439-443` (tenant pattern),
`routes.ts:663` / `routes.ts:701` (the constants),
`TenantDetailLayout.tsx:23`, `TemplateDetailLayout.tsx:24` (the rail lines).
Read `services/atlas-ui/src/lib/breadcrumbs/__tests__/routes.test.ts` first and
follow whatever assertion form it already uses for
`/tenants/[id]/character/presets`.

Module root: `services/atlas-ui`.

### Interfaces

**Consumes** from Task 12: the two route paths.

**Produces:** `TENANT_CHARACTER_MAPLE_LIFE` and `TEMPLATE_CHARACTER_MAPLE_LIFE`
in the route-constant map at the bottom of `routes.ts`.

- [ ] **Step 1: Write the failing breadcrumb tests**

Add to `src/lib/breadcrumbs/__tests__/routes.test.ts`, mirroring the existing
`/character/presets` cases:

| case | assertion |
|---|---|
| `"resolves the tenant Maple Life trail"` | `/tenants/abc/character/maple-life` resolves to the labels `Tenants` → `abc` (or whatever the existing tenant-detail case asserts) → `Character` → `Maple Life`, with the `Character` node non-navigable |
| `"resolves the template Maple Life trail"` | `/templates/abc/character/maple-life` resolves to the equivalent template trail ending in `Maple Life` |
| `"exposes both route constants"` | `ROUTES.TENANT_CHARACTER_MAPLE_LIFE` `toBe("/tenants/[id]/character/maple-life")` and `ROUTES.TEMPLATE_CHARACTER_MAPLE_LIFE` `toBe("/templates/[id]/character/maple-life")` (use the actual exported map name this file already imports) |

- [ ] **Step 2: Run the test and verify it fails**

```
cd services/atlas-ui && npx vitest run src/lib/breadcrumbs/__tests__/routes.test.ts
```

Expected: FAIL — no pattern registered for the two paths.

- [ ] **Step 3: Register the patterns and constants**

After `routes.ts:396`:

```ts
  {
    pattern: "/templates/[id]/character/maple-life",
    label: "Maple Life",
    parent: "/templates/[id]/character",
  },
```

After `routes.ts:443`:

```ts
  {
    pattern: "/tenants/[id]/character/maple-life",
    label: "Maple Life",
    parent: "/tenants/[id]/character",
  },
```

In the constant map, after `TEMPLATE_CHARACTER_PRESETS` (`:663`):

```ts
  TEMPLATE_CHARACTER_MAPLE_LIFE: "/templates/[id]/character/maple-life",
```

and after `TENANT_CHARACTER_PRESETS` (`:701`):

```ts
  TENANT_CHARACTER_MAPLE_LIFE: "/tenants/[id]/character/maple-life",
```

- [ ] **Step 4: Add both rail entries**

`TenantDetailLayout.tsx`, after the `Character Presets` line (`:23`):

```ts
    { title: "Maple Life", href: `/tenants/${id}/character/maple-life` },
```

`TemplateDetailLayout.tsx`, after the `Character Presets` line (`:24`):

```ts
    { title: "Maple Life", href: `/templates/${id}/character/maple-life` },
```

- [ ] **Step 5: Run the full suite and the build**

```
cd services/atlas-ui && npx vitest run src/lib/breadcrumbs
cd services/atlas-ui && npm run test && npm run build && npm run lint
```

Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/lib/breadcrumbs/routes.ts \
        services/atlas-ui/src/lib/breadcrumbs/__tests__/routes.test.ts \
        services/atlas-ui/src/components/features/tenants/TenantDetailLayout.tsx \
        services/atlas-ui/src/components/features/templates/TemplateDetailLayout.tsx
git commit -m "feat(atlas-ui): surface Maple Life in both detail rails and breadcrumbs"
```

---

## Final gate (not a task — the controller runs this)

```
tools/verify.sh
```

Must exit 0 with **no flags** before the branch is called done (CLAUDE.md "Done
means verified"). `--quick` / `--no-docker` do not count.

Then, before opening the PR:

- `frontend-guidelines-reviewer` over the changed TypeScript files (the whole
  change is frontend; `backend-guidelines-reviewer` has nothing to audit — no Go
  file is touched).
- `plan-adherence-reviewer` against this plan.
