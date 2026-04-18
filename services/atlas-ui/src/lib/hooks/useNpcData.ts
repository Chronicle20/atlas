/**
 * React Query hook for NPC data fetching with caching and batch support
 * Provides name and icon URL data for NPCs using atlas-data API and atlas-assets
 */

import { useQuery, useQueries, useQueryClient } from '@tanstack/react-query';
import { useCallback, useMemo, useEffect } from 'react';
import { useTenant } from '@/context/tenant-context';
import { npcsService } from '@/services/api/npcs.service';
import { getAssetIconUrl } from '@/lib/utils/asset-url';

interface NpcDataResult {
  id: number;
  name?: string;
  iconUrl?: string;
  cached: boolean;
  error?: string;
}

interface UseNpcDataOptions {
  enabled?: boolean;
  staleTime?: number;
  gcTime?: number;
  retry?: number;
  onSuccess?: (data: NpcDataResult) => void;
  onError?: (error: Error) => void;
}

interface UseNpcBatchDataOptions extends Omit<UseNpcDataOptions, 'onSuccess' | 'onError'> {
  onSuccess?: (data: NpcDataResult[]) => void;
  onError?: (error: Error) => void;
}

const DEFAULT_OPTIONS = {
  enabled: true,
  staleTime: 30 * 60 * 1000, // 30 minutes
  gcTime: 24 * 60 * 60 * 1000, // 24 hours
  retry: 3,
};

function generateNpcDataQueryKey(npcId: number, tenantId?: string): string[] {
  return ['npc-data', tenantId || '', npcId.toString()];
}

/**
 * Hook for fetching single NPC data (name and icon)
 */
