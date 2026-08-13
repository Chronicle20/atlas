import { describe, expect, it } from "vitest";

import {
  compareRoutesBySeverityThenName,
  findVesselForRoute,
  formatClockMs,
  formatCountdown,
  formatDurationSeconds,
  formatTimeOfDay,
  formatTimeOfDayMs,
  isInstanceStuck,
  nowUtcTimeOfDayMs,
  resolveVesselRoutes,
  segmentSpan,
  timelineAxisTicks,
  timelineHalfWindowMs,
  transitionLabel,
  utcTimeOfDayMs,
} from "@/components/features/transports/transport-format";
import type {
  RouteState,
  ScheduledRoute,
  Vessel,
} from "@/types/models/transport";

const MINUTE = 60_000;

function route(name: string, state: RouteState): ScheduledRoute {
  return {
    id: `id-${name}`,
    attributes: {
      name,
      startMapId: 1,
      stagingMapId: 2,
      enRouteMapIds: [3],
      destinationMapId: 4,
      observationMapId: 5,
      state,
      boardingWindowSeconds: 300,
      preDepartureSeconds: 120,
      travelDurationSeconds: 600,
      cycleIntervalSeconds: 900,
      nextTransitionAt: "",
      nextState: "",
    },
  };
}

function vessel(routeAID: string, routeBID: string): Vessel {
  return {
    id: "vessel-slug",
    attributes: {
      uuid: "vessel-uuid",
      name: "Shared Hull",
      routeAID,
      routeBID,
      turnaroundDelay: 60,
    },
  };
}

describe("compareRoutesBySeverityThenName", () => {
  it("sorts out_of_service above every other state", () => {
    const sorted = [
      route("Bravo", "awaiting_return"),
      route("Alpha", "in_transit"),
      route("Zulu", "out_of_service"),
    ].sort(compareRoutesBySeverityThenName);

    expect(sorted.map((r) => r.attributes.name)).toEqual([
      "Zulu",
      "Alpha",
      "Bravo",
    ]);
  });

  it("orders the remaining states by severity then name", () => {
    const sorted = [
      route("D", "awaiting_return"),
      route("C", "open_entry"),
      route("B", "locked_entry"),
      route("A", "in_transit"),
    ].sort(compareRoutesBySeverityThenName);

    expect(sorted.map((r) => r.attributes.name)).toEqual(["A", "B", "C", "D"]);
  });

  it("breaks severity ties on name", () => {
    const sorted = [
      route("Zeta", "open_entry"),
      route("Alpha", "open_entry"),
    ].sort(compareRoutesBySeverityThenName);

    expect(sorted.map((r) => r.attributes.name)).toEqual(["Alpha", "Zeta"]);
  });
});

describe("transitionLabel", () => {
  it("names the transition from the state being moved to", () => {
    expect(transitionLabel("open_entry")).toBe("boards in");
    expect(transitionLabel("locked_entry")).toBe("closes in");
    expect(transitionLabel("in_transit")).toBe("departs in");
    expect(transitionLabel("awaiting_return")).toBe("arrives in");
  });

  it("has no label for a route with no transition", () => {
    expect(transitionLabel("")).toBeNull();
    expect(transitionLabel("out_of_service")).toBeNull();
  });
});

describe("formatCountdown", () => {
  it("renders mm:ss below an hour", () => {
    expect(formatCountdown(30_000)).toBe("0:30");
    expect(formatCountdown(5 * MINUTE + 3000)).toBe("5:03");
  });

  it("renders h:mm:ss at or above an hour", () => {
    expect(formatCountdown(3600_000 + 5 * MINUTE + 4000)).toBe("1:05:04");
  });

  it("clamps at zero and never goes negative", () => {
    expect(formatCountdown(0)).toBe("0:00");
    expect(formatCountdown(-90_000)).toBe("0:00");
  });
});

describe("formatDurationSeconds", () => {
  it("renders minutes and seconds", () => {
    expect(formatDurationSeconds(900)).toBe("15m");
    expect(formatDurationSeconds(90)).toBe("1m 30s");
    expect(formatDurationSeconds(45)).toBe("45s");
    expect(formatDurationSeconds(3900)).toBe("1h 5m");
  });
});

