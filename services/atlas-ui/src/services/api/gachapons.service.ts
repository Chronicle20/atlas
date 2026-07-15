import { api } from "@/lib/api/client";
import { buildQueryString, type QueryOptions } from "@/lib/api/query-params";
import { fetchAll, fetchPaged, type PagedResult } from "@/services/api/pagination";
import type { GachaponData } from "@/types/models/gachapon";
import type { GachaponRewardData } from "@/types/models/gachapon-reward";

const BASE_PATH = "/api/gachapons";

export const gachaponsService = {
  /**
   * Get every gachapon, draining all pages (task-117). Used by consumers
   * that genuinely need the whole collection.
   */
  async getAllGachapons(options?: QueryOptions): Promise<GachaponData[]> {
    return fetchAll<GachaponData>(`${BASE_PATH}${buildQueryString(options)}`, undefined, options);
  },

  /**
   * Get a single page of gachapons. Used by the Gachapons list view
   * (task-117), which pages server-side.
   */
  async getPage(page: { number: number; size: number }, options?: QueryOptions): Promise<PagedResult<GachaponData>> {
    return fetchPaged<GachaponData>(`${BASE_PATH}${buildQueryString(options)}`, page, options);
  },

  async getGachaponById(id: string): Promise<GachaponData> {
    return api.getOne<GachaponData>(`${BASE_PATH}/${id}`);
  },

  /**
   * Get every prize in this gachapon's pool, draining all pages (task-117)
   * — the detail-page widget renders the full pool, not a page at a time.
   */
  async getPrizePool(gachaponId: string): Promise<GachaponRewardData[]> {
    return fetchAll<GachaponRewardData>(`${BASE_PATH}/${gachaponId}/prize-pool`);
  },
};
