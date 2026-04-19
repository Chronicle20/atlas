import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { npcsService } from "@/services/api/npcs.service";
import { useTenant } from "@/context/tenant-context";
import type { NpcQuestRole } from "@/types/models/npc";
import type { QuestDefinition } from "@/types/models/quest";

export interface NpcQuestEntry {
  quest: QuestDefinition;
  role: NpcQuestRole;
}

export const npcQuestsKeys = {
  all: ["data", "npcs", "quests"] as const,
  byNpc: (tenantId: string | undefined, npcId: number) =>
    ["data", "npcs", "quests", tenantId ?? "no-tenant", npcId] as const,
};

export function deriveNpcQuestRole(
  quest: QuestDefinition,
  npcId: number,
): NpcQuestRole {
  const startMatch = quest.attributes.startRequirements?.npcId === npcId;
  const endMatch = quest.attributes.endActions?.npcId === npcId;

  if (startMatch && endMatch) return "both";
  if (startMatch) return "initiator";
  if (endMatch) return "completer";

  // Outlier: NPC only appears in startActions or endRequirements.
  if (quest.attributes.startActions?.npcId === npcId) return "initiator";
  return "completer";
}

export function useNpcQuests(
  npcId: number,
): UseQueryResult<NpcQuestEntry[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: npcQuestsKeys.byNpc(activeTenant?.id, npcId),
    queryFn: async (): Promise<NpcQuestEntry[]> => {
      const quests = await npcsService.getNpcQuests(npcId);
      return quests.map(quest => ({
        quest,
        role: deriveNpcQuestRole(quest, npcId),
      }));
    },
    enabled: !!activeTenant && npcId > 0,
    staleTime: 10 * 60 * 1000,
    gcTime: 15 * 60 * 1000,
  });
}
