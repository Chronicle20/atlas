import { useTenant } from "@/context/tenant-context";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { hiddenColumns, getColumns } from "@/pages/bans-columns";
import { useMemo, useState } from "react";
import { useBans, useInvalidateBans } from "@/lib/hooks/api/useBans";
import type { Ban } from "@/types/models/ban";
import { BanType, BanTypeLabels } from "@/types/models/ban";
import { CreateBanDialog } from "@/components/features/bans/CreateBanDialog";
import { DeleteBanDialog } from "@/components/features/bans/DeleteBanDialog";
import { ExpireBanDialog } from "@/components/features/bans/ExpireBanDialog";
import { Toaster } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Plus, Shield } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { Skeleton } from "@/components/ui/skeleton";

function BansPageSkeleton() {
  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="flex items-center justify-between">
        <Skeleton className="h-8 w-32" />
        <div className="flex items-center gap-4">
          <Skeleton className="h-9 w-40" />
          <Skeleton className="h-9 w-32" />
        </div>
      </div>
      <div className="space-y-3">
        <Skeleton className="h-10 w-full" />
        {Array.from({ length: 10 }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    </div>
  );
}

export function BansPage() {
  const { activeTenant } = useTenant();
  const navigate = useNavigate();
  const [typeFilter, setTypeFilter] = useState<string>("all");
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [expireDialogOpen, setExpireDialogOpen] = useState(false);
  const [selectedBan, setSelectedBan] = useState<Ban | null>(null);

  const bansQueryOptions = useMemo(
    () => (typeFilter !== "all" ? { type: Number(typeFilter) as BanType } : undefined),
    [typeFilter]
  );
  const bansQuery = useBans(activeTenant, bansQueryOptions);
  const { invalidateAll } = useInvalidateBans();

  const bans = bansQuery.data ?? [];
  const loading = bansQuery.isLoading;
  const error = bansQuery.error?.message ?? null;

  const handleView = (ban: Ban) => navigate(`/bans/${ban.id}`);
  const handleDelete = (ban: Ban) => { setSelectedBan(ban); setDeleteDialogOpen(true); };
  const handleExpire = (ban: Ban) => { setSelectedBan(ban); setExpireDialogOpen(true); };
  const handleDeleteSuccess = () => setSelectedBan(null);
  const handleExpireSuccess = () => setSelectedBan(null);

  const columns = getColumns({
    onView: handleView,
    onDelete: handleDelete,
    onExpire: handleExpire,
  });

  if (loading && bans.length === 0) {
    return <BansPageSkeleton />;
  }

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Shield className="h-6 w-6" />
          <h2 className="text-2xl font-bold tracking-tight">Bans</h2>
        </div>
        <div className="flex items-center gap-4">
          <Select value={typeFilter} onValueChange={setTypeFilter}>
            <SelectTrigger className="w-[180px]">
              <SelectValue placeholder="Filter by type" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Types</SelectItem>
              {Object.entries(BanTypeLabels).map(([value, label]) => (
                <SelectItem key={value} value={value}>
                  {label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button onClick={() => setCreateDialogOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Create Ban
          </Button>
        </div>
      </div>

      <div className="mt-4">
        <DataTableWrapper
          columns={columns}
          data={bans}
          error={error}
          onRefresh={() => invalidateAll()}
          initialVisibilityState={hiddenColumns}
          emptyState={{
            title: "No bans found",
            description: typeFilter !== "all"
              ? "No bans match the selected filter. Try selecting a different type or create a new ban."
              : "There are no bans to display. Create a new ban to get started.",
            action: {
              label: "Create Ban",
              onClick: () => setCreateDialogOpen(true),
            },
          }}
        />
      </div>

      <CreateBanDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
        tenant={activeTenant}
        onSuccess={() => invalidateAll()}
      />

      <DeleteBanDialog
        ban={selectedBan}
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        tenant={activeTenant}
        onSuccess={handleDeleteSuccess}
      />

      <ExpireBanDialog
        ban={selectedBan}
        open={expireDialogOpen}
        onOpenChange={setExpireDialogOpen}
        tenant={activeTenant}
        onSuccess={handleExpireSuccess}
      />

      <Toaster richColors />
    </div>
  );
}