describe("formatTimeOfDay", () => {
  it("renders the UTC time component and never a date", () => {
    // The date here is the schedule's stale computing day - it must not leak.
    expect(formatTimeOfDay("2023-01-01T08:07:00Z")).toBe("08:07");
    expect(formatTimeOfDay("2023-01-01T23:59:00Z")).toBe("23:59");
    expect(formatTimeOfDay("2023-01-01T08:07:00Z")).not.toContain("2023");
  });

  it("returns an em dash for an unparseable timestamp", () => {
    expect(formatTimeOfDay("not a date")).toBe("—");
  });
});

describe("formatTimeOfDayMs / formatClockMs", () => {
  it("renders a milliseconds-since-midnight value as a clock time", () => {
    expect(formatTimeOfDayMs(8 * 60 * MINUTE + 7 * MINUTE)).toBe("08:07");
    expect(formatClockMs(8 * 60 * MINUTE + 7 * MINUTE + 12_000)).toBe(
      "08:07:12",
    );
  });

  it("wraps past a day rather than rendering an impossible hour", () => {
    // Axis ticks past UTC midnight keep counting up so the axis stays
    // monotonic; only the label wraps.
    expect(formatTimeOfDayMs(24 * 60 * MINUTE)).toBe("00:00");
    expect(formatTimeOfDayMs(24 * 60 * MINUTE + 10 * MINUTE)).toBe("00:10");
  });
});

describe("timelineAxisTicks", () => {
  const at = (hours: number, minutes: number) =>
    hours * 60 * MINUTE + minutes * MINUTE;

  it("lands on round wall-clock times, not on offsets from now", () => {
    // now is 12:03:30, so ticks anchored on now would read :03:30 — the axis
    // has to read as a clock instead.
    const ticks = timelineAxisTicks(at(12, 3) + 30_000, 30 * MINUTE);

    expect(ticks.map((tick) => formatTimeOfDayMs(tick.ms))).toEqual([
      "11:40",
      "11:50",
      "12:00",
      "12:10",
      "12:20",
      "12:30",
    ]);
  });

  it("coarsens the interval with the window so the labels never crowd", () => {
    const narrow = timelineAxisTicks(at(12, 0), 10 * MINUTE);
    const wide = timelineAxisTicks(at(12, 0), 30 * MINUTE);

    expect(narrow.map((tick) => formatTimeOfDayMs(tick.ms))).toEqual([
      "11:50",
      "11:55",
      "12:00",
      "12:05",
      "12:10",
    ]);
    expect(wide).toHaveLength(7);
  });

  it("positions each tick as a fraction of the window", () => {
    const ticks = timelineAxisTicks(at(12, 0), 30 * MINUTE);

    expect(ticks[0]?.fraction).toBe(0);
    expect(ticks[ticks.length - 1]?.fraction).toBe(1);
    expect(
      ticks.find((tick) => formatTimeOfDayMs(tick.ms) === "12:00")?.fraction,
    ).toBe(0.5);
  });

  it("keeps counting past UTC midnight so the axis stays monotonic", () => {
    const ticks = timelineAxisTicks(at(23, 55), 30 * MINUTE);

    expect(ticks.map((tick) => tick.ms)).toEqual(
      [...ticks].sort((a, b) => a.ms - b.ms).map((tick) => tick.ms),
    );
    expect(ticks.map((tick) => formatTimeOfDayMs(tick.ms))).toContain("00:10");
  });

  it("returns no ticks for a degenerate window", () => {
    expect(timelineAxisTicks(at(12, 0), 0)).toEqual([]);
  });
});

describe("utcTimeOfDayMs / nowUtcTimeOfDayMs", () => {
  it("measures milliseconds since UTC midnight", () => {
    expect(utcTimeOfDayMs("2023-01-01T00:00:00Z")).toBe(0);
    expect(utcTimeOfDayMs("2023-01-01T01:30:00Z")).toBe(90 * MINUTE);
    expect(nowUtcTimeOfDayMs(Date.parse("2026-08-06T12:15:00Z"))).toBe(
      12 * 60 * MINUTE + 15 * MINUTE,
    );
  });
});

