import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import type { Asset, Compartment } from "@/services/api/inventory.service";

// Mock data hooks used transitively by InventoryCard / InventoryGrid so the
// tests don't reach the real network or IntersectionObserver-based lazy
// loader. These mocks must be declared before the component import so
// vi.mock hoisting picks them up.
vi.mock("@/lib/hooks/useItemData", () => ({
  useItemData: () => ({
    itemData: { iconUrl: "", name: "" },
    isLoading: false,
    hasError: false,
    errorMessage: null,
  }),
  useItemDataCache: () => ({
    warmCache: vi.fn().mockResolvedValue([]),
  }),
}));

vi.mock("@/lib/hooks/useIntersectionObserver", () => ({
  useLazyLoad: () => ({ shouldLoad: true, ref: { current: null } }),
}));

// AssetTooltipContent now uses useEquipmentData. The grid test doesn't
// open tooltips in jsdom, but Radix may still mount the content subtree
// during interaction; stub the hook so it never reaches React Query.
vi.mock("@/lib/hooks/api/useEquipmentData", () => ({
  useEquipmentData: () => ({ data: undefined, isLoading: false, isError: false }),
}));

import { InventoryGrid } from "../InventoryGrid";

const makeCompartment = (capacity: number): Compartment => ({
  type: "compartments",
  id: "comp-1",
  attributes: { type: 1, capacity },
  relationships: {
    assets: { links: { related: "", self: "" }, data: [] },
  },
});

const makeAsset = (overrides: Partial<Asset["attributes"]> = {}): Asset => ({
  type: "assets",
  id: "asset-1",
  attributes: {
    id: 1,
    slot: 1,
    templateId: 1002000,
    expiration: "0001-01-01T00:00:00Z",
    createdAt: "0001-01-01T00:00:00Z",
    quantity: 1,
    ownerId: 0,
    flag: 0,
    rechargeable: 0,
    strength: 0,
    dexterity: 0,
    intelligence: 0,
    luck: 0,
    hp: 0,
    mp: 0,
    weaponAttack: 0,
    magicAttack: 0,
    weaponDefense: 0,
    magicDefense: 0,
    accuracy: 0,
    avoidability: 0,
    hands: 0,
    speed: 0,
    jump: 0,
    slots: 0,
    levelType: 0,
    level: 0,
    experience: 0,
    hammersApplied: 0,
    equippedSince: "0001-01-01T00:00:00Z",
    cashId: "",
    commodityId: 0,
    purchaseBy: 0,
    petId: 0,
    ...overrides,
  },
});

const minimalProps = {
  compartment: makeCompartment(24),
  assets: [],
};

const minimalPropsWithItem = {
  compartment: makeCompartment(24),
  assets: [makeAsset()],
};

describe("InventoryGrid", () => {
  it("uses grid-cols-4 sm:grid-cols-8 lg:grid-cols-12 classes", () => {
    const { container } = render(<InventoryGrid {...minimalProps} />);
    const grid = container.querySelector(".grid");
    expect(grid?.className).toContain("grid-cols-4");
    expect(grid?.className).toContain("sm:grid-cols-8");
    expect(grid?.className).toContain("lg:grid-cols-12");
  });

  it("calls onDeleteAsset when X is clicked, without opening tooltip flicker", async () => {
    const onDelete = vi.fn();
    render(<InventoryGrid {...minimalPropsWithItem} onDeleteAsset={onDelete} />);
    const xButton = screen.getByRole("button", { name: /delete/i });
    await userEvent.click(xButton);
    expect(onDelete).toHaveBeenCalledWith("asset-1");
  });
});
