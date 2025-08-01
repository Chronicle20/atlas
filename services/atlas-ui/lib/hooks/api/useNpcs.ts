/**
 * React Query hooks for NPC management
 * 
 * Provides optimized data fetching, caching, and mutation capabilities for:
 * - NPC discovery and listing (combines shop and conversation data)
 * - Shop management (create, update, delete shops)
 * - Commodity management (add, update, delete items in shops)
 * - Integration with conversations service
 * - Optimistic updates and cache invalidation
 * - Proper error handling and loading states
 */

import { useMutation, useQuery, useQueryClient, type UseMutationResult, type UseQueryResult } from '@tanstack/react-query';
import { npcsService, type NPC, type Shop, type Commodity, type CommodityAttributes, type ShopResponse } from '@/services/api/npcs.service';
import type { Tenant } from '@/types/models/tenant';
import type { ServiceOptions, QueryOptions } from '@/services/api/base.service';

// Query keys for consistent cache management
export const npcKeys = {
  all: ['npcs'] as const,
  lists: () => [...npcKeys.all, 'list'] as const,
  list: (tenant: Tenant | null, options?: QueryOptions) => [...npcKeys.lists(), tenant?.id || 'no-tenant', options] as const,
  details: () => [...npcKeys.all, 'detail'] as const,
  detail: (tenant: Tenant | null, npcId: number) => [...npcKeys.details(), tenant?.id || 'no-tenant', npcId] as const,
  
  // Specialized queries
  shops: () => [...npcKeys.all, 'shops'] as const,
  shop: (tenant: Tenant | null, npcId: number) => [...npcKeys.shops(), tenant?.id || 'no-tenant', npcId] as const,
  withShops: (tenant: Tenant | null) => [...npcKeys.all, 'withShops', tenant?.id || 'no-tenant'] as const,
  withConversations: (tenant: Tenant | null) => [...npcKeys.all, 'withConversations', tenant?.id || 'no-tenant'] as const,
  commodities: () => [...npcKeys.all, 'commodities'] as const,
  commodity: (tenant: Tenant | null, npcId: number, commodityId: string) => [...npcKeys.commodities(), tenant?.id || 'no-tenant', npcId, commodityId] as const,
};

// ============================================================================
// NPC QUERY HOOKS
// ============================================================================

/**
 * Hook to fetch all NPCs for a tenant (combines shop and conversation data)
 */
export function useNPCs(
  tenant: Tenant, 
  options?: QueryOptions
): UseQueryResult<NPC[], Error> {
  return useQuery({
    queryKey: npcKeys.list(tenant, options),
    queryFn: () => npcsService.getAllNPCs(tenant, options),
    enabled: !!tenant?.id,
    staleTime: 2 * 60 * 1000, // 2 minutes (NPC data changes less frequently)
    gcTime: 5 * 60 * 1000, // 5 minutes
  });
}

/**
 * Hook to fetch a specific NPC by ID
 */
export function useNPC(
  tenant: Tenant, 
  npcId: number, 
  options?: ServiceOptions
): UseQueryResult<NPC | null, Error> {
  return useQuery({
    queryKey: npcKeys.detail(tenant, npcId),
    queryFn: () => npcsService.getNPCById(npcId, tenant, options),
    enabled: !!tenant?.id && !!npcId,
    staleTime: 2 * 60 * 1000,
    gcTime: 5 * 60 * 1000,
  });
}

/**
 * Hook to fetch NPCs that have shops
 */
export function useNPCsWithShops(
  tenant: Tenant, 
  options?: QueryOptions
): UseQueryResult<NPC[], Error> {
  return useQuery({
    queryKey: npcKeys.withShops(tenant),
    queryFn: () => npcsService.getNPCsWithShops(tenant, options),
    enabled: !!tenant?.id,
    staleTime: 2 * 60 * 1000,
    gcTime: 5 * 60 * 1000,
  });
}

/**
 * Hook to fetch NPCs that have conversations
 */
export function useNPCsWithConversations(
  tenant: Tenant, 
  options?: QueryOptions
): UseQueryResult<NPC[], Error> {
  return useQuery({
    queryKey: npcKeys.withConversations(tenant),
    queryFn: () => npcsService.getNPCsWithConversations(tenant, options),
    enabled: !!tenant?.id,
    staleTime: 2 * 60 * 1000,
    gcTime: 5 * 60 * 1000,
  });
}

/**
 * Hook to fetch NPC shop details with commodities
 */