describe("timelineHalfWindowMs", () => {
  it("derives the window from median trip spacing, clamped to 10-30 minutes", () => {
    const boats = [0, 15, 30, 45].map((m) => m * MINUTE);
    // median gap 15m * 1.5 = 22.5m, inside the clamp
    expect(timelineHalfWindowMs(boats)).toBe(22.5 * MINUTE);

    const plane = [0, 6, 12, 18].map((m) => m * MINUTE);
    // 6m * 1.5 = 9m, clamped up to 10m
    expect(timelineHalfWindowMs(plane)).toBe(10 * MINUTE);

    const slow = [0, 60, 120].map((m) => m * MINUTE);
    // 60m * 1.5 = 90m, clamped down to 30m
    expect(timelineHalfWindowMs(slow)).toBe(30 * MINUTE);
  });

  it("falls back to the widest window when spacing cannot be measured", () => {
    expect(timelineHalfWindowMs([])).toBe(30 * MINUTE);
    expect(timelineHalfWindowMs([5 * MINUTE])).toBe(30 * MINUTE);
  });
});

describe("segmentSpan", () => {
  const now = 12 * 60 * MINUTE;
  const half = 30 * MINUTE;

  it("spans a segment fully inside the window", () => {
    const span = segmentSpan(now - 10 * MINUTE, now + 10 * MINUTE, now, half);
    expect(span).not.toBeNull();
    expect(span!.left).toBeCloseTo(1 / 3);
    expect(span!.right).toBeCloseTo(2 / 3);
  });

  it("puts a zero-length segment at now in the centre", () => {
    const span = segmentSpan(now, now, now, half);
    expect(span!.left).toBeCloseTo(0.5);
    expect(span!.right).toBeCloseTo(0.5);
  });

  it("clips a segment that overhangs an edge", () => {
    const span = segmentSpan(now - 60 * MINUTE, now, now, half);
    expect(span!.left).toBeCloseTo(0);
    expect(span!.right).toBeCloseTo(0.5);
  });

  it("returns null for a segment entirely outside the window", () => {
    expect(
      segmentSpan(now + 40 * MINUTE, now + 50 * MINUTE, now, half),
    ).toBeNull();
  });

  it("wraps a segment across UTC midnight", () => {
    const nearMidnight = 23 * 60 * MINUTE + 55 * MINUTE; // 23:55
    // 00:05 - 00:15 the next day is 10 to 20 minutes after 23:55
    const span = segmentSpan(5 * MINUTE, 15 * MINUTE, nearMidnight, half);
    expect(span).not.toBeNull();
    expect(span!.left).toBeCloseTo((10 * MINUTE + half) / (2 * half));
  });
});

describe("isInstanceStuck", () => {
  // MaxLifetime = 2 * (60 + 120) = 360s; two thirds of that is 240s.
  const created = "2026-08-06T12:00:00Z";

  it("is false below two thirds of MaxLifetime", () => {
    expect(
      isInstanceStuck(created, 60, 120, Date.parse("2026-08-06T12:03:00Z")),
    ).toBe(false);
  });

  it("is true past two thirds of MaxLifetime", () => {
    expect(
      isInstanceStuck(created, 60, 120, Date.parse("2026-08-06T12:05:00Z")),
    ).toBe(true);
  });
});

describe("resolveVesselRoutes", () => {
  const orbis = route("Orbis to Ellinia", "open_entry");
  const ellinia = route("Ellinia to Orbis", "awaiting_return");

  it("resolves both sides by route name", () => {
    const result = resolveVesselRoutes(
      vessel("Orbis to Ellinia", "Ellinia to Orbis"),
      [orbis, ellinia],
    );

    expect(result.routeA?.id).toBe(orbis.id);
    expect(result.routeB?.id).toBe(ellinia.id);
    expect(result.unresolved).toBe(false);
  });

  it("flags a vessel whose reference matches no route", () => {
    const result = resolveVesselRoutes(
      vessel("Orbis to Ellinia", "Typo to Nowhere"),
      [orbis, ellinia],
    );

    expect(result.routeA?.id).toBe(orbis.id);
    expect(result.routeB).toBeNull();
    expect(result.unresolved).toBe(true);
  });

  it("does not match on the vessel slug", () => {
    const result = resolveVesselRoutes(vessel("vessel-slug", "vessel-slug"), [
      orbis,
    ]);

    expect(result.unresolved).toBe(true);
  });
});

describe("findVesselForRoute", () => {
  const orbis = route("Orbis to Ellinia", "open_entry");

  it("finds the vessel a route belongs to by name", () => {
    const found = findVesselForRoute(orbis, [
      vessel("Ellinia to Orbis", "Orbis to Ellinia"),
    ]);
    expect(found?.id).toBe("vessel-slug");
  });

  it("returns null for an independent route", () => {
    expect(findVesselForRoute(orbis, [vessel("A", "B")])).toBeNull();
  });
});
