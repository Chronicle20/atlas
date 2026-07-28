import { api } from "@/lib/api/client";
import {
  buildQueryString,
  type ServiceOptions,
  type QueryOptions,
} from "@/lib/api/query-params";
import { fetchAll, fetchPaged } from "@/services/api/pagination";
import type { MonsterData, MonsterSpawnMapData } from "@/types/models/monster";

const BASE_PATH = "/api/data/monsters";

// MonstersPage advertises "Results are limited to 50 entries" — searchMonsters
// is a bounded single-page lookup, not a drained browse (task-117).
const SEARCH_RESULT_LIMIT = 50;

function sortById<T extends { id: string }>(rows: T[]): T[] {
  return rows.sort((a, b) => parseInt(a.id) - parseInt(b.id));
}

/**
 * Drain every page for a genuinely unbounded fetch (task-117). Used by
 * `getAllMonsters` — consumers that need the whole collection.
 */
async function fetchAllSorted(options?: QueryOptions): Promise<MonsterData[]> {
  const monsters = await fetchAll<MonsterData>(
    `${BASE_PATH}${buildQueryString(options)}`,
    undefined,
    options,
  );
  return sortById(monsters);
}

/**
 * Fetch a single bounded page for the search box (task-117) — matches the
 * "Results are limited to 50 entries" copy on MonstersPage.
 */
async function fetchSearchPage(options?: QueryOptions): Promise<MonsterData[]> {
  const result = await fetchPaged<MonsterData>(
    `${BASE_PATH}${buildQueryString(options)}`,
    { number: 1, size: SEARCH_RESULT_LIMIT },
    options,
  );
  return sortById(result.data);
}

export const monstersService = {
  async getAllMonsters(options?: QueryOptions): Promise<MonsterData[]> {
    return fetchAllSorted(options);
  },

  async getMonsterById(
    id: string,
    options?: ServiceOptions,
  ): Promise<MonsterData> {
    return api.getOne<MonsterData>(`${BASE_PATH}/${id}`, options);
  },

  async getMonsterName(id: string): Promise<string> {
    const monster = await api.getOne<MonsterData>(`${BASE_PATH}/${id}`);
    return monster.attributes.name;
  },

  async searchMonsters(
    query: string,
    options?: QueryOptions,
  ): Promise<MonsterData[]> {
    return fetchSearchPage({ ...options, search: query });
  },

  /**
   * Get every map a monster spawns on, draining all pages (task-117) — the
   * detail-page widget renders the full spawn list, not a page at a time.
   */
  async getMonsterMaps(monsterId: string): Promise<MonsterSpawnMapData[]> {
    return fetchAll<MonsterSpawnMapData>(`${BASE_PATH}/${monsterId}/maps`);
  },
};
