import { api } from "@/lib/api/client";

interface MobSkillData {
  id: string;
  type: string;
  attributes: {
    name: string;
  };
}

const BASE_PATH = "/api/data/mob-skills";

export const mobSkillsService = {
  async getMobSkillName(skillId: number): Promise<string> {
    const rows = await api.getList<MobSkillData>(`${BASE_PATH}/${skillId}`);
    return rows[0]?.attributes.name ?? "";
  },
};
