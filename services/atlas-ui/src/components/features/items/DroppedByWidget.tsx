import { Link } from "react-router-dom";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useMobData } from "@/lib/hooks/useMobData";
import type { DropData } from "@/types/models/drop";

interface DroppedByWidgetProps {
  drop: DropData;
}

export function DroppedByWidget({ drop }: DroppedByWidgetProps) {
  const monsterId = drop.attributes.monsterId;
  const { name: monsterName, iconUrl, isLoading } = useMobData(monsterId);

  const widget = (
    <Link
      to={`/monsters/${monsterId}`}
      className="flex items-center gap-3 rounded-md border bg-card p-2 hover:bg-accent transition-colors"
    >
      <div className="h-8 w-8 shrink-0 flex items-center justify-center">
        {iconUrl && (
          <img
            src={iconUrl}
            alt={monsterName || String(monsterId)}
            width={32}
            height={32}
            loading="lazy"
            className="max-h-full max-w-full object-contain"
          />
        )}
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate">
          {isLoading && !monsterName ? `Monster #${monsterId}` : monsterName || `Monster #${monsterId}`}
        </p>
        <p className="text-xs font-mono text-muted-foreground">{monsterId}</p>
      </div>
    </Link>
  );

  const tooltipLines: string[] = [
    `Chance: ${drop.attributes.chance.toLocaleString()}`,
    `Min Qty: ${drop.attributes.minimumQuantity}`,
    `Max Qty: ${drop.attributes.maximumQuantity}`,
  ];
  if (drop.attributes.questId > 0) tooltipLines.push(`Quest ID: ${drop.attributes.questId}`);

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
