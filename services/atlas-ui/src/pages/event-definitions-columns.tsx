import { Link } from "react-router-dom";
import { type DataTableColumnDef } from "@/components/data-table-features";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import type { EventDefinition } from "@/types/models/events";

interface ColumnProps {
  /** Active occurrence count per definition id, from a per-definition `eventsService.getOccurrences` call. */
  activeCounts: Record<string, number>;
  /** Id currently being toggled, so its switch can be disabled mid-flight. */
  pendingToggleId: string | null;
  onToggleEnabled: (id: string, name: string, next: boolean) => void;
}

export const getColumns = ({
  activeCounts,
  pendingToggleId,
  onToggleEnabled,
}: ColumnProps): DataTableColumnDef<EventDefinition>[] => [
  {
    accessorKey: "attributes.name",
    header: "Name",
    cell: ({ row }) => (
      <span className="font-medium">{row.original.attributes.name}</span>
    ),
  },
  {
    accessorKey: "attributes.type",
    header: "Type",
    cell: ({ row }) => (
      <span className="font-mono text-sm">{row.original.attributes.type}</span>
    ),
  },
  {
    id: "occurrences",
    header: "Occurrences",
    // FR-UI4: "enabled" must never read as "occurring". This reads
    // `singleOccurrence` off the definition resource — NEVER `type` — so a
    // third event type renders correctly here with no edit to this file. A
    // definition that can have many concurrent occurrences always shows a
    // count linking to the filtered occurrence list, never a single live
    // state; only a genuinely single-occurrence definition may show live
    // state, because at most one occurrence of it can ever exist.
    cell: ({ row }) => {
      const def = row.original;
      const count = activeCounts[def.id] ?? 0;
      if (def.attributes.singleOccurrence) {
        return count > 0 ? (
          <Badge variant="secondary">Active now</Badge>
        ) : (
          <span className="text-sm text-muted-foreground">Not active</span>
        );
      }
      return (
        <Link
          to={`/events/occurrences?definitionId=${def.id}`}
          className="text-sm text-primary hover:underline"
        >
          {count} active
        </Link>
      );
    },
  },
  {
    id: "enabled",
    header: "Enabled",
    cell: ({ row }) => {
      const def = row.original;
      const { id } = def;
      const { name, enabled } = def.attributes;
      return (
        <Switch
          checked={enabled}
          disabled={pendingToggleId === id}
          aria-label={`${enabled ? "Disable" : "Enable"} ${name}`}
          onCheckedChange={(next) => onToggleEnabled(id, name, next)}
        />
      );
    },
  },
];
