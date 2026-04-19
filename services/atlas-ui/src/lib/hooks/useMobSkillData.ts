import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';
import { useTenant } from '@/context/tenant-context';
import { mobSkillsService } from '@/services/api/mob-skills.service';

interface MobSkillDataResult {
  id: number;
  name?: string;
  error?: string;
}

interface UseMobSkillDataOptions {
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

export function useMobSkillData(
  skillId: number,
  hookOptions: UseMobSkillDataOptions = {},
) {
  const options = useMemo(() => ({ ...DEFAULT_OPTIONS, ...hookOptions }), [hookOptions]);
  const { activeTenant } = useTenant();

  const query = useQuery({
    queryKey: ['mob-skill-data', activeTenant?.id || '', skillId.toString()],
    queryFn: async (): Promise<MobSkillDataResult> => {
      try {
        const name = await mobSkillsService.getMobSkillName(skillId);
        return { id: skillId, name };
      } catch (error) {
        return {
          id: skillId,
          error: error instanceof Error ? error.message : 'Unknown error',
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

  return {
    ...query,
    name: query.data?.name,
    hasError: query.data?.error !== undefined,
  };
}
