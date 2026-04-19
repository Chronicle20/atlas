import { useMemo } from "react";
import { useParams, Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Pencil, Plus } from "lucide-react";

import { useTenant } from "@/context/tenant-context";
import { npcsService } from "@/services/api/npcs.service";
import { useNpcData } from "@/lib/hooks/useNpcData";
import { useNpcSpawnMaps } from "@/lib/hooks/api/useNpcSpawnMaps";
import { useNpcQuests, type NpcQuestEntry } from "@/lib/hooks/api/useNpcQuests";
import { useNpcConversation } from "@/lib/hooks/api/useNpcConversation";
import { useItemBatchData } from "@/lib/hooks/useItemData";
import type { NpcQuestRole } from "@/types/models/npc";

import {
  Card,
  CardAction,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";
import { NpcHeader } from "@/components/features/npc/NpcHeader";
import { NpcSpawnMapWidget } from "@/components/features/npc/NpcSpawnMapWidget";
import { NpcQuestWidget } from "@/components/features/npc/NpcQuestWidget";
import { NpcShopCommodityWidget } from "@/components/features/npc/NpcShopCommodityWidget";
import { NpcConversationTreePreview } from "@/components/features/npc/NpcConversationTreePreview";

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
  const commodities = useMemo(() => shop?.included ?? [], [shop]);
  const commodityTemplateIds = useMemo(
    () => commodities.map(c => c.attributes.templateId),
    [commodities],
  );
  const itemBatch = useItemBatchData(commodityTemplateIds);
  const itemDataById = useMemo(() => {
    const m = new Map<number, { name?: string | undefined; iconUrl?: string | undefined }>();
    for (const entry of itemBatch.data) {
      m.set(entry.id, { name: entry.name, iconUrl: entry.iconUrl });
    }
    return m;
  }, [itemBatch.data]);

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

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">
              Shop{hasShop && commodities.length > 0 ? ` (${commodities.length})` : ""}
            </CardTitle>
            {hasShop ? (
              <CardAction>
                <Button
                  variant="ghost"
                  size="icon"
                  asChild
                  title="Edit Shop"
                  aria-label="Edit Shop"
                >
                  <Link to={`/npcs/${npcId}/shop`}>
                    <Pencil className="h-4 w-4" />
                  </Link>
                </Button>
              </CardAction>
            ) : (
              <CardAction>
                <Button
                  variant="ghost"
                  size="icon"
                  asChild
                  title="Create Shop"
                  aria-label="Create Shop"
                >
                  <Link to={`/npcs/${npcId}/shop`}>
                    <Plus className="h-4 w-4" />
                  </Link>
                </Button>
              </CardAction>
            )}
          </CardHeader>
          <CardContent>
            {hasShop ? (
              shopQuery.isLoading ? (
                <div className="space-y-2">
                  <Skeleton className="h-10 w-full" />
                  <Skeleton className="h-10 w-full" />
                  <Skeleton className="h-10 w-full" />
                </div>
              ) : shopQuery.error ? (
                <ErrorDisplay
                  error={shopQuery.error}
                  retry={() => shopQuery.refetch()}
                />
              ) : commodities.length > 0 ? (
                <div className="flex flex-col gap-2">
                  {commodities.map(commodity => {
                    const data = itemDataById.get(commodity.attributes.templateId);
                    return (
                      <NpcShopCommodityWidget
                        key={commodity.id}
                        templateId={commodity.attributes.templateId}
                        mesoPrice={commodity.attributes.mesoPrice}
                        tokenPrice={commodity.attributes.tokenPrice}
                        tokenTemplateId={commodity.attributes.tokenTemplateId}
                        {...(data?.name !== undefined && { name: data.name })}
                        {...(data?.iconUrl !== undefined && { iconUrl: data.iconUrl })}
                      />
                    );
                  })}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">
                  Shop has no commodities configured.
                </p>
              )
            ) : (
              <p className="text-sm text-muted-foreground">No shop configured.</p>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">
              Conversation
              {hasConversation && conversation
                ? ` (${conversation.attributes.states.length})`
                : ""}
            </CardTitle>
            {hasConversation ? (
              <CardAction>
                <Button
                  variant="ghost"
                  size="icon"
                  asChild
                  title="Edit Conversation"
                  aria-label="Edit Conversation"
                >
                  <Link to={`/npcs/${npcId}/conversations`}>
                    <Pencil className="h-4 w-4" />
                  </Link>
                </Button>
              </CardAction>
            ) : (
              <CardAction>
                <Button
                  variant="ghost"
                  size="icon"
                  asChild
                  title="Create Conversation"
                  aria-label="Create Conversation"
                >
                  <Link to={`/npcs/${npcId}/conversations`}>
                    <Plus className="h-4 w-4" />
                  </Link>
                </Button>
              </CardAction>
            )}
          </CardHeader>
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
                <NpcConversationTreePreview conversation={conversation} />
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
