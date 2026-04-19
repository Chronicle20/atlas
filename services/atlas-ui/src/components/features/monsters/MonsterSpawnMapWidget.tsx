import { Link } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import type { MonsterSpawnMapData } from "@/types/models/monster";

interface MonsterSpawnMapWidgetProps {
  entry: MonsterSpawnMapData;
}

export function MonsterSpawnMapWidget({ entry }: MonsterSpawnMapWidgetProps) {
  const { id } = entry;
  const { name, streetName, spawnCount } = entry.attributes;
  return (
    <Link
      to={`/maps/${id}`}
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
