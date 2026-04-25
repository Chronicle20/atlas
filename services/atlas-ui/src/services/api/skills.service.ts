import { api } from "@/lib/api/client";

export interface SkillEffectStatup {
  type: string;
  amount: number;
}

// Mirrors atlas-data effect.RestModel — extend with additional fields as needed.
export interface SkillEffect {
  weaponAttack?: number;
  magicAttack?: number;
  weaponDefense?: number;
  magicDefense?: number;
  accuracy?: number;
  avoidability?: number;
  speed?: number;
  jump?: number;
  hp?: number;
  mp?: number;
  hpR?: number;
  mpR?: number;
  duration?: number;
  overTime?: boolean;
  cooldown?: number;
  MPConsume?: number;
  HPConsume?: number;
  statups?: SkillEffectStatup[];
}

export interface SkillDefinition {
  id: number;
  name: string;
  description: string; // "" when atlas-data not yet upgraded
  action: boolean;
  element: string;
  animationTime: number;
  effects: SkillEffect[];
}

interface SkillResource {
  id: string;
  type: string;
  attributes: {
    name: string;
    description?: string;
    action: boolean;
    element: string;
    animationTime: number;
    effects: SkillEffect[];
  };
}

const BASE_PATH = "/api/data/skills";

export const skillsService = {
  async getSkillName(id: string): Promise<string> {
    const skill = await api.getOne<SkillResource>(`${BASE_PATH}/${id}`);
    return skill.attributes.name;
  },

  async getSkillById(id: string): Promise<SkillDefinition> {
    const skill = await api.getOne<SkillResource>(`${BASE_PATH}/${id}`);
    return {
      id: parseInt(skill.id, 10),
      name: skill.attributes.name,
      description: skill.attributes.description ?? "",
      action: skill.attributes.action,
      element: skill.attributes.element,
      animationTime: skill.attributes.animationTime,
      effects: skill.attributes.effects ?? [],
    };
  },
};
