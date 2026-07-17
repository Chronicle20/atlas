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

export const commoditiesService = {
  /**
   * Get every cash shop commodity that sells the given item, draining all
   * pages (task-117) — the widget renders the full list, not a page at a time.
   */
  async getByItem(itemId: string | number): Promise<ItemCashShopCommodity[]> {
    const rows = await fetchAll<CommodityData>(`/api/data/commodity/by-item/${itemId}`);
    return rows.map((row) => ({
      id: row.id,
      itemId: row.attributes.itemId,
      count: row.attributes.count,
      price: row.attributes.price,
      period: row.attributes.period,
      priority: row.attributes.priority,
      gender: row.attributes.gender,
      onSale: row.attributes.onSale,
    }));
  },
};
