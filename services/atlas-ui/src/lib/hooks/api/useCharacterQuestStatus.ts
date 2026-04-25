import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import type { Tenant } from "@/services/api/tenants.service";
import { questStatusService } from "@/services/api/quest-status.service";
import type { CharacterQuestStatus } from "@/services/api/quest-status.service";

export const characterQuestStatusKeys = {
  all: ["character-quest-status"] as const,
  detail: (tenantId: string | undefined, characterId: string) =>
    ["character-quest-status", tenantId, characterId] as const,
};

export interface CharacterQuestStatusBundle {
  started: CharacterQuestStatus[];
  completed: CharacterQuestStatus[];
}

export function useCharacterQuestStatus(
  tenant: Tenant | null | undefined,
  characterId: string
): UseQueryResult<CharacterQuestStatusBundle, Error> {
  return useQuery({
    queryKey: characterQuestStatusKeys.detail(tenant?.id, characterId),
    queryFn: async () => {
      const [started, completed] = await Promise.all([
        questStatusService.getStartedQuests(characterId),
        questStatusService.getCompletedQuests(characterId),
      ]);
      return { started, completed };
    },
    enabled: !!tenant?.id && !!characterId,
    staleTime: 60 * 1000,
    gcTime: 5 * 60 * 1000,
  });
}
