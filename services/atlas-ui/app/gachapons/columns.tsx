"use client"

import { ColumnDef } from "@tanstack/react-table";
import type { GachaponData } from "@/types/models/gachapon";

export const columns: ColumnDef<GachaponData>[] = [
  {
    accessorKey: "id",
    header: "ID",
    cell: ({ row }) => (
      <span className="font-mono">{row.original.id}</span>
    ),
  },
  {
    accessorKey: "attributes.name",
    header: "Name",
    cell: ({ row }) => (
      <span className="font-medium">{row.original.attributes.name}</span>
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
