/**
 * React Query hooks for character management
 * 
 * Provides optimized data fetching, caching, and mutation capabilities for:
 * - Character retrieval operations (getAll, getById)
 * - Character update operations 
 * - Optimistic updates and cache invalidation
 * - Proper error handling and loading states
 * - Tenant-aware operations
 */

import { useMutation, useQuery, useQueryClient, type UseMutationResult, type UseQueryResult } from '@tanstack/react-query';
import { charactersService } from '@/services/api/characters.service';
import type { Character, UpdateCharacterData } from '@/types/models/character';
import type { Tenant } from '@/types/models/tenant';
import type { ServiceOptions } from '@/services/api/base.service';

// Query keys for consistent cache management
export const characterKeys = {
  all: ['characters'] as const,
  lists: () => [...characterKeys.all, 'list'] as const,
  list: (tenant: Tenant, options?: ServiceOptions) => [...characterKeys.lists(), tenant?.id, options] as const,
  details: () => [...characterKeys.all, 'detail'] as const,
  detail: (tenant: Tenant, characterId: string) => [...characterKeys.details(), tenant?.id, characterId] as const,
};

// ============================================================================
// CHARACTER QUERY HOOKS
// ============================================================================

/**
 * Hook to fetch all characters for a specific tenant
 */
export function useCharacters(tenant: Tenant, options?: ServiceOptions): UseQueryResult<Character[], Error> {
  return useQuery({
    queryKey: characterKeys.list(tenant, options),
    queryFn: () => charactersService.getAll(tenant, options),
    enabled: !!tenant?.id,
    staleTime: 2 * 60 * 1000, // 2 minutes (characters change more frequently)
    gcTime: 5 * 60 * 1000, // 5 minutes
  });
}

/**
 * Hook to fetch a specific character by ID for a tenant
 */
export function useCharacter(
  tenant: Tenant, 
  characterId: string, 
  options?: ServiceOptions
): UseQueryResult<Character, Error> {
  return useQuery({
    queryKey: characterKeys.detail(tenant, characterId),
    queryFn: () => charactersService.getById(tenant, characterId, options),
    enabled: !!tenant?.id && !!characterId,
    staleTime: 2 * 60 * 1000,
    gcTime: 5 * 60 * 1000,
  });
}

// ============================================================================
// CHARACTER MUTATION HOOKS
// ============================================================================

/**
 * Hook to update an existing character
 */
export function useUpdateCharacter(): UseMutationResult<
  void,
  Error,
  { tenant: Tenant; characterId: string; updates: UpdateCharacterData }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ tenant, characterId, updates }) => 
      charactersService.update(tenant, characterId, updates),
    onMutate: async ({ tenant, characterId, updates }) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: characterKeys.detail(tenant, characterId) });
      
      // Snapshot the previous value
      const previousCharacter = queryClient.getQueryData<Character>(
        characterKeys.detail(tenant, characterId)
      );
      
      // Optimistically update the cache if we have the previous character
      if (previousCharacter) {
        const optimisticCharacter: Character = {
          ...previousCharacter,
          attributes: { ...previousCharacter.attributes, ...updates },
        };
        queryClient.setQueryData(characterKeys.detail(tenant, characterId), optimisticCharacter);
      }
      
      return { previousCharacter };
    },
    onError: (error, variables, context) => {
      // Revert optimistic update on error
      if (context?.previousCharacter) {
        queryClient.setQueryData(
          characterKeys.detail(variables.tenant, variables.characterId), 
          context.previousCharacter
        );
      }
      console.error('Failed to update character:', error);
    },
    onSettled: (data, error, variables) => {
      // Invalidate and refetch relevant queries
      queryClient.invalidateQueries({ 
        queryKey: characterKeys.detail(variables.tenant, variables.characterId) 
      });
      queryClient.invalidateQueries({ 
        queryKey: characterKeys.list(variables.tenant) 
      });
    },
  });
}


// ============================================================================
// UTILITY HOOKS
// ============================================================================

/**
 * Hook to invalidate character-related queries
 */
export function useInvalidateCharacters() {
  const queryClient = useQueryClient();
  
  return {
    invalidateAll: () => queryClient.invalidateQueries({ queryKey: characterKeys.all }),
    invalidateList: (tenant: Tenant) => 
      queryClient.invalidateQueries({ queryKey: characterKeys.list(tenant) }),
    invalidateCharacter: (tenant: Tenant, characterId: string) => 
      queryClient.invalidateQueries({ queryKey: characterKeys.detail(tenant, characterId) }),
  };
}

/**
 * Hook to prefetch characters for a tenant
 */
export function usePrefetchCharacters() {
  const queryClient = useQueryClient();

  return (tenant: Tenant, options?: ServiceOptions) => {
    queryClient.prefetchQuery({
      queryKey: characterKeys.list(tenant, options),
      queryFn: () => charactersService.getAll(tenant, options),
      staleTime: 2 * 60 * 1000,
    });
  };
}

/**
 * Hook to prefetch a specific character
 */
export function usePrefetchCharacter() {
  const queryClient = useQueryClient();

  return (tenant: Tenant, characterId: string, options?: ServiceOptions) => {
    queryClient.prefetchQuery({
      queryKey: characterKeys.detail(tenant, characterId),
      queryFn: () => charactersService.getById(tenant, characterId, options),
      staleTime: 2 * 60 * 1000,
    });
  };
}

// Export types for external use
export type { Character, UpdateCharacterData };