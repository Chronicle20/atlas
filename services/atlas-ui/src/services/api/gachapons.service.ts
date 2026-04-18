import { BaseService, type QueryOptions } from './base.service';
import { api } from '@/lib/api/client';
import type { Tenant } from '@/types/models/tenant';
import type { GachaponData } from '@/types/models/gachapon';
import type { GachaponRewardData } from '@/types/models/gachapon-reward';

class GachaponsService extends BaseService {
  protected basePath = '/api/gachapons';

  async getAllGachapons(tenant: Tenant, options?: QueryOptions): Promise<GachaponData[]> {
    api.setTenant(tenant);
    return this.getAll<GachaponData>(options);
  }

  async getGachaponById(id: string, tenant: Tenant): Promise<GachaponData> {
    api.setTenant(tenant);
    return this.getById<GachaponData>(id);
  }

  async getPrizePool(gachaponId: string, tenant: Tenant): Promise<GachaponRewardData[]> {
    api.setTenant(tenant);
    return api.getList<GachaponRewardData>(`${this.basePath}/${gachaponId}/prize-pool`);
  }
}

export const gachaponsService = new GachaponsService();
