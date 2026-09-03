import { Link, useParams, useSearchParams } from "react-router-dom";
import { RefreshCw } from "lucide-react";

import { Button } from "@/components/ui/button";
import { PageLoader } from "@/components/common/PageLoader";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";
import { HoverHighlightProvider } from "@/components/features/maps/HoverHighlightContext";
import { MapImagePanel } from "@/components/features/maps/MapImagePanel";
import { FieldHeader } from "@/components/features/fields/FieldHeader";
import { FieldSummaryPanels } from "@/components/features/fields/FieldSummaryPanels";
import { FieldTabs } from "@/components/features/fields/FieldTabs";
import { FieldCharactersTab } from "@/components/features/fields/FieldCharactersTab";
import { FieldMonstersTab } from "@/components/features/fields/FieldMonstersTab";
import { FieldObjectsTab } from "@/components/features/fields/FieldObjectsTab";
import { useMap } from "@/lib/hooks/api/useMaps";
import { useWorlds } from "@/lib/hooks/api/useWorlds";
import {
  useFieldCharacters,
  useLiveMonsters,
} from "@/lib/hooks/api/useFieldRuntime";
import {
  useMapNpcs,
  useMapObjects,
  useMapPortals,
  useMapReactors,
} from "@/lib/hooks/api/useMapEntities";
import type { PositionedMonster } from "@/services/api/map-entities.service";
import type { LiveMonsterData } from "@/services/api/live-monsters.service";
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
import { cn } from "@/lib/utils";

// FR-19: the field overview reuses MapImagePanel/MapImageOverlay, whose
// monster pins only need id/template/x/y (see MapImageOverlay's
// MonsterMarker + computeMarkers). Live monsters carry their template at
// `attributes.monsterId`, not `attributes.template` — adapt without
// fabricating any of MapMonsterData's other declared-spawn-only fields.
function toPositionedMonsters(
  monsters: LiveMonsterData[] | undefined,
): PositionedMonster[] {
  if (!monsters) return [];
  return monsters.map((m) => ({
    id: m.id,
    attributes: {
      template: m.attributes.monsterId,
      x: m.attributes.x,
      y: m.attributes.y,
    },
  }));
}

const DEFAULT_TAB = "characters";

