import { api } from "@/lib/api/client";
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
  async getByItem(itemId: string | number): Promise<ItemCashShopCommodity[]> {
    const rows = await api.getList<CommodityData>(`/api/data/commodity/by-item/${itemId}`);
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
