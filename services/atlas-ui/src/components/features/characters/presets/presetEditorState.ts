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
  | {
      type: "setField";
      key: string;
      path: PresetFieldPath;
      value: number | string;
    }
  | { type: "addEquip"; key: string; templateId: number }
  | { type: "removeEquip"; key: string; index: number }
  | { type: "setEquipAvg"; key: string; index: number; value: boolean }
  | { type: "addInventory"; key: string; templateId: number }
  | { type: "removeInventory"; key: string; index: number }
  | { type: "setInventoryQty"; key: string; index: number; value: number }
  | { type: "addSkill"; key: string; skillId: number }
  | { type: "addSkills"; key: string; skillIds: number[] }
  | { type: "removeSkill"; key: string; index: number }
  | { type: "setSkillLevel"; key: string; index: number; value: number }
  | { type: "addTag"; key: string; tag: string }
  | { type: "removeTag"; key: string; tag: string }
  | { type: "discard" }
  | { type: "savedOk"; persisted?: CharacterPreset[] };

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

function cloneAttributes(
  a: CharacterPresetAttributes,
): CharacterPresetAttributes {
  return normalizePreset(a);
}

export function initialPresetEditorState(): PresetEditorState {
  return {
    presets: [],
    baseline: [],
    selectedKey: null,
    localSeq: 0,
    loaded: false,
  };
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
  return (
    JSON.stringify(state.presets.map(project)) !==
    JSON.stringify(state.baseline)
  );
}

export function selectedPreset(state: PresetEditorState): WorkingPreset | null {
  if (state.selectedKey === null) return null;
  return state.presets.find((p) => p.key === state.selectedKey) ?? null;
}

export function presetDirty(state: PresetEditorState, key: string): boolean {
  const p = state.presets.find((p) => p.key === key);
  if (!p) return false;
  const base =
    p.id !== undefined ? state.baseline.find((b) => b.id === p.id) : undefined; // no id => freshly added, never had a baseline counterpart
  if (!base) return true;
  return JSON.stringify(project(p)) !== JSON.stringify(base);
}

function updateOne(
  state: PresetEditorState,
  key: string,
  update: (a: CharacterPresetAttributes) => CharacterPresetAttributes,
): PresetEditorState {
  const presets = state.presets.map((p) =>
    p.key === key
      ? { ...p, attributes: update(cloneAttributes(p.attributes)) }
      : p,
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
      const presets: WorkingPreset[] = action.presets.map((p) =>
        p.id !== undefined
          ? { key: p.id, id: p.id, attributes: normalizePreset(p.attributes) }
          : {
              key: `local-${seq++}`,
              attributes: normalizePreset(p.attributes),
            },
      );
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
      const row: WorkingPreset = {
        key,
        attributes: cloneAttributes(DEFAULT_PRESET_ATTRIBUTES),
      };
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
      const row: WorkingPreset = {
        key,
        attributes: cloneAttributes(src.attributes),
      };
      return {
        ...state,
        presets: [...state.presets, row],
        localSeq: state.localSeq + 1,
        selectedKey: key,
      };
    }
    case "removePreset": {
      const presets = state.presets.filter((p) => p.key !== action.key);
      const selectedKey =
        state.selectedKey === action.key ? null : state.selectedKey;
      return { ...state, presets, selectedKey };
    }
    case "setField":
      return updateOne(state, action.key, (a) => {
        if (action.path.startsWith("stats.")) {
          const stat = action.path.slice(
            "stats.".length,
          ) as keyof typeof a.stats;
          return {
            ...a,
            stats: { ...a.stats, [stat]: action.value as number },
          };
        }
        if (action.path === "gender") {
          return { ...a, gender: Number(action.value) === 1 ? 1 : 0 };
        }
        return {
          ...a,
          [action.path]: action.value,
        } as CharacterPresetAttributes;
      });
    case "addEquip":
      return updateOne(state, action.key, (a) => ({
        ...a,
        equipment: [
          ...a.equipment,
          { templateId: action.templateId, useAverageStats: true },
        ],
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
        inventory: [
          ...a.inventory,
          { templateId: action.templateId, quantity: 1 },
        ],
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
          i === action.index
            ? { ...e, quantity: Math.max(1, action.value) }
            : e,
        ),
      }));
    case "addSkill":
      return updateOne(state, action.key, (a) => ({
        ...a,
        skills: [...a.skills, { skillId: action.skillId, level: 1 }],
      }));
    case "addSkills":
      // Batch add (e.g. a whole job family) — skip ids already granted and
      // any duplicate within the incoming list; each new skill starts at 1.
      return updateOne(state, action.key, (a) => {
        const seen = new Set(a.skills.map((s) => s.skillId));
        const added: { skillId: number; level: number }[] = [];
        for (const id of action.skillIds) {
          if (id > 0 && !seen.has(id)) {
            seen.add(id);
            added.push({ skillId: id, level: 1 });
          }
        }
        return added.length ? { ...a, skills: [...a.skills, ...added] } : a;
      });
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
        a.tags.includes(action.tag)
          ? a
          : { ...a, tags: [...a.tags, action.tag] },
      );
    case "removeTag":
      return updateOne(state, action.key, (a) => ({
        ...a,
        tags: a.tags.filter((t) => t !== action.tag),
      }));
    case "discard": {
      let seq = 0;
      const presets: WorkingPreset[] = state.baseline.map((p) =>
        p.id !== undefined
          ? { key: p.id, id: p.id, attributes: normalizePreset(p.attributes) }
          : {
              key: `local-${seq++}`,
              attributes: normalizePreset(p.attributes),
            },
      );
      return { ...state, presets, localSeq: seq, selectedKey: null };
    }
    case "savedOk": {
      // Positional match is safe here: `persisted` is whatever the adapter's
      // save() success callback hands back, and the container always sends
      // presets via projectForSave() in array order — the server is expected
      // to echo the same order back, so index i in `persisted` corresponds to
      // index i in state.presets. We only ever use this to backfill an id
      // that is still undefined (a freshly-created preset); an id that is
      // already set is left untouched, and key/attributes/selectedKey are
      // never touched here.
      //
      // Length guard: the save is async, so `state.presets` may have gained
      // or lost rows (addPreset/removePreset) between the save() call and
      // this success callback firing. If the lengths no longer match, the
      // positional correspondence assumed above no longer holds and blindly
      // zipping `persisted[i]` onto `state.presets[i]` could attach a
      // server-issued id to the WRONG working preset — silent
      // misattribution. Safely degrade instead: skip the backfill entirely
      // and just rebaseline. A newly-created preset simply stays id-less
      // until the next save/reload picks it up correctly.
      const presets =
        action.persisted && action.persisted.length === state.presets.length
          ? state.presets.map((p, i) =>
              p.id === undefined && action.persisted![i]?.id !== undefined
                ? { ...p, id: action.persisted![i]!.id }
                : p,
            )
          : state.presets;
      return { ...state, presets, baseline: presets.map(project) };
    }
  }
}
