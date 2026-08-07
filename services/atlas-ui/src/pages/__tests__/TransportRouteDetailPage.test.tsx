import { beforeEach, describe, expect, it, vi } from "vitest";
import { act, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { TransportRouteDetailPage } from "@/pages/TransportRouteDetailPage";
import { transportsService } from "@/services/api/transports.service";
import type { Tenant } from "@/types/models/tenant";
import type {
  RouteState,
  ScheduledRoute,
  TripSchedule,
} from "@/types/models/transport";

const mockTenant: Tenant = {
  id: "tenant-1",
  attributes: {
    name: "Test Tenant",
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
  },
};

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: mockTenant,
    tenants: [mockTenant],
    loading: false,
    setActiveTenant: vi.fn(),
    refreshTenants: vi.fn(),
    refreshAndSelectTenant: vi.fn(),
    fetchTenantConfiguration: vi.fn(),
  }),
}));

vi.mock("@/components/map-cell", () => ({
  MapCell: ({ mapId }: { mapId: string }) => <span>map:{mapId}</span>,
}));

vi.mock("@/services/api/transports.service", () => ({
  transportsService: {
    getScheduledRoute: vi.fn(),
    getScheduledRoutes: vi.fn(),
    getVessels: vi.fn(),
  },
}));

function scheduledRoute(state: RouteState): ScheduledRoute {
  return {
    id: "r1",
    attributes: {
      name: "Orbis to Ellinia",
      startMapId: 200000100,
      stagingMapId: 200000110,
      enRouteMapIds: [200090010],
      destinationMapId: 101000300,
      observationMapId: 200000111,
      state,
      boardingWindowSeconds: 300,
      preDepartureSeconds: 120,
      travelDurationSeconds: 600,
      cycleIntervalSeconds: 900,
      nextTransitionAt: "2026-08-06T12:05:00Z",
      nextState: "locked_entry",
    },
  };
}

const trips: TripSchedule[] = [
  {
    id: "t1",
    attributes: {
      boardingOpen: "2023-01-01T11:45:00Z",
      boardingClosed: "2023-01-01T11:50:00Z",
      departure: "2023-01-01T11:52:00Z",
      arrival: "2023-01-01T12:02:00Z",
    },
  },
];

