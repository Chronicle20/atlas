import { useMemo, useState } from "react";
import { Check } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import type { AncestryClass } from "@/lib/socket/ancestry";
import type { GridFilters, SortDirection, SortKey } from "@/lib/socket/matrix";
import type { DefinitionKind, DefinitionState, SocketObject } from "@/lib/socket/model";
import { cn } from "@/lib/utils";

export interface GridToolbarProps {
  kind: DefinitionKind;
  /** Absent on the four per-object pages, where the mode is fixed (FR-7.3). */
  onKindChange?: (kind: DefinitionKind) => void;

  objects: SocketObject[];
  selectedKeys: string[];
  /** Absent on the four per-object pages: the column set is fixed (FR-7.3). */
  onSelectedKeysChange?: (keys: string[]) => void;

  baselineKey: string;
  /** Absent on the four per-object pages (FR-7.3). */
  onBaselineChange?: (key: string) => void;

  showFName: boolean;
  onShowFNameChange: (show: boolean) => void;

  filters: GridFilters;
  onFiltersChange: (filters: GridFilters) => void;

  sort: { key: SortKey; direction: SortDirection };
  onSortChange: (sort: { key: SortKey; direction: SortDirection }) => void;

  /** Tenant pages only (FR-4.5). Absent elsewhere. */
  ancestryFilterOptions?: {
    value: AncestryClass[];
    onChange: (value: AncestryClass[]) => void;
  };
}

const STATE_OPTIONS: { value: DefinitionState; label: string }[] = [
  { value: "defined", label: "Defined" },
  { value: "unsupported", label: "Unsupported" },
  { value: "undefined", label: "Undefined" },
];

/**
 * The closed set of real service values in the corpus - `('channel',)` x2418,
 * `('login',)` x389, `('login','channel')` x27, key absent x25 (measured
 * against the seed templates). There is no third service, so the filter
 * offers exactly these two options.
 */
const SERVICE_OPTIONS: { value: string; label: string }[] = [
  { value: "login", label: "Login" },
  { value: "channel", label: "Channel" },
];

const ANCESTRY_OPTIONS: { value: AncestryClass; label: string }[] = [
  { value: "same", label: "Same as template" },
  { value: "modified", label: "Modified" },
  { value: "tenant-only", label: "Tenant only" },
  { value: "missing", label: "Missing" },
  { value: "unsupported", label: "Unsupported" },
];

type HasOptionsChoice = "any" | "yes" | "no";

const HAS_OPTIONS_CHOICES: { value: HasOptionsChoice; label: string }[] = [
  { value: "any", label: "Any" },
  { value: "yes", label: "Has options" },
  { value: "no", label: "No options" },
];

/**
 * A trigger button + popover listbox, shared by every filter that picks from
 * a small closed set of values. `onSelect` reports the clicked value; whether
 * that means "toggle into a multi-select set" or "replace a single choice" is
 * the caller's decision - this component only renders the list and reports
 * clicks, matching the presentational contract of the toolbar as a whole.
 */
function OptionListPopover<T extends string>({
  label,
  options,
  selectedValues,
  onSelect,
  multi = true,
}: {
  label: string;
  options: { value: T; label: string }[];
  selectedValues: T[];
  onSelect: (value: T) => void;
  multi?: boolean;
}) {
  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button type="button" variant="outline" size="sm">
          {label}
          {multi && selectedValues.length > 0 && (
            <span className="text-muted-foreground ml-1 text-xs">
              ({selectedValues.length})
            </span>
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-56 p-1" align="start">
        <ul role="listbox" aria-label={label} aria-multiselectable={multi}>
          {options.map((opt) => {
            const isSelected = selectedValues.includes(opt.value);
            return (
              <li
                key={opt.value}
                role="option"
                aria-selected={isSelected}
                tabIndex={0}
                onClick={() => onSelect(opt.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    onSelect(opt.value);
                  }
                }}
                className={cn(
                  "hover:bg-accent flex cursor-pointer items-center justify-between rounded-sm px-2 py-1.5 text-sm",
                  isSelected && "bg-accent",
                )}
              >
                {opt.label}
                {isSelected && <Check className="h-4 w-4" />}
              </li>
            );
          })}
        </ul>
      </PopoverContent>
    </Popover>
  );
}

const ALL_REGIONS = "__all-regions__";
const ALL_VERSIONS = "__all-versions__";

