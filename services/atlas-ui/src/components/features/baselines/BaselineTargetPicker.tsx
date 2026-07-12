import { useMemo, useState } from 'react';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useTemplates } from '@/lib/hooks/api/useTemplates';
import { useTenants } from '@/lib/hooks/api/useTenants';
import type { CanonicalSelection } from '@/lib/headers';

const CUSTOM = '__custom__';

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

interface BaselineTargetPickerProps {
  value: CanonicalSelection | null;
  onChange: (sel: CanonicalSelection | null) => void;
}

export function BaselineTargetPicker({ value, onChange }: BaselineTargetPickerProps) {
  const { data: templates } = useTemplates();
  const { data: tenants } = useTenants();
  const [selectedKey, setSelectedKey] = useState<string>('');
  const [customRegion, setCustomRegion] = useState('');
  const [customMajor, setCustomMajor] = useState('');
  const [customMinor, setCustomMinor] = useState('');

  const options = useMemo(
    () => dedupeSelections(templates ?? [], tenants ?? []),
    [templates, tenants],
  );

  const isCustom = selectedKey === CUSTOM;
  const customInvalid =
    isCustom &&
    (customRegion !== '' || customMajor !== '' || customMinor !== '') &&
    parseCustomSelection(customRegion, customMajor, customMinor) === null;

  const handleSelect = (key: string) => {
    setSelectedKey(key);
    if (key === CUSTOM) {
      onChange(parseCustomSelection(customRegion, customMajor, customMinor));
      return;
    }
    onChange(options.find((o) => selectionKey(o) === key) ?? null);
  };

  const handleCustomChange = (region: string, major: string, minor: string) => {
    setCustomRegion(region);
    setCustomMajor(major);
    setCustomMinor(minor);
    onChange(parseCustomSelection(region, major, minor));
  };

  return (
    <div className="flex flex-col gap-3" data-testid="baseline-target-picker">
      <Select value={selectedKey} onValueChange={handleSelect}>
        <SelectTrigger className="w-64" aria-label="Region and version">
          <SelectValue placeholder="Select region and version…" />
        </SelectTrigger>
        <SelectContent>
          {options.map((o) => (
            <SelectItem key={selectionKey(o)} value={selectionKey(o)}>
              {o.region} {o.majorVersion}.{o.minorVersion}
            </SelectItem>
          ))}
          <SelectItem value={CUSTOM}>Custom…</SelectItem>
        </SelectContent>
      </Select>
      {isCustom && (
        <div className="flex items-end gap-2">
          <div className="flex flex-col gap-1">
            <Label htmlFor="custom-region">Region</Label>
            <Input
              id="custom-region"
              className="w-28"
              value={customRegion}
              onChange={(e) => handleCustomChange(e.target.value, customMajor, customMinor)}
            />
          </div>
          <div className="flex flex-col gap-1">
            <Label htmlFor="custom-major">Major</Label>
            <Input
              id="custom-major"
              className="w-20"
              inputMode="numeric"
              value={customMajor}
              onChange={(e) => handleCustomChange(customRegion, e.target.value, customMinor)}
            />
          </div>
          <div className="flex flex-col gap-1">
            <Label htmlFor="custom-minor">Minor</Label>
            <Input
              id="custom-minor"
              className="w-20"
              inputMode="numeric"
              value={customMinor}
              onChange={(e) => handleCustomChange(customRegion, customMajor, e.target.value)}
            />
          </div>
        </div>
      )}
      {customInvalid && (
        <p className="text-sm text-destructive">
          Region must be non-empty; major and minor must be non-negative integers.
        </p>
      )}
      {value && (
        <p className="text-sm text-muted-foreground">
          Selected: {value.region} v{value.majorVersion}.{value.minorVersion}
        </p>
      )}
    </div>
  );
}
