"use client"

import { ColumnDef } from "@tanstack/react-table";
import type { MapData } from "@/services/api/maps.service";
import Link from "next/link";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

export const hiddenColumns = ["id"];

export const columns: ColumnDef<MapData>[] = [
  {
    accessorKey: "id",
    header: "ID",
  },
  {
    accessorKey: "attributes.name",
    header: "Name",
    cell: ({ row }) => (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <Link href={`/maps/${row.original.id}`}>
              <Badge variant="secondary">{row.original.attributes.name}</Badge>
            </Link>
          </TooltipTrigger>
          <TooltipContent copyable>
            <p>{row.original.id}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    ),
  },
  {
    accessorKey: "attributes.streetName",
    header: "Street Name",
  },
];
