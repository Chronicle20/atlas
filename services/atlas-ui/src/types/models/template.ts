// Template domain model types
// Re-exported from lib/templates.tsx to centralize type definitions

export interface CharacterTemplate {
    jobIndex: number;
    subJobIndex: number;
    gender: number;
    mapId: number;
    faces: number[];
    hairs: number[];
    hairColors: number[];
    skinColors: number[];
    tops: number[];
    bottoms: number[];
    shoes: number[];
    weapons: number[];
    items: number[];
    skills: number[];
}

export interface TemplateAttributes {
    region: string;
    majorVersion: number;
    minorVersion: number;
    usesPin: boolean;
    characters: {
        templates: CharacterTemplate[];
    };
    npcs: {
        npcId: number;
        impl: string;
    }[];
    socket: {
        handlers: {
            opCode: string;
            validator: string;
            handler: string;
            options: unknown;
        }[];
        writers: {
            opCode: string;
            writer: string;
            options: unknown;
        }[];
    };
    worlds: {
        name: string;
        flag: string;
        serverMessage: string;
        eventMessage: string;
        whyAmIRecommended: string;
        expRate?: number;
        mesoRate?: number;
        itemDropRate?: number;
        questExpRate?: number;
    }[];
    cashShop?: {
        commodities: {
            hourlyExpirations?: {
                templateId: number;
                hours: number;
            }[];
        };
    };
}

export interface Template {
    id: string;
    attributes: TemplateAttributes;
}