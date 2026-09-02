import { Link } from "react-router-dom";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { SurfaceKindBadge } from "@/components/features/maps/SurfaceKindBadge";
import { useFields } from "@/lib/hooks/api/useFields";

// FR-17/FR-18: the runtime read model locator. This task ships a working
// list driven by useFields({}) — the filter bar (world/channel/map) lands
// in Task 16 on top of this same page.
export function FieldsPage() {
  const { data: fields, error, isLoading } = useFields({});
  const rows = fields ?? [];

  return (
    <div className="flex flex-col flex-1 min-h-0 overflow-y-auto space-y-6 p-10 pb-16">
      <div className="flex items-center gap-3">
        <h1 className="text-2xl font-semibold">Fields</h1>
        <SurfaceKindBadge kind="runtime" />
      </div>

      {isLoading && (
        <p className="text-sm text-muted-foreground">Loading fields…</p>
      )}

      {error && (
        <p className="text-sm text-destructive">Failed to load fields.</p>
      )}

      {!isLoading && !error && rows.length === 0 && (
        <p className="text-sm text-muted-foreground">No live fields.</p>
      )}

      {!isLoading && !error && rows.length > 0 && (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>World</TableHead>
              <TableHead>Channel</TableHead>
              <TableHead>Map</TableHead>
              <TableHead>Instance</TableHead>
              <TableHead>Characters</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((field) => {
              const { worldId, channelId, mapId, instanceId, characterCount } =
                field.attributes;
              return (
                <TableRow key={field.id}>
                  <TableCell>{worldId}</TableCell>
                  <TableCell>{channelId}</TableCell>
                  <TableCell>{mapId}</TableCell>
                  <TableCell>
                    <Link
                      to={`/fields/${worldId}/${channelId}/${mapId}/${instanceId}`}
                      className="font-mono text-xs underline"
                    >
                      {instanceId}
                    </Link>
                  </TableCell>
                  <TableCell>{characterCount}</TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      )}
    </div>
  );
}
