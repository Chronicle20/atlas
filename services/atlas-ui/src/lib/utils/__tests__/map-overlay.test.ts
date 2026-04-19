import { worldToOverlayPercent, type MapBounds } from '../map-overlay';

describe('worldToOverlayPercent', () => {
  it('maps positive-origin bounds to percentages', () => {
    const bounds: MapBounds = { x: 0, y: 0, width: 1000, height: 500 };
    expect(worldToOverlayPercent(500, 250, bounds)).toEqual({ left: '50%', top: '50%' });
  });

  it('maps negative-origin bounds (miniMap case)', () => {
    const bounds: MapBounds = { x: -400, y: -300, width: 800, height: 600 };
    expect(worldToOverlayPercent(0, 0, bounds)).toEqual({ left: '50%', top: '50%' });
  });

  it('places entity at exact origin at 0%,0%', () => {
    const bounds: MapBounds = { x: -100, y: -200, width: 400, height: 400 };
    expect(worldToOverlayPercent(-100, -200, bounds)).toEqual({ left: '0%', top: '0%' });
  });

  it('places entity at far corner at 100%,100%', () => {
    const bounds: MapBounds = { x: 10, y: 20, width: 90, height: 80 };
    expect(worldToOverlayPercent(100, 100, bounds)).toEqual({ left: '100%', top: '100%' });
  });

  it('computes percentages outside [0,100] for out-of-bounds coords without throwing', () => {
    const bounds: MapBounds = { x: 0, y: 0, width: 100, height: 100 };
    const r = worldToOverlayPercent(200, -50, bounds);
    expect(r.left).toBe('200%');
    expect(r.top).toBe('-50%');
  });
});
