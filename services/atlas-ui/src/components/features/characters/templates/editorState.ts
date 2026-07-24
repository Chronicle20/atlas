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
        [action.pool]: t[action.pool].filter((_, i) => i !== action.entryIndex),
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
