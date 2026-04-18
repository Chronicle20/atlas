import { api } from "@/lib/api/client";
import { type ServiceOptions, type QueryOptions, type ValidationError } from "@/lib/api/query-params";
import { conversationsService } from "./conversations.service";
import type { NPC, NpcSearchResult, Shop, Commodity, CommodityAttributes, ShopResponse } from "@/types/models/npc";

const BASE_PATH = "/api/npcs";

interface CreateShopInput {
  data: {
    type: "shops";
    id: string;
    attributes: { npcId: number; recharger?: boolean };
    relationships: { commodities: { data: Array<{ type: "commodities"; id: string }> } };
  };
  included: Array<{ type: "commodities"; id: string; attributes: Omit<CommodityAttributes, "id"> }>;
}

interface UpdateShopInput {
  data: {
    type: "shops";
    id: string;
    attributes: { npcId: number; recharger?: boolean };
    relationships: { commodities: { data: Array<{ type: "commodities"; id: string }> } };
  };
  included: Array<{ type: "commodities"; id: string; attributes: CommodityAttributes }>;
}

interface CreateCommodityInput {
  data: { type: "commodities"; attributes: CommodityAttributes };
}

interface UpdateCommodityInput {
  data: { type: "commodities"; attributes: Partial<CommodityAttributes> };
}

function validateCommodity(data: CommodityAttributes): ValidationError[] {
  const errors: ValidationError[] = [];
  if (data.templateId <= 0) errors.push({ field: "templateId", message: "Template ID must be positive" });
  if (data.mesoPrice < 0) errors.push({ field: "mesoPrice", message: "Meso price must be non-negative" });
  if (data.discountRate < 0 || data.discountRate > 100) {
    errors.push({ field: "discountRate", message: "Discount rate must be between 0 and 100" });
  }
  if (data.tokenPrice < 0) errors.push({ field: "tokenPrice", message: "Token price must be non-negative" });
  if (data.period < 0) errors.push({ field: "period", message: "Period must be non-negative" });
  if (data.levelLimit < 0) errors.push({ field: "levelLimit", message: "Level limit must be non-negative" });
  return errors;
}

function throwIfInvalidCommodity(attrs: CommodityAttributes, shouldValidate: boolean): void {
  if (!shouldValidate) return;
  const errors = validateCommodity(attrs);
  if (errors.length > 0) {
    throw new Error(`Commodity validation failed: ${errors.map(e => e.message).join(", ")}`);
  }
}

