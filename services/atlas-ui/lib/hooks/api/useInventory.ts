/**
 * React Query hooks for inventory management
 * 
 * Provides optimized data fetching, caching, and mutation capabilities for:
 * - Inventory retrieval operations (getInventory, getCompartments, getCompartmentAssets)
 * - Asset deletion operations
 * - Optimistic updates and cache invalidation
 * - Proper error handling and loading states
 * - Tenant-aware operations
 */

import { useMutation, useQuery, useQueryClient, type UseMutationResult, type UseQueryResult } from '@tanstack/react-query';
import { inventoryService } from '@/services/api/inventory.service';
import type { 
  Inventory, 
  InventoryResponse, 
  Compartment, 
  Asset, 
  CompartmentType 
} from '@/services/api/inventory.service';
import type { Tenant } from '@/types/models/tenant';
import type { ServiceOptions } from '@/services/api/base.service';

// Query keys for consistent cache management
export const inventoryKeys = {
  all: ['inventory'] as const,
  inventories: () => [...inventoryKeys.all, 'inventory'] as const,
  inventory: (tenant: Tenant, characterId: string, options?: ServiceOptions) => 
    [...inventoryKeys.inventories(), tenant?.id, characterId, options] as const,
  compartments: () => [...inventoryKeys.all, 'compartments'] as const,
  compartmentsList: (tenant: Tenant, characterId: string, options?: ServiceOptions) => 
    [...inventoryKeys.compartments(), tenant?.id, characterId, options] as const,
  compartmentAssets: () => [...inventoryKeys.all, 'compartmentAssets'] as const,
  compartmentAssetsList: (tenant: Tenant, characterId: string, compartmentId: string, options?: ServiceOptions) => 
    [...inventoryKeys.compartmentAssets(), tenant?.id, characterId, compartmentId, options] as const,
  summaries: () => [...inventoryKeys.all, 'summary'] as const,
  summary: (tenant: Tenant, characterId: string, options?: ServiceOptions) => 
    [...inventoryKeys.summaries(), tenant?.id, characterId, options] as const,
  assets: () => [...inventoryKeys.all, 'assets'] as const,
  hasAsset: (tenant: Tenant, characterId: string, assetId: string, options?: ServiceOptions) =>
    [...inventoryKeys.assets(), 'has', tenant?.id, characterId, assetId, options] as const,
};

// ============================================================================
// INVENTORY QUERY HOOKS
// ============================================================================

/**
 * Hook to fetch inventory data for a specific character
 */
export function useInventory(
  tenant: Tenant, 
  characterId: string, 
  options?: ServiceOptions
): UseQueryResult<InventoryResponse, Error> {
  return useQuery({
    queryKey: inventoryKeys.inventory(tenant, characterId, options),
    queryFn: () => inventoryService.getInventory(tenant, characterId, options),
    enabled: !!tenant?.id && !!characterId,
    staleTime: 30 * 1000, // 30 seconds (inventory changes frequently)
    gcTime: 2 * 60 * 1000, // 2 minutes
  });
}

/**
 * Hook to fetch compartments for a specific character
 */
export function useCompartments(
  tenant: Tenant, 
  characterId: string, 
  options?: ServiceOptions
): UseQueryResult<Compartment[], Error> {
  return useQuery({
    queryKey: inventoryKeys.compartmentsList(tenant, characterId, options),
    queryFn: () => inventoryService.getCompartments(tenant, characterId, options),
    enabled: !!tenant?.id && !!characterId,
    staleTime: 30 * 1000,
    gcTime: 2 * 60 * 1000,
  });
}

/**
 * Hook to fetch assets for a specific compartment
 */
export function useCompartmentAssets(
  tenant: Tenant, 
  characterId: string, 
  compartmentId: string,
  options?: ServiceOptions
): UseQueryResult<Asset[], Error> {
  return useQuery({
    queryKey: inventoryKeys.compartmentAssetsList(tenant, characterId, compartmentId, options),
    queryFn: () => inventoryService.getCompartmentAssets(tenant, characterId, compartmentId, options),
    enabled: !!tenant?.id && !!characterId && !!compartmentId,
    staleTime: 30 * 1000,
    gcTime: 2 * 60 * 1000,
  });
}

