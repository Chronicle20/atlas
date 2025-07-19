import type {Tenant} from "@/types/models/tenant";
import type {ApiListResponse, ApiSingleResponse, ApiErrorResponse, ApiSimpleErrorResponse} from "@/types/api/responses";
import {isApiErrorResponse, isApiSimpleErrorResponse} from "@/types/api/responses";
import type {Character as CharacterModel, UpdateCharacterData} from "@/types/models/character";
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
    const responseData: ApiListResponse<Character> = await response.json();
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
    const responseData: ApiSingleResponse<Character> = await response.json();
    return responseData.data;
}

// UpdateCharacterData is now imported from types

export async function updateCharacter(tenant: Tenant, characterId: string, data: UpdateCharacterData): Promise<void> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    
    try {
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
            // Try to parse error details from response body if available
            let errorMessage = "Failed to update character";
            
            try {
                const errorData: ApiErrorResponse | ApiSimpleErrorResponse = await response.json();
                if (isApiErrorResponse(errorData)) {
                    errorMessage = errorData.error.detail;
                } else if (isApiSimpleErrorResponse(errorData)) {
                    errorMessage = errorData.message;
                }
            } catch {
                // Fall back to status-based messages if response body can't be parsed
                if (response.status === 400) {
                    errorMessage = "Invalid map ID or request parameters";
                } else if (response.status === 404) {
                    errorMessage = "Character not found";
                } else if (response.status === 401) {
                    errorMessage = "Authentication failed";
                } else if (response.status === 403) {
                    errorMessage = "Permission denied";
                } else if (response.status === 409) {
                    errorMessage = "Map change conflicts with current character state";
                } else if (response.status >= 500) {
                    errorMessage = "Server error occurred while updating character";
                }
            }
            
            throw new Error(errorMessage);
        }
        
        // API returns 204 No Content on success
    } catch (error: unknown) {
        // Handle network errors and other fetch failures
        if (error instanceof TypeError && error.message.includes('fetch')) {
            throw new Error("Network error: Unable to connect to server");
        } else if (error instanceof Error) {
            // Re-throw API errors with their specific messages
            throw error;
        } else {
            // Handle unexpected error types
            throw new Error("An unexpected error occurred while updating character");
        }
    }
}