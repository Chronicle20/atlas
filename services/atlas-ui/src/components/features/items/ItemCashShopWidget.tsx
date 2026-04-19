import { Gem } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { ItemCashShopCommodity } from "@/types/models/npc";

interface ItemCashShopWidgetProps {
  commodity: ItemCashShopCommodity;
}

export function ItemCashShopWidget({ commodity }: ItemCashShopWidgetProps) {
  const { id, count, price, period, priority, gender, onSale } = commodity;

  const countSuffix = count > 1 ? ` · ${count}×` : "";
  const periodLabel = period > 0 ? `${period} days` : "Permanent";
  const genderLabel = gender === 0 ? "Male" : gender === 1 ? "Female" : null;

  const widget = (
    <div className="flex items-center gap-3 rounded-md border border-amber-300/40 bg-amber-50/50 p-3 dark:bg-amber-950/20">
      <Gem className="h-5 w-5 text-amber-500 shrink-0" />
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate">
          NX Cash · {price.toLocaleString()} NX{countSuffix}
        </p>
        <p className="text-xs text-muted-foreground truncate">{periodLabel}</p>
      </div>
      <div className="flex items-center gap-1">
        {onSale && <Badge variant="default">ON SALE</Badge>}
        {genderLabel && <Badge variant="outline">{genderLabel}</Badge>}
      </div>
    </div>
  );

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>{widget}</TooltipTrigger>
        <TooltipContent>
          <p>SN: {id}</p>
          <p>Priority: {priority}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
