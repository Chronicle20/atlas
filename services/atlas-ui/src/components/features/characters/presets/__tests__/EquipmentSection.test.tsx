import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { EquipmentSection } from "../EquipmentSection";

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
    <button aria-label="combo-add" onClick={() => onAdd(1040002)}>
      combo
    </button>
  ),
}));

describe("EquipmentSection", () => {
  it("lists rows with avg-stats toggle and removes", async () => {
    const onRemove = vi.fn();
    const onSetAvg = vi.fn();
    render(
      <EquipmentSection
        equipment={[{ templateId: 1040002, useAverageStats: true }]}
        onAdd={vi.fn()}
        onRemove={onRemove}
        onSetAvg={onSetAvg}
      />,
    );
    // The toggle carries a VISIBLE label, not just an aria-label.
    expect(screen.getByText(/avg stats/i)).toBeInTheDocument();
    await userEvent.click(
      screen.getByRole("switch", { name: /average stats/i }),
    );
    expect(onSetAvg).toHaveBeenCalledWith(0, false);
    await userEvent.click(
      screen.getByRole("button", { name: /remove equipment 1040002/i }),
    );
    expect(onRemove).toHaveBeenCalledWith(0);
  });

  it("adds via the search combobox", async () => {
    const onAdd = vi.fn();
    render(
      <EquipmentSection
        equipment={[]}
        onAdd={onAdd}
        onRemove={vi.fn()}
        onSetAvg={vi.fn()}
      />,
    );
    await userEvent.click(screen.getByLabelText("combo-add"));
    expect(onAdd).toHaveBeenCalledWith(1040002);
  });

  it("shows empty copy when there is no worn equipment", () => {
    render(
      <EquipmentSection
        equipment={[]}
        onAdd={vi.fn()}
        onRemove={vi.fn()}
        onSetAvg={vi.fn()}
      />,
    );
    expect(screen.getByText(/no worn items/i)).toBeInTheDocument();
  });
});
