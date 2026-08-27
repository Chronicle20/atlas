import type { CharacterLoadout } from "@/services/api/characterRender.service";
import {
  RENDER_DEFAULT_FACE,
  RENDER_DEFAULT_HAIR,
  RENDER_DEFAULT_SKIN,
} from "../templates/previewLoadout";
import type {
  LookDimension,
  MapleLifeClassDraft,
  MapleLifeLookDraft,
  PreviewPicks,
} from "./mapleLifeEditorState";

/** Canonical render slots, in declaration order, for the first four equips. */
const RENDER_SLOTS = ["-5", "-6", "-7", "-11"] as const;

function at(pool: number[], idx: number): number | undefined {
  if (pool.length === 0) return undefined;
  return pool[Math.min(Math.max(idx, 0), pool.length - 1)];
}

/** The client's own expression: anHairEquip[0] = hairColor + 10 * (hairStyle / 10). */
export function composeHair(hairStyle: number, hairColor: number): number {
  return hairColor + 10 * Math.floor(hairStyle / 10);
}

function buildEquipment(draft: MapleLifeClassDraft): Record<string, number> {
  const equipment: Record<string, number> = {};
  const slice = draft.equipment.slice(0, RENDER_SLOTS.length);
  slice.forEach((entry, i) => {
    const slot = RENDER_SLOTS[i];
    if (slot === undefined) return;
    equipment[slot] = entry.templateId;
  });
  return equipment;
}

export function buildMapleLifeLoadout(
  draft: MapleLifeClassDraft,
  look: MapleLifeLookDraft,
  picks: PreviewPicks,
): CharacterLoadout {
  const hairStyle = at(look.hairs, picks.hairIdx) ?? RENDER_DEFAULT_HAIR;
  const hairColor = at(look.hairColors, picks.hairColorIdx) ?? 0;
  return {
    skin: at(look.skinColors, picks.skinIdx) ?? RENDER_DEFAULT_SKIN,
    hair: composeHair(hairStyle, hairColor),
    face: at(look.faces, picks.faceIdx) ?? RENDER_DEFAULT_FACE,
    equipment: buildEquipment(draft),
    gender: look.gender,
  };
}

export function buildMapleLifeVariantLoadout(
  draft: MapleLifeClassDraft,
  look: MapleLifeLookDraft,
  picks: PreviewPicks,
  dimension: LookDimension,
  candidateId: number,
): CharacterLoadout {
  const base = buildMapleLifeLoadout(draft, look, picks);
  switch (dimension) {
    case "faces":
      return { ...base, face: candidateId };
    case "hairs":
      return {
        ...base,
        hair: composeHair(
          candidateId,
          at(look.hairColors, picks.hairColorIdx) ?? 0,
        ),
      };
    case "hairColors":
      return {
        ...base,
        hair: composeHair(
          at(look.hairs, picks.hairIdx) ?? RENDER_DEFAULT_HAIR,
          candidateId,
        ),
      };
    case "skinColors":
      return { ...base, skin: candidateId };
  }
}

export function combinationCount(look: MapleLifeLookDraft): number {
  return (
    look.faces.length *
    look.hairs.length *
    look.hairColors.length *
    look.skinColors.length
  );
}
