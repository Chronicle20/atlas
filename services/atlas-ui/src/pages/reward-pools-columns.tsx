import { type DataTableColumnDef } from "@/components/data-table-features";
import { KindBadge } from "@/components/features/reward-pools/KindBadge";
import { PoolNameCell } from "@/components/features/reward-pools/PoolNameCell";
import type { RewardPoolData } from "@/types/models/reward-pool";

export const poolColumns: DataTableColumnDef<RewardPoolData>[] = [
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
];
