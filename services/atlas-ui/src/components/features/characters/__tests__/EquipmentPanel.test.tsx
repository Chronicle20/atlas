import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { EquipmentPanel } from "../EquipmentPanel";
import type { Asset } from "@/services/api/inventory.service";

const fakeTenant = { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } as never;

const equipped = (slot: number, templateId: number): Asset => ({
  type: "assets", id: String(templateId),
  attributes: { id: templateId, slot, templateId, quantity: 1 } as Asset["attributes"],
});

describe("EquipmentPanel", () => {
  it("renders all 21 slots — empty placeholders when nothing equipped", () => {
    const { container } = render(<EquipmentPanel equipped={[]} tenant={fakeTenant} />);
    expect(container.querySelectorAll(".aspect-square")).toHaveLength(21);
  });

  it("renders an icon for each filled slot, placeholder for the rest", () => {
    const { container } = render(<EquipmentPanel equipped={[equipped(-1, 1002000)]} tenant={fakeTenant} />);
    expect(container.querySelectorAll("img")).toHaveLength(1);
    expect(screen.getByText("Weapon")).toBeInTheDocument(); // empty Weapon slot label
  });

  it("places Hat at row 1 col 3", () => {
    const { container } = render(<EquipmentPanel equipped={[equipped(-1, 1002000)]} tenant={fakeTenant} />);
    const cells = container.querySelectorAll(".aspect-square");
    const hatCell = Array.from(cells).find((c) => {
      const wrapper = c.parentElement as HTMLElement | null;
      return wrapper?.style.gridRow === "1" && wrapper?.style.gridColumn === "3";
    }) as HTMLElement;
    expect(hatCell?.querySelector("img")).toBeTruthy();
  });
});
