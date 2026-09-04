import { vi } from "vitest";
/**
 * Tests for useFields React Query hooks
 */

import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";

import {
  useFields,
  useFieldsForMap,
  fieldKeys,
  fieldQueryOptions,
} from "../useFields";
import { fieldsService, type FieldData } from "@/services/api/fields.service";
import type { Tenant } from "@/types/models/tenant";

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

vi.mock("@/services/api/fields.service", () => ({
  fieldsService: {
    getFields: vi.fn(),
  },
}));

const mockFieldsService = vi.mocked(fieldsService);

const mockFieldData: FieldData = {
  id: "0:2:910340000:00000000-0000-0000-0000-000000000000",
  type: "fields",
  attributes: {
    worldId: 0,
    channelId: 2,
    mapId: 910340000,
    instanceId: "00000000-0000-0000-0000-000000000000",
    characterCount: 3,
  },
};

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
    },
  });

  const Wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  Wrapper.displayName = "QueryClientWrapper";

  return Wrapper;
}

describe("useFields hooks", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("useFields calls the endpoint with no filters", async () => {
    mockFieldsService.getFields.mockResolvedValueOnce([mockFieldData]);

    const { result } = renderHook(() => useFields({}), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(mockFieldsService.getFields).toHaveBeenCalledWith({});
    expect(result.current.data).toEqual([mockFieldData]);
  });

  it("useFields passes each filter through", async () => {
    mockFieldsService.getFields.mockResolvedValueOnce([mockFieldData]);

    const filters = { worldId: 0, channelId: 2, mapId: 910340000 };
    const { result } = renderHook(() => useFields(filters), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(mockFieldsService.getFields).toHaveBeenCalledWith(filters);
  });

  it("useFieldsForMap filters by mapId only", async () => {
    mockFieldsService.getFields.mockResolvedValueOnce([mockFieldData]);

    const { result } = renderHook(() => useFieldsForMap("910340000"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(mockFieldsService.getFields).toHaveBeenCalledWith({
      mapId: 910340000,
    });
  });

  it("runtime cache profile", () => {
    const options = fieldQueryOptions({});
    expect(options.staleTime).toBe(5000);
    expect(options.gcTime).toBe(60000);
  });

  it("no polling", () => {
    const options = fieldQueryOptions({}) as { refetchInterval?: unknown };
    expect(options.refetchInterval).toBeUndefined();
  });

  it("key namespace is disjoint from definition", () => {
    const key = fieldKeys.list({});
    expect(key[0]).toBe("fields");
    expect(key[0]).not.toBe("maps");
  });
});
