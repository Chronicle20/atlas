import { Fragment } from "react";

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
  /** Uppercase role caption under the stop — "start", "en route 2", … */
  role: string;
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
 * Drawn as a transit rail: each stop is a node (a dot, the map, and its role in
 * the chain) and each leg is a stretched connector captioned with the mechanism
 * that moves a character across it. The connectors flex, so the rail fills the
 * card at any width above its `min-w` floor rather than clustering the stops on
 * the left.
 *
 * A single hidden `role="img"` element carries the whole rail's accessible
 * name and sequence summary, so assistive tech gets one labelled figure
 * rather than one per connector. When the route is in transit, that same
 * name appends which stops are currently being traversed — "where the
 * vessel currently is" must survive without colour, and the per-leg
 * highlight is otherwise conveyed only by the connector's colour and weight.
 * The connectors themselves are decorative (`aria-hidden`) because each leg's
 * identity is already conveyed by its visible caption text, not by the
 * connector's colour or dash pattern alone; the stop badges stay ordinary HTML
 * so MapCell's link and copyable tooltip remain reachable — wrapping them in
 * role="img" would hide them from AT.
 */
export function MapFlowRail({ route, tenant }: MapFlowRailProps) {
  const {
    name,
    startMapId,
    stagingMapId,
    enRouteMapIds,
    destinationMapId,
    state,
  } = route.attributes;

  const stops: Stop[] = [
    { mapId: startMapId, role: "start", enRoute: false },
    { mapId: stagingMapId, role: "staging", enRoute: false },
    ...enRouteMapIds.map((mapId, index) => ({
      mapId,
      role: enRouteMapIds.length > 1 ? `en route ${index + 1}` : "en route",
      enRoute: true,
    })),
    { mapId: destinationMapId, role: "destination", enRoute: false },
  ];

  // One leg between each adjacent pair of stops. The caption names the
  // mechanism that moves a character across it. Position in the chain, not
  // just the adjacent stops' en-route flags, decides the caption: the first
  // leg (start→staging) is always "walk in", the last leg (→destination) is
  // always "warp on arrival" — even when there's no en-route stop between
  // staging and the destination, i.e. `enRouteMapIds` is empty and the last
  // leg's neighbours are both non-en-route. Every leg in between is a
  // "warp on departure" that carries the character onto or through the
  // en-route chain.
  const legs: Leg[] = stops.slice(1).map((stop, index) => {
    const previous = stops[index]!;
    const isFirstLeg = index === 0;
    const isLastLeg = index === stops.length - 2;
    if (isFirstLeg) {
      return { caption: "walk in", enRoute: false };
    }
    if (isLastLeg) {
      return { caption: "warp on arrival", enRoute: false };
    }
    return {
      caption: "warp on departure",
      enRoute: stop.enRoute || previous.enRoute,
    };
  });

  const inTransit = state === "in_transit";

  const sequenceSummary = stops
    .map((stop, index) =>
      index < legs.length
        ? `${stop.mapId} (${legs[index]!.caption})`
        : `${stop.mapId}`,
    )
    .join(" then ");

  // "Where the vessel currently is" must be discoverable without colour —
  // the only other signal for the active leg is the connector's colour and
  // weight (see RailLeg), which is aria-hidden. Naming the en-route stops here
  // is the accessible channel for that state.
  const transitClause =
    inTransit && enRouteMapIds.length > 0
      ? ` — currently in transit through ${enRouteMapIds.join(", ")}`
      : "";

  return (
    <div className="space-y-3">
      <span
        role="img"
        aria-label={`Map flow for ${name}: ${sequenceSummary}${transitClause}`}
        className="sr-only"
      />
      <div className="overflow-x-auto pb-1">
        <div className="flex w-full min-w-[640px] items-start">
          {stops.map((stop, index) => (
            <Fragment key={`${stop.mapId}-${index}`}>
              <RailStop
                stop={stop}
                tenant={tenant}
                active={inTransit && stop.enRoute}
              />
              {index < legs.length ? (
                <RailLeg
                  leg={legs[index]!}
                  active={inTransit && legs[index]!.enRoute}
                />
              ) : null}
            </Fragment>
          ))}
        </div>
      </div>
    </div>
  );
}

function RailStop({
  stop,
  tenant,
  active,
}: {
  stop: Stop;
  tenant: Tenant | null;
  active: boolean;
}) {
  return (
    <div className="flex shrink-0 flex-col items-center gap-1.5 px-1.5 text-center">
      <span
        aria-hidden="true"
        data-stop-dot={active ? "active" : "idle"}
        className={cn(
          "mt-1 h-3.5 w-3.5 rounded-full border-[2.5px]",
          active
            ? "border-primary bg-primary"
            : "border-muted-foreground bg-card",
        )}
      />
      <MapCell mapId={String(stop.mapId)} tenant={tenant} />
      <span className="font-mono text-[10px] uppercase tracking-[0.1em] text-muted-foreground">
        {stop.role}
      </span>
    </div>
  );
}

function RailLeg({ leg, active }: { leg: Leg; active: boolean }) {
  // pt-[10px] sits the connector on the stop dots' centre line: the dot's
  // mt-1 (4px) plus half its h-3.5 (7px), less half the connector's 2px rule.
  return (
    <div
      className="flex min-w-[64px] flex-1 flex-col gap-1.5 pt-[10px]"
      data-en-route-active={active ? "true" : undefined}
    >
      <span
        aria-hidden="true"
        className={cn(
          "w-full border-t-2",
          active ? "border-primary" : "border-border",
          // En-route legs are the client-side ride, not a server warp — dashed
          // so the two read differently at a glance.
          leg.enRoute && "border-dashed",
        )}
      />
      <span className="px-1 text-center font-mono text-[10px] leading-tight text-muted-foreground">
        {leg.caption}
      </span>
    </div>
  );
}
