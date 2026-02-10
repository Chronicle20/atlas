import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { portalScriptsService, type PortalScriptData } from '@/services/api/portal-scripts.service';
import { useTenant } from '@/context/tenant-context';

export const portalScriptKeys = {
  all: ['portal-scripts'] as const,
  byPortal: (portalId: string) => [...portalScriptKeys.all, 'portal', portalId] as const,
};

export function usePortalScript(portalId: string): UseQueryResult<PortalScriptData | null, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: portalScriptKeys.byPortal(portalId),
    queryFn: () => portalScriptsService.getScriptsByPortal(portalId, activeTenant!, { useCache: false }),
    enabled: !!portalId && !!activeTenant,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}
