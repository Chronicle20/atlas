import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { BranchRail } from "@/components/features/jobs/branch-rail";
import { visibleRailGroups } from "@/components/features/jobs/rail-groups";

describe("BranchRail", () => {
  it("renders group labels, entry names, and subtree counts", () => {
    render(
      <BranchRail
        groups={visibleRailGroups(83)}
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
        groups={visibleRailGroups(83)}
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
        groups={visibleRailGroups(83)}
        selectedEntryId={100}
        onSelect={() => {}}
      />,
    );
    const warrior = screen.getByRole("button", { name: /Warrior 10/ });
    expect(warrior.style.getPropertyValue("--acc")).toBe("var(--c-warrior)");
  });
});
