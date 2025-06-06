import {Tenant} from "@/app/tenants/columns";
import {tenantHeaders} from "@/lib/headers";

export interface CommodityAttributes {
    templateId: number;
    mesoPrice: number;
    discountRate: number;
    tokenItemId: number;
    tokenPrice: number;
    period: number;
    levelLimit: number;
    unitPrice?: number;
    slotMax?: number;
}

export interface Commodity {
    id: string;
    type: string;
    attributes: CommodityAttributes;
}

export interface CommodityReference {
    type: string;
    id: string;
}

export interface Shop {
    type: string;
    id: string;
    attributes: {
        npcId: number;
    };
    relationships?: {
        commodities: {
            data: CommodityReference[];
        };
    };
    included?: Commodity[];
}

export interface NPC {
    id: number;
    hasShop: boolean;
}

export async function fetchNPCs(tenant: Tenant): Promise<NPC[]> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(rootUrl + "/api/shops", {
        method: "GET",
        headers: tenantHeaders(tenant),
    });
    if (!response.ok) {
        throw new Error("Failed to fetch NPCs.");
    }
    const responseData = await response.json();

    // Extract NPCs from shops data
    return responseData.data.map((shop: Shop) => ({
        id: shop.attributes.npcId,
        hasShop: true
    }));
}

export interface ShopResponse {
    data: Shop;
    included?: Commodity[];
}

export async function fetchNPCShop(tenant: Tenant, npcId: number): Promise<ShopResponse> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(rootUrl + "/api/npcs/" + npcId + "/shop?include=commodities", {
        method: "GET",
        headers: tenantHeaders(tenant),
    });
    if (!response.ok) {
        throw new Error("Failed to fetch NPC shop.");
    }
    return await response.json();
}

export async function createCommodity(tenant: Tenant, npcId: number, commodityAttributes: CommodityAttributes): Promise<Commodity> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(rootUrl + "/api/npcs/" + npcId + "/shop/relationships/commodities", {
        method: "POST",
        headers: tenantHeaders(tenant),
        body: JSON.stringify({
            data: {
                type: "commodities",
                attributes: commodityAttributes
            }
        }),
    });
    if (!response.ok) {
        throw new Error("Failed to create commodity.");
    }
    const responseData = await response.json();
    return responseData.data;
}

export async function updateCommodity(tenant: Tenant, npcId: number, commodityId: string, commodityAttributes: Partial<CommodityAttributes>): Promise<Commodity> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(rootUrl + "/api/npcs/" + npcId + "/shop/relationships/commodities/" + commodityId, {
        method: "PUT",
        headers: tenantHeaders(tenant),
        body: JSON.stringify({
            data: {
                type: "commodities",
                attributes: commodityAttributes
            }
        }),
    });
    if (!response.ok) {
        throw new Error("Failed to update commodity.");
    }
    const responseData = await response.json();
    return responseData.data;
}

export async function deleteCommodity(tenant: Tenant, npcId: number, commodityId: string): Promise<void> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(rootUrl + "/api/npcs/" + npcId + "/shop/relationships/commodities/" + commodityId, {
        method: "DELETE",
        headers: tenantHeaders(tenant),
    });
    if (!response.ok) {
        throw new Error("Failed to delete commodity.");
    }
}

export async function createShop(tenant: Tenant, npcId: number, commodities: Omit<CommodityAttributes, 'id'>[]): Promise<Shop> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;

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

    const response = await fetch(rootUrl + "/api/npcs/" + npcId + "/shop", {
        method: "POST",
        headers: tenantHeaders(tenant),
        body: JSON.stringify({
            data: {
                type: "shops",
                id: `shop-${npcId}`,
                attributes: {
                    npcId: npcId
                },
                relationships: {
                    commodities: {
                        data: commodityReferences
                    }
                }
            },
            included: includedCommodities
        }),
    });
    if (!response.ok) {
        throw new Error("Failed to create shop.");
    }
    const responseData = await response.json();
    return responseData.data;
}

export async function bulkCreateShops(tenant: Tenant, shops: { npcId: number, commodities: Omit<CommodityAttributes, 'id'>[] }[]): Promise<Shop[]> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;

    // Transform each shop to the new format
    const transformedShops = shops.map(shop => {
        // Create commodity data for included section
        const includedCommodities = shop.commodities.map((commodity, index) => ({
            type: "commodities",
            id: `temp-id-${shop.npcId}-${index}`, // Temporary ID, will be replaced by server
            attributes: commodity
        }));

        // Create commodity references for relationships section
        const commodityReferences = includedCommodities.map(commodity => ({
            type: "commodities",
            id: commodity.id
        }));

        return {
            type: "shops",
            id: `shop-${shop.npcId}`,
            attributes: {
                npcId: shop.npcId
            },
            relationships: {
                commodities: {
                    data: commodityReferences
                }
            },
            included: includedCommodities
        };
    });

    const response = await fetch(rootUrl + "/api/shops", {
        method: "POST",
        headers: tenantHeaders(tenant),
        body: JSON.stringify({
            data: transformedShops
        }),
    });
    if (!response.ok) {
        throw new Error("Failed to bulk create shops.");
    }
    const responseData = await response.json();
    return responseData.data;
}

export async function updateShop(tenant: Tenant, npcId: number, commodities: Commodity[]): Promise<Shop> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;

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

    const response = await fetch(rootUrl + "/api/npcs/" + npcId + "/shop", {
        method: "PUT",
        headers: tenantHeaders(tenant),
        body: JSON.stringify({
            data: {
                type: "shops",
                id: `shop-${npcId}`,
                attributes: {
                    npcId: npcId
                },
                relationships: {
                    commodities: {
                        data: commodityReferences
                    }
                },
                included: includedCommodities
            }
        }),
    });
    if (!response.ok) {
        throw new Error("Failed to update shop.");
    }
    const responseData = await response.json();
    return responseData.data;
}

export async function deleteAllShops(tenant: Tenant): Promise<void> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(rootUrl + "/api/shops", {
        method: "DELETE",
        headers: tenantHeaders(tenant),
    });
    if (!response.ok) {
        throw new Error("Failed to delete all shops.");
    }
}

export async function deleteAllCommoditiesForNPC(tenant: Tenant, npcId: number): Promise<void> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(rootUrl + "/api/npcs/" + npcId + "/shop/relationships/commodities", {
        method: "DELETE",
        headers: tenantHeaders(tenant),
    });
    if (!response.ok) {
        throw new Error("Failed to delete all commodities for NPC.");
    }
}
