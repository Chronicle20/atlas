import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";

import { VesselsTable } from "@/components/features/transports/VesselsTable";
import type {
  RouteState,
  ScheduledRoute,
  Vessel,
} from "@/types/models/transport";

function route(name: string, state: RouteState): ScheduledRoute {
  return {
    id: `route-${name}`,
    attributes: {
      name,
      startMapId: 1,
      stagingMapId: 2,
      enRouteMapIds: [3],
      destinationMapId: 4,
      observationMapId: 5,
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

const vessel: Vessel = {
  id: "orbis-ellinia-boat",
  attributes: {
    uuid: "vessel-uuid",
    name: "Orbis–Ellinia Boat",
    routeAID: "Orbis to Ellinia",
    routeBID: "Ellinia to Orbis",
    turnaroundDelay: 60,
  },
};

describe("VesselsTable", () => {
  const outbound = route("Orbis to Ellinia", "open_entry");
  const inbound = route("Ellinia to Orbis", "awaiting_return");

  it("resolves both routes by name and shows their state pills", () => {
    render(<VesselsTable vessels={[vessel]} routes={[outbound, inbound]} />);

    expect(screen.getByText("Orbis–Ellinia Boat")).toBeInTheDocument();
    expect(screen.getByText("Orbis to Ellinia")).toBeInTheDocument();
    expect(screen.getByText("Ellinia to Orbis")).toBeInTheDocument();
    expect(screen.getByText("Boarding")).toBeInTheDocument();
    expect(screen.getByText("Awaiting return")).toBeInTheDocument();
  });

  it("anchors each row on the vessel slug so the board can deep link to it", () => {
    const { container } = render(
      <VesselsTable vessels={[vessel]} routes={[outbound, inbound]} />,
    );

    expect(
      container.querySelector("#vessel-orbis-ellinia-boat"),
    ).not.toBeNull();
  });

  it("flags a vessel whose route reference resolves to nothing", () => {
    render(<VesselsTable vessels={[vessel]} routes={[outbound]} />);

    expect(
      screen.getByText(/both of this vessel's routes will be out of service/i),
    ).toBeInTheDocument();
  });

  it("renders the turnaround delay", () => {
    render(<VesselsTable vessels={[vessel]} routes={[outbound, inbound]} />);

    expect(screen.getByText("1m")).toBeInTheDocument();
  });

  it("does not match a route by id when the vessel reference is a different name", () => {
    // Seed data often makes name and id equal, which would hide a
    // hand-rolled id-based match. Use ids that collide with the OTHER
    // route's name to prove matching goes through resolveVesselRoutes (by
    // attributes.name), not by route.id.
    const trickyVessel: Vessel = {
      id: "route-Ellinia to Orbis",
      attributes: {
        uuid: "vessel-uuid-2",
        name: "Tricky Boat",
        routeAID: "Orbis to Ellinia",
        routeBID: "Ellinia to Orbis",
        turnaroundDelay: 30,
      },
    };

    render(
      <VesselsTable vessels={[trickyVessel]} routes={[outbound, inbound]} />,
    );

    // Both sides resolve correctly by name despite the vessel's id colliding
    // with inbound's route id — if matching fell back to id, routeA would
    // resolve to nothing (no route has id "Orbis to Ellinia").
    expect(screen.queryByText("(no match)")).toBeNull();
    expect(screen.getByText("Boarding")).toBeInTheDocument();
    expect(screen.getByText("Awaiting return")).toBeInTheDocument();
  });

  it("shows a loading state distinct from the empty state", () => {
    render(<VesselsTable vessels={[]} routes={[]} isLoading />);

    expect(screen.getByText(/loading vessels/i)).toBeInTheDocument();
    expect(screen.queryByText(/no vessels configured/i)).toBeNull();
  });

  it("shows an unambiguous error state distinct from the empty state", () => {
    render(<VesselsTable vessels={[]} routes={[]} isError />);

    expect(screen.getByText(/failed to load vessels/i)).toBeInTheDocument();
    // Must not read as "nothing configured" — that's a different state.
    expect(screen.queryByText(/no vessels configured/i)).toBeNull();
  });

  it("shows a plain empty state when zero vessels are configured", () => {
    render(<VesselsTable vessels={[]} routes={[]} />);

    expect(screen.getByText(/no vessels configured/i)).toBeInTheDocument();
    // Must not read as a failure — that's a different state.
    expect(screen.queryByText(/failed to load/i)).toBeNull();
  });

  it("prefers the loading state over the error state when both are set", () => {
    render(<VesselsTable vessels={[]} routes={[]} isLoading isError />);

    expect(screen.getByText(/loading vessels/i)).toBeInTheDocument();
    expect(screen.queryByText(/failed to load vessels/i)).toBeNull();
  });
});
