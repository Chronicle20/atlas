import type {Tenant} from "@/types/models/tenant";
import type {ApiListResponse, ApiSingleResponse} from "@/types/api/responses";
import type {Guild as GuildModel} from "@/types/models/guild";
import {tenantHeaders} from "@/lib/headers";

export interface Guild {
    id: string;
    attributes: {
        worldId: number;
        name: string;
        notice: string;
        points: number;
        capacity: number;
        logo: number;
        logoColor: number;
        logoBackground: number;
        logoBackgroundColor: number;
        leaderId: number;
        members: GuildMember[];
        titles: GuildTitle[];
    };
}

export interface GuildMember {
    characterId: number;
    name: string;
    jobId: number;
    level: number;
    title: number;
    online: boolean;
    allianceTitle: number;
}

export interface GuildTitle {
    name: string;
    index: number;
}

export async function fetchGuilds(tenant: Tenant): Promise<Guild[]> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(rootUrl + "/api/guilds/", {
        method: "GET",
        headers: tenantHeaders(tenant),
    });
    if (!response.ok) {
        throw new Error("Failed to fetch guilds.");
    }
    const responseData: ApiListResponse<Guild> = await response.json();
    return responseData.data;
}

export async function fetchGuild(tenant: Tenant, guildId: string): Promise<Guild> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(rootUrl + "/api/guilds/" + guildId, {
        method: "GET",
        headers: tenantHeaders(tenant),
    });
    if (!response.ok) {
        throw new Error("Failed to fetch guild.");
    }
    const responseData: ApiSingleResponse<Guild> = await response.json();
    return responseData.data;
}