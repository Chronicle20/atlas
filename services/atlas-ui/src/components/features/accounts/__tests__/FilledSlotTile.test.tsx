// services/atlas-ui/src/components/features/accounts/__tests__/FilledSlotTile.test.tsx
import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { FilledSlotTile } from "../FilledSlotTile";
import type { Character } from "@/types/models/character";

vi.mock("@/components/features/characters/CharacterRenderer", () => ({
  CharacterRenderer: ({ character }: { character: Character }) => (
    <div data-testid="renderer" data-name={character.attributes.name} />
  ),
}));

const character = (worldId: number): Character => ({
  id: "5",
  type: "characters",
  attributes: {
    accountId: 1,
    worldId,
    name: "Foo",
  } as Character["attributes"],
});

const worldsWithFlag = [
  { name: "Scania", flag: "https://example.com/scania.png", serverMessage: "", eventMessage: "", whyAmIRecommended: "" },
];

const worldsEmptyFlag = [
  { name: "Scania", flag: "", serverMessage: "", eventMessage: "", whyAmIRecommended: "" },
];

describe("FilledSlotTile", () => {
  it("renders a link to /characters/{id} with character renderer + name", () => {
    render(
      <MemoryRouter>
        <FilledSlotTile character={character(0)} worlds={worldsWithFlag} />
      </MemoryRouter>
    );
    const link = screen.getByRole("link", { name: /Foo/i });
    expect(link).toHaveAttribute("href", "/characters/5");
    expect(screen.getByTestId("renderer")).toBeInTheDocument();
    expect(screen.getByText("Foo")).toBeInTheDocument();
  });

  it("renders an <img> when the world flag is populated", () => {
    const { container } = render(
      <MemoryRouter>
        <FilledSlotTile character={character(0)} worlds={worldsWithFlag} />
      </MemoryRouter>
    );
    const img = container.querySelector('img[src*="scania.png"]');
    expect(img).not.toBeNull();
  });

  it("renders the Globe fallback when worldId is out of range", () => {
    const { container } = render(
      <MemoryRouter>
        <FilledSlotTile character={character(7)} worlds={worldsWithFlag} />
      </MemoryRouter>
    );
    expect(container.querySelector("svg.lucide-globe")).not.toBeNull();
  });

  it("renders the Globe fallback when flag is empty string", () => {
    const { container } = render(
      <MemoryRouter>
        <FilledSlotTile character={character(0)} worlds={worldsEmptyFlag} />
      </MemoryRouter>
    );
    expect(container.querySelector("svg.lucide-globe")).not.toBeNull();
  });
});
