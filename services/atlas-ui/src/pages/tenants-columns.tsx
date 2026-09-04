import { type DataTableColumnDef } from "@/components/data-table-features";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { MoreHorizontal } from "lucide-react";
import { Link } from "react-router-dom";
import type { Tenant, TenantConfig } from "@/types/models/tenant";

interface ColumnProps {
  onDelete?: (id: string) => void;
  onRename?: (id: string) => void;
  configs?: Map<string, TenantConfig>;
}

export const getColumns = ({
  onDelete,
  onRename,
  configs,
}: ColumnProps): DataTableColumnDef<Tenant>[] => [
  {
    accessorKey: "id",
    header: "Id",
  },
  {
    accessorKey: "attributes.name",
    header: "Name",
    cell: ({ row }) => (
      <Link
        to={"/tenants/" + row.original.id + "/properties"}
        className="font-medium text-primary hover:underline"
      >
        {row.original.attributes.name}
      </Link>
    ),
  },
  {
    accessorKey: "attributes.region",
    header: "Region",
  },
  {
    accessorKey: "attributes.majorVersion",
    header: "Major",
  },
  {
    accessorKey: "attributes.minorVersion",
    header: "Minor",
  },
  {
    id: "templateDrift",
    header: "Template",
    cell: ({ row }) => {
      // Strictly `=== true`: the attribute is optional (an older API,
      // or a tenant with no configuration row at all) and `undefined`
      // must read as "no badge", never as truthy-ish. FR-1.3 falls out
      // of the server contract - templateDrift is false whenever
      // baselineTemplateId is empty.
      const config = configs?.get(row.original.id);
      if (config?.attributes.templateDrift !== true) return null;
      return (
        <Tooltip>
          <TooltipTrigger asChild>
            {/*
              `secondary`, not `destructive`: drift is advisory and
              relative to a template row that is itself mutable
              (NFR-4). A tenant an operator edited on purpose is not in
              an error state.
            */}
            <Badge variant="secondary">Differs from template</Badge>
          </TooltipTrigger>
          <TooltipContent>
            Differs from the template this tenant derives from
          </TooltipContent>
        </Tooltip>
      );
    },
  },
  {
    id: "actions",
    cell: ({ row }) => {
      const id = row.getValue("id") as string;

      return (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" className="h-8 w-8 p-0">
              <span className="sr-only">Open menu</span>
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {onRename && (
              <DropdownMenuItem onClick={() => onRename(id)}>
                Rename
              </DropdownMenuItem>
            )}
            {onDelete && (
              <DropdownMenuItem
                className="text-destructive focus:text-destructive"
                onClick={() => onDelete(id)}
              >
                Delete
              </DropdownMenuItem>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      );
    },
  },
];
