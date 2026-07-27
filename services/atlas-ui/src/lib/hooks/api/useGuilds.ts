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

import {
  useQuery,
  useQueryClient,
  keepPreviousData,
  type UseQueryResult,
} from "@tanstack/react-query";
import {
  guildsService,
  type Guild,
  type GuildAttributes,
  type GuildMember,
} from "@/services/api/guilds.service";
import type { PagedResult } from "@/services/api/pagination";
import type { ServiceOptions, QueryOptions } from "@/lib/api/query-params";
import type { Tenant } from "@/types/models/tenant";

// Query keys for consistent cache management
export const guildKeys = {
  all: ["guilds"] as const,
  lists: () => [...guildKeys.all, "list"] as const,
  list: (tenant: Tenant | null, options?: QueryOptions) =>
    [...guildKeys.lists(), tenant?.id, options] as const,
  pagedList: (tenant: Tenant | null, page: number, size: number) =>
    [...guildKeys.lists(), tenant?.id ?? "no-tenant", page, size] as const,
  details: () => [...guildKeys.all, "detail"] as const,
  detail: (tenant: Tenant | null, id: string) =>
    [...guildKeys.details(), tenant?.id, id] as const,

  // Specialized query keys
  searches: () => [...guildKeys.all, "search"] as const,
  search: (
    tenant: Tenant | null,
    searchTerm: string,
    page: number,
    size: number,
  ) => [...guildKeys.searches(), tenant?.id, searchTerm, page, size] as const,
  byWorld: (tenant: Tenant | null, worldId: number) =>
    [...guildKeys.lists(), tenant?.id, "world", worldId] as const,
  withSpace: (tenant: Tenant | null, worldId?: number) =>
    [...guildKeys.lists(), tenant?.id, "space", worldId] as const,
  rankings: (tenant: Tenant | null, worldId?: number, limit?: number) =>
    [...guildKeys.lists(), tenant?.id, "rankings", worldId, limit] as const,
};

// ============================================================================
// GUILD QUERY HOOKS
// ============================================================================

/**
 * Hook to fetch a single page of guilds for a tenant (task-117). Backs the
 * Guilds list view, which pages server-side; keeps the previous page's data
 * on screen while the next page loads. Pass `enabled: false` to suspend
 * fetching (e.g. while a search term is active on the same page).
 */
export function useGuildsPage(
  tenant: Tenant | null,
  page: { number: number; size: number },
  options?: ServiceOptions,
  enabled: boolean = true,
): UseQueryResult<PagedResult<Guild>, Error> {
  return useQuery({
    queryKey: guildKeys.pagedList(tenant, page.number, page.size),
    queryFn: () => guildsService.getPage(page, { ...options, useCache: false }),
    enabled: !!tenant?.id && enabled,
    placeholderData: keepPreviousData,
    gcTime: 10 * 60 * 1000, // 10 minutes
  });
}

/**
 * Hook to fetch a specific guild by ID
 */
export function useGuild(
  tenant: Tenant,
  guildId: string,
  options?: ServiceOptions,
): UseQueryResult<Guild, Error> {
  return useQuery({
    queryKey: guildKeys.detail(tenant, guildId),
    queryFn: () =>
      guildsService.getById(guildId, { ...options, useCache: false }),
    enabled: !!tenant?.id && !!guildId,
    gcTime: 10 * 60 * 1000,
  });
}

/**
 * Hook to fetch guilds by world ID
 */
export function useGuildsByWorld(
  tenant: Tenant,
  worldId: number,
  options?: ServiceOptions,
): UseQueryResult<Guild[], Error> {
  return useQuery({
    queryKey: guildKeys.byWorld(tenant, worldId),
    queryFn: () =>
      guildsService.getByWorld(worldId, { ...options, useCache: false }),
    enabled: !!tenant?.id && worldId !== undefined,
    gcTime: 10 * 60 * 1000,
  });
}

/**
 * Hook to search guilds by name via the server-side `filter[name]`
 * substring match (task-117), a single page at a time. Pass `enabled: false`
 * to suspend fetching (e.g. while the search term is empty).
 */
export function useGuildSearch(
  tenant: Tenant | null,
  searchTerm: string,
  page: { number: number; size: number },
  options?: ServiceOptions,
  enabled: boolean = true,
): UseQueryResult<PagedResult<Guild>, Error> {
  return useQuery({
    queryKey: guildKeys.search(tenant, searchTerm, page.number, page.size),
    queryFn: () =>
      guildsService.search(searchTerm, page, { ...options, useCache: false }),
    enabled: !!tenant?.id && !!searchTerm && enabled,
    placeholderData: keepPreviousData,
    gcTime: 5 * 60 * 1000,
  });
}

/**
 * Hook to fetch guilds with available space
 */
export function useGuildsWithSpace(
  tenant: Tenant,
  worldId?: number,
  options?: ServiceOptions,
): UseQueryResult<Guild[], Error> {
  return useQuery({
    queryKey: guildKeys.withSpace(tenant, worldId),
    queryFn: () =>
      guildsService.getWithSpace(worldId, { ...options, useCache: false }),
    enabled: !!tenant?.id,
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
  options?: ServiceOptions,
): UseQueryResult<Guild[], Error> {
  return useQuery({
    queryKey: guildKeys.rankings(tenant, worldId, limit),
    queryFn: () =>
      guildsService.getRankings(worldId, limit, {
        ...options,
        useCache: false,
      }),
    enabled: !!tenant?.id,
    gcTime: 15 * 60 * 1000,
  });
}

/**
 * Hook to check if a guild exists
 */
export function useGuildExists(
  tenant: Tenant,
  guildId: string,
  options?: ServiceOptions,
): UseQueryResult<boolean, Error> {
  return useQuery({
    queryKey: [...guildKeys.detail(tenant, guildId), "exists"],
    queryFn: () =>
      guildsService.exists(guildId, { ...options, useCache: false }),
    enabled: !!tenant?.id && !!guildId,
    gcTime: 10 * 60 * 1000,
  });
}

/**
 * Hook to get guild member count
 */
export function useGuildMemberCount(
  tenant: Tenant,
  guildId: string,
  options?: ServiceOptions,
): UseQueryResult<number, Error> {
  return useQuery({
    queryKey: [...guildKeys.detail(tenant, guildId), "memberCount"],
    queryFn: () =>
      guildsService.getMemberCount(guildId, { ...options, useCache: false }),
    enabled: !!tenant?.id && !!guildId,
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
    invalidateAll: () =>
      queryClient.invalidateQueries({ queryKey: guildKeys.all }),
    invalidateLists: () =>
      queryClient.invalidateQueries({ queryKey: guildKeys.lists() }),
    invalidateGuild: (tenant: Tenant, guildId: string) =>
      queryClient.invalidateQueries({
        queryKey: guildKeys.detail(tenant, guildId),
      }),
    invalidateByWorld: (tenant: Tenant, worldId: number) =>
      queryClient.invalidateQueries({
        queryKey: guildKeys.byWorld(tenant, worldId),
      }),
    invalidateSearches: () =>
      queryClient.invalidateQueries({ queryKey: guildKeys.searches() }),
    invalidateRankings: () =>
      queryClient.invalidateQueries({
        queryKey: [...guildKeys.lists(), undefined, undefined, "rankings"],
      }),
  };
}

// Export types for external use
export type { Guild, GuildAttributes, GuildMember };
