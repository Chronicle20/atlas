import { describe, it, expect, vi, beforeEach } from "vitest";
import { incubatorRewardsService } from "../incubator-rewards.service";
import { api } from "@/lib/api/client";

vi.mock("@/lib/api/client", () => ({
  api: { getList: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() },
}));

describe("incubatorRewardsService", () => {
  const t = "tenant-1";
  beforeEach(() => vi.clearAllMocks());

  it("list GETs the tenant collection", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (api.getList as any).mockResolvedValue([]);
    await incubatorRewardsService.list(t);
    expect(api.getList).toHaveBeenCalledWith(`/api/tenants/${t}/configurations/incubator-rewards`, undefined);
  });

  it("create POSTs a JSON:API envelope", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (api.post as any).mockResolvedValue({ id: "r1", attributes: { itemId: 2000000, quantity: 1, weight: 50 } });
    await incubatorRewardsService.create(t, { itemId: 2000000, quantity: 1, weight: 50 });
    expect(api.post).toHaveBeenCalledWith(
      `/api/tenants/${t}/configurations/incubator-rewards`,
      { data: { type: "incubator-rewards", attributes: { itemId: 2000000, quantity: 1, weight: 50 } } },
      undefined,
    );
  });

  it("update PATCHes by id with the envelope", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (api.patch as any).mockResolvedValue(undefined);
    await incubatorRewardsService.update(t, "r1", { itemId: 3, quantity: 2, weight: 10 });
    expect(api.patch).toHaveBeenCalledWith(
      `/api/tenants/${t}/configurations/incubator-rewards/r1`,
      { data: { id: "r1", type: "incubator-rewards", attributes: { itemId: 3, quantity: 2, weight: 10 } } },
      undefined,
    );
  });

  it("remove DELETEs by id", async () => {
    await incubatorRewardsService.remove(t, "r1");
    expect(api.delete).toHaveBeenCalledWith(`/api/tenants/${t}/configurations/incubator-rewards/r1`, undefined);
  });

  it("seed POSTs the seed endpoint", async () => {
    await incubatorRewardsService.seed(t);
    expect(api.post).toHaveBeenCalledWith(`/api/tenants/${t}/configurations/incubator-rewards/seed`, {}, undefined);
  });
});
