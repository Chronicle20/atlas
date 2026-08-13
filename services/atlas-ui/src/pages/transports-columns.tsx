import { Link } from "react-router-dom";

import { type DataTableColumnDef } from "@/components/data-table-features";

import { MapCell } from "@/components/map-cell";
import { Countdown } from "@/components/features/transports/Countdown";
import { RouteStatePill } from "@/components/features/transports/RouteStatePill";
import {
  findVesselForRoute,
  formatDurationSeconds,
  transitionLabel,
} from "@/components/features/transports/transport-format";
import type { Tenant } from "@/types/models/tenant";
import type { ScheduledRoute, Vessel } from "@/types/models/transport";

interface ScheduledRouteColumnDeps {
  tenant: Tenant | null;
  vessels: Vessel[];
}

export function createScheduledRouteColumns({
  tenant,
  vessels,
}: ScheduledRouteColumnDeps): DataTableColumnDef<ScheduledRoute>[] {
  return [
    {
      id: "name",
      header: "Route",
      cell: ({ row }) => (
        <Link
          to={`/transports/routes/${row.original.id}`}
          className="font-medium hover:underline"
        >
          {row.original.attributes.name}
        </Link>
      ),
    },
    {
      id: "state",
      header: "State",
      cell: ({ row }) => (
        <RouteStatePill state={row.original.attributes.state} />
      ),
    },
    {
      id: "nextChange",
      header: "Next change",
      cell: ({ row }) => {
        const { nextState, nextTransitionAt } = row.original.attributes;
        const label = transitionLabel(nextState);
        if (!label || !nextTransitionAt) {
          return <span className="text-muted-foreground">—</span>;
        }
        return <Countdown targetAt={nextTransitionAt} label={label} />;
      },
    },
    {
      id: "startMap",
      header: "Start map",
      cell: ({ row }) => (
        <MapCell
          mapId={String(row.original.attributes.startMapId)}
          tenant={tenant}
        />
      ),
    },
    {
      id: "destinationMap",
      header: "Destination map",
      cell: ({ row }) => (
        <MapCell
          mapId={String(row.original.attributes.destinationMapId)}
          tenant={tenant}
        />
      ),
    },
    {
      id: "vessel",
      header: "Vessel",
      cell: ({ row }) => {
        const vessel = findVesselForRoute(row.original, vessels);
        if (!vessel) return <span className="text-muted-foreground">—</span>;
        return (
          <Link
            to={`/transports?tab=vessels#vessel-${vessel.id}`}
            className="hover:underline"
          >
            {vessel.attributes.name}
          </Link>
        );
      },
    },
    {
      id: "cycleInterval",
      header: "Cycle",
      cell: ({ row }) => (
        <span className="tabular-nums">
          {formatDurationSeconds(row.original.attributes.cycleIntervalSeconds)}
        </span>
      ),
    },
  ];
}
