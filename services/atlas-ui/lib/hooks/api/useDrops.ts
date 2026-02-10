import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { dropsService } from '@/services/api/drops.service';
import { useTenant } from '@/context/tenant-context';
import type { DropData, ReactorDropData } from '@/types/models/drop';

export const dropKeys = {
  all: ['drops'] as const,
  monster: (monsterId: string) => [...dropKeys.all, 'monster', monsterId] as const,
  reactor: (reactorId: string) => [...dropKeys.all, 'reactor', reactorId] as const,
};

export function useMonsterDrops(monsterId: string): UseQueryResult<DropData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: dropKeys.monster(monsterId),
    queryFn: () => dropsService.getMonsterDrops(monsterId, activeTenant!),
    enabled: !!monsterId && !!activeTenant,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

export function useReactorDrops(reactorId: string): UseQueryResult<ReactorDropData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: dropKeys.reactor(reactorId),
    queryFn: () => dropsService.getReactorDrops(reactorId, activeTenant!),
    enabled: !!reactorId && !!activeTenant,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}
