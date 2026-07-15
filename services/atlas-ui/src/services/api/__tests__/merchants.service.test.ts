import { describe, expect, it, vi, beforeEach } from "vitest";

const getMock = vi.fn();
vi.mock("@/lib/api/client", () => ({
  api: {
    get: (...args: unknown[]) => getMock(...args),
    getList: vi.fn(),
    getOne: vi.fn(),
  },
}));

import { merchantsService } from "@/services/api/merchants.service";
import type { MerchantShop } from "@/types/models/merchant";

function makeShop(id: string): MerchantShop {
  return {
    id,
    attributes: {
      characterId: 1,
      shopType: 1,
      state: 2,
      title: `Shop ${id}`,
      worldId: 0,
      channelId: 0,
      mapId: 100000000,
      instanceId: "",
      x: 0,
      y: 0,
      permitItemId: 0,
      closeReason: 0,
      mesoBalance: 0,
      listingCount: 0,
    },
  };
}

describe("merchantsService.getShopsPage", () => {
  beforeEach(() => vi.clearAllMocks());

  it("appends page params and returns data + meta", async () => {
    getMock.mockResolvedValue({
      data: [makeShop("1")],
      meta: { total: 1, page: { number: 1, size: 50, last: 1 } },
    });

    const result = await merchantsService.getShopsPage({ number: 1, size: 50 });

    expect(result.data).toEqual([makeShop("1")]);
    expect(result.meta).toEqual({ total: 1, page: { number: 1, size: 50, last: 1 } });

    const [calledUrl] = getMock.mock.calls[0] as [string];
    const params = new URL(calledUrl, "http://example.test").searchParams;
    expect(params.get("page[number]")).toBe("1");
    expect(params.get("page[size]")).toBe("50");
  });
});

describe("merchantsService.getAllShops", () => {
  beforeEach(() => vi.clearAllMocks());

  it("drains every page", async () => {
    getMock
      .mockResolvedValueOnce({
        data: [makeShop("1")],
        meta: { total: 2, page: { number: 1, size: 1, last: 2 } },
      })
      .mockResolvedValueOnce({
        data: [makeShop("2")],
        meta: { total: 2, page: { number: 2, size: 1, last: 2 } },
      });

    const result = await merchantsService.getAllShops();

    expect(result.map((s) => s.id)).toEqual(["1", "2"]);
    expect(getMock).toHaveBeenCalledTimes(2);
  });
});

describe("merchantsService.searchListings", () => {
  beforeEach(() => vi.clearAllMocks());

  it("drains every page of listing search results for an item id", async () => {
    getMock.mockResolvedValue({
      data: [],
      meta: { total: 0, page: { number: 1, size: 250, last: 1 } },
    });

    await merchantsService.searchListings(2000000);

    const [calledUrl] = getMock.mock.calls[0] as [string];
    const url = new URL(calledUrl, "http://example.test");
    expect(url.searchParams.get("itemId")).toBe("2000000");
    expect(url.searchParams.get("page[size]")).toBe("250");
  });
});
