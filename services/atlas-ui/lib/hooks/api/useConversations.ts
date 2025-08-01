/**
 * React Query hooks for NPC conversation management
 * 
 * Provides optimized data fetching, caching, and mutation capabilities for:
 * - Conversation retrieval and management
 * - NPC-specific conversation operations
 * - Batch operations for multiple conversations
 * - State consistency validation
 * - Optimistic updates and cache invalidation
 * - Proper error handling and loading states
 */

import { useMutation, useQuery, useQueryClient, type UseMutationResult, type UseQueryResult } from '@tanstack/react-query';
import { conversationsService, type ConversationCreateRequest, type ConversationUpdateRequest } from '@/services/api/conversations.service';
import type { 
  Conversation, 
  ConversationAttributes,
  DialogueChoice,
  DialogueState,
  GenericActionOperation,
  Condition,
  GenericActionOutcome,
  GenericActionState,
  CraftActionState,
  ListSelectionState,
  ConversationState
} from '@/types/models/conversation';
import type { Tenant } from '@/types/models/tenant';
import type { ServiceOptions, QueryOptions } from '@/services/api/base.service';

// Query keys for consistent cache management
export const conversationKeys = {
  all: ['conversations'] as const,
  lists: () => [...conversationKeys.all, 'list'] as const,
  list: (tenant: Tenant | null, options?: QueryOptions) => [...conversationKeys.lists(), tenant?.id || 'no-tenant', options] as const,
  details: () => [...conversationKeys.all, 'detail'] as const,
  detail: (tenant: Tenant | null, id: string) => [...conversationKeys.details(), tenant?.id || 'no-tenant', id] as const,
  
  // Specialized queries
  byNpc: () => [...conversationKeys.all, 'byNpc'] as const,
  npcConversation: (tenant: Tenant | null, npcId: number) => [...conversationKeys.byNpc(), tenant?.id || 'no-tenant', npcId] as const,
  searches: () => [...conversationKeys.all, 'search'] as const,
  search: (tenant: Tenant | null, searchText: string) => [...conversationKeys.searches(), tenant?.id || 'no-tenant', searchText] as const,
  validation: () => [...conversationKeys.all, 'validation'] as const,
  stateConsistency: (tenant: Tenant | null, id: string) => [...conversationKeys.validation(), tenant?.id || 'no-tenant', id] as const,
  exports: () => [...conversationKeys.all, 'export'] as const,
  export: (tenant: Tenant | null, format: 'json' | 'csv') => [...conversationKeys.exports(), tenant?.id || 'no-tenant', format] as const,
};

// ============================================================================
// CONVERSATION QUERY HOOKS
// ============================================================================

/**
 * Hook to fetch all conversations for a tenant
 */
export function useConversations(
  tenant: Tenant, 
  options?: QueryOptions
): UseQueryResult<Conversation[], Error> {
  return useQuery({
    queryKey: conversationKeys.list(tenant, options),
    queryFn: () => conversationsService.getAll(options),
    enabled: !!tenant?.id,
    staleTime: 3 * 60 * 1000, // 3 minutes (conversations change less frequently)
    gcTime: 10 * 60 * 1000, // 10 minutes
  });
}

/**
 * Hook to fetch a specific conversation by ID
 */
