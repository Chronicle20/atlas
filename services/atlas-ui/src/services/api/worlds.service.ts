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

class WorldsService {
  async getWorlds(): Promise<WorldData[]> {
    return api.getList<WorldData>("/api/worlds/");
  }

  async getChannels(worldId: number): Promise<ChannelData[]> {
    return api.getList<ChannelData>(`/api/worlds/${worldId}/channels`);
  }
}

export const worldsService = new WorldsService();
