import { type ColumnDef } from "@tanstack/react-table";
import type { ReactNode } from "react";
import { ReportStatusBadge } from "@/components/features/reports/ReportStatusBadge";
import { type Report, ReportKindLabels } from "@/types/models/report";

interface ColumnProps {
  onRowClick?: (report: Report) => void;
}

export const hiddenColumns: string[] = [];

/**
 * Wraps cell content so every column participates in "row click navigates
 * to detail" (the shared `DataTable` renders plain `<TableRow>`s with no
 * row-level onClick, so click handling is applied per-cell instead of
 * changing the shared table component).
 */
function clickableCell(content: ReactNode, onClick?: () => void): ReactNode {
  if (!onClick) return content;
  return (
    <div className="cursor-pointer" onClick={onClick}>
      {content}
    </div>
  );
}

export const getColumns = ({ onRowClick }: ColumnProps): ColumnDef<Report>[] => {
  const handleClick = (report: Report) => onRowClick?.(report);

  return [
    {
      accessorKey: "attributes.kind",
      header: "Kind",
      cell: ({ row }) => {
        const kind = row.original.attributes.kind;
        return clickableCell(ReportKindLabels[kind] ?? kind, () =>
          handleClick(row.original),
        );
      },
    },
    {
      accessorKey: "attributes.reporterName",
      header: "Reporter",
      cell: ({ row }) => {
        const { reporterName, reporterId } = row.original.attributes;
        return clickableCell(reporterName || `#${reporterId}`, () =>
          handleClick(row.original),
        );
      },
    },
    {
      accessorKey: "attributes.accusedName",
      header: "Accused",
      cell: ({ row }) => {
        const { accusedName, accusedId } = row.original.attributes;
        return clickableCell(accusedName || `#${accusedId}`, () =>
          handleClick(row.original),
        );
      },
    },
    {
      accessorKey: "attributes.reasonType",
      header: "Reason",
      cell: ({ row }) => {
        return clickableCell(String(row.original.attributes.reasonType), () =>
          handleClick(row.original),
        );
      },
    },
    {
      id: "status",
      header: "Status",
      cell: ({ row }) => {
        return clickableCell(
          <ReportStatusBadge status={row.original.attributes.status} />,
          () => handleClick(row.original),
        );
      },
    },
    {
      accessorKey: "attributes.createdAt",
      header: "Created",
      cell: ({ row }) => {
        return clickableCell(
          new Date(row.original.attributes.createdAt).toLocaleString(),
          () => handleClick(row.original),
        );
      },
    },
  ];
};
