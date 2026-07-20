import { useReducer } from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { PresetEditor, type PresetEditorProps } from "../PresetEditor";
import {
  DEFAULT_PRESET_ATTRIBUTES,
  initialPresetEditorState,
  presetReducer,
  selectedPreset,
} from "../presetEditorState";
import type { CharacterPreset } from "@/types/models/template";

// Mock the heavy leaf sections/preview so this test targets assembly + kebab only.
vi.mock("../PresetPreviewCard", () => ({
  PresetPreviewCard: () => <div data-testid="preview" />,
}));
vi.mock("../ClassAppearanceSection", () => ({
  ClassAppearanceSection: () => <div />,
}));
vi.mock("../SpawnProgressionSection", () => ({
  SpawnProgressionSection: () => <div />,
}));
vi.mock("../EquipmentSection", () => ({ EquipmentSection: () => <div /> }));
vi.mock("../InventorySection", () => ({ InventorySection: () => <div /> }));
vi.mock("../SkillsSection", () => ({ SkillsSection: () => <div /> }));

const base = {
  key: "a1",
  id: "a1",
  attributes: { ...DEFAULT_PRESET_ATTRIBUTES, name: "Hero" },
};
// Typed explicitly (rather than `Object.fromEntries(...) as never`) so it
// can be spread into <PresetEditor> under this repo's strict tsconfig
// (exactOptionalPropertyTypes etc.) without a cast.
const handlers: Omit<PresetEditorProps, "preset" | "onBack" | "onApply"> = {
  onSetField: vi.fn(),
  onAddTag: vi.fn(),
  onRemoveTag: vi.fn(),
  onAddEquip: vi.fn(),
  onRemoveEquip: vi.fn(),
  onSetEquipAvg: vi.fn(),
  onAddInventory: vi.fn(),
  onRemoveInventory: vi.fn(),
  onSetInventoryQty: vi.fn(),
  onAddSkill: vi.fn(),
  onRemoveSkill: vi.fn(),
  onSetSkillLevel: vi.fn(),
  onDuplicate: vi.fn(),
  onRemove: vi.fn(),
};

