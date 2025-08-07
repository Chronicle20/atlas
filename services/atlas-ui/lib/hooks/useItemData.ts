/**
 * React Query hook for Item data fetching with caching and batch support
 * Provides name and icon URL data for items using MapleStory.io API
 */

import { useQuery, useQueries, useQueryClient } from '@tanstack/react-query';
import { useCallback, useMemo, useEffect } from 'react';
import { mapleStoryService } from '@/services/api/maplestory.service';
import type { ItemDataResult } from '@/types/models/maplestory';

interface UseItemDataOptions {
  enabled?: boolean;
  staleTime?: number;
  gcTime?: number;
  retry?: number;
  region?: string;
  version?: string;
  onSuccess?: (data: ItemDataResult) => void;
  onError?: (error: Error) => void;
}

interface UseItemBatchDataOptions extends Omit<UseItemDataOptions, 'onSuccess' | 'onError'> {
  onSuccess?: (data: ItemDataResult[]) => void;
  onError?: (error: Error) => void;
}

const DEFAULT_OPTIONS: Required<Omit<UseItemDataOptions, 'onSuccess' | 'onError' | 'region' | 'version'>> = {
  enabled: true,
  staleTime: 30 * 60 * 1000, // 30 minutes
  gcTime: 24 * 60 * 60 * 1000, // 24 hours
  retry: 3,
};

/**
 * Generate a stable query key for item data
 */
function generateItemDataQueryKey(itemId: number, region?: string, version?: string): string[] {
  return [
    'item-data',
    region || 'GMS',
    version || '214',
    itemId.toString(),
  ];
}

/**
 * Hook for fetching single item data (name and icon)
 */
