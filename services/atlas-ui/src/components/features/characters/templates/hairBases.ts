/** A browsable candidate: `value` is what selection stores/compares; `renderId`
 * is what the preview draws (for hairs, the group's lowest existing color
 * variant — black when present). */
export interface BrowserTile {
  value: number;
  renderId: number;
}

/**
 * The hair enumeration lists every color variant as its own id (base + color
 * digit). Browsing wants one entry per hair, so collapse to the base id
 * (floor/10*10) and render the lowest existing digit — color is chosen on the
 * separate hair-color row, not here.
 */
export function collapseHairBases(ids: number[]): BrowserTile[] {
  const minDigitByBase = new Map<number, number>();
  for (const id of ids) {
    const base = Math.floor(id / 10) * 10;
    const digit = id - base;
    const prev = minDigitByBase.get(base);
    if (prev === undefined || digit < prev) minDigitByBase.set(base, digit);
  }
  return [...minDigitByBase.entries()]
    .sort((a, b) => a[0] - b[0])
    .map(([base, digit]) => ({ value: base, renderId: base + digit }));
}
