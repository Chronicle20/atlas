import { describe, it, expect, vi, beforeEach } from "vitest";
import { rewardPoolsService } from "../reward-pools.service";
import { api } from "@/lib/api/client";
import { fetchAll } from "@/services/api/pagination";

vi.mock("@/lib/api/client", () => ({
  api: { getOne: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() },
}));
vi.mock("@/services/api/pagination", () => ({
  fetchAll: vi.fn().mockResolvedValue([]),
}));

describe("rewardPoolsService", () => {
  beforeEach(() => vi.clearAllMocks());

  it("getAllPools drains /api/gachapons", async () => {
    await rewardPoolsService.getAllPools();
    expect(fetchAll).toHaveBeenCalledWith(
      "/api/gachapons",
      undefined,
      undefined,
    );
  });

  it("createPool posts an id-carrying JSON:API envelope for incubators", async () => {
    await rewardPoolsService.createPool("4170001", {
      name: "Pigmy Egg (Victoria)",
      kind: "incubator",
      npcIds: [1012004],
      commonWeight: 0,
      uncommonWeight: 0,
      rareWeight: 0,
    });
    expect(api.post).toHaveBeenCalledWith("/api/gachapons", {
      data: {
        id: "4170001",
        type: "gachapons",
        attributes: expect.objectContaining({ kind: "incubator" }),
      },
    });
  });

  it("createPool omits id when not supplied", async () => {
    await rewardPoolsService.createPool(undefined, {
      name: "Henesys",
      kind: "gachapon",
      npcIds: [9100100],
      commonWeight: 70,
      uncommonWeight: 25,
      rareWeight: 5,
    });
    const body = (api.post as ReturnType<typeof vi.fn>).mock.calls[0]![1];
    expect(body.data.id).toBeUndefined();
  });

  it("updatePool PATCHes with envelope", async () => {
    const attrs = {
      name: "Henesys",
      kind: "gachapon" as const,
      npcIds: [9100100],
      commonWeight: 60,
      uncommonWeight: 30,
      rareWeight: 10,
    };
    await rewardPoolsService.updatePool("henesys", attrs);
    expect(api.patch).toHaveBeenCalledWith("/api/gachapons/henesys", {
      data: { id: "henesys", type: "gachapons", attributes: attrs },
    });
  });

  it("item CRUD targets the nested collection", async () => {
    await rewardPoolsService.getItems("4170001");
    expect(fetchAll).toHaveBeenCalledWith("/api/gachapons/4170001/items");

    await rewardPoolsService.createItem("4170001", {
      itemId: 2000000,
      quantity: 1,
      tier: "common",
      weight: 50,
    });
    expect(api.post).toHaveBeenCalledWith("/api/gachapons/4170001/items", {
      data: {
        type: "gachapon-items",
        attributes: {
          itemId: 2000000,
          quantity: 1,
          tier: "common",
          weight: 50,
        },
      },
    });

    await rewardPoolsService.updateItem("4170001", "12", {
      itemId: 2000001,
      quantity: 2,
      tier: "common",
      weight: 75,
    });
    expect(api.patch).toHaveBeenCalledWith("/api/gachapons/4170001/items/12", {
      data: {
        id: "12",
        type: "gachapon-items",
        attributes: {
          itemId: 2000001,
          quantity: 2,
          tier: "common",
          weight: 75,
        },
      },
    });

    await rewardPoolsService.removeItem("4170001", "12");
    expect(api.delete).toHaveBeenCalledWith("/api/gachapons/4170001/items/12");
  });

  it("global item CRUD targets /api/global-items", async () => {
    await rewardPoolsService.createGlobalItem({
      itemId: 2000000,
      quantity: 1,
      tier: "common",
    });
    expect(api.post).toHaveBeenCalledWith("/api/global-items", {
      data: {
        type: "global-gachapon-items",
        attributes: { itemId: 2000000, quantity: 1, tier: "common" },
      },
    });
    await rewardPoolsService.removeGlobalItem("3");
    expect(api.delete).toHaveBeenCalledWith("/api/global-items/3");
  });
});