/**
 * Hook to fetch inventory summary with compartment counts
 */
export function useInventorySummary(
  tenant: Tenant, 
  characterId: string, 
  options?: ServiceOptions
): UseQueryResult<{
  totalCompartments: number;
  totalAssets: number;
  compartmentSummary: Array<{
    type: number;
    name: string;
    assetCount: number;
    capacity: number;
  }>;
}, Error> {
  return useQuery({
    queryKey: inventoryKeys.summary(tenant, characterId, options),
    queryFn: () => inventoryService.getInventorySummary(tenant, characterId, options),
    enabled: !!tenant?.id && !!characterId,
    staleTime: 30 * 1000,
    gcTime: 2 * 60 * 1000,
  });
}

/**
 * Hook to check if an asset exists in inventory
 */
export function useHasAsset(
  tenant: Tenant, 
  characterId: string, 
  assetId: string,
  options?: ServiceOptions
): UseQueryResult<boolean, Error> {
  return useQuery({
    queryKey: inventoryKeys.hasAsset(tenant, characterId, assetId, options),
    queryFn: () => inventoryService.hasAsset(tenant, characterId, assetId, options),
    enabled: !!tenant?.id && !!characterId && !!assetId,
    staleTime: 30 * 1000,
    gcTime: 2 * 60 * 1000,
  });
}

// ============================================================================
// INVENTORY MUTATION HOOKS
// ============================================================================

/**
 * Hook to delete an asset from inventory
 */
export function useDeleteAsset(): UseMutationResult<
  void,
  Error,
  { tenant: Tenant; characterId: string; compartmentId: string; assetId: string; options?: ServiceOptions }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ tenant, characterId, compartmentId, assetId, options }) => 
      inventoryService.deleteAsset(tenant, characterId, compartmentId, assetId, options),
    onMutate: async ({ tenant, characterId, compartmentId, assetId }) => {
      // Cancel any outgoing refetches for inventory-related queries
      await queryClient.cancelQueries({ queryKey: inventoryKeys.inventory(tenant, characterId) });
      await queryClient.cancelQueries({ queryKey: inventoryKeys.compartmentsList(tenant, characterId) });
      await queryClient.cancelQueries({ queryKey: inventoryKeys.compartmentAssetsList(tenant, characterId, compartmentId) });
      await queryClient.cancelQueries({ queryKey: inventoryKeys.summary(tenant, characterId) });
      
      // Snapshot the previous values
      const previousInventory = queryClient.getQueryData<InventoryResponse>(
        inventoryKeys.inventory(tenant, characterId)
      );
      const previousCompartmentAssets = queryClient.getQueryData<Asset[]>(
        inventoryKeys.compartmentAssetsList(tenant, characterId, compartmentId)
      );
      
      // Optimistically remove the asset from cache
      if (previousInventory) {
        const optimisticInventory: InventoryResponse = {
          ...previousInventory,
          included: previousInventory.included.filter(item => 
            !(item.type === 'assets' && item.id === assetId)
          ),
        };
        queryClient.setQueryData(inventoryKeys.inventory(tenant, characterId), optimisticInventory);
      }
      
      if (previousCompartmentAssets) {
        const optimisticAssets = previousCompartmentAssets.filter(asset => asset.id !== assetId);
        queryClient.setQueryData(
          inventoryKeys.compartmentAssetsList(tenant, characterId, compartmentId), 
          optimisticAssets
        );
      }
      
      return { previousInventory, previousCompartmentAssets };
    },
    onError: (error, variables, context) => {
      // Revert optimistic updates on error
      if (context?.previousInventory) {
        queryClient.setQueryData(
          inventoryKeys.inventory(variables.tenant, variables.characterId), 
          context.previousInventory
        );
      }
      if (context?.previousCompartmentAssets) {
        queryClient.setQueryData(
          inventoryKeys.compartmentAssetsList(variables.tenant, variables.characterId, variables.compartmentId), 
          context.previousCompartmentAssets
        );
      }
      console.error('Failed to delete asset:', error);
    },
    onSettled: (data, error, variables) => {
      // Invalidate and refetch relevant queries
      queryClient.invalidateQueries({ 
        queryKey: inventoryKeys.inventory(variables.tenant, variables.characterId) 
      });
      queryClient.invalidateQueries({ 
        queryKey: inventoryKeys.compartmentsList(variables.tenant, variables.characterId) 
      });
      queryClient.invalidateQueries({ 
        queryKey: inventoryKeys.compartmentAssetsList(variables.tenant, variables.characterId, variables.compartmentId) 
      });
      queryClient.invalidateQueries({ 
        queryKey: inventoryKeys.summary(variables.tenant, variables.characterId) 
      });
      queryClient.invalidateQueries({ 
        queryKey: inventoryKeys.hasAsset(variables.tenant, variables.characterId, variables.assetId) 
      });
    },
  });
}


