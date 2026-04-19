import { Link } from "react-router-dom";
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
        <div className="flex flex-wrap gap-1.5">
          <Skeleton className="h-6 w-20 rounded-full" />
          <Skeleton className="h-6 w-24 rounded-full" />
          <Skeleton className="h-6 w-20 rounded-full" />
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
      <div className="flex flex-wrap gap-1.5">
        {targets.map((target) => (
          <Link
            key={target}
            to={`/maps/${target}`}
            className="hover:opacity-80 transition-opacity"
          >
            <MapCell mapId={String(target)} tenant={activeTenant} />
          </Link>
        ))}
      </div>
    </section>
  );
}
