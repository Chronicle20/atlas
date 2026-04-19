export interface MapBounds {
  x: number;
  y: number;
  width: number;
  height: number;
}

export function worldToOverlayPercent(
  worldX: number,
  worldY: number,
  bounds: MapBounds,
): { left: string; top: string } {
  const left = ((worldX - bounds.x) / bounds.width) * 100;
  const top = ((worldY - bounds.y) / bounds.height) * 100;
  return { left: `${left}%`, top: `${top}%` };
}
