"use client"

import Link from "next/link";
import Image from "next/image";
import { TableCell, TableRow } from "@/components/ui/table";
import { useMobData } from "@/lib/hooks/useMobData";
import { shouldUnoptimizeImageSrc } from "@/lib/utils/image";
import type { DropData } from "@/types/models/drop";

interface DroppedByTableRowProps {
  drop: DropData;
}

export function DroppedByTableRow({ drop }: DroppedByTableRowProps) {
  const monsterId = drop.attributes.monsterId;
  const { name: monsterName, iconUrl, isLoading } = useMobData(monsterId);

  return (
    <TableRow>
      <TableCell>
        <Link
          href={`/monsters/${monsterId}`}
          className="flex items-center gap-2 hover:underline"
        >
          {iconUrl && (
            <Image
              src={iconUrl}
              alt={monsterName || String(monsterId)}
              width={24}
              height={24}
              unoptimized={shouldUnoptimizeImageSrc(iconUrl)}
              className="object-contain"
            />
          )}
          <span className="font-mono text-primary">{monsterId}</span>
        </Link>
      </TableCell>
      <TableCell>
        {isLoading ? (
          <span className="text-muted-foreground">...</span>
        ) : monsterName ? (
          <Link
            href={`/monsters/${monsterId}`}
            className="font-medium hover:underline"
          >
            {monsterName}
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
