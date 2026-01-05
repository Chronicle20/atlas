/**
 * React Query hook for Mob/Monster data fetching with caching
 * Provides name and icon URL data for mobs using MapleStory.io API
 */

import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback, useMemo } from 'react';
import { mapleStoryService } from '@/services/api/maplestory.service';
import type { MobDataResult } from '@/types/models/maplestory';

interface UseMobDataOptions {
  enabled?: boolean;
  staleTime?: number;
  gcTime?: number;
  retry?: number;
  region?: string;
  version?: string;
}

const DEFAULT_OPTIONS: Required<Omit<UseMobDataOptions, 'region' | 'version'>> = {
  enabled: true,
  staleTime: 30 * 60 * 1000, // 30 minutes
  gcTime: 24 * 60 * 60 * 1000, // 24 hours
  retry: 3,
};

/**
 * Generate a stable query key for mob data
 */
function generateMobDataQueryKey(mobId: number, region?: string, version?: string): string[] {
  return [
    'mob-data',
    region || 'GMS',
    version || '214',
    mobId.toString(),
  ];
}

/**
 * Hook for fetching single mob data (name and icon)
 */
export function useMobData(
  mobId: number,
  hookOptions: UseMobDataOptions = {}
) {
  const options = useMemo(() => ({ ...DEFAULT_OPTIONS, ...hookOptions }), [hookOptions]);
  const queryClient = useQueryClient();

  const queryKey = generateMobDataQueryKey(mobId, options.region, options.version);

  const query = useQuery({
    queryKey,
    queryFn: async (): Promise<MobDataResult> => {
      try {
        return await mapleStoryService.getMobDataWithCache(mobId, options.region, options.version);
      } catch (error) {
        console.error(`Failed to fetch mob data for ID ${mobId}:`, error);
        return {
          id: mobId,
          cached: false,
          error: error instanceof Error ? error.message : 'Unknown error occurred',
        };
      }
    },
    enabled: options.enabled && mobId > 0,
    staleTime: options.staleTime,
    gcTime: options.gcTime,
    retry: (failureCount, error) => {
      const errorMessage = error?.message?.toLowerCase() || '';
      if (errorMessage.includes('404') || errorMessage.includes('not found')) {
        return false;
      }
      return failureCount < options.retry;
    },
    refetchOnWindowFocus: false,
    placeholderData: (previousData) => previousData,
  });

  const invalidate = useCallback(() => {
    queryClient.invalidateQueries({
      queryKey: ['mob-data', mobId.toString()],
    });
  }, [queryClient, mobId]);

  return {
    ...query,
    mobData: query.data,
    name: query.data?.name,
    iconUrl: query.data?.iconUrl,
    hasError: query.data?.error !== undefined,
    errorMessage: query.data?.error,
    cached: query.data?.cached ?? false,
    invalidate,
  };
}
