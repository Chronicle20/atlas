"use client"

import Link from "next/link";
import { TableCell, TableRow } from "@/components/ui/table";
import { useItemName } from "@/lib/hooks/api/useItemStrings";
import type { DropData } from "@/types/models/drop";

interface DropTableRowProps {
  drop: DropData;
}

export function DropTableRow({ drop }: DropTableRowProps) {
  const itemId = String(drop.attributes.itemId);
  const { data: itemName, isLoading: nameLoading } = useItemName(itemId);

  return (
    <TableRow>
      <TableCell>
        <Link
          href={`/items/${itemId}`}
          className="font-mono text-primary hover:underline"
        >
          {itemId}
        </Link>
      </TableCell>
      <TableCell>
        {nameLoading ? (
          <span className="text-muted-foreground">...</span>
        ) : itemName ? (
          <Link
            href={`/items/${itemId}`}
            className="font-medium hover:underline"
          >
            {itemName}
          </Link>
        ) : (
          <span className="text-muted-foreground">-</span>
        )}
      </TableCell>
      <TableCell>{drop.attributes.chance.toLocaleString()}</TableCell>
      <TableCell>{drop.attributes.minimumQuantity}</TableCell>
      <TableCell>{drop.attributes.maximumQuantity}</TableCell>
      <TableCell className="font-mono">{drop.attributes.questId || "-"}</TableCell>
    </TableRow>
  );
}
