/**
 * Event definitions admin surface: enable/disable toggle plus a live view of
 * how many occurrences of each definition are currently running.
 *
 * Backed by atlas-events (task-231 Tasks 13/16, event/definition/resource.go):
 *   GET   /api/events/definitions
 *   PATCH /api/events/definitions/{definitionId}
 */

import { useMemo, useState } from "react";
import { RefreshCw } from "lucide-react";
import { useQueries, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { PageLoader } from "@/components/common/PageLoader";
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
import { eventsService } from "@/services/api/events.service";
import { getColumns } from "./event-definitions-columns";
import { createErrorFromUnknown } from "@/types/api/errors";
import { cn } from "@/lib/utils";

/** Mirrors atlas-events' occurrence.StateActive (event/occurrence/model.go). */
const ACTIVE_STATE = "ACTIVE";

const DEFINITIONS_QUERY_KEY = ["events", "definitions"] as const;

export function EventDefinitionsPage() {
  const queryClient = useQueryClient();
  const [pendingToggleId, setPendingToggleId] = useState<string | null>(null);

  const definitionsQuery = useQuery({
    queryKey: DEFINITIONS_QUERY_KEY,
    queryFn: () => eventsService.getDefinitions(),
  });

  const definitions = useMemo(
    () => definitionsQuery.data?.data ?? [],
    [definitionsQuery.data],
  );

  // FR-UI4: a definition's live occurrence count comes from a per-definition
  // `filter[definitionId]` occurrence query — occurrence attributes carry no
  // definitionId of their own (event/occurrence/rest.go keeps DefinitionId
  // `json:"-"`), so there is no bulk-fetch shortcut.
  const activeCountQueries = useQueries({
    queries: definitions.map((def) => ({
      queryKey: ["events", "occurrences", "active-count", def.id],
      queryFn: () =>
        eventsService.getOccurrences({
          definitionId: def.id,
          state: ACTIVE_STATE,
        }),
    })),
  });

  const activeCounts = useMemo(() => {
    const counts: Record<string, number> = {};
    definitions.forEach((def, index) => {
      counts[def.id] = activeCountQueries[index]?.data?.data.length ?? 0;
    });
    return counts;
  }, [definitions, activeCountQueries]);

  const { isRefreshing, onRefresh } = useGridRefresh([definitionsQuery]);

  const handleToggleEnabled = async (
    id: string,
    name: string,
    next: boolean,
  ) => {
    setPendingToggleId(id);
    try {
      await eventsService.setDefinitionEnabled(id, next);
      toast.success(`${name} is now ${next ? "enabled" : "disabled"}`);
      await queryClient.invalidateQueries({ queryKey: DEFINITIONS_QUERY_KEY });
    } catch (e) {
      toast.error(createErrorFromUnknown(e).message);
    } finally {
      setPendingToggleId(null);
    }
  };

  const columns = useMemo(
    () =>
      getColumns({
        activeCounts,
        pendingToggleId,
        onToggleEnabled: (id, name, next) =>
          void handleToggleEnabled(id, name, next),
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps -- handleToggleEnabled is stable enough per render; re-deriving it isn't a dependency the columns care about.
    [activeCounts, pendingToggleId],
  );

  if (definitionsQuery.isLoading) return <PageLoader />;

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold tracking-tight">Event Definitions</h2>
        <Button
          variant="outline"
          size="icon"
          onClick={onRefresh}
          disabled={isRefreshing}
          title="Refresh"
          aria-busy={isRefreshing}
        >
          <RefreshCw
            className={cn("h-4 w-4", isRefreshing && "animate-spin")}
          />
        </Button>
      </div>

      <DataTableWrapper
        columns={columns}
        data={definitions}
        error={definitionsQuery.error?.message ?? null}
        emptyState={{
          title: "No event definitions found",
          description:
            "Event definitions are seeded by atlas-events at startup.",
        }}
      />
    </div>
  );
}
