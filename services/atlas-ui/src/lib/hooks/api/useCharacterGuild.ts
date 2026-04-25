/**
 * Hook to fetch the guild a character belongs to (if any).
 *
 * Uses the atlas-guilds backend filter `filter[members.id]={characterId}`
 * which returns an array (possibly empty). The hook flattens that to a
 * single `Guild | null` for ergonomic consumption by detail pages.
 */

import { useQuery } from "@tanstack/react-query";
import type { Tenant } from "@/services/api/tenants.service";
import { guildsService, type Guild } from "@/services/api/guilds.service";

export const characterGuildKeys = {
  all: ["character-guild"] as const,
  detail: (tenantId: string | undefined, characterId: string) =>
    ["character-guild", tenantId, characterId] as const,
};

export interface UseCharacterGuildResult {
  guild: Guild | null;
  isLoading: boolean;
  error: Error | null;
}

export function useCharacterGuild(
  tenant: Tenant | null | undefined,
  characterId: string
): UseCharacterGuildResult {
  const query = useQuery({
    queryKey: characterGuildKeys.detail(tenant?.id, characterId),
    queryFn: () => guildsService.getByMemberId(characterId),
    enabled: !!tenant?.id && !!characterId,
    staleTime: 60 * 1000,
    gcTime: 5 * 60 * 1000,
  });

  const guild = query.data?.[0] ?? null;

  return {
    guild: query.isError ? null : guild,
    isLoading: query.isLoading,
    error: query.error ?? null,
  };
}
