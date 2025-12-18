/**
 * React Query hooks for map management
 * 
 * Provides optimized data fetching, caching, and mutation capabilities for:
 * - Basic map operations (CRUD)
 * - Map search and filtering operations
 * - Optimistic updates and cache invalidation
 * - Proper error handling and loading states
 */

import { useMutation, useQuery, useQueryClient, type UseMutationResult, type UseQueryResult } from '@tanstack/react-query';
import { mapsService, type MapData, type MapAttributes } from '@/services/api/maps.service';
import type { ServiceOptions, QueryOptions } from '@/services/api/base.service';

// Query keys for consistent cache management
export const mapKeys = {
  all: ['maps'] as const,
  lists: () => [...mapKeys.all, 'list'] as const,
  list: (options?: QueryOptions) => [...mapKeys.lists(), options] as const,
  details: () => [...mapKeys.all, 'detail'] as const,
  detail: (id: string) => [...mapKeys.details(), id] as const,
  search: () => [...mapKeys.all, 'search'] as const,
  searchByName: (name: string) => [...mapKeys.search(), 'name', name] as const,
  searchByStreet: (streetName: string) => [...mapKeys.search(), 'street', streetName] as const,
};

// ============================================================================
// MAP QUERY HOOKS
// ============================================================================

/**
 * Hook to fetch all maps
 */
export function useMaps(options?: QueryOptions): UseQueryResult<MapData[], Error> {
  return useQuery({
    queryKey: mapKeys.list(options),
    queryFn: () => mapsService.getAllMaps(options),
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
  });
}

/**
 * Hook to fetch a specific map by ID
 */
export function useMap(id: string, options?: ServiceOptions): UseQueryResult<MapData, Error> {
  return useQuery({
    queryKey: mapKeys.detail(id),
    queryFn: () => mapsService.getMapById(id, options),
    enabled: !!id,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

/**
 * Hook to search maps by name
 */
export function useMapsByName(name: string, options?: ServiceOptions): UseQueryResult<MapData[], Error> {
  return useQuery({
    queryKey: mapKeys.searchByName(name),
    queryFn: () => mapsService.searchMapsByName(name, options),
    enabled: !!name && name.trim().length > 0,
    staleTime: 2 * 60 * 1000, // 2 minutes for search results
    gcTime: 5 * 60 * 1000,
  });
}

/**
 * Hook to fetch maps by street name
 */
export function useMapsByStreetName(streetName: string, options?: ServiceOptions): UseQueryResult<MapData[], Error> {
  return useQuery({
    queryKey: mapKeys.searchByStreet(streetName),
    queryFn: () => mapsService.getMapsByStreetName(streetName, options),
    enabled: !!streetName && streetName.trim().length > 0,
    staleTime: 2 * 60 * 1000, // 2 minutes for search results
    gcTime: 5 * 60 * 1000,
  });
}

// ============================================================================
// MAP MUTATION HOOKS
// ============================================================================

/**
 * Hook to create a new map
 */
export function useCreateMap(): UseMutationResult<MapData, Error, MapAttributes> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (attributes: MapAttributes) => mapsService.createMap(attributes),
    onSuccess: (newMap) => {
      // Invalidate and refetch map lists
      queryClient.invalidateQueries({ queryKey: mapKeys.lists() });
      
      // Add the new map to the cache
      queryClient.setQueryData(mapKeys.detail(newMap.id), newMap);
      
      // Invalidate search queries that might include this map
      queryClient.invalidateQueries({ queryKey: mapKeys.search() });
    },
    onError: (error) => {
      console.error('Failed to create map:', error);
    },
  });
}

/**
 * Hook to update an existing map
 */
export function useUpdateMap(): UseMutationResult<
  MapData,
  Error,
  { map: MapData; updates: Partial<MapAttributes> }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ map, updates }) => mapsService.updateMap(map, updates),
    onMutate: async ({ map, updates }) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: mapKeys.detail(map.id) });
      
      // Snapshot the previous value
      const previousMap = queryClient.getQueryData<MapData>(mapKeys.detail(map.id));
      
      // Optimistically update the cache
      const optimisticMap: MapData = {
        ...map,
        attributes: { ...map.attributes, ...updates },
      };
      queryClient.setQueryData(mapKeys.detail(map.id), optimisticMap);
      
      return { previousMap };
    },
    onError: (error, variables, context) => {
      // Revert optimistic update on error
      if (context?.previousMap) {
        queryClient.setQueryData(mapKeys.detail(variables.map.id), context.previousMap);
      }
      console.error('Failed to update map:', error);
    },
    onSettled: (data, error, variables) => {
      // Invalidate and refetch relevant queries
      queryClient.invalidateQueries({ queryKey: mapKeys.detail(variables.map.id) });
      queryClient.invalidateQueries({ queryKey: mapKeys.lists() });
      queryClient.invalidateQueries({ queryKey: mapKeys.search() });
    },
  });
}

/**
 * Hook to delete a map
 */
export function useDeleteMap(): UseMutationResult<void, Error, { id: string }> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id }) => mapsService.deleteMap(id),
    onMutate: async ({ id }) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: mapKeys.detail(id) });
      
      // Snapshot the previous value
      const previousMap = queryClient.getQueryData<MapData>(mapKeys.detail(id));
      
      // Optimistically remove from cache
      queryClient.removeQueries({ queryKey: mapKeys.detail(id) });
      
      return { previousMap };
    },
    onError: (error, variables, context) => {
      // Restore the map to cache on error
      if (context?.previousMap) {
        queryClient.setQueryData(mapKeys.detail(variables.id), context.previousMap);
      }
      console.error('Failed to delete map:', error);
    },
    onSettled: () => {
      // Invalidate map lists and search results
      queryClient.invalidateQueries({ queryKey: mapKeys.lists() });
      queryClient.invalidateQueries({ queryKey: mapKeys.search() });
    },
  });
}

// ============================================================================
// UTILITY HOOKS
// ============================================================================

/**
 * Hook to invalidate map-related queries
 */
export function useInvalidateMaps() {
  const queryClient = useQueryClient();
  
  return {
    invalidateAll: () => queryClient.invalidateQueries({ queryKey: mapKeys.all }),
    invalidateLists: () => queryClient.invalidateQueries({ queryKey: mapKeys.lists() }),
    invalidateSearch: () => queryClient.invalidateQueries({ queryKey: mapKeys.search() }),
    invalidateMap: (id: string) => queryClient.invalidateQueries({ queryKey: mapKeys.detail(id) }),
    invalidateSearchByName: (name: string) => queryClient.invalidateQueries({ queryKey: mapKeys.searchByName(name) }),
    invalidateSearchByStreet: (streetName: string) => queryClient.invalidateQueries({ queryKey: mapKeys.searchByStreet(streetName) }),
  };
}

// Export types for external use
export type { MapData, MapAttributes };