import type { DefinitionState } from "@/lib/socket/model";

/**
 * The single source of the grid's state colours, shared by `PacketGridCell`,
 * `GridLegend` and the drawer's Fields cards. A legend that drifts from the
 * cells it explains is worse than no legend, so the three render from one
 * table rather than three copies of the same Tailwind strings.
 */
export const STATE_CELL_CLASS: Record<DefinitionState, string> = {
  defined: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
  unsupported: "bg-muted/60 text-muted-foreground",
  // Undefined is the absence of ink, matching the prototype: no fill, just
  // the "·" placeholder the cell renders.
  undefined: "",
};

/** The legend's swatch for each state - filled for the two that tint a cell. */
export const STATE_SWATCH_CLASS: Record<DefinitionState, string> = {
  defined: "bg-emerald-500/20 border border-emerald-600/50",
  unsupported: "bg-muted border-border border",
  undefined: "border-muted-foreground/50 border border-dashed",
};

export const STATE_LABEL: Record<DefinitionState, string> = {
  defined: "Defined",
  unsupported: "Unsupported (audited)",
  undefined: "Undefined",
};

/**
 * The bare state word, for places that state it in a slot too narrow for
 * STATE_LABEL's parenthetical qualifier - the drawer's Fields cards, where it
 * occupies the same line a defined card gives its validator.
 */
export const SHORT_STATE_LABEL: Record<DefinitionState, string> = {
  defined: "Defined",
  unsupported: "Unsupported",
  undefined: "Undefined",
};
