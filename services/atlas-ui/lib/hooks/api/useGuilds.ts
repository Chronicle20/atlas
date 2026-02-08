/**
 * React Query hooks for guild data access
 *
 * Provides optimized data fetching and caching for:
 * - Guild listing and detail queries
 * - Guild search and filtering
 * - Guild rankings and statistics
 *
 * Note: The backend guild API is read-only. Guild mutations
 * are handled through in-game systems via Kafka events.
 */

import { useQuery, useQueryClient, type UseQueryResult } from '@tanstack/react-query';
import { guildsService, type Guild, type GuildAttributes, type GuildMember } from '@/services/api/guilds.service';
import type { ServiceOptions, QueryOptions } from '@/services/api/base.service';
import type { Tenant } from '@/types/models/tenant';

// Query keys for consistent cache management
export const guildKeys = {
  all: ['guilds'] as const,
  lists: () => [...guildKeys.all, 'list'] as const,
  list: (tenant: Tenant | null, options?: QueryOptions) => [...guildKeys.lists(), tenant?.id, options] as const,
  details: () => [...guildKeys.all, 'detail'] as const,
  detail: (tenant: Tenant | null, id: string) => [...guildKeys.details(), tenant?.id, id] as const,

  // Specialized query keys
  searches: () => [...guildKeys.all, 'search'] as const,
  search: (tenant: Tenant | null, searchTerm: string, worldId?: number) => [...guildKeys.searches(), tenant?.id, searchTerm, worldId] as const,
  byWorld: (tenant: Tenant | null, worldId: number) => [...guildKeys.lists(), tenant?.id, 'world', worldId] as const,
  withSpace: (tenant: Tenant | null, worldId?: number) => [...guildKeys.lists(), tenant?.id, 'space', worldId] as const,
  rankings: (tenant: Tenant | null, worldId?: number, limit?: number) => [...guildKeys.lists(), tenant?.id, 'rankings', worldId, limit] as const,
};

// ============================================================================
// GUILD QUERY HOOKS
// ============================================================================

/**
 * Hook to fetch all guilds for a tenant
 */
export function useGuilds(tenant: Tenant | null, options?: QueryOptions): UseQueryResult<Guild[], Error> {
  return useQuery({
    queryKey: guildKeys.list(tenant, options),
    queryFn: () => tenant ? guildsService.getAll(tenant, options) : Promise.reject(new Error('Tenant is required')),
    enabled: !!tenant?.id,
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
  });
}

/**
 * Hook to fetch a specific guild by ID
 */
export function useGuild(tenant: Tenant, guildId: string, options?: ServiceOptions): UseQueryResult<Guild, Error> {
  return useQuery({
    queryKey: guildKeys.detail(tenant, guildId),
    queryFn: () => guildsService.getById(tenant, guildId, options),
    enabled: !!tenant?.id && !!guildId,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

/**
 * Hook to fetch guilds by world ID
 */
export function useGuildsByWorld(tenant: Tenant, worldId: number, options?: ServiceOptions): UseQueryResult<Guild[], Error> {
  return useQuery({
    queryKey: guildKeys.byWorld(tenant, worldId),
    queryFn: () => guildsService.getByWorld(tenant, worldId, options),
    enabled: !!tenant?.id && worldId !== undefined,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

/**
 * Hook to search guilds by name
 */
export function useGuildSearch(
  tenant: Tenant,
  searchTerm: string,
  worldId?: number,
  options?: ServiceOptions
): UseQueryResult<Guild[], Error> {
  return useQuery({
    queryKey: guildKeys.search(tenant, searchTerm, worldId),
    queryFn: () => guildsService.search(tenant, searchTerm, worldId, options),
    enabled: !!tenant?.id && !!searchTerm,
    staleTime: 2 * 60 * 1000,
    gcTime: 5 * 60 * 1000,
  });
}

/**
 * Hook to fetch guilds with available space
 */
export function useGuildsWithSpace(tenant: Tenant, worldId?: number, options?: ServiceOptions): UseQueryResult<Guild[], Error> {
  return useQuery({
    queryKey: guildKeys.withSpace(tenant, worldId),
    queryFn: () => guildsService.getWithSpace(tenant, worldId, options),
    enabled: !!tenant?.id,
    staleTime: 3 * 60 * 1000,
    gcTime: 8 * 60 * 1000,
  });
}

/**
 * Hook to fetch guild rankings
 */
export function useGuildRankings(
  tenant: Tenant,
  worldId?: number,
  limit = 50,
  options?: ServiceOptions
): UseQueryResult<Guild[], Error> {
  return useQuery({
    queryKey: guildKeys.rankings(tenant, worldId, limit),
    queryFn: () => guildsService.getRankings(tenant, worldId, limit, options),
    enabled: !!tenant?.id,
    staleTime: 10 * 60 * 1000,
    gcTime: 15 * 60 * 1000,
  });
}

/**
 * Hook to check if a guild exists
 */
export function useGuildExists(tenant: Tenant, guildId: string, options?: ServiceOptions): UseQueryResult<boolean, Error> {
  return useQuery({
    queryKey: [...guildKeys.detail(tenant, guildId), 'exists'],
    queryFn: () => guildsService.exists(tenant, guildId, options),
    enabled: !!tenant?.id && !!guildId,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

/**
 * Hook to get guild member count
 */
export function useGuildMemberCount(tenant: Tenant, guildId: string, options?: ServiceOptions): UseQueryResult<number, Error> {
  return useQuery({
    queryKey: [...guildKeys.detail(tenant, guildId), 'memberCount'],
    queryFn: () => guildsService.getMemberCount(tenant, guildId, options),
    enabled: !!tenant?.id && !!guildId,
    staleTime: 3 * 60 * 1000,
    gcTime: 8 * 60 * 1000,
  });
}

// ============================================================================
// UTILITY HOOKS
// ============================================================================

/**
 * Hook to invalidate guild-related queries
 */
export function useInvalidateGuilds() {
  const queryClient = useQueryClient();

  return {
    invalidateAll: () => queryClient.invalidateQueries({ queryKey: guildKeys.all }),
    invalidateLists: () => queryClient.invalidateQueries({ queryKey: guildKeys.lists() }),
    invalidateGuild: (tenant: Tenant, guildId: string) =>
      queryClient.invalidateQueries({ queryKey: guildKeys.detail(tenant, guildId) }),
    invalidateByWorld: (tenant: Tenant, worldId: number) =>
      queryClient.invalidateQueries({ queryKey: guildKeys.byWorld(tenant, worldId) }),
    invalidateSearches: () => queryClient.invalidateQueries({ queryKey: guildKeys.searches() }),
    invalidateRankings: () => queryClient.invalidateQueries({
      queryKey: [...guildKeys.lists(), undefined, undefined, 'rankings']
    }),
  };
}

// Export types for external use
export type { Guild, GuildAttributes, GuildMember };
