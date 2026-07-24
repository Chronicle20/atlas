import { render, screen, fireEvent, within } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import type { SkillDefinitionWithIcon } from "@/lib/hooks/api/useSkillDefinition";
import type { SkillEffect } from "@/services/api/skills.service";
import { SkillDetail } from "@/components/features/jobs/skill-detail";

function makeDef(
  over?: Partial<SkillDefinitionWithIcon>,
): SkillDefinitionWithIcon {
  const effects = Array.from(
    { length: 20 },
    (_, i) => ({ damage: 105 + 8 * i, MPConsume: 4 + i }) as SkillEffect,
  );
  return {
    id: 1001004,
    name: "Power Strike",
    description: "Strikes a single enemy with a concentrated, powerful blow.",
    action: true,
    element: "",
    animationTime: 600,
    maxLevel: 20,
    effects,
    iconUrl: "http://assets.test/skills/1001004/icon",
    ...over,
  };
}

describe("SkillDetail", () => {
  it("renders header, badges, and description", () => {
    render(<SkillDetail def={makeDef()} accent="--c-warrior" />);
    expect(screen.getByText("Power Strike")).toBeInTheDocument();
    expect(screen.getByText("ID 1001004")).toBeInTheDocument();
    expect(screen.getByText("Active")).toBeInTheDocument();
    expect(screen.getByText("Master Lv 20")).toBeInTheDocument();
    expect(
      screen.getByText(/Strikes a single enemy/),
    ).toBeInTheDocument();
  });

  it("drives the stat readout and table highlight from the slider", () => {
    render(<SkillDetail def={makeDef()} accent="--c-warrior" />);
    expect(screen.getByText("Level 1")).toBeInTheDocument();
    const slider = screen.getByLabelText("Skill level");
    fireEvent.change(slider, { target: { value: "5" } });
    expect(screen.getByText("Level 5")).toBeInTheDocument();
    // readout shows the level-5 row: damage 105 + 8*4 = 137
    expect(screen.getByTestId("stat-readout")).toHaveTextContent("137");
    // the open all-levels table highlights row 5
    const table = screen.getByRole("table");
    const rows = within(table).getAllByRole("row");
    expect(rows[5]).toHaveAttribute("data-selected", "true"); // rows[0] = header
    expect(rows[1]).toHaveAttribute("data-selected", "false");
  });

  it("shows 'No per-level data.' for maxLevel <= 1", () => {
    render(
      <SkillDetail
        def={makeDef({ maxLevel: 1, effects: [] })}
        accent="--c-warrior"
      />,
    );
    expect(screen.getByText("No per-level data.")).toBeInTheDocument();
    expect(screen.queryByLabelText("Skill level")).not.toBeInTheDocument();
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });

  it("shows 'No per-level data.' when the level table is empty", () => {
    render(
      <SkillDetail
        def={makeDef({ effects: [] })}
        accent="--c-warrior"
      />,
    );
    expect(screen.getByText("No per-level data.")).toBeInTheDocument();
  });

  it("resets the slider when remounted for another skill (key pattern)", () => {
    const { rerender } = render(
      <SkillDetail key={1001004} def={makeDef()} accent="--c-warrior" />,
    );
    fireEvent.change(screen.getByLabelText("Skill level"), {
      target: { value: "9" },
    });
    expect(screen.getByText("Level 9")).toBeInTheDocument();
    rerender(
      <SkillDetail
        key={1001005}
        def={makeDef({ id: 1001005, name: "Slash Blast" })}
        accent="--c-warrior"
      />,
    );
    expect(screen.getByText("Level 1")).toBeInTheDocument();
  });
});
