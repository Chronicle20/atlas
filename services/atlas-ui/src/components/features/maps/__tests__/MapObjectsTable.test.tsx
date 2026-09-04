import { render, screen } from "@testing-library/react";
import { MapObjectsTable } from "../MapObjectsTable";
import type { MapObjectData } from "@/services/api/map-entities.service";

const objects: MapObjectData[] = [
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

describe("MapObjectsTable", () => {
  it("renders one row per object", () => {
    render(<MapObjectsTable objects={objects} />);
    expect(screen.getByText("gate")).toBeInTheDocument();
    expect(screen.getByText("menhir0")).toBeInTheDocument();
  });

  it("shows the kind", () => {
    render(<MapObjectsTable objects={objects} />);
    expect(screen.getByText("ENVIRONMENT")).toBeInTheDocument();
    expect(screen.getByText("OBSTACLE")).toBeInTheDocument();
  });

  it("shows the WZ source", () => {
    render(<MapObjectsTable objects={objects} />);
    expect(screen.getByText("effect")).toBeInTheDocument();
    expect(screen.getByText("trapGL")).toBeInTheDocument();
  });

  it("shows the position", () => {
    render(<MapObjectsTable objects={objects} />);
    expect(screen.getByText(/640/)).toBeInTheDocument();
    expect(screen.getByText(/120/)).toBeInTheDocument();
  });

  it("empty state", () => {
    render(<MapObjectsTable objects={[]} />);
    expect(screen.getByText(/no named objects/i)).toBeInTheDocument();
    expect(screen.queryByRole("row")).not.toBeInTheDocument();
  });

  it("loading", () => {
    render(<MapObjectsTable objects={undefined} />);
    expect(screen.getByText(/loading objects/i)).toBeInTheDocument();
  });
});
