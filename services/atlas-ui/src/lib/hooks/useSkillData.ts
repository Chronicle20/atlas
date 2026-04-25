/**
 * React Query hook for Skill data fetching with caching
 * Provides name and icon URL data for skills using atlas-data API and atlas-assets.
 *
 * This hook is now a thin selector over `useSkillDefinition` so the underlying
 * React Query cache is shared between callers that need just name/icon and
 * callers that need the full skill definition.
 */

import { useCallback, useMemo } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useTenant } from '@/context/tenant-context';
import { getAssetIconUrl } from '@/lib/utils/asset-url';
import { useSkillDefinition, skillDefinitionKeys } from './api/useSkillDefinition';

export interface UseSkillDataOptions {
  enabled?: boolean;
}

const DEFAULT_OPTIONS: Required<UseSkillDataOptions> = {
  enabled: true,
};

/**
 * Hook for fetching single skill data (name and icon).
 *
 * Returns the same shape it always has so existing callers
 * (`EntityName`, `EntityWidget`) keep working unchanged.
 */
export function useSkillData(
  skillId: number,
  hookOptions: UseSkillDataOptions = {}
) {
  const options = useMemo(() => ({ ...DEFAULT_OPTIONS, ...hookOptions }), [hookOptions]);
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();

  const query = useSkillDefinition(options.enabled ? activeTenant : null, skillId);

  const invalidate = useCallback(() => {
    queryClient.invalidateQueries({
      queryKey: skillDefinitionKeys.detail(activeTenant?.id, skillId),
    });
  }, [queryClient, activeTenant?.id, skillId]);

  const deterministicIconUrl = activeTenant && skillId > 0
    ? getAssetIconUrl(
        activeTenant.id,
        activeTenant.attributes.region,
        activeTenant.attributes.majorVersion,
        activeTenant.attributes.minorVersion,
        'skill',
        skillId,
      )
    : undefined;

  const skillData = query.data
    ? {
        id: query.data.id,
        name: query.data.name,
        iconUrl: query.data.iconUrl ?? deterministicIconUrl ?? '',
        cached: false,
      }
    : undefined;

  return {
    ...query,
    skillData,
    name: query.data?.name,
    iconUrl: query.data?.iconUrl ?? deterministicIconUrl,
    hasError: query.isError,
    errorMessage: query.error?.message,
    cached: false,
    invalidate,
  };
}
