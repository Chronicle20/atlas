import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { StartingKitSection } from "../StartingKitSection";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: null }),
}));
vi.mock("@/lib/hooks/api/useItemStrings", () => ({
  useItemName: () => ({ data: "Item", isError: false }),
}));
// ItemSearchCombobox's real prop is `onAdd` (see templates/ItemSearchCombobox.tsx) —
// mirrored from presets/__tests__/EquipmentSection.test.tsx and
// presets/__tests__/InventorySection.test.tsx rather than the brief draft's
// `onSelect`.
vi.mock("../../templates/ItemSearchCombobox", () => ({
  ItemSearchCombobox: ({ onAdd }: { onAdd: (id: number) => void }) => (
    <button aria-label="combo-add" onClick={() => onAdd(1040002)}>
      combo
    </button>
  ),
}));

describe("StartingKitSection", () => {
  it("renders an equipment row per entry", () => {
    render(
      <StartingKitSection
        equipment={[{ templateId: 1040021, useAverageStats: true }]}
        inventory={[]}
        onAddEquipment={vi.fn()}
        onRemoveEquipment={vi.fn()}
        onSetEquipmentAvg={vi.fn()}
        onAddInventory={vi.fn()}
        onRemoveInventory={vi.fn()}
        onSetInventoryQty={vi.fn()}
      />,
    );
    expect(screen.getByText("1040021")).toBeInTheDocument();
  });

  it("renders an inventory row per entry", () => {
    render(
      <StartingKitSection
        equipment={[]}
        inventory={[{ templateId: 2000002, quantity: 100 }]}
        onAddEquipment={vi.fn()}
        onRemoveEquipment={vi.fn()}
        onSetEquipmentAvg={vi.fn()}
        onAddInventory={vi.fn()}
        onRemoveInventory={vi.fn()}
        onSetInventoryQty={vi.fn()}
      />,
    );
    expect(screen.getByText("2000002")).toBeInTheDocument();
  });

  it("toggling average stats reports the index", async () => {
    const user = userEvent.setup();
    const onSetEquipmentAvg = vi.fn();
    render(
      <StartingKitSection
        equipment={[{ templateId: 1040021, useAverageStats: true }]}
        inventory={[]}
        onAddEquipment={vi.fn()}
        onRemoveEquipment={vi.fn()}
        onSetEquipmentAvg={onSetEquipmentAvg}
        onAddInventory={vi.fn()}
        onRemoveInventory={vi.fn()}
        onSetInventoryQty={vi.fn()}
      />,
    );
    await user.click(screen.getByRole("switch", { name: /average stats/i }));
    expect(onSetEquipmentAvg).toHaveBeenCalledWith(0, false);
  });

  it("removing an inventory entry reports the index", async () => {
    const user = userEvent.setup();
    const onRemoveInventory = vi.fn();
    render(
      <StartingKitSection
        equipment={[]}
        inventory={[{ templateId: 2000002, quantity: 100 }]}
        onAddEquipment={vi.fn()}
        onRemoveEquipment={vi.fn()}
        onSetEquipmentAvg={vi.fn()}
        onAddInventory={vi.fn()}
        onRemoveInventory={onRemoveInventory}
        onSetInventoryQty={vi.fn()}
      />,
    );
    await user.click(
      screen.getByRole("button", { name: /remove item 2000002/i }),
    );
    expect(onRemoveInventory).toHaveBeenCalledWith(0);
  });
});
