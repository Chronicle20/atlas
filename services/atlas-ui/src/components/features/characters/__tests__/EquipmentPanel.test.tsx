import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { EquipmentPanel } from "../EquipmentPanel";
import type { Asset } from "@/services/api/inventory.service";

const fakeTenant = { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } as never;

const equipped = (slot: number, templateId: number): Asset => ({
  type: "assets", id: String(templateId),
  attributes: { id: templateId, slot, templateId, quantity: 1 } as Asset["attributes"],
});

// SLOT_LAYOUT pinned by the user from the v83 in-game window. These tests
// guard the contract (cell count + a couple of canonical positions) so a
// future regression on coordinates fails loudly.

describe("EquipmentPanel", () => {
  it("renders one cell per SLOT_LAYOUT entry — empty placeholders when nothing equipped", () => {
    const { container } = render(<EquipmentPanel equipped={[]} tenant={fakeTenant} />);
    // 18 slots in the current SLOT_LAYOUT (Mount / Saddle / Pet Equip / Pendant 2 deferred).
    expect(container.querySelectorAll(".aspect-square")).toHaveLength(18);
  });

  it("renders an icon for each filled slot, placeholder for the rest", () => {
    const { container } = render(<EquipmentPanel equipped={[equipped(-1, 1002000)]} tenant={fakeTenant} />);
    expect(container.querySelectorAll("img")).toHaveLength(1);
    // An adjacent slot — Top — is still rendering its empty-slot label.
    expect(screen.getByText("Top")).toBeInTheDocument();
  });

  it("places Hat at row 1 col 2 (per the in-game layout)", () => {
    const { container } = render(<EquipmentPanel equipped={[equipped(-1, 1002000)]} tenant={fakeTenant} />);
    const cells = container.querySelectorAll(".aspect-square");
    const hatCell = Array.from(cells).find((c) => {
      const wrapper = c.parentElement as HTMLElement | null;
      return wrapper?.style.gridRow === "1" && wrapper?.style.gridColumn === "2";
    }) as HTMLElement;
    expect(hatCell?.querySelector("img")).toBeTruthy();
  });

  it("places Top at row 4 col 2 (body row, just right of Cape)", () => {
    const { container } = render(<EquipmentPanel equipped={[equipped(-5, 1040002)]} tenant={fakeTenant} />);
    const cells = container.querySelectorAll(".aspect-square");
    const topCell = Array.from(cells).find((c) => {
      const wrapper = c.parentElement as HTMLElement | null;
      return wrapper?.style.gridRow === "4" && wrapper?.style.gridColumn === "2";
    }) as HTMLElement;
    expect(topCell?.querySelector("img")).toBeTruthy();
  });

  it("places Shoes at row 6 col 3 (bottom-center)", () => {
    const { container } = render(<EquipmentPanel equipped={[equipped(-7, 1072000)]} tenant={fakeTenant} />);
    const cells = container.querySelectorAll(".aspect-square");
    const shoesCell = Array.from(cells).find((c) => {
      const wrapper = c.parentElement as HTMLElement | null;
      return wrapper?.style.gridRow === "6" && wrapper?.style.gridColumn === "3";
    }) as HTMLElement;
    expect(shoesCell?.querySelector("img")).toBeTruthy();
  });
});
