import { useMutation, useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
import { incubatorRewardsService, type IncubatorReward, type IncubatorRewardAttributes } from "@/services/api/incubator-rewards.service";

export const incubatorRewardsKeys = {
  all: ["incubator-rewards"] as const,
  lists: () => [...incubatorRewardsKeys.all, "list"] as const,
  list: (tenantId: string) => [...incubatorRewardsKeys.lists(), tenantId] as const,
};

export function useIncubatorRewards(tenantId: string): UseQueryResult<IncubatorReward[], Error> {
  return useQuery({
    queryKey: incubatorRewardsKeys.list(tenantId),
    queryFn: () => incubatorRewardsService.list(tenantId),
    enabled: !!tenantId,
    gcTime: 10 * 60 * 1000,
  });
}

export function useCreateIncubatorReward() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ tenantId, attributes }: { tenantId: string; attributes: IncubatorRewardAttributes }) =>
      incubatorRewardsService.create(tenantId, attributes),
    onSettled: (_d, _e, vars) => qc.invalidateQueries({ queryKey: incubatorRewardsKeys.list(vars.tenantId) }),
  });
}

export function useUpdateIncubatorReward() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ tenantId, id, attributes }: { tenantId: string; id: string; attributes: IncubatorRewardAttributes }) =>
      incubatorRewardsService.update(tenantId, id, attributes),
    onSettled: (_d, _e, vars) => qc.invalidateQueries({ queryKey: incubatorRewardsKeys.list(vars.tenantId) }),
  });
}

export function useDeleteIncubatorReward() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ tenantId, id }: { tenantId: string; id: string }) => incubatorRewardsService.remove(tenantId, id),
    onSettled: (_d, _e, vars) => qc.invalidateQueries({ queryKey: incubatorRewardsKeys.list(vars.tenantId) }),
  });
}

export function useSeedIncubatorRewards() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ tenantId }: { tenantId: string }) => incubatorRewardsService.seed(tenantId),
    onSettled: (_d, _e, vars) => qc.invalidateQueries({ queryKey: incubatorRewardsKeys.list(vars.tenantId) }),
  });
}
