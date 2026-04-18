import { api } from '@/lib/api/client';
import type { Tenant } from '@/types/models/tenant';
import type { DropData, ReactorDropData } from '@/types/models/drop';

class DropsService {
  async getMonsterDrops(monsterId: string, tenant: Tenant): Promise<DropData[]> {
    return api.getList<DropData>(`/api/monsters/${monsterId}/drops`);
  }

  async getReactorDrops(reactorId: string, tenant: Tenant): Promise<ReactorDropData[]> {
    return api.getList<ReactorDropData>(`/api/reactors/${reactorId}/drops`);
  }

  async getItemDrops(itemId: string, tenant: Tenant): Promise<DropData[]> {
    return api.getList<DropData>(`/api/items/${itemId}/drops`);
  }
}

export const dropsService = new DropsService();
