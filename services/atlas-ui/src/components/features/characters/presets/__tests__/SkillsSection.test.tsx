import { useEffect, useReducer } from "react";
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
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

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: null }),
}));
const useSkillDataMock = vi.fn();
vi.mock("@/lib/hooks/useSkillData", () => ({
  useSkillData: (...a: unknown[]) => useSkillDataMock(...a),
}));
// SkillSearchCombobox/JobSkillsAddButton real props are onAdd/onAddMany —
// mocked so section tests don't need a QueryClient (their own behavior is
// covered in their dedicated test files).
vi.mock("../SkillSearchCombobox", () => ({
  SkillSearchCombobox: ({ onAdd }: { onAdd: (id: number) => void }) => (
    <button aria-label="combo-add" onClick={() => onAdd(1001004)}>
      combo
    </button>
  ),
}));
vi.mock("../JobSkillsAddButton", () => ({
  JobSkillsAddButton: ({
    onAddMany,
  }: {
    onAddMany: (ids: number[]) => void;
  }) => (
    <button aria-label="job-skills-add" onClick={() => onAddMany([1001, 1002])}>
      job
    </button>
  ),
}));

beforeEach(() => {
  useSkillDataMock.mockReturnValue({
    data: { name: "Power Strike", maxLevel: 20 },
    isError: false,
  });
});

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
  onReady: (
    state: PresetEditorState,
    dispatch: React.Dispatch<PresetEditorAction>,
  ) => void;
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
      onAdd={(skillId) =>
        dispatch({ type: "addSkill", key: preset.key, skillId })
      }
      onAddMany={(skillIds) =>
        dispatch({ type: "addSkills", key: preset.key, skillIds })
      }
      onRemove={(index) =>
        dispatch({ type: "removeSkill", key: preset.key, index })
      }
      onSetLevel={(index, value) =>
        dispatch({ type: "setSkillLevel", key: preset.key, index, value })
      }
    />
  );
}

describe("SkillsSection", () => {
  it("shows empty copy when no skills", () => {
    render(
      <SkillsSection
        skills={[]}
        onAdd={vi.fn()}
        onAddMany={vi.fn()}
        onRemove={vi.fn()}
        onSetLevel={vi.fn()}
      />,
    );
    expect(screen.getByText(/grants no skills/i)).toBeInTheDocument();
  });

  it("adds via the search combobox", async () => {
    const onAdd = vi.fn();
    render(
      <SkillsSection
        skills={[]}
        onAdd={onAdd}
        onAddMany={vi.fn()}
        onRemove={vi.fn()}
        onSetLevel={vi.fn()}
      />,
    );
    await userEvent.click(screen.getByLabelText("combo-add"));
    expect(onAdd).toHaveBeenCalledWith(1001004);
  });

  it("edits level (min 1) and removes", async () => {
    const onSetLevel = vi.fn();
    const onRemove = vi.fn();
    render(
      <SkillsSection
        skills={[{ skillId: 1001004, level: 1 }]}
        onAdd={vi.fn()}
        onAddMany={vi.fn()}
        onRemove={onRemove}
        onSetLevel={onSetLevel}
      />,
    );
    const lvl = screen.getByLabelText(/level/i);
    await userEvent.clear(lvl);
    await userEvent.type(lvl, "5");
    expect(onSetLevel).toHaveBeenCalledWith(0, 5);
    await userEvent.click(
      screen.getByRole("button", { name: /remove skill 1001004/i }),
    );
    expect(onRemove).toHaveBeenCalledWith(0);
  });

  it("shows the skill's max level and clamps input above it", async () => {
    const onSetLevel = vi.fn();
    render(
      <SkillsSection
        skills={[{ skillId: 1001004, level: 1 }]}
        onAdd={vi.fn()}
        onAddMany={vi.fn()}
        onRemove={vi.fn()}
        onSetLevel={onSetLevel}
      />,
    );
    // maxLevel 20 from the mocked useSkillData is shown inline.
    expect(screen.getByText("/ 20")).toBeInTheDocument();
    const lvl = screen.getByLabelText(/level/i);
    await userEvent.clear(lvl);
    await userEvent.type(lvl, "99");
    // Every committed value is clamped to the max — never above 20.
    expect(onSetLevel.mock.calls.every(([, v]) => (v as number) <= 20)).toBe(
      true,
    );
    expect(onSetLevel).toHaveBeenCalledWith(0, 20);
  });

  it("does not clamp when the skill has no known max level", async () => {
    useSkillDataMock.mockReturnValue({
      data: { name: "Power Strike" }, // no maxLevel
      isError: false,
    });
    const onSetLevel = vi.fn();
    render(
      <SkillsSection
        skills={[{ skillId: 1001004, level: 1 }]}
        onAdd={vi.fn()}
        onAddMany={vi.fn()}
        onRemove={vi.fn()}
        onSetLevel={onSetLevel}
      />,
    );
    expect(screen.queryByText(/^\/ /)).not.toBeInTheDocument();
    const lvl = screen.getByLabelText(/level/i);
    await userEvent.clear(lvl);
    await userEvent.type(lvl, "99");
    expect(onSetLevel).toHaveBeenCalledWith(0, 99);
  });

  it("job-family button bulk-adds skill ids via onAddMany", async () => {
    const onAddMany = vi.fn();
    render(
      <SkillsSection
        skills={[]}
        onAdd={vi.fn()}
        onAddMany={onAddMany}
        onRemove={vi.fn()}
        onSetLevel={vi.fn()}
      />,
    );
    await userEvent.click(screen.getByLabelText("job-skills-add"));
    expect(onAddMany).toHaveBeenCalledWith([1001, 1002]);
  });

  it("reducer round-trip: clearing and retyping commits exactly the typed value, not a prepend", async () => {
    const seedPresets: CharacterPreset[] = [
      {
        id: "p1",
        attributes: {
          ...DEFAULT_PRESET_ATTRIBUTES,
          skills: [{ skillId: 1001004, level: 5 }],
        },
      },
    ];
    const loaded = presetReducer(initialPresetEditorState(), {
      type: "load",
      presets: seedPresets,
    });
    const initialState = presetReducer(loaded, { type: "select", key: "p1" });

    let latest = initialState;
    render(
      <Harness
        initialState={initialState}
        onReady={(s) => {
          latest = s;
        }}
      />,
    );

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
        attributes: {
          ...DEFAULT_PRESET_ATTRIBUTES,
          skills: [{ skillId: 1001004, level: 5 }],
        },
      },
      {
        id: "p2",
        attributes: {
          ...DEFAULT_PRESET_ATTRIBUTES,
          skills: [{ skillId: 1001004, level: 20 }],
        },
      },
    ];
    const loaded = presetReducer(initialPresetEditorState(), {
      type: "load",
      presets: seedPresets,
    });
    const initialState = presetReducer(loaded, { type: "select", key: "p1" });

    let dispatchFn: React.Dispatch<PresetEditorAction> = () => {};
    render(
      <Harness
        initialState={initialState}
        onReady={(_s, d) => {
          dispatchFn = d;
        }}
      />,
    );

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
