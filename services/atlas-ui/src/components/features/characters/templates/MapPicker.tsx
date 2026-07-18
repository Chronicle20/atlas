import { useEffect, useState } from "react";
import { TriangleAlert } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useMap, useMapsByName } from "@/lib/hooks/api/useMaps";

interface MapPickerProps {
  value: number;
  onChange: (mapId: number) => void;
  /** Test hook: pass 0 to disable debouncing. */
  debounceMs?: number;
}

export function MapPicker({
  value,
  onChange,
  debounceMs = 300,
}: MapPickerProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [debounced, setDebounced] = useState("");

  useEffect(() => {
    // Always go through setTimeout — even at debounceMs=0 — so setState
    // happens in the timer callback, not synchronously in the effect body
    // (react-hooks/set-state-in-effect), matching the pattern established
    // in ItemSearchCombobox (Task 6).
    const handle = setTimeout(() => setDebounced(search), debounceMs);
    return () => clearTimeout(handle);
  }, [search, debounceMs]);

  const current = useMap(String(value));
  const results = useMapsByName(debounced.trim());

  const currentLabel = current.data
    ? `${current.data.attributes.name} · ${current.data.attributes.streetName} · ${value}`
    : `Map ${value}`;
  // atlas-data coverage varies by version: unresolvable is a hint, not an error.
  const unresolved = value > 0 && !current.data && current.isError;

  const manualId = /^\d+$/.test(search.trim())
    ? Number(search.trim())
    : undefined;

  const pick = (mapId: number) => {
    onChange(mapId);
    setOpen(false);
    setSearch("");
  };

  return (
    <div className="space-y-1">
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            type="button"
            variant="outline"
            className="w-full justify-start font-normal"
          >
            {currentLabel}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-96 p-2" align="start">
          <Input
            autoFocus
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search maps by name or enter an id…"
          />
          <ul
            role="listbox"
            className="mt-2 max-h-64 space-y-0.5 overflow-y-auto"
          >
            {(results.data ?? []).map((m) => (
              <li
                key={m.id}
                role="option"
                aria-selected={false}
                tabIndex={0}
                onClick={() => pick(Number(m.id))}
                onKeyDown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    pick(Number(m.id));
                  }
                }}
                className="flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-accent focus-visible:bg-accent"
              >
                <span className="flex-1 truncate text-sm">
                  {m.attributes.name} · {m.attributes.streetName}
                </span>
                <span className="font-mono text-xs text-muted-foreground">
                  {m.id}
                </span>
              </li>
            ))}
            {manualId !== undefined && (
              <li
                role="option"
                aria-selected={false}
                tabIndex={0}
                onClick={() => pick(manualId)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    pick(manualId);
                  }
                }}
                className="cursor-pointer rounded px-2 py-1 text-sm hover:bg-accent focus-visible:bg-accent"
              >
                Use id {manualId}
              </li>
            )}
            {results.isLoading && debounced.trim() && (
              <li className="px-2 py-1 text-sm text-muted-foreground">
                Searching…
              </li>
            )}
            {results.isError && debounced.trim() && (
              <li className="px-2 py-1 text-sm text-warning-foreground">
                Search failed — enter an id manually
              </li>
            )}
          </ul>
        </PopoverContent>
      </Popover>
      {unresolved && (
        <p className="flex items-center gap-1 text-xs text-warning-foreground">
          <TriangleAlert className="size-3" />
          not found in map data for this version
        </p>
      )}
    </div>
  );
}
