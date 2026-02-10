"use client"

import { ColumnDef } from "@tanstack/react-table";
import type { GachaponData } from "@/types/models/gachapon";
import Link from "next/link";

export const columns: ColumnDef<GachaponData>[] = [
  {
    accessorKey: "id",
    header: "ID",
    cell: ({ row }) => (
      <Link href={`/gachapons/${row.original.id}`} className="font-mono text-primary hover:underline">
        {row.original.id}
      </Link>
    ),
  },
  {
    accessorKey: "attributes.name",
    header: "Name",
    cell: ({ row }) => (
      <Link href={`/gachapons/${row.original.id}`} className="font-medium hover:underline">
        {row.original.attributes.name}
      </Link>
    ),
  },
  {
    id: "npcCount",
    header: "NPCs",
    cell: ({ row }) => row.original.attributes.npcIds?.length ?? 0,
  },
  {
    accessorKey: "attributes.commonWeight",
    header: "Common Weight",
  },
  {
    accessorKey: "attributes.uncommonWeight",
    header: "Uncommon Weight",
  },
  {
    accessorKey: "attributes.rareWeight",
    header: "Rare Weight",
  },
];
