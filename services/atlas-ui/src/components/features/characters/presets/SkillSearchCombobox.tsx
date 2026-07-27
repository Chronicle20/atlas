import { useEffect, useState } from "react";
import { useQuery, keepPreviousData } from "@tanstack/react-query";
import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { skillsService } from "@/services/api/skills.service";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { useTenant } from "@/context/tenant-context";

interface SkillSearchComboboxProps {
  existingIds: number[];
  onAdd: (id: number) => void;
  /** Test hook: pass 0 to disable debouncing. */
  debounceMs?: number;
}

const PAGE_SIZE = 50;

/**
 * Search-by-name skill picker mirroring ItemSearchCombobox: debounced term,
 * icon + name + id rows, a "Use id N" escape hatch for numeric input, and
 * term+page held together so a page can never pair with a stale term.
 */
export function SkillSearchCombobox({
  existingIds,
  onAdd,
  debounceMs = 300,
}: SkillSearchComboboxProps) {
  const { activeTenant } = useTenant();
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [settled, setSettled] = useState({ term: "", page: 1 });

  useEffect(() => {
    const handle = setTimeout(() => {
      setSettled({ term: search, page: 1 });
    }, debounceMs);
    return () => clearTimeout(handle);
  }, [search, debounceMs]);

  const query = useQuery({
    queryKey: ["skill-search", settled.term, settled.page],
    queryFn: () =>
      skillsService.searchSkills(settled.term, {
        number: settled.page,
        size: PAGE_SIZE,
      }),
    enabled: open && !!activeTenant && settled.term.trim().length > 0,
    placeholderData: keepPreviousData,
    staleTime: 10 * 60 * 1000,
  });

  const rows = query.data?.skills ?? [];
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
          <Plus className="size-4" /> Add
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
            const inPool = existingIds.includes(row.id);
            return (
              <li
                key={row.id}
                role="option"
                aria-selected={false}
                aria-disabled={inPool}
                tabIndex={inPool ? -1 : 0}
                onClick={() => !inPool && handleAdd(row.id)}
                onKeyDown={(e) => {
                  if ((e.key === "Enter" || e.key === " ") && !inPool) {
                    e.preventDefault();
                    handleAdd(row.id);
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
                      "skill",
                      row.id,
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
