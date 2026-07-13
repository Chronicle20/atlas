/**
 * React Query hooks for the per-tenant marketplace (MTS) configuration.
 *
 * Mirrors the tenant-configuration hooks in useTenants.ts: a query for the
 * single config plus a mutation that PATCHes it with optimistic update +
 * cache invalidation.
 */

import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationResult,
  type UseQueryResult,
} from "@tanstack/react-query";
import {
  mtsConfigService,
  type MtsConfig,
  type MtsConfigAttributes,
} from "@/services/api/mts-config.service";
import type { ServiceOptions } from "@/lib/api/query-params";

export const mtsConfigKeys = {
  all: ["mts-config"] as const,
  details: () => [...mtsConfigKeys.all, "detail"] as const,
  detail: (tenantId: string) => [...mtsConfigKeys.details(), tenantId] as const,
};

/**
 * Fetch the single MTS configuration for a tenant.
 */
export function useMtsConfig(
  tenantId: string,
  options?: ServiceOptions,
): UseQueryResult<MtsConfig, Error> {
  return useQuery({
    queryKey: mtsConfigKeys.detail(tenantId),
    queryFn: () => mtsConfigService.getConfig(tenantId, options),
    enabled: !!tenantId,
    gcTime: 10 * 60 * 1000,
  });
}

/**
 * Update the MTS configuration for a tenant.
 */
export function useUpdateMtsConfig(): UseMutationResult<
  MtsConfig,
  Error,
  { tenantId: string; config: MtsConfig; updates: Partial<MtsConfigAttributes> }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ tenantId, config, updates }) =>
      mtsConfigService.updateConfig(tenantId, config, updates),
    onMutate: async ({ tenantId, config, updates }) => {
      await queryClient.cancelQueries({ queryKey: mtsConfigKeys.detail(tenantId) });
      const previousConfig = queryClient.getQueryData<MtsConfig>(mtsConfigKeys.detail(tenantId));
      const optimisticConfig: MtsConfig = {
        ...config,
        attributes: { ...config.attributes, ...updates },
      };
      queryClient.setQueryData(mtsConfigKeys.detail(tenantId), optimisticConfig);
      return { previousConfig };
    },
    onError: (error, variables, context) => {
      if (context?.previousConfig) {
        queryClient.setQueryData(mtsConfigKeys.detail(variables.tenantId), context.previousConfig);
      }
      console.error("Failed to update MTS configuration:", error);
    },
    onSettled: (_data, _error, variables) => {
      queryClient.invalidateQueries({ queryKey: mtsConfigKeys.detail(variables.tenantId) });
    },
  });
}
