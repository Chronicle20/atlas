import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { questConversationsService } from "@/services/api/quest-conversations.service";
import { useTenant } from "@/context/tenant-context";
import type { QuestConversation } from "@/types/models/conversation";

export const questConversationKeys = {
  all: ["quests", "conversation"] as const,
  byQuest: (tenantId: string | undefined, questId: number) =>
    ["quests", "conversation", tenantId ?? "no-tenant", questId] as const,
};

export function useQuestConversation(
  questId: number,
): UseQueryResult<QuestConversation | null, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: questConversationKeys.byQuest(activeTenant?.id, questId),
    queryFn: () => questConversationsService.getByQuestId(questId),
    enabled: !!activeTenant && questId > 0,
    staleTime: 10 * 60 * 1000,
    gcTime: 15 * 60 * 1000,
    retry: (failureCount, error) => {
      if (
        error &&
        typeof error === "object" &&
        "statusCode" in error &&
        (error as { statusCode: number }).statusCode === 404
      ) {
        return false;
      }
      return failureCount < 2;
    },
  });
}
