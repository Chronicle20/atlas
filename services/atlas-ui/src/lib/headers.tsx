import type { Tenant } from "@/types/models/tenant";

export function tenantHeaders(tenant: Tenant): Headers {
  const headers = new Headers();
  headers.set("TENANT_ID", tenant?.id);
  headers.set("REGION", tenant?.attributes.region);
  headers.set("MAJOR_VERSION", String(tenant?.attributes.majorVersion));
  headers.set("MINOR_VERSION", String(tenant?.attributes.minorVersion));
  return headers;
}

/**
 * The synthetic tenant id used for canonical (deployment-wide) requests.
 * atlas-data's shared scope never reads the tenant id — ResolveScope gates
 * only on X-Atlas-Operator and the shared prefix is keyed by region/version —
 * but the shared REST middleware requires syntactically valid tenant headers,
 * and uuid.Parse accepts the nil UUID.
 */
export const CANONICAL_TENANT_ID = "00000000-0000-0000-0000-000000000000";

export interface CanonicalSelection {
  region: string;
  majorVersion: number;
  minorVersion: number;
}

/**
 * Headers for canonical-scope requests. Baking X-Atlas-Operator in here means
 * a canonical request cannot be assembled without the operator header — one
 * construction path, no drift.
 */
export function canonicalHeaders(sel: CanonicalSelection): Headers {
  const headers = new Headers();
  headers.set("TENANT_ID", CANONICAL_TENANT_ID);
  headers.set("REGION", sel.region);
  headers.set("MAJOR_VERSION", String(sel.majorVersion));
  headers.set("MINOR_VERSION", String(sel.minorVersion));
  headers.set("X-Atlas-Operator", "1");
  return headers;
}
