import { Link } from "react-router-dom";
import { Card, CardContent } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";
import type { LiveMonsterData } from "@/services/api/live-monsters.service";

export interface FieldMonstersTabProps {
  monsters?: LiveMonsterData[];
  error?: Error;
  mapId: number;
}

/**
 * FR-28..FR-31: the field-detail Monsters tab. D13 settled the columns:
 * Object ID, Monster (linked), HP, Position, Spawn — there is no `state`
 * field on the live-monster payload, so FR-31's dead/alive distinction is
 * `hp === 0` styling (a `data-dead` attribute), never a color alone.
 * The spawn column is best-effort (FR-30): `spawnSourceId` is opaque and
 * cannot be correlated to a specific definition spawn row, so it links the
 * map definition's Monsters tab rather than a row.
 */
export function FieldMonstersTab({
  monsters,
  error,
  mapId,
}: FieldMonstersTabProps) {
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

  if (!monsters || monsters.length === 0) {
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
          <TableHead>Spawn</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {monsters.map((monster) => (
          <FieldMonsterRow key={monster.id} monster={monster} mapId={mapId} />
        ))}
      </TableBody>
    </Table>
  );
}

interface FieldMonsterRowProps {
  monster: LiveMonsterData;
  mapId: number;
}

function FieldMonsterRow({ monster, mapId }: FieldMonsterRowProps) {
  const { monsterId, hp, maxHp, x, y, spawnSourceType, spawnSourceId } =
    monster.attributes;
  const dead = hp === 0;
  const hasSpawn = !!spawnSourceType || !!spawnSourceId;

  return (
    <TableRow
      className={cn(dead && "opacity-60")}
      data-dead={dead ? "true" : undefined}
    >
      <TableCell>{monster.id}</TableCell>
      <TableCell>
        <Link to={`/monsters/${monsterId}`} className="underline">
          {monsterId}
        </Link>
      </TableCell>
      <TableCell>{`${hp} / ${maxHp}`}</TableCell>
      <TableCell>{`(${x}, ${y})`}</TableCell>
      <TableCell>
        {hasSpawn ? (
          <Link to={`/maps/${mapId}?tab=monsters`} className="underline">
            {[spawnSourceType, spawnSourceId].filter(Boolean).join(" / ")}
          </Link>
        ) : (
          "—"
        )}
      </TableCell>
    </TableRow>
  );
}
