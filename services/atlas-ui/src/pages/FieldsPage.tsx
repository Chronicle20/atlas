import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { RefreshCw } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { EmptyState } from "@/components/common/EmptyState";
import { SurfaceKindBadge } from "@/components/features/maps/SurfaceKindBadge";
import { FieldsFilterBar } from "@/components/features/fields/FieldsFilterBar";
import { FieldsResultTable } from "@/components/features/fields/FieldsResultTable";
import { FieldDetailPage } from "@/pages/FieldDetailPage";
import { useFields } from "@/lib/hooks/api/useFields";
import { useWorlds, useChannels } from "@/lib/hooks/api/useWorlds";
import { useMapNames } from "@/lib/hooks/api/useMaps";
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
import { useDebounce } from "@/lib/utils/debounce";
import { cn } from "@/lib/utils";

const DEBOUNCE_MS = 250;

// FR-11..FR-16/FR-40: the runtime read model locator. World/channel filter
// the server query; the map filter is client-side (FR-13 needs a name-or-id
// *search*, and the server side only supports an exact mapId match).
//
// bug-fields-ui item 6: there is no separate field-detail route — an
// `?instance=` query param on this same `/fields` path switches the page to
// the detail view. `FieldDetailPage` reads world/channel/map/instance from
// the query string itself, so it needs no props here.
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
  // `mapInput` is the single owner of the raw map-filter text; it is passed
  // to `FieldsFilterBar` as a controlled value (no local copy there). The
  // debounced value drives both filtering and the `?map=` URL param, so
  // Clear filters just resets this one piece of state and everything else
  // follows without a remount.
  const [mapInput, setMapInput] = useState(() => searchParams.get("map") ?? "");
  const debouncedMapQuery = useDebounce(mapInput.trim(), DEBOUNCE_MS);

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

  const trimmedMapQuery = debouncedMapQuery.toLowerCase();
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

  // Mirrors MapsPage's `searchInput`/`debounced` pair: the debounced value
  // is written back to `?map=` once it settles, rather than on every
  // keystroke.
  useEffect(() => {
    const currentMapParam = searchParams.get("map") ?? "";
    if (debouncedMapQuery === currentMapParam) return;
    const next = new URLSearchParams(searchParams);
    if (debouncedMapQuery) {
      next.set("map", debouncedMapQuery);
    } else {
      next.delete("map");
    }
    setSearchParams(next, { replace: true });
    // eslint-disable-next-line react-hooks/exhaustive-deps -- fires only on debounce settling
  }, [debouncedMapQuery]);

  const handleWorldChange = (nextWorldId: number) => {
    setWorldOverride(nextWorldId);
    setChannelId(null);
  };

  const handleClearFilters = () => {
    setWorldOverride(null);
    setChannelId(null);
    setMapInput("");
    const next = new URLSearchParams(searchParams);
    next.delete("map");
    setSearchParams(next, { replace: true });
  };

  const worldName =
    worlds.find((world) => Number(world.id) === worldId)?.attributes.name ??
    String(worldId);
  const channelLabel = channelId === null ? "Any channel" : String(channelId);

  // bug-fields-ui item 6: an `?instance=` param switches this route to the
  // field-detail view — there is no second path.
  if (searchParams.get("instance") !== null) {
    return <FieldDetailPage />;
  }

  return (
    <div className="flex flex-col flex-1 min-h-0 overflow-y-auto space-y-6 p-10 pb-16">
      <div className="flex items-center justify-between gap-4">
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-semibold">Fields</h1>
          <SurfaceKindBadge kind="runtime" />
        </div>

        <div className="flex items-center gap-3">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="outline"
                size="icon"
                onClick={() => void onRefresh()}
                disabled={isRefreshing}
                aria-busy={isRefreshing}
                aria-label="Refresh"
              >
                <RefreshCw
                  className={cn("h-4 w-4", isRefreshing && "animate-spin")}
                />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Refresh</TooltipContent>
          </Tooltip>
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
        worlds={worlds}
        channels={channels}
        worldId={worldId}
        channelId={channelId}
        mapQuery={mapInput}
        resultCount={filteredFields.length}
        onWorldChange={handleWorldChange}
        onChannelChange={setChannelId}
        onMapQueryChange={setMapInput}
        onClear={handleClearFilters}
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
