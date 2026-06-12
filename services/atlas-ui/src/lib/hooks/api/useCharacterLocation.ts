import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import type { Tenant } from "@/services/api/tenants.service";
import { locationsService } from "@/services/api/locations.service";
import type { CharacterLocation } from "@/types/models/location";

export const characterLocationKeys = {
  all: ["character-location"] as const,
  detail: (tenantId: string | undefined, characterId: string) =>
    ["character-location", tenantId, characterId] as const,
};

/**
 * Wraps `GET /api/characters/{characterId}/location` (atlas-maps). Returns the
 * character's current world/channel/map/instance location. The query is keyed
 * by the active tenant id so a tenant switch can't surface another tenant's
 * cached location; the API client carries tenant headers via its singleton.
 */
export function useCharacterLocation(
  tenant: Tenant | null | undefined,
  characterId: string,
): UseQueryResult<CharacterLocation, Error> {
  return useQuery({
    queryKey: characterLocationKeys.detail(tenant?.id, characterId),
    queryFn: () => locationsService.getByCharacterId(characterId),
    enabled: !!tenant?.id && !!characterId,
    staleTime: 60 * 1000,
    gcTime: 5 * 60 * 1000,
    retry: (failureCount, error) => {
      const msg = error?.message?.toLowerCase() ?? "";
      if (msg.includes("404") || msg.includes("not found")) return false;
      return failureCount < 3;
    },
  });
}
