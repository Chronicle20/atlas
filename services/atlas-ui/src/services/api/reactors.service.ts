import { api } from "@/lib/api/client";
import {
  buildQueryString,
  type ServiceOptions,
  type QueryOptions,
} from "@/lib/api/query-params";
import { fetchAll, fetchPaged } from "@/services/api/pagination";
import type { ReactorData } from "@/types/models/reactor";

const BASE_PATH = "/api/data/reactors";

// ReactorsPage advertises "Results are limited to 50 entries" — searchReactors
// is a bounded single-page lookup, not a drained browse (task-117).
const SEARCH_RESULT_LIMIT = 50;

export const reactorsService = {
  /**
   * Get every reactor, draining all pages (task-117).
   */
  async getAllReactors(options?: QueryOptions): Promise<ReactorData[]> {
    return fetchAll<ReactorData>(
      `${BASE_PATH}${buildQueryString(options)}`,
      undefined,
      options,
    );
  },

  async searchReactors(
    query: string,
    options?: QueryOptions,
  ): Promise<ReactorData[]> {
    const finalOptions: QueryOptions = { ...options, search: query };
    const result = await fetchPaged<ReactorData>(
      `${BASE_PATH}${buildQueryString(finalOptions)}`,
      { number: 1, size: SEARCH_RESULT_LIMIT },
      finalOptions,
    );
    return result.data;
  },

  async getReactorById(
    id: string,
    options?: ServiceOptions,
  ): Promise<ReactorData> {
    return api.getOne<ReactorData>(`${BASE_PATH}/${id}`, options);
  },
};
