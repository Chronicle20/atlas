// Mirrors services/atlas-reward-pools reward/processor.go exactly:
// selectTier (weighted over the three tier weights) then selectItem
// (weight-proportional when the merged pool's Σweight > 0, else uniform).
// Global items always enter the merged pool with weight 0.

import type { RewardPoolKind } from "@/types/models/reward-pool";

// The server's roll model has exactly two shapes: gachapon uses tiered
// weights (three fixed tiers, per-item weight within a tier); incubator and
// cash-surprise both use a single flat weight list. Keying this as a Record
// over the full RewardPoolKind union (rather than an `isIncubator` boolean or
// a two-way ternary) makes a future third kind a compile error at every call
// site instead of silently falling into the tiered branch, which was exactly
// the bug that would have hit cash-surprise if these call sites were left
// untouched. Lives here (not in PoolItemsTable.tsx) so it's importable from
// non-component modules without tripping react-refresh/only-export-components.
export const POOL_ITEM_TABLE_LAYOUT: Record<RewardPoolKind, "tiered" | "flat"> =
  {
    gachapon: "tiered",
    incubator: "flat",
    "cash-surprise": "flat",
  };

/**
 * Icon source per pool kind, shared by PoolNameCell (grid) and
 * RewardPoolDetailPage (header). Incubator and cash-surprise pools are
 * identified by an item id (= pool id), so they show that item's icon;
 * gachapon pools show the first configured NPC's icon (the machine).
 *
 * Keyed as a Record over the full RewardPoolKind union (not an
 * isIncubator/ternary check) so a future third kind is a compile error at
 * every call site instead of silently falling into the gachapon NPC-icon
 * branch — the same bug class POOL_ITEM_TABLE_LAYOUT above guards against.
 * Lives here (not in PoolNameCell.tsx) so it's importable from
 * RewardPoolDetailPage.tsx without tripping
 * react-refresh/only-export-components on a component file.
 */
export const ICON_SOURCE: Record<RewardPoolKind, "item" | "npc"> = {
  incubator: "item",
  "cash-surprise": "item",
  gachapon: "npc",
};

export interface ChanceRow {
  key: string;
  chance: number;
  /** true when weighted rows exist in the tier and this zero-weight row can never win */
  excluded: boolean;
}

/**
 * Incubator pools: weight / Σweight. When no item declares a weight
 * (Σweight = 0), the server's selectItem falls back to a uniform pick,
 * so a non-empty zero-total pool yields 1/N per item. Empty pool → empty map.
 */
export function incubatorChances(
  items: { id: string; weight: number }[],
): Map<string, number> {
  if (items.length === 0) return new Map();
  const total = items.reduce((s, i) => s + i.weight, 0);
  return new Map(
    items.map((i) => [i.id, total > 0 ? i.weight / total : 1 / items.length]),
  );
}

export function gachaponChances(
  tierWeights: { common: number; uncommon: number; rare: number },
  rows: { key: string; tier: "common" | "uncommon" | "rare"; weight: number }[],
): Map<string, ChanceRow> {
  const tierTotal =
    tierWeights.common + tierWeights.uncommon + tierWeights.rare;
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

export function tierHasMixedWeights(
  rows: { tier: string; weight: number }[],
  tier: string,
): boolean {
  const tierRows = rows.filter((r) => r.tier === tier);
  return (
    tierRows.some((r) => r.weight > 0) && tierRows.some((r) => r.weight === 0)
  );
}
