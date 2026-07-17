import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { EquipmentCell } from "../EquipmentCell";
import type { Asset } from "@/services/api/inventory.service";

const fakeTenant = { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } as never;
const baseAsset = { type: "assets", id: "1", attributes: { id: 1, slot: -1, templateId: 1002000, quantity: 1, expiration: "0001-01-01T00:00:00Z", flag: 0, owner: "" } } as Asset;

function asset(over: Partial<Asset["attributes"]>): Asset {
  return {
    type: "assets",
    id: "1",
    attributes: {
      flag: 0, owner: "", expiration: "", templateId: 1040000, id: 1, slot: -5,
      createdAt: "", quantity: 1, ownerId: 0, rechargeable: 0, strength: 0,
      dexterity: 0, intelligence: 0, luck: 0, hp: 0, mp: 0, weaponAttack: 0,
      magicAttack: 0, weaponDefense: 0, magicDefense: 0, accuracy: 0,
      avoidability: 0, hands: 0, speed: 0, jump: 0, slots: 0, levelType: 0,
      level: 0, experience: 0, hammersApplied: 0, equippedSince: "",
      cashId: "", commodityId: 0, purchaseBy: 0, petId: 0, ...over,
    },
  };
}

describe("EquipmentCell", () => {
  it("renders the empty placeholder when asset is undefined", () => {
    render(<EquipmentCell slotId={-1} slotName="Hat" tenant={fakeTenant} />);
    expect(screen.getByText("Hat")).toBeInTheDocument();
  });

  it("exposes the slot name on the empty cell via aria-label", () => {
    // The Radix tooltip body ("Hat (empty)") renders into a portal on
    // hover/focus and jsdom doesn't drive that reliably; the same string
    // is mirrored onto the trigger via aria-label so screen readers and
    // tests can reach it without simulating pointer events.
    render(<EquipmentCell slotId={-1} slotName="Hat" tenant={fakeTenant} />);
    expect(screen.getByLabelText("Hat (empty)")).toBeInTheDocument();
  });

  it("renders the asset icon when filled", () => {
    const { container } = render(<EquipmentCell slotId={-1} slotName="Hat" asset={baseAsset} tenant={fakeTenant} />);
    const img = container.querySelector("img") as HTMLImageElement;
    expect(img.src).toContain("/item/1002000/icon.png");
  });
});

describe("EquipmentCell indicators", () => {
  it("renders a lock icon when sealed", () => {
    const { container } = render(
      <EquipmentCell slotId={-5} slotName="Top" asset={asset({ flag: 0x01 })} tenant={fakeTenant} />,
    );
    expect(container.querySelector('[data-testid="seal-icon"]')).toBeTruthy();
  });

  it("renders a tag icon when tagged", () => {
    const { container } = render(
      <EquipmentCell slotId={-5} slotName="Top" asset={asset({ owner: "Chronicle" })} tenant={fakeTenant} />,
    );
    expect(container.querySelector('[data-testid="tag-icon"]')).toBeTruthy();
  });

  it("renders neither when plain", () => {
    const { container } = render(
      <EquipmentCell slotId={-5} slotName="Top" asset={asset({})} tenant={fakeTenant} />,
    );
    expect(container.querySelector('[data-testid="seal-icon"]')).toBeFalsy();
    expect(container.querySelector('[data-testid="tag-icon"]')).toBeFalsy();
  });
});
