import { api } from '@/lib/api/client';
import type { DropData, ReactorDropData } from '@/types/models/drop';

class DropsService {
  async getMonsterDrops(monsterId: string): Promise<DropData[]> {
    return api.getList<DropData>(`/api/monsters/${monsterId}/drops`);
  }

  async getReactorDrops(reactorId: string): Promise<ReactorDropData[]> {
    return api.getList<ReactorDropData>(`/api/reactors/${reactorId}/drops`);
  }

  async getItemDrops(itemId: string): Promise<DropData[]> {
    return api.getList<DropData>(`/api/items/${itemId}/drops`);
  }
}

export const dropsService = new DropsService();
