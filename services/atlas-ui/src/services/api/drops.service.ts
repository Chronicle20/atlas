import { fetchAll } from "@/services/api/pagination";
import type { DropData, ReactorDropData } from "@/types/models/drop";

// Each *DetailPage widget renders the full drop table for one monster/reactor/
// item, not a page at a time — drain every page (task-117).
class DropsService {
  async getMonsterDrops(monsterId: string): Promise<DropData[]> {
    return fetchAll<DropData>(`/api/monsters/${monsterId}/drops`);
  }

  async getReactorDrops(reactorId: string): Promise<ReactorDropData[]> {
    return fetchAll<ReactorDropData>(`/api/reactors/${reactorId}/drops`);
  }

  async getItemDrops(itemId: string): Promise<DropData[]> {
    return fetchAll<DropData>(`/api/items/${itemId}/drops`);
  }
}

export const dropsService = new DropsService();
