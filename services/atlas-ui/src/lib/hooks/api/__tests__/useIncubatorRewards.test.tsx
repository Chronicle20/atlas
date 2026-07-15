import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { useIncubatorRewards, useCreateIncubatorReward } from "../useIncubatorRewards";
import { incubatorRewardsService } from "@/services/api/incubator-rewards.service";

vi.mock("@/services/api/incubator-rewards.service", () => ({
  incubatorRewardsService: { list: vi.fn(), create: vi.fn(), update: vi.fn(), remove: vi.fn(), seed: vi.fn() },
}));

function wrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: ReactNode }) => createElement(QueryClientProvider, { client: qc }, children);
}

describe("useIncubatorRewards", () => {
  beforeEach(() => vi.clearAllMocks());

  it("fetches the reward list for a tenant", async () => {
    (incubatorRewardsService.list as any).mockResolvedValue([{ id: "r1", attributes: { itemId: 1, quantity: 1, weight: 5 } }]);
    const { result } = renderHook(() => useIncubatorRewards("t1"), { wrapper: wrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(incubatorRewardsService.list).toHaveBeenCalledWith("t1");
    expect(result.current.data).toHaveLength(1);
  });

  it("create mutation calls the service", async () => {
    (incubatorRewardsService.create as any).mockResolvedValue({ id: "r2", attributes: { itemId: 2, quantity: 1, weight: 3 } });
    const { result } = renderHook(() => useCreateIncubatorReward(), { wrapper: wrapper() });
    await result.current.mutateAsync({ tenantId: "t1", attributes: { itemId: 2, quantity: 1, weight: 3 } });
    expect(incubatorRewardsService.create).toHaveBeenCalledWith("t1", { itemId: 2, quantity: 1, weight: 3 });
  });
});
