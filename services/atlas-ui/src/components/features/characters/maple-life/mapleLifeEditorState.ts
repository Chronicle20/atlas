import type {
  EquipmentEntry,
  InventoryEntry,
  MapleLifeClassEntry,
  MapleLifeConfig,
  MapleLifeLookOptions,
  MapleLifeStatBlock,
} from "@/types/models/template";

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

export interface MapleLifeClassDraft extends Omit<
  MapleLifeClassEntry,
  "sp" | "spSkillId"
> {
  /** Ten book values parsed from `sp`. Empty when `sp` is unparseable. */
  spBooks: number[];
  /** The `sp` string exactly as loaded, emitted verbatim when spBooks is empty. */
  spRaw: string;
  spSkillId?: number | undefined;
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

export const DEFAULT_PICKS: PreviewPicks = {
  faceIdx: 0,
  hairIdx: 0,
  hairColorIdx: 0,
  skinIdx: 0,
};

export function draftIndex(ordinal: number, gender: number): number {
  return ordinal * GENDER_COUNT + gender;
}

function neutralStats(): MapleLifeStatBlock {
  return { str: 0, dex: 0, int: 0, luk: 0, hp: 0, mp: 0 };
}

function neutralDraft(ordinal: number, gender: number): MapleLifeClassDraft {
  return {
    ordinal,
    gender,
    jobId: 0,
    level: 1,
    mapId: 0,
    stats: neutralStats(),
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

function neutralLook(gender: number): MapleLifeLookDraft {
  return {
    gender,
    faces: [],
    hairs: [],
    hairColors: [],
    skinColors: [],
    present: false,
  };
}

function cloneEquipment(entries: EquipmentEntry[]): EquipmentEntry[] {
  return entries.map((e) => ({ ...e }));
}

function cloneInventory(entries: InventoryEntry[]): InventoryEntry[] {
  return entries.map((e) => ({ ...e }));
}

/** `[]` unless `sp` splits on `,` into exactly ten finite-integer parts. */
export function parseSpPool(sp: string): number[] {
  const parts = sp.split(",");
  if (parts.length !== 10) return [];
  const nums: number[] = [];
  for (const part of parts) {
    const n = Number.parseInt(part, 10);
    if (!Number.isFinite(n)) return [];
    nums.push(n);
  }
  return nums;
}

function buildDrafts(
  config: MapleLifeConfig | undefined,
): MapleLifeClassDraft[] {
  const drafts: MapleLifeClassDraft[] = [];
  for (let ordinal = 0; ordinal < ORDINAL_COUNT; ordinal++) {
    for (let gender = 0; gender < GENDER_COUNT; gender++) {
      const found = config?.classes.find(
        (c) => c.ordinal === ordinal && c.gender === gender,
      );
      drafts.push(
        found
          ? {
              ordinal: found.ordinal,
              gender: found.gender,
              jobId: found.jobId,
              level: found.level,
              mapId: found.mapId,
              stats: { ...found.stats },
              ap: found.ap,
              spBooks: parseSpPool(found.sp),
              spRaw: found.sp,
              spSkillId: found.spSkillId,
              meso: found.meso,
              equipment: cloneEquipment(found.equipment),
              inventory: cloneInventory(found.inventory),
              present: true,
            }
          : neutralDraft(ordinal, gender),
      );
    }
  }
  return drafts;
}

function buildLooks(config: MapleLifeConfig | undefined): MapleLifeLookDraft[] {
  const looks: MapleLifeLookDraft[] = [];
  for (let gender = 0; gender < GENDER_COUNT; gender++) {
    const found = config?.looks.find((l) => l.gender === gender);
    looks.push(
      found
        ? {
            gender: found.gender,
            faces: [...found.faces],
            hairs: [...found.hairs],
            hairColors: [...found.hairColors],
            skinColors: [...found.skinColors],
            present: true,
          }
        : neutralLook(gender),
    );
  }
  return looks;
}

export function initialMapleLifeState(): MapleLifeEditorState {
  return {
    drafts: buildDrafts(undefined),
    looks: buildLooks(undefined),
    baseline: { looks: [], classes: [] },
    ordinal: 0,
    gender: 0,
    picks: {},
    loaded: false,
  };
}

export function selectedDraft(
  state: MapleLifeEditorState,
): MapleLifeClassDraft {
  const draft = state.drafts[draftIndex(state.ordinal, state.gender)];
  if (draft === undefined) {
    throw new Error("maple life editor state corrupted: missing draft slot");
  }
  return draft;
}

export function selectedLook(state: MapleLifeEditorState): MapleLifeLookDraft {
  const look = state.looks[state.gender];
  if (look === undefined) {
    throw new Error("maple life editor state corrupted: missing look slot");
  }
  return look;
}

export function picksFor(
  state: MapleLifeEditorState,
  gender: number,
): PreviewPicks {
  return state.picks[String(gender)] ?? DEFAULT_PICKS;
}

function updateSelected(
  state: MapleLifeEditorState,
  fn: (draft: MapleLifeClassDraft) => MapleLifeClassDraft,
): MapleLifeEditorState {
  const idx = draftIndex(state.ordinal, state.gender);
  const drafts = state.drafts.map((d, i) =>
    i === idx ? { ...fn({ ...d }), present: true } : d,
  );
  return { ...state, drafts };
}

function updateSelectedLook(
  state: MapleLifeEditorState,
  fn: (look: MapleLifeLookDraft) => MapleLifeLookDraft,
): MapleLifeEditorState {
  const looks = state.looks.map((l, i) =>
    i === state.gender ? { ...fn({ ...l }), present: true } : l,
  );
  return { ...state, looks };
}

function projectClass(draft: MapleLifeClassDraft): MapleLifeClassEntry {
  const sp =
    draft.spBooks.length === 10 ? draft.spBooks.join(",") : draft.spRaw;
  if (draft.spSkillId !== undefined) {
    return {
      ordinal: draft.ordinal,
      gender: draft.gender,
      jobId: draft.jobId,
      level: draft.level,
      mapId: draft.mapId,
      stats: { ...draft.stats },
      ap: draft.ap,
      sp,
      spSkillId: draft.spSkillId,
      meso: draft.meso,
      equipment: cloneEquipment(draft.equipment),
      inventory: cloneInventory(draft.inventory),
    };
  }
  return {
    ordinal: draft.ordinal,
    gender: draft.gender,
    jobId: draft.jobId,
    level: draft.level,
    mapId: draft.mapId,
    stats: { ...draft.stats },
    ap: draft.ap,
    sp,
    meso: draft.meso,
    equipment: cloneEquipment(draft.equipment),
    inventory: cloneInventory(draft.inventory),
  };
}

function projectLook(look: MapleLifeLookDraft): MapleLifeLookOptions {
  return {
    gender: look.gender,
    faces: [...look.faces],
    hairs: [...look.hairs],
    hairColors: [...look.hairColors],
    skinColors: [...look.skinColors],
  };
}

export function projectForSave(state: MapleLifeEditorState): MapleLifeConfig {
  return {
    looks: state.looks.filter((l) => l.present).map(projectLook),
    classes: state.drafts.filter((d) => d.present).map(projectClass),
  };
}

export function isDirty(state: MapleLifeEditorState): boolean {
  return (
    JSON.stringify(projectForSave(state)) !== JSON.stringify(state.baseline)
  );
}

export function isEmptyConfig(config: MapleLifeConfig | undefined): boolean {
  return config === undefined || config.classes.length === 0;
}

export function mapleLifeReducer(
  state: MapleLifeEditorState,
  action: MapleLifeAction,
): MapleLifeEditorState {
  switch (action.type) {
    case "load": {
      const next: MapleLifeEditorState = {
        drafts: buildDrafts(action.config),
        looks: buildLooks(action.config),
        baseline: { looks: [], classes: [] },
        ordinal: 0,
        gender: 0,
        picks: {},
        loaded: true,
      };
      return { ...next, baseline: projectForSave(next) };
    }
    case "select":
      return { ...state, ordinal: action.ordinal, gender: action.gender };
    case "setIdentity":
      return updateSelected(state, (d) => ({
        ...d,
        [action.field]: action.value,
      }));
    case "setScalar":
      return updateSelected(state, (d) => ({
        ...d,
        [action.field]: action.value,
      }));
    case "setStat":
      return updateSelected(state, (d) => ({
        ...d,
        stats: { ...d.stats, [action.stat]: action.value },
      }));
    case "setSpBook":
      return updateSelected(state, (d) => {
        const spBooks = [...d.spBooks];
        spBooks[action.index] = action.value;
        return { ...d, spBooks };
      });
    case "setSpSkillId":
      return updateSelected(state, (d) => ({
        ...d,
        spSkillId: action.value,
      }));
    case "addLookEntry":
      return updateSelectedLook(state, (l) => ({
        ...l,
        [action.dimension]: [...l[action.dimension], action.id],
      }));
    case "removeLookEntry":
      return updateSelectedLook(state, (l) => ({
        ...l,
        [action.dimension]: l[action.dimension].filter(
          (_, i) => i !== action.entryIndex,
        ),
      }));
    case "addEquipment":
      return updateSelected(state, (d) => ({
        ...d,
        equipment: [
          ...d.equipment,
          { templateId: action.templateId, useAverageStats: true },
        ],
      }));
    case "removeEquipment":
      return updateSelected(state, (d) => ({
        ...d,
        equipment: d.equipment.filter((_, i) => i !== action.entryIndex),
      }));
    case "setEquipmentAvg":
      return updateSelected(state, (d) => ({
        ...d,
        equipment: d.equipment.map((e, i) =>
          i === action.entryIndex ? { ...e, useAverageStats: action.value } : e,
        ),
      }));
    case "addInventory":
      return updateSelected(state, (d) => ({
        ...d,
        inventory: [
          ...d.inventory,
          { templateId: action.templateId, quantity: 1 },
        ],
      }));
    case "removeInventory":
      return updateSelected(state, (d) => ({
        ...d,
        inventory: d.inventory.filter((_, i) => i !== action.entryIndex),
      }));
    case "setInventoryQty":
      return updateSelected(state, (d) => ({
        ...d,
        inventory: d.inventory.map((e, i) =>
          i === action.entryIndex ? { ...e, quantity: action.value } : e,
        ),
      }));
    case "setPreviewPick": {
      const key = String(state.gender);
      const current = state.picks[key] ?? DEFAULT_PICKS;
      return {
        ...state,
        picks: {
          ...state.picks,
          [key]: { ...current, [action.pick]: action.value },
        },
      };
    }
    case "materialiseAll":
      return {
        ...state,
        drafts: state.drafts.map((d) => ({ ...d, present: true })),
        looks: state.looks.map((l) => ({ ...l, present: true })),
      };
    case "seedFromTemplate": {
      const cloned = structuredClone(action.config);
      return {
        ...state,
        drafts: buildDrafts(cloned),
        looks: buildLooks(cloned),
      };
    }
    case "discard":
      return {
        ...state,
        drafts: buildDrafts(state.baseline),
        looks: buildLooks(state.baseline),
      };
    case "savedOk":
      return { ...state, baseline: projectForSave(state) };
  }
}
