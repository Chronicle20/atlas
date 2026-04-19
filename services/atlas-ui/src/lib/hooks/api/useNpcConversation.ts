import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { conversationsService } from "@/services/api/conversations.service";
import { useTenant } from "@/context/tenant-context";
import type { Conversation } from "@/types/models/conversation";

export const npcConversationKeys = {
  all: ["npcs", "conversation"] as const,
  byNpc: (tenantId: string | undefined, npcId: number) =>
    ["npcs", "conversation", tenantId ?? "no-tenant", npcId] as const,
};

export function useNpcConversation(
  npcId: number,
): UseQueryResult<Conversation | null, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: npcConversationKeys.byNpc(activeTenant?.id, npcId),
    queryFn: () => conversationsService.getByNpcId(npcId),
    enabled: !!activeTenant && npcId > 0,
    staleTime: 10 * 60 * 1000,
    gcTime: 15 * 60 * 1000,
  });
}
