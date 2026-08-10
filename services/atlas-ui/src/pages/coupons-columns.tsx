import { Link } from "react-router-dom";
import { Trash2 } from "lucide-react";
import { type DataTableColumnDef } from "@/components/data-table-features";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import {
  couponStatus,
  formatRewards,
  formatUses,
  formatWindow,
} from "@/lib/utils/coupons";
import type { Coupon } from "@/services/api/coupons.service";

interface ColumnProps {
  /** Sends a PATCH carrying `active` and nothing else. */
  onToggleActive: (coupon: Coupon, next: boolean) => void;
  onDelete: (coupon: Coupon) => void;
  /** Id currently being toggled, so its switch can be disabled mid-flight. */
  pendingToggleId: string | null;
}

export const hiddenColumns = ["id"];

export const getColumns = ({
  onToggleActive,
  onDelete,
  pendingToggleId,
}: ColumnProps): DataTableColumnDef<Coupon>[] => [
  {
    accessorKey: "id",
    header: "Id",
    enableHiding: false,
  },
  {
    id: "code",
    header: "Code",
    accessorFn: (row) => row.attributes.code,
    cell: ({ row }) => (
      <Link
        to={`/coupons/${row.original.id}`}
        className="font-mono font-medium text-primary hover:underline"
      >
        {row.original.attributes.code}
      </Link>
    ),
  },
  {
    id: "status",
    header: "Status",
    accessorFn: (row) => couponStatus(row.attributes),
    cell: ({ row }) => {
      const status = couponStatus(row.original.attributes);
      return (
        <Badge variant={status === "Active" ? "secondary" : "outline"}>
          {status}
        </Badge>
      );
    },
  },
  {
    id: "rewards",
    header: "Rewards",
    accessorFn: (row) => formatRewards(row.attributes.rewards),
    cell: ({ row }) => (
      <span className="text-sm">
        {formatRewards(row.original.attributes.rewards)}
      </span>
    ),
  },
  {
    id: "uses",
    header: "Uses",
    accessorFn: (row) => formatUses(row.attributes),
    cell: ({ row }) => (
      <span className="tabular-nums">
        {formatUses(row.original.attributes)}
      </span>
    ),
  },
  {
    id: "window",
    header: "Window",
    accessorFn: (row) => formatWindow(row.attributes),
    cell: ({ row }) => (
      <span className="text-sm text-muted-foreground">
        {formatWindow(row.original.attributes)}
      </span>
    ),
  },
  {
    id: "actions",
    header: "Actions",
    cell: ({ row }) => {
      const coupon = row.original;
      const { code, active, redemptionCount } = coupon.attributes;
      // The server refuses (409) to delete a coupon that has been redeemed,
      // so the affordance is disabled rather than left to fail.
      const redeemed = redemptionCount > 0;
      return (
        <div className="flex items-center gap-3">
          <Switch
            checked={active}
            disabled={pendingToggleId === coupon.id}
            aria-label={`${active ? "Deactivate" : "Activate"} ${code}`}
            onCheckedChange={(next) => onToggleActive(coupon, next)}
          />
          <Button
            variant="ghost"
            size="sm"
            aria-label={`Delete ${code}`}
            disabled={redeemed}
            title={
              redeemed ? "Redeemed coupons cannot be deleted" : `Delete ${code}`
            }
            onClick={() => onDelete(coupon)}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      );
    },
  },
];
