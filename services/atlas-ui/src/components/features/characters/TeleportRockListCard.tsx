import { useState } from "react";
import { Trash2 } from "lucide-react";
import { toast } from "sonner";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useItemData } from "@/lib/hooks/useItemData";
import { useMap } from "@/lib/hooks/api/useMaps";
import { useRemoveTeleportRockMap } from "@/lib/hooks/api/useTeleportRocks";
import type { TeleportRockListType } from "@/services/api/teleport-rocks.service";
import { AddTeleportRockMapDialog } from "@/components/features/characters/AddTeleportRockMapDialog";

const REGULAR_ROCK_ITEM_ID = 5040000;
const VIP_ROCK_ITEM_ID = 5041000;

interface TeleportRockListCardProps {
  characterId: string;
  list: TeleportRockListType;
  maps: number[];
  capacity: number;
}

interface MapRowProps {
  characterId: string;
  list: TeleportRockListType;
  mapId: number;
}

/**
 * Renders a single map row. Pulled out as its own component (rather than
 * inlined in a `.map()` callback) because `useMap` is a hook and hooks
 * cannot be called from within an array-map callback — each row needs its
 * own top-level hook invocation.
 */
function MapRow({ characterId, list, mapId }: MapRowProps) {
  const { data: map } = useMap(String(mapId));
  const { mutateAsync: removeMap, isPending } = useRemoveTeleportRockMap();

  const mapName = map?.attributes.name ?? `Map ${mapId}`;

  const handleRemove = async () => {
    try {
      await removeMap({ characterId, list, mapId });
      toast.success(`Removed ${mapName} from the ${list} teleport rock list`);
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error
          ? error.message
          : "An unexpected error occurred while removing the map";
      toast.error(errorMessage);
    }
  };

  return (
    <div className="flex items-center justify-between rounded-sm px-2 py-1.5 text-sm">
      <span>{mapName}</span>
      <Button
        variant="ghost"
        size="icon"
        className="h-6 w-6 hover:bg-red-100 hover:text-red-600"
        onClick={handleRemove}
        disabled={isPending}
        aria-label={`Remove map ${mapId}`}
      >
        <Trash2 className="h-3 w-3" />
      </Button>
    </div>
  );
}

export function TeleportRockListCard({
  characterId,
  list,
  maps,
  capacity,
}: TeleportRockListCardProps) {
  const [open, setOpen] = useState(false);
  const { iconUrl } = useItemData(
    list === "vip" ? VIP_ROCK_ITEM_ID : REGULAR_ROCK_ITEM_ID,
  );
  const title =
    list === "vip" ? "VIP Teleport Rocks" : "Regular Teleport Rocks";
  const atCapacity = maps.length >= capacity;

  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        {iconUrl && (
          <img
            src={iconUrl}
            alt={title}
            width={24}
            height={24}
            className="h-6 w-6"
          />
        )}
        <CardTitle className="flex-1">{title}</CardTitle>
        <Badge variant="secondary">{`${maps.length} of ${capacity}`}</Badge>
      </CardHeader>
      <CardContent>
        <ScrollArea className="h-48 rounded-md border">
          <div className="flex flex-col p-2">
            {maps.length === 0 && (
              <p className="text-sm text-muted-foreground p-2">
                No maps added.
              </p>
            )}
            {maps.map((mapId) => (
              <MapRow
                key={mapId}
                characterId={characterId}
                list={list}
                mapId={mapId}
              />
            ))}
          </div>
        </ScrollArea>
      </CardContent>
      <CardFooter>
        <Button
          variant="outline"
          className="w-full"
          aria-label="Add"
          disabled={atCapacity}
          onClick={() => setOpen(true)}
        >
          Add
        </Button>
      </CardFooter>
      {open && (
        <AddTeleportRockMapDialog
          characterId={characterId}
          list={list}
          existingMapIds={maps}
          open={open}
          onOpenChange={setOpen}
        />
      )}
    </Card>
  );
}
