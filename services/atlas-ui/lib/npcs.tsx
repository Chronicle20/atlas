import type {Tenant} from "@/types/models/tenant";
import type {NPC, Shop, Commodity, CommodityAttributes, ShopResponse} from "@/types/models/npc";
import { api } from "@/lib/api/client";
import {fetchConversations} from "@/lib/npc-conversations";

export async function fetchNPCs(tenant: Tenant): Promise<NPC[]> {
    // Fetch NPCs with shops
    api.setTenant(tenant);
    const shops = await api.getList<Shop>('/api/shops');

    // Extract NPCs from shops data
    const npcsWithShops = shops.map((shop: Shop) => ({
        id: shop.attributes.npcId,
        hasShop: true,
        hasConversation: false
    }));

    // Fetch NPCs with conversations
    try {
        const conversations = await fetchConversations(tenant);

        // Extract NPCs from conversations data
        const npcsWithConversations = conversations.map(conversation => ({
            id: conversation.attributes.npcId,
            hasShop: false,
            hasConversation: true
        }));

        // Combine NPCs from both sources, avoiding duplicates
        const npcMap = new Map<number, NPC>();

        // Add NPCs with shops
        npcsWithShops.forEach((npc: NPC) => {
            npcMap.set(npc.id, npc);
        });

        // Add or update NPCs with conversations
        npcsWithConversations.forEach(npc => {
            if (npcMap.has(npc.id)) {
                // NPC already exists (has a shop), update to indicate it also has a conversation
                const existingNpc = npcMap.get(npc.id)!;
                existingNpc.hasConversation = true;
            } else {
                // New NPC (only has conversation)
                npcMap.set(npc.id, npc);
            }
        });

        // Convert map back to array
        return Array.from(npcMap.values());
    } catch (error: unknown) {
        console.error("Failed to fetch NPCs with conversations:", error);
        // If fetching conversations fails, return just the NPCs with shops
        return npcsWithShops;
    }
}


export async function fetchNPCShop(tenant: Tenant, npcId: number): Promise<ShopResponse> {
    api.setTenant(tenant);
    return api.get<ShopResponse>(`/api/npcs/${npcId}/shop?include=commodities`);
}

export async function createCommodity(tenant: Tenant, npcId: number, commodityAttributes: CommodityAttributes): Promise<Commodity> {
    api.setTenant(tenant);
    const response = await api.post<{data: Commodity}>(`/api/npcs/${npcId}/shop/relationships/commodities`, {
        data: {
            type: "commodities",
            attributes: commodityAttributes
        }
    });
    return response.data;
}

export async function updateCommodity(tenant: Tenant, npcId: number, commodityId: string, commodityAttributes: Partial<CommodityAttributes>): Promise<Commodity> {
    api.setTenant(tenant);
    const response = await api.put<{data: Commodity}>(`/api/npcs/${npcId}/shop/relationships/commodities/${commodityId}`, {
        data: {
            type: "commodities",
            attributes: commodityAttributes
        }
    });
    return response.data;
}

export async function deleteCommodity(tenant: Tenant, npcId: number, commodityId: string): Promise<void> {
    api.setTenant(tenant);
    return api.delete<void>(`/api/npcs/${npcId}/shop/relationships/commodities/${commodityId}`);
}

export async function createShop(tenant: Tenant, npcId: number, commodities: Omit<CommodityAttributes, 'id'>[], recharger?: boolean): Promise<Shop> {
    api.setTenant(tenant);

    // Create commodity data for included section
    const includedCommodities = commodities.map((commodity, index) => ({
        type: "commodities",
        id: `temp-id-${index}`, // Temporary ID, will be replaced by server
        attributes: commodity
    }));

    // Create commodity references for relationships section
    const commodityReferences = includedCommodities.map(commodity => ({
        type: "commodities",
        id: commodity.id
    }));

    const response = await api.post<{data: Shop}>(`/api/npcs/${npcId}/shop`, {
        data: {
            type: "shops",
            id: `shop-${npcId}`,
            attributes: {
                npcId: npcId,
                recharger: recharger
            },
            relationships: {
                commodities: {
                    data: commodityReferences
                }
            }
        },
        included: includedCommodities
    });
    return response.data;
}

export async function updateShop(tenant: Tenant, npcId: number, commodities: Commodity[], recharger?: boolean): Promise<Shop> {
    api.setTenant(tenant);

    // Create commodity references for relationships section
    const commodityReferences = commodities.map(commodity => ({
        type: "commodities",
        id: commodity.id
    }));

    // Create included commodities
    const includedCommodities = commodities.map(commodity => ({
        type: "commodities",
        id: commodity.id,
        attributes: commodity.attributes
    }));

    const response = await api.put<{data: Shop}>(`/api/npcs/${npcId}/shop`, {
        data: {
            type: "shops",
            id: `shop-${npcId}`,
            attributes: {
                npcId: npcId,
                recharger: recharger
            },
            relationships: {
                commodities: {
                    data: commodityReferences
                }
            }
        },
        included: includedCommodities
    });
    return response.data;
}

export async function deleteAllShops(tenant: Tenant): Promise<void> {
    api.setTenant(tenant);
    return api.delete<void>('/api/shops');
}

export async function deleteAllCommoditiesForNPC(tenant: Tenant, npcId: number): Promise<void> {
    api.setTenant(tenant);
    return api.delete<void>(`/api/npcs/${npcId}/shop/relationships/commodities`);
}
