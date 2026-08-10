import { Link } from "react-router-dom";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useTenant } from "@/context/tenant-context";
import { useItemName } from "@/lib/hooks/api/useItemStrings";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { formatIncubatorName } from "@/lib/utils/egg-regions";
import { ICON_SOURCE } from "@/lib/utils/reward-pool-chance";
import type { RewardPoolData } from "@/types/models/reward-pool";

export function PoolNameCell({ pool }: { pool: RewardPoolData }) {
  const { activeTenant } = useTenant();
  const isIncubator = pool.attributes.kind === "incubator";
  const iconSource = ICON_SOURCE[pool.attributes.kind];
  const { data: eggName } = useItemName(isIncubator ? pool.id : "");
  const firstNpcId = pool.attributes.npcIds[0];
  const iconUrl =
    iconSource === "item" && activeTenant
      ? getAssetIconUrl(
          activeTenant.id,
          activeTenant.attributes.region,
          activeTenant.attributes.majorVersion,
          activeTenant.attributes.minorVersion,
          "item",
          parseInt(pool.id),
        )
      : iconSource === "npc" && activeTenant && firstNpcId !== undefined
        ? getAssetIconUrl(
            activeTenant.id,
            activeTenant.attributes.region,
            activeTenant.attributes.majorVersion,
            activeTenant.attributes.minorVersion,
            "npc",
            firstNpcId,
          )
        : null;
  const displayName = isIncubator
    ? formatIncubatorName(eggName ?? pool.attributes.name, pool.id)
    : pool.attributes.name;
  return (
    <Link to={`/reward-pools/${pool.id}`} className="hover:underline">
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="inline-flex items-center gap-2 font-medium">
              {iconUrl && (
                <img
                  src={iconUrl}
                  alt=""
                  width={20}
                  height={20}
                  loading="lazy"
                />
              )}
              {displayName}
            </span>
          </TooltipTrigger>
          <TooltipContent copyable>
            <p>{pool.id}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </Link>
  );
}
