import { Link, useParams } from "react-router-dom";
import { SurfaceKindBadge } from "@/components/features/maps/SurfaceKindBadge";

// FR-18: the field-detail header. This task renders the header from route
// params alone; Task 17 adds the resolved world/map names and the tabbed
// content underneath.
export function FieldDetailPage() {
  const { worldId, channelId, mapId, instanceId } = useParams();

  return (
    <div className="flex flex-col flex-1 min-h-0 overflow-y-auto space-y-6 p-10 pb-16">
      <div className="flex items-center gap-3">
        <h1 className="text-2xl font-semibold">Map {mapId}</h1>
        <SurfaceKindBadge kind="runtime" />
      </div>

      <dl className="grid grid-cols-2 gap-x-8 gap-y-1 text-sm max-w-md">
        <dt className="text-muted-foreground">World</dt>
        <dd>{worldId}</dd>
        <dt className="text-muted-foreground">Channel</dt>
        <dd>{channelId}</dd>
        <dt className="text-muted-foreground">Instance</dt>
        <dd className="font-mono text-xs">{instanceId}</dd>
      </dl>

      <Link to={`/maps/${mapId}`} className="text-sm underline">
        View Map Definition
      </Link>
    </div>
  );
}
