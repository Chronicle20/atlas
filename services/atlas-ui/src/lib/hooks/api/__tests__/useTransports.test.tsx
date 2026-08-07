import { vi, beforeEach, describe, expect, it } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";

import type { Tenant } from "@/types/models/tenant";
import {
  transportKeys,
  useInstanceStatuses,
  useScheduledRoute,
  useScheduledRoutes,
  useVessels,
  TRANSPORT_POLL_MS,
} from "../useTransports";
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

let activeTenant: Tenant | null = mockTenant;

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant,
    tenants: [mockTenant],
    loading: false,
    setActiveTenant: vi.fn(),
    refreshTenants: vi.fn(),
    refreshAndSelectTenant: vi.fn(),
    fetchTenantConfiguration: vi.fn(),
  }),
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

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe("useTransports", () => {
  beforeEach(() => {
    activeTenant = mockTenant;
    vi.mocked(transportsService.getScheduledRoutes).mockReset();
    vi.mocked(transportsService.getScheduledRoute).mockReset();
    vi.mocked(transportsService.getInstanceStatuses).mockReset();
    vi.mocked(transportsService.getVessels).mockReset();
  });

  it("polls the scheduled routes every 30 seconds", () => {
    expect(TRANSPORT_POLL_MS).toBe(30_000);
  });

  it("scopes every key to the active tenant", () => {
    expect(transportKeys.scheduled("tenant-1")).toContain("tenant-1");
    expect(transportKeys.scheduledDetail("tenant-1", "r1")).toContain("r1");
    expect(transportKeys.instanceStatus("tenant-1", "ir1")).toContain("ir1");
    expect(transportKeys.vessels("tenant-1")).toContain("tenant-1");
  });

  it("fetches scheduled routes when a tenant is active", async () => {
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([]);

    const { result } = renderHook(() => useScheduledRoutes(), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(transportsService.getScheduledRoutes).toHaveBeenCalled();
  });

  it("does not fetch without an active tenant", () => {
    activeTenant = null;
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([]);

    const { result } = renderHook(() => useScheduledRoutes(), { wrapper });

    expect(result.current.fetchStatus).toBe("idle");
    expect(transportsService.getScheduledRoutes).not.toHaveBeenCalled();
  });

  it("does not fetch a route detail without a routeId", () => {
    vi.mocked(transportsService.getScheduledRoute).mockResolvedValue({
      route: { id: "r1", attributes: {} as never },
      schedule: [],
    });

    const { result } = renderHook(() => useScheduledRoute(""), { wrapper });

    expect(result.current.fetchStatus).toBe("idle");
  });

  it("fans out one status query per instance route", async () => {
    vi.mocked(transportsService.getInstanceStatuses).mockResolvedValue([]);

    const { result } = renderHook(
      () => useInstanceStatuses(["ir1", "ir2", "ir3"]),
      { wrapper },
    );

    await waitFor(() => expect(result.current).toHaveLength(3));
    await waitFor(() =>
      expect(transportsService.getInstanceStatuses).toHaveBeenCalledTimes(3),
    );
    expect(transportsService.getInstanceStatuses).toHaveBeenCalledWith("ir2");
  });

  it("reads vessels from the active tenant's configuration", async () => {
    vi.mocked(transportsService.getVessels).mockResolvedValue([]);

    const { result } = renderHook(() => useVessels(), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(transportsService.getVessels).toHaveBeenCalledWith("tenant-1");
  });
});
