import { api } from "@/lib/api/client";
import { buildQueryString, type QueryOptions } from "@/lib/api/query-params";
import type { ItemStringData } from "@/types/models/item-string";
import {
  getItemType,
  type ItemSearchResult,
  type EquipmentData,
  type ConsumableData,
  type SetupData,
  type EtcData,
  type CashItemData,
  type ItemDetailData,
} from "@/types/models/item";

const BASE_PATH = "/api/data/item-strings";

export const itemsService = {
  async searchItems(query: string, options?: QueryOptions): Promise<ItemSearchResult[]> {
    const searchOptions: QueryOptions = { ...options, search: query };
    const items = await api.getList<ItemStringData>(
      `${BASE_PATH}${buildQueryString(searchOptions)}`,
      searchOptions,
    );
    return items.map((item) => ({
      id: item.id,
      name: item.attributes.name,
      type: getItemType(item.id),
    }));
  },

  async getItemName(itemId: string): Promise<string> {
    const item = await api.getOne<ItemStringData>(`${BASE_PATH}/${itemId}`);
    return item.attributes.name;
  },

  async getEquipment(itemId: string): Promise<EquipmentData> {
    return api.getOne<EquipmentData>(`/api/data/equipment/${itemId}`);
  },

  async getConsumable(itemId: string): Promise<ConsumableData> {
    return api.getOne<ConsumableData>(`/api/data/consumables/${itemId}`);
  },

  async getSetup(itemId: string): Promise<SetupData> {
    return api.getOne<SetupData>(`/api/data/setups/${itemId}`);
  },

  async getEtc(itemId: string): Promise<EtcData> {
    return api.getOne<EtcData>(`/api/data/etcs/${itemId}`);
  },

  async getCashItem(itemId: string): Promise<CashItemData> {
    return api.getOne<CashItemData>(`/api/data/cash/items/${itemId}`);
  },

  async getItemDetail(itemId: string): Promise<ItemDetailData> {
    const type = getItemType(itemId);
    switch (type) {
      case "Equipment": return itemsService.getEquipment(itemId);
      case "Consumable": return itemsService.getConsumable(itemId);
      case "Setup": return itemsService.getSetup(itemId);
      case "Etc": return itemsService.getEtc(itemId);
      case "Cash": return itemsService.getCashItem(itemId);
      default: throw new Error(`Unknown item type for ID ${itemId}`);
    }
  },
};
