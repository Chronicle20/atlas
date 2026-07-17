import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { useRewardPools, useCreatePoolItem, rewardPoolKeys } from "../useRewardPools";
import { rewardPoolsService } from "@/services/api/reward-pools.service";

vi.mock("@/services/api/reward-pools.service", () => ({
  rewardPoolsService: {
    getAllPools: vi.fn().mockResolvedValue([]),
    createItem: vi.fn().mockResolvedValue(undefined),
  },
}));
vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } }),
}));

function wrapper(qc: QueryClient) {
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  );
}

describe("useRewardPools", () => {
  beforeEach(() => vi.clearAllMocks());

  it("fetches the drained pool list", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result } = renderHook(() => useRewardPools(), { wrapper: wrapper(qc) });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(rewardPoolsService.getAllPools).toHaveBeenCalled();
  });

  it("useCreatePoolItem invalidates the pool's items key", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const spy = vi.spyOn(qc, "invalidateQueries");
    const { result } = renderHook(() => useCreatePoolItem(), { wrapper: wrapper(qc) });
    result.current.mutate({ poolId: "4170001", attributes: { itemId: 2000000, quantity: 1, tier: "common", weight: 50 } });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(spy).toHaveBeenCalledWith({ queryKey: rewardPoolKeys.items("4170001") });
  });
});