describe("PresetEditor", () => {
  it("renders backlink, preview, and header name", () => {
    render(<PresetEditor preset={base} onBack={vi.fn()} {...handlers} />);
    expect(
      screen.getByRole("button", { name: /preset library/i }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("preview")).toBeInTheDocument();
    expect(screen.getByText("Hero")).toBeInTheDocument();
  });

  it("backlink calls onBack", async () => {
    const onBack = vi.fn();
    render(<PresetEditor preset={base} onBack={onBack} {...handlers} />);
    await userEvent.click(
      screen.getByRole("button", { name: /preset library/i }),
    );
    expect(onBack).toHaveBeenCalled();
  });

  it("kebab hides Apply when onApply is absent", async () => {
    render(<PresetEditor preset={base} onBack={vi.fn()} {...handlers} />);
    await userEvent.click(
      screen.getByRole("button", { name: /preset actions/i }),
    );
    expect(screen.queryByText(/apply to an account/i)).toBeNull();
  });

  it("kebab shows Apply when onApply is present", async () => {
    render(
      <PresetEditor
        preset={base}
        onBack={vi.fn()}
        onApply={vi.fn()}
        {...handlers}
      />,
    );
    await userEvent.click(
      screen.getByRole("button", { name: /preset actions/i }),
    );
    expect(await screen.findByText(/apply to an account/i)).toBeInTheDocument();
  });

  it("shows job badge and level, with GM suffix only when gm > 0", () => {
    // name deliberately != the job label ("Hero") so the two assertions
    // below can't accidentally collide on the same text node.
    const gm = {
      ...base,
      attributes: {
        ...base.attributes,
        name: "My Warrior",
        jobId: 112,
        level: 30,
        gm: 0,
      },
    };
    const { rerender } = render(
      <PresetEditor preset={gm} onBack={vi.fn()} {...handlers} />,
    );
    expect(screen.getByText("My Warrior")).toBeInTheDocument();
    expect(screen.getByText("Hero")).toBeInTheDocument(); // job badge (jobId 112 = "Hero")
    expect(screen.getByText("Lv 30")).toBeInTheDocument();

    rerender(
      <PresetEditor
        preset={{ ...gm, attributes: { ...gm.attributes, gm: 3 } }}
        onBack={vi.fn()}
        {...handlers}
      />,
    );
    expect(screen.getByText("Lv 30 · GM 3")).toBeInTheDocument();
  });

  it("Apply is disabled with a reachable hint when the preset has no id yet (unsaved)", async () => {
    const unsaved = { key: "local-0", attributes: base.attributes };
    render(
      <PresetEditor
        preset={unsaved}
        onBack={vi.fn()}
        onApply={vi.fn()}
        {...handlers}
      />,
    );
    await userEvent.click(
      screen.getByRole("button", { name: /preset actions/i }),
    );
    const item = await screen.findByText(/apply to an account/i);
    expect(item.closest('[role="menuitem"]')).toHaveAttribute(
      "aria-disabled",
      "true",
    );
    // The hint must be always-visible DOM text (not a `title` on a
    // pointer-events:none disabled item, which never becomes hoverable).
    expect(
      screen.getByText(/save this preset before applying/i),
    ).toBeInTheDocument();
  });

  it("Apply hint is absent once the preset is saved (Apply enabled)", async () => {
    render(
      <PresetEditor
        preset={base}
        onBack={vi.fn()}
        onApply={vi.fn()}
        {...handlers}
      />,
    );
    await userEvent.click(
      screen.getByRole("button", { name: /preset actions/i }),
    );
    await screen.findByText(/apply to an account/i);
    expect(screen.queryByText(/save this preset before applying/i)).toBeNull();
  });

  it("kebab Duplicate/Remove call the editor's onDuplicate/onRemove", async () => {
    const onDuplicate = vi.fn();
    const onRemove = vi.fn();
    render(
      <PresetEditor
        preset={base}
        onBack={vi.fn()}
        {...handlers}
        onDuplicate={onDuplicate}
        onRemove={onRemove}
      />,
    );
    await userEvent.click(
      screen.getByRole("button", { name: /preset actions/i }),
    );
    await userEvent.click(screen.getByRole("menuitem", { name: /duplicate/i }));
    expect(onDuplicate).toHaveBeenCalledTimes(1);

    await userEvent.click(
      screen.getByRole("button", { name: /preset actions/i }),
    );
    await userEvent.click(screen.getByRole("menuitem", { name: /^remove$/i }));
    // Remove requires AlertDialog confirm before firing.
    expect(onRemove).not.toHaveBeenCalled();
    await userEvent.click(screen.getByRole("button", { name: /^remove$/i }));
    expect(onRemove).toHaveBeenCalledTimes(1);
  });

  it("kebab Apply fires onApply when enabled", async () => {
    const onApply = vi.fn();
    render(
      <PresetEditor
        preset={base}
        onBack={vi.fn()}
        onApply={onApply}
        {...handlers}
      />,
    );
    await userEvent.click(
      screen.getByRole("button", { name: /preset actions/i }),
    );
    await userEvent.click(
      screen.getByRole("menuitem", { name: /apply to an account/i }),
    );
    expect(onApply).toHaveBeenCalledTimes(1);
  });
});

describe("PresetEditor wired to the real reducer", () => {
  // A real useReducer host (mirrors how the eventual page will drive
  // PresetEditor) so each dispatch causes a genuine re-render and the
  // controlled inputs reflect committed state — not just a spy call shape.
  // This is what proves a wiring mistake, especially BaseStats' bare stat
  // name, actually corrupts the round-tripped preset.
  function Host({
    initial,
    onSetFieldSpy,
  }: {
    initial: CharacterPreset[];
    onSetFieldSpy: (path: string, value: number | string) => void;
  }) {
    const [state, dispatch] = useReducer(
      presetReducer,
      initialPresetEditorState(),
    );
    const loaded = state.loaded
      ? state
      : presetReducer(state, { type: "load", presets: initial });
    if (!state.loaded) {
      dispatch({ type: "load", presets: initial });
    }
    const key = loaded.presets[0]?.key ?? null;
    const preset = selectedPreset({ ...loaded, selectedKey: key });
    if (!preset || !key) return null;

    return (
      <PresetEditor
        preset={preset}
        onBack={vi.fn()}
        {...handlers}
        onSetField={(path, value) => {
          onSetFieldSpy(path, value);
          dispatch({ type: "setField", key, path, value });
        }}
      />
    );
  }

  it("BaseStats edit dispatches setField with path stats.<stat>, not the bare stat name", async () => {
    const onSetField = vi.fn();
    render(
      <Host
        initial={[
          {
            id: "a1",
            attributes: { ...DEFAULT_PRESET_ATTRIBUTES, name: "Hero" },
          },
        ]}
        onSetFieldSpy={onSetField}
      />,
    );

    const strInput = screen.getByLabelText("STR");
    await userEvent.clear(strInput);
    await userEvent.type(strInput, "9");

    // Every call must use the composed "stats.str" path — never bare "str".
    for (const call of onSetField.mock.calls) {
      expect(call[0]).not.toBe("str");
    }
    expect(onSetField).toHaveBeenCalledWith("stats.str", 9);
    expect((strInput as HTMLInputElement).value).toBe("9");
  });

  it("Identity edits reach the reducer via onSetField with the plain field path", async () => {
    const onSetField = vi.fn();
    render(
      <Host
        initial={[
          {
            id: "a1",
            attributes: { ...DEFAULT_PRESET_ATTRIBUTES, name: "Hero" },
          },
        ]}
        onSetFieldSpy={onSetField}
      />,
    );

    const nameInput = screen.getByLabelText("Name");
    await userEvent.clear(nameInput);
    await userEvent.type(nameInput, "Villain");

    expect(onSetField).toHaveBeenCalledWith("name", "Villain");
    expect((nameInput as HTMLInputElement).value).toBe("Villain");
  });
});

// Equipment/Inventory/Skills index-callback wiring (real DOM interaction
// through the unmocked sections + a real reducer round-trip) lives in
// PresetEditor.sections.test.tsx — those three sections are mocked to a
// trivial stand-in above, so asserting through them here would only prove a
// locally-defined handler works, not that PresetEditor actually wires it in.
