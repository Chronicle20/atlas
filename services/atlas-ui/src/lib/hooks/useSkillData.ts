/**
 * React Query hook for Skill data fetching with caching
 * Provides name and icon URL data for skills using atlas-data API and atlas-assets
 */

import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback, useMemo } from 'react';
import { useTenant } from '@/context/tenant-context';
import { skillsService } from '@/services/api/skills.service';
import { getAssetIconUrl } from '@/lib/utils/asset-url';

interface SkillDataResult {
  id: number;
  name?: string;
  iconUrl?: string;
  cached: boolean;
  error?: string;
}

interface UseSkillDataOptions {
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

function generateSkillDataQueryKey(skillId: number, tenantId?: string): string[] {
  return ['skill-data', tenantId || '', skillId.toString()];
}

/**
 * Hook for fetching single skill data (name and icon)
 */
export function useSkillData(
  skillId: number,
  hookOptions: UseSkillDataOptions = {}
) {
  const options = useMemo(() => ({ ...DEFAULT_OPTIONS, ...hookOptions }), [hookOptions]);
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();

  const queryKey = generateSkillDataQueryKey(skillId, activeTenant?.id);

  const query = useQuery({
    queryKey,
    queryFn: async (): Promise<SkillDataResult> => {
      if (!activeTenant) {
        return { id: skillId, cached: false, error: 'No active tenant' };
      }

      const iconUrl = getAssetIconUrl(
        activeTenant.id,
        activeTenant.attributes.region,
        activeTenant.attributes.majorVersion,
        activeTenant.attributes.minorVersion,
        'skill',
        skillId,
      );

      try {
        const name = await skillsService.getSkillName(skillId.toString(), activeTenant);
        return { id: skillId, name, iconUrl, cached: false };
      } catch (error) {
        console.error(`Failed to fetch skill name for ID ${skillId}:`, error);
        return {
          id: skillId,
          iconUrl,
          cached: false,
          error: error instanceof Error ? error.message : 'Unknown error occurred',
        };
      }
    },
    enabled: options.enabled && skillId > 0 && !!activeTenant,
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
    queryClient.invalidateQueries({ queryKey: ['skill-data', skillId.toString()] });
  }, [queryClient, skillId]);

  return {
    ...query,
    skillData: query.data,
    name: query.data?.name,
    iconUrl: query.data?.iconUrl,
    hasError: query.data?.error !== undefined,
    errorMessage: query.data?.error,
    cached: query.data?.cached ?? false,
    invalidate,
  };
}
