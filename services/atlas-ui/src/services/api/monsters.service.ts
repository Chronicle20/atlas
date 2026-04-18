import { api } from "@/lib/api/client";
import { buildQueryString, type ServiceOptions, type QueryOptions } from "@/lib/api/query-params";
import type { Tenant } from "@/types/models/tenant";
import type { MonsterData } from "@/types/models/monster";

const BASE_PATH = "/api/data/monsters";

async function fetchAllSorted(options?: QueryOptions): Promise<MonsterData[]> {
  const monsters = await api.getList<MonsterData>(`${BASE_PATH}${buildQueryString(options)}`, options);
  return monsters.sort((a, b) => parseInt(a.id) - parseInt(b.id));
}

export const monstersService = {
  async getAllMonsters(_tenant: Tenant, options?: QueryOptions): Promise<MonsterData[]> {
    return fetchAllSorted(options);
  },

  async getMonsterById(id: string, _tenant: Tenant, options?: ServiceOptions): Promise<MonsterData> {
    return api.getOne<MonsterData>(`${BASE_PATH}/${id}`, options);
  },

  async getMonsterName(id: string, _tenant: Tenant): Promise<string> {
    const monster = await api.getOne<MonsterData>(`${BASE_PATH}/${id}`);
    return monster.attributes.name;
  },

  async searchMonsters(query: string, _tenant: Tenant, options?: QueryOptions): Promise<MonsterData[]> {
    return fetchAllSorted({ ...options, search: query });
  },
};
