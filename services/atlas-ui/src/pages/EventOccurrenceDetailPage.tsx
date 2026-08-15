/**
 * Event occurrence detail: header summary, the per-type panel (FR-UI8), and
 * the full transition history (FR-UI7).
 *
 * Backed by atlas-events (task-231 Tasks 13/16, event/occurrence/resource.go):
 *   GET /api/events/occurrences/{occurrenceId}
 *
 * `eventsService.getOccurrence` (Task 35) side-loads the occurrence's
 * `event-occurrence-transitions` and flattens them onto `.transitions`.
 */

import type { ComponentType } from "react";
import { useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
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
import { PageLoader } from "@/components/common/PageLoader";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";
import { EmptyState } from "@/components/common/EmptyState";
import { eventsService } from "@/services/api/events.service";
import { formatDate } from "@/lib/utils/coupons";
import {
  AnniversaryPanel,
  CrimsonBalrogPanel,
  GenericContextPanel,
  type EventTypePanelProps,
} from "./event-occurrence-panels";

/** Mirrors event/occurrence.StateActive (event/occurrence/model.go). */
const ACTIVE_STATE = "ACTIVE";

/**
 * Component lookup with a generic fallback (FR-X3), not a switch that must
 * be edited for the page to keep working: adding a third event type needs no
 * change to this page, only a new panel in event-occurrence-panels.tsx plus
 * one entry here. Keyed by the domain event type
 * (`occurrence.attributes.type`, e.g. "CRIMSON_BALROG") — not
 * `occurrence.type`, which is the fixed JSON:API resource-type string
 * ("event-occurrences") and identical across every occurrence.
 */
const detailPanels: Record<string, ComponentType<EventTypePanelProps>> = {
  CRIMSON_BALROG: CrimsonBalrogPanel,
  ANNIVERSARY: AnniversaryPanel,
};

export function EventOccurrenceDetailPage() {
  const params = useParams();
  const id = params.id as string;

  const occurrenceQuery = useQuery({
    queryKey: ["events", "occurrence", id],
    queryFn: () => eventsService.getOccurrence(id),
  });

  if (occurrenceQuery.isLoading) return <PageLoader />;
  if (occurrenceQuery.error || !occurrenceQuery.data) {
    return (
      <div className="p-10">
        <ErrorDisplay
          error={occurrenceQuery.error ?? "Event occurrence not found"}
          retry={() => void occurrenceQuery.refetch()}
        />
      </div>
    );
  }

  const occurrence = occurrenceQuery.data;
  const attrs = occurrence.attributes;
  const isActive = attrs.state === ACTIVE_STATE;

  // Component lookup with a generic fallback (FR-X3) — a third event type
  // needs no edit here, only an optional new entry in detailPanels.
  const Panel = detailPanels[attrs.type] ?? GenericContextPanel;

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="flex items-center gap-3">
        <h2 className="text-2xl font-bold tracking-tight">{attrs.type}</h2>
        <Badge variant={isActive ? "default" : "secondary"}>
          {attrs.state}
        </Badge>
        <span className="text-muted-foreground font-mono text-sm">
          {attrs.stage}
        </span>
        <span className="text-muted-foreground font-mono text-xs">
          #{occurrence.id}
        </span>
      </div>

      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4 text-sm">
        <div>
          <div className="text-muted-foreground">Started</div>
          <div>{formatDate(attrs.startedAt) ?? "—"}</div>
        </div>
        <div>
          <div className="text-muted-foreground">Completed</div>
          <div>{formatDate(attrs.completedAt) ?? "—"}</div>
        </div>
        <div>
          <div className="text-muted-foreground">Next Transition</div>
          <div>{formatDate(attrs.nextTransitionAt) ?? "—"}</div>
        </div>
        <div>
          <div className="text-muted-foreground">Completion Reason</div>
          <div>{attrs.completionReason ?? "—"}</div>
        </div>
      </div>

      <Panel occurrence={occurrence} />

      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">
            Transition History ({occurrence.transitions.length})
          </CardTitle>
        </CardHeader>
        <CardContent>
          {occurrence.transitions.length === 0 ? (
            <EmptyState
              title="No transitions"
              description="This occurrence has no recorded transitions yet."
            />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>From</TableHead>
                  <TableHead>To</TableHead>
                  <TableHead>Trigger</TableHead>
                  <TableHead>Occurred</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {occurrence.transitions.map((transition) => (
                  <TableRow key={transition.id}>
                    <TableCell className="font-mono text-sm">
                      {transition.fromStage}
                    </TableCell>
                    <TableCell className="font-mono text-sm">
                      {transition.toStage}
                    </TableCell>
                    <TableCell className="font-mono text-sm">
                      {transition.triggerType}
                      {transition.triggerReference
                        ? ` (${transition.triggerReference})`
                        : ""}
                    </TableCell>
                    <TableCell className="text-sm">
                      {formatDate(transition.occurredAt) ?? "—"}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
