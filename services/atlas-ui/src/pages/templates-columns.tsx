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
import type { Template } from "@/types/models/template";

interface ColumnProps {
  onDelete?: (id: string) => void;
  onClone?: (id: string) => void;
  onCreateTenant?: (id: string) => void;
}

export const getColumns = ({
  onDelete,
  onClone,
  onCreateTenant,
}: ColumnProps): DataTableColumnDef<Template>[] => [
  {
    accessorKey: "id",
    header: "Id",
    cell: ({ row }) => {
      const id = row.getValue("id") as string;
      return (
        <Link
          to={"/templates/" + id + "/properties"}
          className="font-mono text-primary hover:underline"
        >
          {id}
        </Link>
      );
    },
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
    id: "seedDrift",
    header: "Seed",
    cell: ({ row }) => {
      // Strictly `=== true`: the attribute is optional (an older API or a
      // fixture may omit it) and `undefined` must read as "no badge", never as
      // truthy-ish. FR-5.2 falls out of the server contract - seedDrift is
      // false whenever shippedRevision is empty.
      if (row.original.attributes.seedDrift !== true) return null;
      return (
        <Tooltip>
          <TooltipTrigger asChild>
            {/*
              `secondary`, not `destructive`: the flag is advisory and
              image-relative (NFR-4). A template an operator edited on purpose
              is not in an error state.
            */}
            <Badge variant="secondary">Differs from image</Badge>
          </TooltipTrigger>
          <TooltipContent>
            Differs from the configuration shipped in this image
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
            {onClone && (
              <DropdownMenuItem onClick={() => onClone(id)}>
                Clone Template
              </DropdownMenuItem>
            )}
            {onCreateTenant && (
              <DropdownMenuItem onClick={() => onCreateTenant(id)}>
                Create Tenant from Template
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
