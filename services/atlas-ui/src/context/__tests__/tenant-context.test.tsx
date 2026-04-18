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

function Harness({ onReady }: { onReady: (ctx: ReturnType<typeof useTenant>) => void }) {
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

  it("invokes api.setTenant and queryClient.clear on tenant switch (not on initial null mount)", async () => {
    const tenantA = makeTenant("aaa");
    const tenantB = makeTenant("bbb");
    getAllTenantsMock.mockResolvedValueOnce([]);

    const queryClient = new QueryClient();
    const clearSpy = vi.spyOn(queryClient, "clear");

    let ctxRef: ReturnType<typeof useTenant> | undefined;
    render(
      <QueryClientProvider client={queryClient}>
        <TenantProvider>
          <Harness onReady={(c) => { ctxRef = c; }} />
        </TenantProvider>
      </QueryClientProvider>
    );

    await waitFor(() => {
      expect(ctxRef).toBeDefined();
    });

    // Initial mount with activeTenant === null must not fire either hook.
    expect(setTenantMock).not.toHaveBeenCalled();
    expect(clearSpy).not.toHaveBeenCalled();

    // Switch to tenant A
    act(() => {
      ctxRef!.setActiveTenant(tenantA);
    });
    await waitFor(() => {
      expect(setTenantMock).toHaveBeenCalledTimes(1);
    });
    expect(setTenantMock).toHaveBeenLastCalledWith(tenantA);
    expect(clearSpy).toHaveBeenCalledTimes(1);

    // Switch to tenant B
    act(() => {
      ctxRef!.setActiveTenant(tenantB);
    });
    await waitFor(() => {
      expect(setTenantMock).toHaveBeenCalledTimes(2);
    });
    expect(setTenantMock).toHaveBeenLastCalledWith(tenantB);
    expect(clearSpy).toHaveBeenCalledTimes(2);
  });
});
