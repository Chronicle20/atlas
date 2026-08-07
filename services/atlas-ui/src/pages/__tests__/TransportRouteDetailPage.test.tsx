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
    vi.mocked(transportsService.getScheduledRoute).mockResolvedValue({
      route: scheduledRoute("open_entry"),
      schedule: trips,
    });

    const { container } = renderDetail();

    await screen.findByRole("heading", { name: "Orbis to Ellinia" });
    expect(container.textContent).not.toContain("2023");
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
