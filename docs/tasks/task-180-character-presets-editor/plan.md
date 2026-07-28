# Character Presets Editor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the two wholesale-duplicated raw-ID character-preset forms with one shared, adapter-parameterized visual editor (card-library landing + focused per-preset editor) under `src/components/features/characters/presets/`, reusing the task-177 building blocks.

**Architecture:** A new sibling folder `presets/` next to the landed task-177 `templates/`. One `CharacterPresetsEditor` container owns a `useReducer` working copy of the whole `presets` array (baseline-snapshot dirty tracking, the task-177 "D1" pattern — NOT react-hook-form), URL-syncs the selected preset via `?preset=<id|key>`, and drives the shared detail action bar. It renders a **card library** when no preset is selected and a **focused editor** (sections + sticky live preview) when one is. Each page wrapper (tenant / template) supplies a small adapter; the tenant adapter additionally exposes an `apply` capability. Zero backend changes, zero shell changes.

**Tech Stack:** React 19, TypeScript, Vite, Vitest + React Testing Library, TanStack React Query, react-router-dom `useSearchParams`, shadcn/ui + Tailwind, `useReducer`. Node 22 via nvm.

## Global Constraints

- **atlas-ui only.** No Go service changes, no seed-JSON changes, no routing/shell/layout changes. Page component export names (`TenantsCharacterPresetsPage`, `TemplatesCharacterPresetsPage`) MUST stay identical — `App.tsx:180-220,341,367` lazy-imports them by name.
- **State model:** `useReducer` + baseline snapshot; dirty = deep JSON compare vs. baseline. NOT react-hook-form. One working array for the whole `presets` list; free switching between presets never loses edits.
- **Save = one PATCH** of the full configuration through the adapter's mutation, spreading `attributes.characters` so the sibling `templates` array survives.
- **Types come only from `@/types/models/template`** (`CharacterPreset`, `CharacterPresetAttributes`, `CharacterPresetStatBlock`, `CharacterPresetEquipmentEntry`, `CharacterPresetInventoryEntry`, `CharacterPresetSkillEntry`). No inline preset-shape redeclaration survives. Validation reuses `presetSchema` from `@/lib/schemas/character-presets.schema`.
- **Unresolvable ids degrade, never block** — placeholder icon + numeric id, still editable/removable. The backend is the validator of record.
- **Every request carries tenant/region/version headers** via the existing API client (already true of every reused hook). Light + dark tokens only; all sprite imagery `image-rendering: pixelated`. Cards/menus/thumbnails/search keyboard operable; `prefers-reduced-motion` respected.
- **Appearance edits write directly to `attributes`** (single-value preset shape) — there is NO separate `previewPicks` layer (unlike task-177). The live preview composites straight from `attributes`.
- **Selection & `?preset=` are id/key-addressed, never index-addressed.** Each working preset carries a UI-only `key: string` (`p.id ?? "local-<n>"`), never persisted, never sent in PATCH, never part of the dirty compare.
- **Additive-only edits to task-177 shared files.** `AppearanceBrowserDialog`, `ItemRow`, and `ApplyPresetDialog` gain backward-compatible props with defaults that preserve every existing call site; task-177 tests stay green as the regression gate.
- **Verification gate (from PRD Acceptance):** from `services/atlas-ui` (nvm node 22): `npm run test`, `npm run lint` (no new errors vs. baseline), `npm run build` all clean; and `tools/lint.sh --check` clean from the repo/worktree root. New test files MUST type-check under the build (`tsc -b`).

**Reused task-177 modules (verified present, do NOT recreate):**
- `templates/AppearanceThumb.tsx` — thumbnail button (url, idLabel, ariaLabel, marked, onSelect). Add a `selected` prop in Task 4.
- `templates/MapPicker.tsx` — `<MapPicker value={number} onChange={(id:number)=>void} />` (verbatim).
- `templates/ItemSearchCombobox.tsx`, `templates/poolSearchConfig.ts` (`POOL_SEARCH_CONFIGS`, `SearchPoolKey`).
- `templates/ItemRow.tsx` — icon + name + mono id + remove ×. Gains optional `trailing` in Task 5.
- `templates/AppearanceBrowserDialog.tsx` — generalized in Task 4.
- `lib/utils/maplestory.ts`: `getDefaultSlotForTemplateId(templateId): number | null`, `synthesizeEquippedAssetsFromTemplateIds`.
- `services/api/characterRender.service.ts`: `filterEquipment(Record<string,number>): Record<string,number>`, `generateCharacterUrl(...)`, `isFemaleCosmeticId`, type `CharacterLoadout { skin; hair; face; equipment: Record<string,number>; gender?: number }`.
- `lib/hooks/useCharacterImage`, `lib/hooks/useSkillData`, `lib/hooks/api/useItemStrings` (`useItemName`), `lib/hooks/api/useItemNames` (`useItemNames`), `lib/hooks/api/useCosmetics` (`useFaceIds`,`useHairIds`), `lib/hooks/api/useMaps`, `lib/hooks/api/useAccounts` (`useAccountSearch(tenant, namePattern): Account[]`), `lib/hooks/api/useTenants` (`useTenant(id): TenantBasic`, `useTenantConfiguration`, `useUpdateTenantConfiguration`), `lib/hooks/api/useTemplates` (`useTemplate`, `useUpdateTemplate`).
- `components/DetailActionBarContext` (`useRegisterDetailActionBar`), `components/common` (`EmptyState`, `ErrorDisplay`, `FormSkeleton`), shadcn primitives, `context/tenant-context` (`useTenant()` → `{ activeTenant }`).

**Prototype:** `docs/tasks/task-180-character-presets-editor/prototype.html` — open the "A · Card library → editor" tab. Visual source of truth for layout/spacing/copy.

---

### Task 1: `presetJobs.ts` — curated jobId → name map

**Files:**
- Create: `services/atlas-ui/src/components/features/characters/presets/presetJobs.ts`
- Test: `services/atlas-ui/src/components/features/characters/presets/__tests__/presetJobs.test.ts`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `PRESET_JOBS: { id: number; name: string }[]` — curated, ascending by id.
  - `jobLabel(id: number): string` — mapped name, else `` `Job ${id}` ``.

- [ ] **Step 1: Write the failing test**

```ts
import { describe, it, expect } from "vitest";
import { PRESET_JOBS, jobLabel } from "../presetJobs";

describe("presetJobs", () => {
  it("maps known job ids to names", () => {
    expect(jobLabel(0)).toBe("Beginner");
    expect(jobLabel(100)).toBe("Warrior");
    expect(jobLabel(900)).toBe("GM");
  });

  it("falls back to Job <id> for unknown ids", () => {
    expect(jobLabel(123456)).toBe("Job 123456");
  });

  it("exposes an ascending, de-duplicated curated list", () => {
    const ids = PRESET_JOBS.map((j) => j.id);
    expect(ids).toEqual([...ids].sort((a, b) => a - b));
    expect(new Set(ids).size).toBe(ids.length);
    expect(PRESET_JOBS.find((j) => j.id === 0)?.name).toBe("Beginner");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/presetJobs.test.ts`
Expected: FAIL — cannot resolve `../presetJobs`.

- [ ] **Step 3: Write minimal implementation**

```ts
// Curated jobId → display name. atlas-data exposes no job-name endpoint
// (jobs.service only serves /jobs/{id}/skills), so names are a client map.
// Covers seed-data ids + common advances; the backend is the validator of
// record, so any unmapped id renders as `Job <id>` and remains selectable.
export const PRESET_JOBS: { id: number; name: string }[] = [
  { id: 0, name: "Beginner" },
  { id: 100, name: "Warrior" },
  { id: 110, name: "Fighter" },
  { id: 111, name: "Crusader" },
  { id: 112, name: "Hero" },
  { id: 120, name: "Page" },
  { id: 121, name: "White Knight" },
  { id: 122, name: "Paladin" },
  { id: 130, name: "Spearman" },
  { id: 131, name: "Dragon Knight" },
  { id: 132, name: "Dark Knight" },
  { id: 200, name: "Magician" },
  { id: 210, name: "Fire/Poison Wizard" },
  { id: 211, name: "Fire/Poison Mage" },
  { id: 212, name: "Fire/Poison Archmage" },
  { id: 220, name: "Ice/Lightning Wizard" },
  { id: 221, name: "Ice/Lightning Mage" },
  { id: 222, name: "Ice/Lightning Archmage" },
  { id: 230, name: "Cleric" },
  { id: 231, name: "Priest" },
  { id: 232, name: "Bishop" },
  { id: 300, name: "Bowman" },
  { id: 310, name: "Hunter" },
  { id: 311, name: "Ranger" },
  { id: 312, name: "Bowmaster" },
  { id: 320, name: "Crossbowman" },
  { id: 321, name: "Sniper" },
  { id: 322, name: "Marksman" },
  { id: 400, name: "Thief" },
  { id: 410, name: "Assassin" },
  { id: 411, name: "Hermit" },
  { id: 412, name: "Night Lord" },
  { id: 420, name: "Bandit" },
  { id: 421, name: "Chief Bandit" },
  { id: 422, name: "Shadower" },
  { id: 500, name: "Pirate" },
  { id: 510, name: "Brawler" },
  { id: 511, name: "Marauder" },
  { id: 512, name: "Buccaneer" },
  { id: 520, name: "Gunslinger" },
  { id: 521, name: "Outlaw" },
  { id: 522, name: "Corsair" },
  { id: 900, name: "GM" },
  { id: 910, name: "SuperGM" },
];

const NAME_BY_ID = new Map(PRESET_JOBS.map((j) => [j.id, j.name]));

export function jobLabel(id: number): string {
  return NAME_BY_ID.get(id) ?? `Job ${id}`;
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/presetJobs.test.ts`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/presets/presetJobs.ts services/atlas-ui/src/components/features/characters/presets/__tests__/presetJobs.test.ts
git commit -m "feat(task-180): curated preset jobId→name map"
```

---

### Task 2: `presetEditorState.ts` — reducer + key/dirty/normalize core

The correctness heart of the feature. Mirrors `templates/editorState.ts` but key-addressed, with appearance/stats written directly to `attributes` and no `previewPicks` layer.

**Files:**
- Create: `services/atlas-ui/src/components/features/characters/presets/presetEditorState.ts`
- Test: `services/atlas-ui/src/components/features/characters/presets/__tests__/presetEditorState.test.ts`

**Interfaces:**
- Consumes: `CharacterPreset`, `CharacterPresetAttributes` from `@/types/models/template`.
- Produces:
  - Types: `WorkingPreset = { key: string; id?: string; attributes: CharacterPresetAttributes }`, `PresetEditorState`, `PresetEditorAction`, `PresetFieldPath`.
  - `DEFAULT_PRESET_ATTRIBUTES: CharacterPresetAttributes`.
  - `normalizePreset(raw): CharacterPresetAttributes`.
  - `initialPresetEditorState(): PresetEditorState`.
  - `presetReducer(state, action): PresetEditorState`.
  - `isDirty(state): boolean` (projects away `key`).
  - `projectForSave(state): CharacterPreset[]` (`{ id?, attributes }`, no `key`).
  - `selectedPreset(state): WorkingPreset | null`.
  - `presetDirty(state, key): boolean` (per-preset diff vs. baseline, for library dirty-dots).

- [ ] **Step 1: Write the failing test**

```ts
import { describe, it, expect } from "vitest";
import {
  DEFAULT_PRESET_ATTRIBUTES,
  initialPresetEditorState,
  presetReducer,
  isDirty,
  projectForSave,
  selectedPreset,
  presetDirty,
  normalizePreset,
  type PresetEditorState,
} from "../presetEditorState";
import type { CharacterPreset } from "@/types/models/template";

const preset = (id: string | undefined, name: string): CharacterPreset => ({
  id,
  attributes: { ...DEFAULT_PRESET_ATTRIBUTES, name },
});

function loaded(presets: CharacterPreset[]): PresetEditorState {
  return presetReducer(initialPresetEditorState(), { type: "load", presets });
}

