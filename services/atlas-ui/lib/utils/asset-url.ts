/**
 * Constructs a deterministic icon URL for an entity from atlas-assets.
 * Icon URLs are static file paths - no API call needed.
 */
export function getAssetIconUrl(
  tenantId: string,
  region: string,
  majorVersion: number,
  minorVersion: number,
  category: 'npc' | 'mob' | 'item' | 'skill' | 'reactor',
  entityId: number,
): string {
  const baseUrl = process.env.NEXT_PUBLIC_ASSET_BASE_URL || '/api/assets';
  const version = `${majorVersion}.${minorVersion}`;
  return `${baseUrl}/${tenantId}/${region}/${version}/${category}/${entityId}/icon.png`;
}
