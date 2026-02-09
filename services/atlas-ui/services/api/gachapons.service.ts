import { BaseService, type QueryOptions } from './base.service';
import { api } from '@/lib/api/client';
import type { Tenant } from '@/types/models/tenant';
import type { GachaponData } from '@/types/models/gachapon';

class GachaponsService extends BaseService {
  protected basePath = '/api/gachapons';

  async getAllGachapons(tenant: Tenant, options?: QueryOptions): Promise<GachaponData[]> {
    api.setTenant(tenant);
    return this.getAll<GachaponData>(options);
  }
}

export const gachaponsService = new GachaponsService();
