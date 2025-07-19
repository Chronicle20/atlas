import {Tenant} from "@/app/tenants/columns";
import {tenantHeaders} from "@/lib/headers";

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
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(rootUrl + "/api/characters/", {
        method: "GET",
        headers: tenantHeaders(tenant),
    });
    if (!response.ok) {
        throw new Error("Failed to fetch characters.");
    }
    const responseData = await response.json();
    return responseData.data;
}

export async function fetchCharacter(tenant: Tenant, characterId: string): Promise<Character> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(rootUrl + "/api/characters/" + characterId, {
        method: "GET",
        headers: tenantHeaders(tenant),
    });
    if (!response.ok) {
        throw new Error("Failed to fetch character.");
    }
    const responseData = await response.json();
    return responseData.data;
}

interface UpdateCharacterData {
    mapId?: number;
}

export async function updateCharacter(tenant: Tenant, characterId: string, data: UpdateCharacterData): Promise<void> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(rootUrl + "/api/cos/characters/" + characterId, {
        method: "PATCH",
        headers: {
            ...tenantHeaders(tenant),
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            data: {
                type: "characters",
                id: characterId,
                attributes: data,
            },
        }),
    });
    
    if (!response.ok) {
        if (response.status === 400) {
            throw new Error("Invalid map ID or request");
        } else if (response.status === 404) {
            throw new Error("Character not found");
        } else {
            throw new Error("Failed to update character");
        }
    }
    
    // API returns 204 No Content on success
}