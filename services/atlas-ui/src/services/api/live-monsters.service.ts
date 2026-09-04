import { api } from "@/lib/api/client";

export interface DamageEntry {
  characterId: number;
  damage: number;
}

export interface StatusEffectEntry {
  sourceSkillId: number;
  sourceSkillLevel: number;
  statuses: Record<string, number>;
  expiresAt: number;
}

export interface LiveMonsterData {
  id: string;
  type: string;
  attributes: {
    worldId: number;
    channelId: number;
    mapId: number;
    instance: string;
    monsterId: number;
    controlCharacterId: number;
    x: number;
    y: number;
    fh: number;
    stance: number;
    team: number;
    maxHp: number;
    hp: number;
    maxMp: number;
    mp: number;
    damageEntries: DamageEntry[];
    experienceEntries: DamageEntry[];
    statusEffects: StatusEffectEntry[];
    controllerHasAggro: boolean;
    nextEligibleRepickAtMs?: number;
    spawnSourceType?: string;
    spawnSourceId?: string;
  };
}

// The backend caps page[size] at 250 (paginate.MaxPageSize), which is also
// the default when unspecified. Request the max explicitly for parity with
// fields.service.ts — the live-monster count for a field is therefore capped
// at paginate.MaxPageSize.
const PAGE_SIZE = 250;

class LiveMonstersService {
  async getMonsters(
    worldId: number,
    channelId: number,
    mapId: number,
    instanceId: string,
  ): Promise<LiveMonsterData[]> {
    const params = new URLSearchParams({ "page[size]": String(PAGE_SIZE) });
    return api.getList<LiveMonsterData>(
      `/api/worlds/${worldId}/channels/${channelId}/maps/${mapId}/instances/${instanceId}/monsters?${params.toString()}`,
    );
  }
}

export const liveMonstersService = new LiveMonstersService();
