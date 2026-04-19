import { Link } from "react-router-dom";
import { UserCircle2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useNpcData } from "@/lib/hooks/useNpcData";
import { useNpcSpawnMaps } from "@/lib/hooks/api/useNpcSpawnMaps";
import type { ItemSellerCommodity } from "@/types/models/npc";

interface ItemNpcShopWidgetProps {
  commodity: ItemSellerCommodity;
}

export function ItemNpcShopWidget({ commodity }: ItemNpcShopWidgetProps) {
  const { npcId, mesoPrice, tokenPrice, tokenTemplateId, discountRate, period, levelLimit } = commodity;
  const { name: npcName, iconUrl, isLoading: npcLoading } = useNpcData(npcId);
  const { data: spawnMaps } = useNpcSpawnMaps(npcId);

  const priceLine = formatPrice(mesoPrice, tokenPrice, tokenTemplateId);
  const primarySpawnMap = spawnMaps && spawnMaps.length > 0 ? spawnMaps[0] : null;
  const extraMapCount = spawnMaps && spawnMaps.length > 1 ? spawnMaps.length - 1 : 0;
  const mapLabel = primarySpawnMap
    ? primarySpawnMap.streetName
      ? `${primarySpawnMap.name} · ${primarySpawnMap.streetName}`
      : primarySpawnMap.name
    : null;

  const widget = (
    <Link
      to={`/npcs/${npcId}/shop`}
      className="flex items-center gap-3 rounded-md border bg-card p-3 hover:bg-accent transition-colors"
    >
      <div className="flex h-8 w-8 shrink-0 items-center justify-center">
        {iconUrl ? (
          <img
            src={iconUrl}
            alt={npcName || `NPC ${npcId}`}
            width={32}
            height={32}
            loading="lazy"
            className="max-h-full max-w-full object-contain"
          />
        ) : (
          <UserCircle2 className="h-7 w-7 text-muted-foreground" />
        )}
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate">
          {npcLoading && !npcName ? `NPC #${npcId}` : npcName || `NPC #${npcId}`}
        </p>
        <p className="text-xs text-muted-foreground truncate">{priceLine}</p>
      </div>
      {mapLabel && (
        <Badge variant="secondary" className="hidden sm:inline-flex">
          {mapLabel}
        </Badge>
      )}
      {extraMapCount > 0 && (
        <Badge variant="outline" className="hidden sm:inline-flex">
          +{extraMapCount}
        </Badge>
      )}
    </Link>
  );

  const tooltipLines: string[] = [];
  if (discountRate > 0) tooltipLines.push(`Discount Rate: ${discountRate}%`);
  if (period > 0) tooltipLines.push(`Period: ${period}h`);
  if (levelLimit > 0) tooltipLines.push(`Level Limit: ${levelLimit}`);

  if (tooltipLines.length === 0) {
    return widget;
  }

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>{widget}</TooltipTrigger>
        <TooltipContent>
          {tooltipLines.map((line, idx) => (
            <p key={idx}>{line}</p>
          ))}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

function formatPrice(mesoPrice: number, tokenPrice: number, tokenTemplateId: number): string {
  const parts: string[] = [];
  if (mesoPrice > 0) parts.push(`${mesoPrice.toLocaleString()} mesos`);
  if (tokenPrice > 0 && tokenTemplateId > 0) {
    parts.push(`${tokenPrice.toLocaleString()} × item ${tokenTemplateId}`);
  }
  if (parts.length === 0) return "Free";
  return parts.join(" · ");
}
