import { useState } from "react";
import { TriangleAlert } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useItemName } from "@/lib/hooks/api/useItemStrings";
import type { SearchPoolKey } from "@/lib/items/poolSearchConfig";
import { useItemSearch } from "./useItemSearch";
import { ItemSearchResults } from "./ItemSearchResults";

interface ItemPickerProps {
  /** 0 means unset. */
  value: number;
  onChange: (id: number) => void;
  /** Defaults to the unfiltered all-compartment pool. */
  poolKey?: SearchPoolKey;
  /** Trigger label rendered when value is 0. */
  placeholder?: string;
  /** Renders a "None" row that clears the value back to 0. */
  allowClear?: boolean;
  disabled?: boolean;
  /** Test hook: pass 0 to disable debouncing. */
  debounceMs?: number;
  /** Applied to the trigger button so a <Label htmlFor> can address it. */
  id?: string;
}

export function ItemPicker({
  value,
  onChange,
  poolKey = "items",
  placeholder = "Select an item…",
  allowClear = false,
  disabled = false,
  debounceMs = 300,
  id,
}: ItemPickerProps) {
  const [open, setOpen] = useState(false);
  const search = useItemSearch({ poolKey, open, debounceMs });
  // Guard on value > 0: String(0) is "0", which is truthy and would fire a
  // lookup for item id 0.
  const current = useItemName(value > 0 ? String(value) : "");

  const label =
    value === 0
      ? placeholder
      : current.data
        ? `${current.data} · ${value}`
        : `Item ${value}`;
  // atlas-data coverage varies by version: unresolvable is a hint, not an error.
  const unresolved = value > 0 && !current.data && current.isError;

  const pick = (nextId: number) => {
    onChange(nextId);
    setOpen(false);
    search.reset();
  };

  return (
    <div className="space-y-1">
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            type="button"
            variant="outline"
            className="w-full justify-start font-normal"
            disabled={disabled}
            {...(id ? { id } : {})}
          >
            {label}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-96 p-2" align="start">
          <Input
            autoFocus
            value={search.search}
            onChange={(e) => search.setSearch(e.target.value)}
            placeholder="Search by name or enter an id…"
          />
          <ItemSearchResults
            rows={search.rows}
            manualId={search.manualId}
            isLoading={search.isLoading}
            isError={search.isError}
            settledTerm={search.settledTerm}
            onPick={pick}
            {...(allowClear
              ? {
                  leadingRow: (
                    <li
                      role="option"
                      aria-selected={false}
                      tabIndex={0}
                      onClick={() => pick(0)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter" || e.key === " ") {
                          e.preventDefault();
                          pick(0);
                        }
                      }}
                      className="cursor-pointer rounded px-2 py-1 text-sm hover:bg-accent focus-visible:bg-accent"
                    >
                      None
                    </li>
                  ),
                }
              : {})}
          />
          {search.hasMore && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="mt-1 w-full"
              onClick={search.loadMore}
            >
              Load more
            </Button>
          )}
        </PopoverContent>
      </Popover>
      {unresolved && (
        <p className="flex items-center gap-1 text-xs text-warning-foreground">
          <TriangleAlert className="size-3" />
          couldn&apos;t resolve this item&apos;s name for this version
        </p>
      )}
    </div>
  );
}
