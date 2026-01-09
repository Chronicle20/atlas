/**
 * React Query hook for Skill data fetching with caching
 * Provides name and icon URL data for skills using MapleStory.io API
 */

import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback, useMemo } from 'react';
import { mapleStoryService } from '@/services/api/maplestory.service';
import type { SkillDataResult } from '@/types/models/maplestory';

interface UseSkillDataOptions {
  enabled?: boolean;
  staleTime?: number;
  gcTime?: number;
  retry?: number;
  region?: string;
  version?: string;
}

const DEFAULT_OPTIONS: Required<Omit<UseSkillDataOptions, 'region' | 'version'>> = {
  enabled: true,
  staleTime: 30 * 60 * 1000, // 30 minutes
  gcTime: 24 * 60 * 60 * 1000, // 24 hours
  retry: 3,
};

/**
 * Generate a stable query key for skill data
 */
function generateSkillDataQueryKey(skillId: number, region?: string, version?: string): string[] {
  return [
    'skill-data',
    region || 'GMS',
    version || '214',
    skillId.toString(),
  ];
}

/**
 * Hook for fetching single skill data (name and icon)
 */
export function useSkillData(
  skillId: number,
  hookOptions: UseSkillDataOptions = {}
) {
  const options = useMemo(() => ({ ...DEFAULT_OPTIONS, ...hookOptions }), [hookOptions]);
  const queryClient = useQueryClient();

  const queryKey = generateSkillDataQueryKey(skillId, options.region, options.version);

  const query = useQuery({
    queryKey,
    queryFn: async (): Promise<SkillDataResult> => {
      try {
        return await mapleStoryService.getSkillDataWithCache(skillId, options.region, options.version);
      } catch (error) {
        console.error(`Failed to fetch skill data for ID ${skillId}:`, error);
        return {
          id: skillId,
          cached: false,
          error: error instanceof Error ? error.message : 'Unknown error occurred',
        };
      }
    },
    enabled: options.enabled && skillId > 0,
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
      queryKey: ['skill-data', skillId.toString()],
    });
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