export const npcsService = {
  /**
   * Combine shop and conversation lookups into a single NPC list.
   */
  async getAllNPCs(options?: QueryOptions): Promise<NPC[]> {
    try {
      const shops = await api.getList<Shop>("/api/shops", options);
      const npcsWithShops: NPC[] = shops.map((shop: Shop) => ({
        id: shop.attributes.npcId,
        hasShop: true,
        hasConversation: false,
      }));

      try {
        const conversations = await conversationsService.getAll();
        const npcsWithConversations: NPC[] = conversations.map(conversation => ({
          id: conversation.attributes.npcId,
          hasShop: false,
          hasConversation: true,
        }));

        const npcMap = new Map<number, NPC>();
        npcsWithShops.forEach(npc => npcMap.set(npc.id, npc));
        npcsWithConversations.forEach(npc => {
          const existing = npcMap.get(npc.id);
          if (existing) existing.hasConversation = true;
          else npcMap.set(npc.id, npc);
        });

        return Array.from(npcMap.values()).sort((a, b) => a.id - b.id);
      } catch (conversationError) {
        console.error("Failed to fetch NPCs with conversations:", conversationError);
        return npcsWithShops.sort((a, b) => a.id - b.id);
      }
    } catch (error) {
      console.error("Failed to fetch NPCs:", error);
      throw new Error("Unable to retrieve NPC data. Please try again later.");
    }
  },

  async searchNpcs(query: string): Promise<NpcSearchResult[]> {
    const npcs = await api.getList<{ id: string; attributes: { name: string } }>(
      `/api/data/npcs?search=${encodeURIComponent(query)}`,
    );
    return npcs.map(npc => ({ id: parseInt(npc.id), name: npc.attributes.name }));
  },

  async getNPCShop(npcId: number, options?: ServiceOptions): Promise<ShopResponse> {
    return api.get<ShopResponse>(`${BASE_PATH}/${npcId}/shop?include=commodities`, options);
  },

  async createShop(
    npcId: number,
    commodities: Omit<CommodityAttributes, "id">[],
    recharger?: boolean,
    options?: ServiceOptions,
  ): Promise<Shop> {
    const shouldValidate = options?.validate !== false;
    for (const commodity of commodities) {
      throwIfInvalidCommodity(commodity as CommodityAttributes, shouldValidate);
    }

    const includedCommodities = commodities.map((commodity, index) => ({
      type: "commodities" as const,
      id: `temp-id-${index}`,
      attributes: commodity,
    }));
    const commodityReferences = includedCommodities.map(c => ({ type: "commodities" as const, id: c.id }));

    const input: CreateShopInput = {
      data: {
        type: "shops",
        id: `shop-${npcId}`,
        attributes: { npcId, ...(recharger !== undefined && { recharger }) },
        relationships: { commodities: { data: commodityReferences } },
      },
      included: includedCommodities,
    };

    const response = await api.post<{ data: Shop }>(`${BASE_PATH}/${npcId}/shop`, input, options);
    return response.data;
  },

  async updateShop(
    npcId: number,
    commodities: Commodity[],
    recharger?: boolean,
    options?: ServiceOptions,
  ): Promise<Shop> {
    const shouldValidate = options?.validate !== false;
    for (const commodity of commodities) {
      throwIfInvalidCommodity(commodity.attributes, shouldValidate);
    }

    const commodityReferences = commodities.map(c => ({ type: "commodities" as const, id: c.id }));
    const includedCommodities = commodities.map(c => ({
      type: "commodities" as const,
      id: c.id,
      attributes: c.attributes,
    }));

    const input: UpdateShopInput = {
      data: {
        type: "shops",
        id: `shop-${npcId}`,
        attributes: { npcId, ...(recharger !== undefined && { recharger }) },
        relationships: { commodities: { data: commodityReferences } },
      },
      included: includedCommodities,
    };

    const response = await api.put<{ data: Shop }>(`${BASE_PATH}/${npcId}/shop`, input, options);
    return response.data;
  },

  async createCommodity(
    npcId: number,
    commodityAttributes: CommodityAttributes,
    options?: ServiceOptions,
  ): Promise<Commodity> {
    const input: CreateCommodityInput = { data: { type: "commodities", attributes: commodityAttributes } };
    const response = await api.post<{ data: Commodity }>(
      `${BASE_PATH}/${npcId}/shop/relationships/commodities`,
      input,
      options,
    );
    return response.data;
  },

  async updateCommodity(
    npcId: number,
    commodityId: string,
    commodityAttributes: Partial<CommodityAttributes>,
    options?: ServiceOptions,
  ): Promise<Commodity> {
    const input: UpdateCommodityInput = { data: { type: "commodities", attributes: commodityAttributes } };
    const response = await api.put<{ data: Commodity }>(
      `${BASE_PATH}/${npcId}/shop/relationships/commodities/${commodityId}`,
      input,
      options,
    );
    return response.data;
  },

  async deleteCommodity(
    npcId: number,
    commodityId: string,
    options?: ServiceOptions,
  ): Promise<void> {
    return api.delete(
      `${BASE_PATH}/${npcId}/shop/relationships/commodities/${commodityId}`,
      options,
    );
  },

  async deleteAllCommoditiesForNPC(
    npcId: number,
    options?: ServiceOptions,
  ): Promise<void> {
    return api.delete(`${BASE_PATH}/${npcId}/shop/relationships/commodities`, options);
  },

  async deleteAllShops(options?: ServiceOptions): Promise<void> {
    return api.delete("/api/shops", options);
  },

  async createCommoditiesBatch(
    npcId: number,
    commodities: CommodityAttributes[],
    options?: ServiceOptions,
  ): Promise<Commodity[]> {
    const results: Commodity[] = [];
    for (const commodity of commodities) {
      try {
        const result = await npcsService.createCommodity(npcId, commodity, options);
        results.push(result);
      } catch (error) {
        console.error(`Failed to create commodity for NPC ${npcId}:`, error);
        throw error;
      }
    }
    return results;
  },

  async getNPCsWithShops(options?: QueryOptions): Promise<NPC[]> {
    const allNPCs = await npcsService.getAllNPCs( options);
    return allNPCs.filter(npc => npc.hasShop);
  },

  async getNPCsWithConversations(options?: QueryOptions): Promise<NPC[]> {
    const allNPCs = await npcsService.getAllNPCs( options);
    return allNPCs.filter(npc => npc.hasConversation);
  },

  async getNPCById(npcId: number, options?: ServiceOptions): Promise<NPC | null> {
    const allNPCs = await npcsService.getAllNPCs( options);
    return allNPCs.find(npc => npc.id === npcId) || null;
  },

  async getNpcName(npcId: number): Promise<string> {
    const npc = await api.getOne<{ id: string; attributes: { name: string } }>(`/api/data/npcs/${npcId}`);
    return npc.attributes.name;
  },
};

export type { NPC, NpcSearchResult, Shop, Commodity, CommodityAttributes, ShopResponse };
