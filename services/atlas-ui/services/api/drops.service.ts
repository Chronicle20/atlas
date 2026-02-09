import { api } from '@/lib/api/client';
import type { Tenant } from '@/types/models/tenant';
import type { DropData, ReactorDropData } from '@/types/models/drop';

class DropsService {
  async getMonsterDrops(monsterId: string, tenant: Tenant): Promise<DropData[]> {
    api.setTenant(tenant);
    return api.getList<DropData>(`/api/monsters/${monsterId}/drops`);
  }

  async getReactorDrops(reactorId: string, tenant: Tenant): Promise<ReactorDropData[]> {
    api.setTenant(tenant);
    return api.getList<ReactorDropData>(`/api/reactors/${reactorId}/drops`);
  }
}

export const dropsService = new DropsService();
