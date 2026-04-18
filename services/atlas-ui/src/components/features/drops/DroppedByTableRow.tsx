import { Link } from "react-router-dom";
import { TableCell, TableRow } from "@/components/ui/table";
import { useMobData } from "@/lib/hooks/useMobData";
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
          to={`/monsters/${monsterId}`}
          className="flex items-center gap-2 hover:underline"
        >
          {iconUrl && (
            <img
              src={iconUrl}
              alt={monsterName || String(monsterId)}
              width={24}
              height={24}
              loading="lazy"
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
            to={`/monsters/${monsterId}`}
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
