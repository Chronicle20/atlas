import { describe, it, expect, vi, beforeEach } from "vitest";
import type { ReactNode } from "react";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  useCreateTenantConfiguration,
  useUpdateTenantConfiguration,
} from "@/lib/hooks/api/useTenants";
import { socketKeys } from "@/lib/hooks/api/socketKeys";
import {
  tenantsService,
  type TenantConfig,
  type TenantConfigAttributes,
} from "@/services/api/tenants.service";

vi.mock("@/services/api/tenants.service", () => ({
  tenantsService: {
    createTenantConfiguration: vi.fn(),
    updateTenantConfiguration: vi.fn(),
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
  id: "tn1",
  attributes: {
    region: "GMS",
    majorVersion: 95,
    minorVersion: 1,
  } as TenantConfigAttributes,
};

describe("useTenants configuration mutations invalidate socketKeys.all (Packet Matrix)", () => {
  beforeEach(() => vi.clearAllMocks());

  it("useCreateTenantConfiguration invalidates socketKeys.all on success", async () => {
    vi.mocked(tenantsService.createTenantConfiguration).mockResolvedValue(
      tenantConfig,
    );
    const qc = newClient();
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    const { result } = renderHook(() => useCreateTenantConfiguration(), {
      wrapper: makeWrapper(qc),
    });
    await result.current.mutateAsync({
      tenantId: "tenant-1",
      attributes: tenantConfig.attributes,
    });

    await waitFor(() => {
      const keys = invalidate.mock.calls.map((c) => c[0]?.queryKey);
      expect(keys).toContainEqual(socketKeys.all);
    });
  });

  it("useUpdateTenantConfiguration invalidates socketKeys.all on settle", async () => {
    vi.mocked(tenantsService.updateTenantConfiguration).mockResolvedValue(
      tenantConfig,
    );
    const qc = newClient();
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    const { result } = renderHook(() => useUpdateTenantConfiguration(), {
      wrapper: makeWrapper(qc),
    });
    await result.current.mutateAsync({
      tenant: tenantConfig,
      updates: { region: "GMS" },
    });

    await waitFor(() => {
      const keys = invalidate.mock.calls.map((c) => c[0]?.queryKey);
      expect(keys).toContainEqual(socketKeys.all);
    });
  });
});
