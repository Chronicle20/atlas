/**
 * Canonical mapping for MapleStory mob-skill ids. String.wz does not ship names
 * for mob skills in the GMS v83 dataset, so the atlas-data API returns empty
 * strings. These ids are stable across versions, so a local lookup keeps the
 * chip labels meaningful without a new backend source.
 *
 * Only well-established ids are listed — an unknown id renders as "#<id>" so
 * nothing is ever mislabelled.
 */
const MOB_SKILL_NAMES: Record<number, string> = {
  100: "Weapon Attack Up",
  101: "Magic Attack Up",
  102: "Weapon Defense Up",
  103: "Magic Defense Up",
  110: "Heal",
  111: "Super Knockback",
  112: "Speed Up",
  113: "Seal",
  114: "Darkness",
  115: "Weakness",
  116: "Stun",
  117: "Curse",
  118: "Poison",
  119: "Slow",
  120: "Seduce",
  121: "Dispel",
  122: "Banish",
  123: "Reverse Controls",
  124: "Stop",
  125: "Weapon Reflect",
  126: "Magic Reflect",
  127: "Physical Immunity",
  128: "Magic Immunity",
  140: "Summon",
  141: "Summon & Attack",
  143: "Bind",
  200: "Summon Reinforcements",
};

export function getMobSkillCanonicalName(skillId: number): string | undefined {
  return MOB_SKILL_NAMES[skillId];
}
