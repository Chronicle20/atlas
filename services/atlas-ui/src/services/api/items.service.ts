import { api } from "@/lib/api/client";
import { buildQueryString, type QueryOptions } from "@/lib/api/query-params";
import type { Tenant } from "@/types/models/tenant";
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
  async searchItems(query: string, _tenant: Tenant, options?: QueryOptions): Promise<ItemSearchResult[]> {
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

  async getItemName(itemId: string, _tenant: Tenant): Promise<string> {
    const item = await api.getOne<ItemStringData>(`${BASE_PATH}/${itemId}`);
    return item.attributes.name;
  },

  async getEquipment(itemId: string, _tenant: Tenant): Promise<EquipmentData> {
    return api.getOne<EquipmentData>(`/api/data/equipment/${itemId}`);
  },

  async getConsumable(itemId: string, _tenant: Tenant): Promise<ConsumableData> {
    return api.getOne<ConsumableData>(`/api/data/consumables/${itemId}`);
  },

  async getSetup(itemId: string, _tenant: Tenant): Promise<SetupData> {
    return api.getOne<SetupData>(`/api/data/setups/${itemId}`);
  },

  async getEtc(itemId: string, _tenant: Tenant): Promise<EtcData> {
    return api.getOne<EtcData>(`/api/data/etcs/${itemId}`);
  },

  async getCashItem(itemId: string, _tenant: Tenant): Promise<CashItemData> {
    return api.getOne<CashItemData>(`/api/data/cash/items/${itemId}`);
  },

  async getItemDetail(itemId: string, tenant: Tenant): Promise<ItemDetailData> {
    const type = getItemType(itemId);
    switch (type) {
      case "Equipment": return itemsService.getEquipment(itemId, tenant);
      case "Consumable": return itemsService.getConsumable(itemId, tenant);
      case "Setup": return itemsService.getSetup(itemId, tenant);
      case "Etc": return itemsService.getEtc(itemId, tenant);
      case "Cash": return itemsService.getCashItem(itemId, tenant);
      default: throw new Error(`Unknown item type for ID ${itemId}`);
    }
  },
};
