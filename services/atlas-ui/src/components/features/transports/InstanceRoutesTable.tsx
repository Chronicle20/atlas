import { useMemo, useState } from "react";
import { AlertTriangle, ChevronDown, ChevronRight } from "lucide-react";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Badge } from "@/components/ui/badge";
import { MapCell } from "@/components/map-cell";
import { Countdown } from "@/components/features/transports/Countdown";
import {
  formatDurationSeconds,
  isInstanceStuck,
} from "@/components/features/transports/transport-format";
import { useClock } from "@/lib/utils/clock";
import {
  useInstanceRoutes,
  useInstanceStatuses,
} from "@/lib/hooks/api/useTransports";
import type { Tenant } from "@/types/models/tenant";
import type { InstanceRoute, InstanceStatus } from "@/types/models/transport";

interface InstanceRoutesTableProps {
  tenant: Tenant | null;
}

/** Column count of the header row, for the loading/error/empty full-width cells. */
const INSTANCE_TABLE_COLUMN_COUNT = 8;

/**
 * The Instance tab's table. It does not report its row count upward — Radix
 * unmounts inactive tab panels, so a count published from here would stay at
 * zero until the tab was first opened. TransportsPage reads the same query
 * directly for its tab label.
 */
export function InstanceRoutesTable({ tenant }: InstanceRoutesTableProps) {
  const routesQuery = useInstanceRoutes();
  const routes = useMemo(() => routesQuery.data ?? [], [routesQuery.data]);
  const routeIds = useMemo(() => routes.map((route) => route.id), [routes]);

  const statusQueries = useInstanceStatuses(routeIds);

  const statusesByRouteId = useMemo(() => {
    const map = new Map<string, InstanceStatus[]>();
    routeIds.forEach((routeId, index) => {
      map.set(routeId, statusQueries[index]?.data ?? []);
    });
    return map;
  }, [routeIds, statusQueries]);

  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const toggle = (routeId: string) => {
    setExpanded((previous) => {
      const next = new Set(previous);
      if (next.has(routeId)) {
        next.delete(routeId);
      } else {
        next.add(routeId);
      }
      return next;
    });
  };

  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-10" />
            <TableHead>Route</TableHead>
            <TableHead>Live</TableHead>
            <TableHead>Capacity</TableHead>
            <TableHead>Boarding window</TableHead>
            <TableHead>Travel</TableHead>
            <TableHead>Start map</TableHead>
            <TableHead>Destination map</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {routesQuery.isLoading ? (
            <TableRow>
              <TableCell
                colSpan={INSTANCE_TABLE_COLUMN_COUNT}
                className="text-center text-muted-foreground"
              >
                Loading instance routes…
              </TableCell>
            </TableRow>
          ) : routesQuery.isError ? (
            <TableRow>
              <TableCell
                colSpan={INSTANCE_TABLE_COLUMN_COUNT}
                className="text-center text-destructive"
              >
                <span className="inline-flex items-center justify-center gap-1.5">
                  <AlertTriangle className="h-4 w-4" aria-hidden="true" />
                  Failed to load instance routes.
                </span>
              </TableCell>
            </TableRow>
          ) : routes.length === 0 ? (
            <TableRow>
              <TableCell
                colSpan={INSTANCE_TABLE_COLUMN_COUNT}
                className="text-center text-muted-foreground"
              >
                No instance routes configured.
              </TableCell>
            </TableRow>
          ) : (
            routes.map((route) => {
              const statuses = statusesByRouteId.get(route.id) ?? [];
              const isExpanded = expanded.has(route.id);
              return (
                <InstanceRouteRows
                  key={route.id}
                  route={route}
                  statuses={statuses}
                  tenant={tenant}
                  isExpanded={isExpanded}
                  onToggle={() => toggle(route.id)}
                />
              );
            })
          )}
        </TableBody>
      </Table>
    </div>
  );
}

function InstanceRouteRows({
  route,
  statuses,
  tenant,
  isExpanded,
  onToggle,
}: {
  route: InstanceRoute;
  statuses: InstanceStatus[];
  tenant: Tenant | null;
  isExpanded: boolean;
  onToggle: () => void;
}) {
  const expandable = statuses.length > 0;

  return (
    <>
      <TableRow>
        <TableCell>
          {expandable ? (
            <button
              type="button"
              onClick={onToggle}
              aria-label={
                isExpanded
                  ? `Collapse ${route.attributes.name}`
                  : `Expand ${route.attributes.name}`
              }
              aria-expanded={isExpanded}
            >
              {isExpanded ? (
                <ChevronDown className="h-4 w-4" />
              ) : (
                <ChevronRight className="h-4 w-4" />
              )}
            </button>
          ) : null}
        </TableCell>
        <TableCell className="font-medium">{route.attributes.name}</TableCell>
        {/* Zero live instances is the steady state, not an error. */}
        <TableCell className="tabular-nums">{statuses.length}</TableCell>
        <TableCell className="tabular-nums">
          {route.attributes.capacity}
        </TableCell>
        <TableCell>
          {formatDurationSeconds(route.attributes.boardingWindowSeconds)}
        </TableCell>
        <TableCell>
          {formatDurationSeconds(route.attributes.travelDurationSeconds)}
        </TableCell>
        <TableCell>
          <MapCell
            mapId={String(route.attributes.startMapId)}
            tenant={tenant}
          />
        </TableCell>
        <TableCell>
          <MapCell
            mapId={String(route.attributes.destinationMapId)}
            tenant={tenant}
          />
        </TableCell>
      </TableRow>

      {isExpanded
        ? statuses.map((status) => (
            <LiveInstanceRow key={status.id} status={status} route={route} />
          ))
        : null}
    </>
  );
}

function LiveInstanceRow({
  status,
  route,
}: {
  status: InstanceStatus;
  route: InstanceRoute;
}) {
  const now = useClock();
  const stuck = isInstanceStuck(
    status.attributes.createdAt,
    route.attributes.boardingWindowSeconds,
    route.attributes.travelDurationSeconds,
    now,
  );

  const boarding = status.attributes.state === "boarding";

  return (
    <TableRow className="bg-muted/40">
      <TableCell />
      <TableCell colSpan={2}>
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="font-mono text-xs">
                {status.id.slice(0, 8)}…
              </span>
            </TooltipTrigger>
            <TooltipContent copyable>
              <p>{status.id}</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </TableCell>
      <TableCell>
        <Badge variant={boarding ? "outline" : "default"}>
          {boarding ? "Boarding" : "In transit"}
        </Badge>
      </TableCell>
      <TableCell className="tabular-nums">
        {status.attributes.characters}
      </TableCell>
      <TableCell colSpan={2}>
        <Countdown
          targetAt={
            boarding
              ? status.attributes.boardingUntil
              : status.attributes.arrivalAt
          }
          label={boarding ? "closes in" : "arrives in"}
        />
      </TableCell>
      <TableCell>
        {stuck ? (
          <span className="inline-flex items-center gap-1.5 text-destructive text-xs">
            <AlertTriangle className="h-3.5 w-3.5" aria-hidden="true" />
            Approaching stuck timeout
          </span>
        ) : null}
      </TableCell>
    </TableRow>
  );
}
