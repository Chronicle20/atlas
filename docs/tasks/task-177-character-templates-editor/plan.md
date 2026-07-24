# Character Templates Editor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the two duplicated character-template forms in atlas-ui with one shared visual editor (segmented selector, icon/thumbnail pools, live atlas-renders preview, add/duplicate/remove lifecycle) used by both the Tenant Details and Template Details pages.

**Architecture:** One shared `CharacterTemplatesEditor` component under `src/components/features/characters/templates/`, driven by a pure `useReducer` state module and parameterized by a small data adapter each page wrapper builds from its existing React Query hooks. New data layer: cosmetics enumeration service/hooks and a batched item-name hook. Zero backend changes, zero layout/shell changes.

**Tech Stack:** Vite + React 19, TypeScript, TanStack React Query 5, shadcn/ui + Tailwind 4, Vitest + Testing Library (vi.* mocks), one new dep: `@radix-ui/react-popover`.

## Global Constraints

- `services/atlas-ui` only. No Go service changes. No changes to `TenantDetailLayout`, `TemplateDetailLayout`, `DetailSidebar`, `App.tsx` routes, or nav.
- Only new dependency allowed: `@radix-ui/react-popover` (for the shadcn Popover primitive). No `cmdk`.
- All commands in `services/atlas-ui` need node 22: prefix with `source ~/.nvm/nvm.sh && nvm use 22 >/dev/null 2>&1 &&`. Run them from `services/atlas-ui/`.
- `npm run build` runs `tsc -b` which type-checks `*.test.ts(x)` — tests and implementation must land together and compile.
- Tests use Vitest (`describe`/`it`/`expect`/`vi`) — NEVER `jest.*`. Colocate under `__tests__/` next to the code. Mock modules with the `vi.mock` + captured-mock-fn pattern (see `src/components/features/characters/__tests__/ApplyPresetDialog.test.tsx` for the house style).
- Named exports on pages/components; `@/` import alias; `import.meta.env.VITE_*` only.
- All sprite imagery gets `image-rendering: pixelated` (Tailwind arbitrary property `[image-rendering:pixelated]`).
- Styling via Tailwind + shadcn tokens only (works in light AND dark). No hardcoded palette values from the prototype.
- Persisted data shape `CharacterTemplate` (`src/types/models/template.ts:4-19`) is unchanged. Preview picks / selection are UI-only state, never persisted.
- The tenant API contract (`TENANT_ID`/`REGION`/`MAJOR_VERSION`/`MINOR_VERSION` headers) is injected by the existing API client — do not hand-roll headers.
- Commit after every task with `git add <files> && git commit`. Commit messages: `feat(task-177): <what>` (or `test(task-177):`/`refactor(task-177):` where apter).
- Final gates (Task 17): `npm run test`, `npm run lint`, `npm run build` clean in `services/atlas-ui`; `tools/lint.sh --check` clean at repo root.

## File Structure (target state)

All new components under `src/components/features/characters/templates/`:

| File | Responsibility |
|---|---|
| `jobNames.ts` | jobIndex→world-name map, segmented-control labels with ordinal suffixes, known class list |
| `editorState.ts` | Pure reducer + actions + selectors for the whole editor state |
| `previewLoadout.ts` | Pure loadout builders (preview + per-thumbnail variant), render-default constants, slot map |
| `poolSearchConfig.ts` | Per-pool item-search filter configs (server subcategory vs client subcategory-set) |
| `TemplateSelector.tsx` | Segmented control + `+ New`, `role="tablist"` |
| `TemplateActionsMenu.tsx` | Kebab menu (Duplicate / Remove-with-confirm) |
| `IdentitySection.tsx` | Class select + Advanced numeric fields, gender select, MapPicker, actions slot |
| `MapPicker.tsx` | Map combobox (name search + manual id + unresolvable-id hint) |
| `AppearanceThumb.tsx` | One cropped composite thumbnail button |
| `AppearancePoolSection.tsx` | Appearance pool row (thumbs + warning + add) |
| `AppearanceBrowserDialog.tsx` | Paginated visual add-browser for all four appearance dimensions |
| `ItemRow.tsx` | Shared icon+name+id row with remove × |
| `ItemSearchCombobox.tsx` | Popover search combobox with manual-id fallback |
| `EquipmentPoolSection.tsx` | Equipment pool rows + combobox add |
| `StartingKitSection.tsx` | Kit items rows + skills rows |
| `PreviewCard.tsx` | Sticky live preview + worn-icon strip |
| `SaveBar.tsx` | Sticky dirty/Discard/Save bar |
| `CharacterTemplatesEditor.tsx` | Assembly: adapter prop, reducer, URL sync, layout, empty/loading/error states |

New data layer: `src/services/api/cosmetics.service.ts`, `src/lib/hooks/api/useCosmetics.ts`, `src/lib/hooks/api/useItemNames.ts`. New primitive: `src/components/ui/popover.tsx`. Modified: `src/services/api/characterRender.service.ts` (extract `isFemaleCosmeticId`), the two page wrappers. Deleted: `src/pages/tenants-character-templates-form.tsx`, `src/pages/templates-character-templates-form.tsx`.

Verified API facts this plan relies on (do not re-derive): `fetchAll` at `src/services/api/pagination.ts:67`; `itemsService.searchItems(filters)` returns `{ items: {id,name,compartment,subcategory,type}[], total, pageNumber, pageSize, lastPage }`; `itemStringsService.getItemString(id)` returns `{ attributes: { name } }`; `useItemName` cache key is `["item-strings","name",<id>]` (`src/lib/hooks/api/useItemStrings.ts:5-8`); `useMapsByName(name)` / `useMap(id)` in `src/lib/hooks/api/useMaps.ts`; `MapData = { id: string, attributes: { name, streetName } }`; `generateCharacterUrl(tenant, region, major, minor, loadout, opts)` at `src/services/api/characterRender.service.ts:98`; `getAssetIconUrl(tenantId, region, major, minor, "item"|"skill", id)` at `src/lib/utils/asset-url.ts:5`; equipment render slots top=`-5`, bottom=`-6`, shoes=`-7`, weapon=`-11` (`src/components/features/characters/EquipmentPanel.tsx:26-50`); weapon subcategory tokens from `services/atlas-data/atlas.com/data/item/filter.go:55-60`; armor tokens `top`/`overall`/`bottom`/`shoes` from `classify.go:39-42`; `useTenant().activeTenant` is `{ id, attributes: { name, region, majorVersion, minorVersion } } | null`.

---

### Task 1: Template labels (`jobNames.ts`)

**Files:**
- Create: `src/components/features/characters/templates/jobNames.ts`
- Test: `src/components/features/characters/templates/__tests__/jobNames.test.ts`

**Interfaces:**
- Consumes: `CharacterTemplate` from `@/types/models/template`.
- Produces: `worldNameFromJobIndex(jobIndex: number): string`, `genderLabel(gender: number): "M" | "F"`, `templateLabels(templates: Pick<CharacterTemplate, "jobIndex" | "gender">[]): string[]`, `KNOWN_CLASSES: readonly { jobIndex: number; subJobIndex: number; label: string }[]`. Later tasks (Selector, ActionsMenu, Editor) use these exact names.

- [ ] **Step 1: Write the failing test**

`src/components/features/characters/templates/__tests__/jobNames.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import {
  worldNameFromJobIndex,
  genderLabel,
  templateLabels,
  KNOWN_CLASSES,
} from "../jobNames";

describe("worldNameFromJobIndex", () => {
  it("maps the four known job indexes (mirrors JobFromIndex)", () => {
    expect(worldNameFromJobIndex(0)).toBe("Cygnus Knights");
    expect(worldNameFromJobIndex(1)).toBe("Adventurer");
    expect(worldNameFromJobIndex(2)).toBe("Aran");
    expect(worldNameFromJobIndex(3)).toBe("Evan");
  });

  it("falls back to Job N for unknown indexes", () => {
    expect(worldNameFromJobIndex(7)).toBe("Job 7");
  });
});

describe("genderLabel", () => {
  it("maps 0 to M and 1 to F", () => {
    expect(genderLabel(0)).toBe("M");
    expect(genderLabel(1)).toBe("F");
  });
});

describe("templateLabels", () => {
  it("labels as <World> · <M|F>", () => {
    expect(templateLabels([{ jobIndex: 1, gender: 0 }])).toEqual([
      "Adventurer · M",
    ]);
  });

  it("suffixes ordinals only on duplicate labels, starting at (2)", () => {
    expect(
      templateLabels([
        { jobIndex: 1, gender: 0 },
        { jobIndex: 1, gender: 1 },
        { jobIndex: 1, gender: 0 },
        { jobIndex: 1, gender: 0 },
      ]),
    ).toEqual([
      "Adventurer · M",
      "Adventurer · F",
      "Adventurer · M (2)",
      "Adventurer · M (3)",
    ]);
  });
});

describe("KNOWN_CLASSES", () => {
  it("lists the four factory-mapped classes with jobIndex.subJobIndex labels", () => {
    expect(KNOWN_CLASSES).toEqual([
      { jobIndex: 0, subJobIndex: 0, label: "Cygnus Knights (0.0)" },
      { jobIndex: 1, subJobIndex: 0, label: "Adventurer (1.0)" },
      { jobIndex: 2, subJobIndex: 0, label: "Aran (2.0)" },
      { jobIndex: 3, subJobIndex: 0, label: "Evan (3.0)" },
    ]);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run (from `services/atlas-ui`): `source ~/.nvm/nvm.sh && nvm use 22 >/dev/null 2>&1 && npm run test -- src/components/features/characters/templates/__tests__/jobNames.test.ts`
Expected: FAIL — cannot resolve `../jobNames`.

- [ ] **Step 3: Write the implementation**

`src/components/features/characters/templates/jobNames.ts`:

```ts
import type { CharacterTemplate } from "@/types/models/template";

// World names mirror atlas-character-factory job/model.go JobFromIndex:
// 0 → Noblesse (Cygnus Knights), 1 → Beginner (Adventurer), 2 → Legend (Aran),
// 3 → Evan. Unknown indexes are permitted (backend is the validator of record).
const WORLD_NAMES: Record<number, string> = {
  0: "Cygnus Knights",
  1: "Adventurer",
  2: "Aran",
  3: "Evan",
};

export function worldNameFromJobIndex(jobIndex: number): string {
  return WORLD_NAMES[jobIndex] ?? `Job ${jobIndex}`;
}

export function genderLabel(gender: number): "M" | "F" {
  return gender === 1 ? "F" : "M";
}

/**
 * Segmented-control labels: "<World> · <M|F>", with " (2)", " (3)" ordinals
 * appended to the second and later occurrences of a duplicate label.
 */
export function templateLabels(
  templates: Pick<CharacterTemplate, "jobIndex" | "gender">[],
): string[] {
  const seen = new Map<string, number>();
  return templates.map((t) => {
    const base = `${worldNameFromJobIndex(t.jobIndex)} · ${genderLabel(t.gender)}`;
    const n = (seen.get(base) ?? 0) + 1;
    seen.set(base, n);
    return n === 1 ? base : `${base} (${n})`;
  });
}

export const KNOWN_CLASSES: readonly {
  jobIndex: number;
  subJobIndex: number;
  label: string;
}[] = [
  { jobIndex: 0, subJobIndex: 0, label: "Cygnus Knights (0.0)" },
  { jobIndex: 1, subJobIndex: 0, label: "Adventurer (1.0)" },
  { jobIndex: 2, subJobIndex: 0, label: "Aran (2.0)" },
  { jobIndex: 3, subJobIndex: 0, label: "Evan (3.0)" },
];
```

- [ ] **Step 4: Run test to verify it passes**

Same command as Step 2. Expected: PASS (6 tests).

- [ ] **Step 5: Commit**

```bash
git add src/components/features/characters/templates/jobNames.ts src/components/features/characters/templates/__tests__/jobNames.test.ts
git commit -m "feat(task-177): template label derivation (jobNames)"
```

### Task 2: Editor state reducer (`editorState.ts`)

**Files:**
- Create: `src/components/features/characters/templates/editorState.ts`
- Test: `src/components/features/characters/templates/__tests__/editorState.test.ts`

**Interfaces:**
- Consumes: `CharacterTemplate` from `@/types/models/template`.
- Produces (later tasks use these exact names):
  - Types: `PoolKey` (all ten array keys), `AppearancePoolKey` (`"faces"|"hairs"|"hairColors"|"skinColors"`), `IdentityField` (`"jobIndex"|"subJobIndex"|"gender"|"mapId"`), `PreviewPicks { faceIdx; hairIdx; hairColorIdx; skinIdx }`, `EditorState { templates; baseline; selectedIndex; previewPicks: Record<number, PreviewPicks>; loaded: boolean }`, `EditorAction` (union below).
  - Functions: `editorReducer(state, action): EditorState`, `initialEditorState(): EditorState`, `normalizeTemplate(raw): CharacterTemplate`, `blankTemplate(): CharacterTemplate`, `cloneTemplate(t): CharacterTemplate`, `isDirty(state): boolean`, `picksFor(state, index): PreviewPicks`, `emptyPoolWarnings(t): AppearancePoolKey[]`, `DEFAULT_PICKS: PreviewPicks`, `PICK_KEY_BY_POOL: Partial<Record<PoolKey, keyof PreviewPicks>>`.
  - Actions all operate on `state.selectedIndex` (kebab/duplicate/remove act on the selected template): `{type:"load"; templates: CharacterTemplate[]}`, `{type:"select"; index: number}`, `{type:"addTemplate"}`, `{type:"duplicateTemplate"}`, `{type:"removeTemplate"}`, `{type:"setIdentity"; field: IdentityField; value: number}`, `{type:"addPoolEntry"; pool: PoolKey; id: number}`, `{type:"removePoolEntry"; pool: PoolKey; entryIndex: number}`, `{type:"setPreviewPick"; pick: keyof PreviewPicks; value: number}`, `{type:"discard"}`, `{type:"savedOk"}`.

- [ ] **Step 1: Write the failing test**

`src/components/features/characters/templates/__tests__/editorState.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import {
  editorReducer,
  initialEditorState,
  normalizeTemplate,
  blankTemplate,
  isDirty,
  picksFor,
  emptyPoolWarnings,
  DEFAULT_PICKS,
  type EditorState,
} from "../editorState";

function loaded(templates: Parameters<typeof normalizeTemplate>[0][]): EditorState {
  return editorReducer(initialEditorState(), {
    type: "load",
    templates: templates.map(normalizeTemplate),
  });
}

const tpl = (over: Partial<ReturnType<typeof blankTemplate>> = {}) =>
  normalizeTemplate({ ...blankTemplate(), ...over });

describe("load", () => {
  it("normalizes missing arrays, snapshots baseline, sets loaded", () => {
    const s = loaded([{ jobIndex: 1 }]);
    expect(s.loaded).toBe(true);
    expect(s.templates[0].faces).toEqual([]);
    expect(s.baseline).toEqual(s.templates);
    expect(isDirty(s)).toBe(false);
  });
});

describe("template lifecycle", () => {
  it("addTemplate appends a blank template and selects it", () => {
    const s = editorReducer(loaded([tpl()]), { type: "addTemplate" });
    expect(s.templates).toHaveLength(2);
    expect(s.selectedIndex).toBe(1);
    expect(s.templates[1]).toEqual(blankTemplate());
  });

  it("duplicateTemplate deep-copies the selected template and selects the copy", () => {
    const s0 = loaded([tpl({ faces: [20000] })]);
    const s = editorReducer(s0, { type: "duplicateTemplate" });
    expect(s.templates).toHaveLength(2);
    expect(s.selectedIndex).toBe(1);
    expect(s.templates[1]).toEqual(s.templates[0]);
    // deep copy: mutating the copy's array must not touch the original
    s.templates[1].faces.push(21000);
    expect(s.templates[0].faces).toEqual([20000]);
  });

  it("removeTemplate selects the nearest remaining index and remaps picks", () => {
    let s = loaded([tpl(), tpl(), tpl()]);
    s = editorReducer(s, { type: "select", index: 2 });
    s = editorReducer(s, { type: "setPreviewPick", pick: "faceIdx", value: 3 });
    s = editorReducer(s, { type: "select", index: 1 });
    s = editorReducer(s, { type: "removeTemplate" }); // removes index 1
    expect(s.templates).toHaveLength(2);
    expect(s.selectedIndex).toBe(1); // nearest remaining
    // picks for old index 2 shifted down to index 1
    expect(picksFor(s, 1).faceIdx).toBe(3);
  });

  it("removing the last remaining template clamps selection to 0", () => {
    const s = editorReducer(loaded([tpl()]), { type: "removeTemplate" });
    expect(s.templates).toHaveLength(0);
    expect(s.selectedIndex).toBe(0);
  });
});

describe("edits", () => {
  it("setIdentity updates the selected template and marks dirty", () => {
    const s = editorReducer(loaded([tpl()]), {
      type: "setIdentity",
      field: "mapId",
      value: 100000000,
    });
    expect(s.templates[0].mapId).toBe(100000000);
    expect(isDirty(s)).toBe(true);
  });

  it("addPoolEntry appends and prevents double-add", () => {
    let s = editorReducer(loaded([tpl()]), {
      type: "addPoolEntry",
      pool: "faces",
      id: 20000,
    });
    s = editorReducer(s, { type: "addPoolEntry", pool: "faces", id: 20000 });
    expect(s.templates[0].faces).toEqual([20000]);
  });

  it("removePoolEntry clamps the matching preview pick", () => {
    let s = loaded([tpl({ faces: [20000, 20001] })]);
    s = editorReducer(s, { type: "setPreviewPick", pick: "faceIdx", value: 1 });
    s = editorReducer(s, {
      type: "removePoolEntry",
      pool: "faces",
      entryIndex: 1,
    });
    expect(s.templates[0].faces).toEqual([20000]);
    expect(picksFor(s, 0).faceIdx).toBe(0);
  });

  it("select never touches templates (free switching keeps edits)", () => {
    let s = loaded([tpl(), tpl()]);
    s = editorReducer(s, { type: "addPoolEntry", pool: "hairs", id: 30030 });
    s = editorReducer(s, { type: "select", index: 1 });
    expect(s.templates[0].hairs).toEqual([30030]);
    expect(isDirty(s)).toBe(true);
  });
});

describe("discard / savedOk", () => {
  it("discard restores baseline and resets picks", () => {
    let s = loaded([tpl()]);
    s = editorReducer(s, { type: "setPreviewPick", pick: "hairIdx", value: 2 });
    s = editorReducer(s, { type: "addPoolEntry", pool: "faces", id: 20000 });
    s = editorReducer(s, { type: "discard" });
    expect(isDirty(s)).toBe(false);
    expect(s.templates[0].faces).toEqual([]);
    expect(picksFor(s, 0)).toEqual(DEFAULT_PICKS);
  });

  it("savedOk re-baselines the working copy", () => {
    let s = loaded([tpl()]);
    s = editorReducer(s, { type: "addPoolEntry", pool: "faces", id: 20000 });
    s = editorReducer(s, { type: "savedOk" });
    expect(isDirty(s)).toBe(false);
    expect(s.baseline[0].faces).toEqual([20000]);
  });
});

