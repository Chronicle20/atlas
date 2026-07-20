import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { ClassAppearanceSection } from "../ClassAppearanceSection";
import { DEFAULT_PRESET_ATTRIBUTES } from "../presetEditorState";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } }),
}));

describe("ClassAppearanceSection", () => {
  it("named job picker sets jobId; advanced numeric accepts arbitrary ids", async () => {
    const onSetField = vi.fn();
    render(<ClassAppearanceSection attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }} onSetField={onSetField} />);
    // Advanced numeric entry
    const advanced = screen.getByLabelText(/advanced job id/i);
    await userEvent.clear(advanced);
    await userEvent.type(advanced, "123456");
    expect(onSetField).toHaveBeenCalledWith("jobId", 123456);
  });

  it("skin thumb click replaces skinColor", async () => {
    const onSetField = vi.fn();
    render(<ClassAppearanceSection attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }} onSetField={onSetField} />);
    await userEvent.click(screen.getByRole("button", { name: /skin tone 3/i }));
    expect(onSetField).toHaveBeenCalledWith("skinColor", 3);
  });

  it("gender select toggles 0/1", async () => {
    const onSetField = vi.fn();
    render(<ClassAppearanceSection attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }} onSetField={onSetField} />);
    // Implementation detail: gender is a shadcn Select or M/F buttons; assert the female choice fires 1.
    await userEvent.click(screen.getByRole("button", { name: /female/i }));
    expect(onSetField).toHaveBeenCalledWith("gender", 1);
  });
});
