import { useMutation, useQuery, useQueryClient, type UseMutationResult, type UseQueryResult } from '@tanstack/react-query';
import { rewardPoolsService } from '@/services/api/reward-pools.service';
import { useTenant } from '@/context/tenant-context';
import type { RewardPoolData, RewardPoolAttributes } from '@/types/models/reward-pool';
import type { RewardPoolItemData, RewardPoolItemAttributes } from '@/types/models/reward-pool-item';
import type { GlobalRewardItemData, GlobalRewardItemAttributes } from '@/types/models/global-reward-item';

export const rewardPoolKeys = {
  all: ['reward-pools'] as const,
  lists: () => [...rewardPoolKeys.all, 'list'] as const,
  list: () => [...rewardPoolKeys.lists()] as const,
  details: () => [...rewardPoolKeys.all, 'detail'] as const,
  detail: (id: string) => [...rewardPoolKeys.details(), id] as const,
  items: (poolId: string) => [...rewardPoolKeys.all, 'items', poolId] as const,
  globalItems: () => [...rewardPoolKeys.all, 'global-items'] as const,
};

export function useRewardPools(): UseQueryResult<RewardPoolData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: rewardPoolKeys.list(),
    queryFn: () => rewardPoolsService.getAllPools(),
    enabled: !!activeTenant,
    gcTime: 10 * 60 * 1000,
  });
}

export function useRewardPool(id: string): UseQueryResult<RewardPoolData, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: rewardPoolKeys.detail(id),
    queryFn: () => rewardPoolsService.getPoolById(id),
    enabled: !!activeTenant && !!id,
    gcTime: 10 * 60 * 1000,
  });
}

export function useRewardPoolItems(poolId: string): UseQueryResult<RewardPoolItemData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: rewardPoolKeys.items(poolId),
    queryFn: () => rewardPoolsService.getItems(poolId),
    enabled: !!activeTenant && !!poolId,
    gcTime: 10 * 60 * 1000,
  });
}

export function useGlobalRewardItems(): UseQueryResult<GlobalRewardItemData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: rewardPoolKeys.globalItems(),
    queryFn: () => rewardPoolsService.getGlobalItems(),
    enabled: !!activeTenant,
    gcTime: 10 * 60 * 1000,
  });
}

export function useCreateRewardPool(): UseMutationResult<void, Error, { id?: string; attributes: RewardPoolAttributes }> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, attributes }) => rewardPoolsService.createPool(id, attributes),
    onSettled: () => qc.invalidateQueries({ queryKey: rewardPoolKeys.lists() }),
  });
}

export function useUpdateRewardPool(): UseMutationResult<void, Error, { id: string; attributes: RewardPoolAttributes }> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, attributes }) => rewardPoolsService.updatePool(id, attributes),
    onSettled: (_d, _e, { id }) => {
      qc.invalidateQueries({ queryKey: rewardPoolKeys.lists() });
      qc.invalidateQueries({ queryKey: rewardPoolKeys.detail(id) });
    },
  });
}

export function useDeleteRewardPool(): UseMutationResult<void, Error, { id: string }> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id }) => rewardPoolsService.removePool(id),
    onSettled: () => qc.invalidateQueries({ queryKey: rewardPoolKeys.lists() }),
  });
}

export function useCreatePoolItem(): UseMutationResult<void, Error, { poolId: string; attributes: Omit<RewardPoolItemAttributes, 'gachaponId'> }> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ poolId, attributes }) => rewardPoolsService.createItem(poolId, attributes),
    onSettled: (_d, _e, { poolId }) => qc.invalidateQueries({ queryKey: rewardPoolKeys.items(poolId) }),
  });
}

export function useUpdatePoolItem(): UseMutationResult<void, Error, { poolId: string; itemRecordId: string; attributes: Omit<RewardPoolItemAttributes, 'gachaponId'> }> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ poolId, itemRecordId, attributes }) => rewardPoolsService.updateItem(poolId, itemRecordId, attributes),
    onSettled: (_d, _e, { poolId }) => qc.invalidateQueries({ queryKey: rewardPoolKeys.items(poolId) }),
  });
}

export function useDeletePoolItem(): UseMutationResult<void, Error, { poolId: string; itemRecordId: string }> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ poolId, itemRecordId }) => rewardPoolsService.removeItem(poolId, itemRecordId),
    onSettled: (_d, _e, { poolId }) => qc.invalidateQueries({ queryKey: rewardPoolKeys.items(poolId) }),
  });
}

export function useCreateGlobalItem(): UseMutationResult<void, Error, { attributes: GlobalRewardItemAttributes }> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ attributes }) => rewardPoolsService.createGlobalItem(attributes),
    onSettled: () => qc.invalidateQueries({ queryKey: rewardPoolKeys.globalItems() }),
  });
}

export function useUpdateGlobalItem(): UseMutationResult<void, Error, { itemRecordId: string; attributes: GlobalRewardItemAttributes }> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ itemRecordId, attributes }) => rewardPoolsService.updateGlobalItem(itemRecordId, attributes),
    onSettled: () => qc.invalidateQueries({ queryKey: rewardPoolKeys.globalItems() }),
  });
}

export function useDeleteGlobalItem(): UseMutationResult<void, Error, { itemRecordId: string }> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ itemRecordId }) => rewardPoolsService.removeGlobalItem(itemRecordId),
    onSettled: () => qc.invalidateQueries({ queryKey: rewardPoolKeys.globalItems() }),
  });
}
