import { api } from "@/lib/api/client";

export interface SkillEffectStatup {
  type: string;
  amount: number;
}

// Partial mirror of atlas-data effect.RestModel (services/atlas-data/.../skill/effect/rest.go):
// the scalar magnitude fields the skill browser renders. Structured fields
// (lt/rb/monsterStatus/cardStats/cureAbnormalStatuses) are intentionally omitted.
// Keys match the Go JSON tags exactly — keep casing (e.g. MPConsume, hpR, MHPRRate).
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
  MHPRRate?: number;
  MMPRRate?: number;
  mhpr?: number;
  mmpr?: number;
  HPConsume?: number;
  MPConsume?: number;
  duration?: number;
  overTime?: boolean;
  cooldown?: number;
  x?: number;
  y?: number;
  mobCount?: number;
  moneyConsume?: number;
  morphId?: number;
  prop?: number;
  itemConsume?: number;
  itemConsumeAmount?: number;
  damage?: number;
  attackCount?: number;
  fixDamage?: number;
  bulletCount?: number;
  bulletConsume?: number;
  statups?: SkillEffectStatup[];
}

export interface SkillDefinition {
  id: number;
  name: string;
  description: string; // "" when atlas-data not yet upgraded
  action: boolean;
  element: string;
  animationTime: number;
  maxLevel?: number; // optional: older atlas-data responses omit it
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
    maxLevel?: number;
    effects?: SkillEffect[];
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
      ...(skill.attributes.maxLevel !== undefined && { maxLevel: skill.attributes.maxLevel }),
      effects: skill.attributes.effects ?? [],
    };
  },
};
