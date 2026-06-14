import type { SkillDefinition } from "@/services/api/skills.service";

export const SKILL_TYPE = ["Passive", "Active", "Buff"] as const;
export type SkillType = (typeof SKILL_TYPE)[number];

/**
 * Derives a display type from existing skill fields (atlas-data has no explicit
 * type). Order matters: a stat-up / sustained effect is a Buff regardless of
 * action; otherwise an action animation means Active; otherwise Passive.
 * Never throws on missing fields.
 */
export function deriveSkillType(
  def: Pick<SkillDefinition, "action" | "effects">,
): SkillType {
  const effects = def?.effects ?? [];
  const hasStatups = effects.some((e) => (e?.statups?.length ?? 0) > 0);
  const sustained = effects.some((e) => e?.overTime === true);
  if (hasStatups || sustained) return "Buff";
  if (def?.action === true) return "Active";
  return "Passive";
}