export function useNPCShop(
  tenant: Tenant, 
  npcId: number, 
  options?: ServiceOptions
): UseQueryResult<ShopResponse, Error> {
  return useQuery({
    queryKey: npcKeys.shop(tenant, npcId),
    queryFn: () => npcsService.getNPCShop(npcId, tenant, options),
    enabled: !!tenant?.id && !!npcId,
    staleTime: 1 * 60 * 1000, // 1 minute (shop data changes more frequently)
    gcTime: 3 * 60 * 1000,
  });
}

// ============================================================================
// SHOP MUTATION HOOKS
// ============================================================================

/**
 * Hook to create a new shop for an NPC
 */
export function useCreateShop(): UseMutationResult<
  Shop,
  Error,
  { 
    npcId: number; 
    commodities: Omit<CommodityAttributes, 'id'>[]; 
    tenant: Tenant; 
    recharger?: boolean; 
    options?: ServiceOptions 
  }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ npcId, commodities, tenant, recharger, options }) => 
      npcsService.createShop(npcId, commodities, tenant, recharger, options),
    onSuccess: (data, { tenant, npcId }) => {
      // Invalidate and refetch relevant queries
      queryClient.invalidateQueries({ queryKey: npcKeys.all });
      queryClient.invalidateQueries({ queryKey: npcKeys.shop(tenant, npcId) });
      queryClient.invalidateQueries({ queryKey: npcKeys.withShops(tenant) });
      queryClient.invalidateQueries({ queryKey: npcKeys.detail(tenant, npcId) });
    },
    onError: (error) => {
      console.error('Failed to create shop:', error);
    },
  });
}

/**
 * Hook to update an existing shop
 */
export function useUpdateShop(): UseMutationResult<
  Shop,
  Error,
  { 
    npcId: number; 
    commodities: Commodity[]; 
    tenant: Tenant; 
    recharger?: boolean; 
    options?: ServiceOptions 
  }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ npcId, commodities, tenant, recharger, options }) => 
      npcsService.updateShop(npcId, commodities, tenant, recharger, options),
    onMutate: async ({ tenant, npcId }) => {
      // Cancel any outgoing refetches for this shop
      await queryClient.cancelQueries({ queryKey: npcKeys.shop(tenant, npcId) });
      
      // Snapshot the previous shop data
      const previousShop = queryClient.getQueryData<ShopResponse>(npcKeys.shop(tenant, npcId));
      
      return { previousShop };
    },
    onError: (error, { tenant, npcId }, context) => {
      // Revert optimistic update on error
      if (context?.previousShop) {
        queryClient.setQueryData(npcKeys.shop(tenant, npcId), context.previousShop);
      }
      console.error('Failed to update shop:', error);
    },
    onSettled: (data, error, { tenant, npcId }) => {
      // Invalidate and refetch relevant queries
      queryClient.invalidateQueries({ queryKey: npcKeys.shop(tenant, npcId) });
      queryClient.invalidateQueries({ queryKey: npcKeys.all });
      queryClient.invalidateQueries({ queryKey: npcKeys.detail(tenant, npcId) });
    },
  });
}

// ============================================================================
// COMMODITY MUTATION HOOKS
// ============================================================================

/**
 * Hook to create a new commodity in an NPC's shop
 */
export function useCreateCommodity(): UseMutationResult<
  Commodity,
  Error,
  { 
    npcId: number; 
    commodityAttributes: CommodityAttributes; 
    tenant: Tenant; 
    options?: ServiceOptions 
  }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ npcId, commodityAttributes, tenant, options }) => 
      npcsService.createCommodity(npcId, commodityAttributes, tenant, options),
    onSuccess: (data, { tenant, npcId }) => {
      // Invalidate shop data to refresh commodities list
      queryClient.invalidateQueries({ queryKey: npcKeys.shop(tenant, npcId) });
      queryClient.invalidateQueries({ queryKey: npcKeys.all });
    },
    onError: (error) => {
      console.error('Failed to create commodity:', error);
    },
  });
}

/**
 * Hook to update an existing commodity
 */