// ============================================================================
// UTILITY HOOKS
// ============================================================================

/**
 * Hook to invalidate inventory-related queries
 */
export function useInvalidateInventory() {
  const queryClient = useQueryClient();
  
  return {
    invalidateAll: () => queryClient.invalidateQueries({ queryKey: inventoryKeys.all }),
    invalidateInventory: (tenant: Tenant, characterId: string) => 
      queryClient.invalidateQueries({ queryKey: inventoryKeys.inventory(tenant, characterId) }),
    invalidateCompartments: (tenant: Tenant, characterId: string) => 
      queryClient.invalidateQueries({ queryKey: inventoryKeys.compartmentsList(tenant, characterId) }),
    invalidateCompartmentAssets: (tenant: Tenant, characterId: string, compartmentId: string) => 
      queryClient.invalidateQueries({ queryKey: inventoryKeys.compartmentAssetsList(tenant, characterId, compartmentId) }),
    invalidateSummary: (tenant: Tenant, characterId: string) => 
      queryClient.invalidateQueries({ queryKey: inventoryKeys.summary(tenant, characterId) }),
    invalidateHasAsset: (tenant: Tenant, characterId: string, assetId: string) => 
      queryClient.invalidateQueries({ queryKey: inventoryKeys.hasAsset(tenant, characterId, assetId) }),
    invalidateLegacy: (tenant: Tenant, characterId: string) => {
      queryClient.invalidateQueries({ queryKey: [...inventoryKeys.inventories(), 'legacy', tenant.id, characterId] });
    },
  };
}

/**
 * Hook to prefetch inventory for a character
 */
export function usePrefetchInventory() {
  const queryClient = useQueryClient();

  return (tenant: Tenant, characterId: string, options?: ServiceOptions) => {
    queryClient.prefetchQuery({
      queryKey: inventoryKeys.inventory(tenant, characterId, options),
      queryFn: () => inventoryService.getInventory(tenant, characterId, options),
      staleTime: 30 * 1000,
    });
  };
}

/**
 * Hook to prefetch compartments for a character
 */
export function usePrefetchCompartments() {
  const queryClient = useQueryClient();

  return (tenant: Tenant, characterId: string, options?: ServiceOptions) => {
    queryClient.prefetchQuery({
      queryKey: inventoryKeys.compartmentsList(tenant, characterId, options),
      queryFn: () => inventoryService.getCompartments(tenant, characterId, options),
      staleTime: 30 * 1000,
    });
  };
}

/**
 * Hook to prefetch compartment assets
 */
export function usePrefetchCompartmentAssets() {
  const queryClient = useQueryClient();

  return (tenant: Tenant, characterId: string, compartmentId: string, options?: ServiceOptions) => {
    queryClient.prefetchQuery({
      queryKey: inventoryKeys.compartmentAssetsList(tenant, characterId, compartmentId, options),
      queryFn: () => inventoryService.getCompartmentAssets(tenant, characterId, compartmentId, options),
      staleTime: 30 * 1000,
    });
  };
}

// ============================================================================
// HELPER HOOKS
// ============================================================================

/**
 * Hook to get compartment type name utility
 */
export function useCompartmentTypeName() {
  return (type: number): string => inventoryService.getCompartmentTypeName(type);
}

/**
 * Hook to get assets for compartment utility
 */
export function useGetAssetsForCompartment() {
  return (compartment: Compartment, included: Array<Compartment | Asset>): Asset[] => 
    inventoryService.getAssetsForCompartment(compartment, included);
}

// Export types for external use
export type { 
  Inventory, 
  InventoryResponse, 
  Compartment, 
  Asset, 
  CompartmentType 
};