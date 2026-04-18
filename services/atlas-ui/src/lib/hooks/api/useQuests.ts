/**
 * React Query hooks for quest definitions.
 */

import { useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
import { questsService, type QuestQueryOptions } from "@/services/api/quests.service";
import type { QuestDefinition } from "@/types/models/quest";
import type { Tenant } from "@/types/models/tenant";
import type { ServiceOptions } from "@/services/api/base.service";

export const questKeys = {
  all: ["quests"] as const,
  lists: () => [...questKeys.all, "list"] as const,
  list: (tenant: Tenant | null, options?: QuestQueryOptions) =>
    [...questKeys.lists(), tenant?.id ?? "no-tenant", options] as const,
  details: () => [...questKeys.all, "detail"] as const,
  detail: (tenant: Tenant | null, id: string) =>
    [...questKeys.details(), tenant?.id ?? "no-tenant", id] as const,
  categories: (tenant: Tenant | null) =>
    [...questKeys.all, "categories", tenant?.id ?? "no-tenant"] as const,
};

export function useQuests(
  tenant: Tenant | null,
  options?: QuestQueryOptions,
): UseQueryResult<QuestDefinition[], Error> {
  return useQuery({
    queryKey: questKeys.list(tenant, options),
    queryFn: () => questsService.getAllQuests(tenant!, options),
    enabled: !!tenant?.id,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

export function useQuestCategories(
  tenant: Tenant | null,
  options?: ServiceOptions,
): UseQueryResult<string[], Error> {
  return useQuery({
    queryKey: questKeys.categories(tenant),
    queryFn: () => questsService.getCategories(tenant!, options),
    enabled: !!tenant?.id,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

export function useQuest(
  tenant: Tenant | null,
  id: string,
  options?: ServiceOptions,
): UseQueryResult<QuestDefinition, Error> {
  return useQuery({
    queryKey: questKeys.detail(tenant, id),
    queryFn: () => questsService.getQuestById(tenant!, id, options),
    enabled: !!tenant?.id && !!id,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

export function useInvalidateQuests() {
  const queryClient = useQueryClient();
  return {
    invalidateAll: () => queryClient.invalidateQueries({ queryKey: questKeys.all }),
    invalidateLists: () => queryClient.invalidateQueries({ queryKey: questKeys.lists() }),
    invalidateDetail: (tenant: Tenant | null, id: string) =>
      queryClient.invalidateQueries({ queryKey: questKeys.detail(tenant, id) }),
  };
}
