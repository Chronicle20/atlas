import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { MapObjectData } from "@/services/api/map-entities.service";

interface MapObjectsTableProps {
  objects?: MapObjectData[];
  error?: Error;
}

export function MapObjectsTable({ objects, error }: MapObjectsTableProps) {
  if (error) {
    return <p className="text-sm text-destructive">Failed to load objects.</p>;
  }

  if (objects === undefined) {
    return <p className="text-sm text-muted-foreground">Loading objects...</p>;
  }

  if (objects.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        No named objects on this map.
      </p>
    );
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Kind</TableHead>
          <TableHead>Name</TableHead>
          <TableHead>WZ Source</TableHead>
          <TableHead>Path</TableHead>
          <TableHead>Position</TableHead>
          <TableHead>Layer</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {objects.map((object) => (
          <TableRow key={object.id}>
            <TableCell>{object.attributes.kind}</TableCell>
            <TableCell>{object.attributes.name}</TableCell>
            <TableCell>{object.attributes.objectSource}</TableCell>
            <TableCell className="font-mono text-xs">
              {object.attributes.l0}/{object.attributes.l1}/
              {object.attributes.l2}
            </TableCell>
            <TableCell className="font-mono">
              ({object.attributes.x}, {object.attributes.y},{" "}
              {object.attributes.z})
            </TableCell>
            <TableCell>{object.attributes.layer}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
