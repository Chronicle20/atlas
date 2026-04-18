import { Link } from "react-router-dom";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { MapCell } from "@/components/map-cell";
import { useTenant } from "@/context/tenant-context";
import type { MapPortalData } from "@/services/api/map-entities.service";

const NONE_MAP_ID = 999999999;

interface ConnectedMapsRowProps {
  mapId: string;
  portals: MapPortalData[] | undefined;
}

export function ConnectedMapsRow({ mapId, portals }: ConnectedMapsRowProps) {
  const { activeTenant } = useTenant();

  if (portals === undefined) {
    return (
      <section>
        <h3 className="text-sm font-semibold mb-2">Connected maps</h3>
        <div className="flex gap-3 overflow-x-auto">
          <Skeleton className="w-48 h-20 rounded-lg flex-shrink-0" />
          <Skeleton className="w-48 h-20 rounded-lg flex-shrink-0" />
          <Skeleton className="w-48 h-20 rounded-lg flex-shrink-0" />
        </div>
      </section>
    );
  }

  const seen = new Set<number>();
  const targets: number[] = [];
  for (const p of portals) {
    const tm = p.attributes.targetMapId;
    if (!tm || tm === NONE_MAP_ID) continue;
    if (String(tm) === mapId) continue;
    if (seen.has(tm)) continue;
    seen.add(tm);
    targets.push(tm);
  }

  if (targets.length === 0) {
    return null;
  }

  return (
    <section>
      <h3 className="text-sm font-semibold mb-2">Connected maps ({targets.length})</h3>
      <div className="flex gap-3 overflow-x-auto pb-2">
        {targets.map((target) => (
          <Link
            key={target}
            to={`/maps/${target}`}
            className="flex-shrink-0"
          >
            <Card className="w-48 h-20 p-3 flex items-center justify-center hover:bg-muted/50 transition-colors">
              <MapCell mapId={String(target)} tenant={activeTenant} />
            </Card>
          </Link>
        ))}
      </div>
    </section>
  );
}
