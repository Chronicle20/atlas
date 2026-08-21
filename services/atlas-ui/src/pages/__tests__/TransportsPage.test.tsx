import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { TransportsPage } from "@/pages/TransportsPage";
import type { Tenant } from "@/types/models/tenant";
import type { ScheduledRoute, RouteState } from "@/types/models/transport";
import { transportsService } from "@/services/api/transports.service";

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
    getScheduledRoutes: vi.fn(),
    getScheduledRoute: vi.fn(),
    getInstanceRoutes: vi.fn(),
    getInstanceStatuses: vi.fn(),
    getVessels: vi.fn(),
  },
}));

function scheduledRoute(
  name: string,
  state: RouteState,
  overrides: Partial<ScheduledRoute["attributes"]> = {},
): ScheduledRoute {
  return {
    id: `route-${name}`,
    attributes: {
      name,
      startMapId: 104000000,
      stagingMapId: 104000001,
      enRouteMapIds: [104000002],
      destinationMapId: 200000100,
      observationMapId: 104000003,
      state,
      boardingWindowSeconds: 300,
      preDepartureSeconds: 120,
      travelDurationSeconds: 600,
      cycleIntervalSeconds: 900,
      nextTransitionAt: "",
      nextState: "",
      ...overrides,
    },
  };
}

function renderPage(initialEntry = "/transports") {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <QueryClientProvider client={client}>
        <TransportsPage />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("TransportsPage", () => {
  beforeEach(() => {
    vi.mocked(transportsService.getScheduledRoutes).mockReset();
    vi.mocked(transportsService.getInstanceRoutes).mockReset();
    vi.mocked(transportsService.getInstanceStatuses).mockReset();
    vi.mocked(transportsService.getVessels).mockReset();
    vi.mocked(transportsService.getInstanceRoutes).mockResolvedValue([]);
    vi.mocked(transportsService.getInstanceStatuses).mockResolvedValue([]);
    vi.mocked(transportsService.getVessels).mockResolvedValue([]);
  });

  it("lists scheduled routes with a state pill and map cells", async () => {
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([
      scheduledRoute("Orbis to Ellinia", "open_entry"),
    ]);

    renderPage();

    expect(await screen.findByText("Orbis to Ellinia")).toBeInTheDocument();
    expect(screen.getByText("Boarding")).toBeInTheDocument();
    expect(screen.getByText("map:104000000")).toBeInTheDocument();
    expect(screen.getByText("map:200000100")).toBeInTheDocument();
  });

  it("sorts out_of_service above every other state", async () => {
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([
      scheduledRoute("Aaa Working", "open_entry"),
      scheduledRoute("Zzz Broken", "out_of_service"),
    ]);

    renderPage();

    await screen.findByText("Zzz Broken");
    const names = screen
      .getAllByRole("row")
      .map((row) => row.textContent ?? "")
      .filter((text) => text.includes("Broken") || text.includes("Working"));

    expect(names[0]).toContain("Zzz Broken");
    expect(names[1]).toContain("Aaa Working");
  });

  it("shows an em dash for an out_of_service route's next change", async () => {
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([
      scheduledRoute("Broken", "out_of_service"),
    ]);

    renderPage();

    await screen.findByText("Broken");
    expect(screen.getAllByText("—").length).toBeGreaterThan(0);
  });

  it("links a route name to its detail page", async () => {
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([
      scheduledRoute("Orbis to Ellinia", "open_entry"),
    ]);

    renderPage();

    const link = await screen.findByRole("link", { name: "Orbis to Ellinia" });
    expect(link).toHaveAttribute(
      "href",
      "/transports/routes/route-Orbis to Ellinia",
    );
  });

  it("defaults to the Scheduled tab and reflects a selection in the URL", async () => {
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([]);

    renderPage();

    const scheduledTab = await screen.findByRole("tab", { name: /Scheduled/ });
    expect(scheduledTab).toHaveAttribute("aria-selected", "true");
  });

  it("honours ?tab= on load", async () => {
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([]);

    renderPage("/transports?tab=instance");

    await waitFor(() =>
      expect(screen.getByRole("tab", { name: /Instance/ })).toHaveAttribute(
        "aria-selected",
        "true",
      ),
    );
  });

  it("carries a count on each tab label", async () => {
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([
      scheduledRoute("A", "open_entry"),
      scheduledRoute("B", "open_entry"),
    ]);

    renderPage();

    expect(
      await screen.findByRole("tab", { name: /Scheduled\s*2/ }),
    ).toBeInTheDocument();
  });

  it("carries the instance count before the Instance tab is ever opened", async () => {
    // Radix unmounts inactive tab panels, so a count published upward by the
    // Instance tab's own table would read 0 on the Scheduled tab and only
    // correct itself once the tab was clicked.
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([]);
    vi.mocked(transportsService.getInstanceRoutes).mockResolvedValue([
      { id: "i1", attributes: {} },
      { id: "i2", attributes: {} },
      { id: "i3", attributes: {} },
    ] as unknown as Awaited<
      ReturnType<typeof transportsService.getInstanceRoutes>
    >);

    renderPage();

    expect(
      await screen.findByRole("tab", { name: /Instance\s*3/ }),
    ).toBeInTheDocument();
    // Still on Scheduled — the count did not require visiting the tab.
    expect(screen.getByRole("tab", { name: /Scheduled/ })).toHaveAttribute(
      "aria-selected",
      "true",
    );
  });

  it("shows a loading message while the scheduled routes fetch is in flight", async () => {
    // A promise that never resolves keeps the query in its initial loading state.
    vi.mocked(transportsService.getScheduledRoutes).mockImplementation(
      () => new Promise(() => {}),
    );

    renderPage();

    expect(
      await screen.findByText(/loading scheduled routes/i),
    ).toBeInTheDocument();
    expect(screen.queryByText(/no scheduled routes configured/i)).toBeNull();
    expect(screen.queryByText(/failed to load scheduled routes/i)).toBeNull();
  });

  it("shows an unambiguous error message when the scheduled routes fetch fails", async () => {
    vi.mocked(transportsService.getScheduledRoutes).mockRejectedValue(
      new Error("network down"),
    );

    renderPage();

    expect(
      await screen.findByText(/failed to load scheduled routes/i),
    ).toBeInTheDocument();
    // Must not read as "nothing configured" — that's a different state.
    expect(screen.queryByText(/no scheduled routes configured/i)).toBeNull();
  });

  it("shows a plain empty message when zero scheduled routes are configured", async () => {
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([]);

    renderPage();

    expect(
      await screen.findByText(/no scheduled routes configured/i),
    ).toBeInTheDocument();
    // Must not read as a failure — that's a different state.
    expect(screen.queryByText(/failed to load/i)).toBeNull();
  });

  it("offers a refresh control on the empty scheduled tab", async () => {
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([]);
    const user = userEvent.setup();

    renderPage();

    await screen.findByText(/no scheduled routes configured/i);
    const refreshButton = screen.getByTestId("empty-state-refresh");

    await user.click(refreshButton);

    await waitFor(() => {
      expect(transportsService.getScheduledRoutes).toHaveBeenCalledTimes(2);
    });
    // Only the header FreshnessIndicator renders a freshness readout — the
    // EmptyState does not duplicate it (D7: no lastUpdatedAt here).
    expect(screen.getAllByText(/updated \d+s ago/i)).toHaveLength(1);
  });
});
