import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useTenant } from "@/context/tenant-context";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { getItemTypeBadgeVariant, type ItemType } from "@/types/models/item";

interface ItemHeaderProps {
  itemId: string;
  itemName: string | null;
  itemType: ItemType;
}

export function ItemHeader({ itemId, itemName, itemType }: ItemHeaderProps) {
  const { activeTenant } = useTenant();
  const iconKey = `${activeTenant?.id ?? ""}-${itemId}`;
  const [failedKey, setFailedKey] = useState<string | null>(null);
  const iconFailed = failedKey === iconKey;

  const displayName = itemName || itemId;
  const iconUrl =
    activeTenant && !iconFailed
      ? getAssetIconUrl(
          activeTenant.id,
          activeTenant.attributes.region,
          activeTenant.attributes.majorVersion,
          activeTenant.attributes.minorVersion,
          "item",
          parseInt(itemId),
        )
      : null;

  return (
    <div className="flex items-center gap-3 flex-wrap">
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <span
              tabIndex={0}
              className="inline-flex items-center gap-3 cursor-help focus:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded"
            >
              {iconUrl && (
                <img
                  src={iconUrl}
                  alt={displayName}
                  width={64}
                  height={64}
                  onError={() => setFailedKey(iconKey)}
                  className="object-contain"
                />
              )}
              <h2 className="text-2xl font-bold tracking-tight">{displayName}</h2>
            </span>
          </TooltipTrigger>
          <TooltipContent copyable>
            <p>{itemId}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
      <Badge variant="secondary" className={getItemTypeBadgeVariant(itemType)}>
        {itemType}
      </Badge>
    </div>
  );
}
