import { act, render, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { TenantProvider, useTenant } from "@/context/tenant-context";
import type { Tenant } from "@/types/models/tenant";

const setTenantMock = vi.fn();
const getAllTenantsMock = vi.fn();

vi.mock("@/lib/api/client", () => ({
  api: {
    setTenant: (...args: unknown[]) => setTenantMock(...args),
  },
}));

vi.mock("@/services/api", () => ({
  tenantsService: {
    getAllTenants: () => getAllTenantsMock(),
    getTenantConfigurationById: () => Promise.resolve({}),
  },
}));

function makeTenant(id: string, region = "GMS"): Tenant {
  return {
    id,
    type: "tenants",
    attributes: {
      region,
      majorVersion: 83,
      minorVersion: 1,
      name: `Tenant ${id}`,
    },
  } as unknown as Tenant;
}

function Harness({
  onReady,
}: {
  onReady: (ctx: ReturnType<typeof useTenant>) => void;
}) {
  const ctx = useTenant();
  onReady(ctx);
  return null;
}

describe("TenantProvider tenant-switch invariants", () => {
  beforeEach(() => {
    setTenantMock.mockReset();
    getAllTenantsMock.mockReset();
    localStorage.clear();
  });

  it("sets api.setTenant synchronously and clears the cache once per distinct-id switch", async () => {
    const tenantA = makeTenant("aaa");
    const tenantB = makeTenant("bbb");
    getAllTenantsMock.mockResolvedValueOnce([]);

    const queryClient = new QueryClient();
    const clearSpy = vi.spyOn(queryClient, "clear");

    let ctxRef: ReturnType<typeof useTenant> | undefined;
    render(
      <QueryClientProvider client={queryClient}>
        <TenantProvider>
          <Harness
            onReady={(c) => {
              ctxRef = c;
            }}
          />
        </TenantProvider>
      </QueryClientProvider>,
    );

    await waitFor(() => {
      expect(ctxRef).toBeDefined();
    });

    // Initial mount with activeTenant === null (empty tenant list) fires neither hook.
    expect(setTenantMock).not.toHaveBeenCalled();
    expect(clearSpy).not.toHaveBeenCalled();

    // Switch to tenant A. Asserting INSIDE the act callback (before React commits
    // and the catch-all effect runs) proves headers are set + cache cleared
    // SYNCHRONOUSLY, ahead of the re-render that recomputes child query keys
    // (FR-2.1 / FR-2.4 — closes task-004 R6).
    act(() => {
      ctxRef!.setActiveTenant(tenantA);
      expect(setTenantMock).toHaveBeenCalledWith(tenantA);
      expect(clearSpy).toHaveBeenCalledTimes(1);
    });
    // After commit, the catch-all effect re-applies the same id (no extra clear).
    expect(setTenantMock).toHaveBeenLastCalledWith(tenantA);
    expect(clearSpy).toHaveBeenCalledTimes(1);

    // Switch to tenant B — second distinct id → second clear.
    act(() => {
      ctxRef!.setActiveTenant(tenantB);
      expect(setTenantMock).toHaveBeenLastCalledWith(tenantB);
      expect(clearSpy).toHaveBeenCalledTimes(2);
    });
    expect(setTenantMock).toHaveBeenLastCalledWith(tenantB);
    expect(clearSpy).toHaveBeenCalledTimes(2);
  });

  it("rehydrates activeTenant on refresh when its attributes change; does not clear cache", async () => {
    const tenantA = makeTenant("aaa");
    const tenantARenamed: Tenant = {
      ...tenantA,
      attributes: { ...tenantA.attributes, name: "Renamed A" },
    } as unknown as Tenant;

    // Initial load returns [tenantA]; refresh returns [tenantARenamed].
    getAllTenantsMock.mockResolvedValueOnce([tenantA]);
    getAllTenantsMock.mockResolvedValueOnce([tenantARenamed]);

    const queryClient = new QueryClient();
    const clearSpy = vi.spyOn(queryClient, "clear");

    let ctxRef: ReturnType<typeof useTenant> | undefined;
    render(
      <QueryClientProvider client={queryClient}>
        <TenantProvider>
          <Harness
            onReady={(c) => {
              ctxRef = c;
            }}
          />
        </TenantProvider>
      </QueryClientProvider>,
    );

    // Wait for initial load to pick tenantA as active.
    await waitFor(() => {
      expect(ctxRef?.activeTenant?.id).toBe("aaa");
    });
    expect(ctxRef?.activeTenant?.attributes.name).toBe("Tenant aaa");

    const clearCallsBefore = clearSpy.mock.calls.length;

    // Refresh; tenantA is still present but with a new name.
    await act(async () => {
      await ctxRef!.refreshTenants();
    });

    await waitFor(() => {
      expect(ctxRef?.activeTenant?.attributes.name).toBe("Renamed A");
    });
    // Same id → id-compare effect must not trigger another cache clear.
    expect(clearSpy).toHaveBeenCalledTimes(clearCallsBefore);
  });

  it("reselects when active tenant was removed from the refreshed list", async () => {
    const tenantA = makeTenant("aaa");
    const tenantB = makeTenant("bbb");

    // Initial load returns [A, B] so A is selected; refresh drops A.
    getAllTenantsMock.mockResolvedValueOnce([tenantA, tenantB]);
    getAllTenantsMock.mockResolvedValueOnce([tenantB]);

    const queryClient = new QueryClient();

    let ctxRef: ReturnType<typeof useTenant> | undefined;
    render(
      <QueryClientProvider client={queryClient}>
        <TenantProvider>
          <Harness
            onReady={(c) => {
              ctxRef = c;
            }}
          />
        </TenantProvider>
      </QueryClientProvider>,
    );

    await waitFor(() => {
      expect(ctxRef?.activeTenant?.id).toBe("aaa");
    });

    await act(async () => {
      await ctxRef!.refreshTenants();
    });

    await waitFor(() => {
      expect(ctxRef?.activeTenant?.id).toBe("bbb");
    });
  });
});
