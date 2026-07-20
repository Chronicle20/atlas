import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { PresetLibrary } from "../PresetLibrary";
import { DEFAULT_PRESET_ATTRIBUTES } from "../presetEditorState";

vi.mock("../PresetCard", () => ({
  PresetCard: ({
    preset,
    onOpen,
  }: {
    preset: { attributes: { name: string } };
    onOpen: () => void;
  }) => <button onClick={onOpen}>{preset.attributes.name}</button>,
}));

const mk = (key: string, name: string, tags: string[], description = "") => ({
  key,
  id: key,
  attributes: { ...DEFAULT_PRESET_ATTRIBUTES, name, tags, description },
});
const presets = [
  mk("a", "Fresh Beginner", ["starter"], "level 1 blank"),
  mk("b", "Test Warrior", ["combat"], "a tank"),
  mk("c", "GM Admin", ["staff"], "godmode"),
];
const base = {
  dirtyKeys: new Set<string>(),
  canApply: true,
  onOpen: vi.fn(),
  onNew: vi.fn(),
  onDuplicate: vi.fn(),
  onApply: vi.fn(),
};

describe("PresetLibrary", () => {
  it("search matches name/description/tags case-insensitively", async () => {
    render(<PresetLibrary presets={presets} {...base} />);
    await userEvent.type(screen.getByRole("searchbox"), "warrior");
    expect(screen.getByText("Test Warrior")).toBeInTheDocument();
    expect(screen.queryByText("GM Admin")).toBeNull();
  });

  it("single-select tag filter narrows the grid", async () => {
    render(<PresetLibrary presets={presets} {...base} />);
    await userEvent.click(screen.getByRole("button", { name: /^staff$/i }));
    expect(screen.getByText("GM Admin")).toBeInTheDocument();
    expect(screen.queryByText("Test Warrior")).toBeNull();
  });

  it("renders the + New affordance and fires onNew", async () => {
    const onNew = vi.fn();
    render(<PresetLibrary presets={presets} {...base} onNew={onNew} />);
    await userEvent.click(screen.getByRole("button", { name: /new preset/i }));
    expect(onNew).toHaveBeenCalled();
  });

  it("empty state when no presets", () => {
    render(<PresetLibrary presets={[]} {...base} />);
    expect(screen.getByText(/no character presets/i)).toBeInTheDocument();
  });
});