export function useNpcData(
  npcId: number,
  hookOptions: UseNpcDataOptions = {}
) {
  const {
    enabled = DEFAULT_OPTIONS.enabled,
    staleTime = DEFAULT_OPTIONS.staleTime,
    gcTime = DEFAULT_OPTIONS.gcTime,
    retry = DEFAULT_OPTIONS.retry,
    onSuccess,
    onError,
  } = hookOptions;

  const options = useMemo(() => ({
    enabled, staleTime, gcTime, retry, onSuccess, onError,
  }), [enabled, staleTime, gcTime, retry, onSuccess, onError]);

  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();

  const queryKey = generateNpcDataQueryKey(npcId, activeTenant?.id);

  const query = useQuery({
    queryKey,
    queryFn: async (): Promise<NpcDataResult> => {
      if (!activeTenant) {
        return { id: npcId, cached: false, error: 'No active tenant' };
      }

      const iconUrl = getAssetIconUrl(
        activeTenant.id,
        activeTenant.attributes.region,
        activeTenant.attributes.majorVersion,
        activeTenant.attributes.minorVersion,
        'npc',
        npcId,
      );

      try {
        const name = await npcsService.getNpcName(npcId, activeTenant);
        const result: NpcDataResult = { id: npcId, name, iconUrl, cached: false };
        if (options.onSuccess) options.onSuccess(result);
        return result;
      } catch (error) {
        console.error(`Failed to fetch NPC name for ID ${npcId}:`, error);
        return {
          id: npcId,
          iconUrl,
          cached: false,
          error: error instanceof Error ? error.message : 'Unknown error occurred',
        };
      }
    },
    enabled: options.enabled && npcId > 0 && !!activeTenant,
    staleTime: options.staleTime,
    gcTime: options.gcTime,
    retry: (failureCount, error) => {
      const errorMessage = error?.message?.toLowerCase() || '';
      if (errorMessage.includes('404') || errorMessage.includes('not found') ||
          errorMessage.includes('400') || errorMessage.includes('bad request')) {
        return false;
      }
      return failureCount < options.retry;
    },
    retryDelay: (attemptIndex) => {
      const baseDelay = 1000;
      const maxDelay = 10000;
      const exponentialDelay = Math.min(baseDelay * Math.pow(2, attemptIndex), maxDelay);
      return exponentialDelay + Math.random() * 1000;
    },
    refetchOnWindowFocus: false,
    refetchOnReconnect: true,
    placeholderData: (previousData) => previousData,
  });

  const handleError = options.onError;
  useEffect(() => {
    if (query.isError && query.error && handleError) {
      handleError(query.error);
    }
  }, [query.isError, query.error, handleError]);

  const invalidate = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['npc-data', npcId.toString()] });
  }, [queryClient, npcId]);

  const prefetchNpc = useCallback((prefetchNpcId: number) => {
    if (!activeTenant) return;
    const prefetchKey = generateNpcDataQueryKey(prefetchNpcId, activeTenant.id);
    queryClient.prefetchQuery({
      queryKey: prefetchKey,
      queryFn: async () => {
        const iconUrl = getAssetIconUrl(
          activeTenant.id, activeTenant.attributes.region,
          activeTenant.attributes.majorVersion, activeTenant.attributes.minorVersion,
          'npc', prefetchNpcId,
        );
        try {
          const name = await npcsService.getNpcName(prefetchNpcId, activeTenant);
          return { id: prefetchNpcId, name, iconUrl, cached: false };
        } catch {
          return { id: prefetchNpcId, iconUrl, cached: false };
        }
      },
      staleTime: options.staleTime,
    });
  }, [queryClient, activeTenant, options.staleTime]);

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
  const {
    enabled = DEFAULT_OPTIONS.enabled,
    staleTime = DEFAULT_OPTIONS.staleTime,
    gcTime = DEFAULT_OPTIONS.gcTime,
    retry = DEFAULT_OPTIONS.retry,
    onSuccess,
    onError,
  } = hookOptions;

  const options = useMemo(() => ({
    enabled, staleTime, gcTime, retry, onSuccess, onError,
  }), [enabled, staleTime, gcTime, retry, onSuccess, onError]);

  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();

  const queries = useQueries({
    queries: npcIds.map((npcId) => ({
      queryKey: generateNpcDataQueryKey(npcId, activeTenant?.id),
      queryFn: async (): Promise<NpcDataResult> => {
        if (!activeTenant) {
          return { id: npcId, cached: false, error: 'No active tenant' };
        }
        const iconUrl = getAssetIconUrl(
          activeTenant.id, activeTenant.attributes.region,
          activeTenant.attributes.majorVersion, activeTenant.attributes.minorVersion,
          'npc', npcId,
        );
        try {
          const name = await npcsService.getNpcName(npcId, activeTenant);
          return { id: npcId, name, iconUrl, cached: false };
        } catch (error) {
          return {
            id: npcId, iconUrl, cached: false,
            error: error instanceof Error ? error.message : 'Unknown error',
          };
        }
      },
      enabled: options.enabled && npcId > 0 && !!activeTenant,
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

  const allData = useMemo(
    () => queries.map(query => query.data).filter(Boolean) as NpcDataResult[],
    [queries]
  );
  const isLoading = queries.some(query => query.isLoading);
  const isError = queries.some(query => query.isError);
  const isSuccess = queries.every(query => query.isSuccess);
  const errors = useMemo(
    () => queries.filter(query => query.error).map(query => query.error),
    [queries]
  );

  const handleSuccess = options.onSuccess;
  useEffect(() => {
    if (isSuccess && allData.length === npcIds.length && handleSuccess) {
      handleSuccess(allData);
    }
  }, [isSuccess, allData.length, npcIds.length, handleSuccess, allData]);

  const handleError = options.onError;
  useEffect(() => {
    if (isError && errors.length > 0 && errors[0] && handleError) {
      handleError(errors[0]);
    }
  }, [isError, errors.length, errors, handleError]);

  const invalidateAll = useCallback(() => {
    npcIds.forEach(npcId => {
      queryClient.invalidateQueries({ queryKey: ['npc-data', npcId.toString()] });
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
      queryClient.removeQueries({ queryKey: ['npc-data', npcId.toString()] });
    } else {
      queryClient.removeQueries({ queryKey: ['npc-data'] });
    }
  }, [queryClient]);

  return { getCacheStats, clearCache };
}
