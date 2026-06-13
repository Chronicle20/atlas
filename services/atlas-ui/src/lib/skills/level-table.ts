import type { SkillEffect } from "@/services/api/skills.service";
import { STATUP_LABELS } from "@/lib/utils/skill-effect-format";

export interface LevelColumn {
  key: string;
  label: string;
}
export interface LevelTable {
  columns: LevelColumn[];
  rows: Array<Record<string, string>>;
}

/**
 * Ordered numeric SkillEffect magnitude fields → human labels. Keys are the
 * exact atlas-data effect.RestModel JSON keys (camelCase, casing significant).
 * Structured fields (lt/rb/monsterStatus/cardStats/cureAbnormalStatuses) and
 * booleans (overTime/skill/repeatEffect) are intentionally excluded.
 */
export const FIELD_LABELS: Array<[keyof SkillEffect, string]> = [
  ["weaponAttack", "Weapon Atk"],
  ["magicAttack", "Magic Atk"],
  ["weaponDefense", "Weapon Def"],
  ["magicDefense", "Magic Def"],
  ["accuracy", "Accuracy"],
  ["avoidability", "Avoid"],
  ["speed", "Speed"],
  ["jump", "Jump"],
  ["hp", "HP"],
  ["mp", "MP"],
  ["hpR", "HP Recovery %"],
  ["mpR", "MP Recovery %"],
  ["mhpr", "HP Recovery"],
  ["mmpr", "MP Recovery"],
  ["MHPRRate", "Max HP Recovery %"],
  ["MMPRRate", "Max MP Recovery %"],
  ["HPConsume", "HP Cost"],
  ["MPConsume", "MP Cost"],
  ["duration", "Duration (ms)"],
  ["cooldown", "Cooldown (ms)"],
  ["damage", "Damage %"],
  ["attackCount", "Attack Count"],
  ["mobCount", "Mob Count"],
  ["prop", "Chance %"],
  ["x", "X"],
  ["y", "Y"],
  ["fixDamage", "Fixed Damage"],
  ["bulletCount", "Bullets"],
  ["bulletConsume", "Bullet Cost"],
  ["morphId", "Morph ID"],
  ["moneyConsume", "Meso Cost"],
  ["itemConsume", "Item ID"],
  ["itemConsumeAmount", "Item Qty"],
];

function isPresent(v: number | undefined): boolean {
  return typeof v === "number" && v !== 0;
}

/**
 * Builds a per-level table: one row per effect (level i+1), one "Level" column
 * plus one column per scalar field with a non-zero value at any level, plus one
 * column per distinct statup type. All-zero/absent columns are omitted.
 */
export function buildLevelTable(effects: SkillEffect[]): LevelTable {
  const columns: LevelColumn[] = [{ key: "level", label: "Level" }];

  // Scalar columns: keep a field iff some level has a non-zero value.
  // FIELD_LABELS only lists numeric fields; cast through unknown to silence the
  // union with boolean/SkillEffectStatup[] that keyof SkillEffect admits.
  for (const [field, label] of FIELD_LABELS) {
    if (effects.some((e) => isPresent(e[field] as number | undefined))) {
      columns.push({ key: field as string, label });
    }
  }

  // Statup columns: union of distinct types across all levels, in first-seen order.
  const statupTypes: string[] = [];
  for (const e of effects) {
    for (const s of e.statups ?? []) {
      if (!statupTypes.includes(s.type)) statupTypes.push(s.type);
    }
  }
  for (const type of statupTypes) {
    columns.push({ key: `statup:${type}`, label: STATUP_LABELS[type] ?? type });
  }

  const rows = effects.map((e, i) => {
    const row: Record<string, string> = { level: String(i + 1) };
    for (const [field] of FIELD_LABELS) {
      if (columns.some((c) => c.key === field)) {
        const v = e[field] as number | undefined;
        row[field as string] = isPresent(v) ? String(v) : "";
      }
    }
    for (const type of statupTypes) {
      const found = (e.statups ?? []).find((s) => s.type === type);
      row[`statup:${type}`] = found !== undefined ? String(found.amount) : "";
    }
    return row;
  });

  return { columns, rows };
}