export function useItemData(
  itemId: number,
  hookOptions: UseItemDataOptions = {}
) {
  const options = useMemo(() => ({ ...DEFAULT_OPTIONS, ...hookOptions }), [hookOptions]);
  const queryClient = useQueryClient();
  
  const queryKey = generateItemDataQueryKey(itemId, options.region, options.version);
  
  const query = useQuery({
    queryKey,
    queryFn: async (): Promise<ItemDataResult> => {
      try {
        const result = await mapleStoryService.getItemDataWithCache(itemId, options.region, options.version);
        
        // Call success callback if provided
        if (options.onSuccess && !result.error) {
          options.onSuccess(result);
        }
        
        return result;
      } catch (error) {
        // Enhanced error logging with more context
        console.error(`Failed to fetch item data for ID ${itemId}:`, error);
        
        // Return structured error result instead of throwing
        const errorResult: ItemDataResult = {
          id: itemId,
          cached: false,
          error: error instanceof Error ? error.message : 'Unknown error occurred',
        };
        
        return errorResult;
      }
    },
    enabled: options.enabled && itemId > 0,
    staleTime: options.staleTime,
    gcTime: options.gcTime,
    retry: (failureCount, error) => {
      // Enhanced retry logic with better error classification
      const errorMessage = error?.message?.toLowerCase() || '';
      
      // Don't retry for client errors (4xx)
      if (errorMessage.includes('404') || 
          errorMessage.includes('not found') ||
          errorMessage.includes('400') ||
          errorMessage.includes('bad request') ||
          errorMessage.includes('unauthorized') ||
          errorMessage.includes('forbidden')) {
        return false;
      }
      
      // Don't retry if we've exceeded the limit
      if (failureCount >= options.retry) {
        return false;
      }
      
      // Log retry attempts for monitoring
      console.warn(`Retrying item data fetch for ID ${itemId}, attempt ${failureCount + 1}/${options.retry}`);
      
      return true;
    },
    retryDelay: (attemptIndex) => {
      // Exponential backoff with jitter
      const baseDelay = 1000; // 1 second
      const maxDelay = 10000; // 10 seconds
      const exponentialDelay = Math.min(baseDelay * Math.pow(2, attemptIndex), maxDelay);
      const jitter = Math.random() * 1000; // Add up to 1 second of jitter
      return exponentialDelay + jitter;
    },
    refetchOnWindowFocus: false,
    refetchOnReconnect: true, // Refetch when connection is restored
    // Keep previous data while refetching to avoid loading flicker
    placeholderData: (previousData) => previousData,
  });

  // Handle error callback with proper dependency management
  useEffect(() => {
    if (query.isError && query.error && options.onError) {
      options.onError(query.error);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query.isError, query.error]); // Remove options.onError from dependencies to prevent infinite loops

  // Invalidate cache for this item
  const invalidate = useCallback(() => {
    queryClient.invalidateQueries({
      queryKey: ['item-data', itemId.toString()],
    });
  }, [queryClient, itemId]);

  // Prefetch related item data
  const prefetchItem = useCallback((prefetchItemId: number) => {
    const prefetchKey = generateItemDataQueryKey(prefetchItemId, options.region, options.version);
    queryClient.prefetchQuery({
      queryKey: prefetchKey,
      queryFn: () => mapleStoryService.getItemDataWithCache(prefetchItemId, options.region, options.version),
      staleTime: options.staleTime,
    });
  }, [queryClient, options.region, options.version, options.staleTime]);

  return {
    ...query,
    itemData: query.data,
    name: query.data?.name,
    iconUrl: query.data?.iconUrl,
    hasError: query.data?.error !== undefined,
    errorMessage: query.data?.error,
    cached: query.data?.cached ?? false,
    invalidate,
    prefetchItem,
  };
}

/**
 * Hook for fetching multiple item data in batch
 */
export function useItemBatchData(
  itemIds: number[],
  hookOptions: UseItemBatchDataOptions = {}
) {
  const options = useMemo(() => ({ ...DEFAULT_OPTIONS, ...hookOptions }), [hookOptions]);
  const queryClient = useQueryClient();
  
  // Create queries for each item ID
  const queries = useQueries({
    queries: itemIds.map((itemId) => ({
      queryKey: generateItemDataQueryKey(itemId, options.region, options.version),
      queryFn: async (): Promise<ItemDataResult> => {
        try {
          const result = await mapleStoryService.getItemDataWithCache(itemId, options.region, options.version);
          
          // Ensure we return a valid result even if the API call partially fails
          const validResult: ItemDataResult = {
            id: itemId,
            cached: result.cached || false,
          };
          
          if (typeof result.name === 'string') {
            validResult.name = result.name;
          }
          
          if (typeof result.iconUrl === 'string') {
            validResult.iconUrl = result.iconUrl;
          }
          
          if (result.error) {
            validResult.error = result.error;
          }
          
          return validResult;
        } catch (error) {
          console.error(`Failed to fetch item data for ID ${itemId}:`, error);
          return {
            id: itemId,
            cached: false,
            error: error instanceof Error ? error.message : 'Unknown error occurred',
          };
        }
      },
      enabled: options.enabled && itemId > 0,
      staleTime: options.staleTime,
      gcTime: options.gcTime,
      retry: (failureCount: number, error: Error) => {
        if (error?.message?.includes('404') || error?.message?.includes('not found')) {
          return false;
        }
        return failureCount < options.retry;
      },
      refetchOnWindowFocus: false,
      placeholderData: (previousData: ItemDataResult | undefined) => previousData,
    })),
  });

  // Aggregate results
  const allData = queries.map(query => query.data).filter(Boolean) as ItemDataResult[];
  const isLoading = queries.some(query => query.isLoading);
  const isError = queries.some(query => query.isError);
  const isSuccess = queries.every(query => query.isSuccess);
  const errors = queries.filter(query => query.error).map(query => query.error);

  // Handle success callback with proper dependency management
  useEffect(() => {
    if (isSuccess && allData.length === itemIds.length && options.onSuccess) {
      options.onSuccess(allData);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isSuccess, allData.length, itemIds.length]); // Remove options.onSuccess from dependencies

  // Handle error callback with proper dependency management
  useEffect(() => {
    if (isError && errors.length > 0 && errors[0] && options.onError) {
      options.onError(errors[0]);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isError, errors.length]); // Remove options.onError from dependencies

  // Batch invalidate
  const invalidateAll = useCallback(() => {
    itemIds.forEach(itemId => {
      queryClient.invalidateQueries({
        queryKey: ['item-data', itemId.toString()],
      });
    });
  }, [queryClient, itemIds]);

  return {
    queries,
    data: allData,
    isLoading,
    isError,
    isSuccess,
    errors,
    invalidateAll,
  };
}

/**
 * Hook for managing item data cache
 */
export function useItemDataCache() {
  const queryClient = useQueryClient();

  const getCacheStats = useCallback(() => {
    const cache = queryClient.getQueryCache();
    const itemDataQueries = cache.findAll({ queryKey: ['item-data'] });
    
    return {
      totalQueries: itemDataQueries.length,
      activeQueries: itemDataQueries.filter(q => q.state.status === 'success').length,
      errorQueries: itemDataQueries.filter(q => q.state.status === 'error').length,
      loadingQueries: itemDataQueries.filter(q => q.state.status === 'pending').length,
    };
  }, [queryClient]);

  const clearCache = useCallback((itemId?: number) => {
    if (itemId) {
      queryClient.removeQueries({
        queryKey: ['item-data', itemId.toString()],
      });
    } else {
      queryClient.removeQueries({
        queryKey: ['item-data'],
      });
    }
  }, [queryClient]);

  const warmCache = useCallback(async (
    itemIds: number[],
    region?: string,
    version?: string
  ) => {
    const warmupPromises = itemIds.map(itemId => {
      const queryKey = generateItemDataQueryKey(itemId, region, version);
      
      return queryClient.prefetchQuery({
        queryKey,
        queryFn: () => mapleStoryService.getItemDataWithCache(itemId, region, version),
        staleTime: DEFAULT_OPTIONS.staleTime,
      });
    });

    return Promise.allSettled(warmupPromises);
  }, [queryClient]);

  return {
    getCacheStats,
    clearCache,
    warmCache,
  };
}

/**
 * Optimized hook for batch item data fetching with debouncing and intelligent caching
 */
export function useOptimizedItemBatchData(
  itemIds: number[],
  hookOptions: UseItemBatchDataOptions = {}
) {
  const options = useMemo(() => ({ ...DEFAULT_OPTIONS, ...hookOptions }), [hookOptions]);
  const queryClient = useQueryClient();
  
  // Debounced batch fetcher to reduce API calls
  const batchFetch = useCallback(async (ids: number[]) => {
    try {
      // Use batch size consistent with NPC implementation
      const results = await mapleStoryService.getItemDataBatch(ids, options.region, options.version, 20);
      
      console.log(`Fetched metadata for ${results.length} items out of ${ids.length} requested using region: ${options.region}, version: ${options.version}`);
      
      // Cache individual results for future single requests
      results.forEach((result) => {
        if (result && result.id) {
          const queryKey = generateItemDataQueryKey(result.id, options.region, options.version);
          queryClient.setQueryData(queryKey, {
            ...result,
            name: typeof result.name === 'string' ? result.name : undefined,
            iconUrl: typeof result.iconUrl === 'string' ? result.iconUrl : undefined,
          });
        }
      });
      
      // Log any items that failed to fetch
      const failedItems = results.filter(r => r.error);
      if (failedItems.length > 0) {
        console.warn(`Failed to fetch metadata for ${failedItems.length} items:`, failedItems);
      }
      
      return results;
    } catch (error) {
      console.error('Batch fetch failed:', error);
      // Return empty results for failed batch
      return ids.map(id => ({
        id,
        cached: false,
        error: error instanceof Error ? error.message : 'Batch fetch failed',
      }));
    }
  }, [options.region, options.version, queryClient]);

  // Use a single query for the entire batch
  const batchQueryKey = [
    'item-data-batch',
    options.region || 'GMS',
    options.version || '214',
    itemIds.sort().join(','), // Stable key based on sorted IDs
  ];

  const query = useQuery({
    queryKey: batchQueryKey,
    queryFn: () => batchFetch(itemIds),
    enabled: options.enabled && itemIds.length > 0,
    staleTime: options.staleTime,
    gcTime: options.gcTime,
    retry: options.retry,
    refetchOnWindowFocus: false,
    refetchOnReconnect: true,
    placeholderData: (previousData) => previousData,
  });

  // Handle success callback with proper dependency management
  useEffect(() => {
    if (query.isSuccess && query.data && options.onSuccess) {
      options.onSuccess(query.data);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query.isSuccess, query.data]); // Remove options.onSuccess from dependencies

  // Handle error callback with proper dependency management
  useEffect(() => {
    if (query.isError && query.error && options.onError) {
      options.onError(query.error);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query.isError, query.error]); // Remove options.onError from dependencies

  // Invalidate batch cache
  const invalidateBatch = useCallback(() => {
    queryClient.invalidateQueries({
      queryKey: ['item-data-batch'],
    });
    
    // Also invalidate individual item queries
    itemIds.forEach(itemId => {
      queryClient.invalidateQueries({
        queryKey: ['item-data', itemId.toString()],
      });
    });
  }, [queryClient, itemIds]);

  return {
    data: query.data || [],
    isLoading: query.isLoading,
    isError: query.isError,
    isSuccess: query.isSuccess,
    error: query.error,
    invalidateBatch,
  };
}