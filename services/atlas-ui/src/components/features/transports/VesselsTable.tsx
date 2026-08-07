import { AlertTriangle } from "lucide-react";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { RouteStatePill } from "@/components/features/transports/RouteStatePill";
import {
  formatDurationSeconds,
  resolveVesselRoutes,
} from "@/components/features/transports/transport-format";
import type { ScheduledRoute, Vessel } from "@/types/models/transport";

interface VesselsTableProps {
  vessels: Vessel[];
  routes: ScheduledRoute[];
  /**
   * Combined loading/error state of the vessels + scheduled-routes queries.
   * Both feed this table (route resolution needs both), so the page — which
   * already owns both queries for the Scheduled tab — passes their combined
   * status down rather than this component fetching on its own. Defaulted
   * to `false` so the table renders straight from props in isolation (e.g.
   * in tests) exactly as the brief's interface describes.
   */
  isLoading?: boolean;
  isError?: boolean;
}

/** Column count of the header row, for the loading/error/empty full-width cells. */
const VESSELS_TABLE_COLUMN_COUNT = 4;

/**
 * Six vessels over twelve routes: the tab exists because the unpaired-vessel
 * fault belongs to the *vessel*, not to either of its routes, and it makes the
 * alternation legible in one glance.
 */
export function VesselsTable({
  vessels,
  routes,
  isLoading = false,
  isError = false,
}: VesselsTableProps) {
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Vessel</TableHead>
            <TableHead>Route A</TableHead>
            <TableHead>Route B</TableHead>
            <TableHead>Turnaround</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableRow>
              <TableCell
                colSpan={VESSELS_TABLE_COLUMN_COUNT}
                className="text-center text-muted-foreground"
              >
                Loading vessels…
              </TableCell>
            </TableRow>
          ) : isError ? (
            <TableRow>
              <TableCell
                colSpan={VESSELS_TABLE_COLUMN_COUNT}
                className="text-center text-destructive"
              >
                <span className="inline-flex items-center justify-center gap-1.5">
                  <AlertTriangle className="h-4 w-4" aria-hidden="true" />
                  Failed to load vessels.
                </span>
              </TableCell>
            </TableRow>
          ) : vessels.length === 0 ? (
            <TableRow>
              <TableCell
                colSpan={VESSELS_TABLE_COLUMN_COUNT}
                className="text-center text-muted-foreground"
              >
                No vessels configured.
              </TableCell>
            </TableRow>
          ) : (
            vessels.map((vessel) => {
              const { routeA, routeB, unresolved } = resolveVesselRoutes(
                vessel,
                routes,
              );
              return (
                <TableRow key={vessel.id} id={`vessel-${vessel.id}`}>
                  <TableCell className="font-medium">
                    {vessel.attributes.name}
                    {unresolved ? (
                      <div className="mt-1 inline-flex items-start gap-1.5 text-xs text-destructive">
                        <AlertTriangle
                          className="mt-0.5 h-3.5 w-3.5 shrink-0"
                          aria-hidden="true"
                        />
                        <span>
                          Unresolved route reference — both of this
                          vessel&apos;s routes will be out of service until it
                          is fixed.
                        </span>
                      </div>
                    ) : null}
                  </TableCell>
                  <VesselRouteCell
                    name={vessel.attributes.routeAID}
                    route={routeA}
                  />
                  <VesselRouteCell
                    name={vessel.attributes.routeBID}
                    route={routeB}
                  />
                  <TableCell>
                    {formatDurationSeconds(vessel.attributes.turnaroundDelay)}
                  </TableCell>
                </TableRow>
              );
            })
          )}
        </TableBody>
      </Table>
    </div>
  );
}

/**
 * Routes are matched by **name**, which is the rule the backend scheduler
 * uses. `name` here is the vessel's raw reference, shown even when it resolves
 * to nothing so the operator can see what the bad value is.
 */
function VesselRouteCell({
  name,
  route,
}: {
  name: string;
  route: ScheduledRoute | null;
}) {
  if (!route) {
    return (
      <TableCell>
        <span className="text-destructive">{name || "—"}</span>
        <span className="ml-1 text-xs text-muted-foreground">(no match)</span>
      </TableCell>
    );
  }
  return (
    <TableCell>
      <div className="flex items-center gap-2">
        <span>{route.attributes.name}</span>
        <RouteStatePill state={route.attributes.state} />
      </div>
    </TableCell>
  );
}
