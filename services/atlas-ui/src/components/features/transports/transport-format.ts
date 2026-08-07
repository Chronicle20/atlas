import type {
  RouteState,
  ScheduledRoute,
  Vessel,
} from "@/types/models/transport";

const MS_PER_DAY = 86_400_000;
const MIN_HALF_WINDOW_MS = 10 * 60_000;
const MAX_HALF_WINDOW_MS = 30 * 60_000;

/**
 * Board ordering: faults first, then the states closest to a change, so a bad
 * route and an imminent departure both sort to the top. `out_of_service` means
 * the scheduler produced no trips at all for that route today.
 */
export const STATE_SEVERITY: Record<RouteState, number> = {
  out_of_service: 0,
  in_transit: 1,
  locked_entry: 2,
  open_entry: 3,
  awaiting_return: 4,
};

export function compareRoutesBySeverityThenName(
  a: ScheduledRoute,
  b: ScheduledRoute,
): number {
  const severity =
    STATE_SEVERITY[a.attributes.state] - STATE_SEVERITY[b.attributes.state];
  if (severity !== 0) return severity;
  return a.attributes.name.localeCompare(b.attributes.name);
}

export function stateLabel(state: RouteState): string {
  switch (state) {
    case "out_of_service":
      return "Out of service";
    case "in_transit":
      return "In transit";
    case "locked_entry":
      return "Boarding closed";
    case "open_entry":
      return "Boarding";
    case "awaiting_return":
      return "Awaiting return";
  }
}

/**
 * The countdown's caption, derived from the state being moved *to*. A route
 * about to enter open_entry "boards in"; one about to enter in_transit
 * "departs in".
 */
export function transitionLabel(nextState: RouteState | ""): string | null {
  switch (nextState) {
    case "open_entry":
      return "boards in";
    case "locked_entry":
      return "closes in";
    case "in_transit":
      return "departs in";
    case "awaiting_return":
      return "arrives in";
    default:
      return null;
  }
}

/** `mm:ss`, or `h:mm:ss` at an hour or more. Clamps at `0:00`. */
export function formatCountdown(msRemaining: number): string {
  const total = Math.max(0, Math.floor(msRemaining / 1000));
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const seconds = total % 60;
  const pad = (value: number) => String(value).padStart(2, "0");
  if (hours > 0) return `${hours}:${pad(minutes)}:${pad(seconds)}`;
  return `${minutes}:${pad(seconds)}`;
}

