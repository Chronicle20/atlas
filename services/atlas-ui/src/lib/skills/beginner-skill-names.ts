// Curated display names for Beginner skills whose atlas-data definition carries an
// empty `name` (verified live, 2026-06-14: GET /api/data/skills/1000 -> name:"").
// Ids + names sourced from libs/atlas-constants/skill/constants.go (Beginner*Id,
// ids 1000-1012). This is a display hint table, not a data change.
export const BEGINNER_SKILL_NAMES: Record<number, string> = {
  1000: "Three Snails",
  1001: "Recovery",
  1002: "Nimble Feet",
  1003: "Soul of Craftsman",
  1004: "Monster Riding",
  1005: "Echo of Hero",
  1006: "Jump Down",
  1007: "Maker",
  1008: "Multi Pet",
  1009: "Bamboo",
  1010: "Invincible",
  1011: "Berserk",
  1012: "Bless of Nymph",
};

/**
 * Display name for a skill. Uses the server-provided name when non-blank; otherwise
 * the curated map; otherwise `Skill <id>`. Driven by a blank server name (FR-3.3),
 * so any future blank-name skill degrades cleanly — never special-cased to job 0.
 */
export function resolveSkillName(
  id: number,
  serverName: string | undefined,
): string {
  if (serverName != null && serverName.trim() !== "") {
    return serverName;
  }
  return BEGINNER_SKILL_NAMES[id] ?? `Skill ${id}`;
}
