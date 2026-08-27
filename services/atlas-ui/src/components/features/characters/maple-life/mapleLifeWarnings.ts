import type { MapleLifeEditorState } from "./mapleLifeEditorState";

export const KNOWN_SP_SKILL_IDS = [1000001, 2000001] as const;
export const SP_SKILL_LABELS: Record<number, string> = {
  1000001: "Improved Max HP Increase (Warrior)",
  2000001: "Improved Max MP Increase (Magician)",
};

export const WARN = {
  unknownSpSkill:
    "This skill id has no coded prerequisite in factory/maple_life.go prerequisiteFor, so no prerequisite will be granted for it. The value is preserved as loaded.",
  absentRow:
    "This (ordinal, gender) row is not in the configuration. A player selecting it is rejected with ErrClassOrdinalUnknown. Edit any field to create it.",
} as const;

export interface MapleLifeWarning {
  /** Dotted path, same key space as the schema's IssueMap. */
  path: string;
  message: string;
}

export function mapleLifeWarnings(
  state: MapleLifeEditorState,
): MapleLifeWarning[] {
  const warnings: MapleLifeWarning[] = [];
  for (const draft of state.drafts) {
    const rowPath = `classes.${draft.ordinal}.${draft.gender}`;
    if (!draft.present) {
      warnings.push({ path: rowPath, message: WARN.absentRow });
      continue;
    }
    if (
      draft.spSkillId !== undefined &&
      draft.spSkillId !== 0 &&
      !(KNOWN_SP_SKILL_IDS as readonly number[]).includes(draft.spSkillId)
    ) {
      warnings.push({
        path: `${rowPath}.spSkillId`,
        message: WARN.unknownSpSkill,
      });
    }
  }
  return warnings;
}

export function warningMap(state: MapleLifeEditorState): Map<string, string[]> {
  const map = new Map<string, string[]>();
  for (const warning of mapleLifeWarnings(state)) {
    const existing = map.get(warning.path);
    if (existing) {
      existing.push(warning.message);
    } else {
      map.set(warning.path, [warning.message]);
    }
  }
  return map;
}
