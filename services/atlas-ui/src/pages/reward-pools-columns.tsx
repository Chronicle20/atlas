import { type ColumnDef } from "@tanstack/react-table";
import { KindBadge } from "@/components/features/reward-pools/KindBadge";
import { PoolNameCell } from "@/components/features/reward-pools/PoolNameCell";
import type { RewardPoolData } from "@/types/models/reward-pool";

export const poolColumns: ColumnDef<RewardPoolData>[] = [
  {
    accessorKey: "attributes.name",
    header: "Name",
    cell: ({ row }) => <PoolNameCell pool={row.original} />,
  },
  {
    accessorKey: "attributes.kind",
    header: "Kind",
    cell: ({ row }) => <KindBadge kind={row.original.attributes.kind} />,
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
