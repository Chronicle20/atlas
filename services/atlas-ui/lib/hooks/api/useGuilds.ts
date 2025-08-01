/**
 * React Query hooks for guild management
 * 
 * Provides optimized data fetching, caching, and mutation capabilities for:
 * - Basic guild operations (CRUD)
 * - Guild member management
 * - Guild search and filtering
 * - Guild rankings and statistics
 * - Optimistic updates and cache invalidation
 * - Proper error handling and loading states
 */

import { useMutation, useQuery, useQueryClient, type UseMutationResult, type UseQueryResult } from '@tanstack/react-query';
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
    staleTime: 2 * 60 * 1000, // 2 minutes for search results
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
    staleTime: 3 * 60 * 1000, // 3 minutes
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
    staleTime: 10 * 60 * 1000, // 10 minutes for rankings
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
// GUILD MUTATION HOOKS
// ============================================================================

/**
 * Hook to create a new guild
 */
export function useCreateGuild(): UseMutationResult<Guild, Error, { tenant: Tenant; attributes: GuildAttributes }> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ tenant, attributes }) => guildsService.create(tenant, attributes),
    onSuccess: (newGuild, { tenant }) => {
      // Invalidate and refetch guild lists
      queryClient.invalidateQueries({ queryKey: guildKeys.lists() });
      
      // Add the new guild to the cache
      queryClient.setQueryData(guildKeys.detail(tenant, newGuild.id), newGuild);
      
      // Update world-specific lists if applicable
      if (newGuild.attributes.worldId) {
        queryClient.invalidateQueries({ 
          queryKey: guildKeys.byWorld(tenant, newGuild.attributes.worldId) 
        });
      }
    },
    onError: (error) => {
      console.error('Failed to create guild:', error);
    },
  });
}

/**
 * Hook to update an existing guild
 */
export function useUpdateGuild(): UseMutationResult<
  Guild,
  Error,
  { tenant: Tenant; guildId: string; updates: Partial<GuildAttributes> }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ tenant, guildId, updates }) => guildsService.update(tenant, guildId, updates),
    onMutate: async ({ tenant, guildId, updates }) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: guildKeys.detail(tenant, guildId) });
      
      // Snapshot the previous value
      const previousGuild = queryClient.getQueryData<Guild>(guildKeys.detail(tenant, guildId));
      
      // Optimistically update the cache
      if (previousGuild) {
        const optimisticGuild: Guild = {
          ...previousGuild,
          attributes: { ...previousGuild.attributes, ...updates },
        };
        queryClient.setQueryData(guildKeys.detail(tenant, guildId), optimisticGuild);
      }
      
      return { previousGuild };
    },
    onError: (error, variables, context) => {
      // Revert optimistic update on error
      if (context?.previousGuild) {
        queryClient.setQueryData(guildKeys.detail(variables.tenant, variables.guildId), context.previousGuild);
      }
      console.error('Failed to update guild:', error);
    },
    onSettled: (data, error, variables) => {
      // Invalidate and refetch relevant queries
      queryClient.invalidateQueries({ queryKey: guildKeys.detail(variables.tenant, variables.guildId) });
      queryClient.invalidateQueries({ queryKey: guildKeys.lists() });
      
      // Invalidate world-specific queries
      if (data?.attributes.worldId) {
        queryClient.invalidateQueries({ 
          queryKey: guildKeys.byWorld(variables.tenant, data.attributes.worldId) 
        });
      }
    },
  });
}

/**
 * Hook to update guild notice
 */
export function useUpdateGuildNotice(): UseMutationResult<Guild, Error, { tenant: Tenant; guildId: string; notice: string }> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ tenant, guildId, notice }) => guildsService.updateNotice(tenant, guildId, notice),
    onMutate: async ({ tenant, guildId, notice }) => {
      await queryClient.cancelQueries({ queryKey: guildKeys.detail(tenant, guildId) });
      
      const previousGuild = queryClient.getQueryData<Guild>(guildKeys.detail(tenant, guildId));
      
      if (previousGuild) {
        const optimisticGuild: Guild = {
          ...previousGuild,
          attributes: { ...previousGuild.attributes, notice },
        };
        queryClient.setQueryData(guildKeys.detail(tenant, guildId), optimisticGuild);
      }
      
      return { previousGuild };
    },
    onError: (error, variables, context) => {
      if (context?.previousGuild) {
        queryClient.setQueryData(guildKeys.detail(variables.tenant, variables.guildId), context.previousGuild);
      }
      console.error('Failed to update guild notice:', error);
    },
    onSettled: (data, error, variables) => {
      queryClient.invalidateQueries({ queryKey: guildKeys.detail(variables.tenant, variables.guildId) });
    },
  });
}

