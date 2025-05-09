import {Tenant} from "@/app/tenants/columns";
import {tenantHeaders} from "@/lib/headers";

export interface Commodity {
    id: string;
    templateId: number;
    mesoPrice: number;
    discountRate: number;
    tokenItemId: number;
    tokenPrice: number;
    period: number;
    levelLimit: number;
}

export interface Shop {
    id: string;
    attributes: {
        id: string;
        npcId: number;
        commodities: Commodity[];
    };
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

export async function fetchNPCShop(tenant: Tenant, npcId: number): Promise<Shop> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(rootUrl + "/api/npcs/" + npcId + "/shop", {
        method: "GET",
        headers: tenantHeaders(tenant),
    });
    if (!response.ok) {
        throw new Error("Failed to fetch NPC shop.");
    }
    const responseData = await response.json();
    return responseData.data;
}

export async function createCommodity(tenant: Tenant, npcId: number, commodity: Omit<Commodity, 'id'>): Promise<Commodity> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(rootUrl + "/api/npcs/" + npcId + "/shop/commodities", {
        method: "POST",
        headers: {
            ...tenantHeaders(tenant),
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(commodity),
    });
    if (!response.ok) {
        throw new Error("Failed to create commodity.");
    }
    const responseData = await response.json();
    return responseData.data;
}

export async function updateCommodity(tenant: Tenant, npcId: number, commodityId: string, commodity: Partial<Commodity>): Promise<Commodity> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(rootUrl + "/api/npcs/" + npcId + "/shop/commodities/" + commodityId, {
        method: "PUT",
        headers: {
            ...tenantHeaders(tenant),
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(commodity),
    });
    if (!response.ok) {
        throw new Error("Failed to update commodity.");
    }
    const responseData = await response.json();
    return responseData.data;
}

export async function deleteCommodity(tenant: Tenant, npcId: number, commodityId: string): Promise<void> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(rootUrl + "/api/npcs/" + npcId + "/shop/commodities/" + commodityId, {
        method: "DELETE",
        headers: tenantHeaders(tenant),
    });
    if (!response.ok) {
        throw new Error("Failed to delete commodity.");
    }
}