import { api } from "@/lib/api/client";

export interface WorldData {
  id: string;
  type: string;
  attributes: {
    name: string;
    state: number;
    message: string;
    eventMessage: string;
    recommended: boolean;
    recommendedMessage: string;
    capacityStatus: number;
    expRate: number;
    mesoRate: number;
    itemDropRate: number;
    questExpRate: number;
  };
}

export interface ChannelData {
  id: string;
  type: string;
  attributes: {
    worldId: number;
    channelId: number;
    ipAddress: string;
    port: number;
    currentCapacity: number;
    maxCapacity: number;
    createdAt: string;
    expRate: number;
    mesoRate: number;
    itemDropRate: number;
    questExpRate: number;
  };
}

// The backend caps page[size] at 250 (paginate.MaxPageSize), which is also
// the default when unspecified — an unparameterised request silently
// truncates at 250. Request the max explicitly.
const PAGE_SIZE = 250;

class WorldsService {
  async getWorlds(): Promise<WorldData[]> {
    return api.getList<WorldData>(`/api/worlds/?page[size]=${PAGE_SIZE}`);
  }

  async getChannels(worldId: number): Promise<ChannelData[]> {
    return api.getList<ChannelData>(
      `/api/worlds/${worldId}/channels?page[size]=${PAGE_SIZE}`,
    );
  }
}

export const worldsService = new WorldsService();
