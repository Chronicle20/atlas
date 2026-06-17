import { describe, it, expect, vi, beforeEach } from "vitest";
import { mtsListingsService, buildBrowseListingsQuery } from "@/services/api/mts-listings.service";
import { api } from "@/lib/api/client";

vi.mock("@/lib/api/client", () => ({
  api: { getList: vi.fn() },
}));

describe("buildBrowseListingsQuery", () => {
  it("emits an empty string when no filters are set", () => {
    expect(buildBrowseListingsQuery({})).toBe("");
  });

  it("emits flat (non-bracketed) query params for each provided filter", () => {
    expect(
      buildBrowseListingsQuery({
        category: "equip",
        subCategory: "weapon",
        saleType: "AUCTION",
        sellerName: "Alice",
        itemId: 1302000,
        page: 2,
        pageSize: 16,
      }),
    ).toBe(
      "?category=equip&subCategory=weapon&saleType=AUCTION&sellerName=Alice&itemId=1302000&page=2&pageSize=16",
    );
  });

  it("omits itemId when it is zero", () => {
    expect(buildBrowseListingsQuery({ itemId: 0, page: 1, pageSize: 16 })).toBe(
      "?page=1&pageSize=16",
    );
  });
});

describe("mtsListingsService.browse", () => {
  beforeEach(() => vi.clearAllMocks());

  it("hits the per-world listings endpoint with the built query", async () => {
    (api.getList as ReturnType<typeof vi.fn>).mockResolvedValue([]);
    await mtsListingsService.browse(0, { saleType: "BUY_NOW", page: 1, pageSize: 16 });
    expect(api.getList).toHaveBeenCalledWith(
      "/api/worlds/0/listings?saleType=BUY_NOW&page=1&pageSize=16",
      undefined,
    );
  });

  it("returns the listing array from the JSON:API list response", async () => {
    const listings = [{ id: "l1", attributes: { templateId: 1302000 } }];
    (api.getList as ReturnType<typeof vi.fn>).mockResolvedValue(listings);
    const res = await mtsListingsService.browse(1, {});
    expect(api.getList).toHaveBeenCalledWith("/api/worlds/1/listings", undefined);
    expect(res).toBe(listings);
  });
});
