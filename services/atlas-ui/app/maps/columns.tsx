"use client"

import { ColumnDef } from "@tanstack/react-table";
import type { MapData } from "@/services/api/maps.service";
import Link from "next/link";
import { Badge } from "@/components/ui/badge";

export const columns: ColumnDef<MapData>[] = [
  {
    accessorKey: "id",
    header: "ID",
    cell: ({ row }) => (
      <Link href={`/maps/${row.original.id}`} className="font-mono text-primary hover:underline">
        {row.original.id}
      </Link>
    ),
  },
  {
    accessorKey: "attributes.name",
    header: "Name",
    cell: ({ row }) => (
      <Link href={`/maps/${row.original.id}`} className="font-medium hover:underline">
        {row.original.attributes.name}
      </Link>
    ),
  },
  {
    accessorKey: "attributes.streetName",
    header: "Street Name",
  },
];
