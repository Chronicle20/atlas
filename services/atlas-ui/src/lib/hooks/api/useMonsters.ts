import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { monstersService } from '@/services/api/monsters.service';
import { useTenant } from '@/context/tenant-context';
import type { MonsterData } from '@/types/models/monster';
import type { ServiceOptions, QueryOptions } from '@/services/api/base.service';

export const monsterKeys = {
  all: ['monsters'] as const,
  lists: () => [...monsterKeys.all, 'list'] as const,
  list: (options?: QueryOptions) => [...monsterKeys.lists(), options] as const,
  details: () => [...monsterKeys.all, 'detail'] as const,
  detail: (id: string) => [...monsterKeys.details(), id] as const,
};

export function useMonsters(options?: QueryOptions): UseQueryResult<MonsterData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: monsterKeys.list(options),
    queryFn: () => monstersService.getAllMonsters(activeTenant!, { ...options, useCache: false }),
    enabled: !!activeTenant,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

export function useMonster(id: string, options?: ServiceOptions): UseQueryResult<MonsterData, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: monsterKeys.detail(id),
    queryFn: () => monstersService.getMonsterById(id, activeTenant!, { ...options, useCache: false }),
    enabled: !!id && !!activeTenant,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}
