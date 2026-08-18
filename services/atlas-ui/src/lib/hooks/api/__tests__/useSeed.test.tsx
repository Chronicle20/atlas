import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import {
  useDropsSeedStatus,
  useGachaponsSeedStatus,
  useNpcConversationsSeedStatus,
  useQuestConversationsSeedStatus,
  useItemConversationsSeedStatus,
  useNpcShopsSeedStatus,
  usePortalScriptsSeedStatus,
  useReactorScriptsSeedStatus,
  useMapActionScriptsSeedStatus,
  useTransportRoutesSeedStatus,
  useTransportVesselsSeedStatus,
  useInstanceRoutesSeedStatus,
  useWzInputStatus,
  useDataStatus,
  useSeedDrops,
  useSeedGachapons,
  useSeedNpcConversations,
  useSeedQuestConversations,
  useSeedItemConversations,
  useSeedNpcShops,
  useSeedPortalScripts,
  useSeedReactorScripts,
  useSeedMapActionScripts,
  useSeedTransportRoutes,
  useSeedTransportVessels,
  useSeedInstanceRoutes,
} from "../useSeed";
import { seedService } from "@/services/api/seed.service";
import * as tenantContext from "@/context/tenant-context";

vi.mock("@/services/api/seed.service", () => ({
  seedService: {
    getDropsSeedStatus: vi.fn(),
    getGachaponsSeedStatus: vi.fn(),
    getNpcConversationsSeedStatus: vi.fn(),
    getQuestConversationsSeedStatus: vi.fn(),
    getItemConversationsSeedStatus: vi.fn(),
    getNpcShopsSeedStatus: vi.fn(),
    getPortalScriptsSeedStatus: vi.fn(),
    getReactorScriptsSeedStatus: vi.fn(),
    getMapActionScriptsSeedStatus: vi.fn(),
    getTransportRoutesSeedStatus: vi.fn(),
    getTransportVesselsSeedStatus: vi.fn(),
    getInstanceRoutesSeedStatus: vi.fn(),
    getWzInputStatus: vi.fn(),
    getDataStatus: vi.fn(),
    seedDrops: vi.fn(),
    seedGachapons: vi.fn(),
    seedNpcConversations: vi.fn(),
    seedQuestConversations: vi.fn(),
    seedItemConversations: vi.fn(),
    seedNpcShops: vi.fn(),
    seedPortalScripts: vi.fn(),
    seedReactorScripts: vi.fn(),
    seedMapActionScripts: vi.fn(),
    seedRoutes: vi.fn(),
    seedVessels: vi.fn(),
    seedInstanceRoutes: vi.fn(),
  },
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: vi.fn(),
}));

const fakeTenant = {
  id: "tenant-1",
  type: "tenants",
  attributes: {
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
    name: "Tenant 1",
  },
};

function makeWrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return {
    qc,
    wrapper: ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={qc}>{children}</QueryClientProvider>
    ),
  };
}

beforeEach(() => vi.clearAllMocks());

