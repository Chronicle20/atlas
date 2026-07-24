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
