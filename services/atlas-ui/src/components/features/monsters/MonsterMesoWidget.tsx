import { Coins } from "lucide-react";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { DropData } from "@/types/models/drop";

interface MonsterMesoWidgetProps {
  drop: DropData;
}

export function MonsterMesoWidget({ drop }: MonsterMesoWidgetProps) {
  const min = drop.attributes.minimumQuantity.toLocaleString();
  const max = drop.attributes.maximumQuantity.toLocaleString();
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="flex items-center gap-3 rounded-md border border-amber-300/40 bg-amber-50/50 dark:border-amber-700/40 dark:bg-amber-950/20 p-2">
            <Coins className="size-5 text-amber-500 shrink-0" />
            <div className="min-w-0 flex-1">
              <div className="text-sm font-medium">Mesos</div>
              <div className="text-xs text-muted-foreground">
                {min} – {max}
              </div>
            </div>
          </div>
        </TooltipTrigger>
        <TooltipContent>
          <div className="flex gap-4">
            <span className="text-muted-foreground">Chance</span>
            <span className="ml-auto">{drop.attributes.chance.toLocaleString()}</span>
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
