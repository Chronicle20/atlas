import { describe, it, expect, vi, beforeEach } from "vitest";
import { mtsListingsService, buildBrowseListingsQuery } from "@/services/api/mts-listings.service";
import { apiClient } from "@/lib/api/client";

vi.mock("@/lib/api/client", () => ({
  apiClient: { get: vi.fn() },
  api: {},
}));

describe("buildBrowseListingsQuery", () => {
  it("emits an empty string when no filters are set", () => {
    expect(buildBrowseListingsQuery({})).toBe("");
  });

  it("emits flat filter params plus standard page[number]/page[size] paging params", () => {
    // filter.page is this module's zero-based caller convention (page=1 is the
    // SECOND page); it must render onto the wire's one-based page[number] as
    // page+1 (task-117's off-by-one).
    expect(
      buildBrowseListingsQuery({
        category: "equip",
        subCategory: "weapon",
        saleType: "auction",
        sellerName: "Alice",
        itemId: 1302000,
        page: 1,
        pageSize: 16,
      }),
    ).toBe(
      "?category=equip&subCategory=weapon&saleType=auction&sellerName=Alice&itemId=1302000&page%5Bnumber%5D=2&page%5Bsize%5D=16",
    );
  });

  it("omits itemId when it is zero, and renders page=0 as page[number]=1", () => {
    expect(buildBrowseListingsQuery({ itemId: 0, page: 0, pageSize: 16 })).toBe(
      "?page%5Bnumber%5D=1&page%5Bsize%5D=16",
    );
  });

  it("never emits the legacy bare page/pageSize params", () => {
    const got = buildBrowseListingsQuery({ page: 2, pageSize: 16 });
    expect(got).not.toMatch(/[?&]page=/);
    expect(got).not.toMatch(/[?&]pageSize=/);
  });
});

describe("mtsListingsService.browse", () => {
  beforeEach(() => vi.clearAllMocks());

  it("hits the per-world listings endpoint with the built query", async () => {
    (apiClient.get as ReturnType<typeof vi.fn>).mockResolvedValue({ data: [], meta: { total: 0, page: { last: 1 } } });
    await mtsListingsService.browse(0, { saleType: "auction", page: 0, pageSize: 16 });
    expect(apiClient.get).toHaveBeenCalledWith(
      "/api/worlds/0/listings?saleType=auction&page%5Bnumber%5D=1&page%5Bsize%5D=16",
      undefined,
    );
  });

  it("returns the page with the authoritative total and lastPage from meta", async () => {
    const listings = [{ id: "l1", attributes: { templateId: 1302000 } }];
    (apiClient.get as ReturnType<typeof vi.fn>).mockResolvedValue({
      data: listings,
      meta: { total: 38, page: { number: 0, size: 16, last: 3 } },
    });
    const res = await mtsListingsService.browse(1, { page: 0, pageSize: 16 });
    expect(res.listings).toBe(listings);
    expect(res.total).toBe(38);
    expect(res.lastPage).toBe(3);
  });

  it("falls back to the page length and last page 1 when meta is absent", async () => {
    const listings = [{ id: "l1", attributes: {} }, { id: "l2", attributes: {} }];
    (apiClient.get as ReturnType<typeof vi.fn>).mockResolvedValue({ data: listings });
    const res = await mtsListingsService.browse(0, {});
    expect(res.total).toBe(2);
    expect(res.lastPage).toBe(1);
  });
});
