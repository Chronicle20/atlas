import { api } from "@/lib/api/client";
import { fetchAll } from "@/services/api/pagination";
import type { ItemCashShopCommodity } from "@/types/models/npc";

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
   * Get one commodity by SERIAL NUMBER. A commodity's JSON:API id IS its SN
   * (atlas-data commodity/reader.go reads it from the `SN` node), which is
   * also what a coupon's CASH_ITEM reward names — atlas-cashshop validates a
   * reward against this very route.
   */
  async getBySerialNumber(
    serialNumber: string | number,
  ): Promise<ItemCashShopCommodity> {
    const row = await api.getOne<CommodityData>(
      `/api/data/commodity/items/${serialNumber}`,
    );
    return toModel(row);
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
