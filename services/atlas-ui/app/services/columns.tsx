"use client";

import { ColumnDef } from "@tanstack/react-table";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { MoreHorizontal } from "lucide-react";
import Link from "next/link";

import { ServiceTypeBadge } from "@/components/features/services";
import type { Service } from "@/types/models/service";
import {
  getServiceTenantCount,
  getServiceTaskCount,
} from "@/types/models/service";

interface ColumnProps {
  onDelete?: (service: Service) => void;
}

export const getColumns = ({ onDelete }: ColumnProps): ColumnDef<Service>[] => [
  {
    accessorKey: "id",
    header: "Service ID",
    cell: ({ row }) => {
      const id = row.getValue("id") as string;
      return (
        <span className="font-mono text-xs">{id}</span>
      );
    },
  },
  {
    accessorKey: "attributes.type",
    header: "Type",
    cell: ({ row }) => {
      const type = row.original.attributes.type;
      return <ServiceTypeBadge type={type} />;
    },
  },
  {
    id: "tenants",
    header: "Tenants",
    cell: ({ row }) => {
      const count = getServiceTenantCount(row.original);
      if (count === 0) {
        return <span className="text-muted-foreground">-</span>;
      }
      return <span>{count}</span>;
    },
  },
  {
    id: "tasks",
    header: "Tasks",
    cell: ({ row }) => {
      const count = getServiceTaskCount(row.original);
      return <span>{count}</span>;
    },
  },
  {
    id: "actions",
    cell: ({ row }) => {
      const service = row.original;

      return (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" className="h-8 w-8 p-0">
              <span className="sr-only">Open menu</span>
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem asChild>
              <Link href={`/services/${service.id}`}>View Service</Link>
            </DropdownMenuItem>
            {onDelete && (
              <DropdownMenuItem
                className="text-destructive focus:text-destructive"
                onClick={() => onDelete(service)}
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
