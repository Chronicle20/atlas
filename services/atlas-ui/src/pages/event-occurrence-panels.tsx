/**
 * Per-event-type detail panels for EventOccurrenceDetailPage.
 *
 * `context` on an occurrence is opaque per-event-type JSON to the backend
 * (event/occurrence/rest.go keeps it a raw `json.RawMessage`) — these panels
 * are the only place its shape is interpreted, so each one is scoped to the
 * specific event type it renders and the field names come straight from the
 * Go source, not a guess:
 *   - CRIMSON_BALROG: events/crimsonbalrog/evaluate.go:29-38
 *   - ANNIVERSARY:    events/anniversary/config.go:56-59
 *
 * EventOccurrenceDetailPage looks these up via a **component lookup with a
 * generic fallback** (FR-X3, `detailPanels` there) — not a switch that must
 * be edited for the page to keep working: adding a third event type needs no
 * change to EventOccurrenceDetailPage's structure, only (optionally) a new
 * panel here plus one entry in that lookup. An occurrence whose type has no
 * bespoke entry falls back to `GenericContextPanel` — the raw context as
 * formatted JSON. (`detailPanels` itself lives in EventOccurrenceDetailPage,
 * not this file, because react-refresh/only-export-components forbids mixing
 * a plain-object export with component exports in the same module.)
 */

import type { ReactNode } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatDate } from "@/lib/utils/coupons";
import type { EventOccurrence } from "@/types/models/events";

export interface EventTypePanelProps {
  occurrence: EventOccurrence;
}

interface CrimsonBalrogAttackMap {
  mapId: number;
  spawnPositions: { x: number; y: number }[];
}

interface CrimsonBalrogVisual {
  name: string;
  showState: string;
  showSubState: string;
  hideState: string;
  hideSubState: string;
}

/** Mirrors events/crimsonbalrog/evaluate.go:29-38's occurrence context. */
interface CrimsonBalrogContext {
  routeId: string;
  voyageId: string;
  worldId: number;
  channelId: number;
  attackMaps: CrimsonBalrogAttackMap[];
  relatedMapIds: number[];
  monsterId: number;
  monsterCount: number;
  backgroundMusic: string;
  visual: CrimsonBalrogVisual;
}

function isCrimsonBalrogContext(
  context: unknown,
): context is CrimsonBalrogContext {
  return (
    !!context &&
    typeof context === "object" &&
    "monsterId" in context &&
    "voyageId" in context
  );
}

function DetailRow({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="flex justify-between gap-4 text-sm">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-mono">{value}</span>
    </div>
  );
}

export function CrimsonBalrogPanel({ occurrence }: EventTypePanelProps) {
  const context = occurrence.attributes.context;
  if (!isCrimsonBalrogContext(context)) {
    return <GenericContextPanel occurrence={occurrence} />;
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">Scope</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          <div className="text-sm font-mono">Voyage: {context.voyageId}</div>
          <DetailRow label="Route" value={context.routeId} />
          <DetailRow label="World" value={context.worldId} />
          <DetailRow label="Channel" value={context.channelId} />
          <DetailRow
            label="Related maps"
            value={context.relatedMapIds.join(", ") || "—"}
          />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">Monster</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          <div className="flex justify-between gap-4 text-sm">
            <span className="text-muted-foreground">Monster Id</span>
            <span className="font-mono">{context.monsterId}</span>
          </div>
          <DetailRow label="Count" value={context.monsterCount} />
          <DetailRow label="Background music" value={context.backgroundMusic} />
          <DetailRow label="Visual" value={context.visual.name} />
        </CardContent>
      </Card>
      {context.attackMaps.length > 0 && (
        <Card className="md:col-span-2">
          <CardHeader>
            <CardTitle className="text-sm font-medium">Attack Maps</CardTitle>
          </CardHeader>
          <CardContent className="space-y-1 text-sm">
            {context.attackMaps.map((map) => (
              <div key={map.mapId} className="flex justify-between">
                <span className="font-mono">{map.mapId}</span>
                <span className="text-muted-foreground">
                  {map.spawnPositions.length} spawn position
                  {map.spawnPositions.length === 1 ? "" : "s"}
                </span>
              </div>
            ))}
          </CardContent>
        </Card>
      )}
    </div>
  );
}

/** Mirrors events/anniversary/config.go:56-59's occurrence context. */
interface AnniversaryContext {
  scheduledEnd: string;
  expMultiplier: number;
  dropMultiplier: number;
  buffSourceId: number;
}

function isAnniversaryContext(context: unknown): context is AnniversaryContext {
  return (
    !!context &&
    typeof context === "object" &&
    "expMultiplier" in context &&
    "scheduledEnd" in context
  );
}

export function AnniversaryPanel({ occurrence }: EventTypePanelProps) {
  const context = occurrence.attributes.context;
  if (!isAnniversaryContext(context)) {
    return <GenericContextPanel occurrence={occurrence} />;
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium">
          Schedule &amp; Multipliers
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        <div className="text-sm">
          Scheduled End:{" "}
          <span className="font-mono">
            {formatDate(context.scheduledEnd) ?? "—"}
          </span>
        </div>
        <div className="flex justify-between gap-4 text-sm">
          <span className="text-muted-foreground">Exp Multiplier</span>
          <span className="font-mono">{context.expMultiplier.toFixed(1)}x</span>
        </div>
        <div className="flex justify-between gap-4 text-sm">
          <span className="text-muted-foreground">Drop Multiplier</span>
          <span className="font-mono">
            {context.dropMultiplier.toFixed(1)}x
          </span>
        </div>
        <DetailRow label="Buff Source" value={context.buffSourceId} />
      </CardContent>
    </Card>
  );
}

/**
 * The raw context, formatted as JSON — used two ways by
 * EventOccurrenceDetailPage:
 *   - as the FR-X3 fallback for any occurrence type with no bespoke panel
 *     above, so an unrecognized `type` still renders something useful;
 *   - as the always-present FR-UI7 "full context" view for a REGISTERED
 *     type too, since a bespoke panel (FR-UI8) supplements the raw context
 *     rather than replacing it.
 */
export function GenericContextPanel({ occurrence }: EventTypePanelProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium">Context</CardTitle>
      </CardHeader>
      <CardContent>
        <pre
          data-testid="occurrence-context-json"
          className="text-xs whitespace-pre-wrap break-all"
        >
          {JSON.stringify(occurrence.attributes.context, null, 2)}
        </pre>
      </CardContent>
    </Card>
  );
}
