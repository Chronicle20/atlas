import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { SkillTooltipContent } from "../SkillTooltipContent";
import type { SkillDefinitionWithIcon } from "@/lib/hooks/api/useSkillDefinition";

const baseDef = (overrides: Partial<SkillDefinitionWithIcon> = {}): SkillDefinitionWithIcon => ({
  id: 1101000, name: "Iron Body", description: "",
  action: false, element: "", animationTime: 0, effects: [],
  iconUrl: "test.png",
  ...overrides,
});

describe("SkillTooltipContent", () => {
  it("renders the name and id muted", () => {
    render(<SkillTooltipContent definition={baseDef()} />);
    expect(screen.getByText("Iron Body")).toBeInTheDocument();
    expect(screen.getByText(/1101000/)).toBeInTheDocument();
  });

  it("omits description when empty", () => {
    render(<SkillTooltipContent definition={baseDef()} />);
    expect(screen.queryByTestId("skill-description")).toBeNull();
  });

  it("renders description when present", () => {
    render(<SkillTooltipContent definition={baseDef({ description: "Boost defense." })} />);
    expect(screen.getByText("Boost defense.")).toBeInTheDocument();
  });

  it("renders cooldown derived from current level effect", () => {
    render(<SkillTooltipContent
      definition={baseDef({ effects: [{ cooldown: 0 }, { cooldown: 30 }] })}
      learnedLevel={2}
    />);
    expect(screen.getByText(/30s/)).toBeInTheDocument();
  });

  it("formats statups via skill-effect-format", () => {
    render(<SkillTooltipContent
      definition={baseDef({ effects: [{ duration: 30000, statups: [{ type: "WeaponAttack", amount: 10 }] }] })}
      learnedLevel={1}
    />);
    expect(screen.getByText(/\+10 Weapon Attack for 30s/)).toBeInTheDocument();
  });
});
