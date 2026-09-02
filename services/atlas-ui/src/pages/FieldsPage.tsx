import { useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { RefreshCw } from "lucide-react";

import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/common/EmptyState";
import { SurfaceKindBadge } from "@/components/features/maps/SurfaceKindBadge";
import { FieldsFilterBar } from "@/components/features/fields/FieldsFilterBar";
import { FieldsResultTable } from "@/components/features/fields/FieldsResultTable";
import { useFields } from "@/lib/hooks/api/useFields";
import { useWorlds, useChannels } from "@/lib/hooks/api/useWorlds";
import { useMapNames } from "@/lib/hooks/api/useMaps";
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
import { cn } from "@/lib/utils";

// FR-11..FR-16/FR-40: the runtime read model locator. World/channel filter
// the server query; the map filter is client-side (FR-13 needs a name-or-id
// *search*, and the server side only supports an exact mapId match).
export function FieldsPage() {
  const [searchParams, setSearchParams] = useSearchParams();

  const worldsQuery = useWorlds();
  const worlds = useMemo(() => worldsQuery.data ?? [], [worldsQuery.data]);

  const defaultWorldId = useMemo(() => {
    if (worlds.length === 0) return 0;
    return Math.min(...worlds.map((world) => Number(world.id)));
  }, [worlds]);

  const [worldOverride, setWorldOverride] = useState<number | null>(null);
  const [channelId, setChannelId] = useState<number | null>(null);
  const [mapQuery, setMapQuery] = useState(() => searchParams.get("map") ?? "");
  // Bumped on Clear filters so `FieldsFilterBar` remounts instead of syncing
  // its local input off a prop change — a remount discards any in-flight
  // debounce timer outright, so a keystroke typed just before Clear can't
  // race the reset and repopulate the field a moment later.
  const [filterResetKey, setFilterResetKey] = useState(0);

  const worldId = worldOverride ?? defaultWorldId;

  const channelsQuery = useChannels(worldId);
  const channels = useMemo(
    () => channelsQuery.data ?? [],
    [channelsQuery.data],
  );

  const fieldsQuery = useFields({
    worldId,
    ...(channelId !== null ? { channelId } : {}),
  });
  const fields = useMemo(() => fieldsQuery.data ?? [], [fieldsQuery.data]);

  const distinctMapIds = useMemo(
    () => Array.from(new Set(fields.map((field) => field.attributes.mapId))),
    [fields],
  );
  const mapNames = useMapNames(distinctMapIds);

  const trimmedMapQuery = mapQuery.trim().toLowerCase();
  const filteredFields = useMemo(() => {
    if (!trimmedMapQuery) return fields;
    return fields.filter((field) => {
      const { mapId } = field.attributes;
      const idMatches = String(mapId).includes(trimmedMapQuery);
      const name = mapNames[mapId]?.toLowerCase() ?? "";
      return idMatches || name.includes(trimmedMapQuery);
    });
  }, [fields, trimmedMapQuery, mapNames]);

  const { isRefreshing, onRefresh, lastUpdatedAt } = useGridRefresh([
    fieldsQuery,
  ]);

  const handleWorldChange = (nextWorldId: number) => {
    setWorldOverride(nextWorldId);
    setChannelId(null);
  };

  const handleMapQueryChange = (value: string) => {
    setMapQuery(value);
    const next = new URLSearchParams(searchParams);
    if (value) {
      next.set("map", value);
    } else {
      next.delete("map");
    }
    setSearchParams(next, { replace: true });
  };

  const handleClearFilters = () => {
    setWorldOverride(null);
    setChannelId(null);
    setMapQuery("");
    setFilterResetKey((key) => key + 1);
    const next = new URLSearchParams(searchParams);
    next.delete("map");
    setSearchParams(next, { replace: true });
  };

  const worldName =
    worlds.find((world) => Number(world.id) === worldId)?.attributes.name ??
    String(worldId);
  const channelLabel = channelId === null ? "Any channel" : String(channelId);

  return (
    <div className="flex flex-col flex-1 min-h-0 overflow-y-auto space-y-6 p-10 pb-16">
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-semibold">Fields</h1>
          <SurfaceKindBadge kind="runtime" />
        </div>

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
              data-testid="fields-last-updated"
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

      <FieldsFilterBar
        key={filterResetKey}
        worlds={worlds}
        channels={channels}
        worldId={worldId}
        channelId={channelId}
        mapQuery={mapQuery}
        onWorldChange={handleWorldChange}
        onChannelChange={setChannelId}
        onMapQueryChange={handleMapQueryChange}
      />

      {fieldsQuery.isLoading && (
        <p className="text-sm text-muted-foreground">Loading fields…</p>
      )}

      {fieldsQuery.isError && (
        <p className="text-sm text-destructive">Failed to load fields.</p>
      )}

      {!fieldsQuery.isLoading &&
        !fieldsQuery.isError &&
        filteredFields.length === 0 && (
          <EmptyState
            title="No live fields match these filters"
            description={`World: ${worldName}, Channel: ${channelLabel}`}
            action={{ label: "Clear filters", onClick: handleClearFilters }}
          />
        )}

      {!fieldsQuery.isLoading &&
        !fieldsQuery.isError &&
        filteredFields.length > 0 && (
          <FieldsResultTable fields={filteredFields} mapNames={mapNames} />
        )}
    </div>
  );
}
