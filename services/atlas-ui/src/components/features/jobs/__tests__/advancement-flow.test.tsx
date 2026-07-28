import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { JOB_GRAPH } from "@/lib/jobs/job-advancement-tree";
import { AdvancementFlow } from "@/components/features/jobs/advancement-flow";

function cell(id: number): HTMLElement {
  return screen.getByTestId(`flow-cell-${id}`);
}

/** Everything except the Evan branch — irrelevant to these fixtures. */
const NO_EVAN: ReadonlySet<number> = new Set(
  Object.values(JOB_GRAPH)
    .map((e) => e.id)
    .filter((id) => id !== 2001 && !(id >= 2200 && id <= 2218)),
);

describe("AdvancementFlow", () => {
  it("tier-aligns the Magician branch: same-tier jobs share a grid column", () => {
    render(
      <AdvancementFlow
        entryId={200}
        available={NO_EVAN}
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
        entryId={900}
        available={NO_EVAN}
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
        entryId={100}
        available={NO_EVAN}
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
        entryId={500}
        available={NO_EVAN}
        selectedJobId={500}
        accent="--c-pirate"
        onSelect={() => {}}
      />,
    );
    expect(screen.getByRole("button", { name: /Brawler/ })).toBeInTheDocument();
  });
});
