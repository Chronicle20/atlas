import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";

import { VesselTimeline } from "@/components/features/transports/VesselTimeline";
import type { TripSchedule } from "@/types/models/transport";

function trip(
  id: string,
  open: string,
  closed: string,
  departure: string,
  arrival: string,
): TripSchedule {
  // The 2023 date is the schedule's stale computing day; only the time matters.
  const at = (hhmm: string) => `2023-01-01T${hhmm}:00Z`;
  return {
    id,
    attributes: {
      boardingOpen: at(open),
      boardingClosed: at(closed),
      departure: at(departure),
      arrival: at(arrival),
    },
  };
}

const nowEpochMs = Date.parse("2026-08-06T12:00:00Z");

describe("VesselTimeline", () => {
  const outbound = [
    trip("a1", "11:45", "11:50", "11:52", "12:02"),
    trip("a2", "12:15", "12:20", "12:22", "12:32"),
  ];
  const inbound = [trip("b1", "12:03", "12:08", "12:10", "12:20")];

  it("renders one lane for an independent route", () => {
    render(
      <VesselTimeline
        lanes={[{ label: "Orbis to Ellinia", trips: outbound }]}
        nowEpochMs={nowEpochMs}
      />,
    );

    expect(screen.getByText("Orbis to Ellinia")).toBeInTheDocument();
    expect(screen.queryByText("Ellinia to Orbis")).toBeNull();
  });

  it("renders both lanes for a shared vessel", () => {
    render(
      <VesselTimeline
        lanes={[
          { label: "Orbis to Ellinia", trips: outbound, emphasised: true },
          { label: "Ellinia to Orbis", trips: inbound },
        ]}
        nowEpochMs={nowEpochMs}
      />,
    );

    expect(screen.getByText("Orbis to Ellinia")).toBeInTheDocument();
    expect(screen.getByText("Ellinia to Orbis")).toBeInTheDocument();
  });

  it("captions each lane in place, so a rail's route is readable off the chart", () => {
    // A legend that lists two names in argument order does not say which rail
    // is which; the name has to sit on the rail it belongs to.
    const { container } = render(
      <VesselTimeline
        lanes={[
          { label: "Orbis to Ellinia", trips: outbound, emphasised: true },
          { label: "Ellinia to Orbis", trips: inbound },
        ]}
        nowEpochMs={nowEpochMs}
      />,
    );

    const labels = Array.from(
      container.querySelectorAll("[data-lane-label]"),
    ).map((node) => node.getAttribute("data-lane-label"));
    expect(labels).toEqual(["Orbis to Ellinia", "Ellinia to Orbis"]);
    // Drawn inside the figure, not in the legend beneath it.
    expect(container.querySelector("svg [data-lane-label]")).not.toBeNull();
  });

  it("draws both lanes of a shared vessel at the same size", () => {
    // Bar height is read as duration on a strip whose whole point is comparing
    // the two sides of one vessel against each other. Using it as the emphasis
    // channel made the partner's trips look like shorter trips.
    const { container } = render(
      <VesselTimeline
        lanes={[
          { label: "Orbis to Ellinia", trips: outbound, emphasised: true },
          { label: "Ellinia to Orbis", trips: inbound },
        ]}
        nowEpochMs={nowEpochMs}
      />,
    );

    const heights = Array.from(
      container.querySelectorAll("[data-segment]"),
    ).map((node) => node.getAttribute("height"));

    expect(new Set(heights).size).toBe(1);
  });

  it("holds the partner lane back with opacity, never with geometry", () => {
    const { container } = render(
      <VesselTimeline
        lanes={[
          { label: "Orbis to Ellinia", trips: outbound, emphasised: true },
          { label: "Ellinia to Orbis", trips: inbound },
        ]}
        nowEpochMs={nowEpochMs}
      />,
    );

    const opacities = Array.from(
      container.querySelectorAll("[data-segment]"),
    ).map((node) => node.getAttribute("opacity"));

    expect(opacities).toContain(null);
    expect(opacities.some((value) => value !== null)).toBe(true);
  });

  it("never dims a lone lane, emphasised or not", () => {
    const { container } = render(
      <VesselTimeline
        lanes={[{ label: "Orbis to Ellinia", trips: outbound }]}
        nowEpochMs={nowEpochMs}
      />,
    );

    for (const node of container.querySelectorAll("[data-segment]")) {
      expect(node.getAttribute("opacity")).toBeNull();
    }
  });

  it("gives every legend key the colour it stands for", () => {
    // Phase is encoded in the strip *only* by colour, so a key that names the
    // phase without showing its swatch explains nothing.
    const { container } = render(
      <VesselTimeline
        lanes={[{ label: "Orbis to Ellinia", trips: outbound }]}
        nowEpochMs={nowEpochMs}
      />,
    );

    for (const kind of ["open", "locked", "transit"]) {
      const swatch = container.querySelector(`[data-legend-swatch="${kind}"]`);
      expect(swatch).not.toBeNull();
      // The swatch's fill must match the class the matching segments carry.
      const segment = container.querySelector(`[data-segment="${kind}"]`);
      const fill = segment?.getAttribute("class")?.replace("fill-", "bg-");
      expect(swatch?.getAttribute("class")).toContain(fill);
    }
  });

  it("is a labelled figure with a NOW marker", () => {
    const { container } = render(
      <VesselTimeline
        lanes={[{ label: "Orbis to Ellinia", trips: outbound }]}
        nowEpochMs={nowEpochMs}
      />,
    );

    expect(
      screen.getByRole("img", { name: /trip timeline/i }),
    ).toBeInTheDocument();
    expect(container.querySelector("[data-now-marker]")).not.toBeNull();
  });

  it("stamps the NOW marker with the instant it is drawn at", () => {
    // A bare "NOW" says where the marker is but not when, which leaves every
    // absolute time on the strip locked inside a segment tooltip.
    render(
      <VesselTimeline
        lanes={[{ label: "Orbis to Ellinia", trips: outbound }]}
        nowEpochMs={Date.parse("2026-08-06T12:00:07Z")}
      />,
    );

    expect(screen.getByText("NOW 12:00:07")).toBeInTheDocument();
  });

  it("draws a time axis of round wall-clock ticks across the window", () => {
    const { container } = render(
      <VesselTimeline
        lanes={[{ label: "Orbis to Ellinia", trips: outbound }]}
        nowEpochMs={nowEpochMs}
      />,
    );

    expect(container.querySelector("[data-time-axis]")).not.toBeNull();

    const ticks = Array.from(
      container.querySelectorAll("[data-axis-tick]"),
    ).map((node) => node.getAttribute("data-axis-tick"));

    // Two boardings 30m apart give a 30m half-window, so the axis spans
    // 11:30-12:30 and lands on the round ten-minute marks inside it.
    expect(ticks).toEqual([
      "11:30",
      "11:40",
      "11:50",
      "12:00",
      "12:10",
      "12:20",
      "12:30",
    ]);
  });

  it("wraps axis labels across UTC midnight rather than running past 24:00", () => {
    const lateTrips = [
      trip("m1", "23:40", "23:45", "23:47", "23:57"),
      trip("m2", "00:10", "00:15", "00:17", "00:27"),
    ];
    const { container } = render(
      <VesselTimeline
        lanes={[{ label: "Orbis to Ellinia", trips: lateTrips }]}
        nowEpochMs={Date.parse("2026-08-06T23:55:00Z")}
      />,
    );

    const ticks = Array.from(
      container.querySelectorAll("[data-axis-tick]"),
    ).map((node) => node.getAttribute("data-axis-tick"));

    expect(ticks).toContain("23:50");
    expect(ticks).toContain("00:10");
    expect(ticks.some((label) => label!.startsWith("24"))).toBe(false);
  });

  it("labels trip times as time of day only, never with a date", () => {
    const { container } = render(
      <VesselTimeline
        lanes={[{ label: "Orbis to Ellinia", trips: outbound }]}
        nowEpochMs={nowEpochMs}
      />,
    );

    expect(container.textContent).not.toContain("2023");
    expect(container.textContent).toContain("11:45");
  });

  it("renders three segments per in-window trip", () => {
    const { container } = render(
      <VesselTimeline
        lanes={[{ label: "Orbis to Ellinia", trips: [outbound[1]!] }]}
        nowEpochMs={nowEpochMs}
      />,
    );

    expect(container.querySelectorAll("[data-segment]")).toHaveLength(3);
  });

  it("renders an explicit worded empty state for a lane with no trips, not a blank rail", () => {
    const { container } = render(
      <VesselTimeline
        lanes={[{ label: "Orbis to Ellinia", trips: [] }]}
        nowEpochMs={nowEpochMs}
      />,
    );

    expect(screen.getByText(/no trips scheduled/i)).toBeInTheDocument();
    expect(container.querySelector("[data-empty-lane]")).not.toBeNull();
    expect(container.querySelectorAll("[data-segment]")).toHaveLength(0);
  });

  it("keeps an empty lane's worded state out of a shared vessel's populated lane", () => {
    // A shared-vessel view where one side genuinely has no trips today must
    // not make the other, populated side look empty too.
    const { container } = render(
      <VesselTimeline
        lanes={[
          { label: "Orbis to Ellinia", trips: outbound, emphasised: true },
          { label: "Ellinia to Orbis", trips: [] },
        ]}
        nowEpochMs={nowEpochMs}
      />,
    );

    expect(container.querySelectorAll("[data-empty-lane]")).toHaveLength(1);
    expect(container.querySelectorAll("[data-segment]").length).toBeGreaterThan(
      0,
    );
  });

  it("conveys each trip's board/close/depart/arrive times through the accessible name, not just position or colour", () => {
    render(
      <VesselTimeline
        lanes={[{ label: "Orbis to Ellinia", trips: [outbound[0]!] }]}
        nowEpochMs={nowEpochMs}
      />,
    );

    const figure = screen.getByRole("img");
    expect(figure).toHaveAccessibleName(
      /boards 11:45, closes 11:50, departs 11:52, arrives 12:02/i,
    );
  });

  it("names an empty lane's absence of trips in the accessible name too", () => {
    render(
      <VesselTimeline
        lanes={[{ label: "Orbis to Ellinia", trips: [] }]}
        nowEpochMs={nowEpochMs}
      />,
    );

    const figure = screen.getByRole("img");
    expect(figure).toHaveAccessibleName(
      /Orbis to Ellinia: no trips scheduled in this window/i,
    );
  });
});
