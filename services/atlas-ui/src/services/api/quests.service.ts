/**
 * Quests Service
 *
 * Provides quest definition data from atlas-data service:
 * - Quest listing with search and filtering
 * - Quest detail retrieval with requirements and rewards
 * - Integration with tenant context
 */

import { BaseService, type ServiceOptions, type QueryOptions } from './base.service';
import type { QuestDefinition, QuestAttributes } from '@/types/models/quest';
import type { Tenant } from '@/types/models/tenant';

/**
 * Query options specific to quests
 */
export interface QuestQueryOptions extends QueryOptions {
    /** Filter by category (parent field) */
    category?: string;
    /** Filter by auto-start flag */
    autoStart?: boolean;
    /** Filter by auto-complete flag */
    autoComplete?: boolean;
    /** Filter by minimum level requirement */
    minLevel?: number;
    /** Filter by maximum level requirement */
    maxLevel?: number;
}

/**
 * Quests service class for fetching quest definitions
 */
class QuestsService extends BaseService {
    protected basePath = '/api/data/quests';

    /**
     * Process service options with quest-specific defaults
     */
    private processServiceOptions(options?: ServiceOptions): ServiceOptions {
        return {
            ...options,
            // Quest data is static, can cache longer
            cacheConfig: options?.cacheConfig || {
                ttl: 10 * 60 * 1000, // 10 minutes
                staleWhileRevalidate: true,
                maxStaleTime: 2 * 60 * 1000, // 2 minutes
            },
        };
    }

    /**
     * Get all quest definitions
     */
    async getAllQuests(tenant: Tenant, options?: QuestQueryOptions): Promise<QuestDefinition[]> {
        const { api } = await import('@/lib/api/client');
        api.setTenant(tenant);

        const processedOptions = this.processServiceOptions(options);
        const quests = await api.getList<QuestDefinition>(this.basePath, processedOptions);

        // Apply client-side filtering for options not supported by API
        let filtered = quests;

        if (options?.category) {
            filtered = filtered.filter(q =>
                q.attributes.parent?.toLowerCase().includes(options.category!.toLowerCase())
            );
        }

        if (options?.autoStart !== undefined) {
            filtered = filtered.filter(q => q.attributes.autoStart === options.autoStart);
        }

        if (options?.autoComplete !== undefined) {
            filtered = filtered.filter(q => q.attributes.autoComplete === options.autoComplete);
        }

        if (options?.minLevel !== undefined) {
            filtered = filtered.filter(q =>
                (q.attributes.startRequirements.levelMin || 0) >= options.minLevel!
            );
        }

        if (options?.maxLevel !== undefined) {
            filtered = filtered.filter(q =>
                (q.attributes.startRequirements.levelMax || 999) <= options.maxLevel!
            );
        }

        // Apply search filter
        if (options?.search) {
            const searchLower = options.search.toLowerCase();
            filtered = filtered.filter(q =>
                q.id.includes(searchLower) ||
                q.attributes.name?.toLowerCase().includes(searchLower) ||
                q.attributes.parent?.toLowerCase().includes(searchLower)
            );
        }

        // Sort by ID by default
        return filtered.sort((a, b) => parseInt(a.id) - parseInt(b.id));
    }

    /**
     * Get a single quest definition by ID
     */
    async getQuestById(tenant: Tenant, questId: string, options?: ServiceOptions): Promise<QuestDefinition> {
        const { api } = await import('@/lib/api/client');
        api.setTenant(tenant);

        const processedOptions = this.processServiceOptions(options);
        return api.getOne<QuestDefinition>(`${this.basePath}/${questId}`, processedOptions);
    }

    /**
     * Get unique categories (parent field values)
     */
    async getCategories(tenant: Tenant, options?: ServiceOptions): Promise<string[]> {
        const quests = await this.getAllQuests(tenant, options);
        const categories = new Set<string>();

        quests.forEach(q => {
            if (q.attributes.parent) {
                categories.add(q.attributes.parent);
            }
        });

        return Array.from(categories).sort();
    }

    /**
     * Get quests by category
     */
    async getQuestsByCategory(tenant: Tenant, category: string, options?: ServiceOptions): Promise<QuestDefinition[]> {
        return this.getAllQuests(tenant, { ...options, category });
    }

    /**
     * Get auto-start quests
     */
    async getAutoStartQuests(tenant: Tenant, options?: ServiceOptions): Promise<QuestDefinition[]> {
        return this.getAllQuests(tenant, { ...options, autoStart: true });
    }

    /**
     * Get auto-complete quests
     */
    async getAutoCompleteQuests(tenant: Tenant, options?: ServiceOptions): Promise<QuestDefinition[]> {
        return this.getAllQuests(tenant, { ...options, autoComplete: true });
    }

    /**
     * Get quests that require a specific NPC
     */
    async getQuestsByNpc(tenant: Tenant, npcId: number, options?: ServiceOptions): Promise<QuestDefinition[]> {
        const quests = await this.getAllQuests(tenant, options);
        return quests.filter(q =>
            q.attributes.startRequirements.npcId === npcId ||
            q.attributes.endRequirements.npcId === npcId ||
            q.attributes.startActions.npcId === npcId ||
            q.attributes.endActions.npcId === npcId
        );
    }

    /**
     * Get quests that reward a specific item
     */
    async getQuestsRewardingItem(tenant: Tenant, itemId: number, options?: ServiceOptions): Promise<QuestDefinition[]> {
        const quests = await this.getAllQuests(tenant, options);
        return quests.filter(q =>
            q.attributes.startActions.items?.some(i => i.id === itemId) ||
            q.attributes.endActions.items?.some(i => i.id === itemId)
        );
    }

    /**
     * Get quests that require a specific item
     */
    async getQuestsRequiringItem(tenant: Tenant, itemId: number, options?: ServiceOptions): Promise<QuestDefinition[]> {
        const quests = await this.getAllQuests(tenant, options);
        return quests.filter(q =>
            q.attributes.startRequirements.items?.some(i => i.id === itemId) ||
            q.attributes.endRequirements.items?.some(i => i.id === itemId)
        );
    }

    /**
     * Get quests that require killing a specific mob
     */
    async getQuestsRequiringMob(tenant: Tenant, mobId: number, options?: ServiceOptions): Promise<QuestDefinition[]> {
        const quests = await this.getAllQuests(tenant, options);
        return quests.filter(q =>
            q.attributes.startRequirements.mobs?.some(m => m.id === mobId) ||
            q.attributes.endRequirements.mobs?.some(m => m.id === mobId)
        );
    }

    /**
     * Get quest chain starting from a specific quest
     */
    async getQuestChain(tenant: Tenant, startQuestId: string, options?: ServiceOptions): Promise<QuestDefinition[]> {
        const chain: QuestDefinition[] = [];
        let currentQuestId: string | null = startQuestId;

        while (currentQuestId) {
            try {
                const quest = await this.getQuestById(tenant, currentQuestId, options);
                chain.push(quest);

                // Check for next quest in chain
                const nextQuestId = quest.attributes.endActions.nextQuest;
                currentQuestId = nextQuestId ? nextQuestId.toString() : null;

                // Prevent infinite loops
                if (chain.length > 100) {
                    console.warn('Quest chain exceeded 100 quests, stopping to prevent infinite loop');
                    break;
                }
            } catch {
                // Quest not found, end of chain
                break;
            }
        }

        return chain;
    }
}

// Create and export a singleton instance
export const questsService = new QuestsService();

// Export types
export type { QuestDefinition, QuestAttributes };