/**
 * Hook to add a member to a guild
 */
export function useAddGuildMember(): UseMutationResult<Guild, Error, { tenant: Tenant; guildId: string; member: GuildMember }> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ tenant, guildId, member }) => guildsService.addMember(tenant, guildId, member),
    onSuccess: (updatedGuild, { tenant, guildId }) => {
      // Update the guild in cache
      queryClient.setQueryData(guildKeys.detail(tenant, guildId), updatedGuild);
      
      // Invalidate related queries
      queryClient.invalidateQueries({ queryKey: guildKeys.lists() });
      queryClient.invalidateQueries({ 
        queryKey: [...guildKeys.detail(tenant, guildId), 'memberCount'] 
      });
    },
    onError: (error) => {
      console.error('Failed to add guild member:', error);
    },
  });
}

/**
 * Hook to remove a member from a guild
 */
export function useRemoveGuildMember(): UseMutationResult<Guild, Error, { tenant: Tenant; guildId: string; characterId: number }> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ tenant, guildId, characterId }) => guildsService.removeMember(tenant, guildId, characterId),
    onSuccess: (updatedGuild, { tenant, guildId }) => {
      // Update the guild in cache
      queryClient.setQueryData(guildKeys.detail(tenant, guildId), updatedGuild);
      
      // Invalidate related queries
      queryClient.invalidateQueries({ queryKey: guildKeys.lists() });
      queryClient.invalidateQueries({ 
        queryKey: [...guildKeys.detail(tenant, guildId), 'memberCount'] 
      });
    },
    onError: (error) => {
      console.error('Failed to remove guild member:', error);
    },
  });
}

/**
 * Hook to update a member's title in a guild
 */
export function useUpdateGuildMemberTitle(): UseMutationResult<
  Guild,
  Error,
  { tenant: Tenant; guildId: string; characterId: number; newTitle: number }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ tenant, guildId, characterId, newTitle }) => 
      guildsService.updateMemberTitle(tenant, guildId, characterId, newTitle),
    onSuccess: (updatedGuild, { tenant, guildId }) => {
      // Update the guild in cache
      queryClient.setQueryData(guildKeys.detail(tenant, guildId), updatedGuild);
    },
    onError: (error) => {
      console.error('Failed to update guild member title:', error);
    },
  });
}

/**
 * Hook to delete a guild
 */
export function useDeleteGuild(): UseMutationResult<void, Error, { tenant: Tenant; guildId: string }> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ tenant, guildId }) => guildsService.delete(tenant, guildId),
    onMutate: async ({ tenant, guildId }) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: guildKeys.detail(tenant, guildId) });
      
      // Snapshot the previous value
      const previousGuild = queryClient.getQueryData<Guild>(guildKeys.detail(tenant, guildId));
      
      // Optimistically remove from cache
      queryClient.removeQueries({ queryKey: guildKeys.detail(tenant, guildId) });
      
      return { previousGuild };
    },
    onError: (error, variables, context) => {
      // Restore the guild to cache on error
      if (context?.previousGuild) {
        queryClient.setQueryData(guildKeys.detail(variables.tenant, variables.guildId), context.previousGuild);
      }
      console.error('Failed to delete guild:', error);
    },
    onSettled: (data, error, variables) => {
      // Invalidate guild lists
      queryClient.invalidateQueries({ queryKey: guildKeys.lists() });
      
      // Invalidate world-specific queries if we know the world
      const guild = queryClient.getQueryData<Guild>(guildKeys.detail(variables.tenant, variables.guildId));
      if (guild?.attributes.worldId) {
        queryClient.invalidateQueries({ 
          queryKey: guildKeys.byWorld(variables.tenant, guild.attributes.worldId) 
        });
      }
    },
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