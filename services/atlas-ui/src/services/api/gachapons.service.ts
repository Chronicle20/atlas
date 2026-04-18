import { api } from "@/lib/api/client";
import { buildQueryString, type QueryOptions } from "@/lib/api/query-params";
import type { Tenant } from "@/types/models/tenant";
import type { GachaponData } from "@/types/models/gachapon";
import type { GachaponRewardData } from "@/types/models/gachapon-reward";

const BASE_PATH = "/api/gachapons";

export const gachaponsService = {
  async getAllGachapons(_tenant: Tenant, options?: QueryOptions): Promise<GachaponData[]> {
    return api.getList<GachaponData>(`${BASE_PATH}${buildQueryString(options)}`, options);
  },

  async getGachaponById(id: string, _tenant: Tenant): Promise<GachaponData> {
    return api.getOne<GachaponData>(`${BASE_PATH}/${id}`);
  },

  async getPrizePool(gachaponId: string, _tenant: Tenant): Promise<GachaponRewardData[]> {
    return api.getList<GachaponRewardData>(`${BASE_PATH}/${gachaponId}/prize-pool`);
  },
};
