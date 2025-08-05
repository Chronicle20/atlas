/**
 * React Query hook for NPC data fetching with caching and batch support
 * Provides name and icon URL data for NPCs using MapleStory.io API
 */

import { useQuery, useQueries, useQueryClient } from '@tanstack/react-query';
import { useCallback, useMemo, useEffect } from 'react';
import { mapleStoryService } from '@/services/api/maplestory.service';
import type { NpcDataResult } from '@/types/models/maplestory';

interface UseNpcDataOptions {
  enabled?: boolean;
  staleTime?: number;
  gcTime?: number;
  retry?: number;
  region?: string;
  version?: string;
  onSuccess?: (data: NpcDataResult) => void;
  onError?: (error: Error) => void;
}

interface UseNpcBatchDataOptions extends Omit<UseNpcDataOptions, 'onSuccess' | 'onError'> {
  onSuccess?: (data: NpcDataResult[]) => void;
  onError?: (error: Error) => void;
}

const DEFAULT_OPTIONS: Required<Omit<UseNpcDataOptions, 'onSuccess' | 'onError' | 'region' | 'version'>> = {
  enabled: true,
  staleTime: 30 * 60 * 1000, // 30 minutes
  gcTime: 24 * 60 * 60 * 1000, // 24 hours
  retry: 3,
};

/**
 * Generate a stable query key for NPC data
 */
function generateNpcDataQueryKey(npcId: number, region?: string, version?: string): string[] {
  return [
    'npc-data',
    region || 'GMS',
    version || '214',
    npcId.toString(),
  ];
}

/**
 * Hook for fetching single NPC data (name and icon)
 */
export function useNpcData(
  npcId: number,
  hookOptions: UseNpcDataOptions = {}
) {
  const options = useMemo(() => ({ ...DEFAULT_OPTIONS, ...hookOptions }), [hookOptions]);
  const queryClient = useQueryClient();
  
  const queryKey = generateNpcDataQueryKey(npcId, options.region, options.version);
  
  const query = useQuery({
    queryKey,
    queryFn: async (): Promise<NpcDataResult> => {
      try {
        const result = await mapleStoryService.getNpcDataWithCache(npcId, options.region, options.version);
        
        // Call success callback if provided
        if (options.onSuccess && !result.error) {
          options.onSuccess(result);
        }
        
        return result;
      } catch (error) {
        // Enhanced error logging with more context
        console.error(`Failed to fetch NPC data for ID ${npcId}:`, error);
        
        // Return structured error result instead of throwing
        const errorResult: NpcDataResult = {
          id: npcId,
          cached: false,
          error: error instanceof Error ? error.message : 'Unknown error occurred',
        };
        
        return errorResult;
      }
    },
    enabled: options.enabled && npcId > 0,
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
      console.warn(`Retrying NPC data fetch for ID ${npcId}, attempt ${failureCount + 1}/${options.retry}`);
      
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
  }, [query.isError, query.error]); // Remove options.onError from dependencies to prevent infinite loops

  // Invalidate cache for this NPC
  const invalidate = useCallback(() => {
    queryClient.invalidateQueries({
      queryKey: ['npc-data', npcId.toString()],
    });
  }, [queryClient, npcId]);

  // Prefetch related NPC data
  const prefetchNpc = useCallback((prefetchNpcId: number) => {
    const prefetchKey = generateNpcDataQueryKey(prefetchNpcId, options.region, options.version);
    queryClient.prefetchQuery({
      queryKey: prefetchKey,
      queryFn: () => mapleStoryService.getNpcDataWithCache(prefetchNpcId, options.region, options.version),
      staleTime: options.staleTime,
    });
  }, [queryClient, options.region, options.version, options.staleTime]);

  return {
    ...query,
    npcData: query.data,
    name: query.data?.name,
    iconUrl: query.data?.iconUrl,
    hasError: query.data?.error !== undefined,
    errorMessage: query.data?.error,
    cached: query.data?.cached ?? false,
    invalidate,
    prefetchNpc,
  };
}

/**
 * Hook for fetching multiple NPC data in batch
 */
