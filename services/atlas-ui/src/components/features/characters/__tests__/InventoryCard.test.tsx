import { render } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import type { Asset } from "@/services/api/inventory.service";

// Mock data hooks used by InventoryCard so the tests don't reach the real
// network or IntersectionObserver-based lazy loader. These mocks must be
// declared before the component import so vi.mock hoisting picks them up.
vi.mock("@/lib/hooks/useItemData", () => ({
  useItemData: () => ({
    itemData: { iconUrl: "https://example.test/icon.png", name: "Test Item" },
    isLoading: false,
    hasError: false,
    errorMessage: null,
  }),
}));

vi.mock("@/lib/hooks/useIntersectionObserver", () => ({
  useLazyLoad: () => ({ shouldLoad: true, ref: { current: null } }),
}));

import { InventoryCard } from "../InventoryCard";

function asset(over: Partial<Asset["attributes"]>): Asset {
  return {
    type: "assets",
    id: "1",
    attributes: {
      flag: 0, owner: "", expiration: "0001-01-01T00:00:00Z", templateId: 1040000, id: 1, slot: 1,
      createdAt: "0001-01-01T00:00:00Z", quantity: 1, ownerId: 0, rechargeable: 0, strength: 0,
      dexterity: 0, intelligence: 0, luck: 0, hp: 0, mp: 0, weaponAttack: 0,
      magicAttack: 0, weaponDefense: 0, magicDefense: 0, accuracy: 0,
      avoidability: 0, hands: 0, speed: 0, jump: 0, slots: 0, levelType: 0,
      level: 0, experience: 0, hammersApplied: 0, equippedSince: "0001-01-01T00:00:00Z",
      cashId: "", commodityId: 0, purchaseBy: 0, petId: 0, ...over,
    },
  };
}

describe("InventoryCard indicators", () => {
  it("renders a lock icon when sealed", () => {
    const { container } = render(<InventoryCard asset={asset({ flag: 0x01 })} />);
    expect(container.querySelector('[data-testid="seal-icon"]')).toBeTruthy();
  });

  it("renders a tag icon when tagged", () => {
    const { container } = render(<InventoryCard asset={asset({ owner: "Chronicle" })} />);
    expect(container.querySelector('[data-testid="tag-icon"]')).toBeTruthy();
  });

  it("renders neither when plain", () => {
    const { container } = render(<InventoryCard asset={asset({})} />);
    expect(container.querySelector('[data-testid="seal-icon"]')).toBeFalsy();
    expect(container.querySelector('[data-testid="tag-icon"]')).toBeFalsy();
  });
});