describe.each([
  [
    "useDropsSeedStatus",
    useDropsSeedStatus,
    "getDropsSeedStatus",
    "dropsSeedStatus",
  ],
  [
    "useGachaponsSeedStatus",
    useGachaponsSeedStatus,
    "getGachaponsSeedStatus",
    "gachaponsSeedStatus",
  ],
  [
    "useNpcConversationsSeedStatus",
    useNpcConversationsSeedStatus,
    "getNpcConversationsSeedStatus",
    "npcConversationsSeedStatus",
  ],
  [
    "useQuestConversationsSeedStatus",
    useQuestConversationsSeedStatus,
    "getQuestConversationsSeedStatus",
    "questConversationsSeedStatus",
  ],
  [
    "useItemConversationsSeedStatus",
    useItemConversationsSeedStatus,
    "getItemConversationsSeedStatus",
    "itemConversationsSeedStatus",
  ],
  [
    "useNpcShopsSeedStatus",
    useNpcShopsSeedStatus,
    "getNpcShopsSeedStatus",
    "npcShopsSeedStatus",
  ],
  [
    "usePortalScriptsSeedStatus",
    usePortalScriptsSeedStatus,
    "getPortalScriptsSeedStatus",
    "portalScriptsSeedStatus",
  ],
  [
    "useReactorScriptsSeedStatus",
    useReactorScriptsSeedStatus,
    "getReactorScriptsSeedStatus",
    "reactorScriptsSeedStatus",
  ],
  [
    "useMapActionScriptsSeedStatus",
    useMapActionScriptsSeedStatus,
    "getMapActionScriptsSeedStatus",
    "mapActionScriptsSeedStatus",
  ],
  [
    "useTransportRoutesSeedStatus",
    useTransportRoutesSeedStatus,
    "getTransportRoutesSeedStatus",
    "transportRoutesSeedStatus",
  ],
  [
    "useTransportVesselsSeedStatus",
    useTransportVesselsSeedStatus,
    "getTransportVesselsSeedStatus",
    "transportVesselsSeedStatus",
  ],
  [
    "useInstanceRoutesSeedStatus",
    useInstanceRoutesSeedStatus,
    "getInstanceRoutesSeedStatus",
    "instanceRoutesSeedStatus",
  ],
] as const)("%s", (_, hook, method, key) => {
  it("enables polling and keys by tenant id when a tenant is active", async () => {
    (
      tenantContext.useTenant as unknown as ReturnType<typeof vi.fn>
    ).mockReturnValue({
      activeTenant: fakeTenant,
    });
    (seedService as unknown as Record<string, ReturnType<typeof vi.fn>>)[
      method
    ]!.mockResolvedValue({
      updatedAt: null,
    });

    const { wrapper, qc } = makeWrapper();
    const { result } = renderHook(() => hook(), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(
      (seedService as unknown as Record<string, ReturnType<typeof vi.fn>>)[
        method
      ],
    ).toHaveBeenCalledWith(fakeTenant);
    expect(qc.getQueryData([key, fakeTenant.id])).toBeDefined();
  });

  it("disables polling when no tenant is active", () => {
    (
      tenantContext.useTenant as unknown as ReturnType<typeof vi.fn>
    ).mockReturnValue({
      activeTenant: null,
    });

    const { wrapper } = makeWrapper();
    const { result } = renderHook(() => hook(), { wrapper });

    expect(result.current.fetchStatus).toBe("idle");
    expect(
      (seedService as unknown as Record<string, ReturnType<typeof vi.fn>>)[
        method
      ],
    ).not.toHaveBeenCalled();
  });
});

describe.each([
  ["useWzInputStatus", useWzInputStatus, "getWzInputStatus", "wzInputStatus"],
  ["useDataStatus", useDataStatus, "getDataStatus", "dataStatus"],
] as const)("%s", (_, hook, method, keyRoot) => {
  it("calls the service with only the tenant and keys the cache by tenant id", async () => {
    (
      tenantContext.useTenant as unknown as ReturnType<typeof vi.fn>
    ).mockReturnValue({
      activeTenant: fakeTenant,
    });
    (seedService as unknown as Record<string, ReturnType<typeof vi.fn>>)[
      method
    ]!.mockResolvedValue({
      updatedAt: null,
    });

    const { wrapper, qc } = makeWrapper();
    const { result } = renderHook(() => hook(), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(
      (seedService as unknown as Record<string, ReturnType<typeof vi.fn>>)[
        method
      ],
    ).toHaveBeenCalledWith(fakeTenant);
    expect(qc.getQueryData([keyRoot, fakeTenant.id])).toBeDefined();
  });
});

describe.each([
  ["useSeedDrops", useSeedDrops, "seedDrops", "dropsSeedStatus"],
  [
    "useSeedGachapons",
    useSeedGachapons,
    "seedGachapons",
    "gachaponsSeedStatus",
  ],
  [
    "useSeedNpcConversations",
    useSeedNpcConversations,
    "seedNpcConversations",
    "npcConversationsSeedStatus",
  ],
  [
    "useSeedQuestConversations",
    useSeedQuestConversations,
    "seedQuestConversations",
    "questConversationsSeedStatus",
  ],
  [
    "useSeedItemConversations",
    useSeedItemConversations,
    "seedItemConversations",
    "itemConversationsSeedStatus",
  ],
  ["useSeedNpcShops", useSeedNpcShops, "seedNpcShops", "npcShopsSeedStatus"],
  [
    "useSeedPortalScripts",
    useSeedPortalScripts,
    "seedPortalScripts",
    "portalScriptsSeedStatus",
  ],
  [
    "useSeedReactorScripts",
    useSeedReactorScripts,
    "seedReactorScripts",
    "reactorScriptsSeedStatus",
  ],
  [
    "useSeedMapActionScripts",
    useSeedMapActionScripts,
    "seedMapActionScripts",
    "mapActionScriptsSeedStatus",
  ],
] as const)("%s mutation", (_, hook, method, statusKeyRoot) => {
  it("invalidates the matching status key on success", async () => {
    (
      tenantContext.useTenant as unknown as ReturnType<typeof vi.fn>
    ).mockReturnValue({
      activeTenant: fakeTenant,
    });
    (seedService as unknown as Record<string, ReturnType<typeof vi.fn>>)[
      method
    ]!.mockResolvedValue(undefined);

    const { wrapper, qc } = makeWrapper();
    const invalidate = vi.spyOn(qc, "invalidateQueries");
    const { result } = renderHook(() => hook(), { wrapper });

    result.current.mutate();
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(invalidate).toHaveBeenCalledWith({
      queryKey: [statusKeyRoot, fakeTenant.id],
    });
  });
});

describe("transport seed mutations", () => {
  it("useSeedTransportRoutes posts to the routes seed endpoint", async () => {
    const spy = vi
      .spyOn(seedService, "seedRoutes")
      .mockResolvedValue(undefined);
    const { result } = renderHook(() => useSeedTransportRoutes(), {
      wrapper: makeWrapper().wrapper,
    });
    result.current.mutate();
    await waitFor(() => expect(spy).toHaveBeenCalledTimes(1));
  });

  it("useSeedTransportVessels posts to the vessels seed endpoint", async () => {
    const spy = vi
      .spyOn(seedService, "seedVessels")
      .mockResolvedValue(undefined);
    const { result } = renderHook(() => useSeedTransportVessels(), {
      wrapper: makeWrapper().wrapper,
    });
    result.current.mutate();
    await waitFor(() => expect(spy).toHaveBeenCalledTimes(1));
  });

  it("useSeedInstanceRoutes posts to the instance-routes seed endpoint", async () => {
    const spy = vi
      .spyOn(seedService, "seedInstanceRoutes")
      .mockResolvedValue(undefined);
    const { result } = renderHook(() => useSeedInstanceRoutes(), {
      wrapper: makeWrapper().wrapper,
    });
    result.current.mutate();
    await waitFor(() => expect(spy).toHaveBeenCalledTimes(1));
  });
});
