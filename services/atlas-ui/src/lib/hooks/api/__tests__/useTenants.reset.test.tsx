import { describe, it, expect, vi, beforeEach } from "vitest";
import type { ReactNode } from "react";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  useResetTenantConfiguration,
  tenantKeys,
} from "@/lib/hooks/api/useTenants";
import { socketKeys } from "@/lib/hooks/api/socketKeys";
import {
  tenantsService,
  type TenantConfig,
  type TenantConfigAttributes,
} from "@/services/api/tenants.service";

vi.mock("@/services/api/tenants.service", () => ({
  tenantsService: {
    reset: vi.fn(),
  },
}));

function makeWrapper(qc: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
  };
}

function newClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
}

const tenantConfig: TenantConfig = {
  id: "t1",
  attributes: {
    region: "GMS",
    majorVersion: 95,
    minorVersion: 1,
  } as TenantConfigAttributes,
};

describe("useResetTenantConfiguration", () => {
  beforeEach(() => vi.clearAllMocks());

  it("calls the service with the id and sections", async () => {
    vi.mocked(tenantsService.reset).mockResolvedValue(tenantConfig);
    const qc = newClient();

    const { result } = renderHook(() => useResetTenantConfiguration(), {
      wrapper: makeWrapper(qc),
    });
    await result.current.mutateAsync({ id: "t1", sections: ["socket"] });

    expect(tenantsService.reset).toHaveBeenCalledWith("t1", ["socket"]);
  });

  it("omits sections for a whole-document reset", async () => {
    vi.mocked(tenantsService.reset).mockResolvedValue(tenantConfig);
    const qc = newClient();

    const { result } = renderHook(() => useResetTenantConfiguration(), {
      wrapper: makeWrapper(qc),
    });
    await result.current.mutateAsync({ id: "t1" });

    expect(tenantsService.reset).toHaveBeenCalledWith("t1", undefined);
  });

  it("invalidates the detail, the list, and socket on success", async () => {
    vi.mocked(tenantsService.reset).mockResolvedValue(tenantConfig);
    const qc = newClient();
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    const { result } = renderHook(() => useResetTenantConfiguration(), {
      wrapper: makeWrapper(qc),
    });
    await result.current.mutateAsync({ id: "t1" });

    await waitFor(() => {
      const keys = invalidate.mock.calls.map((c) => c[0]?.queryKey);
      expect(keys).toContainEqual(tenantKeys.configDetail("t1"));
      expect(keys).toContainEqual(tenantKeys.configLists());
      expect(keys).toContainEqual(socketKeys.all);
    });
  });

  it("invalidates nothing on failure", async () => {
    vi.mocked(tenantsService.reset).mockRejectedValue(
      new Error("reset failed"),
    );
    const qc = newClient();
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    const { result } = renderHook(() => useResetTenantConfiguration(), {
      wrapper: makeWrapper(qc),
    });

    await expect(result.current.mutateAsync({ id: "t1" })).rejects.toThrow(
      "reset failed",
    );

    expect(invalidate).not.toHaveBeenCalled();
  });
});
