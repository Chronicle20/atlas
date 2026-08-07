import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";

import { MapFlowRail } from "@/components/features/transports/MapFlowRail";
import { MapCell } from "@/components/map-cell";
import type { RouteState, ScheduledRoute } from "@/types/models/transport";

vi.mock("@/components/map-cell", () => ({
  MapCell: vi.fn(({ mapId }: { mapId: string }) => <span>map:{mapId}</span>),
}));

function route(state: RouteState): ScheduledRoute {
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
      // the observation map is rendered separately, after the chain
      "map:200000111",
    ]);
  });

  it("captions each leg with what moves a character across it", () => {
    render(<MapFlowRail route={route("open_entry")} tenant={null} />);

    expect(screen.getByText("walk in")).toBeInTheDocument();
    expect(screen.getAllByText("warp on departure").length).toBeGreaterThan(0);
    expect(screen.getByText("warp on arrival")).toBeInTheDocument();
  });

  it("annotates the observation map as an effect origin, not a stop", () => {
    render(<MapFlowRail route={route("open_entry")} tenant={null} />);

    expect(
      screen.getByText(/ARRIVED\/DEPARTED effects fire/i),
    ).toBeInTheDocument();
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
});
