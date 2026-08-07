import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";

import { MapFlowRail } from "@/components/features/transports/MapFlowRail";
import { MapCell } from "@/components/map-cell";
import type {
  RouteState,
  ScheduledRoute,
  ScheduledRouteAttributes,
} from "@/types/models/transport";

vi.mock("@/components/map-cell", () => ({
  MapCell: vi.fn(({ mapId }: { mapId: string }) => <span>map:{mapId}</span>),
}));

function route(
  state: RouteState,
  overrides: Partial<ScheduledRouteAttributes> = {},
): ScheduledRoute {
  return {
    id: "r1",
    attributes: {
      name: "Orbis to Ellinia",
      startMapId: 200000100,
      stagingMapId: 200000110,
      enRouteMapIds: [200090010, 200090011],
      destinationMapId: 101000300,
      observationMapId: 200000111,
      state,
      boardingWindowSeconds: 300,
      preDepartureSeconds: 120,
      travelDurationSeconds: 600,
      cycleIntervalSeconds: 900,
      nextTransitionAt: "",
      nextState: "",
      ...overrides,
    },
  };
}

describe("MapFlowRail", () => {
  it("renders the whole chain in order", () => {
    render(<MapFlowRail route={route("open_entry")} tenant={null} />);

    const stops = screen.getAllByText(/^map:/).map((node) => node.textContent);

    expect(stops).toEqual([
      "map:200000100",
      "map:200000110",
      "map:200090010",
      "map:200090011",
      "map:101000300",
    ]);
  });

  it("labels each stop with its role in the chain", () => {
    render(<MapFlowRail route={route("open_entry")} tenant={null} />);

    expect(screen.getByText("start")).toBeInTheDocument();
    expect(screen.getByText("staging")).toBeInTheDocument();
    expect(screen.getByText("en route · entry")).toBeInTheDocument();
    expect(screen.getByText("destination")).toBeInTheDocument();
  });

  it("draws several en-route maps as one parallel stop, not as successive legs", () => {
    // The service warps staging into enRouteMapIds[0] and never onward through
    // the rest, then drains every en-route map to the destination
    // (transports/transport/processor.go). Chaining them would draw a ride
    // through map 1 then map 2 that no character ever takes — and would put an
    // extra "warp on departure" between them to explain the move.
    const { container } = render(
      <MapFlowRail route={route("open_entry")} tenant={null} />,
    );

    expect(container.querySelectorAll("[data-parallel-maps]")).toHaveLength(1);
    // Exactly three legs: walk in, the single departure warp, and arrival.
    expect(screen.getAllByText("warp on departure")).toHaveLength(1);
    expect(screen.queryByText("en route 2")).toBeNull();
    // The siblings hang off the entry map inside the parallel group.
    const group = container.querySelector("[data-parallel-maps]")!;
    expect(group.textContent).toContain("map:200090011");
    expect(group.textContent).not.toContain("map:200090010");
  });

  it("keeps a single en-route map on the rail with no parallel group", () => {
    const { container } = render(
      <MapFlowRail
        route={route("open_entry", { enRouteMapIds: [200090010] })}
        tenant={null}
      />,
    );

    expect(screen.getByText("en route")).toBeInTheDocument();
    expect(container.querySelector("[data-parallel-maps]")).toBeNull();
  });

  it("names the parallel maps as parallel, and which one a departure lands in", () => {
    // Which of the parallel maps a departure lands in is carried visually by a
    // one-word sub-caption on a single badge; the accessible name is the only
    // other channel for it.
    render(<MapFlowRail route={route("open_entry")} tenant={null} />);

    const figure = screen.getByRole("img", { name: /map flow/i });
    expect(figure).toHaveAccessibleName(
      /200090010 in parallel with 200090011/i,
    );
    expect(figure).toHaveAccessibleName(/departure lands in 200090010/i);
  });

  it("captions each leg with what moves a character across it", () => {
    render(<MapFlowRail route={route("open_entry")} tenant={null} />);

    expect(screen.getByText("walk in")).toBeInTheDocument();
    expect(screen.getAllByText("warp on departure").length).toBeGreaterThan(0);
    expect(screen.getByText("warp on arrival")).toBeInTheDocument();
  });

  it("keeps the observation map off the rail — it is not a stop", () => {
    // It is still reported, as one cell of the detail page's configuration
    // strip; the rail is only the chain a character actually traverses.
    render(<MapFlowRail route={route("open_entry")} tenant={null} />);

    expect(screen.queryByText("map:200000111")).toBeNull();
    expect(screen.queryByText(/ARRIVED\/DEPARTED effects fire/i)).toBeNull();
  });

  it("exposes the rail to assistive tech as a labelled figure", () => {
    render(<MapFlowRail route={route("open_entry")} tenant={null} />);

    const figure = screen.getByRole("img", { name: /map flow/i });
    expect(figure).toBeInTheDocument();
  });

  it("emphasises the en-route segment while the route is in transit", () => {
    const { container } = render(
      <MapFlowRail route={route("in_transit")} tenant={null} />,
    );

    expect(
      container.querySelector("[data-en-route-active='true']"),
    ).not.toBeNull();
  });

  it("passes every map id to MapCell as a string, never a number", () => {
    // Route attributes carry map ids as `number` (T8); MapCell's prop is
    // `string`. JSX text interpolation renders a raw number identically to
    // its stringified form, so a dropped `String(...)` coercion wouldn't
    // show up in rendered text — only in the actual prop value MapCell
    // receives, which this inspects directly.
    render(<MapFlowRail route={route("open_entry")} tenant={null} />);

    const calls = vi.mocked(MapCell).mock.calls;
    expect(calls.length).toBeGreaterThan(0);
    for (const [props] of calls) {
      expect(typeof props.mapId).toBe("string");
    }
  });

  it("captions a direct route with no en-route stop as walk in then arrival", () => {
    // With enRouteMapIds empty, stops collapse to [start, staging,
    // destination]: two legs, not the four the default fixture exercises.
    // The staging→destination leg leads straight into the destination, so
    // it must read as the arrival leg, not a departure.
    render(
      <MapFlowRail
        route={route("open_entry", { enRouteMapIds: [] })}
        tenant={null}
      />,
    );

    expect(screen.getByText("walk in")).toBeInTheDocument();
    expect(screen.getByText("warp on arrival")).toBeInTheDocument();
    expect(screen.queryByText("warp on departure")).not.toBeInTheDocument();
  });

  it("makes the in-transit stops discoverable without colour", () => {
    // The only visual signal for the active leg is an SVG's colour/stroke
    // width, and that SVG is aria-hidden. The accessible name is the only
    // channel left, so it must name the state, not just the sequence.
    render(<MapFlowRail route={route("in_transit")} tenant={null} />);

    const figure = screen.getByRole("img", { name: /map flow/i });
    expect(figure).toHaveAccessibleName(
      /in transit through 200090010, 200090011/i,
    );
  });
});