export function useUpdateCommodity(): UseMutationResult<
  Commodity,
  Error,
  { 
    npcId: number; 
    commodityId: string; 
    commodityAttributes: Partial<CommodityAttributes>; 
    tenant: Tenant; 
    options?: ServiceOptions 
  }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ npcId, commodityId, commodityAttributes, tenant, options }) => 
      npcsService.updateCommodity(npcId, commodityId, commodityAttributes, tenant, options),
    onMutate: async ({ tenant, npcId, commodityId }) => {
      // Cancel any outgoing refetches for this shop
      await queryClient.cancelQueries({ queryKey: npcKeys.shop(tenant, npcId) });
      
      // Snapshot the previous shop data
      const previousShop = queryClient.getQueryData<ShopResponse>(npcKeys.shop(tenant, npcId));
      
      return { previousShop };
    },
    onError: (error, { tenant, npcId }, context) => {
      // Revert optimistic update on error
      if (context?.previousShop) {
        queryClient.setQueryData(npcKeys.shop(tenant, npcId), context.previousShop);
      }
      console.error('Failed to update commodity:', error);
    },
    onSettled: (data, error, { tenant, npcId }) => {
      // Invalidate and refetch relevant queries
      queryClient.invalidateQueries({ queryKey: npcKeys.shop(tenant, npcId) });
      queryClient.invalidateQueries({ queryKey: npcKeys.all });
    },
  });
}

/**
 * Hook to delete a commodity from an NPC's shop
 */
export function useDeleteCommodity(): UseMutationResult<
  void,
  Error,
  { 
    npcId: number; 
    commodityId: string; 
    tenant: Tenant; 
    options?: ServiceOptions 
  }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ npcId, commodityId, tenant, options }) => 
      npcsService.deleteCommodity(npcId, commodityId, tenant, options),
    onMutate: async ({ tenant, npcId, commodityId }) => {
      // Cancel any outgoing refetches for this shop
      await queryClient.cancelQueries({ queryKey: npcKeys.shop(tenant, npcId) });
      
      // Snapshot the previous shop data
      const previousShop = queryClient.getQueryData<ShopResponse>(npcKeys.shop(tenant, npcId));
      
      // Optimistically remove the commodity
      if (previousShop?.data?.relationships?.commodities?.data) {
        const optimisticShop = {
          ...previousShop,
          data: {
            ...previousShop.data,
            relationships: {
              ...previousShop.data.relationships,
              commodities: {
                ...previousShop.data.relationships.commodities,
                data: previousShop.data.relationships.commodities.data.filter(
                  commodity => commodity.id !== commodityId
                )
              }
            }
          },
          // Also filter included commodities if present
          ...(previousShop.included && {
            included: previousShop.included.filter(item => 
              !(item.type === 'commodities' && item.id === commodityId)
            )
          })
        };
        queryClient.setQueryData(npcKeys.shop(tenant, npcId), optimisticShop);
      }
      
      return { previousShop };
    },
    onError: (error, { tenant, npcId }, context) => {
      // Revert optimistic update on error
      if (context?.previousShop) {
        queryClient.setQueryData(npcKeys.shop(tenant, npcId), context.previousShop);
      }
      console.error('Failed to delete commodity:', error);
    },
    onSettled: (data, error, { tenant, npcId }) => {
      // Invalidate and refetch relevant queries
      queryClient.invalidateQueries({ queryKey: npcKeys.shop(tenant, npcId) });
      queryClient.invalidateQueries({ queryKey: npcKeys.all });
    },
  });
}

/**
 * Hook to delete all commodities for a specific NPC
 */
export function useDeleteAllCommoditiesForNPC(): UseMutationResult<
  void,
  Error,
  { 
    npcId: number; 
    tenant: Tenant; 
    options?: ServiceOptions 
  }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ npcId, tenant, options }) => 
      npcsService.deleteAllCommoditiesForNPC(npcId, tenant, options),
    onSuccess: (data, { tenant, npcId }) => {
      // Invalidate and refetch relevant queries
      queryClient.invalidateQueries({ queryKey: npcKeys.shop(tenant, npcId) });
      queryClient.invalidateQueries({ queryKey: npcKeys.all });
    },
    onError: (error) => {
      console.error('Failed to delete all commodities for NPC:', error);
    },
  });
}

/**
 * Hook to delete all shops (bulk operation)
 */
export function useDeleteAllShops(): UseMutationResult<
  void,
  Error,
  { 
    tenant: Tenant; 
    options?: ServiceOptions 
  }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ tenant, options }) => 
      npcsService.deleteAllShops(tenant, options),
    onSuccess: (data, { tenant }) => {
      // Invalidate all NPC-related queries
      queryClient.invalidateQueries({ queryKey: npcKeys.all });
      queryClient.invalidateQueries({ queryKey: npcKeys.withShops(tenant) });
    },
    onError: (error) => {
      console.error('Failed to delete all shops:', error);
    },
  });
}

/**
 * Hook to batch create commodities for an NPC
 */
