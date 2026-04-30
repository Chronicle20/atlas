import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { factoryService, type NameValidityResponse } from "@/services/api/factory.service";
import { useDebounce } from "@/lib/utils/debounce";
import type { Tenant } from "@/types/models/tenant";

export const nameValidityKeys = {
  all: ["name-validity"] as const,
  query: (tenantId: string | undefined, worldId: number, name: string) =>
    [...nameValidityKeys.all, tenantId, worldId, name] as const,
};

export interface UseNameValidityOptions {
  enabled?: boolean;
  debounceMs?: number;
}

export function useNameValidity(
  tenant: Tenant,
  name: string,
  worldId: number,
  options: UseNameValidityOptions = {},
): UseQueryResult<NameValidityResponse, Error> {
  const debounced = useDebounce(name, options.debounceMs ?? 300);
  return useQuery({
    queryKey: nameValidityKeys.query(tenant?.id, worldId, debounced),
    queryFn: () => factoryService.checkNameValidity(tenant, debounced, worldId),
    enabled: !!tenant?.id && (options.enabled ?? true) && debounced.length >= 3,
    staleTime: 0,
  });
}
