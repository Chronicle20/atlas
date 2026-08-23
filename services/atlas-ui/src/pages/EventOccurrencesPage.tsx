/**
 * Event occurrence history: every live/completed/cancelled/failed run of an
 * event definition (FR-UI5), filterable by type, state, world, channel and a
 * start-date range (FR-UI6).
 *
 * Backed by atlas-events (task-231 Tasks 13/16, event/occurrence/resource.go):
 *   GET /api/events/occurrences
 *
 * There is no backend "active vs historical" filter — `filter[state]` IS the
 * active/historical distinction (an ACTIVE state is "active"; COMPLETED,
 * CANCELLED and FAILED are "historical"). Do not invent a second parameter
 * for it.
 */

import { useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { RefreshCw } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { useTenant } from "@/context/tenant-context";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { PageLoader } from "@/components/common/PageLoader";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";
import { EmptyState } from "@/components/common/EmptyState";
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
import { eventsService } from "@/services/api/events.service";
import type { EventOccurrenceFilters } from "@/types/models/events";
import { ACTIVE_STATE, occurrenceColumns } from "./event-occurrences-columns";
import { cn } from "@/lib/utils";

/** Mirrors event/occurrence.model.go's four state constants. */
const OCCURRENCE_STATES = [
  "ACTIVE",
  "COMPLETED",
  "CANCELLED",
  "FAILED",
] as const;

interface FilterFormState {
  type: string;
  state: string;
  worldId: string;
  channelId: string;
  startedAtFrom: string;
  startedAtTo: string;
}

const EMPTY_FILTERS: FilterFormState = {
  type: "",
  state: "",
  worldId: "",
  channelId: "",
  startedAtFrom: "",
  startedAtTo: "",
};

function toEventOccurrenceFilters(
  form: FilterFormState,
  definitionId: string | null,
): EventOccurrenceFilters {
  const filters: EventOccurrenceFilters = {};
  if (definitionId) filters.definitionId = definitionId;
  if (form.type) filters.type = form.type;
  if (form.state) filters.state = form.state;
  if (form.worldId) filters.worldId = Number(form.worldId);
  if (form.channelId) filters.channelId = Number(form.channelId);
  if (form.startedAtFrom || form.startedAtTo) {
    filters.startedAt = {
      ...(form.startedAtFrom ? { from: form.startedAtFrom } : {}),
      ...(form.startedAtTo ? { to: form.startedAtTo } : {}),
    };
  }
  return filters;
}

export function EventOccurrencesPage() {
  const { activeTenant } = useTenant();
  const [searchParams] = useSearchParams();
  const definitionId = searchParams.get("definitionId");
  const [form, setForm] = useState<FilterFormState>(EMPTY_FILTERS);

  const filters = useMemo(
    () => toEventOccurrenceFilters(form, definitionId),
    [form, definitionId],
  );

  const occurrencesQuery = useQuery({
    queryKey: [
      "events",
      "occurrences",
      activeTenant?.id ?? "no-tenant",
      filters,
    ] as const,
    queryFn: () => eventsService.getOccurrences(filters),
    enabled: !!activeTenant,
  });

  const occurrences = occurrencesQuery.data?.data ?? [];

  const { isRefreshing, onRefresh, lastUpdatedAt } = useGridRefresh([
    occurrencesQuery,
  ]);

  const updateForm = (patch: Partial<FilterFormState>) =>
    setForm((prev) => ({ ...prev, ...patch }));

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold tracking-tight">Event Occurrences</h2>
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

      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
        <div className="space-y-1">
          <Label htmlFor="occurrence-type-filter">Type</Label>
          <Input
            id="occurrence-type-filter"
            value={form.type}
            onChange={(e) => updateForm({ type: e.target.value })}
            placeholder="e.g. CRIMSON_BALROG"
          />
        </div>
        <div className="space-y-1">
          <Label htmlFor="occurrence-state-filter">State</Label>
          <select
            id="occurrence-state-filter"
            className="border-input flex h-9 w-full rounded-md border bg-transparent px-3 py-1 text-sm shadow-xs"
            value={form.state}
            onChange={(e) => updateForm({ state: e.target.value })}
          >
            <option value="">All</option>
            {OCCURRENCE_STATES.map((state) => (
              <option key={state} value={state}>
                {state}
              </option>
            ))}
          </select>
        </div>
        <div className="space-y-1">
          <Label htmlFor="occurrence-world-filter">World</Label>
          <Input
            id="occurrence-world-filter"
            type="number"
            value={form.worldId}
            onChange={(e) => updateForm({ worldId: e.target.value })}
          />
        </div>
        <div className="space-y-1">
          <Label htmlFor="occurrence-channel-filter">Channel</Label>
          <Input
            id="occurrence-channel-filter"
            type="number"
            value={form.channelId}
            onChange={(e) => updateForm({ channelId: e.target.value })}
          />
        </div>
        <div className="space-y-1">
          <Label htmlFor="occurrence-started-from-filter">Started from</Label>
          <Input
            id="occurrence-started-from-filter"
            type="date"
            value={form.startedAtFrom}
            onChange={(e) => updateForm({ startedAtFrom: e.target.value })}
          />
        </div>
        <div className="space-y-1">
          <Label htmlFor="occurrence-started-to-filter">Started to</Label>
          <Input
            id="occurrence-started-to-filter"
            type="date"
            value={form.startedAtTo}
            onChange={(e) => updateForm({ startedAtTo: e.target.value })}
          />
        </div>
      </div>

      {occurrencesQuery.isLoading ? (
        <PageLoader />
      ) : occurrencesQuery.error ? (
        <ErrorDisplay
          error={occurrencesQuery.error}
          retry={() => void occurrencesQuery.refetch()}
        />
      ) : occurrences.length === 0 ? (
        <EmptyState
          title="No event occurrences found"
          description="No occurrences match the current filters."
          onRefresh={onRefresh}
          isRefreshing={isRefreshing}
          lastUpdatedAt={lastUpdatedAt}
        />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                {occurrenceColumns.map((column) => (
                  <TableHead key={column.header}>{column.header}</TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {occurrences.map((occurrence) => {
                const isActive = occurrence.attributes.state === ACTIVE_STATE;
                return (
                  <TableRow
                    key={occurrence.id}
                    data-testid={`occurrence-${occurrence.id}`}
                    className={cn(
                      isActive
                        ? "bg-primary/5 font-medium"
                        : "text-muted-foreground",
                    )}
                  >
                    {occurrenceColumns.map((column) => (
                      <TableCell key={column.header}>
                        {column.cell(occurrence)}
                      </TableCell>
                    ))}
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
