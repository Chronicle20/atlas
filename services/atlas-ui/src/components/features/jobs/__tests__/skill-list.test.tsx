import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import type { SkillDefinitionWithIcon } from "@/lib/hooks/api/useSkillDefinition";
import { SkillList } from "@/components/features/jobs/skill-list";

function def(
  id: number,
  name: string,
  over?: Partial<SkillDefinitionWithIcon>,
): SkillDefinitionWithIcon {
  return {
    id,
    name,
    description: "",
    action: true,
    element: "",
    animationTime: 0,
    maxLevel: 20,
    effects: [],
    iconUrl: `http://assets.test/skills/${id}/icon`,
    ...over,
  };
}

const defs = [
  def(1001004, "Power Strike"),
  def(1001005, "Slash Blast"),
  def(1001003, "Iron Body"),
];

function renderList(over?: Partial<Parameters<typeof SkillList>[0]>) {
  return render(
    <SkillList
      jobName="Warrior"
      defs={defs}
      state="ready"
      selectedSkillId={null}
      accent="--c-warrior"
      onSelect={() => {}}
      {...over}
    />,
  );
}

describe("SkillList", () => {
  it("renders rows with name, monospace id, type badge, and master level", () => {
    renderList();
    expect(screen.getByText("Warrior — Skills")).toBeInTheDocument();
    const row = screen.getByRole("button", { name: /Power Strike/ });
    expect(row).toHaveTextContent("1001004");
    expect(row).toHaveTextContent("Active");
    expect(row).toHaveTextContent("Master 20");
  });

  it("marks the selected row pressed and fires onSelect", () => {
    const onSelect = vi.fn();
    renderList({ selectedSkillId: 1001005, onSelect });
    expect(
      screen.getByRole("button", { name: /Slash Blast/ }),
    ).toHaveAttribute("aria-pressed", "true");
    fireEvent.click(screen.getByRole("button", { name: /Power Strike/ }));
    expect(onSelect).toHaveBeenCalledWith(1001004);
  });

  it("filters by case-insensitive name substring and by id substring", () => {
    renderList();
    const input = screen.getByLabelText("Filter skills");
    fireEvent.change(input, { target: { value: "power" } });
    expect(
      screen.getByRole("button", { name: /Power Strike/ }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Slash Blast/ }),
    ).not.toBeInTheDocument();
    fireEvent.change(input, { target: { value: "1001005" } });
    expect(
      screen.getByRole("button", { name: /Slash Blast/ }),
    ).toBeInTheDocument();
    fireEvent.change(input, { target: { value: "zzz" } });
    expect(screen.getByText(/No skills match/)).toBeInTheDocument();
  });

  it("renders the loading, error, empty, and defs-failed states verbatim", () => {
    const { rerender } = renderList({ state: "loading", defs: [] });
    expect(screen.getByTestId("skill-list-loading")).toBeInTheDocument();
    rerender(
      <SkillList
        jobName="Warrior"
        defs={[]}
        state="error"
        selectedSkillId={null}
        accent="--c-warrior"
        onSelect={() => {}}
      />,
    );
    expect(
      screen.getByText("Failed to load this job's skills."),
    ).toBeInTheDocument();
    rerender(
      <SkillList
        jobName="Warrior"
        defs={[]}
        state="empty"
        selectedSkillId={null}
        accent="--c-warrior"
        onSelect={() => {}}
      />,
    );
    expect(screen.getByText("This job grants no skills.")).toBeInTheDocument();
    rerender(
      <SkillList
        jobName="Warrior"
        defs={[]}
        state="defs-failed"
        selectedSkillId={null}
        accent="--c-warrior"
        onSelect={() => {}}
      />,
    );
    expect(screen.getByText("Skill details unavailable.")).toBeInTheDocument();
  });
});
