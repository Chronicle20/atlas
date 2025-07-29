import type {Tenant} from "@/types/models/tenant";
import { api } from "@/lib/api/client";

export interface Map {
    id: string;
    attributes: {
        name: string;
        streetName: string;
    }
}

export async function fetchMap(tenant: Tenant, mapId: string): Promise<Map> {
    api.setTenant(tenant);
    return api.getOne<Map>(`/api/data/maps/${mapId}`);
}