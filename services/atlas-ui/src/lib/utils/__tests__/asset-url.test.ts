import { getAssetIconUrl, getMapImageUrl } from '../asset-url';

describe('getAssetIconUrl', () => {
  it('builds a tenant-scoped icon URL', () => {
    const url = getAssetIconUrl('tenant-a', 'GMS', 83, 1, 'npc', 1002000);
    expect(url).toBe('/api/assets/tenant-a/GMS/83.1/npc/1002000/icon.png');
  });

  it('supports the reactor category', () => {
    const url = getAssetIconUrl('t', 'R', 1, 0, 'reactor', 99);
    expect(url).toBe('/api/assets/t/R/1.0/reactor/99/icon.png');
  });
});

describe('getMapImageUrl', () => {
  it('builds a render.png URL', () => {
    const url = getMapImageUrl('tenant-a', 'GMS', 83, 1, 100000000, 'render');
    expect(url).toBe('/api/assets/tenant-a/GMS/83.1/map/100000000/render.png');
  });

  it('builds a minimap.png URL', () => {
    const url = getMapImageUrl('tenant-a', 'GMS', 83, 1, 100000000, 'minimap');
    expect(url).toBe('/api/assets/tenant-a/GMS/83.1/map/100000000/minimap.png');
  });

  it('accepts a string map id', () => {
    const url = getMapImageUrl('t', 'R', 1, 0, '100000000', 'minimap');
    expect(url).toBe('/api/assets/t/R/1.0/map/100000000/minimap.png');
  });
});