/** FR-2.12: filtered by Region and Version, one checkbox per object. */
function ColumnPicker({
  objects,
  selectedKeys,
  onChange,
}: {
  objects: SocketObject[];
  selectedKeys: string[];
  onChange: (keys: string[]) => void;
}) {
  const [region, setRegion] = useState("");
  const [version, setVersion] = useState("");

  const regions = useMemo(
    () => Array.from(new Set(objects.map((o) => o.region))).sort(),
    [objects],
  );
  const versions = useMemo(
    () =>
      Array.from(
        new Set(objects.map((o) => `${o.majorVersion}.${o.minorVersion}`)),
      ).sort(),
    [objects],
  );

  const filtered = objects.filter(
    (o) =>
      (region === "" || o.region === region) &&
      (version === "" || `${o.majorVersion}.${o.minorVersion}` === version),
  );

  const toggle = (key: string) => {
    onChange(
      selectedKeys.includes(key)
        ? selectedKeys.filter((k) => k !== key)
        : [...selectedKeys, key],
    );
  };

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button type="button" variant="outline" size="sm">
          Columns
          <span className="text-muted-foreground ml-1 text-xs">
            ({selectedKeys.length})
          </span>
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-72 space-y-2" align="start">
        <div className="flex gap-2">
          {/* Radix Select reserves value="" for "no selection" internally,
              so "all" is modelled as its own sentinel item rather than an
              empty-string option. */}
          <Select
            value={region === "" ? ALL_REGIONS : region}
            onValueChange={(v) => setRegion(v === ALL_REGIONS ? "" : v)}
          >
            <SelectTrigger
              aria-label="Filter columns by region"
              className="h-8 flex-1 text-sm"
            >
              <SelectValue placeholder="All regions" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL_REGIONS}>All regions</SelectItem>
              {regions.map((r) => (
                <SelectItem key={r} value={r}>
                  {r}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select
            value={version === "" ? ALL_VERSIONS : version}
            onValueChange={(v) => setVersion(v === ALL_VERSIONS ? "" : v)}
          >
            <SelectTrigger
              aria-label="Filter columns by version"
              className="h-8 flex-1 text-sm"
            >
              <SelectValue placeholder="All versions" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL_VERSIONS}>All versions</SelectItem>
              {versions.map((v) => (
                <SelectItem key={v} value={v}>
                  {v}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <ul className="max-h-64 space-y-1 overflow-auto">
          {filtered.map((o) => {
            const id = `toolbar-column-${o.key}`;
            return (
              <li key={o.key} className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id={id}
                  checked={selectedKeys.includes(o.key)}
                  onChange={() => toggle(o.key)}
                  className="h-4 w-4"
                />
                <label htmlFor={id} className="text-sm">
                  {o.label}
                </label>
              </li>
            );
          })}
          {filtered.length === 0 && (
            <li className="text-muted-foreground px-1 py-1 text-sm">
              No matching objects.
            </li>
          )}
        </ul>
      </PopoverContent>
    </Popover>
  );
}

/** The baseline selector lists only the currently-selected objects (FR-2.9/2.10). */
function BaselineSelector({
  objects,
  selectedKeys,
  baselineKey,
  onChange,
}: {
  objects: SocketObject[];
  selectedKeys: string[];
  baselineKey: string;
  onChange: (key: string) => void;
}) {
  const options = objects.filter((o) => selectedKeys.includes(o.key));

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button type="button" variant="outline" size="sm">
          Baseline
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-56 p-1" align="start">
        <ul role="listbox" aria-label="Baseline">
          {options.map((o) => {
            const isSelected = o.key === baselineKey;
            return (
              <li
                key={o.key}
                role="option"
                aria-selected={isSelected}
                tabIndex={0}
                onClick={() => onChange(o.key)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    onChange(o.key);
                  }
                }}
                className={cn(
                  "hover:bg-accent flex cursor-pointer items-center justify-between rounded-sm px-2 py-1.5 text-sm",
                  isSelected && "bg-accent",
                )}
              >
                {o.label}
                {isSelected && <Check className="h-4 w-4" />}
              </li>
            );
          })}
        </ul>
      </PopoverContent>
    </Popover>
  );
}

/**
 * Controlled toolbar for the packet definition matrix. It owns no filter,
 * sort, search or column-selection state - every control renders the current
 * prop value and reports changes through a handler; `filterRows` / `sortRows`
 * in `lib/socket/matrix.ts` remain the only place that logic lives.
 *
 * FR-7.3: a control whose handler prop is absent is not rendered at all - not
 * disabled. That's how the four per-object pages drop the mode switch, column
 * picker and baseline selector, and how a tenant with no inferred ancestor
 * drops the difference-from-ancestor filter (FR-4.5/8.2).
 */
