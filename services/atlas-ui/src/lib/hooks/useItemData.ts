/**
 * React Query hook for Item data fetching with caching and batch support
 * Provides name and icon URL data for items using atlas-data API and atlas-assets
 */

import { useQuery, useQueries, useQueryClient } from '@tanstack/react-query';
import { useCallback, useMemo, useEffect } from 'react';
import { useTenant } from '@/context/tenant-context';
import { itemsService } from '@/services/api/items.service';
import { getAssetIconUrl } from '@/lib/utils/asset-url';

interface ItemDataResult {
  id: number;
  name?: string;
  iconUrl?: string;
  cached: boolean;
  error?: string;
}

interface UseItemDataOptions {
  enabled?: boolean;
  staleTime?: number;
  gcTime?: number;
  retry?: number;
  onSuccess?: (data: ItemDataResult) => void;
  onError?: (error: Error) => void;
}

interface UseItemBatchDataOptions extends Omit<UseItemDataOptions, 'onSuccess' | 'onError'> {
  onSuccess?: (data: ItemDataResult[]) => void;
  onError?: (error: Error) => void;
}

const DEFAULT_OPTIONS = {
  enabled: true,
  staleTime: 30 * 60 * 1000,
  gcTime: 24 * 60 * 60 * 1000,
  retry: 3,
};

function generateItemDataQueryKey(itemId: number, tenantId?: string): string[] {
  return ['item-data', tenantId || '', itemId.toString()];
}

/**
 * Hook for fetching single item data (name and icon)
 */
export function useItemData(
  itemId: number,
  hookOptions: UseItemDataOptions = {}
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

  const queryKey = generateItemDataQueryKey(itemId, activeTenant?.id);

  const query = useQuery({
    queryKey,
    queryFn: async (): Promise<ItemDataResult> => {
      if (!activeTenant) {
        return { id: itemId, cached: false, error: 'No active tenant' };
      }

      const iconUrl = getAssetIconUrl(
        activeTenant.id,
        activeTenant.attributes.region,
        activeTenant.attributes.majorVersion,
        activeTenant.attributes.minorVersion,
        'item',
        itemId,
      );

      try {
        const name = await itemsService.getItemName(itemId.toString(), activeTenant);
        const result: ItemDataResult = { id: itemId, name, iconUrl, cached: false };
        if (options.onSuccess) options.onSuccess(result);
        return result;
      } catch (error) {
        console.error(`Failed to fetch item name for ID ${itemId}:`, error);
        return {
          id: itemId,
          iconUrl,
          cached: false,
          error: error instanceof Error ? error.message : 'Unknown error occurred',
        };
      }
    },
    enabled: options.enabled && itemId > 0 && !!activeTenant,
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
      return Math.min(baseDelay * Math.pow(2, attemptIndex), maxDelay) + Math.random() * 1000;
    },
    refetchOnWindowFocus: false,
    refetchOnReconnect: true,
    placeholderData: (previousData) => previousData,
  });

  useEffect(() => {
    if (query.isError && query.error && options.onError) {
      options.onError(query.error);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query.isError, query.error]);

  const invalidate = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['item-data', itemId.toString()] });
  }, [queryClient, itemId]);

  const prefetchItem = useCallback((prefetchItemId: number) => {
    if (!activeTenant) return;
    const prefetchKey = generateItemDataQueryKey(prefetchItemId, activeTenant.id);
    queryClient.prefetchQuery({
      queryKey: prefetchKey,
      queryFn: async () => {
        const iconUrl = getAssetIconUrl(
          activeTenant.id, activeTenant.attributes.region,
          activeTenant.attributes.majorVersion, activeTenant.attributes.minorVersion,
          'item', prefetchItemId,
        );
        try {
          const name = await itemsService.getItemName(prefetchItemId.toString(), activeTenant);
          return { id: prefetchItemId, name, iconUrl, cached: false };
        } catch {
          return { id: prefetchItemId, iconUrl, cached: false };
        }
      },
      staleTime: options.staleTime,
    });
  }, [queryClient, activeTenant, options.staleTime]);

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
    queries: itemIds.map((itemId) => ({
      queryKey: generateItemDataQueryKey(itemId, activeTenant?.id),
      queryFn: async (): Promise<ItemDataResult> => {
        if (!activeTenant) {
          return { id: itemId, cached: false, error: 'No active tenant' };
        }
        const iconUrl = getAssetIconUrl(
          activeTenant.id, activeTenant.attributes.region,
          activeTenant.attributes.majorVersion, activeTenant.attributes.minorVersion,
          'item', itemId,
        );
        try {
          const name = await itemsService.getItemName(itemId.toString(), activeTenant);
          return { id: itemId, name, iconUrl, cached: false };
        } catch (error) {
          return {
            id: itemId, iconUrl, cached: false,
            error: error instanceof Error ? error.message : 'Unknown error',
          };
        }
      },
      enabled: options.enabled && itemId > 0 && !!activeTenant,
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

  const allData = queries.map(query => query.data).filter(Boolean) as ItemDataResult[];
  const isLoading = queries.some(query => query.isLoading);
  const isError = queries.some(query => query.isError);
  const isSuccess = queries.every(query => query.isSuccess);
  const errors = queries.filter(query => query.error).map(query => query.error);

  useEffect(() => {
    if (isSuccess && allData.length === itemIds.length && options.onSuccess) {
      options.onSuccess(allData);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isSuccess, allData.length, itemIds.length]);

  useEffect(() => {
    if (isError && errors.length > 0 && errors[0] && options.onError) {
      options.onError(errors[0]);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isError, errors.length]);

  const invalidateAll = useCallback(() => {
    itemIds.forEach(itemId => {
      queryClient.invalidateQueries({ queryKey: ['item-data', itemId.toString()] });
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
  const { activeTenant } = useTenant();

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
      queryClient.removeQueries({ queryKey: ['item-data', itemId.toString()] });
    } else {
      queryClient.removeQueries({ queryKey: ['item-data'] });
    }
  }, [queryClient]);

  const warmCache = useCallback(async (itemIds: number[]) => {
    if (!activeTenant) return [];
    return Promise.allSettled(
      itemIds.map((itemId) => {
        const queryKey = generateItemDataQueryKey(itemId, activeTenant.id);
        return queryClient.prefetchQuery({
          queryKey,
          queryFn: async (): Promise<ItemDataResult> => {
            const iconUrl = getAssetIconUrl(
              activeTenant.id, activeTenant.attributes.region,
              activeTenant.attributes.majorVersion, activeTenant.attributes.minorVersion,
              'item', itemId,
            );
            try {
              const name = await itemsService.getItemName(itemId.toString(), activeTenant);
              return { id: itemId, name, iconUrl, cached: false };
            } catch {
              return { id: itemId, iconUrl, cached: false };
            }
          },
          staleTime: DEFAULT_OPTIONS.staleTime,
        });
      }),
    );
  }, [queryClient, activeTenant]);

  return { getCacheStats, clearCache, warmCache };
}
