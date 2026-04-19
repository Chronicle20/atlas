import { api } from "@/lib/api/client";
import type { ItemSellerCommodity } from "@/types/models/npc";

interface CommodityByItemData {
  id: string;
  attributes: {
    npcId: number;
    templateId: number;
    mesoPrice: number;
    discountRate: number;
    tokenTemplateId: number;
    tokenPrice: number;
    period: number;
    levelLimit: number;
  };
}

export const npcShopCommoditiesService = {
  async getByItem(itemId: string | number): Promise<ItemSellerCommodity[]> {
    const rows = await api.getList<CommodityByItemData>(`/api/commodities/items/${itemId}`);
    return rows.map((row) => ({
      id: row.id,
      npcId: row.attributes.npcId,
      templateId: row.attributes.templateId,
      mesoPrice: row.attributes.mesoPrice,
      discountRate: row.attributes.discountRate,
      tokenTemplateId: row.attributes.tokenTemplateId,
      tokenPrice: row.attributes.tokenPrice,
      period: row.attributes.period,
      levelLimit: row.attributes.levelLimit,
    }));
  },
};
