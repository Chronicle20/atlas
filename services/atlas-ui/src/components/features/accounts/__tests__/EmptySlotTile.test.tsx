// services/atlas-ui/src/components/features/accounts/__tests__/EmptySlotTile.test.tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import type { Character } from "@/types/models/character";
import type { TenantConfigAttributes } from "@/services/api";

vi.mock("@/components/features/characters/CharacterRenderer", () => ({
  CharacterRenderer: ({ character }: { character: Character }) => (
    <div
      data-testid="renderer"
      data-face={character.attributes.face}
      data-hair={character.attributes.hair}
      data-skin={character.attributes.skinColor}
      data-gender={character.attributes.gender}
    />
  ),
}));

import { EmptySlotTile } from "../EmptySlotTile";

const template: TenantConfigAttributes["characters"]["templates"][number] = {
  jobIndex: 0,
  subJobIndex: 0,
  gender: 0,
  mapId: 100000000,
  faces: [20000, 20001],
  hairs: [30030],
  hairColors: [3],
  skinColors: [2],
  tops: [],
  bottoms: [],
  shoes: [],
  weapons: [],
  items: [],
  skills: [],
};

describe("EmptySlotTile", () => {
  it("renders the silhouette via CharacterRenderer using template defaults", () => {
    render(<EmptySlotTile onClick={vi.fn()} template={template} />);
    const renderer = screen.getByTestId("renderer");
    expect(renderer).toHaveAttribute("data-face", "20000");
    // hair = hairs[0] + hairColors[0] = 30030 + 3 = 30033
    expect(renderer).toHaveAttribute("data-hair", "30033");
    expect(renderer).toHaveAttribute("data-skin", "2");
    expect(renderer).toHaveAttribute("data-gender", "0");
  });

  it("falls back to a plus glyph when no template is provided", () => {
    render(<EmptySlotTile onClick={vi.fn()} />);
    expect(screen.queryByTestId("renderer")).toBeNull();
  });

  it("invokes onClick when clicked", async () => {
    const onClick = vi.fn();
    render(<EmptySlotTile onClick={onClick} template={template} />);
    await userEvent.click(
      screen.getByRole("button", { name: /add character to slot/i }),
    );
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("blocks click when disabled", async () => {
    const onClick = vi.fn();
    render(<EmptySlotTile onClick={onClick} template={template} disabled />);
    await userEvent.click(
      screen.getByRole("button", { name: /add character to slot/i }),
    );
    expect(onClick).not.toHaveBeenCalled();
  });
});
