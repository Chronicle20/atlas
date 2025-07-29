import type {Tenant} from "@/types/models/tenant";
import type {Guild} from "@/types/models/guild";
import { api } from "@/lib/api/client";

export async function fetchGuilds(tenant: Tenant): Promise<Guild[]> {
    api.setTenant(tenant);
    return api.getList<Guild>('/api/guilds/');
}

export async function fetchGuild(tenant: Tenant, guildId: string): Promise<Guild> {
    api.setTenant(tenant);
    return api.getOne<Guild>(`/api/guilds/${guildId}`);
}