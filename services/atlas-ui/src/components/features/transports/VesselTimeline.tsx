import { useMemo } from "react";

import {
  formatClockMs,
  formatTimeOfDay,
  formatTimeOfDayMs,
  nowUtcTimeOfDayMs,
  segmentSpan,
  timelineAxisTicks,
  timelineHalfWindowMs,
  utcTimeOfDayMs,
} from "@/components/features/transports/transport-format";
import { cn } from "@/lib/utils";
import type { TripSchedule } from "@/types/models/transport";

export interface TimelineLane {
  label: string;
  trips: TripSchedule[];
  /**
   * The route being viewed. Its caption is weighted and its partner's bars are
   * held back — the two lanes stay the same size either way, so bar height
   * never competes with duration for meaning.
   */
  emphasised?: boolean;
}

interface VesselTimelineProps {
  lanes: TimelineLane[];
  /** Epoch milliseconds; the strip is centred on this instant's UTC time of day. */
  nowEpochMs: number;
}

const WIDTH = 720;
const LANE_HEIGHT = 34;
/**
 * Inset of a trip's bars inside their rail. One value for every lane: bar
 * height is read as duration on a strip whose whole point is comparing two
 * routes' trips against each other, so it must not double as an emphasis
 * channel — a shorter partner bar reads as a shorter trip. Emphasis is the
 * lane caption's weight and the partner's segment opacity instead.
 */
const SEGMENT_INSET = 6;
const LANE_GAP = 12;
/** Room above each lane rail for its route-name caption. */
const LANE_LABEL_HEIGHT = 15;
const TOP_PAD = 18;
/** Room below the last rail for the time axis: rule, ticks, and hour labels. */
const AXIS_HEIGHT = 22;
/** Half the width a `HH:MM` label needs, used to keep the end ticks on-canvas. */
const AXIS_LABEL_HALF_WIDTH = 16;

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
 * left to a bare list of words. The third encoding, horizontal position, is
 * spelled out by the time axis under the last rail: round wall-clock ticks
 * across the window plus a NOW marker stamped with the second it is drawn at,
 * so "when" is legible without hovering a segment for its tooltip.
 */
export function VesselTimeline({ lanes, nowEpochMs }: VesselTimelineProps) {
  const nowMs = nowUtcTimeOfDayMs(nowEpochMs);
  // The marker carries seconds where the axis does not: it is the one time on
  // the strip that moves, and the countdowns beside it tick every second.
  const nowClockLabel = formatClockMs(nowMs);

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

  const axisY =
    TOP_PAD + lanes.length * (LANE_LABEL_HEIGHT + LANE_HEIGHT + LANE_GAP);
  const height = axisY + AXIS_HEIGHT;

  // Not memoised: the window slides with `nowMs`, so this is a fresh handful
  // of ticks on every tick of the clock either way.
  const ticks = timelineAxisTicks(nowMs, halfWindowMs);

  const ariaLabel = useMemo(
    () =>
      `Trip timeline, times UTC. Now ${formatTimeOfDayMs(
        nowUtcTimeOfDayMs(nowEpochMs),
      )}, window plus or minus ${Math.round(
        halfWindowMs / 60_000,
      )} minutes. ${lanes.map(laneAriaPhrase).join(" ")}`,
    [lanes, nowEpochMs, halfWindowMs],
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
                      // Only meaningful against a partner: a lone lane is
                      // never dimmed, whether or not the caller marked it.
                      dimmed={lanes.length > 1 && !lane.emphasised}
                    />
                  ))
                )}
              </g>
            );
          })}

          {/*
            The time axis. Without it the strip shows only that one block sits
            left of another — every absolute time stays locked inside a
            segment's tooltip. Rule and ticks take the border token and the
            labels the muted one, so the clock reads as ground and the trips
            stay the figure.
          */}
          <g data-time-axis="">
            <line
              x1={0}
              y1={axisY}
              x2={WIDTH}
              y2={axisY}
              className="stroke-border"
              strokeWidth={1}
            />
            {ticks.map((tick) => {
              const x = tick.fraction * WIDTH;
              const label = formatTimeOfDayMs(tick.ms);
              return (
                <g key={tick.ms}>
                  <line
                    x1={x}
                    y1={axisY}
                    x2={x}
                    y2={axisY + 4}
                    className="stroke-border"
                    strokeWidth={1}
                  />
                  {/*
                    The first and last ticks sit on the window's edges, where a
                    centred label would hang half off the canvas.
                  */}
                  <text
                    data-axis-tick={label}
                    x={Math.min(
                      WIDTH - AXIS_LABEL_HALF_WIDTH,
                      Math.max(AXIS_LABEL_HALF_WIDTH, x),
                    )}
                    y={axisY + 16}
                    textAnchor="middle"
                    fontSize={10}
                    className="fill-muted-foreground"
                  >
                    {label}
                  </text>
                </g>
              );
            })}
          </g>

          <line
            data-now-marker=""
            x1={WIDTH / 2}
            y1={0}
            x2={WIDTH / 2}
            y2={axisY + 4}
            className="stroke-foreground"
            strokeWidth={1.5}
            strokeDasharray="3 3"
          />
          <text
            data-now-label={nowClockLabel}
            x={WIDTH / 2 + 4}
            y={12}
            className="fill-foreground"
            fontSize={10}
          >
            {`NOW ${nowClockLabel}`}
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
        {/*
          The axis now carries absolute times, so this says what unit those
          are in and stops — the old "first trip boards HH:MM" note was
          standing in for a scale, and a single anchor time is a worse one.
        */}
        <span>times UTC</span>
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
  dimmed,
}: {
  trip: TripSchedule;
  y: number;
  nowMs: number;
  halfWindowMs: number;
  /** Held back behind the emphasised lane; never changes the bar's geometry. */
  dimmed: boolean;
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

  const laneY = y + SEGMENT_INSET;
  const laneHeight = LANE_HEIGHT - SEGMENT_INSET * 2;

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
            // An SVG attribute rather than a utility class: the legend's
            // swatches are matched against this element's class list, and a
            // per-lane opacity class would desynchronise them.
            opacity={dimmed ? 0.65 : undefined}
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
