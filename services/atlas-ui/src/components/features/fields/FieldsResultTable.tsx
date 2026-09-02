import { Link } from "react-router-dom";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { FieldData } from "@/services/api/fields.service";

export interface FieldsResultTableProps {
  fields: FieldData[];
  /** `undefined` = still loading or failed; degrade to the numeric id. */
  mapNames: Record<number, string | undefined>;
}

/**
 * FR-14: Map (name + id, linking to the field), Channel, Instance,
 * Characters. World is deliberately not a column — it's already the filter.
 */
export function FieldsResultTable({
  fields,
  mapNames,
}: FieldsResultTableProps) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Map</TableHead>
          <TableHead>Channel</TableHead>
          <TableHead>Instance</TableHead>
          <TableHead>Characters</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {fields.map((field) => {
          const { worldId, channelId, mapId, instanceId, characterCount } =
            field.attributes;
          const mapName = mapNames[mapId];

          return (
            <TableRow key={field.id}>
              <TableCell>
                <Link
                  to={`/fields/${worldId}/${channelId}/${mapId}/${instanceId}`}
                  className="font-mono text-xs underline"
                >
                  {mapName ? `${mapName} (${mapId})` : mapId}
                </Link>
              </TableCell>
              <TableCell>{channelId}</TableCell>
              <TableCell className="font-mono text-xs">{instanceId}</TableCell>
              <TableCell>{characterCount}</TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}
