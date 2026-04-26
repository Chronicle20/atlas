export type Compartment =
  | "equipment"
  | "use"
  | "setup"
  | "etc"
  | "cash"
  | "unknown";

export const COMPARTMENT_LABELS: Record<Compartment, string> = {
  equipment: "Equipment",
  use: "Use",
  setup: "Setup",
  etc: "Etc",
  cash: "Cash",
  unknown: "Unknown",
};

// Compartments offered in the filter dropdown (excludes "unknown").
export const COMPARTMENT_OPTIONS: Compartment[] = [
  "equipment",
  "use",
  "setup",
  "etc",
  "cash",
];

// Per-compartment subcategory tokens. Order is the dropdown display order.
// Source of truth: docs/tasks/task-028-item-search-filters/taxonomy.md.
export const COMPARTMENT_TAXONOMY: Record<Exclude<Compartment, "unknown">, string[]> = {
  equipment: [
    "hat", "face-accessory", "eye-accessory", "earring", "top", "overall",
    "bottom", "shoes", "gloves", "shield", "cape", "ring", "pendant",
    "belt", "medal", "tamed-mob", "saddle", "pet-equip",
    "one-handed-sword", "one-handed-axe", "one-handed-mace", "dagger",
    "wand", "staff",
    "two-handed-sword", "two-handed-axe", "two-handed-mace",
    "spear", "polearm",
    "bow", "crossbow", "claw", "knuckle", "gun",
    "other",
  ],
  use: [
    "potion", "town-warp", "scroll", "arrow", "throwing-star", "megaphone",
    "summoning-sack", "pet-food", "transformation", "skill-book",
    "mastery-book", "bullet", "monster-card",
    "other",
  ],
  setup: ["chair", "hired-merchant", "other-setup"],
  etc: [
    "crafting-material", "ore", "production-item", "mineral-ore",
    "mineral-refined", "gem-rough", "gem-cut", "monster-drop",
    "magnifying-glass", "quest-item", "simulator", "book-page",
    "other-etc",
  ],
  cash: [
    "pet", "cosmetic-throwing-star", "hired-merchant",
    "teleport-rock", "point-reset", "item-imprint", "megaphone",
    "message-banner", "note", "song-player", "field-effect",
    "death-protection", "store-permit", "cosmetic-coupon", "expression",
    "pet-imprint", "currency-sack", "experience-coupon", "gachapon-coupon",
    "store-search", "pet-consumable", "wedding-ticket", "character-effect",
    "guild-emote", "transformation-coupon", "duey-coupon", "drop-coupon",
    "chalkboard", "pet-evolution", "avatar-megaphone", "character-imprint",
    "cosmetic-membership-coupon", "character-creation", "remote-merchant",
    "pet-multi-consumable", "remote-store",
    "other-cash",
  ],
};

// Subcategory display labels — humanised for the dropdown. Falls back to the
// token itself if no override is registered.
export function subcategoryLabel(token: string): string {
  return token
    .split("-")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

// Per-class checkbox set — Beginner is intentionally absent (PRD §9 Q2 default).
export const CLASS_OPTIONS = ["warrior", "magician", "bowman", "thief", "pirate"] as const;
export type ClassOption = (typeof CLASS_OPTIONS)[number];

// Serialise the in-state class selection into the wire string the server expects.
// - allClasses === true → "any"
// - any per-class checks → comma-joined alphabetical lowercase
// - none → empty string (caller omits the param)
export function serializeClassFilter(selected: Set<ClassOption>, allClasses: boolean): string {
  if (allClasses) return "any";
  if (selected.size === 0) return "";
  return Array.from(selected).sort().join(",");
}

export function parseClassFilter(raw: string | null): { selected: Set<ClassOption>; allClasses: boolean } {
  if (!raw) return { selected: new Set(), allClasses: false };
  if (raw === "any") return { selected: new Set(), allClasses: true };
  const tokens = raw.split(",").filter((t): t is ClassOption => (CLASS_OPTIONS as readonly string[]).includes(t));
  return { selected: new Set(tokens), allClasses: false };
}
