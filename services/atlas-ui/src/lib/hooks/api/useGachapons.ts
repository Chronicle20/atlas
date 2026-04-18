import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { gachaponsService } from '@/services/api/gachapons.service';
import { useTenant } from '@/context/tenant-context';
import type { GachaponData } from '@/types/models/gachapon';
import type { GachaponRewardData } from '@/types/models/gachapon-reward';
import type { QueryOptions } from '@/services/api/base.service';

export const gachaponKeys = {
  all: ['gachapons'] as const,
  lists: () => [...gachaponKeys.all, 'list'] as const,
  list: (options?: QueryOptions) => [...gachaponKeys.lists(), options] as const,
  details: () => [...gachaponKeys.all, 'detail'] as const,
  detail: (id: string) => [...gachaponKeys.details(), id] as const,
  prizePools: () => [...gachaponKeys.all, 'prize-pool'] as const,
  prizePool: (id: string) => [...gachaponKeys.prizePools(), id] as const,
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

export function useGachapon(id: string): UseQueryResult<GachaponData, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: gachaponKeys.detail(id),
    queryFn: () => gachaponsService.getGachaponById(id, activeTenant!),
    enabled: !!activeTenant && !!id,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

export function useGachaponPrizePool(gachaponId: string): UseQueryResult<GachaponRewardData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: gachaponKeys.prizePool(gachaponId),
    queryFn: () => gachaponsService.getPrizePool(gachaponId, activeTenant!),
    enabled: !!activeTenant && !!gachaponId,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}
