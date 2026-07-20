import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { InventorySection } from "../InventorySection";

vi.mock("@/context/tenant-context", () => ({ useTenant: () => ({ activeTenant: null }) }));
vi.mock("@/lib/hooks/api/useItemStrings", () => ({ useItemName: () => ({ data: "Item", isError: false }) }));
// ItemSearchCombobox's real prop is `onAdd` (see templates/ItemSearchCombobox.tsx) —
// mirrored here rather than the brief draft's `onSelect`.
vi.mock("../../templates/ItemSearchCombobox", () => ({
  ItemSearchCombobox: ({ onAdd }: { onAdd: (id: number) => void }) => (
    <button aria-label="combo-add" onClick={() => onAdd(2000000)}>combo</button>
  ),
}));

describe("InventorySection", () => {
  it("shows empty copy when no items", () => {
    render(<InventorySection inventory={[]} onAdd={vi.fn()} onRemove={vi.fn()} onSetQty={vi.fn()} />);
    expect(screen.getByText(/no granted items/i)).toBeInTheDocument();
  });

  it("edits quantity (min 1) and removes", async () => {
    const onSetQty = vi.fn();
    const onRemove = vi.fn();
    render(<InventorySection inventory={[{ templateId: 2000000, quantity: 1 }]}
      onAdd={vi.fn()} onRemove={onRemove} onSetQty={onSetQty} />);
    const qty = screen.getByLabelText(/quantity/i);
    await userEvent.clear(qty);
    await userEvent.type(qty, "10");
    expect(onSetQty).toHaveBeenCalledWith(0, 10);
    await userEvent.click(screen.getByRole("button", { name: /remove item 2000000/i }));
    expect(onRemove).toHaveBeenCalledWith(0);
  });

  it("adds via the search combobox", async () => {
    const onAdd = vi.fn();
    render(<InventorySection inventory={[]} onAdd={onAdd} onRemove={vi.fn()} onSetQty={vi.fn()} />);
    await userEvent.click(screen.getByLabelText("combo-add"));
    expect(onAdd).toHaveBeenCalledWith(2000000);
  });

  it("adds via manual id fallback", async () => {
    const onAdd = vi.fn();
    render(<InventorySection inventory={[]} onAdd={onAdd} onRemove={vi.fn()} onSetQty={vi.fn()} />);
    const manual = screen.getByLabelText(/manual item id/i);
    await userEvent.type(manual, "2000001");
    await userEvent.click(screen.getByRole("button", { name: /add item id/i }));
    expect(onAdd).toHaveBeenCalledWith(2000001);
  });
});
