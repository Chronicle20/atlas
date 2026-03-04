import { useMutation, useQuery, useQueryClient, type UseMutationResult, type UseQueryResult } from '@tanstack/react-query';
import { walletService, type Wallet } from '@/services/api/wallet.service';
import type { Tenant } from '@/types/models/tenant';
import type { ServiceOptions } from '@/services/api/base.service';

export const walletKeys = {
  all: ['wallets'] as const,
  details: () => [...walletKeys.all, 'detail'] as const,
  detail: (tenant: Tenant | null, accountId: string) => [...walletKeys.details(), tenant?.id || 'no-tenant', accountId] as const,
};

export function useWallet(
  tenant: Tenant,
  accountId: string,
  options?: ServiceOptions
): UseQueryResult<Wallet, Error> {
  return useQuery({
    queryKey: walletKeys.detail(tenant, accountId),
    queryFn: () => walletService.getWallet(tenant, accountId, { ...options, useCache: false }),
    enabled: !!tenant?.id && !!accountId,
    staleTime: 2 * 60 * 1000,
    gcTime: 5 * 60 * 1000,
  });
}

export function useUpdateWallet(): UseMutationResult<
  Wallet,
  Error,
  { tenant: Tenant; accountId: string; credit: number; points: number; prepaid: number }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ tenant, accountId, credit, points, prepaid }) =>
      walletService.updateWallet(tenant, accountId, credit, points, prepaid),
    onMutate: async ({ tenant, accountId, credit, points, prepaid }) => {
      await queryClient.cancelQueries({ queryKey: walletKeys.detail(tenant, accountId) });

      const previousWallet = queryClient.getQueryData<Wallet>(walletKeys.detail(tenant, accountId));

      if (previousWallet) {
        const optimisticWallet: Wallet = {
          ...previousWallet,
          attributes: {
            ...previousWallet.attributes,
            credit,
            points,
            prepaid,
          },
        };
        queryClient.setQueryData(walletKeys.detail(tenant, accountId), optimisticWallet);
      }

      return { previousWallet };
    },
    onError: (error, { tenant, accountId }, context) => {
      if (context?.previousWallet) {
        queryClient.setQueryData(walletKeys.detail(tenant, accountId), context.previousWallet);
      }
    },
    onSettled: (data, error, { tenant, accountId }) => {
      queryClient.invalidateQueries({ queryKey: walletKeys.detail(tenant, accountId) });
    },
  });
}
