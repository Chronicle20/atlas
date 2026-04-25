import { useMemo } from "react";
import { useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ChevronDown } from "lucide-react";

import { useTenant } from "@/context/tenant-context";
import { npcsService } from "@/services/api/npcs.service";
import { useNpcData } from "@/lib/hooks/useNpcData";
import { useNpcSpawnMaps } from "@/lib/hooks/api/useNpcSpawnMaps";
import { useNpcQuests, type NpcQuestEntry } from "@/lib/hooks/api/useNpcQuests";
import { useNpcConversation } from "@/lib/hooks/api/useNpcConversation";
import type { NpcQuestRole } from "@/types/models/npc";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";
import { NpcHeader } from "@/components/features/npc/NpcHeader";
import { NpcSpawnMapWidget } from "@/components/features/npc/NpcSpawnMapWidget";
import { NpcQuestWidget } from "@/components/features/npc/NpcQuestWidget";
import { NpcShopCard } from "@/components/features/npc/NpcShopCard";
import { NpcConversationCard } from "@/components/features/npc/conversation/NpcConversationCard";
import { RecipesByNpcCard } from "@/components/features/npc/RecipesByNpcCard";

const ROLE_PRIORITY: Record<NpcQuestRole, number> = {
  initiator: 0,
  both: 1,
  completer: 2,
};

function sortQuestEntries(entries: NpcQuestEntry[]): NpcQuestEntry[] {
  return [...entries].sort((a, b) => {
    const rolePriority = ROLE_PRIORITY[a.role] - ROLE_PRIORITY[b.role];
    if (rolePriority !== 0) return rolePriority;
    const nameA = a.quest.attributes.name || "";
    const nameB = b.quest.attributes.name || "";
    const byName = nameA.localeCompare(nameB);
    if (byName !== 0) return byName;
    return parseInt(a.quest.id) - parseInt(b.quest.id);
  });
}

export function NpcDetailPage() {
  const { activeTenant } = useTenant();
  const params = useParams();
  const npcId = Number(params.id);

  const {
    name: npcName,
    iconUrl: npcIconUrl,
  } = useNpcData(npcId, {
    enabled: npcId > 0,
  });

  const npcQuery = useQuery({
    queryKey: ["npcs", "detail", activeTenant?.id ?? "no-tenant", npcId],
    queryFn: async () => {
      const npcData = await npcsService.getNPCById(npcId);
      return npcData ?? { id: npcId, hasShop: false, hasConversation: false };
    },
    enabled: !!activeTenant && npcId > 0,
  });

  const spawnMapsQuery = useNpcSpawnMaps(npcId);
  const questsQuery = useNpcQuests(npcId);
  const conversationQuery = useNpcConversation(npcId);

  const sortedQuests = useMemo(
    () => sortQuestEntries(questsQuery.data ?? []),
    [questsQuery.data],
  );

  const spawnMaps = spawnMapsQuery.data ?? [];
  const hasShop = npcQuery.data?.hasShop === true;
  const hasConversation = npcQuery.data?.hasConversation === true;
  const conversation = conversationQuery.data ?? null;

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16 overflow-y-auto">
      <NpcHeader npcId={npcId} name={npcName} iconUrl={npcIconUrl} />

      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">
            Spawn Locations{spawnMaps.length > 0 ? ` (${spawnMaps.length})` : ""}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {spawnMapsQuery.isLoading ? (
            <p className="text-sm text-muted-foreground">Loading spawn locations...</p>
          ) : spawnMapsQuery.error ? (
            <ErrorDisplay
              error={spawnMapsQuery.error}
              retry={() => spawnMapsQuery.refetch()}
            />
          ) : spawnMaps.length > 0 ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
              {spawnMaps.map(entry => (
                <NpcSpawnMapWidget
                  key={`${npcId}-${entry.mapId}`}
                  entry={entry}
                />
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              This NPC is not placed on any loaded map.
            </p>
          )}
        </CardContent>
      </Card>

      <NpcShopCard npcId={npcId} hasShop={hasShop} />

      <Card>
        <Collapsible defaultOpen={hasConversation} className="flex flex-col gap-6">
          <CardHeader>
            <CollapsibleTrigger className="group flex items-center gap-2 cursor-pointer text-left">
              <ChevronDown className="h-4 w-4 text-muted-foreground transition-transform group-data-[state=closed]:-rotate-90" />
              <CardTitle className="text-sm font-medium">
                Conversation
                {hasConversation && conversation
                  ? ` (${conversation.attributes.states.length})`
                  : ""}
              </CardTitle>
            </CollapsibleTrigger>
          </CardHeader>
          <CollapsibleContent>
            <CardContent>
              {hasConversation ? (
                conversationQuery.isLoading ? (
                  <div className="space-y-2">
                    <Skeleton className="h-4 w-full" />
                    <Skeleton className="h-4 w-11/12" />
                    <Skeleton className="h-4 w-5/6" />
                  </div>
                ) : conversationQuery.error ? (
                  <ErrorDisplay
                    error={conversationQuery.error}
                    retry={() => conversationQuery.refetch()}
                  />
                ) : conversation ? (
                  <NpcConversationCard conversation={conversation} />
                ) : (
                  <p className="text-sm text-muted-foreground">
                    Conversation data unavailable.
                  </p>
                )
              ) : (
                <p className="text-sm text-muted-foreground">
                  No conversation configured.
                </p>
              )}
            </CardContent>
          </CollapsibleContent>
        </Collapsible>
      </Card>

      <Card>
        <Collapsible defaultOpen={sortedQuests.length > 0} className="flex flex-col gap-6">
          <CardHeader>
            <CollapsibleTrigger className="group flex items-center gap-2 cursor-pointer text-left">
              <ChevronDown className="h-4 w-4 text-muted-foreground transition-transform group-data-[state=closed]:-rotate-90" />
              <CardTitle className="text-sm font-medium">
                Quests{sortedQuests.length > 0 ? ` (${sortedQuests.length})` : ""}
              </CardTitle>
            </CollapsibleTrigger>
          </CardHeader>
          <CollapsibleContent>
            <CardContent>
              {questsQuery.isLoading ? (
                <p className="text-sm text-muted-foreground">Loading quests...</p>
              ) : questsQuery.error ? (
                <ErrorDisplay
                  error={questsQuery.error}
                  retry={() => questsQuery.refetch()}
                />
              ) : sortedQuests.length > 0 ? (
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
                  {sortedQuests.map(entry => (
                    <NpcQuestWidget
                      key={entry.quest.id}
                      quest={entry.quest}
                      role={entry.role}
                    />
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">
                  This NPC does not participate in any quest.
                </p>
              )}
            </CardContent>
          </CollapsibleContent>
        </Collapsible>
      </Card>

      <RecipesByNpcCard npcId={npcId} />
    </div>
  );
}
