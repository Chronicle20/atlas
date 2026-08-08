import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { FIXTURE_JOBS_SORTED } from "@/lib/jobs/__tests__/job-graph-fixtures";
import { buildJobGraph, type JobGraph } from "@/lib/jobs/job-graph";
import type { JobAvailabilityEntry } from "@/services/api/availability.service";
import { AdvancementFlow } from "@/components/features/jobs/advancement-flow";

// The structural fixture source predates identity/wire-id divergence, so
// identity === id here.
const FULL_AVAILABILITY: JobAvailabilityEntry[] = FIXTURE_JOBS_SORTED.map(
  (e) => ({
    id: e.id,
    name: e.name,
    parent: e.parent,
    identity: e.id,
  }),
);

function graphOf(present: ReadonlySet<number>): JobGraph {
  return buildJobGraph(FULL_AVAILABILITY, present);
}

function cell(id: number): HTMLElement {
  return screen.getByTestId(`flow-cell-${id}`);
}

/** Everything except the Evan branch — irrelevant to these fixtures. */
const NO_EVAN: ReadonlySet<number> = new Set(
  FULL_AVAILABILITY.map((e) => e.id).filter(
    (id) => id !== 2001 && !(id >= 2200 && id <= 2218),
  ),
);
const NO_EVAN_GRAPH = graphOf(NO_EVAN);

describe("AdvancementFlow", () => {
  it("tier-aligns the Magician branch: same-tier jobs share a grid column", () => {
    render(
      <AdvancementFlow
        graph={NO_EVAN_GRAPH}
        entryId={200}
        selectedJobId={200}
        accent="--c-magician"
        onSelect={() => {}}
      />,
    );
    // anchors: Beginner (col 1), Magician (col 2), spanning all 3 path rows
    expect(cell(0).style.gridColumn).toBe("1");
    expect(cell(0).style.gridRow).toBe("1 / span 3");
    expect(cell(200).style.gridColumn).toBe("2");
    // 2nd-job tier column: Wizard (F/P) / Wizard (I/L) / Cleric vertically aligned
    expect(cell(210).style.gridColumn).toBe("3");
    expect(cell(210).style.gridRow).toBe("1");
    expect(cell(220).style.gridColumn).toBe("3");
    expect(cell(220).style.gridRow).toBe("2");
    expect(cell(230).style.gridColumn).toBe("3");
    expect(cell(230).style.gridRow).toBe("3");
    // 4th-job tier aligned likewise (2 anchors + chain positions 3..5)
    expect(cell(212).style.gridColumn).toBe("5");
    expect(cell(232).style.gridColumn).toBe("5");
  });

  it("renders the GM line Beginner > GM > Super GM with tier tags", () => {
    render(
      <AdvancementFlow
        graph={NO_EVAN_GRAPH}
        entryId={900}
        selectedJobId={900}
        accent="--c-special"
        onSelect={() => {}}
      />,
    );
    expect(cell(0).style.gridColumn).toBe("1");
    expect(cell(900).style.gridColumn).toBe("2");
    expect(cell(910).style.gridColumn).toBe("3");
    expect(cell(910).style.gridRow).toBe("1");
    expect(screen.getByText("Base")).toBeInTheDocument();
    expect(screen.getByText("1st")).toBeInTheDocument();
    expect(screen.getByText("2nd")).toBeInTheDocument();
  });

  it("marks the selected chip pressed and fires onSelect on click", () => {
    const onSelect = vi.fn();
    render(
      <AdvancementFlow
        graph={NO_EVAN_GRAPH}
        entryId={100}
        selectedJobId={110}
        accent="--c-warrior"
        onSelect={onSelect}
      />,
    );
    expect(screen.getByRole("button", { name: /Fighter/ })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    fireEvent.click(screen.getByRole("button", { name: /Page/ }));
    expect(onSelect).toHaveBeenCalledWith(120);
  });

  it("renders the Pirate branch when the tenant's job set includes it", () => {
    render(
      <AdvancementFlow
        graph={NO_EVAN_GRAPH}
        entryId={500}
        selectedJobId={500}
        accent="--c-pirate"
        onSelect={() => {}}
      />,
    );
    expect(screen.getByRole("button", { name: /Brawler/ })).toBeInTheDocument();
  });

  it("v0.83: every Cygnus branch ends at 3rd job", () => {
    const v83 = buildJobGraph(
      [
        { id: 1000, name: "Noblesse", parent: null, identity: 1000 },
        { id: 1100, name: "Dawn Warrior 1", parent: 1000, identity: 1100 },
        { id: 1110, name: "Dawn Warrior 2", parent: 1100, identity: 1110 },
        { id: 1111, name: "Dawn Warrior 3", parent: 1110, identity: 1111 },
      ],
      new Set([1000, 1100, 1110, 1111]),
    );
    render(
      <AdvancementFlow
        graph={v83}
        entryId={1000}
        selectedJobId={1000}
        accent="--c-cygnus"
        onSelect={() => {}}
      />,
    );
    expect(screen.getByTestId("flow-cell-1111")).toBeInTheDocument();
    expect(screen.queryByTestId("flow-cell-1112")).not.toBeInTheDocument();
  });
});
