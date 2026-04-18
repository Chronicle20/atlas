import { BaseService, type QueryOptions } from './base.service';
import { api } from '@/lib/api/client';
import type { Tenant } from '@/types/models/tenant';
import type { ItemStringData } from '@/types/models/item-string';
import {
  getItemType,
  type ItemSearchResult,
  type EquipmentData,
  type ConsumableData,
  type SetupData,
  type EtcData,
  type CashItemData,
  type ItemDetailData,
} from '@/types/models/item';

class ItemsService extends BaseService {
  protected basePath = '/api/data/item-strings';

  async searchItems(query: string, tenant: Tenant, options?: QueryOptions): Promise<ItemSearchResult[]> {
    api.setTenant(tenant);
    const searchOptions: QueryOptions = {
      ...options,
      search: query,
      useCache: false,
    };
    const items = await this.getAll<ItemStringData>(searchOptions);
    return items.map((item) => ({
      id: item.id,
      name: item.attributes.name,
      type: getItemType(item.id),
    }));
  }

  async getItemName(itemId: string, tenant: Tenant): Promise<string> {
    api.setTenant(tenant);
    const item = await this.getById<ItemStringData>(itemId);
    return item.attributes.name;
  }

  async getEquipment(itemId: string, tenant: Tenant): Promise<EquipmentData> {
    api.setTenant(tenant);
    return api.getOne<EquipmentData>(`/api/data/equipment/${itemId}`);
  }

  async getConsumable(itemId: string, tenant: Tenant): Promise<ConsumableData> {
    api.setTenant(tenant);
    return api.getOne<ConsumableData>(`/api/data/consumables/${itemId}`);
  }

  async getSetup(itemId: string, tenant: Tenant): Promise<SetupData> {
    api.setTenant(tenant);
    return api.getOne<SetupData>(`/api/data/setups/${itemId}`);
  }

  async getEtc(itemId: string, tenant: Tenant): Promise<EtcData> {
    api.setTenant(tenant);
    return api.getOne<EtcData>(`/api/data/etcs/${itemId}`);
  }

  async getCashItem(itemId: string, tenant: Tenant): Promise<CashItemData> {
    api.setTenant(tenant);
    return api.getOne<CashItemData>(`/api/data/cash/items/${itemId}`);
  }

  async getItemDetail(itemId: string, tenant: Tenant): Promise<ItemDetailData> {
    const type = getItemType(itemId);
    switch (type) {
      case "Equipment": return this.getEquipment(itemId, tenant);
      case "Consumable": return this.getConsumable(itemId, tenant);
      case "Setup": return this.getSetup(itemId, tenant);
      case "Etc": return this.getEtc(itemId, tenant);
      case "Cash": return this.getCashItem(itemId, tenant);
      default: throw new Error(`Unknown item type for ID ${itemId}`);
    }
  }
}

export const itemsService = new ItemsService();