export function useNpcBatchData(
  npcIds: number[],
  hookOptions: UseNpcBatchDataOptions = {}
) {
  const options = useMemo(() => ({ ...DEFAULT_OPTIONS, ...hookOptions }), [hookOptions]);
  const queryClient = useQueryClient();
  
  // Create queries for each NPC ID
  const queries = useQueries({
    queries: npcIds.map((npcId) => ({
      queryKey: generateNpcDataQueryKey(npcId, options.region, options.version),
      queryFn: async (): Promise<NpcDataResult> => {
        try {
          const result = await mapleStoryService.getNpcDataWithCache(npcId, options.region, options.version);
          
          // Ensure we return a valid result even if the API call partially fails
          const validResult: NpcDataResult = {
            id: npcId,
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
          console.error(`Failed to fetch NPC data for ID ${npcId}:`, error);
          return {
            id: npcId,
            cached: false,
            error: error instanceof Error ? error.message : 'Unknown error occurred',
          };
        }
      },
      enabled: options.enabled && npcId > 0,
      staleTime: options.staleTime,
      gcTime: options.gcTime,
      retry: (failureCount: number, error: Error) => {
        if (error?.message?.includes('404') || error?.message?.includes('not found')) {
          return false;
        }
        return failureCount < options.retry;
      },
      refetchOnWindowFocus: false,
      placeholderData: (previousData: NpcDataResult | undefined) => previousData,
    })),
  });

  // Aggregate results
  const allData = queries.map(query => query.data).filter(Boolean) as NpcDataResult[];
  const isLoading = queries.some(query => query.isLoading);
  const isError = queries.some(query => query.isError);
  const isSuccess = queries.every(query => query.isSuccess);
  const errors = queries.filter(query => query.error).map(query => query.error);

  // Handle success callback with proper dependency management
  useEffect(() => {
    if (isSuccess && allData.length === npcIds.length && options.onSuccess) {
      options.onSuccess(allData);
    }
  }, [isSuccess, allData.length, npcIds.length]); // Remove options.onSuccess from dependencies

  // Handle error callback with proper dependency management
  useEffect(() => {
    if (isError && errors.length > 0 && errors[0] && options.onError) {
      options.onError(errors[0]);
    }
  }, [isError, errors.length]); // Remove options.onError from dependencies

  // Batch invalidate
  const invalidateAll = useCallback(() => {
    npcIds.forEach(npcId => {
      queryClient.invalidateQueries({
        queryKey: ['npc-data', npcId.toString()],
      });
    });
  }, [queryClient, npcIds]);

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
 * Hook for managing NPC data cache
 */
export function useNpcDataCache() {
  const queryClient = useQueryClient();

  const getCacheStats = useCallback(() => {
    const cache = queryClient.getQueryCache();
    const npcDataQueries = cache.findAll({ queryKey: ['npc-data'] });
    
    return {
      totalQueries: npcDataQueries.length,
      activeQueries: npcDataQueries.filter(q => q.state.status === 'success').length,
      errorQueries: npcDataQueries.filter(q => q.state.status === 'error').length,
      loadingQueries: npcDataQueries.filter(q => q.state.status === 'pending').length,
    };
  }, [queryClient]);

  const clearCache = useCallback((npcId?: number) => {
    if (npcId) {
      queryClient.removeQueries({
        queryKey: ['npc-data', npcId.toString()],
      });
    } else {
      queryClient.removeQueries({
        queryKey: ['npc-data'],
      });
    }
  }, [queryClient]);

  const warmCache = useCallback(async (
    npcIds: number[],
    region?: string,
    version?: string
  ) => {
    const warmupPromises = npcIds.map(npcId => {
      const queryKey = generateNpcDataQueryKey(npcId, region, version);
      
      return queryClient.prefetchQuery({
        queryKey,
        queryFn: () => mapleStoryService.getNpcDataWithCache(npcId, region, version),
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
 * Hook for preloading NPC data
 */
export function useNpcDataPreloader() {
  const queryClient = useQueryClient();

  const preloadNpcData = useCallback(async (
    npcs: Array<{
      npcId: number;
      region?: string;
      version?: string;
    }>
  ) => {
    const preloadPromises = npcs.map(({ npcId, region, version }) => {
      const queryKey = generateNpcDataQueryKey(npcId, region, version);
      
      return queryClient.prefetchQuery({
        queryKey,
        queryFn: () => mapleStoryService.getNpcDataWithCache(npcId, region, version),
        staleTime: DEFAULT_OPTIONS.staleTime,
      });
    });

    return Promise.allSettled(preloadPromises);
  }, [queryClient]);

  return {
    preloadNpcData,
  };
}

/**
 * Optimized hook for batch NPC data fetching with debouncing and intelligent caching
 */
export function useOptimizedNpcBatchData(
  npcIds: number[],
  hookOptions: UseNpcBatchDataOptions = {}
) {
  const options = useMemo(() => ({ ...DEFAULT_OPTIONS, ...hookOptions }), [hookOptions]);
  const queryClient = useQueryClient();
  
  // Debounced batch fetcher to reduce API calls
  const batchFetch = useCallback(async (ids: number[]) => {
    try {
      const results = await mapleStoryService.getNpcDataBatch(ids, options.region, options.version);
      
      // Cache individual results for future single requests
      results.forEach((result) => {
        if (result && result.id) {
          const queryKey = generateNpcDataQueryKey(result.id, options.region, options.version);
          queryClient.setQueryData(queryKey, {
            ...result,
            name: typeof result.name === 'string' ? result.name : undefined,
            iconUrl: typeof result.iconUrl === 'string' ? result.iconUrl : undefined,
          });
        }
      });
      
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
    'npc-data-batch',
    options.region || 'GMS',
    options.version || '214',
    npcIds.sort().join(','), // Stable key based on sorted IDs
  ];

  const query = useQuery({
    queryKey: batchQueryKey,
    queryFn: () => batchFetch(npcIds),
    enabled: options.enabled && npcIds.length > 0,
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
  }, [query.isSuccess, query.data]); // Remove options.onSuccess from dependencies

  // Handle error callback with proper dependency management
  useEffect(() => {
    if (query.isError && query.error && options.onError) {
      options.onError(query.error);
    }
  }, [query.isError, query.error]); // Remove options.onError from dependencies

  // Invalidate batch cache
  const invalidateBatch = useCallback(() => {
    queryClient.invalidateQueries({
      queryKey: ['npc-data-batch'],
    });
    
    // Also invalidate individual NPC queries
    npcIds.forEach(npcId => {
      queryClient.invalidateQueries({
        queryKey: ['npc-data', npcId.toString()],
      });
    });
  }, [queryClient, npcIds]);

  return {
    data: query.data || [],
    isLoading: query.isLoading,
    isError: query.isError,
    isSuccess: query.isSuccess,
    error: query.error,
    invalidateBatch,
  };
}