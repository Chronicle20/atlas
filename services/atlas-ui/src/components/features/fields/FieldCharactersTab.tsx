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
import { useTenant } from "@/context/tenant-context";
import { useCharacter } from "@/lib/hooks/api/useCharacters";
import { useJobNameLookup } from "@/lib/hooks/api/useJobGraph";

export interface FieldCharactersTabProps {
  characterIds: string[];
}

/**
 * FR-23..FR-27: the field-detail Characters tab. The `characters` endpoint
 * (`map` package `rest.go`) returns ids only, no attributes, so each row
 * enriches itself via `useCharacter` (FR-24) — React Query dedupes and
 * caches per character id, and one row's failure never blocks another's.
 * `useJobNameLookup` is a hook: call it once here and pass the resolver
 * down, per its own doc-comment.
 */
export function FieldCharactersTab({ characterIds }: FieldCharactersTabProps) {
  const jobName = useJobNameLookup();

  if (characterIds.length === 0) {
    return (
      <Card>
        <CardContent className="pt-6">
          <p className="text-sm text-muted-foreground">
            No characters are currently in this field.
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Name</TableHead>
          <TableHead>Character ID</TableHead>
          <TableHead>Level</TableHead>
          <TableHead>Job</TableHead>
          <TableHead>Position</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {characterIds.map((id) => (
          <FieldCharacterRow key={id} characterId={id} jobName={jobName} />
        ))}
      </TableBody>
    </Table>
  );
}

interface FieldCharacterRowProps {
  characterId: string;
  jobName: (id: number) => string;
}

function FieldCharacterRow({ characterId, jobName }: FieldCharacterRowProps) {
  const { activeTenant } = useTenant();
  const query = useCharacter(activeTenant!, characterId);
  const character = query.data;

  if (!character) {
    // Pending or errored enrichment (FR-24): the row still renders with the
    // raw id, never unmounts, and never trips an error boundary.
    return (
      <TableRow>
        <TableCell>{characterId}</TableCell>
        <TableCell>{characterId}</TableCell>
        <TableCell>—</TableCell>
        <TableCell>—</TableCell>
        <TableCell>—</TableCell>
      </TableRow>
    );
  }

  const { name, level, jobId, x, y } = character.attributes;

  return (
    <TableRow>
      <TableCell>
        <Link to={`/characters/${characterId}`} className="underline">
          {name}
        </Link>
      </TableCell>
      <TableCell>{characterId}</TableCell>
      <TableCell>{level}</TableCell>
      <TableCell>{jobName(jobId)}</TableCell>
      <TableCell>{`(${x}, ${y})`}</TableCell>
    </TableRow>
  );
}
