import { useMemo } from "react";

import {
  formatTimeOfDay,
  nowUtcTimeOfDayMs,
  segmentSpan,
  timelineHalfWindowMs,
  utcTimeOfDayMs,
} from "@/components/features/transports/transport-format";
import { cn } from "@/lib/utils";
import type { TripSchedule } from "@/types/models/transport";

export interface TimelineLane {
  label: string;
  trips: TripSchedule[];
  /** The route being viewed, drawn with more weight than its partner. */
  emphasised?: boolean;
}

interface VesselTimelineProps {
  lanes: TimelineLane[];
  /** Epoch milliseconds; the strip is centred on this instant's UTC time of day. */
  nowEpochMs: number;
}

const WIDTH = 720;
const LANE_HEIGHT = 34;
const LANE_GAP = 12;
/** Room above each lane rail for its route-name caption. */
const LANE_LABEL_HEIGHT = 15;
const TOP_PAD = 18;

const SEGMENT_STYLE = {
  open: {
    className: "fill-emerald-500/70",
    swatchClassName: "bg-emerald-500/70",
    label: "boarding open",
  },
  locked: {
    className: "fill-amber-500/70",
    swatchClassName: "bg-amber-500/70",
    label: "boarding closed",
  },
  transit: {
    className: "fill-sky-500/70",
    swatchClassName: "bg-sky-500/70",
    label: "in transit",
  },
} as const;

const SEGMENT_KINDS = ["open", "locked", "transit"] as const;

/**
 * A windowed strip of trips around now — one lane per route, two lanes when
 * the caller passes both sides of a shared vessel so the alternation and the
 * turnaround gap are visible.
 *
 * Trip boundaries are UTC times of day (the schedule is anchored on UTC
 * midnight and its date component is stale), so everything here — including
 * the NOW marker — is positioned in milliseconds since UTC midnight via
 * `nowUtcTimeOfDayMs`/`utcTimeOfDayMs`/`segmentSpan`/`timelineHalfWindowMs`
 * (Task 10). Times are labelled UTC for the same reason, and rendered only
 * through `formatTimeOfDay`, which never carries a date. "Now" is whatever
 * instant the caller passes as `nowEpochMs` — this component reads no clock
 * of its own; Task 18's page sources it from the shared `useClock()`.
 *
 * The whole strip is one `role="img"` figure: position and colour are the
 * *only* way the SVG shows which trip is boarding/closed/in-transit and where
 * now sits among them, so — unlike a decorative graphic whose meaning is
 * already duplicated by visible text — that meaning has to live in the
 * accessible name itself. The name spells out, per lane, each trip's board /
 * close / depart / arrive times and where "now" falls, so a screen-reader
 * user gets the same information a sighted user reads off the strip. A lane
 * with no trips renders an explicit worded state (visually, as an SVG
 * caption, and in the accessible name) rather than a blank rail that could
 * read as a failed fetch.
 *
 * Each lane is captioned in place with its route name, and the legend below
 * carries a colour swatch per phase — the strip encodes phase entirely in
 * colour and lane identity entirely in vertical position, so neither can be
 * left to a bare list of words.
 */
