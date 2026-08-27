import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import {
  characterSlotsService,
  type CharacterSlots,
} from "@/services/api/character-slots.service";
import type { Tenant } from "@/types/models/tenant";
import type { ServiceOptions } from "@/lib/api/query-params";

export const characterSlotsKeys = {
  all: ["character-slots"] as const,
  details: () => [...characterSlotsKeys.all, "detail"] as const,
  detail: (tenant: Tenant | null, accountId: string, worldId: number) =>
    [
      ...characterSlotsKeys.details(),
      tenant?.id ?? "no-tenant",
      accountId,
      worldId,
    ] as const,
};

/**
 * Hook to fetch the character-slot count for a single (account, world)
 * pair. `GET accounts/{accountId}/worlds/{worldId}/character-slots`.
 */
export function useCharacterSlots(
  tenant: Tenant,
  accountId: string,
  worldId: number,
  options?: ServiceOptions,
): UseQueryResult<CharacterSlots, Error> {
  return useQuery({
    queryKey: characterSlotsKeys.detail(tenant, accountId, worldId),
    queryFn: () =>
      characterSlotsService.getCharacterSlots(accountId, worldId, {
        ...options,
        useCache: false,
      }),
    enabled: !!tenant?.id && !!accountId,
    gcTime: 5 * 60 * 1000,
  });
}
