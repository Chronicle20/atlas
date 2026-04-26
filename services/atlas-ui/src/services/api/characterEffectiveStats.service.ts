import { api } from "@/lib/api/client";

export interface EffectiveStatBonus {
  source: string;
  statType: string;
  amount: number;
  multiplier: number;
}

export interface EffectiveStats {
  strength: number;
  dexterity: number;
  intelligence: number;
  luck: number;
  maxHP: number;
  maxMP: number;
  weaponAttack: number;
  weaponDefense: number;
  magicAttack: number;
  magicDefense: number;
  accuracy: number;
  avoidability: number;
  speed: number;
  jump: number;
  bonuses: EffectiveStatBonus[];
}

interface EffectiveStatsResource {
  id: string;
  type: string;
  attributes: {
    strength: number;
    dexterity: number;
    luck: number;
    intelligence: number;
    maxHP: number;
    maxMP: number;
    weaponAttack: number;
    weaponDefense: number;
    magicAttack: number;
    magicDefense: number;
    accuracy: number;
    avoidability: number;
    speed: number;
    jump: number;
    bonuses?: EffectiveStatBonus[];
  };
}

export const characterEffectiveStatsService = {
  // The atlas-effective-stats route is keyed by world + channel + character.
  // Offline characters have no current channel; pass 0 so the service still
  // computes base + equip bonuses (channel-scoped buffs simply contribute 0).
  async getByCharacter(
    worldId: number,
    characterId: string,
    channelId = 0,
  ): Promise<EffectiveStats> {
    const stats = await api.getOne<EffectiveStatsResource>(
      `/api/worlds/${worldId}/channels/${channelId}/characters/${characterId}/stats`,
    );
    const a = stats.attributes;
    return {
      strength: a.strength,
      dexterity: a.dexterity,
      intelligence: a.intelligence,
      luck: a.luck,
      maxHP: a.maxHP,
      maxMP: a.maxMP,
      weaponAttack: a.weaponAttack,
      weaponDefense: a.weaponDefense,
      magicAttack: a.magicAttack,
      magicDefense: a.magicDefense,
      accuracy: a.accuracy,
      avoidability: a.avoidability,
      speed: a.speed,
      jump: a.jump,
      bonuses: a.bonuses ?? [],
    };
  },
};
