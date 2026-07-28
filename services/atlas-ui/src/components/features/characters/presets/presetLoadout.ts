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
  "faces" | "hairs" | "hairColors" | "skinColors";

/** Placeable worn template ids in slot-assignment order (unplaceable dropped). */
export function wornTemplateIds(attrs: CharacterPresetAttributes): number[] {
  return attrs.equipment
    .map((e) => e.templateId)
    .filter((id) => getDefaultSlotForTemplateId(id) !== null);
}

/** Worn equipment as a slot→templateId map, later same-slot items overwriting earlier. */
function placedEquipment(
  attrs: CharacterPresetAttributes,
): Record<string, number> {
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

/**
 * Loadout for an appearance-dimension thumbnail: bare mannequin (no worn
 * equipment — gear covers exactly the features being compared) with one
 * dimension substituted. Hair candidates arrive as render-ready ids (base
 * variant, or base+digit for color tiles) — no color arithmetic here.
 */
export function buildPresetVariantLoadout(
  attrs: CharacterPresetAttributes,
  dimension: PresetAppearanceDimension,
  candidateId: number,
): CharacterLoadout {
  const base = { ...buildPresetLoadout(attrs), equipment: {} };
  switch (dimension) {
    case "faces":
      return { ...base, face: candidateId };
    case "hairs":
      return { ...base, hair: candidateId };
    case "hairColors":
      return {
        ...base,
        hair: (attrs.hair || RENDER_DEFAULT_HAIR) + candidateId,
      };
    case "skinColors":
      return { ...base, skin: candidateId };
  }
}
