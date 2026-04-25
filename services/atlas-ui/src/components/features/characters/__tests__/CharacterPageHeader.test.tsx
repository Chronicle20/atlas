import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { CharacterPageHeader } from "../CharacterPageHeader";
import type { Character } from "@/types/models/character";

const baseCharacter = (overrides: Partial<Character["attributes"]> = {}) => ({
  id: "42",
  type: "characters",
  attributes: {
    name: "Aran4th",
    gm: 0,
    ...overrides,
  },
}) as unknown as Character;

describe("CharacterPageHeader", () => {
  it("renders the name and exposes the id via copyable tooltip", async () => {
    render(<CharacterPageHeader character={baseCharacter()} onChangeGm={vi.fn()} onChangeMap={vi.fn()} />);
    expect(screen.getByText("Aran4th")).toBeInTheDocument();
    // tooltip body mounts on hover/focus
    await userEvent.hover(screen.getByText("Aran4th"));
    expect(await screen.findByText("42")).toBeInTheDocument();
  });

  it("shows the GM N badge when gm > 0", () => {
    render(<CharacterPageHeader character={baseCharacter({ gm: 3 })} onChangeGm={vi.fn()} onChangeMap={vi.fn()} />);
    expect(screen.getByText(/GM 3/i)).toBeInTheDocument();
  });

  it("does not show the badge when gm = 0", () => {
    render(<CharacterPageHeader character={baseCharacter()} onChangeGm={vi.fn()} onChangeMap={vi.fn()} />);
    expect(screen.queryByText(/^GM \d/i)).not.toBeInTheDocument();
  });

  it("invokes button handlers", async () => {
    const onChangeGm = vi.fn();
    const onChangeMap = vi.fn();
    render(<CharacterPageHeader character={baseCharacter()} onChangeGm={onChangeGm} onChangeMap={onChangeMap} />);
    await userEvent.click(screen.getByRole("button", { name: /Promote to GM|Change GM/i }));
    await userEvent.click(screen.getByRole("button", { name: /Change Map/i }));
    expect(onChangeGm).toHaveBeenCalled();
    expect(onChangeMap).toHaveBeenCalled();
  });
});