export function GridToolbar({
  kind,
  onKindChange,
  objects,
  selectedKeys,
  onSelectedKeysChange,
  baselineKey,
  onBaselineChange,
  showFName,
  onShowFNameChange,
  filters,
  onFiltersChange,
  sort,
  onSortChange,
  ancestryFilterOptions,
}: GridToolbarProps) {
  const toggleState = (value: DefinitionState) => {
    const next = filters.states.includes(value)
      ? filters.states.filter((s) => s !== value)
      : [...filters.states, value];
    onFiltersChange({ ...filters, states: next });
  };

  const toggleService = (value: string) => {
    const next = filters.services.includes(value)
      ? filters.services.filter((s) => s !== value)
      : [...filters.services, value];
    onFiltersChange({ ...filters, services: next });
  };

  const hasOptionsChoice: HasOptionsChoice =
    filters.hasOptions === null ? "any" : filters.hasOptions ? "yes" : "no";
  const selectHasOptions = (value: HasOptionsChoice) => {
    onFiltersChange({
      ...filters,
      hasOptions: value === "any" ? null : value === "yes",
    });
  };

  return (
    <div className="flex flex-wrap items-center gap-3 border-b p-3">
      <Input
        type="search"
        aria-label="Search definitions"
        placeholder="Search definitions..."
        value={filters.query}
        onChange={(e) => onFiltersChange({ ...filters, query: e.target.value })}
        className="w-56"
      />

      {onKindChange && (
        <RadioGroup
          value={kind}
          onValueChange={(v) => onKindChange(v as DefinitionKind)}
          className="flex flex-row items-center gap-3"
          aria-label="Mode"
        >
          <div className="flex items-center gap-1.5">
            <RadioGroupItem value="handler" id="toolbar-kind-handler" />
            <Label htmlFor="toolbar-kind-handler">Handlers</Label>
          </div>
          <div className="flex items-center gap-1.5">
            <RadioGroupItem value="writer" id="toolbar-kind-writer" />
            <Label htmlFor="toolbar-kind-writer">Writers</Label>
          </div>
        </RadioGroup>
      )}

      {onSelectedKeysChange && (
        <ColumnPicker
          objects={objects}
          selectedKeys={selectedKeys}
          onChange={onSelectedKeysChange}
        />
      )}

      {onBaselineChange && (
        <BaselineSelector
          objects={objects}
          selectedKeys={selectedKeys}
          baselineKey={baselineKey}
          onChange={onBaselineChange}
        />
      )}

      <div className="flex items-center gap-1.5">
        <Switch
          id="toolbar-fname"
          checked={showFName}
          onCheckedChange={onShowFNameChange}
        />
        <Label htmlFor="toolbar-fname">fname</Label>
      </div>

      <OptionListPopover
        label="State"
        options={STATE_OPTIONS}
        selectedValues={filters.states}
        onSelect={toggleState}
      />

      <OptionListPopover
        label="Has options"
        options={HAS_OPTIONS_CHOICES}
        selectedValues={[hasOptionsChoice]}
        onSelect={selectHasOptions}
        multi={false}
      />

      <div className="flex items-center gap-1.5">
        <input
          type="checkbox"
          id="toolbar-options-missing"
          checked={filters.optionsMissingOnly}
          onChange={(e) =>
            onFiltersChange({ ...filters, optionsMissingOnly: e.target.checked })
          }
          className="h-4 w-4"
        />
        <label htmlFor="toolbar-options-missing" className="text-sm">
          Supplies no options
        </label>
      </div>

      <OptionListPopover
        label="Service"
        options={SERVICE_OPTIONS}
        selectedValues={filters.services}
        onSelect={toggleService}
      />

      {ancestryFilterOptions && (
        <OptionListPopover
          label="vs Template"
          options={ANCESTRY_OPTIONS}
          selectedValues={ancestryFilterOptions.value}
          onSelect={(value) => {
            const next = ancestryFilterOptions.value.includes(value)
              ? ancestryFilterOptions.value.filter((v) => v !== value)
              : [...ancestryFilterOptions.value, value];
            ancestryFilterOptions.onChange(next);
          }}
        />
      )}

      <RadioGroup
        value={sort.key}
        onValueChange={(v) =>
          onSortChange({ key: v as SortKey, direction: sort.direction })
        }
        className="flex flex-row items-center gap-3"
        aria-label="Sort by"
      >
        <div className="flex items-center gap-1.5">
          <RadioGroupItem value="opcode" id="toolbar-sort-opcode" />
          <Label htmlFor="toolbar-sort-opcode">Opcode</Label>
        </div>
        <div className="flex items-center gap-1.5">
          <RadioGroupItem value="name" id="toolbar-sort-name" />
          <Label htmlFor="toolbar-sort-name">Name</Label>
        </div>
        <div className="flex items-center gap-1.5">
          <RadioGroupItem value="state" id="toolbar-sort-state" />
          <Label htmlFor="toolbar-sort-state">State</Label>
        </div>
      </RadioGroup>

      <Button
        type="button"
        variant="outline"
        size="sm"
        aria-label="Toggle sort direction"
        onClick={() =>
          onSortChange({
            key: sort.key,
            direction: sort.direction === "asc" ? "desc" : "asc",
          })
        }
      >
        {sort.direction === "asc" ? "Ascending" : "Descending"}
      </Button>
    </div>
  );
}
