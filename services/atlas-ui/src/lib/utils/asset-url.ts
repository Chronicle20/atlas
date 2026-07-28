/**
 * Constructs a deterministic icon URL for an entity from atlas-assets.
 * Icon URLs are static file paths - no API call needed.
 */
export function getAssetIconUrl(
  tenantId: string,
  region: string,
  majorVersion: number,
  minorVersion: number,
  category: "npc" | "mob" | "item" | "skill" | "reactor" | "map",
  entityId: number,
): string {
  const baseUrl = import.meta.env.VITE_ASSET_BASE_URL || "/api/assets";
  const version = `${majorVersion}.${minorVersion}`;
  return `${baseUrl}/${tenantId}/${region}/${version}/${category}/${entityId}/icon.png`;
}

/**
 * Constructs a URL for a world icon extracted from UI.wz/Login.img/ViewAllChar/WorldIcons/{worldId}.
 * The extractor writes these as 20×20 PNGs under the `world-icon` category.
 */
export function getWorldIconUrl(
  tenantId: string,
  region: string,
  majorVersion: number,
  minorVersion: number,
  worldId: number,
): string {
  const baseUrl = import.meta.env.VITE_ASSET_BASE_URL || "/api/assets";
  const version = `${majorVersion}.${minorVersion}`;
  return `${baseUrl}/${tenantId}/${region}/${version}/world-icon/${worldId}/icon.png`;
}

/**
 * Constructs a URL for the sealed-item padlock overlay extracted from
 * UI.wz/UIWindow.img/ItemProtector/Icon. The UI worker writes it under the
 * `ui/item-protector` category; one icon serves every lock duration.
 */
export function getItemProtectorIconUrl(
  tenantId: string,
  region: string,
  majorVersion: number,
  minorVersion: number,
): string {
  const baseUrl = import.meta.env.VITE_ASSET_BASE_URL || "/api/assets";
  const version = `${majorVersion}.${minorVersion}`;
  return `${baseUrl}/${tenantId}/${region}/${version}/ui/item-protector/icon.png`;
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
  kind: "render" | "minimap",
): string {
  const baseUrl = import.meta.env.VITE_ASSET_BASE_URL || "/api/assets";
  const version = `${majorVersion}.${minorVersion}`;
  return `${baseUrl}/${tenantId}/${region}/${version}/map/${mapId}/${kind}.png`;
}
