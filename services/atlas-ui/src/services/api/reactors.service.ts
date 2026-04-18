import { api } from "@/lib/api/client";
import { buildQueryString, type ServiceOptions, type QueryOptions } from "@/lib/api/query-params";
import type { Tenant } from "@/types/models/tenant";
import type { ReactorData } from "@/types/models/reactor";

const BASE_PATH = "/api/data/reactors";

export const reactorsService = {
  async getAllReactors(_tenant: Tenant, options?: QueryOptions): Promise<ReactorData[]> {
    return api.getList<ReactorData>(`${BASE_PATH}${buildQueryString(options)}`, options);
  },

  async searchReactors(query: string, _tenant: Tenant, options?: QueryOptions): Promise<ReactorData[]> {
    const finalOptions: QueryOptions = { ...options, search: query };
    return api.getList<ReactorData>(`${BASE_PATH}${buildQueryString(finalOptions)}`, finalOptions);
  },

  async getReactorById(id: string, _tenant: Tenant, options?: ServiceOptions): Promise<ReactorData> {
    return api.getOne<ReactorData>(`${BASE_PATH}/${id}`, options);
  },
};
