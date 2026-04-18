import { api } from "@/lib/api/client";
import { buildQueryString, type QueryOptions } from "@/lib/api/query-params";
import type { ItemStringData } from "@/types/models/item-string";

const BASE_PATH = "/api/data/item-strings";

export const itemStringsService = {
  async getAllItemStrings(options?: QueryOptions): Promise<ItemStringData[]> {
    return api.getList<ItemStringData>(`${BASE_PATH}${buildQueryString(options)}`, options);
  },

  async getItemString(itemId: string): Promise<ItemStringData> {
    return api.getOne<ItemStringData>(`${BASE_PATH}/${itemId}`);
  },
};
