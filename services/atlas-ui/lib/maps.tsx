import type {Tenant} from "@/types/models/tenant";
import {tenantHeaders} from "@/lib/headers";

export interface Map {
    id: string;
    attributes: {
        name: string;
        streetName: string;
    }
}

export async function fetchMap(tenant: Tenant, mapId: string): Promise<Map> {
    const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
    const response = await fetch(rootUrl + "/api/data/maps/" + mapId, {
        method: "GET",
        headers: tenantHeaders(tenant),
    });
    if (!response.ok) {
        throw new Error("Failed to fetch map.");
    }
    const responseData = await response.json();
    return responseData.data;
}