import { api } from "@/lib/api/client";
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
import type { Compartment } from "@/lib/items/taxonomy";

const BASE_PATH = "/api/data/item-strings";

interface ItemStringSearchData {
  id: string;
  attributes: {
    name: string;
    compartment?: string;
    subcategory?: string;
  };
}

export interface ItemSearchFilters {
  q?: string;
  compartment?: Exclude<Compartment, "unknown">;
  subcategory?: string;
  classes?: string[];
}

function compartmentFromWire(raw: string | undefined): Compartment {
  switch (raw) {
    case "equipment":
    case "use":
    case "setup":
    case "etc":
    case "cash":
      return raw;
    default:
      return "unknown";
  }
}

export function buildItemSearchQuery(filters: ItemSearchFilters): string {
  const params = new URLSearchParams();
  if (filters.q && filters.q.length > 0) params.set("search", filters.q);
  if (filters.compartment) params.set("filter[compartment]", filters.compartment);
  if (filters.subcategory) params.set("filter[subcategory]", filters.subcategory);
  if (filters.classes && filters.classes.length > 0) {
    if (filters.classes.length === 1 && filters.classes[0] === "any") {
      params.set("filter[class]", "any");
    } else {
      params.set("filter[class]", [...filters.classes].sort().join(","));
    }
  }
  const qs = params.toString();
  return qs ? `?${qs}` : "";
}

export const itemsService = {
  async searchItems(filters: ItemSearchFilters): Promise<ItemSearchResult[]> {
    const items = await api.getList<ItemStringSearchData>(`${BASE_PATH}${buildItemSearchQuery(filters)}`);
    return items.map((item) => ({
      id: item.id,
      name: item.attributes.name,
      compartment: compartmentFromWire(item.attributes.compartment),
      subcategory: item.attributes.subcategory ?? "",
      type: getItemType(item.id),
    }));
  },

  async getItemName(itemId: string): Promise<string> {
    const item = await api.getOne<ItemStringSearchData>(`${BASE_PATH}/${itemId}`);
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
