/**
 * Canonical mapping for MapleStory mob-skill ids. String.wz does not ship names
 * for mob skills in the GMS v83 dataset, so the atlas-data API returns empty
 * strings. These ids are stable across versions, so a local lookup keeps the
 * chip labels meaningful without a new backend source.
 *
 * This table mirrors `libs/atlas-constants/monster/skill.go` (the canonical
 * Go source) line-for-line. Keep them in sync — the Go file is the source of
 * truth. An unknown id renders as "#<id>" so nothing is ever mislabelled.
 */
const MOB_SKILL_NAMES: Record<number, string> = {
  // Single-target buffs (apply to caster only)
  100: "Weapon Attack Up",
  101: "Magic Attack Up",
  102: "Weapon Defense Up",
  103: "Magic Defense Up",

  // AoE buffs (apply to nearby allied mobs)
  110: "Weapon Attack Up (AoE)",
  111: "Magic Attack Up (AoE)",
  112: "Weapon Defense Up (AoE)",
  113: "Magic Defense Up (AoE)",
  114: "Heal",
  115: "Speed Up",

  // Player-targeting debuffs / diseases
  120: "Seal",
  121: "Darkness",
  122: "Weakness",
  123: "Stun",
  124: "Curse",
  125: "Poison",
  126: "Slow",
  127: "Dispel",
  128: "Seduce",
  129: "Banish",
  131: "Area Poison",
  132: "Reverse Input",
  133: "Undead",
  134: "Stop Potion",
  135: "Stop Motion",
  136: "Fear",

  // Player-attack immunities / counters
  140: "Physical Immune",
  141: "Magic Immune",
  142: "Hard Skin",
  143: "Physical Counter",
  144: "Magic Counter",
  145: "Physical/Magic Counter",

  // Monster Carnival buffs
  150: "Carnival Attack",
  151: "Carnival Magic Attack",
  152: "Carnival Defense",
  153: "Carnival Magic Defense",
  154: "Carnival Accuracy",
  155: "Carnival Avoidability",
  156: "Carnival Speed",
  157: "Carnival Skill Seal",

  // Summon
  200: "Summon",
};

export function getMobSkillCanonicalName(skillId: number): string | undefined {
  return MOB_SKILL_NAMES[skillId];
}
