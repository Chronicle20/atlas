import {
  Tooltip, TooltipContent, TooltipProvider, TooltipTrigger,
} from "@/components/ui/tooltip";
import type { Asset } from "@/services/api/inventory.service";
import type { Tenant } from "@/services/api/tenants.service";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { AssetTooltipContent } from "./AssetTooltipContent";

interface Props {
  slotId: number;
  slotName: string;
  asset?: Asset | undefined;
  tenant: Tenant;
  itemName?: string | undefined;
}

export function EquipmentCell({ slotName, asset, tenant, itemName }: Props) {
  return (
    <div className="aspect-square border rounded">
      {asset ? (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <div tabIndex={0} className="w-full h-full p-1 cursor-help focus:outline-none focus-visible:ring-2 focus-visible:ring-ring">
                <img
                  src={getAssetIconUrl(
                    tenant.id, tenant.attributes.region,
                    tenant.attributes.majorVersion, tenant.attributes.minorVersion,
                    "item", asset.attributes.templateId,
                  )}
                  alt={itemName ?? slotName}
                  className="w-full h-full object-contain"
                />
              </div>
            </TooltipTrigger>
            <TooltipContent>
              <AssetTooltipContent asset={asset} itemName={itemName} slotName={slotName} />
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      ) : (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <div
                tabIndex={0}
                className="w-full h-full border-dashed border opacity-50 flex items-center justify-center text-xs text-muted-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                aria-label={`${slotName} (empty)`}
              >
                {slotName}
              </div>
            </TooltipTrigger>
            <TooltipContent>{slotName} (empty)</TooltipContent>
          </Tooltip>
        </TooltipProvider>
      )}
    </div>
  );
}