export function useConversation(
  tenant: Tenant, 
  id: string, 
  options?: ServiceOptions
): UseQueryResult<Conversation, Error> {
  return useQuery({
    queryKey: conversationKeys.detail(tenant, id),
    queryFn: () => conversationsService.getById(id, options),
    enabled: !!tenant?.id && !!id,
    staleTime: 3 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

/**
 * Hook to check if a conversation exists
 */
export function useConversationExists(
  tenant: Tenant, 
  id: string, 
  options?: ServiceOptions
): UseQueryResult<boolean, Error> {
  return useQuery({
    queryKey: [...conversationKeys.detail(tenant, id), 'exists'],
    queryFn: () => conversationsService.exists(id, options),
    enabled: !!tenant?.id && !!id,
    staleTime: 2 * 60 * 1000, // 2 minutes for existence checks
    gcTime: 5 * 60 * 1000,
  });
}

/**
 * Hook to fetch conversation for a specific NPC
 */
export function useConversationByNpc(
  tenant: Tenant, 
  npcId: number, 
  options?: QueryOptions
): UseQueryResult<Conversation | null, Error> {
  return useQuery({
    queryKey: conversationKeys.npcConversation(tenant, npcId),
    queryFn: () => conversationsService.getByNpcId(npcId, options),
    enabled: !!tenant?.id && !!npcId,
    staleTime: 3 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

/**
 * Hook to search conversations by text content
 */
export function useConversationSearch(
  tenant: Tenant, 
  searchText: string, 
  options?: QueryOptions
): UseQueryResult<Conversation[], Error> {
  return useQuery({
    queryKey: conversationKeys.search(tenant, searchText),
    queryFn: () => conversationsService.searchByText(searchText, options),
    enabled: !!tenant?.id && !!searchText && searchText.length > 2,
    staleTime: 1 * 60 * 1000, // Search results can be more stale
    gcTime: 3 * 60 * 1000,
  });
}

/**
 * Hook to get conversations by NPC ID with filtering
 */
export function useConversationsByNpc(
  tenant: Tenant, 
  npcId: number, 
  options?: QueryOptions
): UseQueryResult<Conversation[], Error> {
  return useQuery({
    queryKey: [...conversationKeys.byNpc(), tenant?.id || 'no-tenant', npcId, 'all'],
    queryFn: () => conversationsService.getConversationsByNpc(npcId, options),
    enabled: !!tenant?.id && !!npcId,
    staleTime: 3 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

/**
 * Hook to validate conversation state consistency
 */
export function useConversationStateConsistency(
  tenant: Tenant, 
  conversationId: string
): UseQueryResult<{ isValid: boolean; errors: string[] }, Error> {
  return useQuery({
    queryKey: conversationKeys.stateConsistency(tenant, conversationId),
    queryFn: () => conversationsService.validateStateConsistency(conversationId),
    enabled: !!tenant?.id && !!conversationId,
    staleTime: 5 * 60 * 1000, // 5 minutes (validation results don't change often)
    gcTime: 10 * 60 * 1000,
  });
}

/**
 * Hook to export conversation data
 */
export function useConversationExport(
  tenant: Tenant, 
  format: 'json' | 'csv' = 'json',
  options?: QueryOptions
): UseQueryResult<Blob, Error> {
  return useQuery({
    queryKey: conversationKeys.export(tenant, format),
    queryFn: () => conversationsService.export(format, options),
    enabled: !!tenant?.id,
    staleTime: 10 * 60 * 1000, // 10 minutes (exports don't change often)
    gcTime: 15 * 60 * 1000,
  });
}

// ============================================================================
// CONVERSATION MUTATION HOOKS
// ============================================================================

/**
 * Hook to create a new conversation
 */
export function useCreateConversation(): UseMutationResult<
  Conversation,
  Error,
  { 
    conversationAttributes: ConversationAttributes; 
    tenant: Tenant; 
    options?: ServiceOptions 
  }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ conversationAttributes, tenant, options }) => 
      conversationsService.create(conversationAttributes, options),
    onSuccess: (data, { tenant }) => {
      // Invalidate and refetch relevant queries
      queryClient.invalidateQueries({ queryKey: conversationKeys.all });
      queryClient.invalidateQueries({ queryKey: conversationKeys.byNpc() });
      
      // If the conversation has an NPC ID, invalidate NPC-specific queries
      if (data.attributes.npcId) {
        queryClient.invalidateQueries({ 
          queryKey: conversationKeys.npcConversation(tenant, data.attributes.npcId) 
        });
      }
    },
    onError: (error) => {
      console.error('Failed to create conversation:', error);
    },
  });
}

/**
 * Hook to update an existing conversation
 */
export function useUpdateConversation(): UseMutationResult<
  Conversation,
  Error,
  { 
    id: string; 
    conversationAttributes: Partial<ConversationAttributes>; 
    tenant: Tenant; 
    options?: ServiceOptions 
  }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, conversationAttributes, tenant, options }) => 
      conversationsService.update(id, conversationAttributes, options),
    onMutate: async ({ tenant, id }) => {
      // Cancel any outgoing refetches for this conversation
      await queryClient.cancelQueries({ queryKey: conversationKeys.detail(tenant, id) });
      
      // Snapshot the previous conversation data
      const previousConversation = queryClient.getQueryData<Conversation>(conversationKeys.detail(tenant, id));
      
      return { previousConversation };
    },
    onError: (error, { tenant, id }, context) => {
      // Revert optimistic update on error
      if (context?.previousConversation) {
        queryClient.setQueryData(conversationKeys.detail(tenant, id), context.previousConversation);
      }
      console.error('Failed to update conversation:', error);
    },
    onSettled: (data, error, { tenant, id }) => {
      // Invalidate and refetch relevant queries
      queryClient.invalidateQueries({ queryKey: conversationKeys.detail(tenant, id) });
      queryClient.invalidateQueries({ queryKey: conversationKeys.all });
      queryClient.invalidateQueries({ queryKey: conversationKeys.byNpc() });
      
      // Invalidate state consistency validation
      queryClient.invalidateQueries({ queryKey: conversationKeys.stateConsistency(tenant, id) });
      
      // If the conversation has an NPC ID, invalidate NPC-specific queries
      if (data?.attributes.npcId) {
        queryClient.invalidateQueries({ 
          queryKey: conversationKeys.npcConversation(tenant, data.attributes.npcId) 
        });
      }
    },
  });
}

