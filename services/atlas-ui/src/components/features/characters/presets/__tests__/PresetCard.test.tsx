import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { PresetCard } from "../PresetCard";
import { DEFAULT_PRESET_ATTRIBUTES } from "../presetEditorState";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } }),
}));
vi.mock("@/lib/hooks/useCharacterImage", () => ({
  useCharacterImage: () => ({ isLoading: false, isError: false, imageUrl: "http://img/c.png", refetch: vi.fn() }),
}));

const preset = {
  key: "a1", id: "a1",
  attributes: { ...DEFAULT_PRESET_ATTRIBUTES, name: "Test Warrior", jobId: 100, level: 30, gm: 0, description: "A tank", tags: ["PvE"] },
};

describe("PresetCard", () => {
  it("shows name, job badge, level, description, tags", () => {
    render(<PresetCard preset={preset} dirty={false} onOpen={vi.fn()} onDuplicate={vi.fn()} />);
    expect(screen.getByText("Test Warrior")).toBeInTheDocument();
    expect(screen.getByText("Warrior")).toBeInTheDocument();
    expect(screen.getByText(/Lv 30/)).toBeInTheDocument();
    expect(screen.getByText("A tank")).toBeInTheDocument();
    expect(screen.getByText("PvE")).toBeInTheDocument();
  });

  it("shows a dirty-dot when dirty", () => {
    render(<PresetCard preset={preset} dirty onOpen={vi.fn()} onDuplicate={vi.fn()} />);
    expect(screen.getByTestId("dirty-dot")).toBeInTheDocument();
  });

  it("opening (click/Enter) fires onOpen; Duplicate fires onDuplicate", async () => {
    const onOpen = vi.fn();
    const onDuplicate = vi.fn();
    render(<PresetCard preset={preset} dirty={false} onOpen={onOpen} onDuplicate={onDuplicate} />);
    await userEvent.click(screen.getByRole("button", { name: /open preset Test Warrior/i }));
    expect(onOpen).toHaveBeenCalled();
    await userEvent.click(screen.getByRole("button", { name: /duplicate/i }));
    expect(onDuplicate).toHaveBeenCalled();
  });

  it("hides Apply quick-action when onApply is absent", () => {
    render(<PresetCard preset={preset} dirty={false} onOpen={vi.fn()} onDuplicate={vi.fn()} />);
    expect(screen.queryByRole("button", { name: /apply to account/i })).toBeNull();
  });
});
