import { useMemo } from "react";
import { useParams, Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ShoppingBag, MessageCircle } from "lucide-react";

import { useTenant } from "@/context/tenant-context";
import { npcsService } from "@/services/api/npcs.service";
import { useNpcData } from "@/lib/hooks/useNpcData";
import { useNpcSpawnMaps } from "@/lib/hooks/api/useNpcSpawnMaps";
import { useNpcQuests, type NpcQuestEntry } from "@/lib/hooks/api/useNpcQuests";
import { useNpcConversation } from "@/lib/hooks/api/useNpcConversation";
import type { NpcQuestRole } from "@/types/models/npc";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";
import { NpcHeader } from "@/components/features/npc/NpcHeader";
import { NpcSpawnMapWidget } from "@/components/features/npc/NpcSpawnMapWidget";
import { NpcQuestWidget } from "@/components/features/npc/NpcQuestWidget";

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

function getEntryPreview(
  conversation: { attributes: { startState: string; states: Array<{ id: string; type: string; dialogue?: { text: string } }> } } | null,
): string {
  if (!conversation) return "";
  const startState = conversation.attributes.states.find(
    s => s.id === conversation.attributes.startState,
  );
  if (!startState) return "(no dialogue)";
  if (startState.type === "dialogue" && startState.dialogue?.text) {
    return startState.dialogue.text;
  }
  return "(no dialogue)";
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

  const shopQuery = useQuery({
    queryKey: ["npcs", "shop", activeTenant?.id ?? "no-tenant", npcId],
    queryFn: () => npcsService.getNPCShop(npcId),
    enabled: !!activeTenant && npcId > 0 && npcQuery.data?.hasShop === true,
  });

  const conversationQuery = useNpcConversation(npcId);

  const sortedQuests = useMemo(
    () => sortQuestEntries(questsQuery.data ?? []),
    [questsQuery.data],
  );

  const spawnMaps = spawnMapsQuery.data ?? [];
  const hasShop = npcQuery.data?.hasShop === true;
  const hasConversation = npcQuery.data?.hasConversation === true;

  const shop = shopQuery.data;
  const commodities = shop?.included ?? [];
  const rechargerEnabled = shop?.data.attributes.recharger === true;
  const tokenCommodityCount = commodities.filter(
    c => c.attributes.tokenPrice > 0 && c.attributes.tokenTemplateId > 0,
  ).length;

  const conversation = conversationQuery.data ?? null;
  const conversationEntryPreview = useMemo(
    () => getEntryPreview(conversation),
    [conversation],
  );

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

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">Shop</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {hasShop ? (
              shopQuery.isLoading ? (
                <div className="space-y-2">
                  <Skeleton className="h-4 w-40" />
                  <Skeleton className="h-4 w-40" />
                  <Skeleton className="h-4 w-40" />
                  <Skeleton className="h-9 w-32" />
                </div>
              ) : shopQuery.error ? (
                <ErrorDisplay
                  error={shopQuery.error}
                  retry={() => shopQuery.refetch()}
                />
              ) : (
                <>
                  <div className="space-y-1 text-sm">
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Recharger</span>
                      <span>{rechargerEnabled ? "Yes" : "No"}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Commodities</span>
                      <span>{commodities.length}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Tokens</span>
                      <span>
                        {tokenCommodityCount} of {commodities.length}
                      </span>
                    </div>
                  </div>
                  <div className="flex justify-end">
                    <Button asChild>
                      <Link to={`/npcs/${npcId}/shop`}>
                        <ShoppingBag className="h-4 w-4 mr-2" />
                        Edit Shop
                      </Link>
                    </Button>
                  </div>
                </>
              )
            ) : (
              <>
                <p className="text-sm text-muted-foreground">No shop configured.</p>
                <div className="flex justify-end">
                  <Button variant="outline" asChild>
                    <Link to={`/npcs/${npcId}/shop`}>
                      <ShoppingBag className="h-4 w-4 mr-2" />
                      Create Shop
                    </Link>
                  </Button>
                </div>
              </>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">Conversation</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {hasConversation ? (
              conversationQuery.isLoading ? (
                <div className="space-y-2">
                  <Skeleton className="h-4 w-40" />
                  <Skeleton className="h-4 w-40" />
                  <Skeleton className="h-4 w-40" />
                  <Skeleton className="h-9 w-32" />
                </div>
              ) : conversationQuery.error ? (
                <ErrorDisplay
                  error={conversationQuery.error}
                  retry={() => conversationQuery.refetch()}
                />
              ) : conversation ? (
                <>
                  <div className="space-y-1 text-sm">
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">States</span>
                      <span>{conversation.attributes.states.length}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Start State</span>
                      <span className="truncate max-w-[60%] text-right">
                        {conversation.attributes.startState}
                      </span>
                    </div>
                    <div className="flex justify-between gap-2">
                      <span className="text-muted-foreground shrink-0">Entry Preview</span>
                      <span className="truncate max-w-[60%] text-right">
                        {conversationEntryPreview}
                      </span>
                    </div>
                  </div>
                  <div className="flex justify-end">
                    <Button asChild>
                      <Link to={`/npcs/${npcId}/conversations`}>
                        <MessageCircle className="h-4 w-4 mr-2" />
                        Edit Conversation
                      </Link>
                    </Button>
                  </div>
                </>
              ) : (
                <p className="text-sm text-muted-foreground">
                  Conversation data unavailable.
                </p>
              )
            ) : (
              <>
                <p className="text-sm text-muted-foreground">
                  No conversation configured.
                </p>
                <div className="flex justify-end">
                  <Button variant="outline" asChild>
                    <Link to={`/npcs/${npcId}/conversations`}>
                      <MessageCircle className="h-4 w-4 mr-2" />
                      Create Conversation
                    </Link>
                  </Button>
                </div>
              </>
            )}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">
            Quests{sortedQuests.length > 0 ? ` (${sortedQuests.length})` : ""}
          </CardTitle>
        </CardHeader>
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
      </Card>
    </div>
  );
}
