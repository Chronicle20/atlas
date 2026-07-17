import type { CanonicalSelection } from '@/lib/headers';

export function selectionKey(sel: CanonicalSelection): string {
  return `${sel.region}/${sel.majorVersion}.${sel.minorVersion}`;
}

interface HasRegionVersion {
  attributes: { region: string; majorVersion: number; minorVersion: number };
}

/**
 * Deduplicated union of (region, major, minor) combos from templates and
 * tenants, sorted by (region, major, minor). Provenance is irrelevant —
 * these are just seeds for the picker.
 */
export function dedupeSelections(
  templates: HasRegionVersion[],
  tenants: HasRegionVersion[],
): CanonicalSelection[] {
  const map = new Map<string, CanonicalSelection>();
  for (const item of [...templates, ...tenants]) {
    const sel: CanonicalSelection = {
      region: item.attributes.region,
      majorVersion: item.attributes.majorVersion,
      minorVersion: item.attributes.minorVersion,
    };
    map.set(selectionKey(sel), sel);
  }
  return [...map.values()].sort((a, b) => {
    if (a.region !== b.region) return a.region.localeCompare(b.region);
    if (a.majorVersion !== b.majorVersion) return a.majorVersion - b.majorVersion;
    return a.minorVersion - b.minorVersion;
  });
}

/**
 * Validates a custom entry: non-empty region, non-negative integer versions.
 * Returns null while invalid so workflow rows stay disabled.
 */
export function parseCustomSelection(
  region: string,
  major: string,
  minor: string,
): CanonicalSelection | null {
  const trimmed = region.trim();
  if (!trimmed) return null;
  if (!/^\d+$/.test(major) || !/^\d+$/.test(minor)) return null;
  return { region: trimmed, majorVersion: Number(major), minorVersion: Number(minor) };
}
