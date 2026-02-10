import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { gachaponsService } from '@/services/api/gachapons.service';
import { useTenant } from '@/context/tenant-context';
import type { GachaponData } from '@/types/models/gachapon';
import type { QueryOptions } from '@/services/api/base.service';

export const gachaponKeys = {
  all: ['gachapons'] as const,
  lists: () => [...gachaponKeys.all, 'list'] as const,
  list: (options?: QueryOptions) => [...gachaponKeys.lists(), options] as const,
};

export function useGachapons(options?: QueryOptions): UseQueryResult<GachaponData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: gachaponKeys.list(options),
    queryFn: () => gachaponsService.getAllGachapons(activeTenant!, { ...options, useCache: false }),
    enabled: !!activeTenant,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}
