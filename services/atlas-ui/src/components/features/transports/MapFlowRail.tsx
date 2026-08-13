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
  /**
   * The maps this stop covers. More than one only for the en-route stop, whose
   * maps are parallel variants of the same leg — never a sequence.
   */
  mapIds: number[];
  /** Uppercase role caption under the stop — "start", "en route", … */
  role: string;
  enRoute: boolean;
}

interface Leg {
  caption: string;
  enRoute: boolean;
}

/**
 * The ordered chain of maps a character traverses:
 * start → staging → en route → destination.
 *
 * The en-route stop is ONE stop even when `enRouteMapIds` holds several maps.
 * Those maps are parallel variants of the same ride, not successive legs, and
 * the service is unambiguous about it: departure warps everyone in staging into
 * `enRouteMapIds[0]` and nothing ever moves them onward through the rest
 * (`transport/processor.go` — `InTransit` branch), while arrival drains every
 * en-route map straight to the destination (`AwaitingReturn` branch). Drawing
 * them as a chain claimed a character rides map 1, then map 2, then arrives —
 * a trip that does not happen. So the entry map sits on the rail and its
 * siblings hang beneath it off a brace, which is what "same leg, other map"
 * looks like.
 *
 * Drawn as a transit rail: each stop is a node (a dot, the map, and its role in
 * the chain) and each leg is a stretched connector captioned with the mechanism
 * that moves a character across it. The connectors flex, so the rail fills the
 * card at any width above its `min-w` floor rather than clustering the stops on
 * the left.
 *
 * A single hidden `role="img"` element carries the whole rail's accessible
 * name and sequence summary, so assistive tech gets one labelled figure
 * rather than one per connector. That name spells out the parallel maps as
 * parallel, names which one a departure lands in, and — when the route is in
 * transit — which stops are currently being traversed: "where the vessel
 * currently is" must survive without colour, and the per-leg highlight is
 * otherwise conveyed only by the connector's colour and weight. The connectors
 * themselves are decorative (`aria-hidden`) because each leg's identity is
 * already conveyed by its visible caption text, not by the connector's colour
 * or dash pattern alone; the stop badges stay ordinary HTML so MapCell's link
 * and copyable tooltip remain reachable — wrapping them in role="img" would
 * hide them from AT.
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
    { mapIds: [startMapId], role: "start", enRoute: false },
    { mapIds: [stagingMapId], role: "staging", enRoute: false },
    ...(enRouteMapIds.length > 0
      ? [{ mapIds: enRouteMapIds, role: "en route", enRoute: true }]
      : []),
    { mapIds: [destinationMapId], role: "destination", enRoute: false },
  ];

  // One leg between each adjacent pair of stops. The caption names the
  // mechanism that moves a character across it. Position in the chain, not
  // just the adjacent stops' en-route flags, decides the caption: the first
  // leg (start→staging) is always "walk in", the last leg (→destination) is
  // always "warp on arrival". Between them there is at most one leg — the
  // departure warp into the en-route stop — and it exists only when the route
  // has en-route maps at all; a route with none collapses to walk-in and
  // arrival, whose neighbours are both non-en-route.
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
    .map((stop, index) => {
      const maps =
        stop.mapIds.length > 1
          ? `${stop.mapIds[0]} in parallel with ${stop.mapIds.slice(1).join(", ")}`
          : `${stop.mapIds[0]}`;
      return index < legs.length ? `${maps} (${legs[index]!.caption})` : maps;
    })
    .join(" then ");

  // The parallel maps are one leg with several rooms, and which room a
  // departure lands in is carried visually by a single "entry" sub-caption on
  // one badge. Spelling it out is the accessible channel for that.
  const parallelClause =
    enRouteMapIds.length > 1
      ? ` En route runs across ${enRouteMapIds.length} parallel maps: a departure lands in ${enRouteMapIds[0]}, and on arrival every one of them is cleared to the destination.`
      : "";

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
        aria-label={`Map flow for ${name}: ${sequenceSummary}${transitClause}.${parallelClause}`}
        className="sr-only"
      />
      <div className="overflow-x-auto pb-1">
        <div className="flex w-full min-w-[640px] items-start">
          {stops.map((stop, index) => (
            <Fragment key={`${stop.role}-${index}`}>
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
  const [entryMapId, ...siblingMapIds] = stop.mapIds;
  const parallel = siblingMapIds.length > 0;

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
      {/*
        The entry map stays on the rail line so the connectors either side of
        this stop keep meeting a dot, exactly as the single-map stops do.
      */}
      <MapCell mapId={String(entryMapId)} tenant={tenant} />
      <span className="font-mono text-[10px] uppercase tracking-[0.1em] text-muted-foreground">
        {parallel ? `${stop.role} · entry` : stop.role}
      </span>

      {parallel ? (
        <div data-parallel-maps="" className="flex flex-col items-center">
          {/*
            A stub of the same dashed rule the en-route connectors use, hanging
            off the rail — the siblings branch from this stop rather than
            following it.
          */}
          <span
            aria-hidden="true"
            className={cn(
              "h-3 border-l-2 border-dashed",
              active ? "border-primary" : "border-border",
            )}
          />
          <div
            className={cn(
              "flex flex-col items-center gap-1.5 rounded-md border border-dashed px-2 py-1.5",
              active ? "border-primary" : "border-border",
            )}
          >
            {siblingMapIds.map((mapId) => (
              <MapCell key={mapId} mapId={String(mapId)} tenant={tenant} />
            ))}
            <span className="max-w-[9rem] font-mono text-[10px] uppercase leading-tight tracking-[0.1em] text-muted-foreground">
              also en route
            </span>
          </div>
        </div>
      ) : null}
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
