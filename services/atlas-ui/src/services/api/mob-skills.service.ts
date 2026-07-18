import { api } from "@/lib/api/client";
import { fetchPaged } from "@/services/api/pagination";

interface MobSkillSummary {
  id: string;
  type: string;
  attributes: {
    name: string;
  };
}

export interface MobSkillDetailAttributes {
  name: string;
  mp_con: number;
  duration: number;
  hp: number;
  x: number;
  y: number;
  prop: number;
  interval: number;
  count: number;
  limit: number;
  lt_x: number;
  lt_y: number;
  rb_x: number;
  rb_y: number;
  summon_effect: number;
  summons: number[];
}

interface MobSkillDetailData {
  id: string;
  type: string;
  attributes: MobSkillDetailAttributes;
}

const BASE_PATH = "/api/data/mob-skills";

export const mobSkillsService = {
  async getMobSkillName(skillId: number): Promise<string> {
    const result = await fetchPaged<MobSkillSummary>(
      `${BASE_PATH}/${skillId}`,
      { number: 1, size: 1 },
    );
    return result.data[0]?.attributes.name ?? "";
  },

  async getMobSkillDetail(
    skillId: number,
    level: number,
  ): Promise<MobSkillDetailAttributes> {
    const data = await api.getOne<MobSkillDetailData>(
      `${BASE_PATH}/${skillId}/${level}`,
    );
    return data.attributes;
  },
};
