import { Link } from "react-router-dom";
import { Package } from "lucide-react";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useItemData } from "@/lib/hooks/useItemData";
import type { DropData } from "@/types/models/drop";

interface MonsterDropWidgetProps {
  drop: DropData;
}

export function MonsterDropWidget({ drop }: MonsterDropWidgetProps) {
  const itemId = drop.attributes.itemId;
  const { name, iconUrl } = useItemData(itemId);

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <Link
            to={`/items/${itemId}`}
            className="flex items-center gap-3 rounded-md border bg-card p-2 hover:bg-accent transition-colors"
          >
            {iconUrl ? (
              <img
                src={iconUrl}
                alt={name ?? String(itemId)}
                width={32}
                height={32}
                loading="lazy"
                className="object-contain shrink-0"
              />
            ) : (
              <Package className="size-8 text-muted-foreground shrink-0" />
            )}
            <div className="min-w-0 flex-1">
              <div className="text-sm font-medium truncate">
                {name ?? String(itemId)}
              </div>
              <div className="text-xs font-mono text-muted-foreground">
                {itemId}
              </div>
            </div>
          </Link>
        </TooltipTrigger>
        <TooltipContent>
          <div className="space-y-0.5">
            <div className="flex gap-4">
              <span className="text-muted-foreground">Chance</span>
              <span className="ml-auto">{drop.attributes.chance.toLocaleString()}</span>
            </div>
            <div className="flex gap-4">
              <span className="text-muted-foreground">Min Qty</span>
              <span className="ml-auto">{drop.attributes.minimumQuantity}</span>
            </div>
            <div className="flex gap-4">
              <span className="text-muted-foreground">Max Qty</span>
              <span className="ml-auto">{drop.attributes.maximumQuantity}</span>
            </div>
            {drop.attributes.questId > 0 && (
              <div className="flex gap-4">
                <span className="text-muted-foreground">Quest</span>
                <span className="ml-auto font-mono">{drop.attributes.questId}</span>
              </div>
            )}
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
