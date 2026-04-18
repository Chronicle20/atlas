/**
 * Constructs a deterministic icon URL for an entity from atlas-assets.
 * Icon URLs are static file paths - no API call needed.
 */
export function getAssetIconUrl(
  tenantId: string,
  region: string,
  majorVersion: number,
  minorVersion: number,
  category: 'npc' | 'mob' | 'item' | 'skill' | 'reactor' | 'map',
  entityId: number,
): string {
  const baseUrl = import.meta.env.VITE_ASSET_BASE_URL || '/api/assets';
  const version = `${majorVersion}.${minorVersion}`;
  return `${baseUrl}/${tenantId}/${region}/${version}/${category}/${entityId}/icon.png`;
}

/**
 * Constructs a deterministic map image URL from atlas-assets.
 * `kind` selects between the full-map composite render and the in-WZ minimap.
 */
export function getMapImageUrl(
  tenantId: string,
  region: string,
  majorVersion: number,
  minorVersion: number,
  mapId: number | string,
  kind: 'render' | 'minimap',
): string {
  const baseUrl = import.meta.env.VITE_ASSET_BASE_URL || '/api/assets';
  const version = `${majorVersion}.${minorVersion}`;
  return `${baseUrl}/${tenantId}/${region}/${version}/map/${mapId}/${kind}.png`;
}
