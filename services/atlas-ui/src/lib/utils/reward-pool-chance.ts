// Mirrors services/atlas-reward-pools reward/processor.go exactly:
// selectTier (weighted over the three tier weights) then selectItem
// (weight-proportional when the merged pool's Σweight > 0, else uniform).
// Global items always enter the merged pool with weight 0.

export interface ChanceRow {
  key: string;
  chance: number;
  /** true when weighted rows exist in the tier and this zero-weight row can never win */
  excluded: boolean;
}

export function incubatorChances(items: { id: string; weight: number }[]): Map<string, number> {
  const total = items.reduce((s, i) => s + i.weight, 0);
  return new Map(items.map((i) => [i.id, total > 0 ? i.weight / total : 0]));
}

export function gachaponChances(
  tierWeights: { common: number; uncommon: number; rare: number },
  rows: { key: string; tier: "common" | "uncommon" | "rare"; weight: number }[],
): Map<string, ChanceRow> {
  const tierTotal = tierWeights.common + tierWeights.uncommon + tierWeights.rare;
  const result = new Map<string, ChanceRow>();
  for (const tier of ["common", "uncommon", "rare"] as const) {
    const tierRows = rows.filter((r) => r.tier === tier);
    if (tierRows.length === 0) continue;
    const tierChance = tierTotal > 0 ? tierWeights[tier] / tierTotal : 0;
    const weightSum = tierRows.reduce((s, r) => s + r.weight, 0);
    for (const r of tierRows) {
      const within = weightSum > 0 ? r.weight / weightSum : 1 / tierRows.length;
      result.set(r.key, {
        key: r.key,
        chance: tierChance * within,
        excluded: weightSum > 0 && r.weight === 0,
      });
    }
  }
  return result;
}

export function tierHasMixedWeights(rows: { tier: string; weight: number }[], tier: string): boolean {
  const tierRows = rows.filter((r) => r.tier === tier);
  return tierRows.some((r) => r.weight > 0) && tierRows.some((r) => r.weight === 0);
}