export function useCreateCommoditiesBatch(): UseMutationResult<
  Commodity[],
  Error,
  { 
    npcId: number; 
    commodities: CommodityAttributes[]; 
    tenant: Tenant; 
    options?: ServiceOptions 
  }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ npcId, commodities, tenant, options }) => 
      npcsService.createCommoditiesBatch(npcId, commodities, tenant, options),
    onSuccess: (data, { tenant, npcId }) => {
      // Invalidate shop data to refresh commodities list
      queryClient.invalidateQueries({ queryKey: npcKeys.shop(tenant, npcId) });
      queryClient.invalidateQueries({ queryKey: npcKeys.all });
    },
    onError: (error) => {
      console.error('Failed to create commodities batch:', error);
    },
  });
}

// ============================================================================
// UTILITY HOOKS
// ============================================================================

/**
 * Hook to invalidate NPC-related queries for a specific tenant
 */
export function useInvalidateNPCs() {
  const queryClient = useQueryClient();
  
  return {
    /**
     * Invalidate all NPC queries
     */
    invalidateAll: () => queryClient.invalidateQueries({ queryKey: npcKeys.all }),
    
    /**
     * Invalidate all NPC lists for a tenant
     */
    invalidateLists: () => 
      queryClient.invalidateQueries({ queryKey: npcKeys.lists() }),
    
    /**
     * Invalidate specific NPC list for a tenant
     */
    invalidateList: (tenant: Tenant, options?: QueryOptions) =>
      queryClient.invalidateQueries({ queryKey: npcKeys.list(tenant, options) }),
    
    /**
     * Invalidate specific NPC details
     */
    invalidateNPC: (tenant: Tenant, npcId: number) =>
      queryClient.invalidateQueries({ queryKey: npcKeys.detail(tenant, npcId) }),
    
    /**
     * Invalidate NPC shop data
     */
    invalidateShop: (tenant: Tenant, npcId: number) =>
      queryClient.invalidateQueries({ queryKey: npcKeys.shop(tenant, npcId) }),
    
    /**
     * Invalidate NPCs with shops for a tenant
     */
    invalidateWithShops: (tenant: Tenant) =>
      queryClient.invalidateQueries({ queryKey: npcKeys.withShops(tenant) }),
    
    /**
     * Invalidate NPCs with conversations for a tenant
     */
    invalidateWithConversations: (tenant: Tenant) =>
      queryClient.invalidateQueries({ queryKey: npcKeys.withConversations(tenant) }),
    
    /**
     * Invalidate all NPC-related queries for a specific tenant
     */
    invalidateAllForTenant: (tenant: Tenant) => {
      queryClient.invalidateQueries({ queryKey: [...npcKeys.all, tenant.id] });
    },
  };
}

/**
 * Hook to prefetch NPC data for performance optimization
 */
export function usePrefetchNPCs() {
  const queryClient = useQueryClient();
  
  return {
    /**
     * Prefetch all NPCs for a tenant
     */
    prefetchNPCs: (tenant: Tenant, options?: QueryOptions) =>
      queryClient.prefetchQuery({
        queryKey: npcKeys.list(tenant, options),
        queryFn: () => npcsService.getAllNPCs(tenant, options),
        staleTime: 2 * 60 * 1000,
      }),
    
    /**
     * Prefetch specific NPC
     */
    prefetchNPC: (tenant: Tenant, npcId: number, options?: ServiceOptions) =>
      queryClient.prefetchQuery({
        queryKey: npcKeys.detail(tenant, npcId),
        queryFn: () => npcsService.getNPCById(npcId, tenant, options),
        staleTime: 2 * 60 * 1000,
      }),
    
    /**
     * Prefetch NPC shop data
     */
    prefetchShop: (tenant: Tenant, npcId: number, options?: ServiceOptions) =>
      queryClient.prefetchQuery({
        queryKey: npcKeys.shop(tenant, npcId),
        queryFn: () => npcsService.getNPCShop(npcId, tenant, options),
        staleTime: 1 * 60 * 1000,
      }),
    
    /**
     * Prefetch NPCs with shops
     */
    prefetchWithShops: (tenant: Tenant, options?: QueryOptions) =>
      queryClient.prefetchQuery({
        queryKey: npcKeys.withShops(tenant),
        queryFn: () => npcsService.getNPCsWithShops(tenant, options),
        staleTime: 2 * 60 * 1000,
      }),
    
    /**
     * Prefetch NPCs with conversations
     */
    prefetchWithConversations: (tenant: Tenant, options?: QueryOptions) =>
      queryClient.prefetchQuery({
        queryKey: npcKeys.withConversations(tenant),
        queryFn: () => npcsService.getNPCsWithConversations(tenant, options),
        staleTime: 2 * 60 * 1000,
      }),
  };
}

// Export types for external use
export type { NPC, Shop, Commodity, CommodityAttributes, ShopResponse };