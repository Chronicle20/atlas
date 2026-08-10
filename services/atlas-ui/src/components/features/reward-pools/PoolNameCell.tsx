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
import type { RewardPoolData } from "@/types/models/reward-pool";

/**
 * Incubator pools are identified by their egg item id (= pool id): show the
 * egg's item icon + resolved item name (region-appended when known), falling
 * back to the seeded pool name. Cash-surprise pools are identified by their
 * box item id (= pool id) the same way, but keep the plain seeded name — the
 * egg-region formatting is an incubator-only convention. Gachapon pools show
 * the first NPC's icon (the machine) when one is configured.
 *
 * Selected via a Record keyed by RewardPoolKind (not a two-way
 * isIncubator/ternary): the previous binary check rendered every
 * non-incubator kind, including a future new one, through the gachapon
 * NPC-icon branch — a new kind would silently get the wrong icon.
 */
export const ICON_SOURCE: Record<
  RewardPoolData["attributes"]["kind"],
  "item" | "npc"
> = {
  incubator: "item",
  "cash-surprise": "item",
  gachapon: "npc",
};

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
