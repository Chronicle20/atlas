import { useEffect, useMemo, useState } from "react";
import { useQuery, keepPreviousData } from "@tanstack/react-query";
import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { itemsService } from "@/services/api/items.service";
import type { ItemSearchFilters } from "@/services/api/items.service";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { useTenant } from "@/context/tenant-context";
import { POOL_SEARCH_CONFIGS, type SearchPoolKey } from "./poolSearchConfig";

interface ItemSearchComboboxProps {
  poolKey: SearchPoolKey;
  existingIds: number[];
  onAdd: (id: number) => void;
  triggerLabel?: string;
  /** Test hook: pass 0 to disable debouncing. */
  debounceMs?: number;
}

const PAGE_SIZE = 50;

export function ItemSearchCombobox({
  poolKey,
  existingIds,
  onAdd,
  triggerLabel = "Add",
  debounceMs = 300,
}: ItemSearchComboboxProps) {
  const { activeTenant } = useTenant();
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  // The settled query term and its page are held TOGETHER so they can only
  // ever change atomically — the page must never move independently of the
  // term it belongs to. `term` updates only from the debounce timer's
  // callback (async — not a synchronous setState-in-effect, so this stays
  // clean under react-hooks/set-state-in-effect); "Load more" advances
  // `page` via a functional update that leaves `term` untouched. Raw
  // keystrokes update only `search` (below), never `settled` directly —
  // that decoupling is exactly what caused the prior regression: a
  // synchronous page reset on every keystroke could pair the OLD settled
  // term with a NEW page number and fire an un-debounced query.
  const [settled, setSettled] = useState({ term: "", page: 1 });

  useEffect(() => {
    const handle = setTimeout(() => {
      setSettled({ term: search, page: 1 });
    }, debounceMs);
    return () => clearTimeout(handle);
  }, [search, debounceMs]);

  const cfg = POOL_SEARCH_CONFIGS[poolKey];

  const filters: ItemSearchFilters = {
    pageNumber: settled.page,
    pageSize: PAGE_SIZE,
    ...(settled.term ? { q: settled.term } : {}),
    ...(cfg.compartment ? { compartment: cfg.compartment } : {}),
    ...(cfg.subcategory ? { subcategory: cfg.subcategory } : {}),
  };

  const query = useQuery({
    queryKey: ["item-search", poolKey, settled.term, settled.page],
    queryFn: () => itemsService.searchItems(filters),
    enabled: open && !!activeTenant && settled.term.trim().length > 0,
    placeholderData: keepPreviousData,
    staleTime: 10 * 60 * 1000,
  });

  const rows = useMemo(() => {
    const items = query.data?.items ?? [];
    return cfg.clientSubcategories
      ? items.filter((r) => cfg.clientSubcategories!.has(r.subcategory))
      : items;
  }, [query.data, cfg.clientSubcategories]);

  const manualId = /^\d+$/.test(search.trim())
    ? Number(search.trim())
    : undefined;
  const hasMore = (query.data?.lastPage ?? 1) > settled.page;

  const handleAdd = (id: number) => {
    if (existingIds.includes(id)) return;
    onAdd(id);
    setOpen(false);
    setSearch("");
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button type="button" variant="outline" size="sm">
          <Plus className="size-4" /> {triggerLabel}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-80 p-2" align="start">
        <Input
          autoFocus
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search by name or enter an id…"
        />
        <ul
          role="listbox"
          className="mt-2 max-h-64 space-y-0.5 overflow-y-auto"
        >
          {rows.map((row) => {
            const id = Number(row.id);
            const inPool = existingIds.includes(id);
            return (
              <li
                key={row.id}
                role="option"
                aria-selected={false}
                aria-disabled={inPool}
                tabIndex={inPool ? -1 : 0}
                onClick={() => !inPool && handleAdd(id)}
                onKeyDown={(e) => {
                  if ((e.key === "Enter" || e.key === " ") && !inPool) {
                    e.preventDefault();
                    handleAdd(id);
                  }
                }}
                className={
                  inPool
                    ? "flex cursor-not-allowed items-center gap-2 rounded px-2 py-1 opacity-50"
                    : "flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-accent focus-visible:bg-accent"
                }
              >
                {activeTenant && (
                  <img
                    src={getAssetIconUrl(
                      activeTenant.id,
                      activeTenant.attributes.region,
                      activeTenant.attributes.majorVersion,
                      activeTenant.attributes.minorVersion,
                      "item",
                      id,
                    )}
                    alt=""
                    width={24}
                    height={24}
                    loading="lazy"
                    className="[image-rendering:pixelated]"
                    onError={(e) => {
                      (e.target as HTMLImageElement).style.visibility =
                        "hidden";
                    }}
                  />
                )}
                <span className="flex-1 truncate text-sm">{row.name}</span>
                <span className="font-mono text-xs text-muted-foreground">
                  {row.id}
                </span>
                {inPool && (
                  <span className="text-xs text-muted-foreground">Added</span>
                )}
              </li>
            );
          })}
          {manualId !== undefined && (
            <li
              role="option"
              aria-selected={false}
              tabIndex={0}
              onClick={() => handleAdd(manualId)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  handleAdd(manualId);
                }
              }}
              className="cursor-pointer rounded px-2 py-1 text-sm hover:bg-accent focus-visible:bg-accent"
            >
              Use id {manualId}
            </li>
          )}
          {query.isLoading && settled.term && (
            <li className="px-2 py-1 text-sm text-muted-foreground">
              Searching…
            </li>
          )}
          {query.isError && settled.term && (
            <li className="px-2 py-1 text-sm text-warning-foreground">
              Search failed — enter an id manually
            </li>
          )}
          {!query.isLoading &&
            !query.isError &&
            settled.term &&
            rows.length === 0 &&
            manualId === undefined && (
              <li className="px-2 py-1 text-sm text-muted-foreground">
                No matches.
              </li>
            )}
        </ul>
        {hasMore && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="mt-1 w-full"
            onClick={() => setSettled((s) => ({ ...s, page: s.page + 1 }))}
          >
            Load more
          </Button>
        )}
      </PopoverContent>
    </Popover>
  );
}
