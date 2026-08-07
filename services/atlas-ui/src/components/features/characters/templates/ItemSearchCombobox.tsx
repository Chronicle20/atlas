import { useState } from "react";
import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import type { SearchPoolKey } from "@/lib/items/poolSearchConfig";
import { useItemSearch } from "@/components/features/items/item-search/useItemSearch";
import { ItemSearchResults } from "@/components/features/items/item-search/ItemSearchResults";

interface ItemSearchComboboxProps {
  poolKey: SearchPoolKey;
  existingIds: number[];
  onAdd: (id: number) => void;
  triggerLabel?: string;
  /** Test hook: pass 0 to disable debouncing. */
  debounceMs?: number;
}

export function ItemSearchCombobox({
  poolKey,
  existingIds,
  onAdd,
  triggerLabel = "Add",
  debounceMs = 300,
}: ItemSearchComboboxProps) {
  const [open, setOpen] = useState(false);
  const search = useItemSearch({ poolKey, open, debounceMs });

  const handleAdd = (id: number) => {
    if (existingIds.includes(id)) return;
    onAdd(id);
    setOpen(false);
    search.reset();
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
          disabledIds={existingIds}
          onPick={handleAdd}
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
  );
}
