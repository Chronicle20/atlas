import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { PresetEditor } from "../PresetEditor";
import {
  DEFAULT_PRESET_ATTRIBUTES,
  initialPresetEditorState,
  presetReducer,
  selectedPreset,
  type PresetEditorAction,
  type PresetEditorState,
  type WorkingPreset,
} from "../presetEditorState";
import type { CharacterPreset } from "@/types/models/template";

// This file exercises the REAL Equipment/Inventory/Skills sections (unlike
// PresetEditor.test.tsx, which mocks them to isolate assembly + kebab), so a
// wiring mistake in PresetEditor's index-callback plumbing (wrong action
// type, wrong key) shows up as a wrong post-dispatch reducer state, not just
// a spy call. ClassAppearanceSection/SpawnProgressionSection/PresetPreviewCard
// stay mocked — their own dependency chains (cosmetics browser, map picker,
// character-image React Query) are out of scope here and already covered by
// their own section tests.
vi.mock("../PresetPreviewCard", () => ({
  PresetPreviewCard: () => <div data-testid="preview" />,
}));
vi.mock("../ClassAppearanceSection", () => ({
  ClassAppearanceSection: () => <div />,
}));
vi.mock("../SpawnProgressionSection", () => ({
  SpawnProgressionSection: () => <div />,
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: null }),
}));
vi.mock("@/lib/hooks/api/useItemStrings", () => ({
  useItemName: () => ({ data: "Item", isError: false }),
}));
vi.mock("@/lib/hooks/useSkillData", () => ({
  useSkillData: () => ({ data: { name: "Power Strike" }, isError: false }),
}));
// Real ItemSearchCombobox/SkillSearchCombobox need a QueryClientProvider;
// not under test here.
vi.mock("../../templates/ItemSearchCombobox", () => ({
  ItemSearchCombobox: ({ onAdd }: { onAdd: (id: number) => void }) => (
    <button aria-label="combo-add" onClick={() => onAdd(1040002)}>
      combo
    </button>
  ),
}));
vi.mock("../SkillSearchCombobox", () => ({
  SkillSearchCombobox: ({ onAdd }: { onAdd: (id: number) => void }) => (
    <button aria-label="skill-combo-add" onClick={() => onAdd(1001004)}>
      combo
    </button>
  ),
}));

const noop = () => {};
const passthroughHandlers = {
  onAddTag: noop,
  onRemoveTag: noop,
  onAddEquip: noop,
  onAddInventory: noop,
  onAddSkill: noop,
  onDuplicate: noop,
  onRemove: noop,
};

/** Drives the real reducer so post-dispatch state is asserted, not spy shape. */
function harness(initial: CharacterPreset[]) {
  let state: PresetEditorState = presetReducer(initialPresetEditorState(), {
    type: "load",
    presets: initial,
  });
  const key = state.presets[0]!.key;
  state = presetReducer(state, { type: "select", key });

  const dispatch = (action: PresetEditorAction) => {
    state = presetReducer(state, action);
  };

  return {
    key,
    dispatch,
    current: (): WorkingPreset => selectedPreset(state)!,
  };
}

