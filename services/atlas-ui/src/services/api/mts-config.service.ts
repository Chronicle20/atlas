import { api } from "@/lib/api/client";
import type { ServiceOptions } from "@/lib/api/query-params";

/**
 * Per-tenant marketplace (MTS) configuration — the §8 economic knobs.
 *
 * Backed by the atlas-tenants generic `configurations` resource exposed as the
 * `mts-configs` JSON:API type:
 *   GET   /api/tenants/{tenantId}/configurations/mts-configs              (single object)
 *   PATCH /api/tenants/{tenantId}/configurations/mts-configs/{mtsConfigId}
 *
 * Writes use the JSON:API envelope `{data:{type:"mts-configs", id, attributes}}`
 * — bare bodies 400.
 */

export const MTS_CONFIG_RESOURCE_TYPE = "mts-configs";

export interface MtsConfigAttributes {
  listingFee: number;
  commissionRate: number;
  maxActiveListings: number;
  minLevel: number;
  auctionMinHours: number;
  auctionMaxHours: number;
  priceFloor: number;
  pageSize: number;
  minBidIncrement: number;
}

export interface MtsConfig {
  id: string;
  attributes: MtsConfigAttributes;
}

function configPath(tenantId: string): string {
  return `/api/tenants/${tenantId}/configurations/mts-configs`;
}

interface UpdateMtsConfigInput {
  data: { id: string; type: typeof MTS_CONFIG_RESOURCE_TYPE; attributes: MtsConfigAttributes };
}

export const mtsConfigService = {
  /**
   * Fetch the single per-tenant MTS configuration. The backend returns a single
   * JSON:API object (not a list); 404 when no config exists for the tenant.
   */
  async getConfig(tenantId: string, options?: ServiceOptions): Promise<MtsConfig> {
    return api.getOne<MtsConfig>(configPath(tenantId), options);
  },

  /**
   * PATCH the per-tenant MTS configuration. The full attribute set is sent in a
   * JSON:API envelope under the config's own id.
   */
  async updateConfig(
    tenantId: string,
    config: MtsConfig,
    updatedAttributes: Partial<MtsConfigAttributes>,
    options?: ServiceOptions,
  ): Promise<MtsConfig> {
    const attributes: MtsConfigAttributes = { ...config.attributes, ...updatedAttributes };
    const input: UpdateMtsConfigInput = {
      data: { id: config.id, type: MTS_CONFIG_RESOURCE_TYPE, attributes },
    };
    await api.patch<void>(`${configPath(tenantId)}/${config.id}`, input, options);
    return { ...config, attributes };
  },
};
