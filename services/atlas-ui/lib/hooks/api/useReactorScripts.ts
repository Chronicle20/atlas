import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { reactorScriptsService, type ReactorScriptData } from '@/services/api/reactor-scripts.service';
import { useTenant } from '@/context/tenant-context';

export const reactorScriptKeys = {
  all: ['reactor-scripts'] as const,
  byReactor: (reactorId: string) => [...reactorScriptKeys.all, 'reactor', reactorId] as const,
};

export function useReactorScript(reactorId: string): UseQueryResult<ReactorScriptData | null, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: reactorScriptKeys.byReactor(reactorId),
    queryFn: () => reactorScriptsService.getScriptsByReactor(reactorId, activeTenant!, { useCache: false }),
    enabled: !!reactorId && !!activeTenant,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}
