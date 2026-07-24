import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

const useItemNameMock = vi.fn();
vi.mock("@/lib/hooks/api/useItemStrings", () => ({
  useItemName: (...a: unknown[]) => useItemNameMock(...a),
  itemStringKeys: {
    all: ["item-strings"],
    byId: (id: string) => ["item-strings", "name", id],
  },
}));

vi.mock("../ItemSearchCombobox", () => ({
  ItemSearchCombobox: ({ onAdd }: { onAdd: (id: number) => void }) => (
    <button type="button" onClick={() => onAdd(1041002)}>
      mock-add
    </button>
  ),
}));

import { EquipmentPoolSection } from "../EquipmentPoolSection";

beforeEach(() => {
  useItemNameMock.mockReturnValue({ data: "Blue T-Shirt", isError: false });
});

describe("EquipmentPoolSection", () => {
  it("renders icon+name+id rows with the options header", () => {
    render(
      <EquipmentPoolSection
        poolKey="tops"
        title="Tops"
        ids={[1041002]}
        onAdd={vi.fn()}
        onRemove={vi.fn()}
      />,
    );
    expect(
      screen.getByText("1 options · player picks one"),
    ).toBeInTheDocument();
    expect(screen.getByText("Blue T-Shirt")).toBeInTheDocument();
    expect(screen.getByText("1041002")).toBeInTheDocument();
  });

  it("degrades to Unknown item when the name lookup fails, still removable", async () => {
    useItemNameMock.mockReturnValue({ data: undefined, isError: true });
    const onRemove = vi.fn();
    render(
      <EquipmentPoolSection
        poolKey="tops"
        title="Tops"
        ids={[9999999]}
        onAdd={vi.fn()}
        onRemove={onRemove}
      />,
    );
    expect(screen.getByText("Unknown item")).toBeInTheDocument();
    await userEvent.click(
      screen.getByRole("button", { name: /remove 9999999/i }),
    );
    expect(onRemove).toHaveBeenCalledWith(0);
  });

  it("combobox add wires through", async () => {
    const onAdd = vi.fn();
    render(
      <EquipmentPoolSection
        poolKey="tops"
        title="Tops"
        ids={[]}
        onAdd={onAdd}
        onRemove={vi.fn()}
      />,
    );
    await userEvent.click(screen.getByRole("button", { name: "mock-add" }));
    expect(onAdd).toHaveBeenCalledWith(1041002);
  });
});
