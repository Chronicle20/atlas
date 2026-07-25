import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { ClassAppearanceSection } from "../ClassAppearanceSection";
import { DEFAULT_PRESET_ATTRIBUTES } from "../presetEditorState";

// Radix Select/Dialog rely on DOM APIs jsdom does not implement.
Element.prototype.hasPointerCapture ||= () => false;
Element.prototype.scrollIntoView ||= () => {};

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
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

// Enumeration fixture: male bases 30000 (colors 0,2) and 30030 (colors 0,1);
// female base 31000 (colors 0,1). 21xxx faces are female by id convention.
const HAIR_IDS = [30000, 30002, 30030, 30031, 31000, 31001];
const FACE_IDS = [20000, 20001, 20002, 21000, 21001];

beforeEach(() => {
  useFaceIdsMock.mockReturnValue({
    data: FACE_IDS,
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  });
  useHairIdsMock.mockReturnValue({
    data: HAIR_IDS,
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  });
  useItemNamesMock.mockReturnValue({});
});

describe("ClassAppearanceSection", () => {
  it("job combobox picks a named job as a number", async () => {
    const onSetField = vi.fn();
    render(
      <ClassAppearanceSection
        attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }}
        onSetField={onSetField}
      />,
    );
    await userEvent.click(screen.getByRole("combobox", { name: /class/i }));
    await userEvent.click(
      await screen.findByRole("option", { name: /warrior/i }),
    );
    expect(onSetField).toHaveBeenCalledWith("jobId", 100);
    const call = onSetField.mock.calls.find(([field]) => field === "jobId");
    expect(typeof call![1]).toBe("number");
  });

  it("job combobox accepts an arbitrary numeric id via the escape hatch", async () => {
    const onSetField = vi.fn();
    render(
      <ClassAppearanceSection
        attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }}
        onSetField={onSetField}
      />,
    );
    await userEvent.click(screen.getByRole("combobox", { name: /class/i }));
    await userEvent.type(
      screen.getByPlaceholderText(/search by name/i),
      "123456",
    );
    await userEvent.click(
      await screen.findByRole("option", { name: /use id 123456/i }),
    );
    expect(onSetField).toHaveBeenCalledWith("jobId", 123456);
  });

  it("skin thumb click replaces skinColor", async () => {
    const onSetField = vi.fn();
    render(
      <ClassAppearanceSection
        attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }}
        onSetField={onSetField}
      />,
    );
    await userEvent.click(screen.getByRole("button", { name: /skin tone 3/i }));
    expect(onSetField).toHaveBeenCalledWith("skinColor", 3);
  });

  it("gender select toggles 0/1", async () => {
    const onSetField = vi.fn();
    render(
      <ClassAppearanceSection
        attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }}
        onSetField={onSetField}
      />,
    );
    await userEvent.click(screen.getByRole("button", { name: /female/i }));
    expect(onSetField).toHaveBeenCalledWith("gender", 1);
  });

  it("starter rows are gender-filtered (male preset shows no female ids)", () => {
    render(
      <ClassAppearanceSection
        attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }}
        onSetField={vi.fn()}
      />,
    );
    expect(
      screen.getByRole("button", { name: /^hair 30000$/i }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /^hair 31000$/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /^face 21000$/i }),
    ).not.toBeInTheDocument();
  });

  it("hair color row offers only the selected hair's existing variants", () => {
    render(
      <ClassAppearanceSection
        attrs={{ ...DEFAULT_PRESET_ATTRIBUTES, hair: 30030 }}
        onSetField={vi.fn()}
      />,
    );
    // 30030 exists in colors 0 and 1 only.
    expect(
      screen.getByRole("button", { name: /^hair color 0$/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /^hair color 1$/i }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /^hair color 2$/i }),
    ).not.toBeInTheDocument();
  });

  it("selecting a hair without the current color snaps hairColor to a valid digit", async () => {
    const onSetField = vi.fn();
    render(
      <ClassAppearanceSection
        attrs={{ ...DEFAULT_PRESET_ATTRIBUTES, hair: 30030, hairColor: 1 }}
        onSetField={onSetField}
      />,
    );
    // 30000 has colors 0 and 2 — current color 1 doesn't exist for it.
    await userEvent.click(
      screen.getByRole("button", { name: /^hair 30000$/i }),
    );
    expect(onSetField).toHaveBeenCalledWith("hair", 30000);
    expect(onSetField).toHaveBeenCalledWith("hairColor", 0);
  });

  it("selecting a hair that has the current color does NOT touch hairColor", async () => {
    const onSetField = vi.fn();
    render(
      <ClassAppearanceSection
        attrs={{ ...DEFAULT_PRESET_ATTRIBUTES, hair: 30000, hairColor: 0 }}
        onSetField={onSetField}
      />,
    );
    await userEvent.click(
      screen.getByRole("button", { name: /^hair 30030$/i }),
    );
    expect(onSetField).toHaveBeenCalledWith("hair", 30030);
    expect(onSetField).not.toHaveBeenCalledWith("hairColor", expect.anything());
  });

  it("browsing faces and picking a candidate replaces face and closes the dialog", async () => {
    const onSetField = vi.fn();
    render(
      <ClassAppearanceSection
        attrs={{ ...DEFAULT_PRESET_ATTRIBUTES }}
        onSetField={onSetField}
      />,
    );

    await userEvent.click(
      screen.getByRole("button", { name: /browse faces/i }),
    );
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

  it("browsing hairs shows one base entry per hair and snaps color on pick", async () => {
    const onSetField = vi.fn();
    render(
      <ClassAppearanceSection
        attrs={{ ...DEFAULT_PRESET_ATTRIBUTES, hair: 30030, hairColor: 1 }}
        onSetField={onSetField}
      />,
    );

    await userEvent.click(
      screen.getByRole("button", { name: /browse hairs/i }),
    );
    expect(await screen.findByRole("dialog")).toBeInTheDocument();

    // Variants collapse to bases: 30002 is 30000's color-2 variant, not a row.
    expect(
      screen.getByRole("button", { name: /add hair 30000/i }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /add hair 30002/i }),
    ).not.toBeInTheDocument();

    await userEvent.click(
      screen.getByRole("button", { name: /add hair 30000/i }),
    );
    expect(onSetField).toHaveBeenCalledWith("hair", 30000);
    // Current color 1 doesn't exist for 30000 (colors 0,2) — snapped.
    expect(onSetField).toHaveBeenCalledWith("hairColor", 0);
  });
});
