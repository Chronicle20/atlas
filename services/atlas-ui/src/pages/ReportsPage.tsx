import { useTenant } from "@/context/tenant-context";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { hiddenColumns, getColumns } from "@/pages/reports-columns";
import { useMemo, useState } from "react";
import { useReports } from "@/lib/hooks/api/useReports";
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
import {
  type Report,
  ReportStatus,
  ReportStatusLabels,
} from "@/types/models/report";
import { Toaster } from "sonner";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Flag } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { Skeleton } from "@/components/ui/skeleton";

function ReportsPageSkeleton() {
  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="flex items-center justify-between">
        <Skeleton className="h-8 w-32" />
        <Skeleton className="h-9 w-40" />
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

export function ReportsPage() {
  const { activeTenant } = useTenant();
  const navigate = useNavigate();
  const [statusFilter, setStatusFilter] = useState<string>("all");

  const reportsQueryOptions = useMemo(
    () =>
      statusFilter !== "all"
        ? { status: statusFilter as ReportStatus }
        : undefined,
    [statusFilter],
  );

  const reportsQuery = useReports(activeTenant, reportsQueryOptions);
  const { isRefreshing, onRefresh, lastUpdatedAt } = useGridRefresh([
    reportsQuery,
  ]);

  const reports = reportsQuery.data ?? [];
  const loading = reportsQuery.isLoading;
  const error = reportsQuery.error?.message ?? null;

  const handleView = (report: Report) => navigate(`/reports/${report.id}`);

  const columns = getColumns({ onView: handleView });

  if (loading && reports.length === 0) {
    return <ReportsPageSkeleton />;
  }

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Flag className="h-6 w-6" />
          <h2 className="text-2xl font-bold tracking-tight">Reports</h2>
        </div>
        <Select value={statusFilter} onValueChange={setStatusFilter}>
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="Filter by status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Statuses</SelectItem>
            {Object.entries(ReportStatusLabels).map(([value, label]) => (
              <SelectItem key={value} value={value}>
                {label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="mt-4">
        <DataTableWrapper
          columns={columns}
          data={reports}
          error={error}
          onRefresh={onRefresh}
          isRefreshing={isRefreshing}
          lastUpdatedAt={lastUpdatedAt}
          initialVisibilityState={hiddenColumns}
          emptyState={{
            title: "No reports found",
            description:
              statusFilter !== "all"
                ? "No reports match the selected filter. Try selecting a different status."
                : "There are no reports to display.",
          }}
        />
      </div>

      <Toaster richColors />
    </div>
  );
}
