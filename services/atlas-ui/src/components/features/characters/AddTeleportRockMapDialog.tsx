import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useDebounce } from "@/lib/utils/debounce";
import { useMapsByName } from "@/lib/hooks/api/useMaps";
import { useAddTeleportRockMap } from "@/lib/hooks/api/useTeleportRocks";
import type { TeleportRockListType } from "@/services/api/teleport-rocks.service";
import { toast } from "sonner";

interface AddTeleportRockMapDialogProps {
  characterId: string;
  list: TeleportRockListType;
  existingMapIds: number[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function AddTeleportRockMapDialog({
  characterId,
  list,
  existingMapIds,
  open,
  onOpenChange,
}: AddTeleportRockMapDialogProps) {
  const [query, setQuery] = useState("");
  const debounced = useDebounce(query.trim(), 300);
  const { data: results, isLoading } = useMapsByName(debounced);
  const { mutateAsync: addMap, isPending } = useAddTeleportRockMap();

  const filteredResults = (results ?? []).filter(
    (m) => !existingMapIds.includes(Number(m.id)),
  );

  const handleSelect = async (mapId: number, mapName: string) => {
    try {
      await addMap({ characterId, list, mapId });
      toast.success(`Added ${mapName} to the ${list} teleport rock list`);
      onOpenChange(false);
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error
          ? error.message
          : "An unexpected error occurred while adding the map";
      toast.error(errorMessage);
    }
  };

  const handleOpenChange = (newOpen: boolean) => {
    onOpenChange(newOpen);
    if (!newOpen) {
      setQuery("");
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Add Teleport Rock Map</DialogTitle>
          <DialogDescription>
            Search for a map to add to the {list} teleport rock list.
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <Input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search maps…"
            disabled={isPending}
            autoFocus
          />
          <ScrollArea className="h-72 rounded-md border">
            <div className="flex flex-col p-2">
              {isLoading && (
                <p className="text-sm text-muted-foreground p-2">Searching…</p>
              )}
              {!isLoading && debounced && filteredResults.length === 0 && (
                <p className="text-sm text-muted-foreground p-2">
                  No maps found.
                </p>
              )}
              {filteredResults.map((m) => (
                <button
                  key={m.id}
                  type="button"
                  disabled={isPending}
                  onClick={() => handleSelect(Number(m.id), m.attributes.name)}
                  className="flex items-center justify-between rounded-sm px-2 py-1.5 text-sm text-left hover:bg-accent hover:text-accent-foreground disabled:opacity-50"
                >
                  <span>{m.attributes.name}</span>
                  <span className="text-muted-foreground">{m.id}</span>
                </button>
              ))}
            </div>
          </ScrollArea>
        </div>
      </DialogContent>
    </Dialog>
  );
}
