import { Link } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { useMobData } from "@/lib/hooks/useMobData";
import type { LiveMonsterData } from "@/services/api/live-monsters.service";

export interface FieldMonstersTabProps {
  monsters?: LiveMonsterData[] | undefined;
  error?: Error | undefined;
}

/**
 * FR-28..FR-31: the field-detail Monsters tab. Columns are Object ID,
 * Monster (name badge, linked), HP, Position — there is no `state` field on
 * the live-monster payload, so FR-31's dead/alive distinction is `hp === 0`
 * styling (a `data-dead` attribute), never a color alone. There is no Spawn
 * column: `spawnSourceType`/`spawnSourceId` are opaque provenance fields
 * (atlas-monsters stores and echoes them but never interprets them), so they
 * cannot be correlated to a specific definition spawn row.
 */
export function FieldMonstersTab({ monsters, error }: FieldMonstersTabProps) {
  if (error) {
    return (
      <Card>
        <CardContent className="pt-6">
          <p className="text-sm text-destructive">
            Failed to load live monsters.
          </p>
        </CardContent>
      </Card>
    );
  }

  if (monsters === undefined) {
    return (
      <Card>
        <CardContent className="pt-6">
          <p className="text-sm text-muted-foreground">Loading monsters...</p>
        </CardContent>
      </Card>
    );
  }

  if (monsters.length === 0) {
    return (
      <Card>
        <CardContent className="pt-6">
          <p className="text-sm text-muted-foreground">
            No monsters are currently in this field.
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Object ID</TableHead>
          <TableHead>Monster</TableHead>
          <TableHead>HP</TableHead>
          <TableHead>Position</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {monsters.map((monster) => (
          <FieldMonsterRow key={monster.id} monster={monster} />
        ))}
      </TableBody>
    </Table>
  );
}

interface FieldMonsterRowProps {
  monster: LiveMonsterData;
}

function FieldMonsterRow({ monster }: FieldMonsterRowProps) {
  const { monsterId, hp, maxHp, x, y } = monster.attributes;
  const dead = hp === 0;
  const { name } = useMobData(monsterId);

  return (
    <TableRow
      className={cn(dead && "opacity-60")}
      data-dead={dead ? "true" : undefined}
    >
      <TableCell>{monster.id}</TableCell>
      <TableCell>
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <Link to={`/monsters/${monsterId}`}>
                <Badge variant="secondary">{name ?? monsterId}</Badge>
              </Link>
            </TooltipTrigger>
            <TooltipContent copyable>
              <p>{monsterId}</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </TableCell>
      <TableCell>{`${hp} / ${maxHp}`}</TableCell>
      <TableCell>{`(${x}, ${y})`}</TableCell>
    </TableRow>
  );
}
