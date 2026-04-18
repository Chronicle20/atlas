import { api } from "@/lib/api/client";
import { buildQueryString, type QueryOptions } from "@/lib/api/query-params";
import type { Tenant } from "@/types/models/tenant";
import type { ItemStringData } from "@/types/models/item-string";

const BASE_PATH = "/api/data/item-strings";

export const itemStringsService = {
  async getAllItemStrings(_tenant: Tenant, options?: QueryOptions): Promise<ItemStringData[]> {
    return api.getList<ItemStringData>(`${BASE_PATH}${buildQueryString(options)}`, options);
  },

  async getItemString(itemId: string, _tenant: Tenant): Promise<ItemStringData> {
    return api.getOne<ItemStringData>(`${BASE_PATH}/${itemId}`);
  },
};
