import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import type { Tenant } from "@/services/api/tenants.service";
import {
  characterEffectiveStatsService,
  type EffectiveStats,
} from "@/services/api/characterEffectiveStats.service";

export const characterEffectiveStatsKeys = {
  all: ["character-effective-stats"] as const,
  detail: (
    tenantId: string | undefined,
    worldId: number,
    characterId: string,
  ) =>
    ["character-effective-stats", tenantId, worldId, characterId] as const,
};

/**
 * Wraps `GET /api/worlds/{worldId}/channels/0/characters/{characterId}/stats`
 * (atlas-effective-stats). Returns post-equip computed stats — HP/MP caps,
 * primary stats, attack/defense, etc. — so the attributes panel can show
 * `<base> +<bonus>` rather than just the raw character record.
 */
export function useCharacterEffectiveStats(
  tenant: Tenant | null | undefined,
  worldId: number,
  characterId: string,
): UseQueryResult<EffectiveStats, Error> {
  return useQuery({
    queryKey: characterEffectiveStatsKeys.detail(tenant?.id, worldId, characterId),
    queryFn: () =>
      characterEffectiveStatsService.getByCharacter(worldId, characterId),
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
