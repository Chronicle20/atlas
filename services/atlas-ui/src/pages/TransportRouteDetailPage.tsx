import { useMemo, type ReactNode } from "react";
import { useParams } from "react-router-dom";
import { AlertTriangle } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";
import { MapCell } from "@/components/map-cell";
import { useTenant } from "@/context/tenant-context";
import {
  useScheduledRoute,
  useScheduledRoutes,
  useVessels,
} from "@/lib/hooks/api/useTransports";
import { Countdown } from "@/components/features/transports/Countdown";
import { MapFlowRail } from "@/components/features/transports/MapFlowRail";
import { RouteStatePill } from "@/components/features/transports/RouteStatePill";
import {
  VesselTimeline,
  type TimelineLane,
} from "@/components/features/transports/VesselTimeline";
import {
  findVesselForRoute,
  formatDurationSeconds,
  resolveVesselRoutes,
  transitionLabel,
} from "@/components/features/transports/transport-format";
import { useClock } from "@/lib/utils/clock";
import { isNotFoundError } from "@/types/api/errors";

/**
 * One scheduled route's detail: its map chain, configured timings, and a
 * windowed trip timeline. Assembles Tasks 8-17 — no new fetching or
 * formatting logic lives here.
 *
 * Loading, error and route-not-found are three visually distinct states (a
 * failed fetch must never fall through to a healthy-looking empty page); the
 * fourth, "no trips scheduled", is the timeline card's own empty state and is
 * rendered inline once the route itself has loaded successfully.
 */