/** The other side of a shared vessel — a distinct id/name from `scheduledRoute`. */
function partnerRoute(state: RouteState): ScheduledRoute {
  return {
    id: "r2",
    attributes: {
      name: "Ellinia to Orbis",
      startMapId: 101000300,
      stagingMapId: 101000301,
      enRouteMapIds: [200090010],
      destinationMapId: 200000100,
      observationMapId: 101000302,
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

// Deliberately a different time-of-day than `trips` so a lane showing this
// schedule can never be mistaken for a lane showing `trips`.
const partnerTrips: TripSchedule[] = [
  {
    id: "pt1",
    attributes: {
      boardingOpen: "2023-01-01T03:00:00Z",
      boardingClosed: "2023-01-01T03:05:00Z",
      departure: "2023-01-01T03:07:00Z",
      arrival: "2023-01-01T03:17:00Z",
    },
  },
];

function renderDetail() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <MemoryRouter initialEntries={["/transports/routes/r1"]}>
      <QueryClientProvider client={client}>
        <Routes>
          <Route
            path="/transports/routes/:routeId"
            element={<TransportRouteDetailPage />}
          />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("TransportRouteDetailPage", () => {
  beforeEach(() => {
    vi.mocked(transportsService.getScheduledRoute).mockReset();
    vi.mocked(transportsService.getScheduledRoutes).mockReset();
    vi.mocked(transportsService.getVessels).mockReset();
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([]);
    vi.mocked(transportsService.getVessels).mockResolvedValue([]);
  });

  it("shows the route name, state pill and countdown", async () => {
    vi.mocked(transportsService.getScheduledRoute).mockResolvedValue({
      route: scheduledRoute("open_entry"),
      schedule: trips,
    });

    renderDetail();

    expect(
      await screen.findByRole("heading", { name: "Orbis to Ellinia" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Boarding")).toBeInTheDocument();
    expect(screen.getByText("closes in")).toBeInTheDocument();
  });

  it("renders the map chain and the key/value strip", async () => {
    vi.mocked(transportsService.getScheduledRoute).mockResolvedValue({
      route: scheduledRoute("open_entry"),
      schedule: trips,
    });

    renderDetail();

    await screen.findByRole("heading", { name: "Orbis to Ellinia" });
    expect(screen.getByText("map:200000100")).toBeInTheDocument();
    expect(screen.getByText("map:101000300")).toBeInTheDocument();
    expect(screen.getByText("Trips scheduled today")).toBeInTheDocument();
    expect(screen.getByText("1")).toBeInTheDocument();
  });

  it("never renders a schedule date", async () => {
    // The clock is pinned for the same reason as the tick test below: the
    // time axis labels round wall-clock hours, and adjacent `HH:MM` labels
    // concatenate in `textContent`. Run for real at 23:2x UTC and the axis
    // emits "23:20" then "23:30" — "23:2023:30" — which contains a literal
    // "2023" that has nothing to do with a rendered date. Pinning makes the
    // axis deterministic so the year check below means what it says.
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-06T12:00:00Z"));

    try {
      vi.mocked(transportsService.getScheduledRoute).mockResolvedValue({
        route: scheduledRoute("open_entry"),
        schedule: trips,
      });

      const { container } = renderDetail();

      await vi.waitFor(() => {
        expect(
          screen.getByRole("heading", { name: "Orbis to Ellinia" }),
        ).toBeInTheDocument();
      });

      const text = container.textContent ?? "";
      // Date *shapes*, not one hard-coded year: the fixture's 2023 dates must
      // not surface in any format the page might reach for.
      expect(text).not.toMatch(/\d{4}-\d{2}-\d{2}/); // ISO
      expect(text).not.toMatch(/\d{1,2}\/\d{1,2}\/\d{2,4}/); // locale short
      expect(text).not.toMatch(
        /\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)/,
      ); // month name

      // A bare year has no separator to key off, so the clock-derived labels
      // — axis ticks and the NOW marker, all pure `HH:MM(:SS)` by
      // construction — come out first. Otherwise their concatenation is
      // indistinguishable from a rendered year.
      const clockless = container.cloneNode(true) as HTMLElement;
      clockless
        .querySelectorAll("[data-axis-tick], [data-now-label]")
        .forEach((node) => node.remove());
      expect(clockless.textContent ?? "").not.toContain("2023");
    } finally {
      vi.useRealTimers();
    }
  });

  it("replaces the timeline with a fault message when there are no trips", async () => {
    vi.mocked(transportsService.getScheduledRoute).mockResolvedValue({
      route: scheduledRoute("out_of_service"),
      schedule: [],
    });

    renderDetail();

    await screen.findByText("Orbis to Ellinia");
    expect(screen.getByText(/no trips were scheduled/i)).toBeInTheDocument();
    expect(screen.getByText(/scheduler drops any trip/i)).toBeInTheDocument();
  });

  it("names the unresolved-vessel cause when the route's vessel does not pair", async () => {
    vi.mocked(transportsService.getScheduledRoute).mockResolvedValue({
      route: scheduledRoute("out_of_service"),
      schedule: [],
    });
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([
      scheduledRoute("out_of_service"),
    ]);
    vi.mocked(transportsService.getVessels).mockResolvedValue([
      {
        id: "boat",
        attributes: {
          uuid: "u",
          name: "Boat",
          routeAID: "Orbis to Ellinia",
          routeBID: "Missing Route",
          turnaroundDelay: 60,
        },
      },
    ]);

    renderDetail();

    await screen.findByText("Orbis to Ellinia");
    expect(
      await screen.findByText(/vessel .*does not resolve/i),
    ).toBeInTheDocument();
  });

  it("assembles two distinct lanes when the vessel's partner route resolves with a schedule", async () => {
    const own = scheduledRoute("open_entry");
    const partner = partnerRoute("open_entry");

    vi.mocked(transportsService.getScheduledRoute).mockImplementation(
      (routeId: string) =>
        routeId === "r2"
          ? Promise.resolve({ route: partner, schedule: partnerTrips })
          : Promise.resolve({ route: own, schedule: trips }),
    );
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([
      own,
      partner,
    ]);
    vi.mocked(transportsService.getVessels).mockResolvedValue([
      {
        id: "vessel1",
        attributes: {
          uuid: "u",
          name: "Ferry",
          routeAID: own.attributes.name,
          routeBID: partner.attributes.name,
          turnaroundDelay: 60,
        },
      },
    ]);

    renderDetail();

    await screen.findByRole("heading", { name: "Orbis to Ellinia" });
    const figure = await screen.findByRole("img", { name: /trip timeline/i });

    // VesselTimeline's SVG height is TOP_PAD + lanes.length *
    // (LANE_LABEL_HEIGHT + LANE_HEIGHT + LANE_GAP) + AXIS_HEIGHT = 18 + n*61
    // + 22 — 162 is only reachable with exactly two lanes, so this is an
    // objective count, not just a text-presence guess.
    expect(figure).toHaveAttribute("viewBox", "0 0 720 162");

    const ariaLabel = figure.getAttribute("aria-label") ?? "";
    // Content, not just count: each lane's own trip time must be present,
    // proving the partner's *distinct* schedule reached the second lane
    // rather than the own route's schedule being duplicated into it.
    expect(ariaLabel).toContain("Orbis to Ellinia: boards 11:45");
    expect(ariaLabel).toContain("Ellinia to Orbis: boards 03:00");
  });

  it("renders only the own lane while the partner's schedule is still in flight", async () => {
    const own = scheduledRoute("open_entry");
    const partner = partnerRoute("open_entry");

    vi.mocked(transportsService.getScheduledRoute).mockImplementation(
      (routeId: string) =>
        routeId === "r2"
          ? new Promise(() => {
              /* the partner's detail query never settles in this test */
            })
          : Promise.resolve({ route: own, schedule: trips }),
    );
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([
      own,
      partner,
    ]);
    vi.mocked(transportsService.getVessels).mockResolvedValue([
      {
        id: "vessel1",
        attributes: {
          uuid: "u",
          name: "Ferry",
          routeAID: own.attributes.name,
          routeBID: partner.attributes.name,
          turnaroundDelay: 60,
        },
      },
    ]);

    renderDetail();

    await screen.findByRole("heading", { name: "Orbis to Ellinia" });
    const figure = screen.getByRole("img", { name: /trip timeline/i });

    // Single-lane height (18 + 1*61 = 79): the partner is resolved (it's in
    // `routes`) but its schedule hasn't arrived, so the page must fall back
    // to the solo-lane form rather than rendering a half-built second lane.
    expect(figure).toHaveAttribute("viewBox", "0 0 720 101");
    expect(figure.getAttribute("aria-label") ?? "").not.toContain(
      "Ellinia to Orbis",
    );
  });

  it("resolves the partner correctly even when the viewed route is routeB, not routeA", async () => {
    const own = scheduledRoute("open_entry");
    const partner = partnerRoute("open_entry");

    vi.mocked(transportsService.getScheduledRoute).mockImplementation(
      (routeId: string) =>
        routeId === "r2"
          ? Promise.resolve({ route: partner, schedule: partnerTrips })
          : Promise.resolve({ route: own, schedule: trips }),
    );
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([
      own,
      partner,
    ]);
    vi.mocked(transportsService.getVessels).mockResolvedValue([
      {
        id: "vessel1",
        attributes: {
          uuid: "u",
          name: "Ferry",
          // Swapped vs. the two tests above: the viewed route ("r1") is
          // routeB here, its partner is routeA. A self-comparison bug that
          // only excludes whichever side happens to be routeA (instead of
          // comparing both sides' ids against the viewed route) would put
          // the viewed route into both lanes.
          routeAID: partner.attributes.name,
          routeBID: own.attributes.name,
          turnaroundDelay: 60,
        },
      },
    ]);

    renderDetail();

    await screen.findByRole("heading", { name: "Orbis to Ellinia" });
    const figure = await screen.findByRole("img", { name: /trip timeline/i });

    expect(figure).toHaveAttribute("viewBox", "0 0 720 162");
    const ariaLabel = figure.getAttribute("aria-label") ?? "";
    expect(ariaLabel).toContain("Orbis to Ellinia: boards 11:45");
    expect(ariaLabel).toContain("Ellinia to Orbis: boards 03:00");

    // The guard itself: the viewed route's own trip time must appear
    // exactly once. Two occurrences would mean the "partner" resolved back
    // to the viewed route, duplicating it into both lanes.
    const ownOccurrences = ariaLabel.split("boards 11:45").length - 1;
    expect(ownOccurrences).toBe(1);
  });

  it("shows a distinct loading state before the route resolves", () => {
    vi.mocked(transportsService.getScheduledRoute).mockReturnValue(
      new Promise(() => {
        /* never resolves */
      }),
    );

    renderDetail();

    expect(screen.getByText("Loading route…")).toBeInTheDocument();
    expect(screen.queryByText("Orbis to Ellinia")).not.toBeInTheDocument();
    expect(screen.queryByTestId("error-display")).not.toBeInTheDocument();
  });

  it("shows a distinct error state on a generic fetch failure", async () => {
    vi.mocked(transportsService.getScheduledRoute).mockRejectedValue({
      message: "Internal Server Error",
      statusCode: 500,
      code: "SERVER_ERROR",
    });

    renderDetail();

    expect(await screen.findByTestId("error-display")).toBeInTheDocument();
    expect(screen.getByText("Internal Server Error")).toBeInTheDocument();
    expect(screen.queryByText(/route not found/i)).not.toBeInTheDocument();
    expect(screen.queryByText("Orbis to Ellinia")).not.toBeInTheDocument();
  });

  it("falls back to a generic message when the error carries none", async () => {
    vi.mocked(transportsService.getScheduledRoute).mockRejectedValue({
      message: "",
      statusCode: 0,
      code: "NETWORK_ERROR",
    });

    renderDetail();

    expect(await screen.findByTestId("error-display")).toBeInTheDocument();
    expect(screen.getByText("Failed to load this route.")).toBeInTheDocument();
  });

  it("shows a distinct route-not-found state on a 404", async () => {
    vi.mocked(transportsService.getScheduledRoute).mockRejectedValue({
      message: "route r1 not found",
      statusCode: 404,
      code: "NOT_FOUND",
    });

    renderDetail();

    expect(await screen.findByTestId("error-display")).toBeInTheDocument();
    expect(screen.getByText("Route not found")).toBeInTheDocument();
    expect(
      screen.getByText(/this route no longer exists/i),
    ).toBeInTheDocument();
    expect(screen.queryByText("Orbis to Ellinia")).not.toBeInTheDocument();
  });

  it("ticks the timeline's now marker off the shared clock", async () => {
    // Fake timers must be installed *before* the page mounts: the shared
    // clock (src/lib/utils/clock.ts) starts its one setInterval on the first
    // subscriber, and a fake-timer switch after that point can't reach in and
    // fake an interval that a real `setInterval` already registered.
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-06T12:00:00Z"));

    try {
      vi.mocked(transportsService.getScheduledRoute).mockResolvedValue({
        route: scheduledRoute("open_entry"),
        schedule: trips,
      });

      renderDetail();

      // vi.waitFor advances fake timers itself while it polls, so the
      // query's queryFn promise (a microtask) still resolves and the page
      // renders, without needing real wall-clock time.
      await vi.waitFor(() => {
        expect(
          screen.getByRole("heading", { name: "Orbis to Ellinia" }),
        ).toBeInTheDocument();
      });

      const initialLabel = screen
        .getByRole("img", { name: /trip timeline/i })
        .getAttribute("aria-label");
      expect(initialLabel).toContain("Now 12:00");

      act(() => {
        vi.advanceTimersByTime(65_000);
      });

      const updatedLabel = screen
        .getByRole("img", { name: /trip timeline/i })
        .getAttribute("aria-label");
      expect(updatedLabel).not.toBe(initialLabel);
      expect(updatedLabel).toContain("Now 12:01");
    } finally {
      vi.useRealTimers();
    }
  });
});
