import { useEffect, useReducer } from "react";
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { InventorySection } from "../InventorySection";
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
vi.mock("@/lib/hooks/api/useItemStrings", () => ({
  useItemName: () => ({ data: "Item", isError: false }),
}));
// ItemSearchCombobox's real prop is `onAdd` (see templates/ItemSearchCombobox.tsx) —
// mirrored here rather than the brief draft's `onSelect`.
vi.mock("../../templates/ItemSearchCombobox", () => ({
  ItemSearchCombobox: ({ onAdd }: { onAdd: (id: number) => void }) => (
    <button aria-label="combo-add" onClick={() => onAdd(2000000)}>
      combo
    </button>
  ),
}));

/**
 * Wires InventorySection to the REAL presetReducer (not inert vi.fn() mocks)
 * so a keystroke's dispatch actually round-trips through the reducer's
 * Math.max(1, value) clamp back into the `value` prop, the same way it does
 * in the live editor. `onReady` is invoked after every render with the
 * current state and the raw dispatch fn so tests can both assert on
 * committed reducer state and drive preset switches.
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
    <InventorySection
      inventory={preset.attributes.inventory}
      onAdd={(templateId) =>
        dispatch({ type: "addInventory", key: preset.key, templateId })
      }
      onRemove={(index) =>
        dispatch({ type: "removeInventory", key: preset.key, index })
      }
      onSetQty={(index, value) =>
        dispatch({ type: "setInventoryQty", key: preset.key, index, value })
      }
    />
  );
}

describe("InventorySection", () => {
  it("shows empty copy when no items", () => {
    render(
      <InventorySection
        inventory={[]}
        onAdd={vi.fn()}
        onRemove={vi.fn()}
        onSetQty={vi.fn()}
      />,
    );
    expect(screen.getByText(/no granted items/i)).toBeInTheDocument();
  });

  it("edits quantity (min 1) and removes", async () => {
    const onSetQty = vi.fn();
    const onRemove = vi.fn();
    render(
      <InventorySection
        inventory={[{ templateId: 2000000, quantity: 1 }]}
        onAdd={vi.fn()}
        onRemove={onRemove}
        onSetQty={onSetQty}
      />,
    );
    const qty = screen.getByLabelText(/quantity/i);
    await userEvent.clear(qty);
    await userEvent.type(qty, "10");
    expect(onSetQty).toHaveBeenCalledWith(0, 10);
    await userEvent.click(
      screen.getByRole("button", { name: /remove item 2000000/i }),
    );
    expect(onRemove).toHaveBeenCalledWith(0);
  });

  it("adds via the search combobox", async () => {
    const onAdd = vi.fn();
    render(
      <InventorySection
        inventory={[]}
        onAdd={onAdd}
        onRemove={vi.fn()}
        onSetQty={vi.fn()}
      />,
    );
    await userEvent.click(screen.getByLabelText("combo-add"));
    expect(onAdd).toHaveBeenCalledWith(2000000);
  });

  it("adds via manual id fallback", async () => {
    const onAdd = vi.fn();
    render(
      <InventorySection
        inventory={[]}
        onAdd={onAdd}
        onRemove={vi.fn()}
        onSetQty={vi.fn()}
      />,
    );
    const manual = screen.getByLabelText(/manual item id/i);
    await userEvent.type(manual, "2000001");
    await userEvent.click(screen.getByRole("button", { name: /add item id/i }));
    expect(onAdd).toHaveBeenCalledWith(2000001);
  });

  it("reducer round-trip: clearing and retyping commits exactly the typed value, not a prepend", async () => {
    const seedPresets: CharacterPreset[] = [
      {
        id: "p1",
        attributes: {
          ...DEFAULT_PRESET_ATTRIBUTES,
          inventory: [{ templateId: 2000000, quantity: 5 }],
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

    const qty = screen.getByLabelText(/quantity/i);
    await userEvent.clear(qty);
    await userEvent.type(qty, "8");
    await userEvent.tab(); // blur

    const preset = latest.presets.find((p) => p.key === "p1");
    expect(preset?.attributes.inventory[0]?.quantity).toBe(8);
  });

  it("resyncs the draft from a new value when NOT focused (preset switch)", async () => {
    const seedPresets: CharacterPreset[] = [
      {
        id: "p1",
        attributes: {
          ...DEFAULT_PRESET_ATTRIBUTES,
          inventory: [{ templateId: 2000000, quantity: 5 }],
        },
      },
      {
        id: "p2",
        attributes: {
          ...DEFAULT_PRESET_ATTRIBUTES,
          inventory: [{ templateId: 2000000, quantity: 20 }],
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

    const qty = screen.getByLabelText(/quantity/i);
    expect(qty).toHaveValue(5);
    expect(qty).not.toHaveFocus();

    // Switching presets while the field is NOT focused must resync the draft.
    act(() => {
      dispatchFn({ type: "select", key: "p2" });
    });

    expect(screen.getByLabelText(/quantity/i)).toHaveValue(20);
  });
});
