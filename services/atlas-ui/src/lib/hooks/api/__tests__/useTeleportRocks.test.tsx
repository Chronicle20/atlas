import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";

import {
  useTeleportRockMaps,
  useAddTeleportRockMap,
  useRemoveTeleportRockMap,
  teleportRockKeys,
} from "../useTeleportRocks";
import {
  teleportRocksService,
  type TeleportRockLists,
} from "@/services/api/teleport-rocks.service";
import type { Tenant } from "@/types/models/tenant";

const mockTenant: Tenant = {
  id: "tenant-1",
  attributes: {
    name: "Test Tenant",
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
  },
} as unknown as Tenant;

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

vi.mock("@/services/api/teleport-rocks.service", () => ({
  teleportRocksService: {
    getByCharacterId: vi.fn(),
    addMap: vi.fn(),
    removeMap: vi.fn(),
  },
}));

const mockTeleportRocksService = vi.mocked(teleportRocksService);

const mockLists: TeleportRockLists = {
  regular: [101000000],
  vip: [200000100],
  regularCapacity: 10,
  vipCapacity: 15,
};

const mockListsAfterAdd: TeleportRockLists = {
  regular: [101000000, 102000000],
  vip: [200000100],
  regularCapacity: 10,
  vipCapacity: 15,
};

const mockListsAfterRemove: TeleportRockLists = {
  regular: [101000000],
  vip: [],
  regularCapacity: 10,
  vipCapacity: 15,
};

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });

  const Wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  Wrapper.displayName = "QueryClientWrapper";

  return { Wrapper, queryClient };
}

describe("useTeleportRocks hooks", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("useTeleportRockMaps", () => {
    it("does not fetch when tenant is null", () => {
      const { Wrapper } = createWrapper();
      const { result } = renderHook(() => useTeleportRockMaps(null, "1"), {
        wrapper: Wrapper,
      });

      expect(result.current.isFetching).toBe(false);
      expect(mockTeleportRocksService.getByCharacterId).not.toHaveBeenCalled();
    });

    it("does not fetch when characterId is empty", () => {
      const { Wrapper } = createWrapper();
      const { result } = renderHook(() => useTeleportRockMaps(mockTenant, ""), {
        wrapper: Wrapper,
      });

      expect(result.current.isFetching).toBe(false);
      expect(mockTeleportRocksService.getByCharacterId).not.toHaveBeenCalled();
    });

    it("fetches and returns the flattened lists when tenant and characterId are present", async () => {
      mockTeleportRocksService.getByCharacterId.mockResolvedValueOnce(
        mockLists,
      );
      const { Wrapper } = createWrapper();

      const { result } = renderHook(
        () => useTeleportRockMaps(mockTenant, "1"),
        { wrapper: Wrapper },
      );

      await waitFor(() => expect(result.current.isSuccess).toBe(true));

      expect(mockTeleportRocksService.getByCharacterId).toHaveBeenCalledWith(
        "1",
      );
      expect(result.current.data).toEqual(mockLists);
    });
  });

  describe("useAddTeleportRockMap", () => {
    it("writes the returned list into the read hook's cache key on success", async () => {
      mockTeleportRocksService.getByCharacterId.mockResolvedValueOnce(
        mockLists,
      );
      mockTeleportRocksService.addMap.mockResolvedValueOnce(mockListsAfterAdd);

      const { queryClient } = createWrapper();
      const WrapperWithClient = ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      );

      // Prime the cache the same way the read hook would.
      const { result: readResult } = renderHook(
        () => useTeleportRockMaps(mockTenant, "1"),
        { wrapper: WrapperWithClient },
      );
      await waitFor(() => expect(readResult.current.isSuccess).toBe(true));
      expect(readResult.current.data).toEqual(mockLists);

      const { result: mutateResult } = renderHook(
        () => useAddTeleportRockMap(),
        { wrapper: WrapperWithClient },
      );

      mutateResult.current.mutate({
        characterId: "1",
        list: "regular",
        mapId: 102000000,
      });

      await waitFor(() => expect(mutateResult.current.isSuccess).toBe(true));

      expect(mockTeleportRocksService.addMap).toHaveBeenCalledWith(
        "1",
        "regular",
        102000000,
      );

      expect(
        queryClient.getQueryData(teleportRockKeys.detail(mockTenant.id, "1")),
      ).toEqual(mockListsAfterAdd);

      // The read hook re-renders against the same cache entry.
      await waitFor(() =>
        expect(readResult.current.data).toEqual(mockListsAfterAdd),
      );
    });
  });

  describe("useRemoveTeleportRockMap", () => {
    it("writes the returned list into the read hook's cache key on success", async () => {
      mockTeleportRocksService.getByCharacterId.mockResolvedValueOnce(
        mockLists,
      );
      mockTeleportRocksService.removeMap.mockResolvedValueOnce(
        mockListsAfterRemove,
      );

      const { queryClient } = createWrapper();
      const WrapperWithClient = ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      );

      const { result: readResult } = renderHook(
        () => useTeleportRockMaps(mockTenant, "1"),
        { wrapper: WrapperWithClient },
      );
      await waitFor(() => expect(readResult.current.isSuccess).toBe(true));
      expect(readResult.current.data).toEqual(mockLists);

      const { result: mutateResult } = renderHook(
        () => useRemoveTeleportRockMap(),
        { wrapper: WrapperWithClient },
      );

      mutateResult.current.mutate({
        characterId: "1",
        list: "vip",
        mapId: 200000100,
      });

      await waitFor(() => expect(mutateResult.current.isSuccess).toBe(true));

      expect(mockTeleportRocksService.removeMap).toHaveBeenCalledWith(
        "1",
        "vip",
        200000100,
      );

      expect(
        queryClient.getQueryData(teleportRockKeys.detail(mockTenant.id, "1")),
      ).toEqual(mockListsAfterRemove);

      await waitFor(() =>
        expect(readResult.current.data).toEqual(mockListsAfterRemove),
      );
    });
  });

  describe("teleportRockKeys", () => {
    it("generates correct query keys", () => {
      expect(teleportRockKeys.all).toEqual(["teleport-rocks"]);
      expect(teleportRockKeys.detail("tenant-1", "1")).toEqual([
        "teleport-rocks",
        "tenant-1",
        "1",
      ]);
    });
  });
});
