import type {Tenant} from "@/types/models/tenant";
import type {UpdateCharacterData} from "@/types/models/character";
import {api} from "@/lib/api/client";

export interface Character {
    id: string;
    attributes: {
        accountId: number;
        worldId: number;
        name: string;
        level: number;
        experience: number;
        gachaponExperience: number;
        strength: number;
        dexterity: number;
        intelligence: number;
        luck: number;
        hp: number;
        maxHp: number;
        mp: number;
        maxMp: number;
        meso: number;
        hpMpUsed: number;
        jobId: number;
        skinColor: number;
        gender: number;
        fame: number;
        hair: number;
        face: number;
        mapId: number;
        spawnPoint: number;
        gm: number;
        x: number;
        y: number;
        stance: number;
    };
}

export async function fetchCharacters(tenant: Tenant): Promise<Character[]> {
    // Set tenant for this request
    api.setTenant(tenant);
    
    // Use the centralized API client to fetch characters
    return api.getList<Character>("/api/characters/");
}

export async function fetchCharacter(tenant: Tenant, characterId: string): Promise<Character> {
    // Set tenant for this request
    api.setTenant(tenant);
    
    // Use the centralized API client to fetch a single character
    return api.getOne<Character>(`/api/characters/${characterId}`);
}

export async function updateCharacter(tenant: Tenant, characterId: string, data: UpdateCharacterData): Promise<void> {
    // Set tenant for this request
    api.setTenant(tenant);
    
    // Prepare the JSON:API formatted request body
    const requestBody = {
        data: {
            type: "characters",
            id: characterId,
            attributes: data,
        },
    };
    
    // Use the centralized API client to update the character
    // The API client handles all error cases and status codes automatically
    await api.patch<void>(`/api/cos/characters/${characterId}`, requestBody);
}