import { Card, CardContent } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { MapObjectData } from "@/services/api/map-entities.service";

export interface TrackedObject {
  id: string; // "{KIND}:{name}"
  kind: string;
  name: string;
  state: number;
}

interface FieldObjectsTabProps {
  defined?: MapObjectData[] | undefined;
  definedError?: Error | undefined;
  tracked?: TrackedObject[] | undefined;
  trackedError?: Error | undefined;
}

/**
 * FR-32..FR-33, FR-38: the field-detail Map Objects tab. Merges two
 * independent sources on the id both already share (`{KIND}:{name}`):
 * `defined` (this map's declared objects, from `atlas-data`) and `tracked`
 * (this field's live object state, from [278] — Task 22 supplies it; until
 * then the caller passes `undefined`, a real "no state tracked" field, not a
 * stub). Tracked objects render first with their current state; the
 * remainder of `defined` (definition ids minus tracked ids) render under a
 * divider at default state. Both sources empty is a normal empty state
 * (FR-32), never a 404.
 */
export function FieldObjectsTab({
  defined,
  definedError,
  tracked,
  trackedError,
}: FieldObjectsTabProps) {
  if (definedError || trackedError) {
    return (
      <Card>
        <CardContent className="pt-6">
          <p className="text-sm text-destructive">
            Failed to load map objects.
          </p>
        </CardContent>
      </Card>
    );
  }

  // `defined` is this map's declared objects; a still-in-flight query is
  // the loading state (`tracked` legitimately stays undefined until Task 22
  // wires it up, per the docstring above, so it never gates loading).
  if (defined === undefined) {
    return (
      <Card>
        <CardContent className="pt-6">
          <p className="text-sm text-muted-foreground">
            Loading map objects...
          </p>
        </CardContent>
      </Card>
    );
  }

  const trackedObjects = tracked ?? [];
  const trackedIds = new Set(trackedObjects.map((object) => object.id));
  const untrackedObjects = defined.filter(
    (object) => !trackedIds.has(object.id),
  );

  if (trackedObjects.length === 0 && untrackedObjects.length === 0) {
    return (
      <Card>
        <CardContent className="pt-6">
          <p className="text-sm text-muted-foreground">
            No map objects are declared or tracked for this field.
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        Object names are pass-through: the server does not validate a name
        against the map's declared objects (FR-38).
      </p>

      {trackedObjects.length > 0 && (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Kind</TableHead>
              <TableHead>Name</TableHead>
              <TableHead>State</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {trackedObjects.map((object) => (
              <TableRow key={object.id}>
                <TableCell>{object.kind}</TableCell>
                <TableCell>{object.name}</TableCell>
                <TableCell>{object.state}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {untrackedObjects.length > 0 && (
        <div className="space-y-2">
          <p className="text-sm text-muted-foreground">
            Defined on the map, no state tracked in this field
          </p>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Kind</TableHead>
                <TableHead>Name</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {untrackedObjects.map((object) => (
                <TableRow key={object.id}>
                  <TableCell>{object.attributes.kind}</TableCell>
                  <TableCell>{object.attributes.name}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
