import { api } from "@/lib/api/client";
import { type ServiceOptions, type QueryOptions } from "@/lib/api/query-params";
import type { CharacterQuestStatus, QuestState } from "@/types/models/quest";
import type { Tenant } from "@/types/models/tenant";

const BASE_PATH = "/api/characters";

export interface QuestStatusQueryOptions extends QueryOptions {
  state?: QuestState;
}

export const questStatusService = {
  async getByCharacterId(
    _tenant: Tenant,
    characterId: string,
    options?: QuestStatusQueryOptions,
  ): Promise<CharacterQuestStatus[]> {
    const url = `${BASE_PATH}/${characterId}/quests`;
    const statuses = await api.getList<CharacterQuestStatus>(url, options);
    if (options?.state !== undefined) {
      return statuses.filter(s => s.attributes.state === options.state);
    }
    return statuses;
  },

  async getStartedQuests(_tenant: Tenant, characterId: string, options?: ServiceOptions): Promise<CharacterQuestStatus[]> {
    return api.getList<CharacterQuestStatus>(`${BASE_PATH}/${characterId}/quests/started`, options);
  },

  async getCompletedQuests(_tenant: Tenant, characterId: string, options?: ServiceOptions): Promise<CharacterQuestStatus[]> {
    return api.getList<CharacterQuestStatus>(`${BASE_PATH}/${characterId}/quests/completed`, options);
  },

  async getQuestStatus(
    _tenant: Tenant,
    characterId: string,
    questId: string,
    options?: ServiceOptions,
  ): Promise<CharacterQuestStatus> {
    return api.getOne<CharacterQuestStatus>(`${BASE_PATH}/${characterId}/quests/${questId}`, options);
  },

  async hasStartedQuest(tenant: Tenant, characterId: string, questId: string, options?: ServiceOptions): Promise<boolean> {
    try {
      const status = await questStatusService.getQuestStatus(tenant, characterId, questId, options);
      return status.attributes.state === 1;
    } catch {
      return false;
    }
  },

  async hasCompletedQuest(tenant: Tenant, characterId: string, questId: string, options?: ServiceOptions): Promise<boolean> {
    try {
      const status = await questStatusService.getQuestStatus(tenant, characterId, questId, options);
      return status.attributes.state === 2;
    } catch {
      return false;
    }
  },

  async getCompletionCount(tenant: Tenant, characterId: string, questId: string, options?: ServiceOptions): Promise<number> {
    try {
      const status = await questStatusService.getQuestStatus(tenant, characterId, questId, options);
      return status.attributes.completedCount;
    } catch {
      return 0;
    }
  },

  async forfeitQuest(_tenant: Tenant, characterId: string, questId: string, options?: ServiceOptions): Promise<void> {
    await api.post<void>(`${BASE_PATH}/${characterId}/quests/${questId}/forfeit`, {}, options);
  },

  async getQuestStats(
    tenant: Tenant,
    characterId: string,
    options?: ServiceOptions,
  ): Promise<{ started: number; completed: number; total: number }> {
    const allStatuses = await questStatusService.getByCharacterId(tenant, characterId, options);
    const started = allStatuses.filter(s => s.attributes.state === 1).length;
    const completed = allStatuses.filter(s => s.attributes.state === 2).length;
    return { started, completed, total: allStatuses.length };
  },
};

export type { CharacterQuestStatus, QuestState };
