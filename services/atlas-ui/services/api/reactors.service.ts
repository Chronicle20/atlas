import { BaseService, type ServiceOptions, type QueryOptions } from './base.service';
import { api } from '@/lib/api/client';
import type { Tenant } from '@/types/models/tenant';
import type { ReactorData } from '@/types/models/reactor';

class ReactorsService extends BaseService {
  protected basePath = '/api/data/reactors';

  async getAllReactors(tenant: Tenant, options?: QueryOptions): Promise<ReactorData[]> {
    api.setTenant(tenant);
    return this.getAll<ReactorData>(options);
  }

  async getReactorById(id: string, tenant: Tenant, options?: ServiceOptions): Promise<ReactorData> {
    api.setTenant(tenant);
    return this.getById<ReactorData>(id, options);
  }
}

export const reactorsService = new ReactorsService();
