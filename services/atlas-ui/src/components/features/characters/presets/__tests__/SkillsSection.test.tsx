import { useEffect, useReducer } from "react";
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { SkillsSection } from "../SkillsSection";
import {
  DEFAULT_PRESET_ATTRIBUTES,
  initialPresetEditorState,
  presetReducer,
  selectedPreset,
  type PresetEditorAction,
  type PresetEditorState,
} from "../presetEditorState";
import type { CharacterPreset } from "@/types/models/template";

vi.mock("@/context/tenant-context", () => ({ useTenant: () => ({ activeTenant: null }) }));
vi.mock("@/lib/hooks/useSkillData", () => ({
  useSkillData: () => ({ data: { name: "Power Strike" }, isError: false }),
}));

/**
 * Wires SkillsSection to the REAL presetReducer (not inert vi.fn() mocks) so
 * a keystroke's dispatch actually round-trips through the reducer's
 * Math.max(1, value) clamp back into the `value` prop, the same way it does
 * in the live editor. Modeled on InventorySection's Harness.
 */
function Harness({
  initialState,
  onReady,
}: {
  initialState: PresetEditorState;
  onReady: (state: PresetEditorState, dispatch: React.Dispatch<PresetEditorAction>) => void;
}) {
  const [state, dispatch] = useReducer(presetReducer, initialState);
  useEffect(() => {
    onReady(state, dispatch);
  });
  const preset = selectedPreset(state);
  if (!preset) return null;
  return (
    <SkillsSection
      skills={preset.attributes.skills}
      onAdd={(skillId) => dispatch({ type: "addSkill", key: preset.key, skillId })}
      onRemove={(index) => dispatch({ type: "removeSkill", key: preset.key, index })}
      onSetLevel={(index, value) => dispatch({ type: "setSkillLevel", key: preset.key, index, value })}
    />
  );
}

describe("SkillsSection", () => {
  it("shows empty copy when no skills", () => {
    render(<SkillsSection skills={[]} onAdd={vi.fn()} onRemove={vi.fn()} onSetLevel={vi.fn()} />);
    expect(screen.getByText(/grants no skills/i)).toBeInTheDocument();
  });

  it("adds by numeric id", async () => {
    const onAdd = vi.fn();
    render(<SkillsSection skills={[]} onAdd={onAdd} onRemove={vi.fn()} onSetLevel={vi.fn()} />);
    await userEvent.type(screen.getByLabelText(/skill id/i), "1001004");
    await userEvent.click(screen.getByRole("button", { name: /add skill/i }));
    expect(onAdd).toHaveBeenCalledWith(1001004);
  });

  it("edits level (min 1) and removes", async () => {
    const onSetLevel = vi.fn();
    const onRemove = vi.fn();
    render(<SkillsSection skills={[{ skillId: 1001004, level: 1 }]}
      onAdd={vi.fn()} onRemove={onRemove} onSetLevel={onSetLevel} />);
    const lvl = screen.getByLabelText(/level/i);
    await userEvent.clear(lvl);
    await userEvent.type(lvl, "5");
    expect(onSetLevel).toHaveBeenCalledWith(0, 5);
    await userEvent.click(screen.getByRole("button", { name: /remove skill 1001004/i }));
    expect(onRemove).toHaveBeenCalledWith(0);
  });

  it("reducer round-trip: clearing and retyping commits exactly the typed value, not a prepend", async () => {
    const seedPresets: CharacterPreset[] = [
      {
        id: "p1",
        attributes: { ...DEFAULT_PRESET_ATTRIBUTES, skills: [{ skillId: 1001004, level: 5 }] },
      },
    ];
    const loaded = presetReducer(initialPresetEditorState(), { type: "load", presets: seedPresets });
    const initialState = presetReducer(loaded, { type: "select", key: "p1" });

    let latest = initialState;
    render(<Harness initialState={initialState} onReady={(s) => { latest = s; }} />);

    const lvl = screen.getByLabelText(/level/i);
    await userEvent.clear(lvl);
    await userEvent.type(lvl, "8");
    await userEvent.tab(); // blur

    const preset = latest.presets.find((p) => p.key === "p1");
    expect(preset?.attributes.skills[0]?.level).toBe(8);
  });

  it("resyncs the draft from a new value when NOT focused (preset switch)", async () => {
    const seedPresets: CharacterPreset[] = [
      {
        id: "p1",
        attributes: { ...DEFAULT_PRESET_ATTRIBUTES, skills: [{ skillId: 1001004, level: 5 }] },
      },
      {
        id: "p2",
        attributes: { ...DEFAULT_PRESET_ATTRIBUTES, skills: [{ skillId: 1001004, level: 20 }] },
      },
    ];
    const loaded = presetReducer(initialPresetEditorState(), { type: "load", presets: seedPresets });
    const initialState = presetReducer(loaded, { type: "select", key: "p1" });

    let dispatchFn: React.Dispatch<PresetEditorAction> = () => {};
    render(<Harness initialState={initialState} onReady={(_s, d) => { dispatchFn = d; }} />);

    const lvl = screen.getByLabelText(/level/i);
    expect(lvl).toHaveValue(5);
    expect(lvl).not.toHaveFocus();

    // Switching presets while the field is NOT focused must resync the draft.
    act(() => {
      dispatchFn({ type: "select", key: "p2" });
    });

    expect(screen.getByLabelText(/level/i)).toHaveValue(20);
  });
});