/**
 * Hook to partially update a conversation (PATCH)
 */
export function usePatchConversation(): UseMutationResult<
  Conversation,
  Error,
  { 
    id: string; 
    updates: Partial<ConversationAttributes>; 
    tenant: Tenant; 
    options?: ServiceOptions 
  }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, updates, tenant, options }) => 
      conversationsService.patch(id, updates, options),
    onMutate: async ({ tenant, id }) => {
      // Cancel any outgoing refetches for this conversation
      await queryClient.cancelQueries({ queryKey: conversationKeys.detail(tenant, id) });
      
      // Snapshot the previous conversation data
      const previousConversation = queryClient.getQueryData<Conversation>(conversationKeys.detail(tenant, id));
      
      return { previousConversation };
    },
    onError: (error, { tenant, id }, context) => {
      // Revert optimistic update on error
      if (context?.previousConversation) {
        queryClient.setQueryData(conversationKeys.detail(tenant, id), context.previousConversation);
      }
      console.error('Failed to patch conversation:', error);
    },
    onSettled: (data, error, { tenant, id }) => {
      // Invalidate and refetch relevant queries
      queryClient.invalidateQueries({ queryKey: conversationKeys.detail(tenant, id) });
      queryClient.invalidateQueries({ queryKey: conversationKeys.all });
      queryClient.invalidateQueries({ queryKey: conversationKeys.byNpc() });
      
      // Invalidate state consistency validation
      queryClient.invalidateQueries({ queryKey: conversationKeys.stateConsistency(tenant, id) });
    },
  });
}

/**
 * Hook to delete a conversation
 */
export function useDeleteConversation(): UseMutationResult<
  void,
  Error,
  { 
    id: string; 
    tenant: Tenant; 
    options?: ServiceOptions 
  }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, tenant, options }) => 
      conversationsService.delete(id, options),
    onMutate: async ({ tenant, id }) => {
      // Cancel any outgoing refetches for this conversation
      await queryClient.cancelQueries({ queryKey: conversationKeys.detail(tenant, id) });
      
      // Snapshot the previous conversation data for potential rollback
      const previousConversation = queryClient.getQueryData<Conversation>(conversationKeys.detail(tenant, id));
      
      // Optimistically remove the conversation from lists
      queryClient.removeQueries({ queryKey: conversationKeys.detail(tenant, id) });
      
      return { previousConversation };
    },
    onError: (error, { tenant, id }, context) => {
      // Restore the conversation on error
      if (context?.previousConversation) {
        queryClient.setQueryData(conversationKeys.detail(tenant, id), context.previousConversation);
      }
      console.error('Failed to delete conversation:', error);
    },
    onSettled: (data, error, { tenant, id }) => {
      // Invalidate and refetch relevant queries
      queryClient.invalidateQueries({ queryKey: conversationKeys.all });
      queryClient.invalidateQueries({ queryKey: conversationKeys.byNpc() });
      
      // Remove any cached state consistency validation
      queryClient.removeQueries({ queryKey: conversationKeys.stateConsistency(tenant, id) });
    },
  });
}

// ============================================================================
// BATCH OPERATION HOOKS
// ============================================================================

/**
 * Hook to batch create multiple conversations
 */