export function VesselTimeline({ lanes, nowEpochMs }: VesselTimelineProps) {
  const nowMs = nowUtcTimeOfDayMs(nowEpochMs);
  // Reuses formatTimeOfDay (rather than re-deriving HH:MM padding) by
  // handing it an ISO string built from the same epoch instant.
  const nowLabel = formatTimeOfDay(new Date(nowEpochMs).toISOString());

  const halfWindowMs = useMemo(
    () =>
      timelineHalfWindowMs(
        lanes.flatMap((lane) =>
          lane.trips.map((trip) =>
            utcTimeOfDayMs(trip.attributes.boardingOpen),
          ),
        ),
      ),
    [lanes],
  );

  const height =
    TOP_PAD + lanes.length * (LANE_LABEL_HEIGHT + LANE_HEIGHT + LANE_GAP);

  const ariaLabel = useMemo(
    () =>
      `Trip timeline, times UTC. Now ${nowLabel}, window plus or minus ${Math.round(
        halfWindowMs / 60_000,
      )} minutes. ${lanes.map(laneAriaPhrase).join(" ")}`,
    [lanes, nowLabel, halfWindowMs],
  );

  return (
    <div className="space-y-2">
      <div className="overflow-x-auto">
        <svg
          role="img"
          aria-label={ariaLabel}
          viewBox={`0 0 ${WIDTH} ${height}`}
          width="100%"
          style={{ minWidth: `${WIDTH / 2}px` }}
        >
          {lanes.map((lane, laneIndex) => {
            const blockTop =
              TOP_PAD +
              laneIndex * (LANE_LABEL_HEIGHT + LANE_HEIGHT + LANE_GAP);
            const y = blockTop + LANE_LABEL_HEIGHT;
            const isEmpty = lane.trips.length === 0;
            return (
              <g key={lane.label} data-empty-lane={isEmpty ? "" : undefined}>
                {/*
                  Which rail is which route has to be readable off the chart
                  itself — a legend listing two names in the order they happen
                  to be passed does not say which lane belongs to which.
                */}
                <text
                  data-lane-label={lane.label}
                  x={0}
                  y={blockTop + 10}
                  fontSize={11}
                  className={
                    lane.emphasised
                      ? "fill-foreground font-medium"
                      : "fill-muted-foreground"
                  }
                >
                  {lane.label}
                </text>
                <rect
                  x={0}
                  y={y}
                  width={WIDTH}
                  height={LANE_HEIGHT}
                  className={cn("fill-muted", isEmpty && "fill-muted/50")}
                  rx={4}
                  stroke={isEmpty ? "currentColor" : undefined}
                  strokeOpacity={isEmpty ? 0.3 : undefined}
                  strokeDasharray={isEmpty ? "4 4" : undefined}
                />
                {isEmpty ? (
                  <text
                    x={WIDTH / 2}
                    y={y + LANE_HEIGHT / 2 + 4}
                    textAnchor="middle"
                    fontSize={11}
                    className="fill-muted-foreground"
                  >
                    No trips scheduled in this window
                  </text>
                ) : (
                  lane.trips.map((trip) => (
                    <TripSegments
                      key={trip.id}
                      trip={trip}
                      y={y}
                      nowMs={nowMs}
                      halfWindowMs={halfWindowMs}
                      emphasised={lane.emphasised ?? false}
                    />
                  ))
                )}
              </g>
            );
          })}

          <line
            data-now-marker=""
            x1={WIDTH / 2}
            y1={0}
            x2={WIDTH / 2}
            y2={height}
            className="stroke-foreground"
            strokeWidth={1.5}
            strokeDasharray="3 3"
          />
          <text
            x={WIDTH / 2 + 4}
            y={12}
            className="fill-foreground"
            fontSize={10}
          >
            NOW
          </text>
        </svg>
      </div>

      {/*
        Each key carries the swatch it names. Naming the three phases without
        showing their colours leaves the strip's only encoding unexplained;
        the lane names are drawn on the lanes themselves, not repeated here.
      */}
      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
        {SEGMENT_KINDS.map((kind) => (
          <span key={kind} className="inline-flex items-center gap-1.5">
            <span
              aria-hidden="true"
              data-legend-swatch={kind}
              className={cn(
                "inline-block h-2.5 w-3.5 rounded-[2px]",
                SEGMENT_STYLE[kind].swatchClassName,
              )}
            />
            {SEGMENT_STYLE[kind].label}
          </span>
        ))}
        <span>
          times UTC
          {lanes[0]?.trips[0]
            ? ` · first trip boards ${formatTimeOfDay(
                lanes[0].trips[0].attributes.boardingOpen,
              )}`
            : ""}
        </span>
      </div>
    </div>
  );
}

/** Per-lane sentence for the SVG's accessible name — see the component doc. */
function laneAriaPhrase(lane: TimelineLane): string {
  if (lane.trips.length === 0) {
    return `${lane.label}: no trips scheduled in this window.`;
  }
  const trips = lane.trips
    .map((trip) => {
      const { boardingOpen, boardingClosed, departure, arrival } =
        trip.attributes;
      return `boards ${formatTimeOfDay(boardingOpen)}, closes ${formatTimeOfDay(
        boardingClosed,
      )}, departs ${formatTimeOfDay(departure)}, arrives ${formatTimeOfDay(
        arrival,
      )}`;
    })
    .join("; ");
  return `${lane.label}: ${trips}.`;
}

function TripSegments({
  trip,
  y,
  nowMs,
  halfWindowMs,
  emphasised,
}: {
  trip: TripSchedule;
  y: number;
  nowMs: number;
  halfWindowMs: number;
  emphasised: boolean;
}) {
  const { boardingOpen, boardingClosed, departure, arrival } = trip.attributes;

  const parts = [
    {
      kind: "open" as const,
      start: utcTimeOfDayMs(boardingOpen),
      end: utcTimeOfDayMs(boardingClosed),
    },
    {
      kind: "locked" as const,
      start: utcTimeOfDayMs(boardingClosed),
      end: utcTimeOfDayMs(departure),
    },
    {
      kind: "transit" as const,
      start: utcTimeOfDayMs(departure),
      end: utcTimeOfDayMs(arrival),
    },
  ];

  const laneY = emphasised ? y + 4 : y + 10;
  const laneHeight = emphasised ? LANE_HEIGHT - 8 : LANE_HEIGHT - 20;

  return (
    <>
      {parts.map((part) => {
        const span = segmentSpan(part.start, part.end, nowMs, halfWindowMs);
        if (!span || span.right <= span.left) return null;
        return (
          <rect
            key={`${trip.id}-${part.kind}`}
            data-segment={part.kind}
            x={span.left * WIDTH}
            y={laneY}
            width={(span.right - span.left) * WIDTH}
            height={laneHeight}
            rx={2}
            className={SEGMENT_STYLE[part.kind].className}
          >
            <title>
              {`${SEGMENT_STYLE[part.kind].label} ${formatTimeOfDay(
                part.kind === "open"
                  ? boardingOpen
                  : part.kind === "locked"
                    ? boardingClosed
                    : departure,
              )} UTC`}
            </title>
          </rect>
        );
      })}
    </>
  );
}
