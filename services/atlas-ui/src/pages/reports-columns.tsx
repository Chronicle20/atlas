import { type DataTableColumnDef } from "@/components/data-table-features";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { MoreHorizontal, Eye } from "lucide-react";
import { ReportStatusBadge } from "@/components/features/reports/ReportStatusBadge";
import {
  type Report,
  ReportKindLabels,
  formatReportReason,
} from "@/types/models/report";

interface ColumnProps {
  onView?: (report: Report) => void;
}

export const hiddenColumns: string[] = [];

export const getColumns = ({
  onView,
}: ColumnProps): DataTableColumnDef<Report>[] => {
  return [
    {
      accessorKey: "attributes.kind",
      header: "Kind",
      cell: ({ row }) => {
        const kind = row.original.attributes.kind;
        return ReportKindLabels[kind] ?? kind;
      },
    },
    {
      accessorKey: "attributes.reporterName",
      header: "Reporter",
      cell: ({ row }) => {
        const { reporterName, reporterId } = row.original.attributes;
        return reporterName || `#${reporterId}`;
      },
    },
    {
      accessorKey: "attributes.accusedName",
      header: "Accused",
      cell: ({ row }) => {
        const { accusedName, accusedId } = row.original.attributes;
        return accusedName || `#${accusedId}`;
      },
    },
    {
      accessorKey: "attributes.reasonType",
      header: "Reason",
      cell: ({ row }) => {
        const { kind, reasonType } = row.original.attributes;
        return formatReportReason(kind, reasonType);
      },
    },
    {
      id: "status",
      header: "Status",
      cell: ({ row }) => (
        <ReportStatusBadge status={row.original.attributes.status} />
      ),
    },
    {
      accessorKey: "attributes.createdAt",
      header: "Created",
      cell: ({ row }) =>
        new Date(row.original.attributes.createdAt).toLocaleString(),
    },
    {
      id: "actions",
      cell: ({ row }) => {
        const report = row.original;

        return (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="h-8 w-8 p-0">
                <span className="sr-only">Open menu</span>
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => onView?.(report)}>
                <Eye className="mr-2 h-4 w-4" />
                View Details
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        );
      },
    },
  ];
};
