import { BaseService, type QueryOptions } from './base.service';
import { api } from '@/lib/api/client';
import type { Tenant } from '@/types/models/tenant';
import type { ItemStringData } from '@/types/models/item-string';

class ItemStringsService extends BaseService {
  protected basePath = '/api/data/item-strings';

  async getAllItemStrings(tenant: Tenant, options?: QueryOptions): Promise<ItemStringData[]> {
    api.setTenant(tenant);
    return this.getAll<ItemStringData>(options);
  }

  async getItemString(itemId: string, tenant: Tenant): Promise<ItemStringData> {
    api.setTenant(tenant);
    return this.getById<ItemStringData>(itemId);
  }
}

export const itemStringsService = new ItemStringsService();
