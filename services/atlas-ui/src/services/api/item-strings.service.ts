import { api } from "@/lib/api/client";
import type { ItemStringData } from "@/types/models/item-string";

const BASE_PATH = "/api/data/item-strings";

export const itemStringsService = {
  async getItemString(itemId: string): Promise<ItemStringData> {
    return api.getOne<ItemStringData>(`${BASE_PATH}/${itemId}`);
  },
};
