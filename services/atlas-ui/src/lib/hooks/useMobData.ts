/**
 * React Query hook for Mob/Monster data fetching with caching
 * Provides name and icon URL data for mobs using atlas-data API and atlas-assets
 */

import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback, useMemo } from 'react';
import { useTenant } from '@/context/tenant-context';
import { monstersService } from '@/services/api/monsters.service';
import { getAssetIconUrl } from '@/lib/utils/asset-url';

interface MobDataResult {
  id: number;
  name?: string;
  iconUrl?: string;
  cached: boolean;
  error?: string;
}

interface UseMobDataOptions {
  enabled?: boolean;
  staleTime?: number;
  gcTime?: number;
  retry?: number;
}

const DEFAULT_OPTIONS = {
  enabled: true,
  staleTime: 30 * 60 * 1000,
  gcTime: 24 * 60 * 60 * 1000,
  retry: 3,
};

function generateMobDataQueryKey(mobId: number, tenantId?: string): string[] {
  return ['mob-data', tenantId || '', mobId.toString()];
}

/**
 * Hook for fetching single mob data (name and icon)
 */
export function useMobData(
  mobId: number,
  hookOptions: UseMobDataOptions = {}
) {
  const options = useMemo(() => ({ ...DEFAULT_OPTIONS, ...hookOptions }), [hookOptions]);
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();

  const queryKey = generateMobDataQueryKey(mobId, activeTenant?.id);

  const query = useQuery({
    queryKey,
    queryFn: async (): Promise<MobDataResult> => {
      if (!activeTenant) {
        return { id: mobId, cached: false, error: 'No active tenant' };
      }

      const iconUrl = getAssetIconUrl(
        activeTenant.id,
        activeTenant.attributes.region,
        activeTenant.attributes.majorVersion,
        activeTenant.attributes.minorVersion,
        'mob',
        mobId,
      );

      try {
        const name = await monstersService.getMonsterName(mobId.toString(), activeTenant);
        return { id: mobId, name, iconUrl, cached: false };
      } catch (error) {
        console.error(`Failed to fetch mob name for ID ${mobId}:`, error);
        return {
          id: mobId,
          iconUrl,
          cached: false,
          error: error instanceof Error ? error.message : 'Unknown error occurred',
        };
      }
    },
    enabled: options.enabled && mobId > 0 && !!activeTenant,
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
    queryClient.invalidateQueries({ queryKey: ['mob-data', mobId.toString()] });
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
