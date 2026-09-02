import { Link } from "react-router-dom";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useFieldsForMap } from "@/lib/hooks/api/useFields";
import { useLiveMonsters } from "@/lib/hooks/api/useFieldRuntime";
import type { FieldData } from "@/services/api/fields.service";

// Beyond this many rows, per-row monster fan-out stops (D12): one request
// spanning every world/channel is fine, but N per-row monster requests is not.
const MONSTER_FANOUT_CAP = 12;

interface LiveFieldRowProps {
  field: FieldData;
  enabled: boolean;
}

function LiveFieldRow({ field, enabled }: LiveFieldRowProps) {
  const { worldId, channelId, mapId, instanceId, characterCount } =
    field.attributes;
  const { data: monsters, error: monstersError } = useLiveMonsters(
    worldId,
    channelId,
    mapId,
    instanceId,
    enabled,
  );
  const monsterCount =
    enabled && !monstersError && monsters ? monsters.length : "—";

  return (
    <TableRow>
      <TableCell>{worldId}</TableCell>
      <TableCell>{channelId}</TableCell>
      <TableCell>
        <Link
          to={`/fields/${worldId}/${channelId}/${mapId}/${instanceId}`}
          className="font-mono text-xs underline"
        >
          {instanceId}
        </Link>
      </TableCell>
      <TableCell>{characterCount}</TableCell>
      <TableCell>{monsterCount}</TableCell>
    </TableRow>
  );
}

interface LiveFieldsSectionProps {
  mapId: string;
}

export function LiveFieldsSection({ mapId }: LiveFieldsSectionProps) {
  const { data: fields, error } = useFieldsForMap(mapId);

  const rows = fields ?? [];
  const overCap = rows.length > MONSTER_FANOUT_CAP;

  return (
    <section className="space-y-2">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Live Fields</h2>
        <Link
          to={`/fields?map=${mapId}`}
          className="text-sm underline text-muted-foreground"
        >
          View all in Fields
        </Link>
      </div>

      {error && (
        <p className="text-sm text-destructive">Failed to load live fields.</p>
      )}

      {!error && rows.length === 0 && (
        <p className="text-sm text-muted-foreground">
          No live fields for this map.
        </p>
      )}

      {!error && rows.length > 0 && (
        <>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>World</TableHead>
                <TableHead>Channel</TableHead>
                <TableHead>Instance</TableHead>
                <TableHead>Characters</TableHead>
                <TableHead>Live Monsters</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((field, index) => (
                <LiveFieldRow
                  key={field.id}
                  field={field}
                  enabled={index < MONSTER_FANOUT_CAP}
                />
              ))}
            </TableBody>
          </Table>
          {overCap && (
            <p className="text-sm text-muted-foreground">
              Showing monster counts for the first {MONSTER_FANOUT_CAP} fields.
            </p>
          )}
        </>
      )}
    </section>
  );
}
