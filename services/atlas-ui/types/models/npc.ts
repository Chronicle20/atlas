// NPC domain model types
// Re-exported from lib/npcs.tsx to centralize type definitions

export interface NPC {
    id: number;
    name?: string;
    iconUrl?: string;
    hasShop: boolean;
    hasConversation: boolean;
}

// API response data type is defined in @/types/models/maplestory.ts as NpcApiData

// Shop-related types
export interface Shop {
    type: string;
    id: string;
    attributes: ShopAttributes;
    relationships?: {
        commodities: {
            data: CommodityReference[];
        };
    };
    included?: Commodity[];
}

export interface ShopAttributes {
    npcId: number;
    recharger?: boolean;
}

export interface Commodity {
    id: string;
    type: string;
    attributes: CommodityAttributes;
}

export interface CommodityAttributes {
    templateId: number;
    mesoPrice: number;
    discountRate: number;
    tokenTemplateId: number;
    tokenPrice: number;
    period: number;
    levelLimit: number;
    unitPrice?: number;
    slotMax?: number;
}

export interface CommodityReference {
    type: string;
    id: string;
}

export interface ShopResponse {
    data: Shop;
    included?: Commodity[];
}