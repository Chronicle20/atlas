import { Link } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import type { NpcSpawnMap } from "@/types/models/npc";

interface NpcSpawnMapWidgetProps {
  entry: NpcSpawnMap;
}

export function NpcSpawnMapWidget({ entry }: NpcSpawnMapWidgetProps) {
  const { mapId, name, streetName, spawnCount } = entry;
  return (
    <Link
      to={`/maps/${mapId}`}
      className="flex flex-col gap-1 rounded-md border bg-card p-3 hover:bg-accent transition-colors"
    >
      <div className="flex items-center gap-2 flex-wrap">
        <span className="text-sm font-medium truncate">{name}</span>
        {streetName && <Badge variant="secondary">{streetName}</Badge>}
      </div>
      <div>
        <Badge variant="outline">
          {spawnCount === 1 ? "1 spawn" : `${spawnCount} spawns`}
        </Badge>
      </div>
    </Link>
  );
}
