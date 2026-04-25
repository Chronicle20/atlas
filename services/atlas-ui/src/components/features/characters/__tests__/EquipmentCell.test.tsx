import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { EquipmentCell } from "../EquipmentCell";
import type { Asset } from "@/services/api/inventory.service";

const fakeTenant = { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } as never;
const baseAsset = { type: "assets", id: "1", attributes: { id: 1, slot: -1, templateId: 1002000, quantity: 1, expiration: "0001-01-01T00:00:00Z" } } as Asset;

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
