import { Link } from "react-router-dom";
import { SurfaceKindBadge } from "@/components/features/maps/SurfaceKindBadge";

interface FieldHeaderProps {
  worldId: string;
  channelId: string;
  mapId: string;
  instanceId: string;
  mapName: string;
  worldName?: string;
}

/**
 * FR-18: the map name is the primary title and world/channel/instance are
 * rendered more prominently than the map id — the map id only appears via
 * the "View Map Definition" link's href, not as visible text.
 */
export function FieldHeader({
  worldId,
  channelId,
  mapId,
  instanceId,
  mapName,
  worldName,
}: FieldHeaderProps) {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-3">
        <h1 className="text-2xl font-semibold">{mapName}</h1>
        <SurfaceKindBadge kind="runtime" />
      </div>
      <p className="text-sm font-medium">
        World {worldName ?? worldId} / Channel {channelId} / Instance{" "}
        <span className="font-mono text-xs">{instanceId}</span>
      </p>
      <Link to={`/maps/${mapId}`} className="text-sm underline w-fit">
        View Map Definition
      </Link>
    </div>
  );
}