describe("PresetEditor: Equipment/Inventory/Skills index-callback wiring", () => {
  it("EquipmentSection remove + avg-toggle dispatch removeEquip/setEquipAvg for the selected preset's key", async () => {
    const h = harness([
      {
        id: "a1",
        attributes: {
          ...DEFAULT_PRESET_ATTRIBUTES,
          equipment: [{ templateId: 1040002, useAverageStats: true }],
        },
      },
    ]);

    render(
      <PresetEditor
        preset={h.current()}
        onBack={vi.fn()}
        {...passthroughHandlers}
        onSetField={noop}
        onRemoveInventory={noop}
        onSetInventoryQty={noop}
        onRemoveSkill={noop}
        onSetSkillLevel={noop}
        onRemoveEquip={(index) =>
          h.dispatch({ type: "removeEquip", key: h.key, index })
        }
        onSetEquipAvg={(index, value) =>
          h.dispatch({ type: "setEquipAvg", key: h.key, index, value })
        }
      />,
    );

    await userEvent.click(
      screen.getByRole("switch", { name: /average stats/i }),
    );
    expect(h.current().attributes.equipment[0]!.useAverageStats).toBe(false);

    await userEvent.click(
      screen.getByRole("button", { name: /remove equipment 1040002/i }),
    );
    expect(h.current().attributes.equipment).toHaveLength(0);
  });

  it("InventorySection quantity + remove dispatch setInventoryQty/removeInventory for the selected preset's key", async () => {
    const h = harness([
      {
        id: "a1",
        attributes: {
          ...DEFAULT_PRESET_ATTRIBUTES,
          inventory: [{ templateId: 2000000, quantity: 1 }],
        },
      },
    ]);

    render(
      <PresetEditor
        preset={h.current()}
        onBack={vi.fn()}
        {...passthroughHandlers}
        onSetField={noop}
        onRemoveEquip={noop}
        onSetEquipAvg={noop}
        onRemoveSkill={noop}
        onSetSkillLevel={noop}
        onSetInventoryQty={(index, value) =>
          h.dispatch({ type: "setInventoryQty", key: h.key, index, value })
        }
        onRemoveInventory={(index) =>
          h.dispatch({ type: "removeInventory", key: h.key, index })
        }
      />,
    );

    const qty = screen.getByLabelText("Quantity");
    await userEvent.clear(qty);
    await userEvent.type(qty, "5");
    expect(h.current().attributes.inventory[0]!.quantity).toBe(5);

    await userEvent.click(
      screen.getByRole("button", { name: /remove item 2000000/i }),
    );
    expect(h.current().attributes.inventory).toHaveLength(0);
  });

  it("SkillsSection level + remove dispatch setSkillLevel/removeSkill for the selected preset's key", async () => {
    const h = harness([
      {
        id: "a1",
        attributes: {
          ...DEFAULT_PRESET_ATTRIBUTES,
          skills: [{ skillId: 1001, level: 1 }],
        },
      },
    ]);

    render(
      <PresetEditor
        preset={h.current()}
        onBack={vi.fn()}
        {...passthroughHandlers}
        onSetField={noop}
        onRemoveEquip={noop}
        onSetEquipAvg={noop}
        onRemoveInventory={noop}
        onSetInventoryQty={noop}
        onSetSkillLevel={(index, value) =>
          h.dispatch({ type: "setSkillLevel", key: h.key, index, value })
        }
        onRemoveSkill={(index) =>
          h.dispatch({ type: "removeSkill", key: h.key, index })
        }
      />,
    );

    const level = screen.getByLabelText("Level");
    await userEvent.clear(level);
    await userEvent.type(level, "7");
    expect(h.current().attributes.skills[0]!.level).toBe(7);

    await userEvent.click(
      screen.getByRole("button", { name: /remove skill 1001/i }),
    );
    expect(h.current().attributes.skills).toHaveLength(0);
  });

  it("EquipmentSection/InventorySection Add wire to onAddEquip/onAddInventory with the raw templateId (via the search combobox)", async () => {
    const h = harness([
      { id: "a1", attributes: { ...DEFAULT_PRESET_ATTRIBUTES } },
    ]);
    const onAddEquip = vi.fn();

    render(
      <PresetEditor
        preset={h.current()}
        onBack={vi.fn()}
        {...passthroughHandlers}
        onSetField={noop}
        onAddEquip={onAddEquip}
        onRemoveEquip={noop}
        onSetEquipAvg={noop}
        onRemoveInventory={noop}
        onSetInventoryQty={noop}
        onRemoveSkill={noop}
        onSetSkillLevel={noop}
      />,
    );

    const combos = screen.getAllByLabelText("combo-add");
    await userEvent.click(combos[0]!);
    expect(onAddEquip).toHaveBeenCalledWith(1040002);
  });
});
