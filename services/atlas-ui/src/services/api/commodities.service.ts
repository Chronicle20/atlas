import { fetchAll, fetchPaged } from "@/services/api/pagination";
import type { ItemCashShopCommodity } from "@/types/models/npc";

/** atlas-rest caps page[size] here (libs/atlas-rest .../paginate/params.go). */
const MAX_PAGE_SIZE = 250;

interface CommodityData {
  id: string;
  attributes: {
    itemId: number;
    count: number;
    price: number;
    period: number;
    priority: number;
    gender: number;
    onSale: boolean;
  };
}

function toModel(row: CommodityData): ItemCashShopCommodity {
  return {
    id: row.id,
    itemId: row.attributes.itemId,
    count: row.attributes.count,
    price: row.attributes.price,
    period: row.attributes.period,
    priority: row.attributes.priority,
    gender: row.attributes.gender,
    onSale: row.attributes.onSale,
  };
}

export const commoditiesService = {
  /**
   * Drain the whole cash-shop catalog — every commodity, for the active
   * tenant's version. A commodity's JSON:API id IS its SERIAL NUMBER
   * (atlas-data commodity/reader.go reads it from the `SN` node), which is
   * what a coupon's CASH_ITEM reward names.
   *
   * Callers want the catalog, not a page, because the only useful questions
   * are set questions: "does this item have a serial at all?" and "which
   * serials does it have?". Answering those per item via
   * `/data/commodity/by-item/{id}` would be far worse — that handler drains
   * every commodity server-side on EVERY call and filters in Go
   * (commodity/resource.go#handleGetCommoditiesByItemRequest), so one drain
   * here replaces N full server-side drains there.
   *
   * Page 1 is fetched first to learn the page count, then the remainder go
   * out in parallel — the shared `fetchAll` walks pages serially, which is
   * ~36 sequential round trips at the live catalog's size (8,941 rows).
   */
  async drainAll(): Promise<ItemCashShopCommodity[]> {
    const page = { number: 1, size: MAX_PAGE_SIZE };
    const first = await fetchPaged<CommodityData>(
      "/api/data/commodity/items",
      page,
    );
    // No envelope means the endpoint is unpaginated: that response is all of it.
    if (first.meta === null) return first.data.map(toModel);

    const lastPage = first.meta.page.last;
    const rest = await Promise.all(
      Array.from({ length: Math.max(0, lastPage - 1) }, (_, i) =>
        fetchPaged<CommodityData>("/api/data/commodity/items", {
          number: i + 2,
          size: MAX_PAGE_SIZE,
        }),
      ),
    );

    return [first, ...rest].flatMap((result) => result.data.map(toModel));
  },

  /**
   * Get every cash shop commodity that sells the given item, draining all
   * pages (task-117) — the widget renders the full list, not a page at a time.
   */
  async getByItem(itemId: string | number): Promise<ItemCashShopCommodity[]> {
    const rows = await fetchAll<CommodityData>(
      `/api/data/commodity/by-item/${itemId}`,
    );
    return rows.map(toModel);
  },
};
