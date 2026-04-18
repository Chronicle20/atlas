import { BaseService } from './base.service';
import { api } from '@/lib/api/client';
import type { Tenant } from '@/types/models/tenant';

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

class SkillsService extends BaseService {
  protected basePath = '/api/data/skills';

  async getSkillName(id: string, tenant: Tenant): Promise<string> {
    const skill = await this.getById<SkillData>(id);
    return skill.attributes.name;
  }
}

export const skillsService = new SkillsService();
