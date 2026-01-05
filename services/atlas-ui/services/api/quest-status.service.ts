/**
 * Quest Status Service
 *
 * Provides character quest status data from atlas-quest service:
 * - Character quest status listing (started, completed)
 * - Quest progress tracking
 * - Integration with tenant context
 */

import { BaseService, type ServiceOptions, type QueryOptions } from './base.service';
import type { CharacterQuestStatus, QuestState } from '@/types/models/quest';
import type { Tenant } from '@/types/models/tenant';

/**
 * Query options specific to quest statuses
 */
export interface QuestStatusQueryOptions extends QueryOptions {
    /** Filter by quest state */
    state?: QuestState;
}

/**
 * Quest Status service class for fetching character quest data
 */
class QuestStatusService extends BaseService {
    protected basePath = '/api/characters';

    /**
     * Process service options with quest status-specific defaults
     */
    private processServiceOptions(options?: ServiceOptions): ServiceOptions {
        return {
            ...options,
            // Quest status data changes, use shorter cache
            cacheConfig: options?.cacheConfig || {
                ttl: 1 * 60 * 1000, // 1 minute
                staleWhileRevalidate: true,
                maxStaleTime: 30 * 1000, // 30 seconds
            },
        };
    }

    /**
     * Get all quest statuses for a character
     */
    async getByCharacterId(
        tenant: Tenant,
        characterId: string,
        options?: QuestStatusQueryOptions
    ): Promise<CharacterQuestStatus[]> {
        const { api } = await import('@/lib/api/client');
        api.setTenant(tenant);

        const processedOptions = this.processServiceOptions(options);
        const url = `${this.basePath}/${characterId}/quests`;
        const statuses = await api.getList<CharacterQuestStatus>(url, processedOptions);

        // Apply state filter if specified
        if (options?.state !== undefined) {
            return statuses.filter(s => s.attributes.state === options.state);
        }

        return statuses;
    }

    /**
     * Get started quests for a character
     */
    async getStartedQuests(
        tenant: Tenant,
        characterId: string,
        options?: ServiceOptions
    ): Promise<CharacterQuestStatus[]> {
        const { api } = await import('@/lib/api/client');
        api.setTenant(tenant);

        const processedOptions = this.processServiceOptions(options);
        const url = `${this.basePath}/${characterId}/quests/started`;
        return api.getList<CharacterQuestStatus>(url, processedOptions);
    }

    /**
     * Get completed quests for a character
     */
    async getCompletedQuests(
        tenant: Tenant,
        characterId: string,
        options?: ServiceOptions
    ): Promise<CharacterQuestStatus[]> {
        const { api } = await import('@/lib/api/client');
        api.setTenant(tenant);

        const processedOptions = this.processServiceOptions(options);
        const url = `${this.basePath}/${characterId}/quests/completed`;
        return api.getList<CharacterQuestStatus>(url, processedOptions);
    }

    /**
     * Get a specific quest status for a character
     */
    async getQuestStatus(
        tenant: Tenant,
        characterId: string,
        questId: string,
        options?: ServiceOptions
    ): Promise<CharacterQuestStatus> {
        const { api } = await import('@/lib/api/client');
        api.setTenant(tenant);

        const processedOptions = this.processServiceOptions(options);
        const url = `${this.basePath}/${characterId}/quests/${questId}`;
        return api.getOne<CharacterQuestStatus>(url, processedOptions);
    }

    /**
     * Check if a character has started a quest
     */
    async hasStartedQuest(
        tenant: Tenant,
        characterId: string,
        questId: string,
        options?: ServiceOptions
    ): Promise<boolean> {
        try {
            const status = await this.getQuestStatus(tenant, characterId, questId, options);
            return status.attributes.state === 1; // Started
        } catch {
            return false;
        }
    }

    /**
     * Check if a character has completed a quest
     */
    async hasCompletedQuest(
        tenant: Tenant,
        characterId: string,
        questId: string,
        options?: ServiceOptions
    ): Promise<boolean> {
        try {
            const status = await this.getQuestStatus(tenant, characterId, questId, options);
            return status.attributes.state === 2; // Completed
        } catch {
            return false;
        }
    }

    /**
     * Get quest completion count for a character
     */
    async getCompletionCount(
        tenant: Tenant,
        characterId: string,
        questId: string,
        options?: ServiceOptions
    ): Promise<number> {
        try {
            const status = await this.getQuestStatus(tenant, characterId, questId, options);
            return status.attributes.completedCount;
        } catch {
            return 0;
        }
    }

    /**
     * Get quest statistics for a character
     */
    async getQuestStats(
        tenant: Tenant,
        characterId: string,
        options?: ServiceOptions
    ): Promise<{ started: number; completed: number; total: number }> {
        const allStatuses = await this.getByCharacterId(tenant, characterId, options);

        const started = allStatuses.filter(s => s.attributes.state === 1).length;
        const completed = allStatuses.filter(s => s.attributes.state === 2).length;

        return {
            started,
            completed,
            total: allStatuses.length,
        };
    }
}

// Create and export a singleton instance
export const questStatusService = new QuestStatusService();

// Export types
export type { CharacterQuestStatus, QuestState };
