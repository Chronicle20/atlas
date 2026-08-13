import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { InstanceRoutesTable } from "@/components/features/transports/InstanceRoutesTable";
import { transportsService } from "@/services/api/transports.service";
import type { Tenant } from "@/types/models/tenant";
import type { InstanceRoute, InstanceStatus } from "@/types/models/transport";

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
    getInstanceRoutes: vi.fn(),
    getInstanceStatuses: vi.fn(),
  },
}));

const route: InstanceRoute = {
  id: "ir1",
  attributes: {
    name: "Ereve Sky Ferry",
    startMapId: 130000000,
    transitMapIds: [130000010],
    destinationMapId: 130000200,
    capacity: 10,
    boardingWindowSeconds: 60,
    travelDurationSeconds: 120,
  },
};

function status(
  overrides: Partial<InstanceStatus["attributes"]> = {},
): InstanceStatus {
  return {
    id: "11111111-2222-3333-4444-555555555555",
    attributes: {
      routeId: "ir1",
      state: "boarding",
      characters: 3,
      boardingUntil: "2026-08-06T12:01:00Z",
      arrivalAt: "2026-08-06T12:03:00Z",
      createdAt: "2026-08-06T12:00:00Z",
      ...overrides,
    },
  };
}

function renderTable() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <InstanceRoutesTable tenant={mockTenant} />
    </QueryClientProvider>,
  );
}

describe("InstanceRoutesTable", () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    vi.setSystemTime(new Date("2026-08-06T12:00:30Z"));
    vi.mocked(transportsService.getInstanceRoutes).mockReset();
    vi.mocked(transportsService.getInstanceStatuses).mockReset();
    vi.mocked(transportsService.getInstanceRoutes).mockResolvedValue([route]);
  });

  it("lists every instance route with its capacity and durations", async () => {
    vi.mocked(transportsService.getInstanceStatuses).mockResolvedValue([]);

    renderTable();

    expect(await screen.findByText("Ereve Sky Ferry")).toBeInTheDocument();
    expect(screen.getByText("10")).toBeInTheDocument();
    expect(screen.getByText("map:130000000")).toBeInTheDocument();
  });

  it("renders zero live instances as a plain 0 with no expander", async () => {
    vi.mocked(transportsService.getInstanceStatuses).mockResolvedValue([]);

    renderTable();

    await screen.findByText("Ereve Sky Ferry");
    expect(screen.queryByRole("button", { name: /expand/i })).toBeNull();
  });

  it("expands a route with live instances to per-instance rows", async () => {
    vi.mocked(transportsService.getInstanceStatuses).mockResolvedValue([
      status(),
    ]);

    renderTable();

    const expander = await screen.findByRole("button", { name: /expand/i });
    await userEvent.click(expander);

    // truncated instance id
    expect(screen.getByText(/11111111/)).toBeInTheDocument();
    // character count
    expect(screen.getByText("3")).toBeInTheDocument();
  });

  it("flags an instance past two thirds of its route's MaxLifetime", async () => {
    // MaxLifetime = 2 * (60 + 120) = 360s; two thirds = 240s.
    vi.setSystemTime(new Date("2026-08-06T12:05:00Z"));
    vi.mocked(transportsService.getInstanceStatuses).mockResolvedValue([
      status({ state: "in_transit" }),
    ]);

    renderTable();

    const expander = await screen.findByRole("button", { name: /expand/i });
    await userEvent.click(expander);

    expect(screen.getByText(/approaching stuck timeout/i)).toBeInTheDocument();
  });

  it("shows a loading row while the instance routes fetch is in flight", async () => {
    // A promise that never resolves keeps the query in its initial loading state.
    vi.mocked(transportsService.getInstanceRoutes).mockImplementation(
      () => new Promise(() => {}),
    );

    renderTable();

    expect(
      await screen.findByText(/loading instance routes/i),
    ).toBeInTheDocument();
    expect(screen.queryByText("Ereve Sky Ferry")).toBeNull();
  });

  it("shows an unambiguous error row when the instance routes fetch fails", async () => {
    vi.mocked(transportsService.getInstanceRoutes).mockRejectedValue(
      new Error("network down"),
    );

    renderTable();

    expect(
      await screen.findByText(/failed to load instance routes/i),
    ).toBeInTheDocument();
    // Must not read as "nothing configured" — that's a different state.
    expect(screen.queryByText(/no instance routes configured/i)).toBeNull();
  });

  it("shows a plain empty row when zero instance routes are configured", async () => {
    vi.mocked(transportsService.getInstanceRoutes).mockResolvedValue([]);
    vi.mocked(transportsService.getInstanceStatuses).mockResolvedValue([]);

    renderTable();

    expect(
      await screen.findByText(/no instance routes configured/i),
    ).toBeInTheDocument();
    // Must not read as a failure — that's a different state.
    expect(screen.queryByText(/failed to load/i)).toBeNull();
  });
});
