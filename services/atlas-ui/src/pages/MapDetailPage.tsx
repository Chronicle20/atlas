import { useParams } from "react-router-dom";
import { useMap } from "@/lib/hooks/api/useMaps";
import {
  useMapMonsters,
  useMapNpcs,
  useMapPortals,
  useMapReactors,
} from "@/lib/hooks/api/useMapEntities";
import { PageLoader } from "@/components/common/PageLoader";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";
import { MapHeader } from "@/components/features/maps/MapHeader";
import { MapImagePanel } from "@/components/features/maps/MapImagePanel";
import { MapEntitySummary } from "@/components/features/maps/MapEntitySummary";
import { ConnectedMapsRow } from "@/components/features/maps/ConnectedMapsRow";
import { MapDetailTabs } from "@/components/features/maps/MapDetailTabs";
import { HoverHighlightProvider } from "@/components/features/maps/HoverHighlightContext";

export function MapDetailPage() {
  const params = useParams();
  const id = params.id as string;

  const { data: map, isLoading, error, refetch } = useMap(id);
  const { data: portals, error: portalsError } = useMapPortals(id);
  const { data: npcs, error: npcsError } = useMapNpcs(id);
  const { data: monsters, error: monstersError } = useMapMonsters(id);
  const { data: reactors, error: reactorsError } = useMapReactors(id);

  if (isLoading) {
    return <PageLoader />;
  }

  if (error || !map) {
    return (
      <div className="p-10">
        <ErrorDisplay error={error ?? "Map not found"} retry={() => refetch()} />
      </div>
    );
  }

  const attrs = map.attributes;
  const spawnCount = monstersError ? undefined : monsters?.length;

  return (
    <div className="flex flex-col flex-1 min-h-0 overflow-y-auto space-y-6 p-10 pb-16">
      <MapHeader
        mapId={map.id}
        name={attrs.name}
        streetName={attrs.streetName}
        spawnCount={spawnCount}
      />

      <HoverHighlightProvider>
        <div className="grid gap-4 md:grid-cols-[2fr_1fr]">
          <MapImagePanel
            mapId={map.id}
            mapName={attrs.name}
            initialKind="render"
            mapArea={attrs.mapArea ?? null}
            portals={portals}
            npcs={npcs}
            monsters={monsters}
            reactors={reactors}
          />
          <MapEntitySummary
            npcs={npcs}
            npcsError={npcsError}
            monsters={monsters}
            monstersError={monstersError}
          />
        </div>

        <ConnectedMapsRow mapId={id} portals={portals} />

        <MapDetailTabs
          mapId={id}
          portals={portals}
          portalsError={portalsError}
          monsters={monsters}
          monstersError={monstersError}
          reactors={reactors}
          reactorsError={reactorsError}
        />
      </HoverHighlightProvider>
    </div>
  );
}
