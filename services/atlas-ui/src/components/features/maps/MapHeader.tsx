import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

interface MapHeaderProps {
  mapId: string;
  name: string;
  streetName?: string;
  spawnCount: number | undefined;
}

export function MapHeader({ mapId, name, streetName, spawnCount }: MapHeaderProps) {
  return (
    <div className="flex flex-col gap-2">
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <h2
              tabIndex={0}
              className="text-2xl font-bold tracking-tight cursor-help inline-block w-fit focus:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded"
            >
              {name}
            </h2>
          </TooltipTrigger>
          <TooltipContent copyable>
            <p>{mapId}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
      <div className="flex items-center gap-2 flex-wrap">
        {streetName && <Badge variant="secondary">{streetName}</Badge>}
        {spawnCount === undefined ? (
          <Skeleton className="h-5 w-20 rounded-full" />
        ) : (
          <Badge variant="outline">
            {spawnCount === 1 ? "1 spawn" : `${spawnCount} spawns`}
          </Badge>
        )}
      </div>
    </div>
  );
}
