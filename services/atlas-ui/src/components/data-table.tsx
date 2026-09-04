import {
  type ColumnVisibilityState,
  flexRender,
  type RowData,
  useTable,
} from "@tanstack/react-table";
import {
  dataTableFeatures,
  type DataTableColumnDef,
} from "@/components/data-table-features";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import React from "react";
import { Button } from "./ui/button";
import { RefreshCw, MoreVertical } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

interface DataTableHeaderAction {
  icon?: React.ReactNode;
  label: string;
  onClick: () => void;
}

interface DataTableProps<TData extends RowData> {
  initialVisibilityState?: string[];
  columns: DataTableColumnDef<TData>[];
  data: TData[];
  onRefresh?: () => void;
  isRefreshing?: boolean;
  headerActions?: DataTableHeaderAction[];
}

export function DataTable<TData extends RowData>({
  initialVisibilityState,
  columns,
  data,
  onRefresh,
  isRefreshing,
  headerActions,
}: DataTableProps<TData>) {
  const state = Object.fromEntries(
    (initialVisibilityState || []).map((col) => [
      col.replace(/\./g, "_"),
      false,
    ]),
  );
  const [columnVisibility, setColumnVisibility] =
    React.useState<ColumnVisibilityState>(state);

  const table = useTable({
    features: dataTableFeatures,
    data,
    columns,
    onColumnVisibilityChange: setColumnVisibility,
    state: {
      columnVisibility,
    },
  });

  return (
    <div className="w-full space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex gap-2 items-center">
          {onRefresh && (
            <Button
              variant="outline"
              size="icon"
              onClick={onRefresh}
              disabled={isRefreshing}
              className="hover:bg-accent cursor-pointer"
              title="Refresh"
              aria-busy={isRefreshing}
            >
              <RefreshCw
                className={cn("h-4 w-4", isRefreshing && "animate-spin")}
              />
            </Button>
          )}
        </div>
        <div className="flex items-center gap-2">
          {headerActions && headerActions.length > 0 && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm">
                  <MoreVertical className="h-4 w-4 mr-2" />
                  Actions
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                {headerActions.map((action, index) => (
                  <DropdownMenuItem key={index} onClick={action.onClick}>
                    {action.icon && <span className="mr-2">{action.icon}</span>}
                    {action.label}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>
      </div>

      <div className="w-full min-w-0 rounded-md border overflow-auto max-h-[calc(100vh-16rem)]">
        <Table className="[&>div]:overflow-visible table-fixed w-full">
          <TableHeader className="sticky top-0 z-10 bg-sidebar">
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => {
                  return (
                    <TableHead
                      key={header.id}
                      style={{ width: header.getSize() }}
                    >
                      {header.isPlaceholder
                        ? null
                        : flexRender(
                            header.column.columnDef.header,
                            header.getContext(),
                          )}
                    </TableHead>
                  );
                })}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows?.length ? (
              table.getRowModel().rows.map((row) => (
                <TableRow
                  key={row.id}
                  data-state={row.getIsSelected() && "selected"}
                >
                  {row.getVisibleCells().map((cell) => (
                    <TableCell
                      key={cell.id}
                      style={{ width: cell.column.getSize() }}
                    >
                      {flexRender(
                        cell.column.columnDef.cell,
                        cell.getContext(),
                      )}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="h-24 text-center"
                >
                  No results.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
