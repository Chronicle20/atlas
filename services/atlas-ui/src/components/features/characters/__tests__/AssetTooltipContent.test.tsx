import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { AssetTooltipContent } from "../AssetTooltipContent";
import type { Asset } from "@/services/api/inventory.service";

const baseAsset = (overrides: Partial<Asset["attributes"]> = {}): Asset => ({
  type: "assets", id: "999",
  attributes: {
    id: 999, slot: -1, templateId: 1002000, expiration: "0001-01-01T00:00:00Z",
    createdAt: "0001-01-01T00:00:00Z", quantity: 1, ownerId: 0, flag: 0, rechargeable: 0,
    strength: 0, dexterity: 0, intelligence: 0, luck: 0, hp: 0, mp: 0,
    weaponAttack: 0, magicAttack: 0, weaponDefense: 0, magicDefense: 0, accuracy: 0, avoidability: 0,
    hands: 0, speed: 0, jump: 0, slots: 0, levelType: 0, level: 0, experience: 0, hammersApplied: 0,
    equippedSince: "0001-01-01T00:00:00Z",
    cashId: "", commodityId: 0, purchaseBy: 0, petId: 0,
    ...overrides,
  },
});

describe("AssetTooltipContent", () => {
  it("omits zero numeric fields", () => {
    render(<AssetTooltipContent asset={baseAsset({ strength: 10 })} itemName="Test Hat" />);
    expect(screen.getByText(/STR/)).toBeInTheDocument();
    expect(screen.queryByText(/^DEX$/)).not.toBeInTheDocument();
  });

  it("omits zero-date fields", () => {
    render(<AssetTooltipContent asset={baseAsset()} itemName="Test" />);
    expect(screen.queryByText(/Expires/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Equipped Since/i)).not.toBeInTheDocument();
  });

  it("renders a real expiration date", () => {
    render(<AssetTooltipContent asset={baseAsset({ expiration: "2027-01-01T00:00:00Z" })} itemName="Test" />);
    expect(screen.getByText(/Expires/i)).toBeInTheDocument();
  });

  it("formats numbers > 9999 with separators", () => {
    render(<AssetTooltipContent asset={baseAsset({ experience: 12345 })} itemName="X" />);
    expect(screen.getByText(/12,345/)).toBeInTheDocument();
  });

  it("renders the slotName override when provided", () => {
    render(<AssetTooltipContent asset={baseAsset()} itemName="X" slotName="Hat" />);
    expect(screen.getByText(/Hat/)).toBeInTheDocument();
  });

  it("includes asset id and slot in footer", () => {
    render(<AssetTooltipContent asset={baseAsset({ slot: -1 })} itemName="X" />);
    expect(screen.getByText(/Asset ID/i)).toBeInTheDocument();
    expect(screen.getByText(/-1/)).toBeInTheDocument();
  });
});
