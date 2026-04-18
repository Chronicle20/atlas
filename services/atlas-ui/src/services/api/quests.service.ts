import { api } from "@/lib/api/client";
import { type ServiceOptions, type QueryOptions } from "@/lib/api/query-params";
import type { QuestDefinition, QuestAttributes } from "@/types/models/quest";

const BASE_PATH = "/api/data/quests";

export interface QuestQueryOptions extends QueryOptions {
  category?: string;
  autoStart?: boolean;
  autoComplete?: boolean;
  minLevel?: number;
  maxLevel?: number;
}

function applyFilters(quests: QuestDefinition[], options?: QuestQueryOptions): QuestDefinition[] {
  if (!options) return quests;
  let filtered = quests;

  if (options.category) {
    filtered = filtered.filter(q =>
      q.attributes.parent?.toLowerCase().includes(options.category!.toLowerCase()),
    );
  }
  if (options.autoStart !== undefined) {
    filtered = filtered.filter(q => q.attributes.autoStart === options.autoStart);
  }
  if (options.autoComplete !== undefined) {
    filtered = filtered.filter(q => q.attributes.autoComplete === options.autoComplete);
  }
  if (options.minLevel !== undefined) {
    filtered = filtered.filter(q =>
      (q.attributes.startRequirements.levelMin || 0) >= options.minLevel!,
    );
  }
  if (options.maxLevel !== undefined) {
    filtered = filtered.filter(q =>
      (q.attributes.startRequirements.levelMax || 999) <= options.maxLevel!,
    );
  }
  if (options.search) {
    const s = options.search.toLowerCase();
    filtered = filtered.filter(q =>
      q.id.includes(s) ||
      q.attributes.name?.toLowerCase().includes(s) ||
      q.attributes.parent?.toLowerCase().includes(s),
    );
  }
  return filtered;
}

export const questsService = {
  async getAllQuests(options?: QuestQueryOptions): Promise<QuestDefinition[]> {
    const quests = await api.getList<QuestDefinition>(BASE_PATH, options);
    return applyFilters(quests, options).sort((a, b) => parseInt(a.id) - parseInt(b.id));
  },

  async getQuestById(questId: string, options?: ServiceOptions): Promise<QuestDefinition> {
    return api.getOne<QuestDefinition>(`${BASE_PATH}/${questId}`, options);
  },

  async getCategories(options?: ServiceOptions): Promise<string[]> {
    const quests = await questsService.getAllQuests( options);
    const categories = new Set<string>();
    quests.forEach(q => {
      if (q.attributes.parent) categories.add(q.attributes.parent);
    });
    return Array.from(categories).sort();
  },

  async getQuestsByCategory(category: string, options?: ServiceOptions): Promise<QuestDefinition[]> {
    return questsService.getAllQuests({ ...options, category });
  },

  async getAutoStartQuests(options?: ServiceOptions): Promise<QuestDefinition[]> {
    return questsService.getAllQuests({ ...options, autoStart: true });
  },

  async getAutoCompleteQuests(options?: ServiceOptions): Promise<QuestDefinition[]> {
    return questsService.getAllQuests({ ...options, autoComplete: true });
  },

  async getQuestsByNpc(npcId: number, options?: ServiceOptions): Promise<QuestDefinition[]> {
    const quests = await questsService.getAllQuests( options);
    return quests.filter(q =>
      q.attributes.startRequirements.npcId === npcId ||
      q.attributes.endRequirements.npcId === npcId ||
      q.attributes.startActions.npcId === npcId ||
      q.attributes.endActions.npcId === npcId,
    );
  },

  async getQuestsRewardingItem(itemId: number, options?: ServiceOptions): Promise<QuestDefinition[]> {
    const quests = await questsService.getAllQuests( options);
    return quests.filter(q =>
      q.attributes.startActions.items?.some(i => i.id === itemId) ||
      q.attributes.endActions.items?.some(i => i.id === itemId),
    );
  },

  async getQuestsRequiringItem(itemId: number, options?: ServiceOptions): Promise<QuestDefinition[]> {
    const quests = await questsService.getAllQuests( options);
    return quests.filter(q =>
      q.attributes.startRequirements.items?.some(i => i.id === itemId) ||
      q.attributes.endRequirements.items?.some(i => i.id === itemId),
    );
  },

  async getQuestsRequiringMob(mobId: number, options?: ServiceOptions): Promise<QuestDefinition[]> {
    const quests = await questsService.getAllQuests( options);
    return quests.filter(q =>
      q.attributes.startRequirements.mobs?.some(m => m.id === mobId) ||
      q.attributes.endRequirements.mobs?.some(m => m.id === mobId),
    );
  },

  async getQuestChain(startQuestId: string, options?: ServiceOptions): Promise<QuestDefinition[]> {
    const chain: QuestDefinition[] = [];
    let currentQuestId: string | null = startQuestId;

    while (currentQuestId) {
      try {
        const quest = await questsService.getQuestById( currentQuestId, options);
        chain.push(quest);
        const nextQuestId = quest.attributes.endActions.nextQuest;
        currentQuestId = nextQuestId ? nextQuestId.toString() : null;
        if (chain.length > 100) {
          console.warn("Quest chain exceeded 100 quests, stopping to prevent infinite loop");
          break;
        }
      } catch {
        break;
      }
    }

    return chain;
  },
};

export type { QuestDefinition, QuestAttributes };
