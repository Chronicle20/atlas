"use client"

import { ColumnDef } from "@tanstack/react-table";
import type { MonsterData } from "@/types/models/monster";
import Link from "next/link";
import { Badge } from "@/components/ui/badge";

export const columns: ColumnDef<MonsterData>[] = [
  {
    accessorKey: "id",
    header: "ID",
    cell: ({ row }) => (
      <Link href={`/monsters/${row.original.id}`} className="font-mono text-primary hover:underline">
        {row.original.id}
      </Link>
    ),
  },
  {
    accessorKey: "attributes.name",
    header: "Name",
    cell: ({ row }) => (
      <Link href={`/monsters/${row.original.id}`} className="font-medium hover:underline">
        {row.original.attributes.name}
      </Link>
    ),
  },
  {
    accessorKey: "attributes.level",
    header: "Level",
  },
  {
    accessorKey: "attributes.hp",
    header: "HP",
    cell: ({ row }) => row.original.attributes.hp.toLocaleString(),
  },
  {
    accessorKey: "attributes.mp",
    header: "MP",
    cell: ({ row }) => row.original.attributes.mp.toLocaleString(),
  },
  {
    accessorKey: "attributes.experience",
    header: "EXP",
    cell: ({ row }) => row.original.attributes.experience.toLocaleString(),
  },
  {
    accessorKey: "attributes.weapon_attack",
    header: "W.ATK",
  },
  {
    accessorKey: "attributes.magic_attack",
    header: "M.ATK",
  },
  {
    id: "tags",
    header: "Tags",
    cell: ({ row }) => {
      const { boss, undead, friendly } = row.original.attributes;
      return (
        <div className="flex gap-1">
          {boss && <Badge variant="destructive">Boss</Badge>}
          {undead && <Badge variant="secondary">Undead</Badge>}
          {friendly && <Badge variant="outline">Friendly</Badge>}
        </div>
      );
    },
  },
];

export const hiddenColumns = ["attributes.mp", "attributes.weapon_attack", "attributes.magic_attack"];
