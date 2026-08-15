/**
 * Row/column shape for EventOccurrencesPage.
 *
 * This deliberately does NOT use `DataTableColumnDef`/`DataTableWrapper`
 * (the shared TanStack table used elsewhere, e.g. `event-definitions-columns.tsx`):
 * FR-UI5 requires active occurrences to be "readily distinguishable" from
 * historical ones — a per-row `data-testid` and a per-row class — and the
 * shared `DataTable` component (`@/components/data-table.tsx`) renders every
 * `<TableRow>` itself with no row-level testid/className hook. A plain
 * column-descriptor + table here keeps that per-row control without
 * modifying the shared table for one page.
 */

import type { ReactNode } from "react";
import type { EventOccurrence } from "@/types/models/events";
import { formatDate } from "@/lib/utils/coupons";

/** Mirrors event/occurrence.StateActive (event/occurrence/model.go). */
export const ACTIVE_STATE = "ACTIVE";

export interface OccurrenceColumn {
  header: string;
  cell: (occurrence: EventOccurrence) => ReactNode;
}

/**
 * Best-effort "wN chM" scope summary. `context` is opaque JSON
 * (event/occurrence/rest.go keeps it a raw `json.RawMessage` and worldId/
 * channelId are DB-only columns never serialized onto the wire model), so
 * this only renders a scope when the occurrence's own context happens to
 * carry numeric `worldId`/`channelId` keys — many event types do, since it's
 * how they scope themselves, but it isn't guaranteed for every type.
 */
function scopeSummary(context: unknown): string {
  if (context && typeof context === "object") {
    const c = context as Record<string, unknown>;
    if (typeof c.worldId === "number" && typeof c.channelId === "number") {
      return `w${c.worldId} ch${c.channelId}`;
    }
  }
  return "—";
}

export const occurrenceColumns: OccurrenceColumn[] = [
  {
    header: "Id",
    cell: (occ) => (
      <span className="font-mono text-xs">{occ.id.slice(0, 8)}</span>
    ),
  },
  {
    header: "Type",
    cell: (occ) => (
      <span className="font-mono text-sm">{occ.attributes.type}</span>
    ),
  },
  {
    header: "State",
    cell: (occ) => occ.attributes.state,
  },
  {
    header: "Stage",
    cell: (occ) => occ.attributes.stage,
  },
  {
    header: "Scope",
    cell: (occ) => scopeSummary(occ.attributes.context),
  },
  {
    header: "Started",
    cell: (occ) => formatDate(occ.attributes.startedAt) ?? "—",
  },
  {
    header: "Completed",
    cell: (occ) => formatDate(occ.attributes.completedAt) ?? "—",
  },
  {
    header: "Reason",
    cell: (occ) => occ.attributes.completionReason ?? "—",
  },
];
