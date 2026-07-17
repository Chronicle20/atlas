// Interim hypothesis mapping incubator egg item ids (= reward-pool ids) to their
// MapleStory region. 4170008 is intentionally absent (region unconfirmed).
const EGG_REGIONS: Record<number, string> = {
  4170000: "Henesys",
  4170001: "Ellinia",
  4170002: "Perion",
  4170003: "Kerning City",
  4170004: "El Nath",
  4170005: "Ludibrium",
  4170006: "Orbis",
  4170007: "Aqua Road",
  4170009: "Nautilus",
};

/** Region label for an egg item id, or null when unknown (e.g. 4170008 or a non-egg id). */
export function eggRegionLabel(eggId: number | string): string | null {
  const id = typeof eggId === "string" ? parseInt(eggId, 10) : eggId;
  if (Number.isNaN(id)) return null;
  return EGG_REGIONS[id] ?? null;
}

/** Append " (Region)" to an incubator pool's display name when the region is known. */
export function formatIncubatorName(baseName: string, eggId: number | string): string {
  const region = eggRegionLabel(eggId);
  return region ? `${baseName} (${region})` : baseName;
}
