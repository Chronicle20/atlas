import { api } from "@/lib/api/client";
import { buildQueryString, type QueryOptions } from "@/lib/api/query-params";
import { fetchAll } from "@/services/api/pagination";
import type { RewardPoolData, RewardPoolAttributes } from "@/types/models/reward-pool";
import type { RewardPoolItemData, RewardPoolItemAttributes } from "@/types/models/reward-pool-item";
import type { GlobalRewardItemData, GlobalRewardItemAttributes } from "@/types/models/global-reward-item";

const BASE_PATH = "/api/gachapons"; // REST identity deliberately unchanged (design §5)
const GLOBAL_PATH = "/api/global-items";

export const REWARD_POOL_TYPE = "gachapons";
export const REWARD_POOL_ITEM_TYPE = "gachapon-items";
export const GLOBAL_REWARD_ITEM_TYPE = "global-gachapon-items";

export const rewardPoolsService = {
  /** Drain the whole pool collection (small: machines + ten eggs) so kind tabs count exactly. */
  async getAllPools(options?: QueryOptions): Promise<RewardPoolData[]> {
    return fetchAll<RewardPoolData>(`${BASE_PATH}${buildQueryString(options)}`, undefined, options);
  },

  async getPoolById(id: string): Promise<RewardPoolData> {
    return api.getOne<RewardPoolData>(`${BASE_PATH}/${id}`);
  },

  /** id is client-supplied for incubator pools (the egg item id); omitted for classic gachapons. */
  async createPool(id: string | undefined, attributes: RewardPoolAttributes): Promise<void> {
    await api.post(BASE_PATH, { data: { ...(id !== undefined ? { id } : {}), type: REWARD_POOL_TYPE, attributes } });
  },

  async updatePool(id: string, attributes: RewardPoolAttributes): Promise<void> {
    await api.patch(`${BASE_PATH}/${id}`, { data: { id, type: REWARD_POOL_TYPE, attributes } });
  },

  async removePool(id: string): Promise<void> {
    await api.delete(`${BASE_PATH}/${id}`);
  },

  async getItems(poolId: string): Promise<RewardPoolItemData[]> {
    return fetchAll<RewardPoolItemData>(`${BASE_PATH}/${poolId}/items`);
  },

  async createItem(poolId: string, attributes: Omit<RewardPoolItemAttributes, "gachaponId">): Promise<void> {
    await api.post(`${BASE_PATH}/${poolId}/items`, { data: { type: REWARD_POOL_ITEM_TYPE, attributes } });
  },

  async updateItem(poolId: string, itemRecordId: string, attributes: Omit<RewardPoolItemAttributes, "gachaponId">): Promise<void> {
    await api.patch(`${BASE_PATH}/${poolId}/items/${itemRecordId}`, {
      data: { id: itemRecordId, type: REWARD_POOL_ITEM_TYPE, attributes },
    });
  },

  async removeItem(poolId: string, itemRecordId: string): Promise<void> {
    await api.delete(`${BASE_PATH}/${poolId}/items/${itemRecordId}`);
  },

  async getGlobalItems(): Promise<GlobalRewardItemData[]> {
    return fetchAll<GlobalRewardItemData>(GLOBAL_PATH);
  },

  async createGlobalItem(attributes: GlobalRewardItemAttributes): Promise<void> {
    await api.post(GLOBAL_PATH, { data: { type: GLOBAL_REWARD_ITEM_TYPE, attributes } });
  },

  async updateGlobalItem(itemRecordId: string, attributes: GlobalRewardItemAttributes): Promise<void> {
    await api.patch(`${GLOBAL_PATH}/${itemRecordId}`, {
      data: { id: itemRecordId, type: GLOBAL_REWARD_ITEM_TYPE, attributes },
    });
  },

  async removeGlobalItem(itemRecordId: string): Promise<void> {
    await api.delete(`${GLOBAL_PATH}/${itemRecordId}`);
  },
};