export function useCreateConversationsBatch(): UseMutationResult<
  Conversation[],
  Error,
  { 
    conversations: ConversationAttributes[]; 
    tenant: Tenant; 
    options?: ServiceOptions;
    batchOptions?: { 
      batchSize?: number; 
      delayBetweenBatches?: number; 
      continueOnError?: boolean 
    }
  }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ conversations, tenant, options, batchOptions }) => 
      conversationsService.createBatch(conversations, options, batchOptions),
    onSuccess: (data, { tenant }) => {
      // Invalidate all conversation-related queries
      queryClient.invalidateQueries({ queryKey: conversationKeys.all });
      queryClient.invalidateQueries({ queryKey: conversationKeys.byNpc() });
      
      // Invalidate NPC-specific queries for created conversations
      data.forEach(conversation => {
        if (conversation.attributes.npcId) {
          queryClient.invalidateQueries({ 
            queryKey: conversationKeys.npcConversation(tenant, conversation.attributes.npcId) 
          });
        }
      });
    },
    onError: (error) => {
      console.error('Failed to create conversations batch:', error);
    },
  });
}

/**
 * Hook to batch update multiple conversations
 */
export function useUpdateConversationsBatch(): UseMutationResult<
  Conversation[],
  Error,
  { 
    updates: Array<{ id: string; data: Partial<ConversationAttributes> }>; 
    tenant: Tenant; 
    options?: ServiceOptions;
    batchOptions?: { 
      batchSize?: number; 
      delayBetweenBatches?: number; 
      continueOnError?: boolean 
    }
  }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ updates, tenant, options, batchOptions }) => 
      conversationsService.updateBatch(updates, options, batchOptions),
    onSuccess: (data, { tenant }) => {
      // Invalidate all conversation-related queries
      queryClient.invalidateQueries({ queryKey: conversationKeys.all });
      queryClient.invalidateQueries({ queryKey: conversationKeys.byNpc() });
      
      // Invalidate specific conversation details and state consistency
      data.forEach(conversation => {
        queryClient.invalidateQueries({ queryKey: conversationKeys.detail(tenant, conversation.id) });
        queryClient.invalidateQueries({ queryKey: conversationKeys.stateConsistency(tenant, conversation.id) });
        
        if (conversation.attributes.npcId) {
          queryClient.invalidateQueries({ 
            queryKey: conversationKeys.npcConversation(tenant, conversation.attributes.npcId) 
          });
        }
      });
    },
    onError: (error) => {
      console.error('Failed to update conversations batch:', error);
    },
  });
}

/**
 * Hook to batch delete multiple conversations
 */
export function useDeleteConversationsBatch(): UseMutationResult<
  void,
  Error,
  { 
    ids: string[]; 
    tenant: Tenant; 
    options?: ServiceOptions;
    batchOptions?: { 
      batchSize?: number; 
      delayBetweenBatches?: number; 
      continueOnError?: boolean 
    }
  }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ ids, tenant, options, batchOptions }) => 
      conversationsService.deleteBatch(ids, options, batchOptions),
    onMutate: async ({ tenant, ids }) => {
      // Cancel outgoing refetches for affected conversations
      const cancelPromises = ids.map(id =>
        queryClient.cancelQueries({ queryKey: conversationKeys.detail(tenant, id) })
      );
      await Promise.all(cancelPromises);
      
      // Snapshot previous conversation data
      const previousConversations = new Map<string, Conversation>();
      
      // Optimistically remove conversations
      ids.forEach(id => {
        const previousConversation = queryClient.getQueryData<Conversation>(conversationKeys.detail(tenant, id));
        if (previousConversation) {
          previousConversations.set(id, previousConversation);
        }
        queryClient.removeQueries({ queryKey: conversationKeys.detail(tenant, id) });
      });
      
      return { previousConversations };
    },
    onError: (error, { tenant, ids }, context) => {
      // Restore conversations on error
      if (context?.previousConversations) {
        ids.forEach(id => {
          const previousConversation = context.previousConversations.get(id);
          if (previousConversation) {
            queryClient.setQueryData(conversationKeys.detail(tenant, id), previousConversation);
          }
        });
      }
      console.error('Failed to delete conversations batch:', error);
    },
    onSettled: (data, error, { tenant, ids }) => {
      // Invalidate all conversation-related queries
      queryClient.invalidateQueries({ queryKey: conversationKeys.all });
      queryClient.invalidateQueries({ queryKey: conversationKeys.byNpc() });
      
      // Remove cached state consistency validations
      ids.forEach(id => {
        queryClient.removeQueries({ queryKey: conversationKeys.stateConsistency(tenant, id) });
      });
    },
  });
}

