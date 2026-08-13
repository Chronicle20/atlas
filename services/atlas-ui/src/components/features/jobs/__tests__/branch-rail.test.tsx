import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { FIXTURE_JOBS_SORTED } from "@/lib/jobs/__tests__/job-graph-fixtures";
import { buildJobGraph } from "@/lib/jobs/job-graph";
import type { JobAvailabilityEntry } from "@/services/api/availability.service";
import { BranchRail } from "@/components/features/jobs/branch-rail";
import { visibleRailGroups } from "@/components/features/jobs/rail-groups";

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

/** Everything except the Evan branch (introduced after this tenant's version). */
const NO_EVAN: ReadonlySet<number> = new Set(
  FULL_AVAILABILITY.map((e) => e.id).filter(
    (id) => id !== 2001 && !(id >= 2200 && id <= 2218),
  ),
);
const NO_EVAN_GRAPH = buildJobGraph(FULL_AVAILABILITY, NO_EVAN);

describe("BranchRail", () => {
  it("renders group labels, entry names, and subtree counts", () => {
    render(
      <BranchRail
        groups={visibleRailGroups(NO_EVAN_GRAPH)}
        selectedEntryId={100}
        onSelect={() => {}}
      />,
    );
    expect(screen.getByText("Explorers")).toBeInTheDocument();
    expect(screen.getByText("Cygnus Knights")).toBeInTheDocument();
    expect(screen.getByText("Legends")).toBeInTheDocument();
    expect(screen.getByText("Special")).toBeInTheDocument();
    const warrior = screen.getByRole("button", { name: /Warrior 10/ });
    expect(warrior).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: /^GM 2$/ })).toHaveAttribute(
      "aria-pressed",
      "false",
    );
  });

  it("fires onSelect with the entry id", () => {
    const onSelect = vi.fn();
    render(
      <BranchRail
        groups={visibleRailGroups(NO_EVAN_GRAPH)}
        selectedEntryId={100}
        onSelect={onSelect}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: /Magician/ }));
    expect(onSelect).toHaveBeenCalledWith(200);
  });

  it("scopes the branch accent token per entry", () => {
    render(
      <BranchRail
        groups={visibleRailGroups(NO_EVAN_GRAPH)}
        selectedEntryId={100}
        onSelect={() => {}}
      />,
    );
    const warrior = screen.getByRole("button", { name: /Warrior 10/ });
    expect(warrior.style.getPropertyValue("--acc")).toBe("var(--c-warrior)");
  });
});
