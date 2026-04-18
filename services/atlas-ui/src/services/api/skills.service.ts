import { api } from "@/lib/api/client";
import type { Tenant } from "@/types/models/tenant";

interface SkillData {
  id: string;
  type: string;
  attributes: {
    name: string;
    action: boolean;
    element: string;
    animationTime: number;
  };
}

const BASE_PATH = "/api/data/skills";

export const skillsService = {
  async getSkillName(id: string, _tenant: Tenant): Promise<string> {
    const skill = await api.getOne<SkillData>(`${BASE_PATH}/${id}`);
    return skill.attributes.name;
  },
};