/** Human-readable configured duration, e.g. `15m`, `1m 30s`, `1h 5m`. */
export function formatDurationSeconds(seconds: number): string {
  if (seconds <= 0) return "0s";
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const rest = seconds % 60;
  if (hours > 0) return minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`;
  if (minutes > 0) return rest > 0 ? `${minutes}m ${rest}s` : `${minutes}m`;
  return `${rest}s`;
}

/**
 * Renders a trip boundary's UTC time of day and nothing else.
 *
 * Trip-schedule timestamps carry the date of the day the schedule was computed
 * — a stale date whose only meaningful part is the time. The schedule is
 * anchored on UTC midnight, so the UTC components are the real boarding and
 * departure times; every schedule surface goes through this function and
 * labels its times UTC.
 */
export function formatTimeOfDay(iso: string): string {
  const parsed = new Date(iso);
  if (Number.isNaN(parsed.getTime())) return "—";
  const pad = (value: number) => String(value).padStart(2, "0");
  return `${pad(parsed.getUTCHours())}:${pad(parsed.getUTCMinutes())}`;
}

/** Milliseconds since UTC midnight for a schedule timestamp. */
export function utcTimeOfDayMs(iso: string): number {
  const parsed = new Date(iso);
  if (Number.isNaN(parsed.getTime())) return 0;
  return (
    parsed.getUTCHours() * 3_600_000 +
    parsed.getUTCMinutes() * 60_000 +
    parsed.getUTCSeconds() * 1000 +
    parsed.getUTCMilliseconds()
  );
}

/** Milliseconds since UTC midnight for an epoch instant. */
export function nowUtcTimeOfDayMs(now: number): number {
  return ((now % MS_PER_DAY) + MS_PER_DAY) % MS_PER_DAY;
}

/**
 * Half-width of the timeline window, derived from the median gap between
 * consecutive boarding-open times rather than from `cycleInterval`: a shared
 * vessel's real spacing is arrival-plus-turnaround, not either route's
 * configured cycle. Clamped to 10-30 minutes so the 6-minute plane does not
 * put ten legs on screen and the 15-minute boat does not show two.
 */
export function timelineHalfWindowMs(boardingOpenMs: number[]): number {
  if (boardingOpenMs.length < 2) return MAX_HALF_WINDOW_MS;

  const sorted = [...boardingOpenMs].sort((a, b) => a - b);
  const gaps: number[] = [];
  for (let i = 1; i < sorted.length; i++) {
    const gap = sorted[i]! - sorted[i - 1]!;
    if (gap > 0) gaps.push(gap);
  }
  if (gaps.length === 0) return MAX_HALF_WINDOW_MS;

  gaps.sort((a, b) => a - b);
  const middle = Math.floor(gaps.length / 2);
  const median =
    gaps.length % 2 === 0
      ? (gaps[middle - 1]! + gaps[middle]!) / 2
      : gaps[middle]!;

  return Math.min(
    MAX_HALF_WINDOW_MS,
    Math.max(MIN_HALF_WINDOW_MS, 1.5 * median),
  );
}

/**
 * Fractional [left, right] extent of a time-of-day segment inside the window
 * centred on `nowMs`, clipped to the window's edges. Null when the segment
 * falls entirely outside.
 *
 * The segment is anchored on whichever ±1-day placement of its start lands
 * closest to now, which is what lets a window spanning UTC midnight draw
 * late-evening and early-morning trips on one strip.
 */
export function segmentSpan(
  startMs: number,
  endMs: number,
  nowMs: number,
  halfWindowMs: number,
): { left: number; right: number } | null {
  const duration =
    endMs >= startMs ? endMs - startMs : endMs + MS_PER_DAY - startMs;

  let startOffset = startMs - nowMs;
  for (const candidate of [
    startMs - MS_PER_DAY,
    startMs,
    startMs + MS_PER_DAY,
  ]) {
    const offset = candidate - nowMs;
    if (Math.abs(offset) < Math.abs(startOffset)) {
      startOffset = offset;
    }
  }

  const endOffset = startOffset + duration;
  if (endOffset < -halfWindowMs || startOffset > halfWindowMs) return null;

  const toFraction = (offset: number) =>
    (Math.min(halfWindowMs, Math.max(-halfWindowMs, offset)) + halfWindowMs) /
    (2 * halfWindowMs);

  return { left: toFraction(startOffset), right: toFraction(endOffset) };
}

/**
 * Whether an instance is approaching the stuck-timeout force-warp.
 *
 * The server sweeps on `now - createdAt > MaxLifetime`, where MaxLifetime is
 * `2 × (boardingWindow + travelDuration)`. Warning at two thirds of the same
 * quantity keeps the warning and the action in agreement.
 */
export function isInstanceStuck(
  createdAt: string,
  boardingWindowSeconds: number,
  travelDurationSeconds: number,
  now: number,
): boolean {
  const created = Date.parse(createdAt);
  if (Number.isNaN(created)) return false;
  const maxLifetimeMs =
    2 * (boardingWindowSeconds + travelDurationSeconds) * 1000;
  if (maxLifetimeMs <= 0) return false;
  return now - created > (2 / 3) * maxLifetimeMs;
}

/**
 * Resolves a vessel's two sides against the scheduled-route list **by name**,
 * which is the rule the backend scheduler itself uses. An unresolved side is
 * not cosmetic: the scheduler returns an empty schedule for the vessel, which
 * drives *both* of its routes to out_of_service.
 *
 * Vessel ids are configuration slugs, not names — do not match on `vessel.id`
 * even where the seed data happens to make them equal.
 */
export function resolveVesselRoutes(
  vessel: Vessel,
  routes: ScheduledRoute[],
): {
  routeA: ScheduledRoute | null;
  routeB: ScheduledRoute | null;
  unresolved: boolean;
} {
  const byName = (name: string) =>
    routes.find((route) => route.attributes.name === name) ?? null;

  const routeA = byName(vessel.attributes.routeAID);
  const routeB = byName(vessel.attributes.routeBID);

  return { routeA, routeB, unresolved: routeA === null || routeB === null };
}

/** The shared vessel a route belongs to, or null when it runs independently. */
export function findVesselForRoute(
  route: ScheduledRoute,
  vessels: Vessel[],
): Vessel | null {
  return (
    vessels.find(
      (vessel) =>
        vessel.attributes.routeAID === route.attributes.name ||
        vessel.attributes.routeBID === route.attributes.name,
    ) ?? null
  );
}
