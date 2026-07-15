import { useQuery, keepPreviousData, type UseQueryResult } from '@tanstack/react-query';
import { gachaponsService } from '@/services/api/gachapons.service';
import type { PagedResult } from '@/services/api/pagination';
import { useTenant } from '@/context/tenant-context';
import type { GachaponData } from '@/types/models/gachapon';
import type { GachaponRewardData } from '@/types/models/gachapon-reward';
import type { QueryOptions } from '@/lib/api/query-params';

export const gachaponKeys = {
  all: ['gachapons'] as const,
  lists: () => [...gachaponKeys.all, 'list'] as const,
  list: (options?: QueryOptions) => [...gachaponKeys.lists(), options] as const,
  pagedList: (page: number, size: number) => [...gachaponKeys.lists(), 'page', page, size] as const,
  details: () => [...gachaponKeys.all, 'detail'] as const,
  detail: (id: string) => [...gachaponKeys.details(), id] as const,
  prizePools: () => [...gachaponKeys.all, 'prize-pool'] as const,
  prizePool: (id: string) => [...gachaponKeys.prizePools(), id] as const,
};

export function useGachapons(options?: QueryOptions): UseQueryResult<GachaponData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: gachaponKeys.list(options),
    queryFn: () => gachaponsService.getAllGachapons({ ...options, useCache: false }),
    enabled: !!activeTenant,
    gcTime: 10 * 60 * 1000,
  });
}

/**
 * Hook to fetch a single page of gachapons (task-117). Backs the Gachapons
 * list view, which pages server-side; keeps the previous page's data on
 * screen while the next page loads.
 */
export function useGachaponsPage(
  page: { number: number; size: number },
): UseQueryResult<PagedResult<GachaponData>, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: gachaponKeys.pagedList(page.number, page.size),
    queryFn: () => gachaponsService.getPage(page, { useCache: false }),
    enabled: !!activeTenant,
    placeholderData: keepPreviousData,
    gcTime: 10 * 60 * 1000,
  });
}

export function useGachapon(id: string): UseQueryResult<GachaponData, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: gachaponKeys.detail(id),
    queryFn: () => gachaponsService.getGachaponById(id),
    enabled: !!activeTenant && !!id,
    gcTime: 10 * 60 * 1000,
  });
}

export function useGachaponPrizePool(gachaponId: string): UseQueryResult<GachaponRewardData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: gachaponKeys.prizePool(gachaponId),
    queryFn: () => gachaponsService.getPrizePool(gachaponId),
    enabled: !!activeTenant && !!gachaponId,
    gcTime: 10 * 60 * 1000,
  });
}