describe("emptyPoolWarnings", () => {
  it("flags the four appearance pools the factory rejects when empty", () => {
    expect(emptyPoolWarnings(blankTemplate())).toEqual([
      "faces",
      "hairs",
      "hairColors",
      "skinColors",
    ]);
    expect(
      emptyPoolWarnings(
        tpl({ faces: [1], hairs: [1], hairColors: [0], skinColors: [0] }),
      ),
    ).toEqual([]);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `source ~/.nvm/nvm.sh && nvm use 22 >/dev/null 2>&1 && npm run test -- src/components/features/characters/templates/__tests__/editorState.test.ts`
Expected: FAIL — cannot resolve `../editorState`.

- [ ] **Step 3: Write the implementation**

`src/components/features/characters/templates/editorState.ts`:

```ts
import type { CharacterTemplate } from "@/types/models/template";

export type PoolKey =
  | "faces"
  | "hairs"
  | "hairColors"
  | "skinColors"
  | "tops"
  | "bottoms"
  | "shoes"
  | "weapons"
  | "items"
  | "skills";

export type AppearancePoolKey = "faces" | "hairs" | "hairColors" | "skinColors";
export type IdentityField = "jobIndex" | "subJobIndex" | "gender" | "mapId";

export interface PreviewPicks {
  faceIdx: number;
  hairIdx: number;
  hairColorIdx: number;
  skinIdx: number;
}

export const DEFAULT_PICKS: PreviewPicks = {
  faceIdx: 0,
  hairIdx: 0,
  hairColorIdx: 0,
  skinIdx: 0,
};

export const PICK_KEY_BY_POOL: Partial<Record<PoolKey, keyof PreviewPicks>> = {
  faces: "faceIdx",
  hairs: "hairIdx",
  hairColors: "hairColorIdx",
  skinColors: "skinIdx",
};

export interface EditorState {
  templates: CharacterTemplate[];
  baseline: CharacterTemplate[];
  selectedIndex: number;
  /** Per-template-index UI-only preview picks. Never persisted. */
  previewPicks: Record<number, PreviewPicks>;
  loaded: boolean;
}

export type EditorAction =
  | { type: "load"; templates: CharacterTemplate[] }
  | { type: "select"; index: number }
  | { type: "addTemplate" }
  | { type: "duplicateTemplate" }
  | { type: "removeTemplate" }
  | { type: "setIdentity"; field: IdentityField; value: number }
  | { type: "addPoolEntry"; pool: PoolKey; id: number }
  | { type: "removePoolEntry"; pool: PoolKey; entryIndex: number }
  | { type: "setPreviewPick"; pick: keyof PreviewPicks; value: number }
  | { type: "discard" }
  | { type: "savedOk" };

export function normalizeTemplate(
  raw: Partial<CharacterTemplate> | undefined,
): CharacterTemplate {
  return {
    jobIndex: raw?.jobIndex ?? 0,
    subJobIndex: raw?.subJobIndex ?? 0,
    gender: raw?.gender ?? 0,
    mapId: raw?.mapId ?? 0,
    faces: raw?.faces ?? [],
    hairs: raw?.hairs ?? [],
    hairColors: raw?.hairColors ?? [],
    skinColors: raw?.skinColors ?? [],
    tops: raw?.tops ?? [],
    bottoms: raw?.bottoms ?? [],
    shoes: raw?.shoes ?? [],
    weapons: raw?.weapons ?? [],
    items: raw?.items ?? [],
    skills: raw?.skills ?? [],
  };
}

export function blankTemplate(): CharacterTemplate {
  return normalizeTemplate(undefined);
}

export function cloneTemplate(t: CharacterTemplate): CharacterTemplate {
  return {
    ...t,
    faces: [...t.faces],
    hairs: [...t.hairs],
    hairColors: [...t.hairColors],
    skinColors: [...t.skinColors],
    tops: [...t.tops],
    bottoms: [...t.bottoms],
    shoes: [...t.shoes],
    weapons: [...t.weapons],
    items: [...t.items],
    skills: [...t.skills],
  };
}

export function initialEditorState(): EditorState {
  return {
    templates: [],
    baseline: [],
    selectedIndex: 0,
    previewPicks: {},
    loaded: false,
  };
}

function clampIndex(index: number, length: number): number {
  if (length <= 0) return 0;
  return Math.min(Math.max(index, 0), length - 1);
}

/** Shift previewPicks keys after a removal at `removed`. */
function remapPicksAfterRemove(
  picks: Record<number, PreviewPicks>,
  removed: number,
): Record<number, PreviewPicks> {
  const out: Record<number, PreviewPicks> = {};
  for (const [key, value] of Object.entries(picks)) {
    const i = Number(key);
    if (i === removed) continue;
    out[i > removed ? i - 1 : i] = value;
  }
  return out;
}

function updateSelected(
  state: EditorState,
  update: (t: CharacterTemplate) => CharacterTemplate,
): EditorState {
  const t = state.templates[state.selectedIndex];
  if (!t) return state;
  const templates = [...state.templates];
  templates[state.selectedIndex] = update(cloneTemplate(t));
  return { ...state, templates };
}

export function editorReducer(
  state: EditorState,
  action: EditorAction,
): EditorState {
  switch (action.type) {
    case "load": {
      const templates = action.templates.map(normalizeTemplate);
      return {
        templates,
        baseline: templates.map(cloneTemplate),
        selectedIndex: clampIndex(state.selectedIndex, templates.length),
        previewPicks: {},
        loaded: true,
      };
    }
    case "select":
      return {
        ...state,
        selectedIndex: clampIndex(action.index, state.templates.length),
      };
    case "addTemplate": {
      const templates = [...state.templates, blankTemplate()];
      return { ...state, templates, selectedIndex: templates.length - 1 };
    }
    case "duplicateTemplate": {
      const src = state.templates[state.selectedIndex];
      if (!src) return state;
      const templates = [...state.templates, cloneTemplate(src)];
      const previewPicks = { ...state.previewPicks };
      const srcPicks = state.previewPicks[state.selectedIndex];
      if (srcPicks) previewPicks[templates.length - 1] = { ...srcPicks };
      return {
        ...state,
        templates,
        previewPicks,
        selectedIndex: templates.length - 1,
      };
    }
    case "removeTemplate": {
      if (!state.templates[state.selectedIndex]) return state;
      const templates = state.templates.filter(
        (_, i) => i !== state.selectedIndex,
      );
      return {
        ...state,
        templates,
        previewPicks: remapPicksAfterRemove(
          state.previewPicks,
          state.selectedIndex,
        ),
        selectedIndex: clampIndex(state.selectedIndex, templates.length),
      };
    }
    case "setIdentity":
      return updateSelected(state, (t) => ({
        ...t,
        [action.field]: action.value,
      }));
    case "addPoolEntry":
      return updateSelected(state, (t) =>
        t[action.pool].includes(action.id)
          ? t
          : { ...t, [action.pool]: [...t[action.pool], action.id] },
      );
    case "removePoolEntry": {
      const next = updateSelected(state, (t) => ({
        ...t,
        [action.pool]: t[action.pool].filter(
          (_, i) => i !== action.entryIndex,
        ),
      }));
      const pickKey = PICK_KEY_BY_POOL[action.pool];
      if (!pickKey) return next;
      const poolLen =
        next.templates[next.selectedIndex]?.[action.pool].length ?? 0;
      const picks = picksFor(next, next.selectedIndex);
      return {
        ...next,
        previewPicks: {
          ...next.previewPicks,
          [next.selectedIndex]: {
            ...picks,
            [pickKey]: clampIndex(picks[pickKey], poolLen),
          },
        },
      };
    }
    case "setPreviewPick":
      return {
        ...state,
        previewPicks: {
          ...state.previewPicks,
          [state.selectedIndex]: {
            ...picksFor(state, state.selectedIndex),
            [action.pick]: action.value,
          },
        },
      };
    case "discard":
      return {
        ...state,
        templates: state.baseline.map(cloneTemplate),
        previewPicks: {},
        selectedIndex: clampIndex(state.selectedIndex, state.baseline.length),
      };
    case "savedOk":
      return { ...state, baseline: state.templates.map(cloneTemplate) };
  }
}

/** Dirty = working copy differs from baseline. Arrays are tiny; deep compare is fine. */
export function isDirty(state: EditorState): boolean {
  return JSON.stringify(state.templates) !== JSON.stringify(state.baseline);
}

export function picksFor(state: EditorState, index: number): PreviewPicks {
  return state.previewPicks[index] ?? DEFAULT_PICKS;
}

/**
 * Empty appearance pools the factory rejects creations against
 * (mirrors templatesService.validateTemplateConsistency's per-template checks).
 */
export function emptyPoolWarnings(t: CharacterTemplate): AppearancePoolKey[] {
  return (["faces", "hairs", "hairColors", "skinColors"] as const).filter(
    (k) => t[k].length === 0,
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Same command as Step 2. Expected: PASS (12 tests).

- [ ] **Step 5: Commit**

```bash
git add src/components/features/characters/templates/editorState.ts src/components/features/characters/templates/__tests__/editorState.test.ts
git commit -m "feat(task-177): editor state reducer"
```

### Task 3: Loadout builders (`previewLoadout.ts`) + `isFemaleCosmeticId`

**Files:**
- Create: `src/components/features/characters/templates/previewLoadout.ts`
- Modify: `src/services/api/characterRender.service.ts:58-62` (extract `isFemaleCosmeticId`)
- Test: `src/components/features/characters/templates/__tests__/previewLoadout.test.ts`

**Interfaces:**
- Consumes: `CharacterTemplate`, `PreviewPicks` (Task 2), `CharacterLoadout` from `@/services/api/characterRender.service`.
- Produces: `RENDER_DEFAULT_SKIN = 0`, `RENDER_DEFAULT_HAIR = 30030`, `RENDER_DEFAULT_FACE = 20000`, `EQUIP_SLOT_BY_POOL = { tops: "-5", bottoms: "-6", shoes: "-7", weapons: "-11" }`, `type EquipmentPoolKey = "tops"|"bottoms"|"shoes"|"weapons"`, `buildPreviewLoadout(t, picks): CharacterLoadout`, `buildVariantLoadout(t, picks, dimension: AppearancePoolKey, candidateId): CharacterLoadout`, and `isFemaleCosmeticId(id: number): boolean` exported from `characterRender.service.ts`. PreviewCard (Task 14), AppearancePoolSection (Task 11), and AppearanceBrowserDialog (Task 12) consume these.

- [ ] **Step 1: Write the failing test**

`src/components/features/characters/templates/__tests__/previewLoadout.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import { isFemaleCosmeticId } from "@/services/api/characterRender.service";
import { blankTemplate, normalizeTemplate, DEFAULT_PICKS } from "../editorState";
import {
  buildPreviewLoadout,
  buildVariantLoadout,
  RENDER_DEFAULT_FACE,
  RENDER_DEFAULT_HAIR,
  RENDER_DEFAULT_SKIN,
  EQUIP_SLOT_BY_POOL,
} from "../previewLoadout";

const tpl = normalizeTemplate({
  gender: 1,
  faces: [21000, 21001],
  hairs: [31000, 31030],
  hairColors: [0, 7],
  skinColors: [0, 3],
  tops: [1041002],
  bottoms: [1061002],
  shoes: [1072001],
  weapons: [1302000],
});

describe("buildPreviewLoadout", () => {
  it("uses picked pool entries; hair = base hair + color digit", () => {
    const lo = buildPreviewLoadout(tpl, {
      faceIdx: 1,
      hairIdx: 1,
      hairColorIdx: 1,
      skinIdx: 1,
    });
    expect(lo.face).toBe(21001);
    expect(lo.hair).toBe(31030 + 7);
    expect(lo.skin).toBe(3);
    expect(lo.gender).toBe(1);
  });

  it("maps first-of-pool equipment onto the render slots", () => {
    const lo = buildPreviewLoadout(tpl, DEFAULT_PICKS);
    expect(lo.equipment).toEqual({
      [EQUIP_SLOT_BY_POOL.tops]: 1041002,
      [EQUIP_SLOT_BY_POOL.bottoms]: 1061002,
      [EQUIP_SLOT_BY_POOL.shoes]: 1072001,
      [EQUIP_SLOT_BY_POOL.weapons]: 1302000,
    });
  });

  it("falls back to render defaults for empty pools (fresh + New template)", () => {
    const lo = buildPreviewLoadout(blankTemplate(), DEFAULT_PICKS);
    expect(lo).toEqual({
      skin: RENDER_DEFAULT_SKIN,
      hair: RENDER_DEFAULT_HAIR,
      face: RENDER_DEFAULT_FACE,
      equipment: {},
      gender: 0,
    });
  });

  it("clamps out-of-range picks instead of reading past the pool", () => {
    const lo = buildPreviewLoadout(tpl, {
      faceIdx: 9,
      hairIdx: 9,
      hairColorIdx: 9,
      skinIdx: 9,
    });
    expect(lo.face).toBe(21001);
  });
});

describe("buildVariantLoadout", () => {
  it("substitutes only the varied dimension", () => {
    expect(buildVariantLoadout(tpl, DEFAULT_PICKS, "faces", 22000).face).toBe(
      22000,
    );
    // hair candidate keeps the picked color digit (pick 0 → digit 0)
    expect(buildVariantLoadout(tpl, DEFAULT_PICKS, "hairs", 32000).hair).toBe(
      32000,
    );
    // hairColor candidate applies to the picked base hair
    expect(
      buildVariantLoadout(tpl, DEFAULT_PICKS, "hairColors", 5).hair,
    ).toBe(31000 + 5);
    expect(
      buildVariantLoadout(tpl, DEFAULT_PICKS, "skinColors", 2).skin,
    ).toBe(2);
  });
});

describe("isFemaleCosmeticId", () => {
  it("implements the v83 (id/1000)%10 === 1 convention", () => {
    expect(isFemaleCosmeticId(21000)).toBe(true); // female face
    expect(isFemaleCosmeticId(20000)).toBe(false); // male face
    expect(isFemaleCosmeticId(31000)).toBe(true); // female hair
    expect(isFemaleCosmeticId(30030)).toBe(false); // male hair
    expect(isFemaleCosmeticId(0)).toBe(false);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `source ~/.nvm/nvm.sh && nvm use 22 >/dev/null 2>&1 && npm run test -- src/components/features/characters/templates/__tests__/previewLoadout.test.ts`
Expected: FAIL — `isFemaleCosmeticId` not exported / `../previewLoadout` unresolved.

- [ ] **Step 3: Extract `isFemaleCosmeticId` in `characterRender.service.ts`**

Replace the body of `resolveGender` (`src/services/api/characterRender.service.ts:58-62`) so the convention lives in one exported helper:

```ts
/**
 * v83 id convention shared by faces and hairs: (id/1000)%10 === 1 ⇒ female.
 * Same rule the Go service's ResolveGender applies to faces.
 */
export function isFemaleCosmeticId(id: number): boolean {
  return id > 0 && Math.floor(id / 1000) % 10 === 1;
}

/**
 * Mirror of the Go service's ResolveGender. An explicit 0/1 wins; otherwise
 * infer from the face id via the v83 convention (face/1000)%10 === 1 ⇒ female.
 * A non-positive / unknown face resolves to male (0).
 */
export function resolveGender(gender: number | undefined, face: number): 0 | 1 {
  if (gender === 0 || gender === 1) return gender;
  return isFemaleCosmeticId(face) ? 1 : 0;
}
```

- [ ] **Step 4: Write `previewLoadout.ts`**

`src/components/features/characters/templates/previewLoadout.ts`:

```ts
import type { CharacterTemplate } from "@/types/models/template";
import type { CharacterLoadout } from "@/services/api/characterRender.service";
import type { AppearancePoolKey, PreviewPicks } from "./editorState";

// Render defaults for empty pools — the task-078 "default clothing" posture:
// a fresh "+ New" template still previews a body; the empty-pool warning is
// the operator's signal, not a broken image.
export const RENDER_DEFAULT_SKIN = 0;
export const RENDER_DEFAULT_HAIR = 30030;
export const RENDER_DEFAULT_FACE = 20000;

// Atlas-canonical render slot ids (EquipmentPanel SLOT_LAYOUT).
export const EQUIP_SLOT_BY_POOL = {
  tops: "-5",
  bottoms: "-6",
  shoes: "-7",
  weapons: "-11",
} as const;

export type EquipmentPoolKey = keyof typeof EQUIP_SLOT_BY_POOL;

function at(pool: number[], idx: number): number | undefined {
  if (pool.length === 0) return undefined;
  return pool[Math.min(Math.max(idx, 0), pool.length - 1)];
}

export function buildPreviewLoadout(
  t: CharacterTemplate,
  picks: PreviewPicks,
): CharacterLoadout {
  const baseHair = at(t.hairs, picks.hairIdx) ?? RENDER_DEFAULT_HAIR;
  const colorDigit = at(t.hairColors, picks.hairColorIdx) ?? 0;
  const equipment: Record<string, number> = {};
  for (const pool of Object.keys(EQUIP_SLOT_BY_POOL) as EquipmentPoolKey[]) {
    const first = t[pool][0];
    if (first !== undefined) equipment[EQUIP_SLOT_BY_POOL[pool]] = first;
  }
  return {
    skin: at(t.skinColors, picks.skinIdx) ?? RENDER_DEFAULT_SKIN,
    hair: baseHair + colorDigit,
    face: at(t.faces, picks.faceIdx) ?? RENDER_DEFAULT_FACE,
    equipment,
    gender: t.gender,
  };
}

/**
 * Loadout for a thumbnail/browser candidate: current preview picks with one
 * appearance dimension substituted (v83 convention: rendered hair id =
 * base hair id + color digit).
 */
export function buildVariantLoadout(
  t: CharacterTemplate,
  picks: PreviewPicks,
  dimension: AppearancePoolKey,
  candidateId: number,
): CharacterLoadout {
  const base = buildPreviewLoadout(t, picks);
  switch (dimension) {
    case "faces":
      return { ...base, face: candidateId };
    case "hairs":
      return {
        ...base,
        hair: candidateId + (at(t.hairColors, picks.hairColorIdx) ?? 0),
      };
    case "hairColors":
      return {
        ...base,
        hair: (at(t.hairs, picks.hairIdx) ?? RENDER_DEFAULT_HAIR) + candidateId,
      };
    case "skinColors":
      return { ...base, skin: candidateId };
  }
}
```

- [ ] **Step 5: Run tests to verify they pass**

Same command as Step 2, plus the existing render-service suite must stay green:
`npm run test -- src/services/api/__tests__` — Expected: PASS, no regressions in `resolveGender` tests.

- [ ] **Step 6: Commit**

```bash
git add src/components/features/characters/templates/previewLoadout.ts src/components/features/characters/templates/__tests__/previewLoadout.test.ts src/services/api/characterRender.service.ts
git commit -m "feat(task-177): preview loadout builders + isFemaleCosmeticId"
```

### Task 4: Cosmetics enumeration (`cosmetics.service.ts` + `useCosmetics.ts`)

**Files:**
- Create: `src/services/api/cosmetics.service.ts`
- Create: `src/lib/hooks/api/useCosmetics.ts`
- Test: `src/services/api/__tests__/cosmetics.service.test.ts`

**Interfaces:**
- Consumes: `fetchAll` from `@/services/api/pagination`; `useTenant` from `@/context/tenant-context`.
- Produces: `cosmeticsService.getAllFaceIds(): Promise<number[]>`, `cosmeticsService.getAllHairIds(): Promise<number[]>` (sorted ascending numeric ids); hooks `useFaceIds(): UseQueryResult<number[], Error>`, `useHairIds(): UseQueryResult<number[], Error>`; `cosmeticsKeys` query-key factory. AppearanceBrowserDialog (Task 12) consumes the hooks.

- [ ] **Step 1: Write the failing test**

`src/services/api/__tests__/cosmetics.service.test.ts`:

```ts
import { describe, it, expect, vi, beforeEach } from "vitest";

const fetchAllMock = vi.fn();
vi.mock("@/services/api/pagination", () => ({
  fetchAll: (...a: unknown[]) => fetchAllMock(...a),
}));

import { cosmeticsService } from "@/services/api/cosmetics.service";

beforeEach(() => fetchAllMock.mockReset());

describe("cosmeticsService", () => {
  it("enumerates faces via fetchAll and returns sorted numeric ids", async () => {
    fetchAllMock.mockResolvedValue([
      { id: "20001", attributes: { cash: false } },
      { id: "20000", attributes: { cash: false } },
      { id: "21000", attributes: { cash: true } },
    ]);
    await expect(cosmeticsService.getAllFaceIds()).resolves.toEqual([
      20000, 20001, 21000,
    ]);
    expect(fetchAllMock).toHaveBeenCalledWith("/api/data/cosmetics/faces");
  });

  it("enumerates hairs and drops non-numeric ids", async () => {
    fetchAllMock.mockResolvedValue([
      { id: "30030", attributes: { cash: false } },
      { id: "bogus", attributes: { cash: false } },
    ]);
    await expect(cosmeticsService.getAllHairIds()).resolves.toEqual([30030]);
    expect(fetchAllMock).toHaveBeenCalledWith("/api/data/cosmetics/hairs");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `source ~/.nvm/nvm.sh && nvm use 22 >/dev/null 2>&1 && npm run test -- src/services/api/__tests__/cosmetics.service.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Write the service**

`src/services/api/cosmetics.service.ts`:

```ts
import { fetchAll } from "@/services/api/pagination";

const BASE_PATH = "/api/data/cosmetics";

// JSON:API row shape of /api/data/cosmetics/faces|hairs (live-verified in the
// PRD: 536 faces / 1520 hairs, attributes carry only {cash}).
interface CosmeticData {
  id: string;
  attributes: { cash: boolean };
}

async function getAllIds(kind: "faces" | "hairs"): Promise<number[]> {
  const rows = await fetchAll<CosmeticData>(`${BASE_PATH}/${kind}`);
  return rows
    .map((row) => Number.parseInt(row.id, 10))
    .filter((id) => Number.isFinite(id))
    .sort((a, b) => a - b);
}

export const cosmeticsService = {
  getAllFaceIds: (): Promise<number[]> => getAllIds("faces"),
  getAllHairIds: (): Promise<number[]> => getAllIds("hairs"),
};
```

- [ ] **Step 4: Write the hooks**

`src/lib/hooks/api/useCosmetics.ts`:

```ts
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { cosmeticsService } from "@/services/api/cosmetics.service";
import { useTenant } from "@/context/tenant-context";

export const cosmeticsKeys = {
  all: ["cosmetics"] as const,
  faces: () => [...cosmeticsKeys.all, "faces"] as const,
  hairs: () => [...cosmeticsKeys.all, "hairs"] as const,
};

// WZ data changes only with re-ingest; TenantProvider clears all caches on
// tenant switch, so a long staleTime is safe.
const WZ_STALE_TIME = 60 * 60 * 1000;

export function useFaceIds(): UseQueryResult<number[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: cosmeticsKeys.faces(),
    queryFn: () => cosmeticsService.getAllFaceIds(),
    enabled: !!activeTenant,
    staleTime: WZ_STALE_TIME,
    gcTime: WZ_STALE_TIME,
  });
}

export function useHairIds(): UseQueryResult<number[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: cosmeticsKeys.hairs(),
    queryFn: () => cosmeticsService.getAllHairIds(),
    enabled: !!activeTenant,
    staleTime: WZ_STALE_TIME,
    gcTime: WZ_STALE_TIME,
  });
}
```

- [ ] **Step 5: Run test to verify it passes**

Same command as Step 2. Expected: PASS (2 tests).

- [ ] **Step 6: Commit**

```bash
git add src/services/api/cosmetics.service.ts src/lib/hooks/api/useCosmetics.ts src/services/api/__tests__/cosmetics.service.test.ts
git commit -m "feat(task-177): cosmetics enumeration service + hooks"
```

### Task 5: Batched item names (`useItemNames.ts`)

**Files:**
- Create: `src/lib/hooks/api/useItemNames.ts`
- Test: `src/lib/hooks/api/__tests__/useItemNames.test.tsx`

**Interfaces:**
- Consumes: `itemStringKeys` from `@/lib/hooks/api/useItemStrings` (cache-key sharing with `useItemName`), `itemStringsService` from `@/services/api/item-strings.service`, `useTenant`.
- Produces: `useItemNames(ids: number[]): Record<number, string | undefined>` — `undefined` while loading or on error (callers render placeholder + id). AppearanceBrowserDialog (Task 12) consumes it.

- [ ] **Step 1: Write the failing test**

`src/lib/hooks/api/__tests__/useItemNames.test.tsx`:

```tsx
import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";

const getItemStringMock = vi.fn();
vi.mock("@/services/api/item-strings.service", () => ({
  itemStringsService: {
    getItemString: (...a: unknown[]) => getItemStringMock(...a),
  },
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

import { useItemNames } from "../useItemNames";
import { itemStringKeys } from "../useItemStrings";

function wrapper(client: QueryClient) {
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

beforeEach(() => getItemStringMock.mockReset());

describe("useItemNames", () => {
  it("resolves names per id and keys the shared useItemName cache", async () => {
    getItemStringMock.mockImplementation((id: string) =>
      Promise.resolve({ attributes: { name: `Item ${id}` } }),
    );
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const { result } = renderHook(() => useItemNames([20000, 30030]), {
      wrapper: wrapper(client),
    });
    await waitFor(() =>
      expect(result.current).toEqual({
        20000: "Item 20000",
        30030: "Item 30030",
      }),
    );
    // cache entries share useItemName's key shape → lookups merge across UI
    expect(client.getQueryData(itemStringKeys.byId("20000"))).toBe(
      "Item 20000",
    );
  });

  it("returns undefined for ids whose lookup fails", async () => {
    getItemStringMock.mockRejectedValue(new Error("404"));
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const { result } = renderHook(() => useItemNames([99999]), {
      wrapper: wrapper(client),
    });
    await waitFor(() => expect(getItemStringMock).toHaveBeenCalled());
    expect(result.current[99999]).toBeUndefined();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `source ~/.nvm/nvm.sh && nvm use 22 >/dev/null 2>&1 && npm run test -- src/lib/hooks/api/__tests__/useItemNames.test.tsx`
Expected: FAIL — module not found.

- [ ] **Step 3: Write the implementation**

`src/lib/hooks/api/useItemNames.ts`:

```ts
import { useQueries } from "@tanstack/react-query";
import { itemStringsService } from "@/services/api/item-strings.service";
import { itemStringKeys } from "@/lib/hooks/api/useItemStrings";
import { useTenant } from "@/context/tenant-context";

/**
 * Batched per-id item-name resolution. Reuses useItemName's query keys so
 * individual lookups cache-share across the browser grid, pool rows, and
 * future visits. `undefined` = still loading or failed (caller degrades to
 * placeholder + numeric id).
 */
export function useItemNames(
  ids: number[],
): Record<number, string | undefined> {
  const { activeTenant } = useTenant();
  const results = useQueries({
    queries: ids.map((id) => ({
      queryKey: itemStringKeys.byId(String(id)),
      queryFn: async () => {
        const item = await itemStringsService.getItemString(String(id));
        return item.attributes.name;
      },
      enabled: !!activeTenant,
      staleTime: 10 * 60 * 1000,
      gcTime: 30 * 60 * 1000,
      retry: 1,
    })),
  });
  const names: Record<number, string | undefined> = {};
  ids.forEach((id, i) => {
    names[id] = results[i]?.data;
  });
  return names;
}
```

- [ ] **Step 4: Run test to verify it passes**

Same command as Step 2. Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add src/lib/hooks/api/useItemNames.ts src/lib/hooks/api/__tests__/useItemNames.test.tsx
git commit -m "feat(task-177): batched item-name hook"
```

### Task 6: Popover primitive + pool search configs + `ItemSearchCombobox`

**Files:**
- Modify: `package.json` (add `"@radix-ui/react-popover": "^1.1.6"` to dependencies via `npm install @radix-ui/react-popover`)
- Create: `src/components/ui/popover.tsx`
- Create: `src/components/features/characters/templates/poolSearchConfig.ts`
- Create: `src/components/features/characters/templates/ItemSearchCombobox.tsx`
- Test: `src/components/features/characters/templates/__tests__/ItemSearchCombobox.test.tsx`

**Interfaces:**
- Consumes: `itemsService.searchItems` (`@/services/api/items.service`), `getAssetIconUrl`, `useTenant`.
- Produces:
  - `type SearchPoolKey = "tops" | "bottoms" | "shoes" | "weapons" | "items"`.
  - `WEAPON_SUBCATEGORIES: ReadonlySet<string>`, `POOL_SEARCH_CONFIGS: Record<SearchPoolKey, PoolSearchConfig>` where `PoolSearchConfig = { compartment?: "equipment"; subcategory?: string; clientSubcategories?: ReadonlySet<string> }`.
  - `<ItemSearchCombobox poolKey existingIds onAdd triggerLabel? debounceMs? />` with props `{ poolKey: SearchPoolKey; existingIds: number[]; onAdd: (id: number) => void; triggerLabel?: string; debounceMs?: number }`. EquipmentPoolSection (Task 13) and StartingKitSection (Task 13) consume it.

- [ ] **Step 1: Install the dep and add the shadcn Popover primitive**

Run: `source ~/.nvm/nvm.sh && nvm use 22 >/dev/null 2>&1 && npm install @radix-ui/react-popover`

`src/components/ui/popover.tsx` (standard shadcn, matching the local dialog.tsx style):

```tsx
import * as React from "react";
import * as PopoverPrimitive from "@radix-ui/react-popover";
import { cn } from "@/lib/utils";

const Popover = PopoverPrimitive.Root;
const PopoverTrigger = PopoverPrimitive.Trigger;
const PopoverAnchor = PopoverPrimitive.Anchor;

const PopoverContent = React.forwardRef<
  React.ElementRef<typeof PopoverPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof PopoverPrimitive.Content>
>(({ className, align = "center", sideOffset = 4, ...props }, ref) => (
  <PopoverPrimitive.Portal>
    <PopoverPrimitive.Content
      ref={ref}
      align={align}
      sideOffset={sideOffset}
      className={cn(
        "z-50 w-72 rounded-md border bg-popover p-4 text-popover-foreground shadow-md outline-none",
        "data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0",
        className,
      )}
      {...props}
    />
  </PopoverPrimitive.Portal>
));
PopoverContent.displayName = PopoverPrimitive.Content.displayName;

export { Popover, PopoverTrigger, PopoverContent, PopoverAnchor };
```

- [ ] **Step 2: Write `poolSearchConfig.ts`**

```ts
// Search-filter strategy per pool (design D4). parseFilters in atlas-data
// item/filter.go accepts exactly ONE filter[subcategory] token, so pools that
// span multiple subcategories filter client-side on the returned rows'
// subcategory field instead.

export type SearchPoolKey = "tops" | "bottoms" | "shoes" | "weapons" | "items";

// The 16 weapon tokens registered in atlas-data item/filter.go:55-60
// (pet-equip deliberately excluded — not a starting-weapon candidate).
export const WEAPON_SUBCATEGORIES: ReadonlySet<string> = new Set([
  "one-handed-sword",
  "one-handed-axe",
  "one-handed-mace",
  "dagger",
  "wand",
  "staff",
  "two-handed-sword",
  "two-handed-axe",
  "two-handed-mace",
  "spear",
  "polearm",
  "bow",
  "crossbow",
  "claw",
  "knuckle",
  "gun",
]);

export interface PoolSearchConfig {
  compartment?: "equipment";
  /** Server-side single-token filter[subcategory]. */
  subcategory?: string;
  /** Client-side post-filter over result rows' subcategory. */
  clientSubcategories?: ReadonlySet<string>;
}

export const POOL_SEARCH_CONFIGS: Record<SearchPoolKey, PoolSearchConfig> = {
  // Aran's 1042167 is an overall — the tops pool legitimately contains overalls.
  tops: { compartment: "equipment", clientSubcategories: new Set(["top", "overall"]) },
  bottoms: { compartment: "equipment", subcategory: "bottom" },
  shoes: { compartment: "equipment", subcategory: "shoes" },
  weapons: { compartment: "equipment", clientSubcategories: WEAPON_SUBCATEGORIES },
  // Starting-kit items search all compartments.
  items: {},
};
```

- [ ] **Step 3: Write the failing test**

`src/components/features/characters/templates/__tests__/ItemSearchCombobox.test.tsx`:

```tsx
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const searchItemsMock = vi.fn();
vi.mock("@/services/api/items.service", () => ({
  itemsService: { searchItems: (...a: unknown[]) => searchItemsMock(...a) },
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

import { ItemSearchCombobox } from "../ItemSearchCombobox";

function renderBox(
  props: Partial<React.ComponentProps<typeof ItemSearchCombobox>> = {},
) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <ItemSearchCombobox
        poolKey="weapons"
        existingIds={[]}
        onAdd={vi.fn()}
        debounceMs={0}
        {...props}
      />
    </QueryClientProvider>,
  );
}

const page = (items: unknown[]) => ({
  items,
  total: items.length,
  pageNumber: 1,
  pageSize: 50,
  lastPage: 1,
});

beforeEach(() => searchItemsMock.mockReset());

describe("ItemSearchCombobox", () => {
  it("searches with the pool's server filters and adds a clicked row", async () => {
    searchItemsMock.mockResolvedValue(
      page([
        { id: "1302000", name: "Sword", compartment: "equipment", subcategory: "one-handed-sword", type: "Equipment" },
      ]),
    );
    const onAdd = vi.fn();
    renderBox({ poolKey: "bottoms", onAdd });
    await userEvent.click(screen.getByRole("button", { name: /add/i }));
    await userEvent.type(screen.getByRole("textbox"), "pants");
    await waitFor(() =>
      expect(searchItemsMock).toHaveBeenCalledWith(
        expect.objectContaining({
          q: "pants",
          compartment: "equipment",
          subcategory: "bottom",
          pageNumber: 1,
          pageSize: 50,
        }),
      ),
    );
    await userEvent.click(await screen.findByRole("option", { name: /Sword/ }));
    expect(onAdd).toHaveBeenCalledWith(1302000);
  });

  it("client-filters weapons to the 16 weapon subcategories", async () => {
    searchItemsMock.mockResolvedValue(
      page([
        { id: "1302000", name: "Sword", compartment: "equipment", subcategory: "one-handed-sword", type: "Equipment" },
        { id: "1802000", name: "Pet Leash", compartment: "equipment", subcategory: "pet-equip", type: "Equipment" },
      ]),
    );
    renderBox();
    await userEvent.click(screen.getByRole("button", { name: /add/i }));
    await userEvent.type(screen.getByRole("textbox"), "s");
    expect(await screen.findByRole("option", { name: /Sword/ })).toBeInTheDocument();
    expect(screen.queryByText(/Pet Leash/)).not.toBeInTheDocument();
  });

  it("offers the manual Use id fallback for numeric input", async () => {
    searchItemsMock.mockResolvedValue(page([]));
    const onAdd = vi.fn();
    renderBox({ onAdd });
    await userEvent.click(screen.getByRole("button", { name: /add/i }));
    await userEvent.type(screen.getByRole("textbox"), "1402001");
    await userEvent.click(
      await screen.findByRole("option", { name: /use id 1402001/i }),
    );
    expect(onAdd).toHaveBeenCalledWith(1402001);
  });

  it("marks rows already in the pool and does not re-add them", async () => {
    searchItemsMock.mockResolvedValue(
      page([
        { id: "1302000", name: "Sword", compartment: "equipment", subcategory: "one-handed-sword", type: "Equipment" },
      ]),
    );
    const onAdd = vi.fn();
    renderBox({ existingIds: [1302000], onAdd });
    await userEvent.click(screen.getByRole("button", { name: /add/i }));
    await userEvent.type(screen.getByRole("textbox"), "s");
    const row = await screen.findByRole("option", { name: /Sword/ });
    expect(row).toHaveAttribute("aria-disabled", "true");
    await userEvent.click(row);
    expect(onAdd).not.toHaveBeenCalled();
  });
});
```

- [ ] **Step 4: Run test to verify it fails**

Run: `source ~/.nvm/nvm.sh && nvm use 22 >/dev/null 2>&1 && npm run test -- src/components/features/characters/templates/__tests__/ItemSearchCombobox.test.tsx`
Expected: FAIL — `../ItemSearchCombobox` unresolved.

- [ ] **Step 5: Write `ItemSearchCombobox.tsx`**

```tsx
import { useEffect, useMemo, useState } from "react";
import { useQuery, keepPreviousData } from "@tanstack/react-query";
import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { itemsService } from "@/services/api/items.service";
import type { ItemSearchFilters } from "@/services/api/items.service";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { useTenant } from "@/context/tenant-context";
import {
  POOL_SEARCH_CONFIGS,
  type SearchPoolKey,
} from "./poolSearchConfig";

interface ItemSearchComboboxProps {
  poolKey: SearchPoolKey;
  existingIds: number[];
  onAdd: (id: number) => void;
  triggerLabel?: string;
  /** Test hook: pass 0 to disable debouncing. */
  debounceMs?: number;
}

const PAGE_SIZE = 50;

export function ItemSearchCombobox({
  poolKey,
  existingIds,
  onAdd,
  triggerLabel = "Add",
  debounceMs = 300,
}: ItemSearchComboboxProps) {
  const { activeTenant } = useTenant();
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [debounced, setDebounced] = useState("");
  const [page, setPage] = useState(1);

  useEffect(() => {
    if (debounceMs === 0) {
      setDebounced(search);
      setPage(1);
      return;
    }
    const handle = setTimeout(() => {
      setDebounced(search);
      setPage(1);
    }, debounceMs);
    return () => clearTimeout(handle);
  }, [search, debounceMs]);

  const cfg = POOL_SEARCH_CONFIGS[poolKey];

  const filters: ItemSearchFilters = {
    pageNumber: page,
    pageSize: PAGE_SIZE,
    ...(debounced ? { q: debounced } : {}),
    ...(cfg.compartment ? { compartment: cfg.compartment } : {}),
    ...(cfg.subcategory ? { subcategory: cfg.subcategory } : {}),
  };

  const query = useQuery({
    queryKey: ["item-search", poolKey, debounced, page],
    queryFn: () => itemsService.searchItems(filters),
    enabled: open && !!activeTenant && debounced.trim().length > 0,
    placeholderData: keepPreviousData,
    staleTime: 10 * 60 * 1000,
  });

  const rows = useMemo(() => {
    const items = query.data?.items ?? [];
    return cfg.clientSubcategories
      ? items.filter((r) => cfg.clientSubcategories!.has(r.subcategory))
      : items;
  }, [query.data, cfg.clientSubcategories]);

  const manualId = /^\d+$/.test(search.trim())
    ? Number(search.trim())
    : undefined;
  const hasMore = (query.data?.lastPage ?? 1) > page;

  const handleAdd = (id: number) => {
    if (existingIds.includes(id)) return;
    onAdd(id);
    setOpen(false);
    setSearch("");
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button type="button" variant="outline" size="sm">
          <Plus className="size-4" /> {triggerLabel}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-80 p-2" align="start">
        <Input
          autoFocus
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search by name or enter an id…"
        />
        <ul role="listbox" className="mt-2 max-h-64 space-y-0.5 overflow-y-auto">
          {rows.map((row) => {
            const id = Number(row.id);
            const inPool = existingIds.includes(id);
            return (
              <li
                key={row.id}
                role="option"
                aria-selected={false}
                aria-disabled={inPool}
                tabIndex={inPool ? -1 : 0}
                onClick={() => !inPool && handleAdd(id)}
                onKeyDown={(e) => {
                  if ((e.key === "Enter" || e.key === " ") && !inPool) {
                    e.preventDefault();
                    handleAdd(id);
                  }
                }}
                className={
                  inPool
                    ? "flex cursor-not-allowed items-center gap-2 rounded px-2 py-1 opacity-50"
                    : "flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-accent focus-visible:bg-accent"
                }
              >
                {activeTenant && (
                  <img
                    src={getAssetIconUrl(
                      activeTenant.id,
                      activeTenant.attributes.region,
                      activeTenant.attributes.majorVersion,
                      activeTenant.attributes.minorVersion,
                      "item",
                      id,
                    )}
                    alt=""
                    width={24}
                    height={24}
                    loading="lazy"
                    className="[image-rendering:pixelated]"
                    onError={(e) => {
                      (e.target as HTMLImageElement).style.visibility =
                        "hidden";
                    }}
                  />
                )}
                <span className="flex-1 truncate text-sm">{row.name}</span>
                <span className="font-mono text-xs text-muted-foreground">
                  {row.id}
                </span>
                {inPool && (
                  <span className="text-xs text-muted-foreground">Added</span>
                )}
              </li>
            );
          })}
          {manualId !== undefined && (
            <li
              role="option"
              aria-selected={false}
              tabIndex={0}
              onClick={() => handleAdd(manualId)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  handleAdd(manualId);
                }
              }}
              className="cursor-pointer rounded px-2 py-1 text-sm hover:bg-accent focus-visible:bg-accent"
            >
              Use id {manualId}
            </li>
          )}
          {query.isLoading && debounced && (
            <li className="px-2 py-1 text-sm text-muted-foreground">
              Searching…
            </li>
          )}
          {!query.isLoading &&
            debounced &&
            rows.length === 0 &&
            manualId === undefined && (
              <li className="px-2 py-1 text-sm text-muted-foreground">
                No matches.
              </li>
            )}
        </ul>
        {hasMore && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="mt-1 w-full"
            onClick={() => setPage((p) => p + 1)}
          >
            Load more
          </Button>
        )}
      </PopoverContent>
    </Popover>
  );
}
```

Note: "Load more" advances the page (with `keepPreviousData` the previous rows stay rendered while the next page loads). This matches the design's load-more-not-numbered-pagination decision; if reviewers prefer accumulation across pages, accumulate rows in a `useState` keyed by `debounced` — either satisfies FR-6.2.

- [ ] **Step 6: Run test to verify it passes**

Same command as Step 4. Expected: PASS (4 tests).

- [ ] **Step 7: Commit**

```bash
git add package.json package-lock.json src/components/ui/popover.tsx src/components/features/characters/templates/poolSearchConfig.ts src/components/features/characters/templates/ItemSearchCombobox.tsx src/components/features/characters/templates/__tests__/ItemSearchCombobox.test.tsx
git commit -m "feat(task-177): popover primitive + item search combobox"
```

### Task 7: `MapPicker`

**Files:**
- Create: `src/components/features/characters/templates/MapPicker.tsx`
- Test: `src/components/features/characters/templates/__tests__/MapPicker.test.tsx`

**Interfaces:**
- Consumes: `useMapsByName`, `useMap` from `@/lib/hooks/api/useMaps` (`MapData = { id: string; attributes: { name: string; streetName: string } }`); Popover primitive (Task 6).
- Produces: `<MapPicker value onChange debounceMs? />` with props `{ value: number; onChange: (mapId: number) => void; debounceMs?: number }`. IdentitySection (Task 10) consumes it.

- [ ] **Step 1: Write the failing test**

`src/components/features/characters/templates/__tests__/MapPicker.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";

const useMapMock = vi.fn();
const useMapsByNameMock = vi.fn();
vi.mock("@/lib/hooks/api/useMaps", () => ({
  useMap: (...a: unknown[]) => useMapMock(...a),
  useMapsByName: (...a: unknown[]) => useMapsByNameMock(...a),
}));

import { MapPicker } from "../MapPicker";

const mushroomTown = {
  id: "10000",
  attributes: { name: "Mushroom Town", streetName: "Maple Road" },
};

beforeEach(() => {
  useMapMock.mockReset();
  useMapsByNameMock.mockReset();
  useMapsByNameMock.mockReturnValue({ data: [], isLoading: false });
});

describe("MapPicker", () => {
  it("shows <name> · <streetName> · <id> when the id resolves", () => {
    useMapMock.mockReturnValue({ data: mushroomTown, isError: false });
    render(<MapPicker value={10000} onChange={vi.fn()} />);
    expect(
      screen.getByRole("button", {
        name: /Mushroom Town · Maple Road · 10000/,
      }),
    ).toBeInTheDocument();
  });

  it("shows Map <id> with a warning hint when unresolvable (non-blocking)", () => {
    useMapMock.mockReturnValue({ data: undefined, isError: true });
    render(<MapPicker value={999999999} onChange={vi.fn()} />);
    expect(screen.getByText(/Map 999999999/)).toBeInTheDocument();
    expect(screen.getByText(/not found in map data/i)).toBeInTheDocument();
  });

  it("search results select a map by id", async () => {
    useMapMock.mockReturnValue({ data: undefined, isError: false });
    useMapsByNameMock.mockReturnValue({
      data: [mushroomTown],
      isLoading: false,
    });
    const onChange = vi.fn();
    render(<MapPicker value={0} onChange={onChange} debounceMs={0} />);
    await userEvent.click(screen.getByRole("button", { name: /Map 0/ }));
    await userEvent.type(screen.getByRole("textbox"), "mush");
    await userEvent.click(
      await screen.findByRole("option", { name: /Mushroom Town/ }),
    );
    expect(onChange).toHaveBeenCalledWith(10000);
  });

  it("numeric input offers the manual Use id fallback", async () => {
    useMapMock.mockReturnValue({ data: undefined, isError: false });
    const onChange = vi.fn();
    render(<MapPicker value={0} onChange={onChange} debounceMs={0} />);
    await userEvent.click(screen.getByRole("button", { name: /Map 0/ }));
    await userEvent.type(screen.getByRole("textbox"), "100000000");
    await userEvent.click(
      await screen.findByRole("option", { name: /use id 100000000/i }),
    );
    expect(onChange).toHaveBeenCalledWith(100000000);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `source ~/.nvm/nvm.sh && nvm use 22 >/dev/null 2>&1 && npm run test -- src/components/features/characters/templates/__tests__/MapPicker.test.tsx`
Expected: FAIL — `../MapPicker` unresolved.

- [ ] **Step 3: Write `MapPicker.tsx`**

```tsx
import { useEffect, useState } from "react";
import { TriangleAlert } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useMap, useMapsByName } from "@/lib/hooks/api/useMaps";

interface MapPickerProps {
  value: number;
  onChange: (mapId: number) => void;
  /** Test hook: pass 0 to disable debouncing. */
  debounceMs?: number;
}

export function MapPicker({ value, onChange, debounceMs = 300 }: MapPickerProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [debounced, setDebounced] = useState("");

  useEffect(() => {
    if (debounceMs === 0) {
      setDebounced(search);
      return;
    }
    const handle = setTimeout(() => setDebounced(search), debounceMs);
    return () => clearTimeout(handle);
  }, [search, debounceMs]);

  const current = useMap(String(value));
  const results = useMapsByName(debounced.trim());

  const currentLabel = current.data
    ? `${current.data.attributes.name} · ${current.data.attributes.streetName} · ${value}`
    : `Map ${value}`;
  // atlas-data coverage varies by version: unresolvable is a hint, not an error.
  const unresolved = value > 0 && !current.data && current.isError;

  const manualId = /^\d+$/.test(search.trim())
    ? Number(search.trim())
    : undefined;

  const pick = (mapId: number) => {
    onChange(mapId);
    setOpen(false);
    setSearch("");
  };

  return (
    <div className="space-y-1">
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            type="button"
            variant="outline"
            className="w-full justify-start font-normal"
          >
            {currentLabel}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-96 p-2" align="start">
          <Input
            autoFocus
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search maps by name or enter an id…"
          />
          <ul role="listbox" className="mt-2 max-h-64 space-y-0.5 overflow-y-auto">
            {(results.data ?? []).map((m) => (
              <li
                key={m.id}
                role="option"
                aria-selected={false}
                tabIndex={0}
                onClick={() => pick(Number(m.id))}
                onKeyDown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    pick(Number(m.id));
                  }
                }}
                className="flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-accent focus-visible:bg-accent"
              >
                <span className="flex-1 truncate text-sm">
                  {m.attributes.name} · {m.attributes.streetName}
                </span>
                <span className="font-mono text-xs text-muted-foreground">
                  {m.id}
                </span>
              </li>
            ))}
            {manualId !== undefined && (
              <li
                role="option"
                aria-selected={false}
                tabIndex={0}
                onClick={() => pick(manualId)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    pick(manualId);
                  }
                }}
                className="cursor-pointer rounded px-2 py-1 text-sm hover:bg-accent focus-visible:bg-accent"
              >
                Use id {manualId}
              </li>
            )}
            {results.isLoading && debounced.trim() && (
              <li className="px-2 py-1 text-sm text-muted-foreground">
                Searching…
              </li>
            )}
          </ul>
        </PopoverContent>
      </Popover>
      {unresolved && (
        <p className="flex items-center gap-1 text-xs text-amber-600 dark:text-amber-500">
          <TriangleAlert className="size-3" />
          not found in map data for this version
        </p>
      )}
    </div>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Same command as Step 2. Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add src/components/features/characters/templates/MapPicker.tsx src/components/features/characters/templates/__tests__/MapPicker.test.tsx
git commit -m "feat(task-177): starting-map picker"
```

### Task 8: `TemplateSelector`

**Files:**
- Create: `src/components/features/characters/templates/TemplateSelector.tsx`
- Test: `src/components/features/characters/templates/__tests__/TemplateSelector.test.tsx`

**Interfaces:**
- Consumes: `templateLabels` (Task 1).
- Produces: `<TemplateSelector templates selectedIndex onSelect onAdd />` with props `{ templates: Pick<CharacterTemplate, "jobIndex" | "gender">[]; selectedIndex: number; onSelect: (index: number) => void; onAdd: () => void }`. Editor (Task 15) consumes it.

- [ ] **Step 1: Write the failing test**

`src/components/features/characters/templates/__tests__/TemplateSelector.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { TemplateSelector } from "../TemplateSelector";

const templates = [
  { jobIndex: 1, gender: 0 },
  { jobIndex: 1, gender: 1 },
  { jobIndex: 1, gender: 0 },
];

describe("TemplateSelector", () => {
  it("renders tablist segments with derived labels incl. ordinals", () => {
    render(
      <TemplateSelector
        templates={templates}
        selectedIndex={0}
        onSelect={vi.fn()}
        onAdd={vi.fn()}
      />,
    );
    expect(screen.getByRole("tablist")).toBeInTheDocument();
    expect(
      screen.getByRole("tab", { name: "Adventurer · M" }),
    ).toHaveAttribute("aria-selected", "true");
    expect(
      screen.getByRole("tab", { name: "Adventurer · F" }),
    ).toHaveAttribute("aria-selected", "false");
    expect(
      screen.getByRole("tab", { name: "Adventurer · M (2)" }),
    ).toBeInTheDocument();
  });

  it("clicking a segment selects it; + New adds", async () => {
    const onSelect = vi.fn();
    const onAdd = vi.fn();
    render(
      <TemplateSelector
        templates={templates}
        selectedIndex={0}
        onSelect={onSelect}
        onAdd={onAdd}
      />,
    );
    await userEvent.click(screen.getByRole("tab", { name: "Adventurer · F" }));
    expect(onSelect).toHaveBeenCalledWith(1);
    await userEvent.click(screen.getByRole("button", { name: /new/i }));
    expect(onAdd).toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `source ~/.nvm/nvm.sh && nvm use 22 >/dev/null 2>&1 && npm run test -- src/components/features/characters/templates/__tests__/TemplateSelector.test.tsx`
Expected: FAIL — `../TemplateSelector` unresolved.

- [ ] **Step 3: Write `TemplateSelector.tsx`**

```tsx
import { Plus } from "lucide-react";
import type { CharacterTemplate } from "@/types/models/template";
import { cn } from "@/lib/utils";
import { templateLabels } from "./jobNames";

interface TemplateSelectorProps {
  templates: Pick<CharacterTemplate, "jobIndex" | "gender">[];
  selectedIndex: number;
  onSelect: (index: number) => void;
  onAdd: () => void;
}

/**
 * Segmented control (recessed track, flat text segments) — no thumbnails by
 * design: sprites always mean "rendered output", never navigation.
 */
export function TemplateSelector({
  templates,
  selectedIndex,
  onSelect,
  onAdd,
}: TemplateSelectorProps) {
  const labels = templateLabels(templates);
  return (
    <div
      role="tablist"
      aria-label="Character templates"
      className="flex flex-wrap items-center gap-1 rounded-lg bg-muted p-1"
    >
      {labels.map((label, index) => (
        <button
          key={index}
          type="button"
          role="tab"
          aria-selected={index === selectedIndex}
          onClick={() => onSelect(index)}
          className={cn(
            "rounded-md px-3 py-1.5 text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
            index === selectedIndex
              ? "bg-background font-medium shadow-sm"
              : "text-muted-foreground hover:text-foreground",
          )}
        >
          {label}
        </button>
      ))}
      <button
        type="button"
        onClick={onAdd}
        className="flex items-center gap-1 rounded-md px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        <Plus className="size-4" /> New
      </button>
    </div>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Same command as Step 2. Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add src/components/features/characters/templates/TemplateSelector.tsx src/components/features/characters/templates/__tests__/TemplateSelector.test.tsx
git commit -m "feat(task-177): segmented template selector"
```

### Task 9: `TemplateActionsMenu`

**Files:**
- Create: `src/components/features/characters/templates/TemplateActionsMenu.tsx`
- Test: `src/components/features/characters/templates/__tests__/TemplateActionsMenu.test.tsx`

**Interfaces:**
- Consumes: shadcn `DropdownMenu` + `AlertDialog` primitives (already in `src/components/ui/`).
- Produces: `<TemplateActionsMenu label onDuplicate onRemove />` with props `{ label: string; onDuplicate: () => void; onRemove: () => void }`. `onRemove` fires only after confirm. IdentitySection header (Task 10) hosts it via its `actions` slot; Editor (Task 15) wires the callbacks. No Edit-as-JSON item (user decision).

- [ ] **Step 1: Write the failing test**

`src/components/features/characters/templates/__tests__/TemplateActionsMenu.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { TemplateActionsMenu } from "../TemplateActionsMenu";

describe("TemplateActionsMenu", () => {
  it("offers Duplicate and Remove, and nothing else", async () => {
    render(
      <TemplateActionsMenu
        label="Adventurer · M"
        onDuplicate={vi.fn()}
        onRemove={vi.fn()}
      />,
    );
    await userEvent.click(
      screen.getByRole("button", { name: /template actions/i }),
    );
    expect(
      screen.getByRole("menuitem", { name: /duplicate template/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /remove template/i }),
    ).toBeInTheDocument();
    expect(screen.queryByText(/json/i)).not.toBeInTheDocument();
  });

  it("Duplicate fires immediately", async () => {
    const onDuplicate = vi.fn();
    render(
      <TemplateActionsMenu
        label="Adventurer · M"
        onDuplicate={onDuplicate}
        onRemove={vi.fn()}
      />,
    );
    await userEvent.click(
      screen.getByRole("button", { name: /template actions/i }),
    );
    await userEvent.click(
      screen.getByRole("menuitem", { name: /duplicate template/i }),
    );
    expect(onDuplicate).toHaveBeenCalled();
  });

  it("Remove requires confirm and shows the template label", async () => {
    const onRemove = vi.fn();
    render(
      <TemplateActionsMenu
        label="Adventurer · M"
        onDuplicate={vi.fn()}
        onRemove={onRemove}
      />,
    );
    await userEvent.click(
      screen.getByRole("button", { name: /template actions/i }),
    );
    await userEvent.click(
      screen.getByRole("menuitem", { name: /remove template/i }),
    );
    // confirm dialog: not yet removed
    expect(onRemove).not.toHaveBeenCalled();
    expect(screen.getByText(/Adventurer · M/)).toBeInTheDocument();
    expect(
      screen.getByText(/players can no longer create this class\/gender/i),
    ).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: /^remove$/i }));
    expect(onRemove).toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `source ~/.nvm/nvm.sh && nvm use 22 >/dev/null 2>&1 && npm run test -- src/components/features/characters/templates/__tests__/TemplateActionsMenu.test.tsx`
Expected: FAIL — `../TemplateActionsMenu` unresolved.

- [ ] **Step 3: Write `TemplateActionsMenu.tsx`**

```tsx
import { useState } from "react";
import { Copy, MoreHorizontal, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";

interface TemplateActionsMenuProps {
  label: string;
  onDuplicate: () => void;
  onRemove: () => void;
}

export function TemplateActionsMenu({
  label,
  onDuplicate,
  onRemove,
}: TemplateActionsMenuProps) {
  const [confirmOpen, setConfirmOpen] = useState(false);

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label="Template actions"
          >
            <MoreHorizontal className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem onSelect={onDuplicate}>
            <Copy className="size-4" /> Duplicate template
          </DropdownMenuItem>
          <DropdownMenuItem
            variant="destructive"
            onSelect={() => setConfirmOpen(true)}
          >
            <Trash2 className="size-4" /> Remove template
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove {label}?</AlertDialogTitle>
            <AlertDialogDescription>
              Players can no longer create this class/gender until it is
              re-added. This takes effect on save.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                setConfirmOpen(false);
                onRemove();
              }}
            >
              Remove
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
```

Note: if the local `dropdown-menu.tsx` has no `variant` prop on `DropdownMenuItem`, drop the prop and add `className="text-destructive focus:text-destructive"` instead — check the file before using it.

- [ ] **Step 4: Run test to verify it passes**

Same command as Step 2. Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add src/components/features/characters/templates/TemplateActionsMenu.tsx src/components/features/characters/templates/__tests__/TemplateActionsMenu.test.tsx
git commit -m "feat(task-177): template kebab actions menu"
```

### Task 10: `IdentitySection`

**Files:**
- Create: `src/components/features/characters/templates/IdentitySection.tsx`
- Test: `src/components/features/characters/templates/__tests__/IdentitySection.test.tsx`

**Interfaces:**
- Consumes: `KNOWN_CLASSES` (Task 1), `IdentityField` (Task 2), `MapPicker` (Task 7), shadcn `Select`, `Input`, `Label`.
- Produces: `<IdentitySection template onSetIdentity actions />` with props `{ template: CharacterTemplate; onSetIdentity: (field: IdentityField, value: number) => void; actions?: React.ReactNode }`. The `actions` slot renders top-right of the section header, aligned with the "Identity" title (FR-3.1 kebab anchor). Editor (Task 15) consumes it.

- [ ] **Step 1: Write the failing test**

`src/components/features/characters/templates/__tests__/IdentitySection.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";

const useMapMock = vi.fn();
vi.mock("@/lib/hooks/api/useMaps", () => ({
  useMap: (...a: unknown[]) => useMapMock(...a),
  useMapsByName: () => ({ data: [], isLoading: false }),
}));

import { blankTemplate, normalizeTemplate } from "../editorState";
import { IdentitySection } from "../IdentitySection";

beforeEach(() => {
  useMapMock.mockReturnValue({ data: undefined, isError: false });
});

describe("IdentitySection", () => {
  it("selecting a known class sets jobIndex and subJobIndex", async () => {
    const onSetIdentity = vi.fn();
    render(
      <IdentitySection template={blankTemplate()} onSetIdentity={onSetIdentity} />,
    );
    await userEvent.click(screen.getByRole("combobox", { name: /class/i }));
    await userEvent.click(
      await screen.findByRole("option", { name: /Aran \(2\.0\)/ }),
    );
    expect(onSetIdentity).toHaveBeenCalledWith("jobIndex", 2);
    expect(onSetIdentity).toHaveBeenCalledWith("subJobIndex", 0);
  });

  it("Advanced mode accepts arbitrary numeric job/subJob (backend validates)", async () => {
    const onSetIdentity = vi.fn();
    render(
      <IdentitySection template={blankTemplate()} onSetIdentity={onSetIdentity} />,
    );
    await userEvent.click(screen.getByRole("button", { name: /advanced/i }));
    const jobInput = screen.getByLabelText(/job index/i);
    await userEvent.clear(jobInput);
    await userEvent.type(jobInput, "1");
    const subInput = screen.getByLabelText(/sub job index/i);
    await userEvent.clear(subInput);
    await userEvent.type(subInput, "1"); // Dual Blade 1.1 — permitted
    expect(onSetIdentity).toHaveBeenCalledWith("jobIndex", 1);
    expect(onSetIdentity).toHaveBeenCalledWith("subJobIndex", 1);
  });

  it("gender select maps Male/Female to 0/1", async () => {
    const onSetIdentity = vi.fn();
    render(
      <IdentitySection template={blankTemplate()} onSetIdentity={onSetIdentity} />,
    );
    await userEvent.click(screen.getByRole("combobox", { name: /gender/i }));
    await userEvent.click(await screen.findByRole("option", { name: /female/i }));
    expect(onSetIdentity).toHaveBeenCalledWith("gender", 1);
  });

  it("unknown class combos display in the closed class control", () => {
    render(
      <IdentitySection
        template={normalizeTemplate({ jobIndex: 1, subJobIndex: 1 })}
        onSetIdentity={vi.fn()}
      />,
    );
    expect(screen.getByText(/Adventurer \(1\.1\)/)).toBeInTheDocument();
  });

  it("renders the actions slot in the header", () => {
    render(
      <IdentitySection
        template={blankTemplate()}
        onSetIdentity={vi.fn()}
        actions={<button type="button">kebab-here</button>}
      />,
    );
    expect(
      screen.getByRole("button", { name: "kebab-here" }),
    ).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `source ~/.nvm/nvm.sh && nvm use 22 >/dev/null 2>&1 && npm run test -- src/components/features/characters/templates/__tests__/IdentitySection.test.tsx`
Expected: FAIL — `../IdentitySection` unresolved.

- [ ] **Step 3: Write `IdentitySection.tsx`**

```tsx
import { useState, type ReactNode } from "react";
import type { CharacterTemplate } from "@/types/models/template";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { KNOWN_CLASSES, worldNameFromJobIndex } from "./jobNames";
import type { IdentityField } from "./editorState";
import { MapPicker } from "./MapPicker";

interface IdentitySectionProps {
  template: CharacterTemplate;
  onSetIdentity: (field: IdentityField, value: number) => void;
  /** Rendered top-right of the section header (kebab anchor — FR-3.1). */
  actions?: ReactNode;
}

export function IdentitySection({
  template,
  onSetIdentity,
  actions,
}: IdentitySectionProps) {
  const [advanced, setAdvanced] = useState(false);

  const classValue = `${template.jobIndex}.${template.subJobIndex}`;
  const known = KNOWN_CLASSES.find(
    (c) => c.jobIndex === template.jobIndex && c.subJobIndex === template.subJobIndex,
  );
  const classLabel =
    known?.label ??
    `${worldNameFromJobIndex(template.jobIndex)} (${classValue})`;

  const parseNumeric = (raw: string): number | undefined => {
    const n = Number(raw);
    return raw.trim() !== "" && Number.isFinite(n) ? n : undefined;
  };

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold">Identity</h3>
        {actions}
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="space-y-1">
          <Label htmlFor="tpl-class">Class</Label>
          <Select
            value={known ? classValue : ""}
            onValueChange={(v) => {
              const [job, sub] = v.split(".").map(Number);
              onSetIdentity("jobIndex", job);
              onSetIdentity("subJobIndex", sub);
            }}
          >
            <SelectTrigger id="tpl-class" aria-label="Class">
              <SelectValue placeholder={classLabel}>{classLabel}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              {KNOWN_CLASSES.map((c) => (
                <SelectItem
                  key={c.label}
                  value={`${c.jobIndex}.${c.subJobIndex}`}
                >
                  {c.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            type="button"
            variant="link"
            size="sm"
            className="h-auto p-0 text-xs"
            onClick={() => setAdvanced((a) => !a)}
          >
            Advanced
          </Button>
          {advanced && (
            <div className="grid grid-cols-2 gap-2">
              <div className="space-y-1">
                <Label htmlFor="tpl-job-index" className="text-xs">
                  Job index
                </Label>
                <Input
                  id="tpl-job-index"
                  inputMode="numeric"
                  defaultValue={template.jobIndex}
                  onChange={(e) => {
                    const n = parseNumeric(e.target.value);
                    if (n !== undefined) onSetIdentity("jobIndex", n);
                  }}
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="tpl-subjob-index" className="text-xs">
                  Sub job index
                </Label>
                <Input
                  id="tpl-subjob-index"
                  inputMode="numeric"
                  defaultValue={template.subJobIndex}
                  onChange={(e) => {
                    const n = parseNumeric(e.target.value);
                    if (n !== undefined) onSetIdentity("subJobIndex", n);
                  }}
                />
              </div>
            </div>
          )}
        </div>
        <div className="space-y-1">
          <Label htmlFor="tpl-gender">Gender</Label>
          <Select
            value={String(template.gender)}
            onValueChange={(v) => onSetIdentity("gender", Number(v))}
          >
            <SelectTrigger id="tpl-gender" aria-label="Gender">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="0">Male</SelectItem>
              <SelectItem value="1">Female</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1 sm:col-span-2">
          <Label>Starting map</Label>
          <MapPicker
            value={template.mapId}
            onChange={(mapId) => onSetIdentity("mapId", mapId)}
          />
        </div>
      </div>
    </section>
  );
}
```

Note: `Select` in jsdom needs `SelectValue` children for the closed-control text; keep the explicit `{classLabel}` child so unknown combos (e.g. 1.1) display without being a selectable value. Radix Select requires `hasPointerCapture` stubs in some jsdom versions — `src/test/setup.ts` already stubs `matchMedia`; if `userEvent.click` on the combobox throws, add the standard `Element.prototype.hasPointerCapture ||= () => false; Element.prototype.scrollIntoView ||= () => {}` stubs to the test file top.

- [ ] **Step 4: Run test to verify it passes**

Same command as Step 2. Expected: PASS (5 tests).

- [ ] **Step 5: Commit**

```bash
git add src/components/features/characters/templates/IdentitySection.tsx src/components/features/characters/templates/__tests__/IdentitySection.test.tsx
git commit -m "feat(task-177): identity section (class/gender/map)"
```

### Task 11: `AppearanceThumb` + `AppearancePoolSection`

**Files:**
- Create: `src/components/features/characters/templates/AppearanceThumb.tsx`
- Create: `src/components/features/characters/templates/AppearancePoolSection.tsx`
- Test: `src/components/features/characters/templates/__tests__/AppearancePoolSection.test.tsx`

**Interfaces:**
- Consumes: `buildVariantLoadout` (Task 3), `generateCharacterUrl` (`characterRender.service`), `useTenant`, `emptyPoolWarnings` semantics (renders its own empty warning), `AppearanceBrowserDialog` (Task 12 — this task renders a stub trigger; the dialog import lands in Task 12, so this section accepts the dialog as a render-prop to stay independently testable).
- Produces:
  - `<AppearanceThumb url idLabel selected? onSelect? onRemove? marked? />` props `{ url: string; idLabel: string | number; selected?: boolean; onSelect?: () => void; onRemove?: () => void; marked?: boolean }` with crop constants exported: `THUMB_SIZE = 76`, `THUMB_OFFSET_X = -74`, `THUMB_OFFSET_Y = -70` (starting values from the prototype; tune visually at the end).
  - `<AppearancePoolSection dimension title template picks onPick onRemoveEntry renderAddDialog />` props `{ dimension: AppearancePoolKey; title: string; template: CharacterTemplate; picks: PreviewPicks; onPick: (pick: keyof PreviewPicks, idx: number) => void; onRemoveEntry: (entryIndex: number) => void; renderAddDialog: (open: boolean, onOpenChange: (o: boolean) => void) => React.ReactNode }`. Editor (Task 15) supplies `renderAddDialog` closing over `AppearanceBrowserDialog`.

- [ ] **Step 1: Write the failing test**

`src/components/features/characters/templates/__tests__/AppearancePoolSection.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

import { normalizeTemplate, DEFAULT_PICKS } from "../editorState";
import { AppearancePoolSection } from "../AppearancePoolSection";

const tpl = normalizeTemplate({ faces: [20000, 21000] });

function renderSection(over: Record<string, unknown> = {}) {
  return render(
    <AppearancePoolSection
      dimension="faces"
      title="Faces"
      template={tpl}
      picks={DEFAULT_PICKS}
      onPick={vi.fn()}
      onRemoveEntry={vi.fn()}
      renderAddDialog={() => null}
      {...over}
    />,
  );
}

describe("AppearancePoolSection", () => {
  it("renders one thumb per pool entry with id captions", () => {
    renderSection();
    expect(screen.getByText("20000")).toBeInTheDocument();
    expect(screen.getByText("21000")).toBeInTheDocument();
  });

  it("clicking a thumb sets the preview pick (UI-only)", async () => {
    const onPick = vi.fn();
    renderSection({ onPick });
    await userEvent.click(
      screen.getByRole("button", { name: /preview face 21000/i }),
    );
    expect(onPick).toHaveBeenCalledWith("faceIdx", 1);
  });

  it("the picked thumb is marked pressed", () => {
    renderSection({ picks: { ...DEFAULT_PICKS, faceIdx: 1 } });
    expect(
      screen.getByRole("button", { name: /preview face 21000/i }),
    ).toHaveAttribute("aria-pressed", "true");
  });

  it("each thumb has a remove affordance", async () => {
    const onRemoveEntry = vi.fn();
    renderSection({ onRemoveEntry });
    await userEvent.click(
      screen.getByRole("button", { name: /remove face 20000/i }),
    );
    expect(onRemoveEntry).toHaveBeenCalledWith(0);
  });

  it("empty pool shows the non-blocking factory warning", () => {
    renderSection({ template: normalizeTemplate({}) });
    expect(
      screen.getByText(/character creation will fail while this pool is empty/i),
    ).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `source ~/.nvm/nvm.sh && nvm use 22 >/dev/null 2>&1 && npm run test -- src/components/features/characters/templates/__tests__/AppearancePoolSection.test.tsx`
Expected: FAIL — `../AppearancePoolSection` unresolved.

- [ ] **Step 3: Write `AppearanceThumb.tsx`**

```tsx
import { useState } from "react";
import { ImageOff, X } from "lucide-react";
import { cn } from "@/lib/utils";

// Crop of the 192×256 stand1 render at resize=2 down to the head region.
// Starting values from prototype.html; tune against real renders at the end.
export const THUMB_SIZE = 76;
export const THUMB_OFFSET_X = -74;
export const THUMB_OFFSET_Y = -70;

interface AppearanceThumbProps {
  url: string;
  idLabel: string | number;
  ariaLabel: string;
  selected?: boolean;
  onSelect?: () => void;
  onRemove?: () => void;
  removeAriaLabel?: string;
  marked?: boolean;
}

export function AppearanceThumb({
  url,
  idLabel,
  ariaLabel,
  selected = false,
  onSelect,
  onRemove,
  removeAriaLabel,
  marked = false,
}: AppearanceThumbProps) {
  const [failed, setFailed] = useState(false);

  return (
    <div className="group relative">
      <button
        type="button"
        aria-label={ariaLabel}
        aria-pressed={selected}
        onClick={onSelect}
        disabled={marked}
        className={cn(
          "relative overflow-hidden rounded-md border bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
          selected && "ring-2 ring-primary",
          marked && "cursor-not-allowed opacity-50",
        )}
        style={{ width: THUMB_SIZE, height: THUMB_SIZE }}
      >
        {failed ? (
          <span className="flex h-full w-full items-center justify-center text-muted-foreground">
            <ImageOff className="size-5" />
          </span>
        ) : (
          <img
            src={url}
            alt=""
            width={192}
            height={256}
            loading="lazy"
            onError={() => setFailed(true)}
            className="absolute max-w-none [image-rendering:pixelated]"
            style={{ left: THUMB_OFFSET_X, top: THUMB_OFFSET_Y }}
          />
        )}
        <span className="absolute inset-x-0 bottom-0 bg-background/80 text-center font-mono text-[10px] leading-4">
          {idLabel}
        </span>
      </button>
      {onRemove && (
        <button
          type="button"
          aria-label={removeAriaLabel ?? `Remove ${idLabel}`}
          onClick={onRemove}
          className="absolute -right-1.5 -top-1.5 hidden size-5 items-center justify-center rounded-full border bg-background text-muted-foreground shadow-sm hover:text-destructive group-focus-within:flex group-hover:flex"
        >
          <X className="size-3" />
        </button>
      )}
    </div>
  );
}
```

- [ ] **Step 4: Write `AppearancePoolSection.tsx`**

```tsx
import { useState, type ReactNode } from "react";
import { Plus, TriangleAlert } from "lucide-react";
import type { CharacterTemplate } from "@/types/models/template";
import { Button } from "@/components/ui/button";
import { generateCharacterUrl } from "@/services/api/characterRender.service";
import { useTenant } from "@/context/tenant-context";
import {
  PICK_KEY_BY_POOL,
  type AppearancePoolKey,
  type PreviewPicks,
} from "./editorState";
import { buildVariantLoadout } from "./previewLoadout";
import { AppearanceThumb } from "./AppearanceThumb";

interface AppearancePoolSectionProps {
  dimension: AppearancePoolKey;
  title: string;
  template: CharacterTemplate;
  picks: PreviewPicks;
  onPick: (pick: keyof PreviewPicks, idx: number) => void;
  onRemoveEntry: (entryIndex: number) => void;
  /** Editor supplies the AppearanceBrowserDialog here (open state owned locally). */
  renderAddDialog: (
    open: boolean,
    onOpenChange: (open: boolean) => void,
  ) => ReactNode;
}

// Singular noun for aria labels, e.g. "Preview face 20000".
const NOUN: Record<AppearancePoolKey, string> = {
  faces: "face",
  hairs: "hair",
  hairColors: "hair color",
  skinColors: "skin tone",
};

export function AppearancePoolSection({
  dimension,
  title,
  template,
  picks,
  onPick,
  onRemoveEntry,
  renderAddDialog,
}: AppearancePoolSectionProps) {
  const { activeTenant } = useTenant();
  const [addOpen, setAddOpen] = useState(false);
  const pickKey = PICK_KEY_BY_POOL[dimension]!;
  const pool = template[dimension];

  return (
    <section className="space-y-2">
      <div className="flex items-center gap-2">
        <h3 className="text-sm font-semibold">{title}</h3>
        <span className="text-xs text-muted-foreground">
          {pool.length} options · player picks one
        </span>
        {pool.length === 0 && (
          <span className="flex items-center gap-1 text-xs text-amber-600 dark:text-amber-500">
            <TriangleAlert className="size-3" />
            character creation will fail while this pool is empty
          </span>
        )}
      </div>
      <div className="flex flex-wrap items-start gap-2">
        {activeTenant &&
          pool.map((id, idx) => (
            <AppearanceThumb
              key={`${id}-${idx}`}
              url={generateCharacterUrl(
                activeTenant.id,
                activeTenant.attributes.region,
                activeTenant.attributes.majorVersion,
                activeTenant.attributes.minorVersion,
                buildVariantLoadout(template, picks, dimension, id),
                { stance: "stand1", resize: 2 },
              )}
              idLabel={id}
              ariaLabel={`Preview ${NOUN[dimension]} ${id}`}
              selected={picks[pickKey] === idx}
              onSelect={() => onPick(pickKey, idx)}
              onRemove={() => onRemoveEntry(idx)}
              removeAriaLabel={`Remove ${NOUN[dimension]} ${id}`}
            />
          ))}
        <Button
          type="button"
          variant="outline"
          className="h-[76px] w-[76px] flex-col gap-1 text-xs"
          onClick={() => setAddOpen(true)}
        >
          <Plus className="size-4" /> Add
        </Button>
      </div>
      {renderAddDialog(addOpen, setAddOpen)}
    </section>
  );
}
```

- [ ] **Step 5: Run test to verify it passes**

Same command as Step 2. Expected: PASS (5 tests).

- [ ] **Step 6: Commit**

```bash
git add src/components/features/characters/templates/AppearanceThumb.tsx src/components/features/characters/templates/AppearancePoolSection.tsx src/components/features/characters/templates/__tests__/AppearancePoolSection.test.tsx
git commit -m "feat(task-177): appearance pool thumbnails"
```

### Task 12: `AppearanceBrowserDialog`

**Files:**
- Create: `src/components/features/characters/templates/AppearanceBrowserDialog.tsx`
- Test: `src/components/features/characters/templates/__tests__/AppearanceBrowserDialog.test.tsx`

**Interfaces:**
- Consumes: `useFaceIds`/`useHairIds` (Task 4), `useItemNames` (Task 5), `isFemaleCosmeticId` (Task 3), `buildVariantLoadout` (Task 3), `AppearanceThumb` (Task 11), shadcn `Dialog`, `Switch`, `ErrorDisplay` from `@/components/common`.
- Produces: `<AppearanceBrowserDialog dimension template picks open onOpenChange onAdd />` with props `{ dimension: AppearancePoolKey; template: CharacterTemplate; picks: PreviewPicks; open: boolean; onOpenChange: (open: boolean) => void; onAdd: (id: number) => void }`. Candidate sources per dimension: faces/hairs from the cosmetics hooks (gender-filtered with show-all toggle), hairColors digits 0–7, skinColors 0–9. `PAGE_SIZE = 24` exported. Editor (Task 15) consumes it via `renderAddDialog`.

- [ ] **Step 1: Write the failing test**

`src/components/features/characters/templates/__tests__/AppearanceBrowserDialog.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

const useFaceIdsMock = vi.fn();
const useHairIdsMock = vi.fn();
vi.mock("@/lib/hooks/api/useCosmetics", () => ({
  useFaceIds: (...a: unknown[]) => useFaceIdsMock(...a),
  useHairIds: (...a: unknown[]) => useHairIdsMock(...a),
}));

const useItemNamesMock = vi.fn();
vi.mock("@/lib/hooks/api/useItemNames", () => ({
  useItemNames: (...a: unknown[]) => useItemNamesMock(...a),
}));

import { normalizeTemplate, DEFAULT_PICKS } from "../editorState";
import { AppearanceBrowserDialog, PAGE_SIZE } from "../AppearanceBrowserDialog";

// 20000-20009 male, 21000-21009 female
const faceIds = [
  ...Array.from({ length: 30 }, (_, i) => 20000 + i),
  ...Array.from({ length: 10 }, (_, i) => 21000 + i),
];

function renderDialog(over: Record<string, unknown> = {}) {
  return render(
    <AppearanceBrowserDialog
      dimension="faces"
      template={normalizeTemplate({ gender: 0, faces: [20000] })}
      picks={DEFAULT_PICKS}
      open
      onOpenChange={vi.fn()}
      onAdd={vi.fn()}
      {...over}
    />,
  );
}

beforeEach(() => {
  useFaceIdsMock.mockReturnValue({
    data: faceIds,
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  });
  useHairIdsMock.mockReturnValue({
    data: [],
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  });
  useItemNamesMock.mockReturnValue({ 20001: "Male 2" });
});

describe("AppearanceBrowserDialog", () => {
  it("gender-filters candidates by the id convention, with a show-all toggle", async () => {
    renderDialog();
    // male template: female id 21000 hidden
    expect(screen.queryByText("21000")).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("switch", { name: /show all/i }));
    expect(screen.getByText("21000")).toBeInTheDocument();
  });

  it("caps the grid at PAGE_SIZE per page and pages through candidates", async () => {
    renderDialog();
    // 30 male faces → page 1 shows PAGE_SIZE, 20029 is on page 2
    expect(screen.getAllByRole("button", { name: /add face/i })).toHaveLength(
      PAGE_SIZE,
    );
    expect(screen.queryByText("20029")).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: /next/i }));
    expect(screen.getByText("20029")).toBeInTheDocument();
  });

  it("marks already-in-pool ids as disabled", () => {
    renderDialog();
    expect(
      screen.getByRole("button", { name: /add face 20000/i }),
    ).toBeDisabled();
  });

  it("clicking a candidate adds it", async () => {
    const onAdd = vi.fn();
    renderDialog({ onAdd });
    await userEvent.click(
      screen.getByRole("button", { name: /add face 20001/i }),
    );
    expect(onAdd).toHaveBeenCalledWith(20001);
  });

  it("resolves names for the current page", () => {
    renderDialog();
    expect(useItemNamesMock).toHaveBeenCalled();
    expect(screen.getByText("Male 2")).toBeInTheDocument();
  });

  it("hairColors offers digits 0-7 on the current base hair (no enumeration)", () => {
    renderDialog({
      dimension: "hairColors",
      template: normalizeTemplate({ hairs: [30030], hairColors: [0] }),
    });
    expect(screen.getAllByRole("button", { name: /add hair color/i })).toHaveLength(8);
    expect(
      screen.getByRole("button", { name: /add hair color 0/i }),
    ).toBeDisabled();
  });

  it("skinColors offers 0-9 rendered previews", () => {
    renderDialog({
      dimension: "skinColors",
      template: normalizeTemplate({}),
    });
    expect(screen.getAllByRole("button", { name: /add skin tone/i })).toHaveLength(10);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `source ~/.nvm/nvm.sh && nvm use 22 >/dev/null 2>&1 && npm run test -- src/components/features/characters/templates/__tests__/AppearanceBrowserDialog.test.tsx`
Expected: FAIL — `../AppearanceBrowserDialog` unresolved.

- [ ] **Step 3: Write `AppearanceBrowserDialog.tsx`**

```tsx
import { useMemo, useState } from "react";
import type { CharacterTemplate } from "@/types/models/template";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { ErrorDisplay } from "@/components/common";
import {
  generateCharacterUrl,
  isFemaleCosmeticId,
} from "@/services/api/characterRender.service";
import { useTenant } from "@/context/tenant-context";
import { useFaceIds, useHairIds } from "@/lib/hooks/api/useCosmetics";
import { useItemNames } from "@/lib/hooks/api/useItemNames";
import type { AppearancePoolKey, PreviewPicks } from "./editorState";
import { buildVariantLoadout } from "./previewLoadout";
import { AppearanceThumb } from "./AppearanceThumb";

export const PAGE_SIZE = 24;

const HAIR_COLOR_DIGITS = [0, 1, 2, 3, 4, 5, 6, 7];
// No enumeration endpoint exists for skins; seed data uses 0-3. Offer 0-9
// with rendered previews and let the operator judge (PRD open question 1).
const SKIN_IDS = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9];

const TITLES: Record<AppearancePoolKey, string> = {
  faces: "Browse faces",
  hairs: "Browse hairs",
  hairColors: "Add hair colors",
  skinColors: "Add skin tones",
};

const NOUN: Record<AppearancePoolKey, string> = {
  faces: "face",
  hairs: "hair",
  hairColors: "hair color",
  skinColors: "skin tone",
};

interface AppearanceBrowserDialogProps {
  dimension: AppearancePoolKey;
  template: CharacterTemplate;
  picks: PreviewPicks;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onAdd: (id: number) => void;
}

export function AppearanceBrowserDialog({
  dimension,
  template,
  picks,
  open,
  onOpenChange,
  onAdd,
}: AppearanceBrowserDialogProps) {
  const { activeTenant } = useTenant();
  const [showAll, setShowAll] = useState(false);
  const [page, setPage] = useState(0);

  const isEnumerated = dimension === "faces" || dimension === "hairs";
  const faces = useFaceIds();
  const hairs = useHairIds();
  const enumQuery = dimension === "faces" ? faces : hairs;

  const candidates = useMemo(() => {
    if (dimension === "hairColors") return HAIR_COLOR_DIGITS;
    if (dimension === "skinColors") return SKIN_IDS;
    const all = enumQuery.data ?? [];
    if (showAll) return all;
    const wantFemale = template.gender === 1;
    return all.filter((id) => isFemaleCosmeticId(id) === wantFemale);
  }, [dimension, enumQuery.data, showAll, template.gender]);

  const pageCount = Math.max(1, Math.ceil(candidates.length / PAGE_SIZE));
  const clampedPage = Math.min(page, pageCount - 1);
  const pageIds = candidates.slice(
    clampedPage * PAGE_SIZE,
    (clampedPage + 1) * PAGE_SIZE,
  );

  // Names only exist for faces/hairs (item-strings covers them by id; the
  // search index does NOT — enumerate + resolve per page, never search).
  const names = useItemNames(isEnumerated ? pageIds : []);

  const inPool = template[dimension];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{TITLES[dimension]}</DialogTitle>
        </DialogHeader>
        {isEnumerated && (
          <div className="flex items-center gap-2">
            <Switch
              id="show-all-genders"
              checked={showAll}
              onCheckedChange={(v) => {
                setShowAll(v);
                setPage(0);
              }}
              aria-label="Show all genders"
            />
            <Label htmlFor="show-all-genders" className="text-sm">
              Show all genders
            </Label>
          </div>
        )}
        {isEnumerated && enumQuery.isError ? (
          <ErrorDisplay
            error={`Failed to enumerate ${dimension}`}
            retry={() => void enumQuery.refetch()}
          />
        ) : (
          <>
            <div className="grid max-h-[420px] grid-cols-4 gap-2 overflow-y-auto sm:grid-cols-6">
              {activeTenant &&
                pageIds.map((id) => (
                  <div key={id} className="flex flex-col items-center gap-0.5">
                    <AppearanceThumb
                      url={generateCharacterUrl(
                        activeTenant.id,
                        activeTenant.attributes.region,
                        activeTenant.attributes.majorVersion,
                        activeTenant.attributes.minorVersion,
                        buildVariantLoadout(template, picks, dimension, id),
                        { stance: "stand1", resize: 2 },
                      )}
                      idLabel={id}
                      ariaLabel={`Add ${NOUN[dimension]} ${id}`}
                      marked={inPool.includes(id)}
                      onSelect={() => onAdd(id)}
                    />
                    {isEnumerated && (
                      <span className="max-w-[76px] truncate text-[10px] text-muted-foreground">
                        {names[id] ?? "…"}
                      </span>
                    )}
                  </div>
                ))}
              {isEnumerated && enumQuery.isLoading && (
                <p className="col-span-full text-sm text-muted-foreground">
                  Loading candidates…
                </p>
              )}
            </div>
            {pageCount > 1 && (
              <div className="flex items-center justify-between">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={clampedPage === 0}
                  onClick={() => setPage((p) => Math.max(0, p - 1))}
                >
                  Previous
                </Button>
                <span className="text-xs text-muted-foreground">
                  Page {clampedPage + 1} of {pageCount}
                </span>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={clampedPage >= pageCount - 1}
                  onClick={() => setPage((p) => p + 1)}
                >
                  Next
                </Button>
              </div>
            )}
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
```

Note: `marked` thumbs are disabled buttons (Task 11), so clicking an in-pool id is a no-op — double-add prevention at both the UI and reducer layers.

- [ ] **Step 4: Run test to verify it passes**

Same command as Step 2. Expected: PASS (7 tests).

- [ ] **Step 5: Commit**

```bash
git add src/components/features/characters/templates/AppearanceBrowserDialog.tsx src/components/features/characters/templates/__tests__/AppearanceBrowserDialog.test.tsx
git commit -m "feat(task-177): visual appearance browser dialog"
```

### Task 13: `ItemRow` + `EquipmentPoolSection` + `StartingKitSection`

**Files:**
- Create: `src/components/features/characters/templates/ItemRow.tsx`
- Create: `src/components/features/characters/templates/EquipmentPoolSection.tsx`
- Create: `src/components/features/characters/templates/StartingKitSection.tsx`
- Test: `src/components/features/characters/templates/__tests__/EquipmentPoolSection.test.tsx`
- Test: `src/components/features/characters/templates/__tests__/StartingKitSection.test.tsx`

**Interfaces:**
- Consumes: `useItemName` (`@/lib/hooks/api/useItemStrings`), `useSkillData` (`@/lib/hooks/useSkillData` — returns `{ name?: string; iconUrl?: string; ... }`), `getAssetIconUrl`, `useTenant`, `ItemSearchCombobox` (Task 6), `EquipmentPoolKey` (Task 3).
- Produces:
  - `<ItemRow id onRemove removeAriaLabel />` props `{ id: number; onRemove: () => void; removeAriaLabel: string }` — icon + name (or `Unknown item`) + mono id + remove ×.
  - `<EquipmentPoolSection poolKey title ids onAdd onRemove />` props `{ poolKey: EquipmentPoolKey; title: string; ids: number[]; onAdd: (id: number) => void; onRemove: (entryIndex: number) => void }` — header `<n> options · player picks one`.
  - `<StartingKitSection items skills onAddItem onRemoveItem onAddSkill onRemoveSkill />` props `{ items: number[]; skills: number[]; onAddItem: (id: number) => void; onRemoveItem: (entryIndex: number) => void; onAddSkill: (id: number) => void; onRemoveSkill: (entryIndex: number) => void }` — items header `<n> granted`; skills add via numeric input with name/icon lookup (no browser in v1); skills empty copy: "This class starts with no granted skills."
  - Editor (Task 15) consumes both sections.

- [ ] **Step 1: Write the failing tests**

`src/components/features/characters/templates/__tests__/EquipmentPoolSection.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

const useItemNameMock = vi.fn();
vi.mock("@/lib/hooks/api/useItemStrings", () => ({
  useItemName: (...a: unknown[]) => useItemNameMock(...a),
  itemStringKeys: {
    all: ["item-strings"],
    byId: (id: string) => ["item-strings", "name", id],
  },
}));

vi.mock("../ItemSearchCombobox", () => ({
  ItemSearchCombobox: ({ onAdd }: { onAdd: (id: number) => void }) => (
    <button type="button" onClick={() => onAdd(1041002)}>
      mock-add
    </button>
  ),
}));

import { EquipmentPoolSection } from "../EquipmentPoolSection";

beforeEach(() => {
  useItemNameMock.mockReturnValue({ data: "Blue T-Shirt", isError: false });
});

describe("EquipmentPoolSection", () => {
  it("renders icon+name+id rows with the options header", () => {
    render(
      <EquipmentPoolSection
        poolKey="tops"
        title="Tops"
        ids={[1041002]}
        onAdd={vi.fn()}
        onRemove={vi.fn()}
      />,
    );
    expect(screen.getByText("1 options · player picks one")).toBeInTheDocument();
    expect(screen.getByText("Blue T-Shirt")).toBeInTheDocument();
    expect(screen.getByText("1041002")).toBeInTheDocument();
  });

  it("degrades to Unknown item when the name lookup fails, still removable", async () => {
    useItemNameMock.mockReturnValue({ data: undefined, isError: true });
    const onRemove = vi.fn();
    render(
      <EquipmentPoolSection
        poolKey="tops"
        title="Tops"
        ids={[9999999]}
        onAdd={vi.fn()}
        onRemove={onRemove}
      />,
    );
    expect(screen.getByText("Unknown item")).toBeInTheDocument();
    await userEvent.click(
      screen.getByRole("button", { name: /remove 9999999/i }),
    );
    expect(onRemove).toHaveBeenCalledWith(0);
  });

  it("combobox add wires through", async () => {
    const onAdd = vi.fn();
    render(
      <EquipmentPoolSection
        poolKey="tops"
        title="Tops"
        ids={[]}
        onAdd={onAdd}
        onRemove={vi.fn()}
      />,
    );
    await userEvent.click(screen.getByRole("button", { name: "mock-add" }));
    expect(onAdd).toHaveBeenCalledWith(1041002);
  });
});
```

`src/components/features/characters/templates/__tests__/StartingKitSection.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

const useItemNameMock = vi.fn();
vi.mock("@/lib/hooks/api/useItemStrings", () => ({
  useItemName: (...a: unknown[]) => useItemNameMock(...a),
  itemStringKeys: {
    all: ["item-strings"],
    byId: (id: string) => ["item-strings", "name", id],
  },
}));

const useSkillDataMock = vi.fn();
vi.mock("@/lib/hooks/useSkillData", () => ({
  useSkillData: (...a: unknown[]) => useSkillDataMock(...a),
}));

vi.mock("../ItemSearchCombobox", () => ({
  ItemSearchCombobox: ({ onAdd }: { onAdd: (id: number) => void }) => (
    <button type="button" onClick={() => onAdd(2000000)}>
      mock-add-item
    </button>
  ),
}));

import { StartingKitSection } from "../StartingKitSection";

beforeEach(() => {
  useItemNameMock.mockReturnValue({ data: "Red Potion", isError: false });
  useSkillDataMock.mockReturnValue({
    name: "Three Snails",
    iconUrl: "/icon.png",
  });
});

describe("StartingKitSection", () => {
  it("items header shows <n> granted and rows render", () => {
    render(
      <StartingKitSection
        items={[2000000]}
        skills={[]}
        onAddItem={vi.fn()}
        onRemoveItem={vi.fn()}
        onAddSkill={vi.fn()}
        onRemoveSkill={vi.fn()}
      />,
    );
    expect(screen.getByText("1 granted")).toBeInTheDocument();
    expect(screen.getByText("Red Potion")).toBeInTheDocument();
  });

  it("empty skills shows the class-specific empty copy", () => {
    render(
      <StartingKitSection
        items={[]}
        skills={[]}
        onAddItem={vi.fn()}
        onRemoveItem={vi.fn()}
        onAddSkill={vi.fn()}
        onRemoveSkill={vi.fn()}
      />,
    );
    expect(
      screen.getByText(/this class starts with no granted skills/i),
    ).toBeInTheDocument();
  });

  it("skill rows resolve names; numeric add dispatches", async () => {
    const onAddSkill = vi.fn();
    render(
      <StartingKitSection
        items={[]}
        skills={[1000]}
        onAddItem={vi.fn()}
        onRemoveItem={vi.fn()}
        onAddSkill={onAddSkill}
        onRemoveSkill={vi.fn()}
      />,
    );
    expect(screen.getByText("Three Snails")).toBeInTheDocument();
    await userEvent.type(
      screen.getByRole("textbox", { name: /skill id/i }),
      "1001",
    );
    await userEvent.click(screen.getByRole("button", { name: /add skill/i }));
    expect(onAddSkill).toHaveBeenCalledWith(1001);
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `source ~/.nvm/nvm.sh && nvm use 22 >/dev/null 2>&1 && npm run test -- src/components/features/characters/templates/__tests__/EquipmentPoolSection.test.tsx src/components/features/characters/templates/__tests__/StartingKitSection.test.tsx`
Expected: FAIL — modules unresolved.

- [ ] **Step 3: Write `ItemRow.tsx`**

```tsx
import { useState } from "react";
import { Package, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useItemName } from "@/lib/hooks/api/useItemStrings";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { useTenant } from "@/context/tenant-context";

interface ItemRowProps {
  id: number;
  onRemove: () => void;
  removeAriaLabel: string;
}

/** Icon + display name + mono id + remove ×. Bad ids degrade, never block. */
export function ItemRow({ id, onRemove, removeAriaLabel }: ItemRowProps) {
  const { activeTenant } = useTenant();
  const name = useItemName(String(id));
  const [iconFailed, setIconFailed] = useState(false);

  return (
    <div className="flex items-center gap-2 rounded-md border px-2 py-1.5">
      {activeTenant && !iconFailed ? (
        <img
          src={getAssetIconUrl(
            activeTenant.id,
            activeTenant.attributes.region,
            activeTenant.attributes.majorVersion,
            activeTenant.attributes.minorVersion,
            "item",
            id,
          )}
          alt=""
          width={28}
          height={28}
          loading="lazy"
          onError={() => setIconFailed(true)}
          className="[image-rendering:pixelated]"
        />
      ) : (
        <Package className="size-7 p-1 text-muted-foreground" />
      )}
      <span className="flex-1 truncate text-sm">
        {name.data ?? (name.isError ? "Unknown item" : "…")}
      </span>
      <span className="font-mono text-xs text-muted-foreground">{id}</span>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        aria-label={removeAriaLabel}
        onClick={onRemove}
      >
        <X className="size-4" />
      </Button>
    </div>
  );
}
```

- [ ] **Step 4: Write `EquipmentPoolSection.tsx`**

```tsx
import { ItemSearchCombobox } from "./ItemSearchCombobox";
import { ItemRow } from "./ItemRow";
import type { EquipmentPoolKey } from "./previewLoadout";

interface EquipmentPoolSectionProps {
  poolKey: EquipmentPoolKey;
  title: string;
  ids: number[];
  onAdd: (id: number) => void;
  onRemove: (entryIndex: number) => void;
}

export function EquipmentPoolSection({
  poolKey,
  title,
  ids,
  onAdd,
  onRemove,
}: EquipmentPoolSectionProps) {
  return (
    <section className="space-y-2">
      <div className="flex items-center justify-between">
        <div className="flex items-baseline gap-2">
          <h3 className="text-sm font-semibold">{title}</h3>
          <span className="text-xs text-muted-foreground">
            {ids.length} options · player picks one
          </span>
        </div>
        <ItemSearchCombobox poolKey={poolKey} existingIds={ids} onAdd={onAdd} />
      </div>
      <div className="space-y-1">
        {ids.map((id, idx) => (
          <ItemRow
            key={`${id}-${idx}`}
            id={id}
            onRemove={() => onRemove(idx)}
            removeAriaLabel={`Remove ${id}`}
          />
        ))}
      </div>
    </section>
  );
}
```

- [ ] **Step 5: Write `StartingKitSection.tsx`**

```tsx
import { useState } from "react";
import { Sparkles, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useSkillData } from "@/lib/hooks/useSkillData";
import { ItemSearchCombobox } from "./ItemSearchCombobox";
import { ItemRow } from "./ItemRow";

interface StartingKitSectionProps {
  items: number[];
  skills: number[];
  onAddItem: (id: number) => void;
  onRemoveItem: (entryIndex: number) => void;
  onAddSkill: (id: number) => void;
  onRemoveSkill: (entryIndex: number) => void;
}

function SkillRow({
  id,
  onRemove,
}: {
  id: number;
  onRemove: () => void;
}) {
  const skill = useSkillData(id);
  const [iconFailed, setIconFailed] = useState(false);
  return (
    <div className="flex items-center gap-2 rounded-md border px-2 py-1.5">
      {skill.iconUrl && !iconFailed ? (
        <img
          src={skill.iconUrl}
          alt=""
          width={28}
          height={28}
          loading="lazy"
          onError={() => setIconFailed(true)}
          className="[image-rendering:pixelated]"
        />
      ) : (
        <Sparkles className="size-7 p-1 text-muted-foreground" />
      )}
      <span className="flex-1 truncate text-sm">
        {skill.name ?? "Unknown skill"}
      </span>
      <span className="font-mono text-xs text-muted-foreground">{id}</span>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        aria-label={`Remove skill ${id}`}
        onClick={onRemove}
      >
        <X className="size-4" />
      </Button>
    </div>
  );
}

export function StartingKitSection({
  items,
  skills,
  onAddItem,
  onRemoveItem,
  onAddSkill,
  onRemoveSkill,
}: StartingKitSectionProps) {
  const [skillInput, setSkillInput] = useState("");
  const skillId = /^\d+$/.test(skillInput.trim())
    ? Number(skillInput.trim())
    : undefined;

  return (
    <section className="space-y-4">
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <div className="flex items-baseline gap-2">
            <h3 className="text-sm font-semibold">Starting items</h3>
            <span className="text-xs text-muted-foreground">
              {items.length} granted
            </span>
          </div>
          <ItemSearchCombobox
            poolKey="items"
            existingIds={items}
            onAdd={onAddItem}
          />
        </div>
        <div className="space-y-1">
          {items.map((id, idx) => (
            <ItemRow
              key={`${id}-${idx}`}
              id={id}
              onRemove={() => onRemoveItem(idx)}
              removeAriaLabel={`Remove ${id}`}
            />
          ))}
        </div>
      </div>
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <div className="flex items-baseline gap-2">
            <h3 className="text-sm font-semibold">Starting skills</h3>
            <span className="text-xs text-muted-foreground">
              {skills.length} granted
            </span>
          </div>
          <div className="flex items-center gap-1">
            <Input
              aria-label="Skill id"
              inputMode="numeric"
              className="h-8 w-28"
              placeholder="Skill id…"
              value={skillInput}
              onChange={(e) => setSkillInput(e.target.value)}
            />
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={skillId === undefined || skills.includes(skillId)}
              onClick={() => {
                if (skillId !== undefined) {
                  onAddSkill(skillId);
                  setSkillInput("");
                }
              }}
            >
              Add skill
            </Button>
          </div>
        </div>
        {skills.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            This class starts with no granted skills.
          </p>
        ) : (
          <div className="space-y-1">
            {skills.map((id, idx) => (
              <SkillRow key={`${id}-${idx}`} id={id} onRemove={() => onRemoveSkill(idx)} />
            ))}
          </div>
        )}
      </div>
    </section>
  );
}
```

- [ ] **Step 6: Run tests to verify they pass**

Same command as Step 2. Expected: PASS (6 tests).

- [ ] **Step 7: Commit**

```bash
git add src/components/features/characters/templates/ItemRow.tsx src/components/features/characters/templates/EquipmentPoolSection.tsx src/components/features/characters/templates/StartingKitSection.tsx src/components/features/characters/templates/__tests__/EquipmentPoolSection.test.tsx src/components/features/characters/templates/__tests__/StartingKitSection.test.tsx
git commit -m "feat(task-177): equipment + starting kit sections"
```

### Task 14: `PreviewCard`

**Files:**
- Create: `src/components/features/characters/templates/PreviewCard.tsx`
- Test: `src/components/features/characters/templates/__tests__/PreviewCard.test.tsx`

**Interfaces:**
- Consumes: `buildPreviewLoadout`, `EQUIP_SLOT_BY_POOL` (Task 3), `useCharacterImage` (`@/lib/hooks/useCharacterImage` — takes `MapleStoryCharacterData`, returns `{ imageUrl, isLoading, isError, refetch, ... }`), `useItemName`, `getAssetIconUrl`, `useTenant`, shadcn `Skeleton`, `Tooltip`.
- Produces: `<PreviewCard template picks />` props `{ template: CharacterTemplate; picks: PreviewPicks }`. Sticky styling classes live here (`lg:sticky lg:top-4`). No id/value table (user decision). Editor (Task 15) consumes it.

- [ ] **Step 1: Write the failing test**

`src/components/features/characters/templates/__tests__/PreviewCard.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

const useCharacterImageMock = vi.fn();
vi.mock("@/lib/hooks/useCharacterImage", () => ({
  useCharacterImage: (...a: unknown[]) => useCharacterImageMock(...a),
}));

const useItemNameMock = vi.fn();
vi.mock("@/lib/hooks/api/useItemStrings", () => ({
  useItemName: (...a: unknown[]) => useItemNameMock(...a),
  itemStringKeys: {
    all: ["item-strings"],
    byId: (id: string) => ["item-strings", "name", id],
  },
}));

import { normalizeTemplate, DEFAULT_PICKS } from "../editorState";
import { PreviewCard } from "../PreviewCard";

const tpl = normalizeTemplate({
  gender: 0,
  faces: [20000],
  hairs: [30030],
  hairColors: [2],
  skinColors: [1],
  tops: [1041002, 1041003],
  weapons: [1302000],
});

beforeEach(() => {
  useItemNameMock.mockReturnValue({ data: "Item", isError: false });
  useCharacterImageMock.mockReturnValue({
    imageUrl: "/api/assets/t1/GMS/83.1/character/abc.png",
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  });
});

describe("PreviewCard", () => {
  it("builds the loadout from picks + first-of-pool equipment", () => {
    render(<PreviewCard template={tpl} picks={DEFAULT_PICKS} />);
    const character = useCharacterImageMock.mock.calls[0][0];
    expect(character).toMatchObject({
      tenant: "t1",
      region: "GMS",
      majorVersion: 83,
      minorVersion: 1,
      skinColor: 1,
      hair: 30032, // 30030 + digit 2
      face: 20000,
      gender: 0,
      equipment: { "-5": 1041002, "-11": 1302000 },
    });
    const options = useCharacterImageMock.mock.calls[0][1];
    expect(options).toMatchObject({ stance: "stand1", resize: 2 });
  });

  it("renders the composited image and the worn-equipment icon strip", () => {
    render(<PreviewCard template={tpl} picks={DEFAULT_PICKS} />);
    expect(screen.getByRole("img", { name: /live preview/i })).toHaveAttribute(
      "src",
      "/api/assets/t1/GMS/83.1/character/abc.png",
    );
    // first-of-pool only: tops + weapons = 2 worn icons
    expect(screen.getAllByTestId("worn-icon")).toHaveLength(2);
  });

  it("shows the error + retry state when the render fails", () => {
    const refetch = vi.fn();
    useCharacterImageMock.mockReturnValue({
      imageUrl: undefined,
      isLoading: false,
      isError: true,
      refetch,
    });
    render(<PreviewCard template={tpl} picks={DEFAULT_PICKS} />);
    expect(screen.getByText(/preview failed/i)).toBeInTheDocument();
    screen.getByRole("button", { name: /retry/i }).click();
    expect(refetch).toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `source ~/.nvm/nvm.sh && nvm use 22 >/dev/null 2>&1 && npm run test -- src/components/features/characters/templates/__tests__/PreviewCard.test.tsx`
Expected: FAIL — `../PreviewCard` unresolved.

- [ ] **Step 3: Write `PreviewCard.tsx`**

```tsx
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { CharacterTemplate } from "@/types/models/template";
import type { MapleStoryCharacterData } from "@/types/models/maplestory";
import { useCharacterImage } from "@/lib/hooks/useCharacterImage";
import { useItemName } from "@/lib/hooks/api/useItemStrings";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { useTenant } from "@/context/tenant-context";
import type { PreviewPicks } from "./editorState";
import {
  buildPreviewLoadout,
  EQUIP_SLOT_BY_POOL,
  type EquipmentPoolKey,
} from "./previewLoadout";

interface PreviewCardProps {
  template: CharacterTemplate;
  picks: PreviewPicks;
}

function WornIcon({ id }: { id: number }) {
  const { activeTenant } = useTenant();
  const name = useItemName(String(id));
  if (!activeTenant) return null;
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <img
          data-testid="worn-icon"
          src={getAssetIconUrl(
            activeTenant.id,
            activeTenant.attributes.region,
            activeTenant.attributes.majorVersion,
            activeTenant.attributes.minorVersion,
            "item",
            id,
          )}
          alt={name.data ?? String(id)}
          width={28}
          height={28}
          loading="lazy"
          className="rounded border bg-muted/40 p-0.5 [image-rendering:pixelated]"
        />
      </TooltipTrigger>
      <TooltipContent>
        {name.data ?? "Unknown item"} · {id}
      </TooltipContent>
    </Tooltip>
  );
}

export function PreviewCard({ template, picks }: PreviewCardProps) {
  const { activeTenant } = useTenant();
  const loadout = buildPreviewLoadout(template, picks);

  const character: MapleStoryCharacterData = {
    id: "template-preview",
    name: "preview",
    level: 1,
    jobId: 0,
    hair: loadout.hair,
    face: loadout.face,
    skinColor: loadout.skin,
    gender: template.gender,
    equipment: loadout.equipment as MapleStoryCharacterData["equipment"],
    tenant: activeTenant?.id ?? "",
    region: activeTenant?.attributes.region ?? "",
    majorVersion: activeTenant?.attributes.majorVersion ?? 0,
    minorVersion: activeTenant?.attributes.minorVersion ?? 0,
  };

  const image = useCharacterImage(
    character,
    { stance: "stand1", resize: 2 },
    { enabled: !!activeTenant },
  );

  const wornIds = (
    Object.keys(EQUIP_SLOT_BY_POOL) as EquipmentPoolKey[]
  ).flatMap((pool) => (template[pool][0] !== undefined ? [template[pool][0]] : []));

  return (
    <TooltipProvider>
      <div className="rounded-lg border bg-card p-3 lg:sticky lg:top-4">
        <p className="text-xs font-medium text-muted-foreground">
          Live preview
        </p>
        <div className="mx-auto mt-2 flex h-[200px] w-[154px] items-end justify-center rounded-md bg-gradient-to-b from-primary/5 to-primary/15">
          {image.isLoading && <Skeleton className="h-[160px] w-[120px]" />}
          {image.isError && (
            <div className="flex flex-col items-center gap-2 pb-6 text-center">
              <p className="text-xs text-muted-foreground">Preview failed</p>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => void image.refetch()}
              >
                Retry
              </Button>
            </div>
          )}
          {!image.isLoading && !image.isError && image.imageUrl && (
            <img
              src={image.imageUrl}
              alt="Live preview of the selected template"
              width={192}
              height={256}
              className="max-h-full w-auto [image-rendering:pixelated] drop-shadow-[0_6px_4px_rgba(0,0,0,0.25)]"
            />
          )}
        </div>
        {wornIds.length > 0 && (
          <div className="mt-2 flex justify-center gap-1">
            {wornIds.map((id) => (
              <WornIcon key={id} id={id} />
            ))}
          </div>
        )}
        <p className="mt-2 text-center text-xs text-muted-foreground">
          Composited from the highlighted picks and first-of-pool equipment.
        </p>
      </div>
    </TooltipProvider>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Same command as Step 2. Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add src/components/features/characters/templates/PreviewCard.tsx src/components/features/characters/templates/__tests__/PreviewCard.test.tsx
git commit -m "feat(task-177): sticky live preview card"
```

### Task 15: `SaveBar` + `CharacterTemplatesEditor` assembly

**Files:**
- Create: `src/components/features/characters/templates/SaveBar.tsx`
- Create: `src/components/features/characters/templates/CharacterTemplatesEditor.tsx`
- Test: `src/components/features/characters/templates/__tests__/CharacterTemplatesEditor.test.tsx`

**Interfaces:**
- Consumes: everything from Tasks 1–14; `useSearchParams` (`react-router-dom`); `EmptyState`, `ErrorDisplay`, `FormSkeleton` from `@/components/common`.
- Produces:
  - `interface TemplatesEditorAdapter { templates: CharacterTemplate[] | undefined; isLoading: boolean; error: Error | null; save: (templates: CharacterTemplate[], onSuccess: () => void) => void; isSaving: boolean }` — exported from `CharacterTemplatesEditor.tsx`.
  - `<CharacterTemplatesEditor adapter />` props `{ adapter: TemplatesEditorAdapter }`.
  - `<SaveBar dirty isSaving onSave onDiscard />` props `{ dirty: boolean; isSaving: boolean; onSave: () => void; onDiscard: () => void }` (Discard confirm dialog owned inside; onDiscard fires post-confirm).
  - Page wrappers (Task 16) consume the adapter type + editor.

- [ ] **Step 1: Write the failing test**

`src/components/features/characters/templates/__tests__/CharacterTemplatesEditor.test.tsx`:

```tsx
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter, Route, Routes, useSearchParams } from "react-router-dom";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

// Sections with their own data needs are exercised in their own suites; stub
// them here so the editor test focuses on assembly, URL sync, and save flow.
vi.mock("../IdentitySection", () => ({
  IdentitySection: ({ actions }: { actions?: React.ReactNode }) => (
    <div data-testid="identity">{actions}</div>
  ),
}));
vi.mock("../AppearancePoolSection", () => ({
  AppearancePoolSection: () => <div data-testid="appearance-pool" />,
}));
vi.mock("../EquipmentPoolSection", () => ({
  EquipmentPoolSection: () => <div data-testid="equipment-pool" />,
}));
vi.mock("../StartingKitSection", () => ({
  StartingKitSection: () => <div data-testid="starting-kit" />,
}));
vi.mock("../PreviewCard", () => ({
  PreviewCard: () => <div data-testid="preview-card" />,
}));

import { normalizeTemplate } from "../editorState";
import {
  CharacterTemplatesEditor,
  type TemplatesEditorAdapter,
} from "../CharacterTemplatesEditor";

function TplProbe() {
  const [params] = useSearchParams();
  return <output data-testid="tpl-param">{params.get("tpl") ?? ""}</output>;
}

function renderEditor(
  adapter: Partial<TemplatesEditorAdapter> = {},
  initialEntry = "/edit",
) {
  const full: TemplatesEditorAdapter = {
    templates: [
      normalizeTemplate({ jobIndex: 1, gender: 0 }),
      normalizeTemplate({ jobIndex: 1, gender: 1 }),
    ],
    isLoading: false,
    error: null,
    save: vi.fn(),
    isSaving: false,
    ...adapter,
  };
  render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route
          path="/edit"
          element={
            <>
              <CharacterTemplatesEditor adapter={full} />
              <TplProbe />
            </>
          }
        />
      </Routes>
    </MemoryRouter>,
  );
  return full;
}

beforeEach(() => vi.clearAllMocks());

describe("CharacterTemplatesEditor", () => {
  it("shows skeleton while loading and ErrorDisplay on error (both contexts identical)", () => {
    renderEditor({ templates: undefined, isLoading: true });
    expect(screen.getByTestId("form-skeleton")).toBeInTheDocument();
  });

  it("renders ErrorDisplay for load errors", () => {
    renderEditor({
      templates: undefined,
      isLoading: false,
      error: new Error("boom"),
    });
    expect(screen.getByTestId("error-display")).toBeInTheDocument();
  });

  it("empty configuration shows the explanatory empty state with Add", async () => {
    renderEditor({ templates: [] });
    expect(screen.getByTestId("empty-state")).toBeInTheDocument();
    await userEvent.click(
      screen.getByRole("button", { name: /add template/i }),
    );
    expect(screen.getByRole("tablist")).toBeInTheDocument();
  });

  it("restores selection from ?tpl= deep link", () => {
    renderEditor({}, "/edit?tpl=1");
    expect(
      screen.getByRole("tab", { name: "Adventurer · F" }),
    ).toHaveAttribute("aria-selected", "true");
  });

  it("clamps out-of-range ?tpl= to 0 and writes it back", async () => {
    renderEditor({}, "/edit?tpl=99");
    expect(
      screen.getByRole("tab", { name: "Adventurer · M" }),
    ).toHaveAttribute("aria-selected", "true");
    await waitFor(() =>
      expect(screen.getByTestId("tpl-param")).toHaveTextContent("0"),
    );
  });

  it("selecting a template syncs ?tpl=", async () => {
    renderEditor();
    await userEvent.click(screen.getByRole("tab", { name: "Adventurer · F" }));
    await waitFor(() =>
      expect(screen.getByTestId("tpl-param")).toHaveTextContent("1"),
    );
  });

  it("save passes the working array; success resets the dirty bar", async () => {
    const save = vi.fn((_tpls: unknown, onSuccess: () => void) => onSuccess());
    renderEditor({ save });
    // + New makes it dirty
    await userEvent.click(screen.getByRole("button", { name: /new/i }));
    expect(screen.getByText(/unsaved changes/i)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: /^save$/i }));
    expect(save).toHaveBeenCalled();
    expect((save.mock.calls[0][0] as unknown[]).length).toBe(3);
    expect(screen.getByText(/no unsaved changes/i)).toBeInTheDocument();
  });

  it("discard reverts to baseline after confirm", async () => {
    renderEditor();
    await userEvent.click(screen.getByRole("button", { name: /new/i }));
    expect(screen.getAllByRole("tab")).toHaveLength(3);
    await userEvent.click(screen.getByRole("button", { name: /discard/i }));
    await userEvent.click(
      screen.getByRole("button", { name: /discard changes/i }),
    );
    expect(screen.getAllByRole("tab")).toHaveLength(2);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `source ~/.nvm/nvm.sh && nvm use 22 >/dev/null 2>&1 && npm run test -- src/components/features/characters/templates/__tests__/CharacterTemplatesEditor.test.tsx`
Expected: FAIL — modules unresolved.

- [ ] **Step 3: Write `SaveBar.tsx`**

```tsx
import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";

interface SaveBarProps {
  dirty: boolean;
  isSaving: boolean;
  onSave: () => void;
  onDiscard: () => void;
}

export function SaveBar({ dirty, isSaving, onSave, onDiscard }: SaveBarProps) {
  const [confirmOpen, setConfirmOpen] = useState(false);

  return (
    <div className="sticky bottom-0 z-10 mt-4 flex items-center justify-between gap-3 rounded-lg border bg-background/95 p-3 backdrop-blur">
      <p
        className={
          dirty ? "text-sm font-medium" : "text-sm text-muted-foreground"
        }
      >
        {dirty ? "Unsaved changes" : "No unsaved changes"}
      </p>
      <div className="flex gap-2">
        <Button
          type="button"
          variant="outline"
          disabled={!dirty || isSaving}
          onClick={() => setConfirmOpen(true)}
        >
          Discard
        </Button>
        <Button type="button" disabled={!dirty || isSaving} onClick={onSave}>
          {isSaving ? "Saving…" : "Save"}
        </Button>
      </div>
      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Discard unsaved changes?</AlertDialogTitle>
            <AlertDialogDescription>
              All edits since the last save will be reverted.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep editing</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                setConfirmOpen(false);
                onDiscard();
              }}
            >
              Discard changes
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
```

- [ ] **Step 4: Write `CharacterTemplatesEditor.tsx`**

```tsx
import { useEffect, useMemo, useReducer } from "react";
import { useSearchParams } from "react-router-dom";
import type { CharacterTemplate } from "@/types/models/template";
import { EmptyState, ErrorDisplay, FormSkeleton } from "@/components/common";
import {
  editorReducer,
  initialEditorState,
  isDirty,
  picksFor,
  type PreviewPicks,
  type AppearancePoolKey,
} from "./editorState";
import { templateLabels } from "./jobNames";
import { TemplateSelector } from "./TemplateSelector";
import { TemplateActionsMenu } from "./TemplateActionsMenu";
import { IdentitySection } from "./IdentitySection";
import { AppearancePoolSection } from "./AppearancePoolSection";
import { AppearanceBrowserDialog } from "./AppearanceBrowserDialog";
import { EquipmentPoolSection } from "./EquipmentPoolSection";
import { StartingKitSection } from "./StartingKitSection";
import { PreviewCard } from "./PreviewCard";
import { SaveBar } from "./SaveBar";
import type { EquipmentPoolKey } from "./previewLoadout";

export interface TemplatesEditorAdapter {
  templates: CharacterTemplate[] | undefined;
  isLoading: boolean;
  error: Error | null;
  /** Fire the context's PATCH; call onSuccess only when it lands. */
  save: (templates: CharacterTemplate[], onSuccess: () => void) => void;
  isSaving: boolean;
}

interface CharacterTemplatesEditorProps {
  adapter: TemplatesEditorAdapter;
}

const APPEARANCE_SECTIONS: { dimension: AppearancePoolKey; title: string }[] = [
  { dimension: "faces", title: "Faces" },
  { dimension: "hairs", title: "Hairs" },
  { dimension: "hairColors", title: "Hair colors" },
  { dimension: "skinColors", title: "Skin tones" },
];

const EQUIPMENT_SECTIONS: { poolKey: EquipmentPoolKey; title: string }[] = [
  { poolKey: "tops", title: "Tops" },
  { poolKey: "bottoms", title: "Bottoms" },
  { poolKey: "shoes", title: "Shoes" },
  { poolKey: "weapons", title: "Weapons" },
];

export function CharacterTemplatesEditor({
  adapter,
}: CharacterTemplatesEditorProps) {
  const [state, dispatch] = useReducer(editorReducer, undefined, initialEditorState);
  const [searchParams, setSearchParams] = useSearchParams();

  // Seed once per data arrival while pristine. Never clobber dirty edits:
  // post-save refetches land while templates === baseline, so reloading then
  // is a no-op for content; user edits keep the working copy authoritative.
  useEffect(() => {
    if (adapter.templates && !state.loaded) {
      dispatch({ type: "load", templates: adapter.templates });
    }
  }, [adapter.templates, state.loaded]);

  // URL → selection on load; clamp and write back invalid values.
  useEffect(() => {
    if (!state.loaded) return;
    const raw = searchParams.get("tpl") ?? "0";
    const parsed = Number.parseInt(raw, 10);
    const clamped = Number.isFinite(parsed)
      ? Math.min(Math.max(parsed, 0), Math.max(state.templates.length - 1, 0))
      : 0;
    if (clamped !== state.selectedIndex) {
      dispatch({ type: "select", index: clamped });
    }
    if (String(clamped) !== raw) {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          next.set("tpl", String(clamped));
          return next;
        },
        { replace: true },
      );
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- run on load + template-count changes only
  }, [state.loaded, state.templates.length]);

  const syncSelection = (index: number) => {
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        next.set("tpl", String(index));
        return next;
      },
      { replace: true },
    );
  };

  const select = (index: number) => {
    dispatch({ type: "select", index });
    syncSelection(index);
  };
  const addTemplate = () => {
    dispatch({ type: "addTemplate" });
    syncSelection(state.templates.length); // new template's index
  };
  const duplicateTemplate = () => {
    dispatch({ type: "duplicateTemplate" });
    syncSelection(state.templates.length);
  };
  const removeTemplate = () => {
    dispatch({ type: "removeTemplate" });
    syncSelection(
      Math.min(state.selectedIndex, Math.max(state.templates.length - 2, 0)),
    );
  };

  const dirty = useMemo(() => isDirty(state), [state]);

  if (adapter.isLoading || (!state.loaded && !adapter.error)) {
    return <FormSkeleton fields={6} data-testid="form-skeleton" />;
  }
  if (adapter.error) {
    return <ErrorDisplay error={adapter.error} />;
  }

  const template = state.templates[state.selectedIndex];
  const picks = picksFor(state, state.selectedIndex);
  const labels = templateLabels(state.templates);

  if (state.templates.length === 0) {
    return (
      <EmptyState
        title="No character templates"
        description="Templates define which classes, looks, and starting gear players can pick at character creation. Add one to get started."
        action={{ label: "Add template", onClick: addTemplate }}
      />
    );
  }

  return (
    <div className="space-y-4">
      <TemplateSelector
        templates={state.templates}
        selectedIndex={state.selectedIndex}
        onSelect={select}
        onAdd={addTemplate}
      />
      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_252px]">
        <div className="order-2 space-y-6 lg:order-1">
          {template && (
            <>
              <IdentitySection
                template={template}
                onSetIdentity={(field, value) =>
                  dispatch({ type: "setIdentity", field, value })
                }
                actions={
                  <TemplateActionsMenu
                    label={labels[state.selectedIndex] ?? ""}
                    onDuplicate={duplicateTemplate}
                    onRemove={removeTemplate}
                  />
                }
              />
              {APPEARANCE_SECTIONS.map(({ dimension, title }) => (
                <AppearancePoolSection
                  key={dimension}
                  dimension={dimension}
                  title={title}
                  template={template}
                  picks={picks}
                  onPick={(pick: keyof PreviewPicks, idx: number) =>
                    dispatch({ type: "setPreviewPick", pick, value: idx })
                  }
                  onRemoveEntry={(entryIndex) =>
                    dispatch({ type: "removePoolEntry", pool: dimension, entryIndex })
                  }
                  renderAddDialog={(open, onOpenChange) => (
                    <AppearanceBrowserDialog
                      dimension={dimension}
                      template={template}
                      picks={picks}
                      open={open}
                      onOpenChange={onOpenChange}
                      onAdd={(id) =>
                        dispatch({ type: "addPoolEntry", pool: dimension, id })
                      }
                    />
                  )}
                />
              ))}
              {EQUIPMENT_SECTIONS.map(({ poolKey, title }) => (
                <EquipmentPoolSection
                  key={poolKey}
                  poolKey={poolKey}
                  title={title}
                  ids={template[poolKey]}
                  onAdd={(id) =>
                    dispatch({ type: "addPoolEntry", pool: poolKey, id })
                  }
                  onRemove={(entryIndex) =>
                    dispatch({ type: "removePoolEntry", pool: poolKey, entryIndex })
                  }
                />
              ))}
              <StartingKitSection
                items={template.items}
                skills={template.skills}
                onAddItem={(id) =>
                  dispatch({ type: "addPoolEntry", pool: "items", id })
                }
                onRemoveItem={(entryIndex) =>
                  dispatch({ type: "removePoolEntry", pool: "items", entryIndex })
                }
                onAddSkill={(id) =>
                  dispatch({ type: "addPoolEntry", pool: "skills", id })
                }
                onRemoveSkill={(entryIndex) =>
                  dispatch({ type: "removePoolEntry", pool: "skills", entryIndex })
                }
              />
            </>
          )}
        </div>
        <div className="order-1 lg:order-2">
          {template && <PreviewCard template={template} picks={picks} />}
        </div>
      </div>
      <SaveBar
        dirty={dirty}
        isSaving={adapter.isSaving}
        onSave={() =>
          adapter.save(state.templates, () => dispatch({ type: "savedOk" }))
        }
        onDiscard={() => dispatch({ type: "discard" })}
      />
    </div>
  );
}
```

Note: `FormSkeleton` may not accept `data-testid` passthrough — check `src/components/common/FormSkeleton.tsx`; if not, wrap it: `<div data-testid="form-skeleton"><FormSkeleton fields={6} /></div>`.

- [ ] **Step 5: Run test to verify it passes**

Same command as Step 2. Expected: PASS (8 tests).

- [ ] **Step 6: Commit**

```bash
git add src/components/features/characters/templates/SaveBar.tsx src/components/features/characters/templates/CharacterTemplatesEditor.tsx src/components/features/characters/templates/__tests__/CharacterTemplatesEditor.test.tsx
git commit -m "feat(task-177): shared character templates editor assembly"
```

### Task 16: Page wrappers + delete the duplicated forms

**Files:**
- Modify: `src/pages/TenantsCharacterTemplatesPage.tsx`
- Modify: `src/pages/TemplatesCharacterTemplatesPage.tsx`
- Delete: `src/pages/tenants-character-templates-form.tsx`
- Delete: `src/pages/templates-character-templates-form.tsx`
- Test: `src/pages/__tests__/TenantsCharacterTemplatesPage.test.tsx`
- Test: `src/pages/__tests__/TemplatesCharacterTemplatesPage.test.tsx`

**Interfaces:**
- Consumes: `CharacterTemplatesEditor` + `TemplatesEditorAdapter` (Task 15); `useTenantConfiguration`/`useUpdateTenantConfiguration` (`@/lib/hooks/api/useTenants`); `useTemplate`/`useUpdateTemplate` (`@/lib/hooks/api/useTemplates`); `toast` from `sonner`.
- Produces: the two route components unchanged in name/route; PATCH payloads must spread `...attributes.characters` so sibling keys (presets) survive — this is the exact shape the old forms sent.

- [ ] **Step 1: Write the failing tests**

`src/pages/__tests__/TenantsCharacterTemplatesPage.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@/components/features/tenants/TenantDetailLayout", () => ({
  TenantDetailLayout: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="tenant-layout">{children}</div>
  ),
}));

// Capture the adapter handed to the shared editor.
const editorMock = vi.fn();
vi.mock(
  "@/components/features/characters/templates/CharacterTemplatesEditor",
  () => ({
    CharacterTemplatesEditor: (props: unknown) => {
      editorMock(props);
      return <div data-testid="shared-editor" />;
    },
  }),
);

const useTenantConfigurationMock = vi.fn();
const mutateMock = vi.fn();
vi.mock("@/lib/hooks/api/useTenants", () => ({
  useTenantConfiguration: (...a: unknown[]) => useTenantConfigurationMock(...a),
  useUpdateTenantConfiguration: () => ({
    mutate: mutateMock,
    isPending: false,
  }),
}));

import { TenantsCharacterTemplatesPage } from "../TenantsCharacterTemplatesPage";

const templates = [
  {
    jobIndex: 1, subJobIndex: 0, gender: 0, mapId: 0,
    faces: [20000], hairs: [], hairColors: [], skinColors: [],
    tops: [], bottoms: [], shoes: [], weapons: [], items: [], skills: [],
  },
];
const presets = [{ attributes: { name: "keep-me" } }];
const tenant = {
  id: "t1",
  attributes: {
    region: "GMS", majorVersion: 83, minorVersion: 1, usesPin: false,
    characters: { templates, presets },
    npcs: [], socket: { handlers: [], writers: [] }, worlds: [],
  },
};

beforeEach(() => {
  vi.clearAllMocks();
  useTenantConfigurationMock.mockReturnValue({
    data: tenant,
    isLoading: false,
    error: null,
  });
});

describe("TenantsCharacterTemplatesPage", () => {
  it("renders the shared editor inside the tenant layout with adapter data", () => {
    render(
      <MemoryRouter initialEntries={["/tenants/t1/character/templates"]}>
        <TenantsCharacterTemplatesPage />
      </MemoryRouter>,
    );
    expect(screen.getByTestId("tenant-layout")).toBeInTheDocument();
    expect(screen.getByTestId("shared-editor")).toBeInTheDocument();
    const { adapter } = editorMock.mock.calls[0][0] as {
      adapter: { templates: unknown };
    };
    expect(adapter.templates).toEqual(templates);
  });

  it("save PATCHes the full characters object, preserving presets", () => {
    render(
      <MemoryRouter initialEntries={["/tenants/t1/character/templates"]}>
        <TenantsCharacterTemplatesPage />
      </MemoryRouter>,
    );
    const { adapter } = editorMock.mock.calls[0][0] as {
      adapter: {
        save: (t: unknown[], onSuccess: () => void) => void;
      };
    };
    const newTemplates = [...templates, { jobIndex: 2 }];
    adapter.save(newTemplates, vi.fn());
    expect(mutateMock).toHaveBeenCalledWith(
      {
        tenant,
        updates: {
          characters: { templates: newTemplates, presets },
        },
      },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
  });
});
```

`src/pages/__tests__/TemplatesCharacterTemplatesPage.test.tsx` (same pattern, template context):

```tsx
import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@/components/features/templates/TemplateDetailLayout", () => ({
  TemplateDetailLayout: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="template-layout">{children}</div>
  ),
}));

const editorMock = vi.fn();
vi.mock(
  "@/components/features/characters/templates/CharacterTemplatesEditor",
  () => ({
    CharacterTemplatesEditor: (props: unknown) => {
      editorMock(props);
      return <div data-testid="shared-editor" />;
    },
  }),
);

const useTemplateMock = vi.fn();
const mutateMock = vi.fn();
vi.mock("@/lib/hooks/api/useTemplates", () => ({
  useTemplate: (...a: unknown[]) => useTemplateMock(...a),
  useUpdateTemplate: () => ({ mutate: mutateMock, isPending: false }),
}));

import { TemplatesCharacterTemplatesPage } from "../TemplatesCharacterTemplatesPage";

const templates = [
  {
    jobIndex: 0, subJobIndex: 0, gender: 1, mapId: 0,
    faces: [], hairs: [], hairColors: [], skinColors: [],
    tops: [], bottoms: [], shoes: [], weapons: [], items: [], skills: [],
  },
];
const presets = [{ attributes: { name: "keep-me" } }];
const template = {
  id: "tmpl-1",
  attributes: {
    region: "GMS", majorVersion: 83, minorVersion: 1, usesPin: false,
    characters: { templates, presets },
    npcs: [], socket: { handlers: [], writers: [] }, worlds: [],
  },
};

beforeEach(() => {
  vi.clearAllMocks();
  useTemplateMock.mockReturnValue({
    data: template,
    isLoading: false,
    error: null,
  });
});

describe("TemplatesCharacterTemplatesPage", () => {
  it("renders the shared editor inside the template layout", () => {
    render(
      <MemoryRouter initialEntries={["/templates/tmpl-1/character/templates"]}>
        <TemplatesCharacterTemplatesPage />
      </MemoryRouter>,
    );
    expect(screen.getByTestId("template-layout")).toBeInTheDocument();
    expect(screen.getByTestId("shared-editor")).toBeInTheDocument();
  });

  it("save PATCHes by id, preserving presets", () => {
    render(
      <MemoryRouter initialEntries={["/templates/tmpl-1/character/templates"]}>
        <TemplatesCharacterTemplatesPage />
      </MemoryRouter>,
    );
    const { adapter } = editorMock.mock.calls[0][0] as {
      adapter: { save: (t: unknown[], onSuccess: () => void) => void };
    };
    adapter.save(templates, vi.fn());
    expect(mutateMock).toHaveBeenCalledWith(
      {
        id: "tmpl-1",
        updates: { characters: { templates, presets } },
      },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `source ~/.nvm/nvm.sh && nvm use 22 >/dev/null 2>&1 && npm run test -- src/pages/__tests__/TenantsCharacterTemplatesPage.test.tsx src/pages/__tests__/TemplatesCharacterTemplatesPage.test.tsx`
Expected: FAIL — pages still render the old `TemplatesForm`.

- [ ] **Step 3: Rewrite `TenantsCharacterTemplatesPage.tsx`**

```tsx
import { useParams } from "react-router-dom";
import { toast } from "sonner";
import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";
import {
  CharacterTemplatesEditor,
  type TemplatesEditorAdapter,
} from "@/components/features/characters/templates/CharacterTemplatesEditor";
import {
  useTenantConfiguration,
  useUpdateTenantConfiguration,
} from "@/lib/hooks/api/useTenants";

export function TenantsCharacterTemplatesPage() {
  const { id } = useParams();
  const tenantQuery = useTenantConfiguration(id ?? "");
  const updateTenantConfig = useUpdateTenantConfiguration();
  const tenant = tenantQuery.data;

  const adapter: TemplatesEditorAdapter = {
    templates: tenant?.attributes.characters.templates,
    isLoading: tenantQuery.isLoading,
    error: tenantQuery.error ?? null,
    isSaving: updateTenantConfig.isPending,
    save: (templates, onSuccess) => {
      if (!tenant) return;
      updateTenantConfig.mutate(
        {
          tenant,
          updates: {
            characters: { ...tenant.attributes.characters, templates },
          },
        },
        {
          onSuccess: () => {
            toast.success("Successfully saved tenant configuration.");
            onSuccess();
          },
          onError: (error) =>
            toast.error(
              `Failed to update tenant configuration: ${error.message}`,
            ),
        },
      );
    },
  };

  return (
    <TenantDetailLayout>
      <CharacterTemplatesEditor adapter={adapter} />
    </TenantDetailLayout>
  );
}
```

- [ ] **Step 4: Rewrite `TemplatesCharacterTemplatesPage.tsx`**

```tsx
import { useParams } from "react-router-dom";
import { toast } from "sonner";
import { TemplateDetailLayout } from "@/components/features/templates/TemplateDetailLayout";
import {
  CharacterTemplatesEditor,
  type TemplatesEditorAdapter,
} from "@/components/features/characters/templates/CharacterTemplatesEditor";
import { useTemplate, useUpdateTemplate } from "@/lib/hooks/api/useTemplates";

export function TemplatesCharacterTemplatesPage() {
  const { id } = useParams();
  const templateQuery = useTemplate(String(id ?? ""));
  const updateTemplate = useUpdateTemplate();
  const template = templateQuery.data;

  const adapter: TemplatesEditorAdapter = {
    templates: template?.attributes.characters.templates,
    isLoading: templateQuery.isLoading,
    error: templateQuery.error ?? null,
    isSaving: updateTemplate.isPending,
    save: (templates, onSuccess) => {
      if (!template) return;
      updateTemplate.mutate(
        {
          id: template.id,
          updates: {
            characters: { ...template.attributes.characters, templates },
          },
        },
        {
          onSuccess: () => {
            toast.success("Successfully saved template.");
            onSuccess();
          },
          onError: (error) =>
            toast.error(`Failed to update template: ${error.message}`),
        },
      );
    },
  };

  return (
    <TemplateDetailLayout>
      <CharacterTemplatesEditor adapter={adapter} />
    </TemplateDetailLayout>
  );
}
```

- [ ] **Step 5: Delete the old forms and verify nothing references them**

```bash
rm src/pages/tenants-character-templates-form.tsx src/pages/templates-character-templates-form.tsx
grep -rn "character-templates-form" src/
```

Expected: grep returns nothing. (FR-1.3: the inline `CharacterTemplate` redeclaration dies with the template-side file.)

- [ ] **Step 6: Run tests to verify they pass**

Same command as Step 2. Expected: PASS (4 tests). Then run the whole suite: `npm run test` — Expected: PASS, no regressions.

- [ ] **Step 7: Commit**

```bash
git add -A src/pages
git commit -m "feat(task-177): wire shared editor into both pages; delete duplicated forms"
```

### Task 17: Full verification gates + visual tuning

**Files:**
- Possibly modify: `src/components/features/characters/templates/AppearanceThumb.tsx` (crop offsets), any file lint fix mode touches.

- [ ] **Step 1: Run the atlas-ui gates**

From `services/atlas-ui`:

```bash
source ~/.nvm/nvm.sh && nvm use 22 >/dev/null 2>&1
npm run test
npm run lint
npm run build
```

Expected: all clean. `npm run lint` must introduce **no new** errors versus the repo baseline (pre-existing lint debt is tolerated; new files must be clean). `npm run build` type-checks the new tests via `tsc -b`.

- [ ] **Step 2: Repo-root lint guard**

From the repo root (worktree root):

```bash
tools/lint.sh          # fix mode first — rewrites formatting in place
git diff --stat        # review what fix mode touched; only formatting expected
tools/lint.sh --check
```

Expected: `--check` exits 0. If fix mode reformatted files, commit them: `git add -A && git commit -m "style(task-177): lint fix mode"`.

- [ ] **Step 3: Visual smoke + crop tuning (dev server)**

```bash
cd services/atlas-ui && npm run dev
```

Open `http://localhost:5173/tenants/<tenant-id>/character/templates` against a live ingress (default proxy `localhost:8080`). Verify against the PRD acceptance list: selector labels + `?tpl=` deep link, + New / Duplicate / Remove, appearance thumbs crop the head region (tune `THUMB_OFFSET_X`/`THUMB_OFFSET_Y` in `AppearanceThumb.tsx` if the face isn't centered — prototype start `-74/-70`), browser dialog paging + gender filter, equipment combobox, live preview updates on picks, save/discard round-trip, light + dark themes. If offsets changed, commit: `git commit -am "feat(task-177): tune thumbnail crop offsets"`.

If no live environment is reachable, state that explicitly in the completion report — do NOT claim visual verification that didn't happen.

- [ ] **Step 4: Final commit + code review**

Verify the tree is clean (`git status`), then run `superpowers:requesting-code-review` (per repo rule: always before a PR). Reviewers: `plan-adherence-reviewer` + `frontend-guidelines-reviewer` (TS-only change — no Go reviewer needed). Findings go to `docs/tasks/task-177-character-templates-editor/audit.md`.

---

## Self-Review Notes (already applied)

- FR coverage: FR-1 → Tasks 15/16; FR-2 → Tasks 8/15; FR-3 → Tasks 9/15; FR-4 → Tasks 7/10; FR-5 → Tasks 3/4/11/12; FR-6 → Tasks 6/13; FR-7 → Task 13; FR-8 → Tasks 3/14; FR-9 → Tasks 2/15/16; FR-10 → global constraints + Task 17.
- Dropped by user decision (do NOT implement): selector thumbnails, Edit-as-JSON, preview id/value table, Creator-Preview mode, skill browser.
- Type consistency spot-checks: `TemplatesEditorAdapter.save(templates, onSuccess)` is used identically in Tasks 15 and 16; `PoolKey`/`AppearancePoolKey`/`PreviewPicks`/`IdentityField` names match across Tasks 2, 10, 11, 12, 15; `EquipmentPoolKey` comes from `previewLoadout.ts` (Task 3) everywhere.
- Known flex points called out inline: `DropdownMenuItem` variant prop (Task 9 note), `FormSkeleton` testid passthrough (Task 15 note), radix Select jsdom stubs (Task 10 note), load-more accumulation (Task 6 note). These are check-the-file-first notes, not TBDs — each has a concrete fallback spelled out.

