import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import type { Tenant } from "@/services/api/tenants.service";
import {
  characterSkillsService,
  type CharacterSkill,
} from "@/services/api/characterSkills.service";

export const characterSkillsKeys = {
  all: ["character-skills"] as const,
  detail: (tenantId: string | undefined, characterId: string) =>
    ["character-skills", tenantId, characterId] as const,
};

export function useCharacterSkills(
  tenant: Tenant | null | undefined,
  characterId: string,
): UseQueryResult<CharacterSkill[], Error> {
  return useQuery({
    queryKey: characterSkillsKeys.detail(tenant?.id, characterId),
    queryFn: () => characterSkillsService.getByCharacterId(characterId),
    enabled: !!tenant?.id && !!characterId,
    staleTime: 60 * 1000,
    gcTime: 5 * 60 * 1000,
  });
}