// ============================================================================
// UTILITY HOOKS
// ============================================================================

/**
 * Hook to invalidate conversation-related queries for a specific tenant
 */
export function useInvalidateConversations() {
  const queryClient = useQueryClient();
  
  return {
    /**
     * Invalidate all conversation queries
     */
    invalidateAll: () => queryClient.invalidateQueries({ queryKey: conversationKeys.all }),
    
    /**
     * Invalidate all conversation lists for a tenant
     */
    invalidateLists: () => 
      queryClient.invalidateQueries({ queryKey: conversationKeys.lists() }),
    
    /**
     * Invalidate specific conversation list for a tenant
     */
    invalidateList: (tenant: Tenant, options?: QueryOptions) =>
      queryClient.invalidateQueries({ queryKey: conversationKeys.list(tenant, options) }),
    
    /**
     * Invalidate specific conversation details
     */
    invalidateConversation: (tenant: Tenant, id: string) =>
      queryClient.invalidateQueries({ queryKey: conversationKeys.detail(tenant, id) }),
    
    /**
     * Invalidate NPC conversation data
     */
    invalidateNpcConversation: (tenant: Tenant, npcId: number) =>
      queryClient.invalidateQueries({ queryKey: conversationKeys.npcConversation(tenant, npcId) }),
    
    /**
     * Invalidate conversation search results
     */
    invalidateSearches: () =>
      queryClient.invalidateQueries({ queryKey: conversationKeys.searches() }),
    
    /**
     * Invalidate state consistency validation
     */
    invalidateStateConsistency: (tenant: Tenant, id: string) =>
      queryClient.invalidateQueries({ queryKey: conversationKeys.stateConsistency(tenant, id) }),
    
    /**
     * Invalidate all conversation-related queries for a specific tenant
     */
    invalidateAllForTenant: (tenant: Tenant) => {
      queryClient.invalidateQueries({ queryKey: [...conversationKeys.all, tenant.id] });
    },
    
    /**
     * Clear conversations cache
     */
    clearCache: () => {
      conversationsService.clearCache();
      queryClient.invalidateQueries({ queryKey: conversationKeys.all });
    },
  };
}

/**
 * Hook to prefetch conversation data for performance optimization
 */
export function usePrefetchConversations() {
  const queryClient = useQueryClient();
  
  return {
    /**
     * Prefetch all conversations for a tenant
     */
    prefetchConversations: (tenant: Tenant, options?: QueryOptions) =>
      queryClient.prefetchQuery({
        queryKey: conversationKeys.list(tenant, options),
        queryFn: () => conversationsService.getAll(options),
        staleTime: 3 * 60 * 1000,
      }),
    
    /**
     * Prefetch specific conversation
     */
    prefetchConversation: (tenant: Tenant, id: string, options?: ServiceOptions) =>
      queryClient.prefetchQuery({
        queryKey: conversationKeys.detail(tenant, id),
        queryFn: () => conversationsService.getById(id, options),
        staleTime: 3 * 60 * 1000,
      }),
    
    /**
     * Prefetch conversation for specific NPC
     */
    prefetchNpcConversation: (tenant: Tenant, npcId: number, options?: QueryOptions) =>
      queryClient.prefetchQuery({
        queryKey: conversationKeys.npcConversation(tenant, npcId),
        queryFn: () => conversationsService.getByNpcId(npcId, options),
        staleTime: 3 * 60 * 1000,
      }),
    
    /**
     * Prefetch state consistency validation
     */
    prefetchStateConsistency: (tenant: Tenant, conversationId: string) =>
      queryClient.prefetchQuery({
        queryKey: conversationKeys.stateConsistency(tenant, conversationId),
        queryFn: () => conversationsService.validateStateConsistency(conversationId),
        staleTime: 5 * 60 * 1000,
      }),
  };
}

/**
 * Hook to get conversation cache statistics
 */
export function useConversationCacheStats() {
  return {
    getCacheStats: () => conversationsService.getCacheStats(),
  };
}

// Export types for external use
export type { 
  Conversation, 
  ConversationAttributes,
  DialogueChoice,
  DialogueState,
  GenericActionOperation,
  Condition,
  GenericActionOutcome,
  GenericActionState,
  CraftActionState,
  ListSelectionState,
  ConversationState,
  ConversationCreateRequest,
  ConversationUpdateRequest
};