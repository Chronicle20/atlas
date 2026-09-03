import { Badge } from "@/components/ui/badge";
import { SurfaceKindBadge } from "@/components/features/maps/SurfaceKindBadge";

interface FieldHeaderProps {
  worldId: string;
  channelId: string;
  instanceId: string;
  mapName: string;
  worldName?: string | undefined;
}

/**
 * FR-18: the map name is the primary title, on its own line. The runtime
 * badge and the world/channel/instance identifiers render as badges on the
 * row below it (bug-fields-ui item 7) — never a raw id where a resolved
 * name is available, and channel is shown 1-indexed (display only; the
 * value used in links, queries, and the API stays 0-based).
 */
export function FieldHeader({
  worldId,
  channelId,
  instanceId,
  mapName,
  worldName,
}: FieldHeaderProps) {
  const channelLabel = String(Number(channelId) + 1);

  return (
    <div className="flex flex-col gap-2">
      <h1 className="text-2xl font-semibold">{mapName}</h1>
      <div className="flex flex-wrap items-center gap-2">
        <SurfaceKindBadge kind="runtime" />
        <Badge variant="outline">World {worldName ?? worldId}</Badge>
        <Badge variant="outline">Channel {channelLabel}</Badge>
        <Badge variant="outline" className="font-mono text-xs">
          Instance {instanceId}
        </Badge>
      </div>
    </div>
  );
}