describe("presetEditorState", () => {
  it("load assigns a stable key (id when present, local-<n> otherwise) and is not dirty", () => {
    const s = loaded([preset("a1", "One"), preset(undefined, "Two")]);
    expect(s.presets[0].key).toBe("a1");
    expect(s.presets[1].key).toBe("local-0");
    expect(s.loaded).toBe(true);
    expect(s.selectedKey).toBeNull();
    expect(isDirty(s)).toBe(false);
  });

  it("dirty compare ignores the UI-only key", () => {
    let s = loaded([preset("a1", "One")]);
    // mutate key directly to prove it is excluded
    s = { ...s, presets: [{ ...s.presets[0], key: "different" }] };
    expect(isDirty(s)).toBe(false);
  });

  it("projectForSave emits { id?, attributes } with no key", () => {
    const s = loaded([preset("a1", "One"), preset(undefined, "Two")]);
    const out = projectForSave(s);
    expect(out).toEqual([
      { id: "a1", attributes: { ...DEFAULT_PRESET_ATTRIBUTES, name: "One" } },
      { attributes: { ...DEFAULT_PRESET_ATTRIBUTES, name: "Two" } },
    ]);
    expect(Object.keys(out[0])).not.toContain("key");
  });

  it("addPreset appends defaults, selects the new key, marks dirty", () => {
    let s = loaded([preset("a1", "One")]);
    s = presetReducer(s, { type: "addPreset" });
    expect(s.presets).toHaveLength(2);
    expect(s.presets[1].attributes.name).toBe("New preset");
    expect(s.selectedKey).toBe(s.presets[1].key);
    expect(s.presets[1].key).toMatch(/^local-/);
    expect(isDirty(s)).toBe(true);
  });

  it("duplicatePreset deep-copies, gives a fresh key/no id, selects it", () => {
    let s = loaded([preset("a1", "One")]);
    s = presetReducer(s, { type: "duplicatePreset", key: "a1" });
    expect(s.presets).toHaveLength(2);
    expect(s.presets[1].id).toBeUndefined();
    expect(s.presets[1].key).not.toBe("a1");
    expect(s.presets[1].attributes.name).toBe("One");
    // deep copy: mutating source arrays does not touch the copy
    s.presets[0].attributes.tags.push("x");
    expect(s.presets[1].attributes.tags).toEqual([]);
    expect(s.selectedKey).toBe(s.presets[1].key);
  });

  it("removePreset drops the row and clears selection to library", () => {
    let s = loaded([preset("a1", "One"), preset("b2", "Two")]);
    s = presetReducer(s, { type: "select", key: "a1" });
    s = presetReducer(s, { type: "removePreset", key: "a1" });
    expect(s.presets.map((p) => p.id)).toEqual(["b2"]);
    expect(s.selectedKey).toBeNull();
  });

  it("setField writes appearance/stat/identity directly to attributes", () => {
    let s = loaded([preset("a1", "One")]);
    s = presetReducer(s, { type: "select", key: "a1" });
    s = presetReducer(s, { type: "setField", key: "a1", path: "face", value: 21000 });
    s = presetReducer(s, { type: "setField", key: "a1", path: "gender", value: 1 });
    s = presetReducer(s, { type: "setField", key: "a1", path: "stats.str", value: 12 });
    expect(s.presets[0].attributes.face).toBe(21000);
    expect(s.presets[0].attributes.gender).toBe(1);
    expect(s.presets[0].attributes.stats.str).toBe(12);
    expect(isDirty(s)).toBe(true);
  });

  it("equipment add/remove/avg edits the working preset", () => {
    let s = loaded([preset("a1", "One")]);
    s = presetReducer(s, { type: "addEquip", key: "a1", templateId: 1040002 });
    expect(s.presets[0].attributes.equipment).toEqual([
      { templateId: 1040002, useAverageStats: true },
    ]);
    s = presetReducer(s, { type: "setEquipAvg", key: "a1", index: 0, value: false });
    expect(s.presets[0].attributes.equipment[0].useAverageStats).toBe(false);
    s = presetReducer(s, { type: "removeEquip", key: "a1", index: 0 });
    expect(s.presets[0].attributes.equipment).toEqual([]);
  });

  it("inventory + skills add/remove/qty/level edit the working preset", () => {
    let s = loaded([preset("a1", "One")]);
    s = presetReducer(s, { type: "addInventory", key: "a1", templateId: 2000000 });
    s = presetReducer(s, { type: "setInventoryQty", key: "a1", index: 0, value: 5 });
    expect(s.presets[0].attributes.inventory).toEqual([
      { templateId: 2000000, quantity: 5 },
    ]);
    s = presetReducer(s, { type: "addSkill", key: "a1", skillId: 1001004 });
    s = presetReducer(s, { type: "setSkillLevel", key: "a1", index: 0, value: 3 });
    expect(s.presets[0].attributes.skills).toEqual([
      { skillId: 1001004, level: 3 },
    ]);
    s = presetReducer(s, { type: "removeSkill", key: "a1", index: 0 });
    s = presetReducer(s, { type: "removeInventory", key: "a1", index: 0 });
    expect(s.presets[0].attributes.skills).toEqual([]);
    expect(s.presets[0].attributes.inventory).toEqual([]);
  });

  it("tags add/remove is case-preserving and de-duplicated", () => {
    let s = loaded([preset("a1", "One")]);
    s = presetReducer(s, { type: "addTag", key: "a1", tag: "PvP" });
    s = presetReducer(s, { type: "addTag", key: "a1", tag: "PvP" });
    expect(s.presets[0].attributes.tags).toEqual(["PvP"]);
    s = presetReducer(s, { type: "removeTag", key: "a1", tag: "PvP" });
    expect(s.presets[0].attributes.tags).toEqual([]);
  });

  it("discard restores baseline and returns to library", () => {
    let s = loaded([preset("a1", "One")]);
    s = presetReducer(s, { type: "select", key: "a1" });
    s = presetReducer(s, { type: "setField", key: "a1", path: "level", value: 99 });
    expect(isDirty(s)).toBe(true);
    s = presetReducer(s, { type: "discard" });
    expect(isDirty(s)).toBe(false);
    expect(s.presets[0].attributes.level).toBe(1);
    expect(s.selectedKey).toBeNull();
  });

  it("savedOk rebaselines to the current working copy", () => {
    let s = loaded([preset("a1", "One")]);
    s = presetReducer(s, { type: "setField", key: "a1", path: "level", value: 50 });
    expect(isDirty(s)).toBe(true);
    s = presetReducer(s, { type: "savedOk" });
    expect(isDirty(s)).toBe(false);
    expect(presetDirty(s, "a1")).toBe(false);
  });

  it("presetDirty reflects a single preset's diff vs baseline", () => {
    let s = loaded([preset("a1", "One"), preset("b2", "Two")]);
    s = presetReducer(s, { type: "setField", key: "b2", path: "level", value: 7 });
    expect(presetDirty(s, "a1")).toBe(false);
    expect(presetDirty(s, "b2")).toBe(true);
  });

  it("normalizePreset fills missing fields from defaults", () => {
    const attrs = normalizePreset({ name: "Partial" } as never);
    expect(attrs.jobId).toBe(0);
    expect(attrs.stats).toEqual(DEFAULT_PRESET_ATTRIBUTES.stats);
    expect(attrs.equipment).toEqual([]);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/presetEditorState.test.ts`
Expected: FAIL — cannot resolve `../presetEditorState`.

- [ ] **Step 3: Write minimal implementation**

```ts
import type {
  CharacterPreset,
  CharacterPresetAttributes,
} from "@/types/models/template";

export interface WorkingPreset {
  /** UI-only stable key. Never persisted, never sent, never in the dirty compare. */
  key: string;
  id?: string;
  attributes: CharacterPresetAttributes;
}

export interface PresetEditorState {
  presets: WorkingPreset[];
  /** Baseline snapshot for dirty compare — projected shape (no key). */
  baseline: CharacterPreset[];
  /** null = library view; otherwise a working preset's key. */
  selectedKey: string | null;
  /** Monotonic counter feeding local-<n> keys. */
  localSeq: number;
  loaded: boolean;
}

/** Identity/appearance/spawn/stat fields settable via setField. Typed, not `any`. */
export type PresetFieldPath =
  | "name"
  | "defaultName"
  | "description"
  | "jobId"
  | "gender"
  | "face"
  | "hair"
  | "hairColor"
  | "skinColor"
  | "mapId"
  | "level"
  | "meso"
  | "gm"
  | "stats.str"
  | "stats.dex"
  | "stats.int"
  | "stats.luk"
  | "stats.hp"
  | "stats.mp";

export type PresetEditorAction =
  | { type: "load"; presets: CharacterPreset[] }
  | { type: "select"; key: string | null }
  | { type: "addPreset" }
  | { type: "duplicatePreset"; key: string }
  | { type: "removePreset"; key: string }
  | { type: "setField"; key: string; path: PresetFieldPath; value: number | string }
  | { type: "addEquip"; key: string; templateId: number }
  | { type: "removeEquip"; key: string; index: number }
  | { type: "setEquipAvg"; key: string; index: number; value: boolean }
  | { type: "addInventory"; key: string; templateId: number }
  | { type: "removeInventory"; key: string; index: number }
  | { type: "setInventoryQty"; key: string; index: number; value: number }
  | { type: "addSkill"; key: string; skillId: number }
  | { type: "removeSkill"; key: string; index: number }
  | { type: "setSkillLevel"; key: string; index: number; value: number }
  | { type: "addTag"; key: string; tag: string }
  | { type: "removeTag"; key: string; tag: string }
  | { type: "discard" }
  | { type: "savedOk" };

export const DEFAULT_PRESET_ATTRIBUTES: CharacterPresetAttributes = {
  name: "New preset",
  description: "",
  tags: [],
  jobId: 0,
  gender: 0,
  face: 20000,
  hair: 30030,
  hairColor: 0,
  skinColor: 0,
  mapId: 0,
  level: 1,
  meso: 0,
  gm: 0,
  stats: { str: 4, dex: 4, int: 4, luk: 4, hp: 50, mp: 5 },
  defaultName: "",
  equipment: [],
  inventory: [],
  skills: [],
};

export function normalizePreset(
  raw: Partial<CharacterPresetAttributes> | undefined,
): CharacterPresetAttributes {
  const d = DEFAULT_PRESET_ATTRIBUTES;
  return {
    name: raw?.name ?? d.name,
    description: raw?.description ?? d.description,
    tags: raw?.tags ? [...raw.tags] : [],
    jobId: raw?.jobId ?? d.jobId,
    gender: raw?.gender ?? d.gender,
    face: raw?.face ?? d.face,
    hair: raw?.hair ?? d.hair,
    hairColor: raw?.hairColor ?? d.hairColor,
    skinColor: raw?.skinColor ?? d.skinColor,
    mapId: raw?.mapId ?? d.mapId,
    level: raw?.level ?? d.level,
    meso: raw?.meso ?? d.meso,
    gm: raw?.gm ?? d.gm,
    stats: { ...d.stats, ...raw?.stats },
    defaultName: raw?.defaultName ?? d.defaultName,
    equipment: raw?.equipment ? raw.equipment.map((e) => ({ ...e })) : [],
    inventory: raw?.inventory ? raw.inventory.map((e) => ({ ...e })) : [],
    skills: raw?.skills ? raw.skills.map((e) => ({ ...e })) : [],
  };
}

function cloneAttributes(a: CharacterPresetAttributes): CharacterPresetAttributes {
  return normalizePreset(a);
}

export function initialPresetEditorState(): PresetEditorState {
  return { presets: [], baseline: [], selectedKey: null, localSeq: 0, loaded: false };
}

function project(p: WorkingPreset): CharacterPreset {
  return p.id !== undefined
    ? { id: p.id, attributes: p.attributes }
    : { attributes: p.attributes };
}

export function projectForSave(state: PresetEditorState): CharacterPreset[] {
  return state.presets.map(project);
}

export function isDirty(state: PresetEditorState): boolean {
  return JSON.stringify(state.presets.map(project)) !== JSON.stringify(state.baseline);
}

export function selectedPreset(state: PresetEditorState): WorkingPreset | null {
  if (state.selectedKey === null) return null;
  return state.presets.find((p) => p.key === state.selectedKey) ?? null;
}

export function presetDirty(state: PresetEditorState, key: string): boolean {
  const idx = state.presets.findIndex((p) => p.key === key);
  if (idx < 0) return false;
  const base = state.baseline[idx];
  if (!base) return true; // freshly added row with no baseline counterpart
  return JSON.stringify(project(state.presets[idx])) !== JSON.stringify(base);
}

function updateOne(
  state: PresetEditorState,
  key: string,
  update: (a: CharacterPresetAttributes) => CharacterPresetAttributes,
): PresetEditorState {
  const presets = state.presets.map((p) =>
    p.key === key ? { ...p, attributes: update(cloneAttributes(p.attributes)) } : p,
  );
  return { ...state, presets };
}

export function presetReducer(
  state: PresetEditorState,
  action: PresetEditorAction,
): PresetEditorState {
  switch (action.type) {
    case "load": {
      let seq = 0;
      const presets: WorkingPreset[] = action.presets.map((p) => ({
        key: p.id ?? `local-${seq++}`,
        id: p.id,
        attributes: normalizePreset(p.attributes),
      }));
      return {
        presets,
        baseline: presets.map(project),
        selectedKey: null,
        localSeq: seq,
        loaded: true,
      };
    }
    case "select":
      return { ...state, selectedKey: action.key };
    case "addPreset": {
      const key = `local-${state.localSeq}`;
      const row: WorkingPreset = { key, attributes: cloneAttributes(DEFAULT_PRESET_ATTRIBUTES) };
      return {
        ...state,
        presets: [...state.presets, row],
        localSeq: state.localSeq + 1,
        selectedKey: key,
      };
    }
    case "duplicatePreset": {
      const src = state.presets.find((p) => p.key === action.key);
      if (!src) return state;
      const key = `local-${state.localSeq}`;
      const row: WorkingPreset = { key, attributes: cloneAttributes(src.attributes) };
      return {
        ...state,
        presets: [...state.presets, row],
        localSeq: state.localSeq + 1,
        selectedKey: key,
      };
    }
    case "removePreset": {
      const presets = state.presets.filter((p) => p.key !== action.key);
      const selectedKey = state.selectedKey === action.key ? null : state.selectedKey;
      return { ...state, presets, selectedKey };
    }
    case "setField":
      return updateOne(state, action.key, (a) => {
        if (action.path.startsWith("stats.")) {
          const stat = action.path.slice("stats.".length) as keyof typeof a.stats;
          return { ...a, stats: { ...a.stats, [stat]: action.value as number } };
        }
        if (action.path === "gender") {
          return { ...a, gender: (action.value as number) === 1 ? 1 : 0 };
        }
        return { ...a, [action.path]: action.value } as CharacterPresetAttributes;
      });
    case "addEquip":
      return updateOne(state, action.key, (a) => ({
        ...a,
        equipment: [...a.equipment, { templateId: action.templateId, useAverageStats: true }],
      }));
    case "removeEquip":
      return updateOne(state, action.key, (a) => ({
        ...a,
        equipment: a.equipment.filter((_, i) => i !== action.index),
      }));
    case "setEquipAvg":
      return updateOne(state, action.key, (a) => ({
        ...a,
        equipment: a.equipment.map((e, i) =>
          i === action.index ? { ...e, useAverageStats: action.value } : e,
        ),
      }));
    case "addInventory":
      return updateOne(state, action.key, (a) => ({
        ...a,
        inventory: [...a.inventory, { templateId: action.templateId, quantity: 1 }],
      }));
    case "removeInventory":
      return updateOne(state, action.key, (a) => ({
        ...a,
        inventory: a.inventory.filter((_, i) => i !== action.index),
      }));
    case "setInventoryQty":
      return updateOne(state, action.key, (a) => ({
        ...a,
        inventory: a.inventory.map((e, i) =>
          i === action.index ? { ...e, quantity: Math.max(1, action.value) } : e,
        ),
      }));
    case "addSkill":
      return updateOne(state, action.key, (a) => ({
        ...a,
        skills: [...a.skills, { skillId: action.skillId, level: 1 }],
      }));
    case "removeSkill":
      return updateOne(state, action.key, (a) => ({
        ...a,
        skills: a.skills.filter((_, i) => i !== action.index),
      }));
    case "setSkillLevel":
      return updateOne(state, action.key, (a) => ({
        ...a,
        skills: a.skills.map((e, i) =>
          i === action.index ? { ...e, level: Math.max(1, action.value) } : e,
        ),
      }));
    case "addTag":
      return updateOne(state, action.key, (a) =>
        a.tags.includes(action.tag) ? a : { ...a, tags: [...a.tags, action.tag] },
      );
    case "removeTag":
      return updateOne(state, action.key, (a) => ({
        ...a,
        tags: a.tags.filter((t) => t !== action.tag),
      }));
    case "discard": {
      let seq = 0;
      const presets: WorkingPreset[] = state.baseline.map((p) => ({
        key: p.id ?? `local-${seq++}`,
        id: p.id,
        attributes: normalizePreset(p.attributes),
      }));
      return { ...state, presets, localSeq: seq, selectedKey: null };
    }
    case "savedOk":
      return { ...state, baseline: state.presets.map(project) };
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/presetEditorState.test.ts`
Expected: PASS (all cases).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/presets/presetEditorState.ts services/atlas-ui/src/components/features/characters/presets/__tests__/presetEditorState.test.ts
git commit -m "feat(task-180): key-addressed preset editor reducer + dirty/save projection"
```

---

### Task 3: `presetLoadout.ts` — preview loadout builder

**Files:**
- Create: `services/atlas-ui/src/components/features/characters/presets/presetLoadout.ts`
- Test: `services/atlas-ui/src/components/features/characters/presets/__tests__/presetLoadout.test.ts`

**Interfaces:**
- Consumes: `CharacterPresetAttributes` (`@/types/models/template`); `CharacterLoadout`, `filterEquipment` (`@/services/api/characterRender.service`); `getDefaultSlotForTemplateId` (`@/lib/utils/maplestory`); `RENDER_DEFAULT_SKIN/HAIR/FACE` (`../templates/previewLoadout`).
- Produces:
  - `buildPresetLoadout(attrs): CharacterLoadout`.
  - `buildPresetVariantLoadout(attrs, dimension, id): CharacterLoadout` where `dimension` is `"faces" | "hairs" | "hairColors" | "skinColors"`.
  - `wornTemplateIds(attrs): number[]` — placeable worn ids in slot order (for the worn strip / preview card).

- [ ] **Step 1: Write the failing test**

```ts
import { describe, it, expect } from "vitest";
import {
  buildPresetLoadout,
  buildPresetVariantLoadout,
  wornTemplateIds,
} from "../presetLoadout";
import { DEFAULT_PRESET_ATTRIBUTES } from "../presetEditorState";

const attrs = (o: Partial<typeof DEFAULT_PRESET_ATTRIBUTES>) => ({
  ...DEFAULT_PRESET_ATTRIBUTES,
  ...o,
});

describe("presetLoadout", () => {
  it("composites hair as base + color and passes gender/face/skin", () => {
    const l = buildPresetLoadout(
      attrs({ hair: 30030, hairColor: 2, face: 20001, skinColor: 3, gender: 1 }),
    );
    expect(l.hair).toBe(30032);
    expect(l.face).toBe(20001);
    expect(l.skin).toBe(3);
    expect(l.gender).toBe(1);
  });

  it("places worn equipment on canonical slots and drops unplaceable ids", () => {
    const l = buildPresetLoadout(
      attrs({
        equipment: [
          { templateId: 1040002, useAverageStats: true }, // top -> -5
          { templateId: 1302000, useAverageStats: true }, // weapon -> -11
          { templateId: 2000000, useAverageStats: true }, // use item -> null, skipped
        ],
      }),
    );
    expect(l.equipment["-5"]).toBe(1040002);
    expect(l.equipment["-11"]).toBe(1302000);
    expect(Object.values(l.equipment)).not.toContain(2000000);
  });

  it("later same-slot item overwrites earlier (map semantics)", () => {
    const l = buildPresetLoadout(
      attrs({
        equipment: [
          { templateId: 1040002, useAverageStats: true }, // top -5
          { templateId: 1040010, useAverageStats: true }, // top -5 (wins)
        ],
      }),
    );
    expect(l.equipment["-5"]).toBe(1040010);
  });

  it("empty loadout falls back to render defaults", () => {
    const l = buildPresetLoadout(attrs({ hair: 0, hairColor: 0, face: 0, skinColor: 0 }));
    // hair 0 + 0 stays 0 only if attrs provide it; defaults apply when fields are the
    // documented empty sentinels — assert the fallback path for face/skin explicitly:
    const l2 = buildPresetLoadout({ ...DEFAULT_PRESET_ATTRIBUTES });
    expect(l2.face).toBe(20000);
    expect(l2.hair).toBe(30030);
    expect(l2.skin).toBe(0);
    expect(l).toBeDefined();
  });

  it("variant loadout substitutes one dimension", () => {
    const base = attrs({ hair: 30030, hairColor: 1, face: 20000, skinColor: 0 });
    expect(buildPresetVariantLoadout(base, "faces", 21000).face).toBe(21000);
    expect(buildPresetVariantLoadout(base, "hairs", 30040).hair).toBe(30041);
    expect(buildPresetVariantLoadout(base, "hairColors", 5).hair).toBe(30035);
    expect(buildPresetVariantLoadout(base, "skinColors", 4).skin).toBe(4);
  });

  it("wornTemplateIds lists placeable ids only", () => {
    expect(
      wornTemplateIds(
        attrs({
          equipment: [
            { templateId: 1040002, useAverageStats: true },
            { templateId: 2000000, useAverageStats: true },
          ],
        }),
      ),
    ).toEqual([1040002]);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/presetLoadout.test.ts`
Expected: FAIL — cannot resolve `../presetLoadout`.

- [ ] **Step 3: Write minimal implementation**

```ts
import type { CharacterPresetAttributes } from "@/types/models/template";
import {
  filterEquipment,
  type CharacterLoadout,
} from "@/services/api/characterRender.service";
import { getDefaultSlotForTemplateId } from "@/lib/utils/maplestory";
import {
  RENDER_DEFAULT_SKIN,
  RENDER_DEFAULT_HAIR,
  RENDER_DEFAULT_FACE,
} from "../templates/previewLoadout";

export type PresetAppearanceDimension =
  | "faces"
  | "hairs"
  | "hairColors"
  | "skinColors";

/** Placeable worn template ids in slot-assignment order (unplaceable dropped). */
export function wornTemplateIds(attrs: CharacterPresetAttributes): number[] {
  return attrs.equipment
    .map((e) => e.templateId)
    .filter((id) => getDefaultSlotForTemplateId(id) !== null);
}

/** Worn equipment as a slot→templateId map, later same-slot items overwriting earlier. */
function placedEquipment(attrs: CharacterPresetAttributes): Record<string, number> {
  const eq: Record<string, number> = {};
  for (const e of attrs.equipment) {
    const slot = getDefaultSlotForTemplateId(e.templateId);
    if (slot === null) continue;
    eq[String(slot)] = e.templateId;
  }
  return filterEquipment(eq);
}

export function buildPresetLoadout(
  attrs: CharacterPresetAttributes,
): CharacterLoadout {
  const baseHair = attrs.hair || RENDER_DEFAULT_HAIR;
  return {
    skin: attrs.skinColor || RENDER_DEFAULT_SKIN,
    hair: baseHair + attrs.hairColor,
    face: attrs.face || RENDER_DEFAULT_FACE,
    equipment: placedEquipment(attrs),
    gender: attrs.gender,
  };
}

export function buildPresetVariantLoadout(
  attrs: CharacterPresetAttributes,
  dimension: PresetAppearanceDimension,
  candidateId: number,
): CharacterLoadout {
  const base = buildPresetLoadout(attrs);
  switch (dimension) {
    case "faces":
      return { ...base, face: candidateId };
    case "hairs":
      return { ...base, hair: candidateId + attrs.hairColor };
    case "hairColors":
      return { ...base, hair: (attrs.hair || RENDER_DEFAULT_HAIR) + candidateId };
    case "skinColors":
      return { ...base, skin: candidateId };
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/presetLoadout.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/presets/presetLoadout.ts services/atlas-ui/src/components/features/characters/presets/__tests__/presetLoadout.test.ts
git commit -m "feat(task-180): preset preview loadout builder"
```

---

### Task 4: Generalize `AppearanceBrowserDialog` (additive props, §8)

Introduce a backward-compatible seam so the shared browser supports single-select "replace" semantics for presets while templates keep "add"-to-pool behavior. Task-177 browser tests are the regression gate.

**Files:**
- Modify: `services/atlas-ui/src/components/features/characters/templates/AppearanceBrowserDialog.tsx`
- Modify: `services/atlas-ui/src/components/features/characters/templates/AppearanceThumb.tsx` (add optional `selected` ring prop)
- Modify (call site): `services/atlas-ui/src/components/features/characters/templates/CharacterTemplatesEditor.tsx:236-248`
- Test: existing `templates/__tests__/AppearanceBrowserDialog.test.tsx` MUST stay green (run, don't rewrite); add cases for the new `selectMode="replace"` path.

**Interfaces:**
- Consumes: `AppearancePoolKey`, `PreviewPicks` (`./editorState`); `CharacterLoadout` (`@/services/api/characterRender.service`).
- Produces (new prop shape for `AppearanceBrowserDialog`):
  ```ts
  interface AppearanceBrowserDialogProps {
    dimension: AppearancePoolKey;
    gender: number;
    variantLoadout: (dimension: AppearancePoolKey, id: number) => CharacterLoadout;
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onSelect: (id: number) => void;
    selectMode?: "add" | "replace"; // default "add"
    markedIds?: number[];            // "add" mode: ids already in the pool
    selectedId?: number;             // "replace" mode: the current single value
  }
  ```
  `AppearanceThumb` gains `selected?: boolean` (renders a `ring-2 ring-primary` when true), keeping existing `marked` behavior.

- [ ] **Step 1: Write the failing test (new replace-mode case appended to the existing file)**

Add to `templates/__tests__/AppearanceBrowserDialog.test.tsx`:

```tsx
it("replace mode: clicking a thumb calls onSelect and shows a selection ring on selectedId", async () => {
  const onSelect = vi.fn();
  render(
    <AppearanceBrowserDialog
      dimension="skinColors"
      gender={0}
      variantLoadout={(_d, id) => ({ skin: id, hair: 30030, face: 20000, equipment: {}, gender: 0 })}
      open
      onOpenChange={() => {}}
      onSelect={onSelect}
      selectMode="replace"
      selectedId={2}
    />,
  );
  // skin candidates 0-9 render; the selectedId thumb is marked selected
  const selected = await screen.findByRole("button", { name: /skin tone 2/i });
  expect(selected.className).toMatch(/ring/);
  const other = screen.getByRole("button", { name: /skin tone 5/i });
  await userEvent.click(other);
  expect(onSelect).toHaveBeenCalledWith(5);
});
```

(Keep every pre-existing test in the file — they exercise `selectMode` defaulting to `"add"`.)

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/templates/__tests__/AppearanceBrowserDialog.test.tsx`
Expected: the new case FAILS (props not yet accepted); pre-existing cases still pass.

- [ ] **Step 3: Update `AppearanceThumb.tsx`**

Add to its props interface: `selected?: boolean;`. In the button's `className`, append (via the existing `cn`/template): when `selected`, `"ring-2 ring-primary"`; keep the existing `marked` styling. Update `aria-checked`/label semantics only if the component already exposes them; otherwise leave the existing `ariaLabel` untouched.

- [ ] **Step 4: Update `AppearanceBrowserDialog.tsx` to the new prop shape**

Replace `template`/`picks`/`onAdd` with `gender`, `variantLoadout`, `onSelect`, `selectMode = "add"`, `markedIds`, `selectedId`. Concretely:
- Gender filter: replace `template.gender === 1` with `gender === 1`.
- Candidate render URL: replace `buildVariantLoadout(template, picks, dimension, id)` with `variantLoadout(dimension, id)`.
- `inPool`/`marked`: replace `template[dimension]` with `markedIds ?? []`; a thumb is `marked={selectMode === "add" && (markedIds ?? []).includes(id)}`.
- New: `selected={selectMode === "replace" && selectedId === id}`.
- `onSelect`: on thumb click call `onSelect(id)`; in `"replace"` mode also `onOpenChange(false)` (close on pick). In `"add"` mode keep the dialog open (unchanged behavior).

- [ ] **Step 5: Update the templates call site (`CharacterTemplatesEditor.tsx:236-248`)**

```tsx
<AppearanceBrowserDialog
  dimension={dimension}
  gender={template.gender}
  variantLoadout={(dim, id) => buildVariantLoadout(template, picks, dim, id)}
  open={open}
  onOpenChange={onOpenChange}
  onSelect={(id) => dispatch({ type: "addPoolEntry", pool: dimension, id })}
  selectMode="add"
  markedIds={template[dimension]}
/>
```

Add the import: `import { buildVariantLoadout } from "./previewLoadout";` (if not already present).

- [ ] **Step 6: Run the full templates browser + editor suites**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/templates/__tests__/AppearanceBrowserDialog.test.tsx src/components/features/characters/templates/__tests__/CharacterTemplatesEditor.test.tsx`
Expected: PASS (all pre-existing cases + the new replace-mode case). If a pre-existing assertion referenced the old props, adjust ONLY the test's prop wiring, not its behavior expectations.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/templates/AppearanceBrowserDialog.tsx services/atlas-ui/src/components/features/characters/templates/AppearanceThumb.tsx services/atlas-ui/src/components/features/characters/templates/CharacterTemplatesEditor.tsx services/atlas-ui/src/components/features/characters/templates/__tests__/AppearanceBrowserDialog.test.tsx
git commit -m "feat(task-180): generalize AppearanceBrowserDialog with add/replace select modes"
```

---

### Task 5: Add optional `trailing` slot to `ItemRow` (additive, §9)

**Files:**
- Modify: `services/atlas-ui/src/components/features/characters/templates/ItemRow.tsx`
- Test: `services/atlas-ui/src/components/features/characters/templates/__tests__/ItemRow.test.tsx` (create if absent)

**Interfaces:**
- Produces: `ItemRow` props gain `trailing?: React.ReactNode` rendered between the mono id and the remove ×. Undefined by default → identical to current output.

- [ ] **Step 1: Write the failing test**

```tsx
import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { ItemRow } from "../ItemRow";

vi.mock("@/context/tenant-context", () => ({ useTenant: () => ({ activeTenant: null }) }));
vi.mock("@/lib/hooks/api/useItemStrings", () => ({
  useItemName: () => ({ data: "Test Item", isError: false }),
}));

describe("ItemRow", () => {
  it("renders trailing content when provided", () => {
    render(
      <ItemRow id={1040002} onRemove={() => {}} removeAriaLabel="Remove"
        trailing={<span data-testid="trailing">avg</span>} />,
    );
    expect(screen.getByTestId("trailing")).toBeInTheDocument();
  });

  it("omits trailing region cleanly when not provided", () => {
    render(<ItemRow id={1040002} onRemove={() => {}} removeAriaLabel="Remove" />);
    expect(screen.queryByTestId("trailing")).toBeNull();
    expect(screen.getByText("1040002")).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/templates/__tests__/ItemRow.test.tsx`
Expected: FAIL — `trailing` not rendered.

- [ ] **Step 3: Implement**

In `ItemRow.tsx`: add `trailing?: React.ReactNode;` to `ItemRowProps`; render `{trailing}` between the `<span className="font-mono ...">{id}</span>` and the remove `<Button>`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/templates/__tests__/ItemRow.test.tsx`
Expected: PASS. Also run `npx vitest run src/components/features/characters/templates/__tests__/EquipmentPoolSection.test.tsx src/components/features/characters/templates/__tests__/StartingKitSection.test.tsx` — must stay green (unaffected default).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/templates/ItemRow.tsx services/atlas-ui/src/components/features/characters/templates/__tests__/ItemRow.test.tsx
git commit -m "feat(task-180): ItemRow optional trailing slot"
```

---

### Task 6: `ApplyPresetDialog` — optional `initialPresetId` (UI-only, §10)

**Files:**
- Modify: `services/atlas-ui/src/components/features/characters/ApplyPresetDialog.tsx`
- Test: `services/atlas-ui/src/components/features/characters/__tests__/ApplyPresetDialog.test.tsx` (create if absent; if present, add cases)

**Interfaces:**
- Produces: `ApplyPresetDialogProps` gains `initialPresetId?: string`. When set and it resolves to a saved preset, seed `defaultValues.presetId` to it; the preset grid pre-selects that preset. Reading presets from **saved** config is unchanged (apply uses last-saved).

- [ ] **Step 1: Write the failing test**

```tsx
// Mock the tenant-config + services + mutation hooks so the dialog renders with
// two saved presets, then assert initialPresetId pre-selects the second.
it("pre-selects initialPresetId when provided", async () => {
  renderApplyDialog({ initialPresetId: "b2" }); // helper wires two presets a1,b2
  const radio = await screen.findByRole("radio", { name: /Preset Two/i });
  expect(radio).toHaveAttribute("aria-checked", "true");
});
```

Wire the mocks mirroring the existing dialog's hooks (`useTenantConfiguration` returns `attributes.characters.presets` = `[{id:"a1",...},{id:"b2",...}]`, `useServices` returns a channel service serving worldId 0, `useCreateCharacterFromPreset`/`useNameValidity` return stubs).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/__tests__/ApplyPresetDialog.test.tsx`
Expected: FAIL — no pre-selection (default `presetId: ""`).

- [ ] **Step 3: Implement**

- Add `initialPresetId?: string;` to `ApplyPresetDialogProps` and destructure it.
- In the open-reset effect (`ApplyPresetDialog.tsx:148-152`), seed `presetId` from `initialPresetId` if it matches a saved preset:
  ```ts
  useEffect(() => {
    if (open) {
      const preset = initialPresetId && presets.some((p) => p.id === initialPresetId)
        ? initialPresetId
        : "";
      form.reset({ presetId: preset, worldId: 0, name: "" });
    }
  }, [open, form, initialPresetId, presets]);
  ```
  (`presets` is already computed above the effect; move its declaration above the effect if TS flags use-before-declaration, or reference `tenantConfigQuery.data` directly.)

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/__tests__/ApplyPresetDialog.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/ApplyPresetDialog.tsx services/atlas-ui/src/components/features/characters/__tests__/ApplyPresetDialog.test.tsx
git commit -m "feat(task-180): ApplyPresetDialog optional initialPresetId pre-selection"
```

---

### Task 7: `PresetPreviewCard.tsx` — sticky live preview

Thin variant of `templates/PreviewCard.tsx`, sourcing the loadout from `buildPresetLoadout` and the worn strip from `wornTemplateIds`.

**Files:**
- Create: `services/atlas-ui/src/components/features/characters/presets/PresetPreviewCard.tsx`
- Test: `services/atlas-ui/src/components/features/characters/presets/__tests__/PresetPreviewCard.test.tsx`

**Interfaces:**
- Consumes: `CharacterPresetAttributes`; `buildPresetLoadout`, `wornTemplateIds` (`./presetLoadout`); `useCharacterImage`; `useTenant()` (`@/context/tenant-context`); `useItemName`.
- Produces: `<PresetPreviewCard attrs={CharacterPresetAttributes} />`.

- [ ] **Step 1: Write the failing test**

```tsx
import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { PresetPreviewCard } from "../PresetPreviewCard";
import { DEFAULT_PRESET_ATTRIBUTES } from "../presetEditorState";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } },
  }),
}));
vi.mock("@/lib/hooks/useCharacterImage", () => ({
  useCharacterImage: () => ({ isLoading: false, isError: false, imageUrl: "http://img/preview.png", refetch: vi.fn() }),
}));
vi.mock("@/lib/hooks/api/useItemStrings", () => ({ useItemName: () => ({ data: "Item" }) }));

describe("PresetPreviewCard", () => {
  it("renders the composited preview image", () => {
    render(<PresetPreviewCard attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }} />);
    expect(screen.getByRole("img", { name: /live preview/i })).toHaveAttribute("src", "http://img/preview.png");
  });

  it("shows a worn-icon for each placeable equipment id", () => {
    render(
      <PresetPreviewCard
        attrs={{ ...DEFAULT_PRESET_ATTRIBUTES, equipment: [{ templateId: 1040002, useAverageStats: true }] }}
      />,
    );
    expect(screen.getAllByTestId("worn-icon")).toHaveLength(1);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/PresetPreviewCard.test.tsx`
Expected: FAIL — cannot resolve `../PresetPreviewCard`.

- [ ] **Step 3: Implement** (mirror `PreviewCard.tsx`; build character data from `buildPresetLoadout`)

```tsx
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Tooltip, TooltipContent, TooltipProvider, TooltipTrigger,
} from "@/components/ui/tooltip";
import type { CharacterPresetAttributes } from "@/types/models/template";
import type { MapleStoryCharacterData } from "@/types/models/maplestory";
import { useCharacterImage } from "@/lib/hooks/useCharacterImage";
import { useItemName } from "@/lib/hooks/api/useItemStrings";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { useTenant } from "@/context/tenant-context";
import { buildPresetLoadout, wornTemplateIds } from "./presetLoadout";

function WornIcon({ id }: { id: number }) {
  const { activeTenant } = useTenant();
  const name = useItemName(String(id));
  if (!activeTenant) return null;
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <img
          data-testid="worn-icon"
          src={getAssetIconUrl(activeTenant.id, activeTenant.attributes.region, activeTenant.attributes.majorVersion, activeTenant.attributes.minorVersion, "item", id)}
          alt={name.data ?? String(id)}
          width={28} height={28} loading="lazy"
          className="rounded border bg-muted/40 p-0.5 [image-rendering:pixelated]"
        />
      </TooltipTrigger>
      <TooltipContent>{name.data ?? "Unknown item"} · {id}</TooltipContent>
    </Tooltip>
  );
}

export function PresetPreviewCard({ attrs }: { attrs: CharacterPresetAttributes }) {
  const { activeTenant } = useTenant();
  const loadout = buildPresetLoadout(attrs);

  const character: MapleStoryCharacterData = {
    id: "preset-preview",
    name: "preview",
    level: attrs.level,
    jobId: attrs.jobId,
    hair: loadout.hair,
    face: loadout.face,
    skinColor: loadout.skin,
    gender: attrs.gender,
    equipment: loadout.equipment,
    tenant: activeTenant?.id ?? "",
    region: activeTenant?.attributes.region ?? "",
    majorVersion: activeTenant?.attributes.majorVersion ?? 0,
    minorVersion: activeTenant?.attributes.minorVersion ?? 0,
  };

  const image = useCharacterImage(character, { stance: "stand1", resize: 2 }, { enabled: !!activeTenant });
  const worn = wornTemplateIds(attrs);

  return (
    <TooltipProvider>
      <div className="rounded-lg border bg-card p-3 lg:sticky lg:top-4">
        <p className="text-xs font-medium text-muted-foreground">Live preview</p>
        <div className="mx-auto mt-2 flex h-[200px] w-[154px] items-end justify-center rounded-md bg-gradient-to-b from-primary/5 to-primary/15">
          {image.isLoading && <Skeleton className="h-[160px] w-[120px]" />}
          {image.isError && (
            <div className="flex flex-col items-center gap-2 pb-6 text-center">
              <p className="text-xs text-muted-foreground">Preview failed</p>
              <Button type="button" variant="outline" size="sm" onClick={() => void image.refetch()}>Retry</Button>
            </div>
          )}
          {!image.isLoading && !image.isError && image.imageUrl && (
            <img src={image.imageUrl} alt="Live preview of the selected preset" width={192} height={256}
              className="max-h-full w-auto [image-rendering:pixelated] drop-shadow-[0_6px_4px_rgba(0,0,0,0.25)]" />
          )}
        </div>
        {worn.length > 0 && (
          <div className="mt-2 flex flex-wrap justify-center gap-1">
            {worn.map((id) => <WornIcon key={id} id={id} />)}
          </div>
        )}
        <p className="mt-2 text-center text-xs text-muted-foreground">
          Composited from this preset's appearance and worn items.
        </p>
      </div>
    </TooltipProvider>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/PresetPreviewCard.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/presets/PresetPreviewCard.tsx services/atlas-ui/src/components/features/characters/presets/__tests__/PresetPreviewCard.test.tsx
git commit -m "feat(task-180): preset live preview card"
```

---

### Task 8: `IdentitySection.tsx`

**Files:**
- Create: `services/atlas-ui/src/components/features/characters/presets/IdentitySection.tsx`
- Test: `.../presets/__tests__/IdentitySection.test.tsx`

**Interfaces:**
- Produces:
  ```ts
  interface IdentitySectionProps {
    attrs: CharacterPresetAttributes;
    onSetField: (path: "name" | "defaultName" | "description", value: string) => void;
    onAddTag: (tag: string) => void;
    onRemoveTag: (tag: string) => void;
    actions?: React.ReactNode; // kebab menu, aligned top-right
  }
  ```

- [ ] **Step 1: Write the failing test**

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { IdentitySection } from "../IdentitySection";
import { DEFAULT_PRESET_ATTRIBUTES } from "../presetEditorState";

describe("IdentitySection", () => {
  it("edits name and calls onSetField", async () => {
    const onSetField = vi.fn();
    render(<IdentitySection attrs={{ ...DEFAULT_PRESET_ATTRIBUTES, name: "" }}
      onSetField={onSetField} onAddTag={vi.fn()} onRemoveTag={vi.fn()} />);
    await userEvent.type(screen.getByLabelText(/^name/i), "Hero");
    expect(onSetField).toHaveBeenCalledWith("name", expect.any(String));
  });

  it("adds and removes tags", async () => {
    const onAddTag = vi.fn();
    const onRemoveTag = vi.fn();
    render(<IdentitySection attrs={{ ...DEFAULT_PRESET_ATTRIBUTES, tags: ["PvP"] }}
      onSetField={vi.fn()} onAddTag={onAddTag} onRemoveTag={onRemoveTag} />);
    await userEvent.click(screen.getByRole("button", { name: /remove tag PvP/i }));
    expect(onRemoveTag).toHaveBeenCalledWith("PvP");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/IdentitySection.test.tsx`
Expected: FAIL — cannot resolve `../IdentitySection`.

- [ ] **Step 3: Implement**

Card/section wrapper matching the prototype: header row with title "Identity" + `{actions}` on the right. Fields: `name` (Input, `maxLength={64}`, `aria-label="Name"`, required marker), `defaultName` (Input, `aria-label="Default character name"`, helper "empty = prompt on apply"), `description` (Input/Textarea, `maxLength={512}`). Tags: chip row — each tag a button `aria-label={`Remove tag ${tag}`}` with an `X`; a `+` opens a small inline input (or the same add-dialog idiom as the deleted form's `TagsField`) that calls `onAddTag(trimmed)` for non-empty trimmed values. All inputs are controlled from `attrs` and call the corresponding `onSetField`/`onAddTag`/`onRemoveTag`. Use shadcn `Input`, `Button`, `Label`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/IdentitySection.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/presets/IdentitySection.tsx services/atlas-ui/src/components/features/characters/presets/__tests__/IdentitySection.test.tsx
git commit -m "feat(task-180): preset Identity section"
```

---

### Task 9: `ClassAppearanceSection.tsx`

Named job picker + gender + single-select appearance thumbnails + generalized cosmetics browser.

**Files:**
- Create: `services/atlas-ui/src/components/features/characters/presets/ClassAppearanceSection.tsx`
- Test: `.../presets/__tests__/ClassAppearanceSection.test.tsx`

**Interfaces:**
- Consumes: `PRESET_JOBS`, `jobLabel` (`./presetJobs`); `buildPresetVariantLoadout`, `PresetAppearanceDimension` (`./presetLoadout`); generalized `AppearanceBrowserDialog` + `AppearanceThumb` (`../templates/...`); `useTenant()`, `generateCharacterUrl`.
- Produces:
  ```ts
  interface ClassAppearanceSectionProps {
    attrs: CharacterPresetAttributes;
    onSetField: (
      path: "jobId" | "gender" | "face" | "hair" | "hairColor" | "skinColor",
      value: number,
    ) => void;
  }
  ```

- [ ] **Step 1: Write the failing test**

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { ClassAppearanceSection } from "../ClassAppearanceSection";
import { DEFAULT_PRESET_ATTRIBUTES } from "../presetEditorState";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } }),
}));

describe("ClassAppearanceSection", () => {
  it("named job picker sets jobId; advanced numeric accepts arbitrary ids", async () => {
    const onSetField = vi.fn();
    render(<ClassAppearanceSection attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }} onSetField={onSetField} />);
    // Advanced numeric entry
    const advanced = screen.getByLabelText(/advanced job id/i);
    await userEvent.clear(advanced);
    await userEvent.type(advanced, "123456");
    expect(onSetField).toHaveBeenCalledWith("jobId", 123456);
  });

  it("skin thumb click replaces skinColor", async () => {
    const onSetField = vi.fn();
    render(<ClassAppearanceSection attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }} onSetField={onSetField} />);
    await userEvent.click(screen.getByRole("button", { name: /skin tone 3/i }));
    expect(onSetField).toHaveBeenCalledWith("skinColor", 3);
  });

  it("gender select toggles 0/1", async () => {
    const onSetField = vi.fn();
    render(<ClassAppearanceSection attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }} onSetField={onSetField} />);
    // Implementation detail: gender is a shadcn Select or M/F buttons; assert the female choice fires 1.
    await userEvent.click(screen.getByRole("button", { name: /female/i }));
    expect(onSetField).toHaveBeenCalledWith("gender", 1);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/ClassAppearanceSection.test.tsx`
Expected: FAIL — cannot resolve `../ClassAppearanceSection`.

- [ ] **Step 3: Implement**

- **Class**: a searchable select over `PRESET_JOBS` (shadcn `Command`/`Popover` combobox, or a `Select` with a filter input) showing `jobLabel(attrs.jobId)` as the trigger label; selecting fires `onSetField("jobId", id)`. Plus an **Advanced** numeric `Input` (`aria-label="Advanced job id"`, controlled to `attrs.jobId`) firing `onSetField("jobId", Number(e.target.value))`.
- **Gender**: two buttons (Male/Female) or a `Select`; Male → `onSetField("gender", 0)`, Female → `onSetField("gender", 1)`; the active one is highlighted from `attrs.gender`.
- **Appearance rows** — for each of `face` / `hair` / `hairColor` (0–7) / `skinColor` (0–9): a horizontal strip of `AppearanceThumb`s rendered via `generateCharacterUrl(...activeTenant..., buildPresetVariantLoadout(attrs, dimension, id), { stance: "stand1", resize: 2 })`. Candidate lists: faces/hairs show a small on-hand set (the current value plus a few) with a trailing `+` opening `AppearanceBrowserDialog` in `selectMode="replace"`; hairColor 0–7 and skin 0–9 render inline thumbs directly. Each thumb: `selected={attrs[field] === id}`, `ariaLabel={`${noun} ${id}`}`, `onSelect={() => onSetField(field, id)}`.
  - Map dimension→field: `faces→face`, `hairs→hair`, `hairColors→hairColor`, `skinColors→skinColor`.
  - The `+` browser passes `dimension`, `gender={attrs.gender}`, `variantLoadout={(dim, id) => buildPresetVariantLoadout(attrs, dim, id)}`, `selectedId={attrs[field]}`, `onSelect={(id) => onSetField(field, id)}`, `selectMode="replace"`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/ClassAppearanceSection.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/presets/ClassAppearanceSection.tsx services/atlas-ui/src/components/features/characters/presets/__tests__/ClassAppearanceSection.test.tsx
git commit -m "feat(task-180): preset Class & appearance section"
```

---

### Task 10: `SpawnProgressionSection.tsx`

**Files:**
- Create: `.../presets/SpawnProgressionSection.tsx`
- Test: `.../presets/__tests__/SpawnProgressionSection.test.tsx`

**Interfaces:**
- Consumes: `MapPicker` (`../templates/MapPicker`).
- Produces:
  ```ts
  interface SpawnProgressionSectionProps {
    attrs: CharacterPresetAttributes;
    onSetField: (path: "mapId" | "level" | "gm" | "meso", value: number) => void;
  }
  ```

- [ ] **Step 1: Write the failing test**

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { SpawnProgressionSection } from "../SpawnProgressionSection";
import { DEFAULT_PRESET_ATTRIBUTES } from "../presetEditorState";

vi.mock("../../templates/MapPicker", () => ({
  MapPicker: ({ value, onChange }: { value: number; onChange: (n: number) => void }) => (
    <button aria-label="map-picker" onClick={() => onChange(100000000)}>map:{value}</button>
  ),
}));

describe("SpawnProgressionSection", () => {
  it("wires MapPicker to mapId", async () => {
    const onSetField = vi.fn();
    render(<SpawnProgressionSection attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }} onSetField={onSetField} />);
    await userEvent.click(screen.getByLabelText("map-picker"));
    expect(onSetField).toHaveBeenCalledWith("mapId", 100000000);
  });

  it("edits level within 1..250", async () => {
    const onSetField = vi.fn();
    render(<SpawnProgressionSection attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }} onSetField={onSetField} />);
    const level = screen.getByLabelText(/^level/i);
    await userEvent.clear(level);
    await userEvent.type(level, "30");
    expect(onSetField).toHaveBeenCalledWith("level", 30);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/SpawnProgressionSection.test.tsx`
Expected: FAIL — cannot resolve `../SpawnProgressionSection`.

- [ ] **Step 3: Implement**

Section with `<MapPicker value={attrs.mapId} onChange={(id) => onSetField("mapId", id)} />` plus three numeric `Input` steppers: Level (`type="number"`, `min={1}`, `max={250}`, `aria-label="Level"`), GM level (`min={0}`, `aria-label="GM level"`), Meso (`min={0}`, `aria-label="Meso"`). Each `onChange` fires `onSetField(path, Number(e.target.value))`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/SpawnProgressionSection.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/presets/SpawnProgressionSection.tsx services/atlas-ui/src/components/features/characters/presets/__tests__/SpawnProgressionSection.test.tsx
git commit -m "feat(task-180): preset Spawn & progression section"
```

---

### Task 11: `BaseStatsSection.tsx`

**Files:**
- Create: `.../presets/BaseStatsSection.tsx`
- Test: `.../presets/__tests__/BaseStatsSection.test.tsx`

**Interfaces:**
- Produces:
  ```ts
  interface BaseStatsSectionProps {
    attrs: CharacterPresetAttributes;
    onSetStat: (stat: "str" | "dex" | "int" | "luk" | "hp" | "mp", value: number) => void;
  }
  ```

- [ ] **Step 1: Write the failing test**

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { BaseStatsSection } from "../BaseStatsSection";
import { DEFAULT_PRESET_ATTRIBUTES } from "../presetEditorState";

describe("BaseStatsSection", () => {
  it("renders all six stats and edits STR", async () => {
    const onSetStat = vi.fn();
    render(<BaseStatsSection attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }} onSetStat={onSetStat} />);
    for (const s of ["STR", "DEX", "INT", "LUK", "HP", "MP"]) {
      expect(screen.getByLabelText(s)).toBeInTheDocument();
    }
    const str = screen.getByLabelText("STR");
    await userEvent.clear(str);
    await userEvent.type(str, "13");
    expect(onSetStat).toHaveBeenCalledWith("str", 13);
  });

  it("notes stats are written verbatim", () => {
    render(<BaseStatsSection attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }} onSetStat={vi.fn()} />);
    expect(screen.getByText(/written verbatim/i)).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/BaseStatsSection.test.tsx`
Expected: FAIL — cannot resolve `../BaseStatsSection`.

- [ ] **Step 3: Implement**

Grid of six numeric `Input`s over `(["str","dex","int","luk","hp","mp"] as const)`, each `aria-label={stat.toUpperCase()}`, `type="number"`, `min={0}`, controlled to `attrs.stats[stat]`, firing `onSetStat(stat, Number(e.target.value))`. A muted helper line: "Written verbatim to the created character (not derived from level)."

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/BaseStatsSection.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/presets/BaseStatsSection.tsx services/atlas-ui/src/components/features/characters/presets/__tests__/BaseStatsSection.test.tsx
git commit -m "feat(task-180): preset Base stats section"
```

---

### Task 12: `EquipmentSection.tsx`

Flat "Worn items" list with per-row avg-stats toggle; add via subcategory selector → `ItemSearchCombobox`, plus manual-id fallback.

**Files:**
- Create: `.../presets/EquipmentSection.tsx`
- Test: `.../presets/__tests__/EquipmentSection.test.tsx`

**Interfaces:**
- Consumes: `ItemRow` (now with `trailing`), `ItemSearchCombobox`, `POOL_SEARCH_CONFIGS`/`SearchPoolKey` (`../templates/...`); shadcn `Switch`, `Select`, `Input`.
- Produces:
  ```ts
  interface EquipmentSectionProps {
    equipment: CharacterPresetEquipmentEntry[];
    onAdd: (templateId: number) => void;
    onRemove: (index: number) => void;
    onSetAvg: (index: number, value: boolean) => void;
  }
  ```

- [ ] **Step 1: Write the failing test**

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { EquipmentSection } from "../EquipmentSection";

vi.mock("@/context/tenant-context", () => ({ useTenant: () => ({ activeTenant: null }) }));
vi.mock("@/lib/hooks/api/useItemStrings", () => ({ useItemName: () => ({ data: "Item", isError: false }) }));
vi.mock("../../templates/ItemSearchCombobox", () => ({
  ItemSearchCombobox: ({ onSelect }: { onSelect: (id: number) => void }) => (
    <button aria-label="combo-add" onClick={() => onSelect(1040002)}>combo</button>
  ),
}));

describe("EquipmentSection", () => {
  it("lists rows with avg-stats toggle and removes", async () => {
    const onRemove = vi.fn();
    const onSetAvg = vi.fn();
    render(<EquipmentSection
      equipment={[{ templateId: 1040002, useAverageStats: true }]}
      onAdd={vi.fn()} onRemove={onRemove} onSetAvg={onSetAvg} />);
    await userEvent.click(screen.getByRole("switch", { name: /average stats/i }));
    expect(onSetAvg).toHaveBeenCalledWith(0, false);
    await userEvent.click(screen.getByRole("button", { name: /remove equipment 1040002/i }));
    expect(onRemove).toHaveBeenCalledWith(0);
  });

  it("adds via the search combobox", async () => {
    const onAdd = vi.fn();
    render(<EquipmentSection equipment={[]} onAdd={onAdd} onRemove={vi.fn()} onSetAvg={vi.fn()} />);
    await userEvent.click(screen.getByLabelText("combo-add"));
    expect(onAdd).toHaveBeenCalledWith(1040002);
  });

  it("adds via manual id fallback", async () => {
    const onAdd = vi.fn();
    render(<EquipmentSection equipment={[]} onAdd={onAdd} onRemove={vi.fn()} onSetAvg={vi.fn()} />);
    const manual = screen.getByLabelText(/manual item id/i);
    await userEvent.type(manual, "1302000");
    await userEvent.click(screen.getByRole("button", { name: /add item id/i }));
    expect(onAdd).toHaveBeenCalledWith(1302000);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/EquipmentSection.test.tsx`
Expected: FAIL — cannot resolve `../EquipmentSection`.

- [ ] **Step 3: Implement**

- Section header "Worn items" + empty copy "No worn items." when `equipment.length === 0`.
- Each entry → `<ItemRow id={e.templateId} removeAriaLabel={`Remove equipment ${e.templateId}`} onRemove={() => onRemove(i)} trailing={<Switch aria-label="Use average stats" checked={e.useAverageStats} onCheckedChange={(v) => onSetAvg(i, v)} />} />`.
- Add controls: a subcategory `Select` (options: Tops/Bottoms/Shoes/Weapons → `SearchPoolKey`; "All" → `"items"`) that drives the `ItemSearchCombobox` `poolKey`/config (`POOL_SEARCH_CONFIGS[selected]`); the combobox's `onSelect={(id) => onAdd(id)}`.
- Manual fallback: a numeric `Input` (`aria-label="Manual item id"`) + a Button (`aria-label="Add item id"`) firing `onAdd(Number(value))` when the parsed value is a positive integer, then clearing the input.

  *Note:* confirm `ItemSearchCombobox`'s actual prop names by reading `templates/ItemSearchCombobox.tsx` before wiring — pass whatever pool/config + `onSelect` it exposes (the templates `EquipmentPoolSection` is the reference call site).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/EquipmentSection.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/presets/EquipmentSection.tsx services/atlas-ui/src/components/features/characters/presets/__tests__/EquipmentSection.test.tsx
git commit -m "feat(task-180): preset Equipment section"
```

---

### Task 13: `InventorySection.tsx`

**Files:**
- Create: `.../presets/InventorySection.tsx`
- Test: `.../presets/__tests__/InventorySection.test.tsx`

**Interfaces:**
- Consumes: `ItemRow` (`trailing`), `ItemSearchCombobox` (`items` config).
- Produces:
  ```ts
  interface InventorySectionProps {
    inventory: CharacterPresetInventoryEntry[];
    onAdd: (templateId: number) => void;
    onRemove: (index: number) => void;
    onSetQty: (index: number, value: number) => void;
  }
  ```

- [ ] **Step 1: Write the failing test**

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { InventorySection } from "../InventorySection";

vi.mock("@/context/tenant-context", () => ({ useTenant: () => ({ activeTenant: null }) }));
vi.mock("@/lib/hooks/api/useItemStrings", () => ({ useItemName: () => ({ data: "Item", isError: false }) }));
vi.mock("../../templates/ItemSearchCombobox", () => ({
  ItemSearchCombobox: ({ onSelect }: { onSelect: (id: number) => void }) => (
    <button aria-label="combo-add" onClick={() => onSelect(2000000)}>combo</button>
  ),
}));

describe("InventorySection", () => {
  it("shows empty copy when no items", () => {
    render(<InventorySection inventory={[]} onAdd={vi.fn()} onRemove={vi.fn()} onSetQty={vi.fn()} />);
    expect(screen.getByText(/no granted items/i)).toBeInTheDocument();
  });

  it("edits quantity (min 1) and removes", async () => {
    const onSetQty = vi.fn();
    const onRemove = vi.fn();
    render(<InventorySection inventory={[{ templateId: 2000000, quantity: 1 }]}
      onAdd={vi.fn()} onRemove={onRemove} onSetQty={onSetQty} />);
    const qty = screen.getByLabelText(/quantity/i);
    await userEvent.clear(qty);
    await userEvent.type(qty, "10");
    expect(onSetQty).toHaveBeenCalledWith(0, 10);
    await userEvent.click(screen.getByRole("button", { name: /remove item 2000000/i }));
    expect(onRemove).toHaveBeenCalledWith(0);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/InventorySection.test.tsx`
Expected: FAIL — cannot resolve `../InventorySection`.

- [ ] **Step 3: Implement**

Mirror EquipmentSection but: `trailing` is a quantity `Input` (`type="number"`, `min={1}`, `aria-label="Quantity"`) firing `onSetQty(i, Math.max(1, Number(v)))`; `removeAriaLabel={`Remove item ${e.templateId}`}`; the add combobox always uses the `items` config (`POOL_SEARCH_CONFIGS.items`, searches all compartments); empty copy "No granted items." Keep the same manual-id fallback (`aria-label="Manual item id"` + "Add item id" button).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/InventorySection.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/presets/InventorySection.tsx services/atlas-ui/src/components/features/characters/presets/__tests__/InventorySection.test.tsx
git commit -m "feat(task-180): preset Inventory section"
```

---

### Task 14: `SkillsSection.tsx`

Numeric-id add + name/icon lookup, per-row level stepper. No skill browser.

**Files:**
- Create: `.../presets/SkillsSection.tsx`
- Test: `.../presets/__tests__/SkillsSection.test.tsx`

**Interfaces:**
- Consumes: `useSkillData` (`@/lib/hooks/useSkillData`), `getAssetIconUrl`, `useTenant()`.
- Produces:
  ```ts
  interface SkillsSectionProps {
    skills: CharacterPresetSkillEntry[];
    onAdd: (skillId: number) => void;
    onRemove: (index: number) => void;
    onSetLevel: (index: number, value: number) => void;
  }
  ```
  A local `SkillRow` component resolves name/icon via `useSkillData` (read `templates/StartingKitSection.tsx` for the exact `useSkillData` call shape before wiring).

- [ ] **Step 1: Write the failing test**

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { SkillsSection } from "../SkillsSection";

vi.mock("@/context/tenant-context", () => ({ useTenant: () => ({ activeTenant: null }) }));
vi.mock("@/lib/hooks/useSkillData", () => ({
  useSkillData: () => ({ data: { name: "Power Strike" }, isError: false }),
}));

describe("SkillsSection", () => {
  it("shows empty copy when no skills", () => {
    render(<SkillsSection skills={[]} onAdd={vi.fn()} onRemove={vi.fn()} onSetLevel={vi.fn()} />);
    expect(screen.getByText(/grants no skills/i)).toBeInTheDocument();
  });

  it("adds by numeric id", async () => {
    const onAdd = vi.fn();
    render(<SkillsSection skills={[]} onAdd={onAdd} onRemove={vi.fn()} onSetLevel={vi.fn()} />);
    await userEvent.type(screen.getByLabelText(/skill id/i), "1001004");
    await userEvent.click(screen.getByRole("button", { name: /add skill/i }));
    expect(onAdd).toHaveBeenCalledWith(1001004);
  });

  it("edits level (min 1) and removes", async () => {
    const onSetLevel = vi.fn();
    const onRemove = vi.fn();
    render(<SkillsSection skills={[{ skillId: 1001004, level: 1 }]}
      onAdd={vi.fn()} onRemove={onRemove} onSetLevel={onSetLevel} />);
    const lvl = screen.getByLabelText(/level/i);
    await userEvent.clear(lvl);
    await userEvent.type(lvl, "5");
    expect(onSetLevel).toHaveBeenCalledWith(0, 5);
    await userEvent.click(screen.getByRole("button", { name: /remove skill 1001004/i }));
    expect(onRemove).toHaveBeenCalledWith(0);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/SkillsSection.test.tsx`
Expected: FAIL — cannot resolve `../SkillsSection`.

- [ ] **Step 3: Implement**

- Empty copy "This preset grants no skills." when `skills.length === 0`.
- Each entry → a `SkillRow`: skill icon `getAssetIconUrl(...activeTenant..., "skill", skillId)` with placeholder fallback on error, name via `useSkillData(skillId)` (falling back to "Unknown skill"), mono id, a level `Input` (`type="number"`, `min={1}`, `aria-label="Level"`) firing `onSetLevel(i, Math.max(1, Number(v)))`, and a remove Button `aria-label={`Remove skill ${skillId}`}`.
- Add flow: a numeric `Input` (`aria-label="Skill id"`) + Button (`aria-label="Add skill"`) firing `onAdd(Number(value))` for a positive integer, then clearing.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/SkillsSection.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/presets/SkillsSection.tsx services/atlas-ui/src/components/features/characters/presets/__tests__/SkillsSection.test.tsx
git commit -m "feat(task-180): preset Skills section"
```

---

### Task 15: `PresetActionsMenu.tsx` + `PresetEditor.tsx`

The focused per-preset editor: two-column (sections | sticky preview), kebab menu, backlink. Assembles Tasks 7–14.

**Files:**
- Create: `.../presets/PresetActionsMenu.tsx`
- Create: `.../presets/PresetEditor.tsx`
- Test: `.../presets/__tests__/PresetEditor.test.tsx`

**Interfaces:**
- Consumes: all section components (Tasks 8–14), `PresetPreviewCard` (Task 7), `WorkingPreset`, `PresetFieldPath` (`./presetEditorState`).
- Produces:
  - `PresetActionsMenu` props: `{ onDuplicate(): void; onRemove(): void; onApply?(): void; canApply: boolean; applyDisabledReason?: string }` — shadcn `DropdownMenu` kebab (Duplicate; "Apply to an account…" shown only when `onApply` is defined, disabled with a tooltip/hint when `!canApply`; Remove with a confirm `AlertDialog`).
  - `PresetEditor` props:
    ```ts
    interface PresetEditorProps {
      preset: WorkingPreset;
      onBack: () => void;
      onSetField: (path: PresetFieldPath, value: number | string) => void;
      onAddTag: (tag: string) => void;
      onRemoveTag: (tag: string) => void;
      onAddEquip: (templateId: number) => void;
      onRemoveEquip: (index: number) => void;
      onSetEquipAvg: (index: number, value: boolean) => void;
      onAddInventory: (templateId: number) => void;
      onRemoveInventory: (index: number) => void;
      onSetInventoryQty: (index: number, value: number) => void;
      onAddSkill: (skillId: number) => void;
      onRemoveSkill: (index: number) => void;
      onSetSkillLevel: (index: number, value: number) => void;
      onDuplicate: () => void;
      onRemove: () => void;
      onApply?: () => void; // present only in tenant context
    }
    ```

- [ ] **Step 1: Write the failing test**

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { PresetEditor } from "../PresetEditor";
import { DEFAULT_PRESET_ATTRIBUTES } from "../presetEditorState";

// Mock the heavy leaf sections/preview so this test targets assembly + kebab only.
vi.mock("../PresetPreviewCard", () => ({ PresetPreviewCard: () => <div data-testid="preview" /> }));
vi.mock("../ClassAppearanceSection", () => ({ ClassAppearanceSection: () => <div /> }));
vi.mock("../SpawnProgressionSection", () => ({ SpawnProgressionSection: () => <div /> }));
vi.mock("../EquipmentSection", () => ({ EquipmentSection: () => <div /> }));
vi.mock("../InventorySection", () => ({ InventorySection: () => <div /> }));
vi.mock("../SkillsSection", () => ({ SkillsSection: () => <div /> }));

const base = { key: "a1", id: "a1", attributes: { ...DEFAULT_PRESET_ATTRIBUTES, name: "Hero" } };
const handlers = Object.fromEntries(
  ["onSetField","onAddTag","onRemoveTag","onAddEquip","onRemoveEquip","onSetEquipAvg","onAddInventory","onRemoveInventory","onSetInventoryQty","onAddSkill","onRemoveSkill","onSetSkillLevel","onDuplicate","onRemove"].map((k) => [k, vi.fn()]),
);

describe("PresetEditor", () => {
  it("renders backlink, preview, and header name", () => {
    render(<PresetEditor preset={base} onBack={vi.fn()} {...(handlers as never)} />);
    expect(screen.getByRole("button", { name: /preset library/i })).toBeInTheDocument();
    expect(screen.getByTestId("preview")).toBeInTheDocument();
    expect(screen.getByText("Hero")).toBeInTheDocument();
  });

  it("backlink calls onBack", async () => {
    const onBack = vi.fn();
    render(<PresetEditor preset={base} onBack={onBack} {...(handlers as never)} />);
    await userEvent.click(screen.getByRole("button", { name: /preset library/i }));
    expect(onBack).toHaveBeenCalled();
  });

  it("kebab hides Apply when onApply is absent", async () => {
    render(<PresetEditor preset={base} onBack={vi.fn()} {...(handlers as never)} />);
    await userEvent.click(screen.getByRole("button", { name: /preset actions/i }));
    expect(screen.queryByText(/apply to an account/i)).toBeNull();
  });

  it("kebab shows Apply when onApply is present", async () => {
    render(<PresetEditor preset={base} onBack={vi.fn()} onApply={vi.fn()} {...(handlers as never)} />);
    await userEvent.click(screen.getByRole("button", { name: /preset actions/i }));
    expect(await screen.findByText(/apply to an account/i)).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/PresetEditor.test.tsx`
Expected: FAIL — cannot resolve `../PresetEditor`.

- [ ] **Step 3: Implement `PresetActionsMenu.tsx`** (model on `templates/TemplateActionsMenu.tsx`): shadcn `DropdownMenu`, trigger `aria-label="Preset actions"`. Items: Duplicate → `onDuplicate`; conditional "Apply to an account…" (only when `onApply`) → `onApply`, `disabled={!canApply}`; Remove → opens an `AlertDialog` confirm → `onRemove`.

- [ ] **Step 4: Implement `PresetEditor.tsx`**

- Backlink Button (`variant="ghost"`, `aria-label="Preset library"`, "← Preset library") → `onBack`.
- Header: preset name (`preset.attributes.name`), a job badge (`jobLabel(preset.attributes.jobId)`), `Lv {level}` (+ `· GM {gm}` when `gm > 0`), and `<PresetActionsMenu>` top-right with `canApply = preset.id !== undefined` and `applyDisabledReason="Save this preset before applying"`.
- Two-column grid `lg:grid-cols-[minmax(0,1fr)_252px]` (identical idiom to `CharacterTemplatesEditor`): left `order-2 lg:order-1` stacks `IdentitySection` (passing `actions` = nothing here since the kebab is in the header — or move the kebab into IdentitySection's `actions` slot to match task-177; choose the header placement per the prototype), `ClassAppearanceSection`, `SpawnProgressionSection`, `BaseStatsSection`, `EquipmentSection`, `InventorySection`, `SkillsSection`; right `order-1 lg:order-2` holds `<PresetPreviewCard attrs={preset.attributes} />`.
- Wire each section's callbacks to the corresponding `on*` props, mapping stat edits to `onSetField("stats." + stat, value)` and appearance/identity edits to `onSetField(path, value)`.

- [ ] **Step 5: Run test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/PresetEditor.test.tsx`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/presets/PresetActionsMenu.tsx services/atlas-ui/src/components/features/characters/presets/PresetEditor.tsx services/atlas-ui/src/components/features/characters/presets/__tests__/PresetEditor.test.tsx
git commit -m "feat(task-180): focused preset editor + actions menu"
```

---

### Task 16: `PresetCard.tsx` — library card

**Files:**
- Create: `.../presets/PresetCard.tsx`
- Test: `.../presets/__tests__/PresetCard.test.tsx`

**Interfaces:**
- Consumes: `jobLabel` (`./presetJobs`); `buildPresetLoadout` + `useCharacterImage` (or a lightweight card renderer) for the sprite; `useTenant()`.
- Produces:
  ```ts
  interface PresetCardProps {
    preset: WorkingPreset;
    dirty: boolean;
    onOpen: () => void;
    onDuplicate: () => void;
    onApply?: () => void; // hover quick-action + only when apply capability present
  }
  ```

- [ ] **Step 1: Write the failing test**

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { PresetCard } from "../PresetCard";
import { DEFAULT_PRESET_ATTRIBUTES } from "../presetEditorState";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } }),
}));
vi.mock("@/lib/hooks/useCharacterImage", () => ({
  useCharacterImage: () => ({ isLoading: false, isError: false, imageUrl: "http://img/c.png", refetch: vi.fn() }),
}));

const preset = {
  key: "a1", id: "a1",
  attributes: { ...DEFAULT_PRESET_ATTRIBUTES, name: "Test Warrior", jobId: 100, level: 30, gm: 0, description: "A tank", tags: ["PvE"] },
};

describe("PresetCard", () => {
  it("shows name, job badge, level, description, tags", () => {
    render(<PresetCard preset={preset} dirty={false} onOpen={vi.fn()} onDuplicate={vi.fn()} />);
    expect(screen.getByText("Test Warrior")).toBeInTheDocument();
    expect(screen.getByText("Warrior")).toBeInTheDocument();
    expect(screen.getByText(/Lv 30/)).toBeInTheDocument();
    expect(screen.getByText("A tank")).toBeInTheDocument();
    expect(screen.getByText("PvE")).toBeInTheDocument();
  });

  it("shows a dirty-dot when dirty", () => {
    render(<PresetCard preset={preset} dirty onOpen={vi.fn()} onDuplicate={vi.fn()} />);
    expect(screen.getByTestId("dirty-dot")).toBeInTheDocument();
  });

  it("opening (click/Enter) fires onOpen; Duplicate fires onDuplicate", async () => {
    const onOpen = vi.fn();
    const onDuplicate = vi.fn();
    render(<PresetCard preset={preset} dirty={false} onOpen={onOpen} onDuplicate={onDuplicate} />);
    await userEvent.click(screen.getByRole("button", { name: /open preset Test Warrior/i }));
    expect(onOpen).toHaveBeenCalled();
    await userEvent.click(screen.getByRole("button", { name: /duplicate/i }));
    expect(onDuplicate).toHaveBeenCalled();
  });

  it("hides Apply quick-action when onApply is absent", () => {
    render(<PresetCard preset={preset} dirty={false} onOpen={vi.fn()} onDuplicate={vi.fn()} />);
    expect(screen.queryByRole("button", { name: /apply to account/i })).toBeNull();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/PresetCard.test.tsx`
Expected: FAIL — cannot resolve `../PresetCard`.

- [ ] **Step 3: Implement**

Keyboard-activatable card (a `role="button"` container or an inner `<button aria-label={`Open preset ${name}`}>` around the sprite/text) firing `onOpen`. Content: rendered sprite on a tinted stage (`useCharacterImage` on `buildPresetLoadout(preset.attributes)`, skeleton/error handled), name, job badge `jobLabel(jobId)`, `Lv {level}` (`· GM {gm}` when `gm > 0`), 2-line-clamped description, tag chips. A `data-testid="dirty-dot"` indicator when `dirty`. Hover/focus quick-actions overlay: a Duplicate button (`aria-label="Duplicate"`) → `onDuplicate`, and — only when `onApply` — an Apply button (`aria-label="Apply to account"`) → `onApply`. Quick-action clicks must `stopPropagation` so they don't also trigger `onOpen`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/PresetCard.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/presets/PresetCard.tsx services/atlas-ui/src/components/features/characters/presets/__tests__/PresetCard.test.tsx
git commit -m "feat(task-180): preset library card"
```

---

### Task 17: `LibraryToolbar.tsx` + `NewPresetCard.tsx` + `PresetLibrary.tsx`

Search + single-select tag filter + New affordances + responsive grid + empty state.

**Files:**
- Create: `.../presets/LibraryToolbar.tsx`
- Create: `.../presets/NewPresetCard.tsx`
- Create: `.../presets/PresetLibrary.tsx`
- Test: `.../presets/__tests__/PresetLibrary.test.tsx`

**Interfaces:**
- Consumes: `PresetCard` (Task 16), `presetDirty` (`./presetEditorState`), `EmptyState`.
- Produces:
  ```ts
  interface PresetLibraryProps {
    presets: WorkingPreset[];
    dirtyKeys: Set<string>;          // keys with unsaved edits (from presetDirty)
    canApply: boolean;               // apply capability present
    onOpen: (key: string) => void;
    onNew: () => void;
    onDuplicate: (key: string) => void;
    onApply: (key: string) => void;
  }
  ```
  `LibraryToolbar` props: `{ query; onQuery; tags: string[]; activeTag: string | null; onTag; onNew }`. `NewPresetCard` props: `{ onNew: () => void }`.

- [ ] **Step 1: Write the failing test**

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { PresetLibrary } from "../PresetLibrary";
import { DEFAULT_PRESET_ATTRIBUTES } from "../presetEditorState";

vi.mock("../PresetCard", () => ({
  PresetCard: ({ preset, onOpen }: { preset: { attributes: { name: string } }; onOpen: () => void }) => (
    <button onClick={onOpen}>{preset.attributes.name}</button>
  ),
}));

const mk = (key: string, name: string, tags: string[], description = "") => ({
  key, id: key, attributes: { ...DEFAULT_PRESET_ATTRIBUTES, name, tags, description },
});
const presets = [
  mk("a", "Fresh Beginner", ["starter"], "level 1 blank"),
  mk("b", "Test Warrior", ["combat"], "a tank"),
  mk("c", "GM Admin", ["staff"], "godmode"),
];
const base = { dirtyKeys: new Set<string>(), canApply: true, onOpen: vi.fn(), onNew: vi.fn(), onDuplicate: vi.fn(), onApply: vi.fn() };

describe("PresetLibrary", () => {
  it("search matches name/description/tags case-insensitively", async () => {
    render(<PresetLibrary presets={presets} {...base} />);
    await userEvent.type(screen.getByRole("searchbox"), "warrior");
    expect(screen.getByText("Test Warrior")).toBeInTheDocument();
    expect(screen.queryByText("GM Admin")).toBeNull();
  });

  it("single-select tag filter narrows the grid", async () => {
    render(<PresetLibrary presets={presets} {...base} />);
    await userEvent.click(screen.getByRole("button", { name: /^staff$/i }));
    expect(screen.getByText("GM Admin")).toBeInTheDocument();
    expect(screen.queryByText("Test Warrior")).toBeNull();
  });

  it("renders the + New affordance and fires onNew", async () => {
    const onNew = vi.fn();
    render(<PresetLibrary presets={presets} {...base} onNew={onNew} />);
    await userEvent.click(screen.getByRole("button", { name: /new preset/i }));
    expect(onNew).toHaveBeenCalled();
  });

  it("empty state when no presets", () => {
    render(<PresetLibrary presets={[]} {...base} />);
    expect(screen.getByText(/no character presets/i)).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/PresetLibrary.test.tsx`
Expected: FAIL — cannot resolve `../PresetLibrary`.

- [ ] **Step 3: Implement `NewPresetCard.tsx`**: a dashed-border card button (`aria-label="New preset"`) → `onNew`, appended after the grid.

- [ ] **Step 4: Implement `LibraryToolbar.tsx`**: a search `Input` (`type="search"`, `role="searchbox"`, `aria-label="Search presets"`) bound to `query`/`onQuery`; a tag chip row — `All` chip + one chip per distinct tag (`aria-label={tag}`), single-select, active chip highlighted, toggling `onTag(tag|null)`; a `+ New preset` Button (`aria-label="New preset"`) → `onNew`.

- [ ] **Step 5: Implement `PresetLibrary.tsx`**:
- Empty state (`presets.length === 0`): shared `EmptyState` titled "No character presets" with an "Add preset" action → `onNew`.
- Derive `tags` = distinct tags across all presets. Local state `query`, `activeTag`.
- Filter: a preset matches when (`query` empty OR its lowercased name/description/tags contain the lowercased query) AND (`activeTag` null OR its tags include `activeTag`).
- Render `LibraryToolbar` then a responsive grid of `PresetCard` for each filtered preset (`dirty={dirtyKeys.has(key)}`, `onOpen={() => onOpen(key)}`, `onDuplicate={() => onDuplicate(key)}`, `onApply` only when `canApply` → `() => onApply(key)`), followed by `NewPresetCard`.

- [ ] **Step 6: Run test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/PresetLibrary.test.tsx`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/presets/LibraryToolbar.tsx services/atlas-ui/src/components/features/characters/presets/NewPresetCard.tsx services/atlas-ui/src/components/features/characters/presets/PresetLibrary.tsx services/atlas-ui/src/components/features/characters/presets/__tests__/PresetLibrary.test.tsx
git commit -m "feat(task-180): preset card-library landing view"
```

---

### Task 18: `AccountPickerDialog.tsx` — apply flow account search

**Files:**
- Create: `.../presets/AccountPickerDialog.tsx`
- Test: `.../presets/__tests__/AccountPickerDialog.test.tsx`

**Interfaces:**
- Consumes: `useAccountSearch(tenant, namePattern): { data?: Account[]; isLoading; isError }` (`@/lib/hooks/api/useAccounts`); `Account` (`@/types/models/account`); `Tenant`.
- Produces:
  ```ts
  interface AccountPickerDialogProps {
    tenant: Tenant;
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onPick: (accountId: number) => void; // Account.id is a string → Number(id)
  }
  ```
  Debounced name search; empty/loading/error states; selecting a result fires `onPick(Number(account.id))` and closes.

- [ ] **Step 1: Write the failing test**

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { AccountPickerDialog } from "../AccountPickerDialog";

const useAccountSearchMock = vi.fn();
vi.mock("@/lib/hooks/api/useAccounts", () => ({
  useAccountSearch: (...a: unknown[]) => useAccountSearchMock(...a),
}));

const tenant = { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } as never;

describe("AccountPickerDialog", () => {
  it("searches and picks an account (id → number)", async () => {
    useAccountSearchMock.mockReturnValue({
      data: [{ id: "42", attributes: { name: "operator" } }],
      isLoading: false, isError: false,
    });
    const onPick = vi.fn();
    render(<AccountPickerDialog tenant={tenant} open onOpenChange={vi.fn()} onPick={onPick} />);
    await userEvent.type(screen.getByRole("searchbox"), "oper");
    await userEvent.click(await screen.findByRole("button", { name: /operator/i }));
    expect(onPick).toHaveBeenCalledWith(42);
  });

  it("shows empty state when no results", () => {
    useAccountSearchMock.mockReturnValue({ data: [], isLoading: false, isError: false });
    render(<AccountPickerDialog tenant={tenant} open onOpenChange={vi.fn()} onPick={vi.fn()} />);
    expect(screen.getByText(/no accounts/i)).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/AccountPickerDialog.test.tsx`
Expected: FAIL — cannot resolve `../AccountPickerDialog`.

- [ ] **Step 3: Implement**

shadcn `Dialog` titled "Select an account". A search `Input` (`role="searchbox"`, `aria-label="Search accounts"`); debounce the value (e.g. a small `useEffect` + `setTimeout(200ms)` into `debounced` state) and pass it to `useAccountSearch(tenant, debounced)`. Render: loading → a small skeleton/"Searching…"; error → `ErrorDisplay`; empty (non-empty query, no rows) → "No accounts match." ; results → a selectable list, each a `<button>` showing `account.attributes.name` (+ id) firing `onPick(Number(account.id))` then `onOpenChange(false)`. Before any query, prompt "Type to search accounts."

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/AccountPickerDialog.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/presets/AccountPickerDialog.tsx services/atlas-ui/src/components/features/characters/presets/__tests__/AccountPickerDialog.test.tsx
git commit -m "feat(task-180): account picker dialog for apply flow"
```

---

### Task 19: `CharacterPresetsEditor.tsx` — container (reducer + URL sync + action bar + apply orchestration)

The orchestrator that composes the whole feature and defines the adapter interface consumed by the page wrappers.

**Files:**
- Create: `.../presets/CharacterPresetsEditor.tsx`
- Test: `.../presets/__tests__/CharacterPresetsEditor.test.tsx`

**Interfaces:**
- Consumes: everything above; `useReducer`, `useSearchParams`; `useRegisterDetailActionBar`; `presetSchema` (`@/lib/schemas/character-presets.schema`) for save-time validation; `ApplyPresetDialog` + `AccountPickerDialog`; `Tenant` (`@/types/models/tenant`).
- Produces:
  ```ts
  export interface PresetsEditorAdapter {
    presets: CharacterPreset[] | undefined;
    isLoading: boolean;
    error: Error | null;
    save: (presets: CharacterPreset[], onSuccess: () => void) => void;
    isSaving: boolean;
    apply?: { tenant: Tenant };
  }
  export function CharacterPresetsEditor({ adapter }: { adapter: PresetsEditorAdapter }): JSX.Element;
  ```

**URL-sync contract (mirrors task-177, key-addressed):**
- Seed-once effect (`deps: [adapter.presets, state.loaded]`): first data → `dispatch({type:"load"})`. The `loaded` guard keeps the reducer authoritative against post-save refetches.
- Deep-link-on-load effect (`deps: [state.loaded]`, runs once): read `?preset=`; resolve against a working preset by `id` **then** `key`; if matched, `dispatch({type:"select", key})`; else leave `selectedKey = null`. Never errors on stale/unknown ids.
- `syncSelection(key: string | null)` owns ALL URL writes with `{ replace: true }`: sets `?preset=<resolvedIdOrKey>` when selecting, **deletes** the param when returning to the library. Every internal mutation handler (`select`, `add`, `duplicate`, `remove`, `discard`) calls `syncSelection` with the reducer's own post-mutation selection — no length-watching effect. When selecting, prefer the preset's `id` in the URL, falling back to its `key`.

- [ ] **Step 1: Write the failing test**

```tsx
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { CharacterPresetsEditor, type PresetsEditorAdapter } from "../CharacterPresetsEditor";
import { DEFAULT_PRESET_ATTRIBUTES } from "../presetEditorState";

// Mock the action bar + heavy leaves; keep library + editor real enough to assert flow.
vi.mock("@/components/DetailActionBarContext", () => ({ useRegisterDetailActionBar: vi.fn() }));
vi.mock("../PresetEditor", () => ({
  PresetEditor: ({ preset, onBack }: { preset: { attributes: { name: string } }; onBack: () => void }) => (
    <div><span>editor:{preset.attributes.name}</span><button onClick={onBack}>back</button></div>
  ),
}));
vi.mock("../PresetLibrary", () => ({
  PresetLibrary: ({ presets, onOpen, onNew }: { presets: { key: string; attributes: { name: string } }[]; onOpen: (k: string) => void; onNew: () => void }) => (
    <div>
      {presets.map((p) => <button key={p.key} onClick={() => onOpen(p.key)}>open:{p.attributes.name}</button>)}
      <button onClick={onNew}>new</button>
    </div>
  ),
}));

const presets = [
  { id: "a1", attributes: { ...DEFAULT_PRESET_ATTRIBUTES, name: "One" } },
  { id: "b2", attributes: { ...DEFAULT_PRESET_ATTRIBUTES, name: "Two" } },
];
const adapter = (over: Partial<PresetsEditorAdapter> = {}): PresetsEditorAdapter => ({
  presets, isLoading: false, error: null, isSaving: false, save: vi.fn(), ...over,
});

const renderAt = (url: string, a: PresetsEditorAdapter) =>
  render(<MemoryRouter initialEntries={[url]}><CharacterPresetsEditor adapter={a} /></MemoryRouter>);

describe("CharacterPresetsEditor", () => {
  it("shows the library when no ?preset=", () => {
    renderAt("/", adapter());
    expect(screen.getByText("open:One")).toBeInTheDocument();
    expect(screen.queryByText(/^editor:/)).toBeNull();
  });

  it("deep-links ?preset=<id> into the focused editor", async () => {
    renderAt("/?preset=b2", adapter());
    expect(await screen.findByText("editor:Two")).toBeInTheDocument();
  });

  it("unresolvable ?preset= falls back to library without error", () => {
    renderAt("/?preset=nope", adapter());
    expect(screen.getByText("open:One")).toBeInTheDocument();
  });

  it("opening a card then back toggles editor/library", async () => {
    renderAt("/", adapter());
    await userEvent.click(screen.getByText("open:Two"));
    expect(await screen.findByText("editor:Two")).toBeInTheDocument();
    await userEvent.click(screen.getByText("back"));
    await waitFor(() => expect(screen.getByText("open:One")).toBeInTheDocument());
  });

  it("registers the action bar and Save projects the array (id-only, no key)", async () => {
    const { useRegisterDetailActionBar } = await import("@/components/DetailActionBarContext");
    const save = vi.fn();
    renderAt("/", adapter({ save }));
    // grab the last registration's onSave
    const calls = (useRegisterDetailActionBar as unknown as { mock: { calls: unknown[][] } }).mock.calls;
    const reg = calls.map((c) => c[0]).filter(Boolean).at(-1) as { onSave: () => void };
    reg.onSave();
    expect(save).toHaveBeenCalledWith(
      [
        { id: "a1", attributes: expect.objectContaining({ name: "One" }) },
        { id: "b2", attributes: expect.objectContaining({ name: "Two" }) },
      ],
      expect.any(Function),
    );
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/CharacterPresetsEditor.test.tsx`
Expected: FAIL — cannot resolve `../CharacterPresetsEditor`.

- [ ] **Step 3: Implement**

- `useReducer(presetReducer, undefined, initialPresetEditorState)`; `useSearchParams`.
- Seed-once + deep-link-on-load effects and `syncSelection` per the URL-sync contract above.
- Handlers: `open(key)` → `dispatch(select) + syncSelection(resolvedIdOrKey)`; `back()` → `dispatch({type:"select", key:null}) + syncSelection(null)`; `newPreset()` → `dispatch(addPreset)` then `syncSelection` with the newly-selected key (read it from the returned state via a post-dispatch `useEffect` on `state.selectedKey`, or compute the key as `local-${state.localSeq}` before dispatch — prefer syncing from `selectedKey` in a dedicated effect that mirrors, NOT watches length); `duplicate(key)`, `remove(key)`, `discard()` similarly sync from the reducer's resulting `selectedKey`.

  > Implementation note to avoid the task-177 router race: keep a single effect `useEffect(() => { syncSelection(currentSelectionUrlValue) }, [state.selectedKey])` is acceptable ONLY if it derives the URL value purely from `state.selectedKey` + the preset's id — but the safer, task-177-proven approach is to call `syncSelection` inside each handler with the value the reducer will land on. Since the reducer's post-mutation `selectedKey` isn't known synchronously in the handler, resolve it: for `open`/`duplicate`/`add` you know the target key; for `remove`/`discard` the target is `null`. Use those known values.

- Dirty: `isDirty(state)`; per-preset `dirtyKeys = new Set(state.presets.filter(p => presetDirty(state, p.key)).map(p => p.key))`.
- Action bar: `useRegisterDetailActionBar(state.loaded && state.presets.length >= 0 ? { dirty, isSaving: adapter.isSaving, onSave, onDiscard: discard } : null)`. `onSave` validates each `projectForSave(state)` entry with `presetSchema` (surface the first failure as a toast and abort if invalid — reuse the deleted form's field-error mapping shape); on valid, `adapter.save(projectForSave(state), () => dispatch({type:"savedOk"}))`. Field-level API errors from the mutation's `onError` map `meta.path` `presets[<id>].<field>` to a toast referencing that preset's name (the mutation onError lives in the page adapter's `save`, so pass through a toast; deep field-highlighting is best-effort — a toast naming preset+field satisfies FR-9.3).
- Seed-once gate: while `!state.loaded`, show `adapter.error ? <ErrorDisplay/> : <FormSkeleton fields={6}/>`.
- Render: `selectedPreset(state)` → `<PresetEditor .../>` with all `on*` handlers wired to dispatch, `onApply` present ONLY when `adapter.apply` AND `preset.id` (else undefined); otherwise `<PresetLibrary .../>` with `canApply={!!adapter.apply}`, `onApply={(key) => startApply(key)}`.
- **Apply orchestration** (only when `adapter.apply`): local state `applyKey: string | null`, `applyAccountId: number | null`. `startApply(key)` sets `applyKey` (if the target preset has no `id`, toast "Save this preset before applying" and abort; if the preset is dirty, toast "Apply uses the last saved version" — non-blocking) and opens `<AccountPickerDialog tenant={adapter.apply.tenant} .../>`. On `onPick(accountId)` → set `applyAccountId`, close the picker, open `<ApplyPresetDialog tenant={adapter.apply.tenant} accountId={accountId} initialPresetId={selectedPresetIdForApplyKey} open onOpenChange=.../>`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/components/features/characters/presets/__tests__/CharacterPresetsEditor.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/presets/CharacterPresetsEditor.tsx services/atlas-ui/src/components/features/characters/presets/__tests__/CharacterPresetsEditor.test.tsx
git commit -m "feat(task-180): CharacterPresetsEditor container with URL sync + apply orchestration"
```

---

### Task 20: Page wrappers rewrite + deletions

Wire both pages to the shared editor; delete the two duplicated forms and their orphaned test.

**Files:**
- Modify: `services/atlas-ui/src/pages/TenantsCharacterPresetsPage.tsx`
- Modify: `services/atlas-ui/src/pages/TemplatesCharacterPresetsPage.tsx`
- Delete: `services/atlas-ui/src/pages/tenants-character-presets-form.tsx`
- Delete: `services/atlas-ui/src/pages/templates-character-presets-form.tsx`
- Delete: `services/atlas-ui/src/pages/__tests__/templates-character-presets-form.test.tsx` (tests the deleted form)
- Delete (dead re-export shim, no importers): `services/atlas-ui/src/pages/character-presets-schema.ts`
- Test: `services/atlas-ui/src/pages/__tests__/TenantsCharacterPresetsPage.test.tsx`, `.../TemplatesCharacterPresetsPage.test.tsx`

**Interfaces:**
- Consumes: `CharacterPresetsEditor`, `PresetsEditorAdapter` (Task 19); `useTenantConfiguration`/`useUpdateTenantConfiguration`/`useTenant` (tenant); `useTemplate`/`useUpdateTemplate` (template).
- Produces: unchanged exports `TenantsCharacterPresetsPage`, `TemplatesCharacterPresetsPage` (App.tsx lazy-imports them by name — must not rename).

- [ ] **Step 1: Write the failing adapter tests** (mirror `pages/__tests__/TenantsCharacterTemplatesPage.test.tsx`)

Tenant page test — assert (a) the editor receives an adapter whose `save` PATCHes `{ characters: { ...existing, presets } }` preserving the sibling `templates` array, and (b) `apply.tenant` is supplied:

```tsx
// mock CharacterPresetsEditor to capture props; mock useTenantConfiguration/
// useUpdateTenantConfiguration/useTenant/sonner + TenantDetailLayout as in the
// templates page test. Then:
it("save spreads characters so the sibling templates array survives", () => {
  render(<MemoryRouter><TenantsCharacterPresetsPage /></MemoryRouter>);
  const adapter = editorMock.mock.calls.at(-1)[0].adapter;
  adapter.save([{ attributes: { name: "P" } }], () => {});
  expect(mutateMock).toHaveBeenCalledWith(
    expect.objectContaining({
      updates: { characters: expect.objectContaining({ templates: expect.any(Array), presets: [{ attributes: { name: "P" } }] }) },
    }),
    expect.anything(),
  );
});
it("supplies apply.tenant capability", () => {
  render(<MemoryRouter><TenantsCharacterPresetsPage /></MemoryRouter>);
  const adapter = editorMock.mock.calls.at(-1)[0].adapter;
  expect(adapter.apply?.tenant).toBeTruthy();
});
```

Template page test — same `templates`-survival assertion via `useTemplate`/`useUpdateTemplate`, and assert the adapter has **no** `apply` capability.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-ui && npx vitest run src/pages/__tests__/TenantsCharacterPresetsPage.test.tsx src/pages/__tests__/TemplatesCharacterPresetsPage.test.tsx`
Expected: FAIL — pages still render the old forms / adapter not shaped yet.

- [ ] **Step 3: Rewrite `TenantsCharacterPresetsPage.tsx`**

```tsx
import { useParams } from "react-router-dom";
import { toast } from "sonner";
import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";
import {
  CharacterPresetsEditor,
  type PresetsEditorAdapter,
} from "@/components/features/characters/presets/CharacterPresetsEditor";
import {
  useTenantConfiguration,
  useUpdateTenantConfiguration,
  useTenant,
} from "@/lib/hooks/api/useTenants";

export function TenantsCharacterPresetsPage() {
  const { id } = useParams();
  const tenantQuery = useTenantConfiguration(id ?? "");
  const updateTenantConfig = useUpdateTenantConfiguration();
  const tenantBasicQuery = useTenant(id ?? "");
  const tenant = tenantQuery.data;

  const adapter: PresetsEditorAdapter = {
    presets: tenant?.attributes.characters.presets,
    isLoading: tenantQuery.isLoading,
    error: tenantQuery.error ?? null,
    isSaving: updateTenantConfig.isPending,
    ...(tenantBasicQuery.data ? { apply: { tenant: tenantBasicQuery.data } } : {}),
    save: (presets, onSuccess) => {
      if (!tenant) return;
      updateTenantConfig.mutate(
        { tenant, updates: { characters: { ...tenant.attributes.characters, presets } } },
        {
          onSuccess: () => { toast.success("Successfully saved presets."); onSuccess(); },
          onError: (error) => toast.error(`Failed to update presets: ${error.message}`),
        },
      );
    },
  };

  return (
    <TenantDetailLayout>
      <CharacterPresetsEditor adapter={adapter} />
    </TenantDetailLayout>
  );
}
```

- [ ] **Step 4: Rewrite `TemplatesCharacterPresetsPage.tsx`** (mirror, `useTemplate`/`useUpdateTemplate`, spreading `template.attributes.characters`, **no** `apply`):

```tsx
import { useParams } from "react-router-dom";
import { toast } from "sonner";
import { TemplateDetailLayout } from "@/components/features/templates/TemplateDetailLayout";
import {
  CharacterPresetsEditor,
  type PresetsEditorAdapter,
} from "@/components/features/characters/presets/CharacterPresetsEditor";
import { useTemplate, useUpdateTemplate } from "@/lib/hooks/api/useTemplates";

export function TemplatesCharacterPresetsPage() {
  const { id } = useParams();
  const templateQuery = useTemplate(String(id ?? ""));
  const updateTemplate = useUpdateTemplate();
  const template = templateQuery.data;

  const adapter: PresetsEditorAdapter = {
    presets: template?.attributes.characters.presets,
    isLoading: templateQuery.isLoading,
    error: templateQuery.error ?? null,
    isSaving: updateTemplate.isPending,
    save: (presets, onSuccess) => {
      if (!template) return;
      updateTemplate.mutate(
        { id: template.id, updates: { characters: { ...template.attributes.characters, presets } } },
        {
          onSuccess: () => { toast.success("Successfully saved template."); onSuccess(); },
          onError: (error) => toast.error(`Failed to update template: ${error.message}`),
        },
      );
    },
  };

  return (
    <TemplateDetailLayout>
      <CharacterPresetsEditor adapter={adapter} />
    </TemplateDetailLayout>
  );
}
```

- [ ] **Step 5: Delete the obsolete files**

```bash
git rm services/atlas-ui/src/pages/tenants-character-presets-form.tsx \
       services/atlas-ui/src/pages/templates-character-presets-form.tsx \
       services/atlas-ui/src/pages/__tests__/templates-character-presets-form.test.tsx \
       services/atlas-ui/src/pages/character-presets-schema.ts
```

(If `git rm` reports `character-presets-schema.ts` still referenced, re-grep `grep -rn "pages/character-presets-schema" services/atlas-ui/src` and remove the importer first — the pre-task grep found none.)

- [ ] **Step 6: Run tests to verify they pass + the whole preset/adapter surface**

Run: `cd services/atlas-ui && npx vitest run src/pages/__tests__/TenantsCharacterPresetsPage.test.tsx src/pages/__tests__/TemplatesCharacterPresetsPage.test.tsx`
Expected: PASS (both survival + capability assertions).

- [ ] **Step 7: Commit**

```bash
git add -A services/atlas-ui/src/pages
git commit -m "feat(task-180): wire preset pages to shared editor; delete duplicated forms"
```

---

### Task 21: Full verification & cleanup gate

**Files:** none (verification only). Fix any failures found before committing the gate.

- [ ] **Step 1: Ensure Node 22 and install**

Run (from worktree root):
```bash
cd services/atlas-ui
. "$NVM_DIR/nvm.sh" 2>/dev/null || source ~/.nvm/nvm.sh
nvm use 22
npm ci  # or npm install if lockfile unchanged
```
Expected: node v22.x active.

- [ ] **Step 2: Full test suite**

Run: `cd services/atlas-ui && npm run test`
Expected: PASS — all preset suites plus the untouched task-177 templates suites (regression gate for the shared-file edits) green.

- [ ] **Step 3: Type-check + build**

Run: `cd services/atlas-ui && npm run build`
Expected: clean. This type-checks new `*.test.ts(x)` under `tsc -b` (per the PRD/CLAUDE note that a green test ≠ a green build). Fix any type errors.

- [ ] **Step 4: Lint (no new errors vs. baseline)**

Run: `cd services/atlas-ui && npm run lint`
Expected: no NEW errors attributable to task-180 files. Compare against the known pre-existing lint baseline; do not introduce new violations.

- [ ] **Step 5: Repo-root shared lint**

Run (from worktree root): `tools/lint.sh --check`
Expected: clean. If it reports Prettier/ESLint diffs in touched files, run `tools/lint.sh` (fix mode) and re-check.

- [ ] **Step 6: Manual smoke (optional but recommended)**

Per `superpowers:verification-before-completion`: run `npm run dev`, open a tenant's Character Presets page — verify the library renders cards, `?preset=<id>` deep-links the editor, appearance thumbs update the preview, Save preserves the sibling templates array (check the PATCH payload in the network tab), and the template page hides Apply. Capture nothing that requires a live backend beyond what the existing app already serves.

- [ ] **Step 7: Commit any fixes**

```bash
git add -A
git commit -m "chore(task-180): satisfy build/lint/test verification gate"
```

---

## Self-Review

**1. Spec coverage** (design §§ + PRD FRs → task):
- §4/§11 shared component + adapter, delete forms, zero shell → Tasks 19, 20. FR-1 ✓
- §4 card library + §9 landing (search/tag/+New/dirty-dot/empty/card order) → Tasks 16, 17. FR-2 ✓
- §10 apply-to-account (account picker → ApplyPresetDialog scoped) → Tasks 6, 18, 19. FR-3 ✓
- §9 Identity (name/defaultName/description/tags) → Task 8. FR-4 ✓
- §9/§8 Class & appearance (named job picker + advanced numeric, gender, single-select thumbs + generalized browser) → Tasks 1, 4, 9. FR-5 ✓
- §9 Spawn & progression (MapPicker + level/GM/meso) → Task 10. FR-6 ✓
- §9 Base stats (verbatim) → Task 11. FR-7 ✓
- §9 Equipment (avg toggle + subcat search + manual id) / Inventory (qty) / Skills (numeric-id + lookup, no browser) → Tasks 5, 12, 13, 14. FR-8 ✓
- §7 live preview + §5 state/save (reducer, one array, free switching, sticky save bar, +New/Duplicate defaults, field-error mapping) → Tasks 2, 3, 7, 15, 19. FR-9 ✓
- §6 URL sync `?preset=` (deep-link, unresolvable→library, session key) → Task 19. FR-10 ✓
- §2/FR-11 theming/pixelated/keyboard → enforced per-component (Global Constraints) + Task 21. ✓
- §13 testing (reducer, library, card actions, URL sync, sections, apply handoff, both adapters) → each component task + Tasks 19, 20. PRD §8 ✓
- PRD Acceptance verification gate → Task 21. ✓

**2. Placeholder scan:** No "TBD/implement later/handle edge cases". The two "read the reference component before wiring" notes (ItemSearchCombobox props in Task 12/13; useSkillData shape in Task 14) are explicit, bounded lookups against named existing files, not deferred design — the exact reference call sites (`templates/EquipmentPoolSection.tsx`, `templates/StartingKitSection.tsx`) are named. Acceptable.

**3. Type consistency:** `WorkingPreset`, `PresetEditorState`, `PresetFieldPath`, `PresetEditorAction`, `DEFAULT_PRESET_ATTRIBUTES`, `projectForSave`, `selectedPreset`, `presetDirty`, `isDirty` (Task 2) are used consistently in Tasks 15/16/17/19. `buildPresetLoadout`/`buildPresetVariantLoadout`/`wornTemplateIds` + `PresetAppearanceDimension` (Task 3) match uses in Tasks 7/9/16. `PresetsEditorAdapter` (Task 19) matches the page wrappers (Task 20). `AppearanceBrowserDialog`'s new props (`gender`/`variantLoadout`/`onSelect`/`selectMode`/`markedIds`/`selectedId`) are consistent between the templates call site (Task 4) and the preset call site (Task 9). `ItemRow.trailing` (Task 5) is consumed in Tasks 12/13. `ApplyPresetDialog.initialPresetId` (Task 6) is consumed in Task 19. Section prop callback names (`onSetField`/`onSetStat`/`onAdd`/`onRemove`/`onSetAvg`/`onSetQty`/`onSetLevel`/`onAddTag`/`onRemoveTag`) are consistent between each section task and Task 15's `PresetEditor` wiring.