export function TransportRouteDetailPage() {
  const { routeId = "" } = useParams();
  const { activeTenant } = useTenant();
  const now = useClock();

  const detailQuery = useScheduledRoute(routeId);
  const routesQuery = useScheduledRoutes();
  const vesselsQuery = useVessels();

  const detail = detailQuery.data;
  const routes = useMemo(() => routesQuery.data ?? [], [routesQuery.data]);
  const vessels = useMemo(() => vesselsQuery.data ?? [], [vesselsQuery.data]);

  const vessel = useMemo(
    () => (detail ? findVesselForRoute(detail.route, vessels) : null),
    [detail, vessels],
  );

  // Computed once and shared by `partner` and `vesselUnresolved` below —
  // `resolveVesselRoutes` walks the full route list by name, so it isn't
  // free, and both derivations need the same routeA/routeB resolution.
  const vesselResolution = useMemo(
    () => (vessel ? resolveVesselRoutes(vessel, routes) : null),
    [vessel, routes],
  );

  const partner = useMemo(() => {
    if (!detail || !vesselResolution) return null;
    const { routeA, routeB } = vesselResolution;
    if (routeA && routeA.id !== detail.route.id) return routeA;
    if (routeB && routeB.id !== detail.route.id) return routeB;
    return null;
  }, [detail, vesselResolution]);

  // Disabled (via useScheduledRoute's `!!routeId` gate) whenever there is no
  // partner, so a solo route triggers no extra fetch.
  const partnerDetailQuery = useScheduledRoute(partner?.id ?? "");

  const vesselUnresolved = vesselResolution?.unresolved ?? false;

  const lanes: TimelineLane[] = useMemo(() => {
    if (!detail) return [];
    const own: TimelineLane = {
      label: detail.route.attributes.name,
      trips: detail.schedule,
      emphasised: true,
    };
    const partnerSchedule = partnerDetailQuery.data;
    if (partner && partnerSchedule) {
      return [
        own,
        { label: partner.attributes.name, trips: partnerSchedule.schedule },
      ];
    }
    return [own];
  }, [detail, partner, partnerDetailQuery.data]);

  if (detailQuery.isLoading) {
    return <DetailSkeleton />;
  }

  if (detailQuery.isError || !detail) {
    const notFound = isNotFoundError(detailQuery.error);
    return (
      <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
        <ErrorDisplay
          title={notFound ? "Route not found" : "Error"}
          error={
            notFound
              ? "This route no longer exists. It may have been removed or renamed."
              : detailQuery.error?.message || "Failed to load this route."
          }
        />
      </div>
    );
  }

  const attributes = detail.route.attributes;
  const label = transitionLabel(attributes.nextState);

  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 overflow-y-auto p-10 pb-16">
      <div className="flex flex-wrap items-center gap-3">
        <h2 className="text-2xl font-bold tracking-tight">{attributes.name}</h2>
        <RouteStatePill state={attributes.state} />
        {label && attributes.nextTransitionAt ? (
          <Countdown targetAt={attributes.nextTransitionAt} label={label} />
        ) : null}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Map flow</CardTitle>
        </CardHeader>
        <CardContent>
          <MapFlowRail route={detail.route} tenant={activeTenant} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Configuration</CardTitle>
        </CardHeader>
        <CardContent>
          {/*
            A ruled cell grid, not free-floating label/value pairs: these are
            the seeded numbers an operator reads off against each other, and
            the 1px gap over `bg-border` is what draws the rules between them.
          */}
          <dl className="grid grid-cols-[repeat(auto-fit,minmax(150px,1fr))] gap-px border bg-border">
            <Field label="Observation map">
              <MapCell
                mapId={String(attributes.observationMapId)}
                tenant={activeTenant}
              />
            </Field>
            <Field label="Boarding window">
              {formatDurationSeconds(attributes.boardingWindowSeconds)}
            </Field>
            <Field label="Pre-departure">
              {formatDurationSeconds(attributes.preDepartureSeconds)}
            </Field>
            <Field label="Travel duration">
              {formatDurationSeconds(attributes.travelDurationSeconds)}
            </Field>
            <Field label="Cycle interval">
              {formatDurationSeconds(attributes.cycleIntervalSeconds)}
            </Field>
            <Field label="Trips scheduled today">
              {String(detail.schedule.length)}
            </Field>
            <Field label="Shared vessel">
              {vessel ? vessel.attributes.name : "—"}
            </Field>
          </dl>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Trip timeline</CardTitle>
        </CardHeader>
        <CardContent>
          {detail.schedule.length === 0 ? (
            <ScheduleFault
              vesselName={vessel?.attributes.name ?? null}
              vesselUnresolved={vesselUnresolved}
            />
          ) : (
            <VesselTimeline lanes={lanes} nowEpochMs={now} />
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function DetailSkeleton() {
  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
      <span className="sr-only">Loading route…</span>
      <Skeleton className="h-8 w-64" />
      <Skeleton className="h-40 w-full" />
      <Skeleton className="h-32 w-full" />
      <Skeleton className="h-40 w-full" />
    </div>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="bg-card px-3 py-2.5">
      <dt className="font-mono text-[10px] uppercase tracking-[0.13em] text-muted-foreground">
        {label}
      </dt>
      <dd className="mt-1 font-mono text-[13px] tabular-nums">{children}</dd>
    </div>
  );
}

/**
 * A route with no trips is a configuration fault, and there are exactly two
 * producible causes: a cycle-interval / travel-duration combination that
 * leaves no trip fitting inside the day (the scheduler drops a trip unless it
 * arrives before end of day), or membership in a vessel whose partner does not
 * resolve (which returns an empty schedule for both sides).
 */
function ScheduleFault({
  vesselName,
  vesselUnresolved,
}: {
  vesselName: string | null;
  vesselUnresolved: boolean;
}) {
  return (
    <div className="flex items-start gap-2 text-sm text-destructive">
      <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
      <div className="space-y-1">
        <p>No trips were scheduled for this route today.</p>
        {vesselUnresolved && vesselName ? (
          <p>
            Its shared vessel <strong>{vesselName}</strong> does not resolve —
            one of its route references matches no route, which zeroes the
            schedule for both sides. Fix the vessel configuration.
          </p>
        ) : (
          <p>
            Check the route&apos;s cycle interval and travel duration: the
            scheduler drops any trip that would not arrive before the end of the
            day, so a cycle longer than the day&apos;s remaining room produces
            no trips at all.
          </p>
        )}
      </div>
    </div>
  );
}