// FR-18..FR-22, FR-40: the field-detail page. There is no
// `GET /api/fields/{id}` — liveness is "holds at least one character", so an
// empty (resolved) result from `useFieldCharacters` is the torn-down signal
// (FR-22), not an error.
export function FieldDetailPage() {
  const {
    worldId: worldIdParam,
    channelId: channelIdParam,
    mapId,
    instanceId,
  } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();

  const worldId = Number(worldIdParam);
  const channelId = Number(channelIdParam);
  const numericMapId = Number(mapId);
  const paramsValid =
    !!mapId &&
    !!instanceId &&
    !Number.isNaN(worldId) &&
    !Number.isNaN(channelId) &&
    !Number.isNaN(numericMapId);

  const mapQuery = useMap(mapId ?? "");
  const worldsQuery = useWorlds();
  const charactersQuery = useFieldCharacters(
    worldId,
    channelId,
    numericMapId,
    instanceId ?? "",
    paramsValid,
  );
  const monstersQuery = useLiveMonsters(
    worldId,
    channelId,
    numericMapId,
    instanceId ?? "",
    paramsValid,
  );
  const objectsQuery = useMapObjects(mapId ?? "");
  const portalsQuery = useMapPortals(mapId ?? "");
  const npcsQuery = useMapNpcs(mapId ?? "");
  const reactorsQuery = useMapReactors(mapId ?? "");

  const { isRefreshing, onRefresh, lastUpdatedAt } = useGridRefresh([
    mapQuery,
    charactersQuery,
    monstersQuery,
    objectsQuery,
    portalsQuery,
    npcsQuery,
    reactorsQuery,
  ]);

  const tab = searchParams.get("tab") ?? DEFAULT_TAB;
  const handleTabChange = (next: string) => {
    const nextParams = new URLSearchParams(searchParams);
    nextParams.set("tab", next);
    setSearchParams(nextParams, { replace: true });
  };

  if (!paramsValid) {
    return (
      <div className="p-10">
        <ErrorDisplay error="This field's route is invalid." />
      </div>
    );
  }

  if (
    mapQuery.isLoading ||
    worldsQuery.isLoading ||
    charactersQuery.isLoading
  ) {
    return <PageLoader />;
  }

  if (mapQuery.error || !mapQuery.data) {
    return (
      <div className="p-10">
        <ErrorDisplay
          error={mapQuery.error ?? "Map not found"}
          retry={() => mapQuery.refetch()}
        />
      </div>
    );
  }

  if (charactersQuery.error) {
    return (
      <div className="p-10">
        <ErrorDisplay
          error={charactersQuery.error}
          retry={() => charactersQuery.refetch()}
        />
      </div>
    );
  }

  const characterIds = charactersQuery.data?.map((c) => c.id) ?? [];
  const characterCount = characterIds.length;

  // FR-22: the characters query resolved with no rows — the torn-down
  // signal, not an error. A genuinely live field cannot be empty.
  if (characterCount === 0) {
    return (
      <div className="flex flex-col flex-1 min-h-0 overflow-y-auto space-y-6 p-10 pb-16">
        <div className="flex flex-col items-center justify-center gap-3 p-16 text-center">
          <h2 className="text-lg font-semibold">
            This field may have been torn down
          </h2>
          <p className="text-sm text-muted-foreground max-w-sm">
            No characters remain in this field, so it may have already been torn
            down.
          </p>
          <Link to="/fields" className="text-sm underline">
            Back to Fields
          </Link>
        </div>
      </div>
    );
  }

  const attrs = mapQuery.data.attributes;
  const worldName = worldsQuery.data?.find(
    (world) => world.id === String(worldId),
  )?.attributes.name;

  return (
    <div className="flex flex-col flex-1 min-h-0 overflow-y-auto space-y-6 p-10 pb-16">
      <div className="flex items-start justify-between gap-4">
        <FieldHeader
          worldId={worldIdParam ?? ""}
          channelId={channelIdParam ?? ""}
          mapId={mapId ?? ""}
          instanceId={instanceId ?? ""}
          mapName={attrs.name}
          worldName={worldName}
        />

        <div className="flex items-center gap-3">
          <Button
            variant="outline"
            size="sm"
            onClick={() => void onRefresh()}
            disabled={isRefreshing}
            aria-busy={isRefreshing}
          >
            <RefreshCw
              className={cn("h-4 w-4 mr-1.5", isRefreshing && "animate-spin")}
            />
            Refresh
          </Button>
          {lastUpdatedAt != null && lastUpdatedAt > 0 && (
            <span
              className="text-xs text-muted-foreground"
              title={new Date(lastUpdatedAt).toISOString()}
            >
              Last updated{" "}
              {new Date(lastUpdatedAt).toLocaleTimeString(undefined, {
                hour: "2-digit",
                minute: "2-digit",
              })}
            </span>
          )}
        </div>
      </div>

      <HoverHighlightProvider>
        <div className="grid gap-4 md:grid-cols-[2fr_1fr]">
          <MapImagePanel
            mapId={mapId ?? ""}
            mapName={attrs.name}
            mapArea={attrs.mapArea ?? null}
            portals={portalsQuery.data}
            npcs={npcsQuery.data}
            monsters={toPositionedMonsters(monstersQuery.data)}
            reactors={reactorsQuery.data}
          />
          <FieldSummaryPanels
            characterCount={characterCount}
            liveMonsters={monstersQuery.data}
          />
        </div>

        <FieldTabs
          characterCount={characterCount}
          monsterCount={monstersQuery.data?.length ?? 0}
          objectCount={objectsQuery.data?.length ?? 0}
          tab={tab}
          onTabChange={handleTabChange}
          characters={<FieldCharactersTab characterIds={characterIds} />}
          monsters={
            <FieldMonstersTab
              monsters={monstersQuery.data}
              error={monstersQuery.error ?? undefined}
              mapId={numericMapId}
            />
          }
          objects={
            <FieldObjectsTab
              defined={objectsQuery.data}
              definedError={objectsQuery.error ?? undefined}
            />
          }
        />
      </HoverHighlightProvider>
    </div>
  );
}
