import { Link } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
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
import type { FieldData } from "@/services/api/fields.service";

export interface FieldsResultTableProps {
  fields: FieldData[];
  /** `undefined` = still loading or failed; degrade to the numeric id. */
  mapNames: Record<number, string | undefined>;
}

/**
 * FR-14: Channel, Map (name badge, linking to the field), Instance,
 * Characters. World is deliberately not a column — it's already the filter.
 * Channel is shown 1-indexed (display only; the value in the link/query
 * stays 0-based, bug-fields-ui item 15).
 */
export function FieldsResultTable({
  fields,
  mapNames,
}: FieldsResultTableProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>
          Results
          <span className="ml-2 text-muted-foreground font-normal">
            ({fields.length} {fields.length === 1 ? "field" : "fields"})
          </span>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Channel</TableHead>
              <TableHead>Map</TableHead>
              <TableHead>Instance</TableHead>
              <TableHead>Characters</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {fields.map((field) => {
              const { worldId, channelId, mapId, instanceId, characterCount } =
                field.attributes;
              const mapName = mapNames[mapId];
              const href = `/fields?world=${worldId}&channel=${channelId}&map=${mapId}&instance=${instanceId}`;

              return (
                <TableRow key={field.id}>
                  <TableCell>{channelId + 1}</TableCell>
                  <TableCell>
                    <TooltipProvider>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Link to={href}>
                            <Badge variant="secondary">
                              {mapName ?? mapId}
                            </Badge>
                          </Link>
                        </TooltipTrigger>
                        <TooltipContent copyable>
                          <p>{mapId}</p>
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  </TableCell>
                  <TableCell className="font-mono text-xs">
                    {instanceId}
                  </TableCell>
                  <TableCell>{characterCount}</TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
