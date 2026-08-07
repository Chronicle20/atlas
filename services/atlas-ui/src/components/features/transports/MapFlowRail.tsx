import { MapCell } from "@/components/map-cell";
import { cn } from "@/lib/utils";
import type { Tenant } from "@/types/models/tenant";
import type { ScheduledRoute } from "@/types/models/transport";

interface MapFlowRailProps {
  route: ScheduledRoute;
  tenant: Tenant | null;
}

interface Stop {
  mapId: number;
  enRoute: boolean;
}

interface Leg {
  caption: string;
  enRoute: boolean;
}

/**
 * The ordered chain of maps a character traverses:
 * start → staging → en-route… → destination.
 *
 * A single hidden `role="img"` element carries the whole rail's accessible
 * name and sequence summary, so assistive tech gets one labelled figure
 * rather than one per connector. The per-leg SVG arrows are decorative
 * (`aria-hidden`) because each leg's meaning is already conveyed by its
 * visible caption text, not by the arrow's colour or dash pattern alone; the
 * stop badges stay ordinary HTML so MapCell's link and copyable tooltip
 * remain reachable — wrapping them in role="img" would hide them from AT.
 */
export function MapFlowRail({ route, tenant }: MapFlowRailProps) {
  const {
    name,
    startMapId,
    stagingMapId,
    enRouteMapIds,
    destinationMapId,
    observationMapId,
    state,
  } = route.attributes;

  const stops: Stop[] = [
    { mapId: startMapId, enRoute: false },
    { mapId: stagingMapId, enRoute: false },
    ...enRouteMapIds.map((mapId) => ({ mapId, enRoute: true })),
    { mapId: destinationMapId, enRoute: false },
  ];

  // One leg between each adjacent pair of stops. The caption names the
  // mechanism that moves a character across it.
  const legs: Leg[] = stops.slice(1).map((stop, index) => {
    const previous = stops[index]!;
    if (!previous.enRoute && !stop.enRoute && index === 0) {
      return { caption: "walk in", enRoute: false };
    }
    if (stop.enRoute) {
      return { caption: "warp on departure", enRoute: true };
    }
    if (previous.enRoute) {
      return { caption: "warp on arrival", enRoute: false };
    }
    return { caption: "warp on departure", enRoute: false };
  });

  const inTransit = state === "in_transit";

  const sequenceSummary = stops
    .map((stop, index) =>
      index < legs.length
        ? `${stop.mapId} (${legs[index]!.caption})`
        : `${stop.mapId}`,
    )
    .join(" then ");

  return (
    <div className="space-y-3">
      <span
        role="img"
        aria-label={`Map flow for ${name}: ${sequenceSummary}`}
        className="sr-only"
      />
      <div className="overflow-x-auto">
        <div className="flex min-w-max items-start gap-2">
          {stops.map((stop, index) => (
            <div
              key={`${stop.mapId}-${index}`}
              className="flex items-start gap-2"
            >
              <div className="flex flex-col items-center gap-1 pt-1">
                <MapCell mapId={String(stop.mapId)} tenant={tenant} />
              </div>
              {index < legs.length ? (
                <RailLeg
                  leg={legs[index]!}
                  active={inTransit && legs[index]!.enRoute}
                />
              ) : null}
            </div>
          ))}
        </div>
      </div>

      <p className="text-xs text-muted-foreground">
        Observation map — where ARRIVED/DEPARTED effects fire; characters never
        travel here.{" "}
        <span className="align-middle">
          <MapCell mapId={String(observationMapId)} tenant={tenant} />
        </span>
      </p>
    </div>
  );
}

function RailLeg({ leg, active }: { leg: Leg; active: boolean }) {
  return (
    <div className="flex flex-col items-center gap-0.5">
      <svg
        aria-hidden="true"
        width="72"
        height="12"
        viewBox="0 0 72 12"
        data-en-route-active={active ? "true" : undefined}
        className={cn("text-muted-foreground", active && "text-primary")}
      >
        <line
          x1="0"
          y1="6"
          x2="64"
          y2="6"
          stroke="currentColor"
          strokeWidth={active ? 3 : 1.5}
          strokeDasharray={leg.enRoute ? "4 3" : undefined}
        />
        <path d="M64 2 L72 6 L64 10 Z" fill="currentColor" />
      </svg>
      <span className="text-[10px] text-muted-foreground whitespace-nowrap">
        {leg.caption}
      </span>
    </div>
  );
}
