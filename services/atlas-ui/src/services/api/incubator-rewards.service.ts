import { api } from "@/lib/api/client";
import type { ServiceOptions } from "@/lib/api/query-params";

/**
 * Per-tenant incubator reward pool. Backed by the atlas-tenants `configurations`
 * resource exposed as the `incubator-rewards` JSON:API collection:
 *   GET/POST /api/tenants/{tenantId}/configurations/incubator-rewards
 *   PATCH/DELETE .../{incubatorRewardId}
 *   POST .../seed   (repopulate from the seed pool)
 * Writes use the JSON:API envelope {data:{type:"incubator-rewards",...}} — bare bodies 400.
 */
export const INCUBATOR_REWARDS_RESOURCE_TYPE = "incubator-rewards";

export interface IncubatorRewardAttributes {
  itemId: number;
  quantity: number;
  weight: number;
}

export interface IncubatorReward {
  id: string;
  attributes: IncubatorRewardAttributes;
}

function path(tenantId: string): string {
  return `/api/tenants/${tenantId}/configurations/incubator-rewards`;
}

export const incubatorRewardsService = {
  async list(tenantId: string, options?: ServiceOptions): Promise<IncubatorReward[]> {
    return api.getList<IncubatorReward>(path(tenantId), options);
  },
  async create(tenantId: string, attributes: IncubatorRewardAttributes, options?: ServiceOptions): Promise<IncubatorReward> {
    return api.post<IncubatorReward>(
      path(tenantId),
      { data: { type: INCUBATOR_REWARDS_RESOURCE_TYPE, attributes } },
      options,
    );
  },
  async update(tenantId: string, id: string, attributes: IncubatorRewardAttributes, options?: ServiceOptions): Promise<void> {
    await api.patch<void>(
      `${path(tenantId)}/${id}`,
      { data: { id, type: INCUBATOR_REWARDS_RESOURCE_TYPE, attributes } },
      options,
    );
  },
  async remove(tenantId: string, id: string, options?: ServiceOptions): Promise<void> {
    await api.delete<void>(`${path(tenantId)}/${id}`, options);
  },
  async seed(tenantId: string, options?: ServiceOptions): Promise<void> {
    await api.post<void>(`${path(tenantId)}/seed`, {}, options);
  },
};
