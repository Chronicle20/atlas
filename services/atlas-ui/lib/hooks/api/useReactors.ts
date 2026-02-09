import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { reactorsService } from '@/services/api/reactors.service';
import { useTenant } from '@/context/tenant-context';
import type { ReactorData } from '@/types/models/reactor';
import type { ServiceOptions, QueryOptions } from '@/services/api/base.service';

export const reactorKeys = {
  all: ['reactors'] as const,
  lists: () => [...reactorKeys.all, 'list'] as const,
  list: (options?: QueryOptions) => [...reactorKeys.lists(), options] as const,
  details: () => [...reactorKeys.all, 'detail'] as const,
  detail: (id: string) => [...reactorKeys.details(), id] as const,
};

export function useReactors(options?: QueryOptions): UseQueryResult<ReactorData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: reactorKeys.list(options),
    queryFn: () => reactorsService.getAllReactors(activeTenant!, options),
    enabled: !!activeTenant,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

export function useReactor(id: string, options?: ServiceOptions): UseQueryResult<ReactorData, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: reactorKeys.detail(id),
    queryFn: () => reactorsService.getReactorById(id, activeTenant!, options),
    enabled: !!id && !!activeTenant,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}
