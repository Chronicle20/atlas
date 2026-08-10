// Search-filter strategy per pool (design D4). parseFilters in atlas-data
// item/filter.go accepts exactly ONE filter[subcategory] token, so pools that
// span multiple subcategories filter client-side on the returned rows'
// subcategory field instead.

export type SearchPoolKey =
  "tops" | "bottoms" | "shoes" | "weapons" | "items" | "cashSurpriseBoxes";

// The 16 weapon tokens registered in atlas-data item/filter.go:55-60
// (pet-equip deliberately excluded — not a starting-weapon candidate).
export const WEAPON_SUBCATEGORIES: ReadonlySet<string> = new Set([
  "one-handed-sword",
  "one-handed-axe",
  "one-handed-mace",
  "dagger",
  "wand",
  "staff",
  "two-handed-sword",
  "two-handed-axe",
  "two-handed-mace",
  "spear",
  "polearm",
  "bow",
  "crossbow",
  "claw",
  "knuckle",
  "gun",
]);

export interface PoolSearchConfig {
  compartment?: "equipment" | "cash";
  /** Server-side single-token filter[subcategory]. */
  subcategory?: string;
  /** Client-side post-filter over result rows' subcategory. */
  clientSubcategories?: ReadonlySet<string>;
}

export const POOL_SEARCH_CONFIGS: Record<SearchPoolKey, PoolSearchConfig> = {
  // Aran's 1042167 is an overall — the tops pool legitimately contains overalls.
  tops: {
    compartment: "equipment",
    clientSubcategories: new Set(["top", "overall"]),
  },
  bottoms: { compartment: "equipment", subcategory: "bottom" },
  shoes: { compartment: "equipment", subcategory: "shoes" },
  weapons: {
    compartment: "equipment",
    clientSubcategories: WEAPON_SUBCATEGORIES,
  },
  // Starting-kit items search all compartments.
  items: {},
  // Cash Shop Surprise boxes. 522xxxx is ClassificationGachaponCoupon
  // (libs/atlas-constants/item/constants.go:90), which atlas-data maps to the
  // `gachapon-coupon` subcategory token (item/classify.go). That token is the
  // narrowest server-side filter that still contains the stock box 5222000 —
  // useItemSearch's "Use id N" escape hatch covers a tenant that registered a
  // box outside the range in cashShop.surprise.boxTemplateIds.
  cashSurpriseBoxes: { compartment: "cash", subcategory: "gachapon-coupon" },
};
