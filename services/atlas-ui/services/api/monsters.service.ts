import { BaseService, type ServiceOptions, type QueryOptions } from './base.service';
import { api } from '@/lib/api/client';
import type { Tenant } from '@/types/models/tenant';
import type { MonsterData } from '@/types/models/monster';

class MonstersService extends BaseService {
  protected basePath = '/api/data/monsters';

  async getAllMonsters(tenant: Tenant, options?: QueryOptions): Promise<MonsterData[]> {
    api.setTenant(tenant);
    const monsters = await this.getAll<MonsterData>(options);
    return monsters.sort((a, b) => parseInt(a.id) - parseInt(b.id));
  }

  async getMonsterById(id: string, tenant: Tenant, options?: ServiceOptions): Promise<MonsterData> {
    api.setTenant(tenant);
    return this.getById<MonsterData>(id, options);
  }

  async getMonsterName(id: string, tenant: Tenant): Promise<string> {
    api.setTenant(tenant);
    const monster = await this.getById<MonsterData>(id);
    return monster.attributes.name;
  }
}

export const monstersService = new MonstersService();
