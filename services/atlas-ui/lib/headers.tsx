import type {Tenant} from "@/types/models/tenant";

export function tenantHeaders(tenant: Tenant): Headers {
    const headers = new Headers();
    headers.set("TENANT_ID", tenant?.id);
    headers.set("REGION", tenant?.attributes.region);
    headers.set("MAJOR_VERSION", String(tenant?.attributes.majorVersion));
    headers.set("MINOR_VERSION", String(tenant?.attributes.minorVersion));
    return headers
}