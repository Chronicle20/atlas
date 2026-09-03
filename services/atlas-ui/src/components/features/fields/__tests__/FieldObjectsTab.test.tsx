import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  FieldObjectsTab,
  type TrackedObject,
} from "@/components/features/fields/FieldObjectsTab";
import type { MapObjectData } from "@/services/api/map-entities.service";

const defined: MapObjectData[] = [
  {
    id: "ENVIRONMENT:gate",
    type: "map-objects",
    attributes: {
      kind: "ENVIRONMENT",
      name: "gate",
      objectSource: "effect",
      l0: "quest",
      l1: "gate",
      l2: "1",
      x: 640,
      y: 120,
      z: 0,
      layer: 3,
    },
  },
  {
    id: "OBSTACLE:menhir0",
    type: "map-objects",
    attributes: {
      kind: "OBSTACLE",
      name: "menhir0",
      objectSource: "trapGL",
      l0: "ckPQ",
      l1: "menhir",
      l2: "0",
      x: -30,
      y: 45,
      z: 7,
      layer: 2,
    },
  },
];

const trackedRow: TrackedObject[] = [
  { id: "OBSTACLE:menhir0", kind: "OBSTACLE", name: "menhir0", state: 3 },
];

const DIVIDER_TEXT = "Defined on the map, no state tracked in this field";

describe("FieldObjectsTab", () => {
  it("untracked only", () => {
    render(<FieldObjectsTab defined={defined} tracked={undefined} />);

    expect(screen.getByText("gate")).toBeInTheDocument();
    expect(screen.getByText("menhir0")).toBeInTheDocument();
    expect(screen.getByText(DIVIDER_TEXT)).toBeInTheDocument();
  });

  it("untracked only, empty tracked array", () => {
    render(<FieldObjectsTab defined={defined} tracked={[]} />);

    expect(screen.getByText("gate")).toBeInTheDocument();
    expect(screen.getByText("menhir0")).toBeInTheDocument();
    expect(screen.getByText(DIVIDER_TEXT)).toBeInTheDocument();
  });

  it("tracked only", () => {
    render(<FieldObjectsTab defined={[]} tracked={trackedRow} />);

    expect(screen.getByText("menhir0")).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
    expect(screen.queryByText(DIVIDER_TEXT)).not.toBeInTheDocument();
  });

  it("both", () => {
    render(<FieldObjectsTab defined={defined} tracked={trackedRow} />);

    expect(screen.getAllByText("menhir0")).toHaveLength(1);
    const trackedRowEl = screen.getByText("menhir0").closest("tr");
    expect(trackedRowEl).not.toBeNull();
    expect(trackedRowEl).toHaveTextContent("3");

    expect(screen.getByText("gate")).toBeInTheDocument();
    expect(screen.getByText(DIVIDER_TEXT)).toBeInTheDocument();
  });

  it("an object in both appears once (FR-33)", () => {
    render(<FieldObjectsTab defined={defined} tracked={trackedRow} />);

    expect(screen.getAllByText("menhir0")).toHaveLength(1);
  });

  it("both empty is a normal empty state", () => {
    render(<FieldObjectsTab defined={[]} tracked={[]} />);

    expect(screen.getByText(/no map objects/i)).toBeInTheDocument();
    expect(screen.queryByText(/404/)).not.toBeInTheDocument();
    expect(screen.queryByText(/failed to load/i)).not.toBeInTheDocument();
  });

  it("rows carry their kind", () => {
    render(<FieldObjectsTab defined={defined} tracked={undefined} />);

    expect(screen.getByText("ENVIRONMENT")).toBeInTheDocument();
    expect(screen.getByText("OBSTACLE")).toBeInTheDocument();
  });

  it("names note pass-through", () => {
    render(<FieldObjectsTab defined={defined} tracked={undefined} />);

    expect(
      screen.getByText(/does not validate a name against the map/i),
    ).toBeInTheDocument();
  });
});
