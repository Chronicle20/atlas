import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { ClassAppearanceSection } from "../ClassAppearanceSection";
import { DEFAULT_PRESET_ATTRIBUTES } from "../presetEditorState";

// Radix Select/Dialog rely on DOM APIs jsdom does not implement.
Element.prototype.hasPointerCapture ||= () => false;
Element.prototype.scrollIntoView ||= () => {};

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } }),
}));

const useFaceIdsMock = vi.fn();
const useHairIdsMock = vi.fn();
vi.mock("@/lib/hooks/api/useCosmetics", () => ({
  useFaceIds: (...a: unknown[]) => useFaceIdsMock(...a),
  useHairIds: (...a: unknown[]) => useHairIdsMock(...a),
}));

const useItemNamesMock = vi.fn();
vi.mock("@/lib/hooks/api/useItemNames", () => ({
  useItemNames: (...a: unknown[]) => useItemNamesMock(...a),
}));

beforeEach(() => {
  useFaceIdsMock.mockReturnValue({
    data: [20000, 20001, 20002],
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  });
  useHairIdsMock.mockReturnValue({
    data: [],
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  });
  useItemNamesMock.mockReturnValue({});
});

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

  it("named-job Select dispatches onSetField(jobId, <number>) for a known job", async () => {
    const onSetField = vi.fn();
    render(<ClassAppearanceSection attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }} onSetField={onSetField} />);
    await userEvent.click(screen.getByRole("combobox", { name: /class/i }));
    await userEvent.click(await screen.findByRole("option", { name: /warrior/i }));
    expect(onSetField).toHaveBeenCalledWith("jobId", 100);
    // Payload must be a number, not the SelectItem's string value.
    const call = onSetField.mock.calls.find(([field]) => field === "jobId");
    expect(typeof call![1]).toBe("number");
  });

  it("browsing faces and picking a candidate replaces face and closes the dialog", async () => {
    const onSetField = vi.fn();
    render(<ClassAppearanceSection attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }} onSetField={onSetField} />);

    await userEvent.click(screen.getByRole("button", { name: /browse faces/i }));
    expect(await screen.findByRole("dialog")).toBeInTheDocument();

    // Default face is 20000 (already in-pool/selected); pick a different
    // enumerated candidate surfaced by the mocked useFaceIds hook.
    await userEvent.click(
      screen.getByRole("button", { name: /add face 20001/i }),
    );

    expect(onSetField).toHaveBeenCalledWith("face", 20001);
    // Replace mode closes the dialog on pick.
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});
