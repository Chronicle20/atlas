import { api } from "@/lib/api/client";

export interface CharacterSkill {
  id: string;
  level: number;
  masterLevel: number;
  expiration: string;
  cooldownExpiresAt: string;
}

interface CharacterSkillResource {
  id: string;
  type: string;
  attributes: {
    level: number;
    masterLevel: number;
    expiration: string;
    cooldownExpiresAt: string;
  };
}

const BASE_PATH = "/api/characters";

export const characterSkillsService = {
  async getByCharacterId(characterId: string): Promise<CharacterSkill[]> {
    const list = await api.getList<CharacterSkillResource>(
      `${BASE_PATH}/${characterId}/skills`,
    );
    return list.map((r) => ({
      id: r.id,
      level: r.attributes.level,
      masterLevel: r.attributes.masterLevel,
      expiration: r.attributes.expiration,
      cooldownExpiresAt: r.attributes.cooldownExpiresAt,
    }));
  },
};
