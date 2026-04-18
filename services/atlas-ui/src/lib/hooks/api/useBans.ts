/**
 * React Query hooks for ban management.
 */

import { useMutation, useQuery, useQueryClient, type UseMutationResult, type UseQueryResult } from "@tanstack/react-query";
import { bansService, type BanQueryOptions } from "@/services/api/bans.service";
import type { Ban, BanType, CreateBanRequest } from "@/types/models/ban";
import type { Tenant } from "@/types/models/tenant";
import type { ServiceOptions } from "@/lib/api/query-params";

export const banKeys = {
  all: ["bans"] as const,
  lists: () => [...banKeys.all, "list"] as const,
  list: (tenant: Tenant | null, options?: BanQueryOptions) =>
    [...banKeys.lists(), tenant?.id ?? "no-tenant", options] as const,
  details: () => [...banKeys.all, "detail"] as const,
  detail: (tenant: Tenant | null, id: string) =>
    [...banKeys.details(), tenant?.id ?? "no-tenant", id] as const,
};

export function useBans(
  tenant: Tenant | null,
  options?: BanQueryOptions,
): UseQueryResult<Ban[], Error> {
  return useQuery({
    queryKey: banKeys.list(tenant, options),
    queryFn: () => bansService.getAllBans(tenant!, options),
    enabled: !!tenant?.id,
    staleTime: 60 * 1000,
    gcTime: 5 * 60 * 1000,
  });
}

export function useBan(
  tenant: Tenant | null,
  id: string,
  options?: ServiceOptions,
): UseQueryResult<Ban, Error> {
  return useQuery({
    queryKey: banKeys.detail(tenant, id),
    queryFn: () => bansService.getBanById(tenant!, id, options),
    enabled: !!tenant?.id && !!id,
    staleTime: 60 * 1000,
    gcTime: 5 * 60 * 1000,
  });
}

export function useBansByType(
  tenant: Tenant | null,
  type: BanType,
  options?: ServiceOptions,
): UseQueryResult<Ban[], Error> {
  return useQuery({
    queryKey: [...banKeys.lists(), tenant?.id ?? "no-tenant", "type", type],
    queryFn: () => bansService.getBansByType(tenant!, type, options),
    enabled: !!tenant?.id,
    staleTime: 60 * 1000,
    gcTime: 5 * 60 * 1000,
  });
}

export function useCreateBan(): UseMutationResult<Ban, Error, { tenant: Tenant; data: CreateBanRequest }> {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ tenant, data }) => bansService.createBan(tenant, data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: banKeys.all }),
  });
}

export function useDeleteBan(): UseMutationResult<void, Error, { tenant: Tenant; id: string }> {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ tenant, id }) => bansService.deleteBan(tenant, id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: banKeys.all }),
  });
}

export function useExpireBan(): UseMutationResult<void, Error, { tenant: Tenant; id: string }> {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ tenant, id }) => bansService.expireBan(tenant, id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: banKeys.all }),
  });
}

export function useInvalidateBans() {
  const queryClient = useQueryClient();
  return {
    invalidateAll: () => queryClient.invalidateQueries({ queryKey: banKeys.all }),
    invalidateLists: () => queryClient.invalidateQueries({ queryKey: banKeys.lists() }),
  };
}
