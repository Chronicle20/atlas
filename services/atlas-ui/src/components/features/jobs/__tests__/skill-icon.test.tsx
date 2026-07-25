import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import type { SkillDefinitionWithIcon } from "@/lib/hooks/api/useSkillDefinition";
import { SkillIcon } from "@/components/features/jobs/skill-icon";

const def: SkillDefinitionWithIcon = {
  id: 1001004,
  name: "Power Strike",
  description: "",
  action: true,
  element: "",
  animationTime: 0,
  maxLevel: 20,
  effects: [],
  iconUrl: "http://assets.test/skills/1001004/icon",
};

describe("SkillIcon", () => {
  it("renders the real icon image", () => {
    render(<SkillIcon def={def} name="Power Strike" />);
    const img = screen.getByRole("img", { name: "Power Strike" });
    expect(img).toHaveAttribute("src", def.iconUrl);
  });

  it("falls back to the Sparkles glyph when the image errors", () => {
    render(<SkillIcon def={def} name="Power Strike" />);
    fireEvent.error(screen.getByRole("img", { name: "Power Strike" }));
    expect(
      screen.getByTestId("skill-icon-fallback-1001004"),
    ).toBeInTheDocument();
    expect(screen.queryByRole("img")).not.toBeInTheDocument();
  });
});
