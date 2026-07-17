import { type ColumnDef } from "@tanstack/react-table";
import { Link } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { useTenant } from "@/context/tenant-context";
import { useItemName } from "@/lib/hooks/api/useItemStrings";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import type { RewardPoolData } from "@/types/models/reward-pool";

/**
 * Incubator pools are identified by their egg item id (= pool id): show the
 * egg's item icon + resolved item name, falling back to the seeded pool name.
 */
function PoolNameCell({ pool }: { pool: RewardPoolData }) {
  const { activeTenant } = useTenant();
  const isIncubator = pool.attributes.kind === "incubator";
  const { data: eggName } = useItemName(isIncubator ? pool.id : "");
  const iconUrl =
    isIncubator && activeTenant
      ? getAssetIconUrl(activeTenant.id, activeTenant.attributes.region, activeTenant.attributes.majorVersion, activeTenant.attributes.minorVersion, "item", parseInt(pool.id))
      : null;
  return (
    <Link to={`/reward-pools/${pool.id}`} className="hover:underline">
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="inline-flex items-center gap-2 font-medium">
              {iconUrl && <img src={iconUrl} alt="" width={20} height={20} loading="lazy" />}
              {isIncubator ? (eggName ?? pool.attributes.name) : pool.attributes.name}
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

export const poolColumns: ColumnDef<RewardPoolData>[] = [
  {
    accessorKey: "attributes.name",
    header: "Name",
    cell: ({ row }) => <PoolNameCell pool={row.original} />,
  },
  {
    accessorKey: "attributes.kind",
    header: "Kind",
    cell: ({ row }) =>
      row.original.attributes.kind === "incubator" ? (
        <Badge className="bg-amber-500/15 text-amber-600 dark:text-amber-400 border-transparent">Incubator</Badge>
      ) : (
        <Badge variant="secondary">Gachapon</Badge>
      ),
  },
  {
    id: "details",
    header: "Details",
    cell: ({ row }) => {
      const a = row.original.attributes;
      if (a.kind === "incubator") {
        return <span className="text-muted-foreground font-mono text-sm">egg {row.original.id}</span>;
      }
      return (
        <span className="text-muted-foreground text-sm">
          C/U/R {a.commonWeight}·{a.uncommonWeight}·{a.rareWeight} — {a.npcIds.length} NPC{a.npcIds.length === 1 ? "" : "s"}
        </span>
      );
    },
  },
];
