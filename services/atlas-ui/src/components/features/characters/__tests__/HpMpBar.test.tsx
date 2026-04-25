import { render } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { HpMpBar } from "../HpMpBar";

describe("HpMpBar", () => {
  it("renders cur / max text", () => {
    const { getByText } = render(<HpMpBar label="HP" cur={50} max={100} colorClass="bg-red-500" />);
    expect(getByText(/50/).textContent).toMatch(/50.*100/);
  });

  it("computes width as a percentage", () => {
    const { container } = render(<HpMpBar label="HP" cur={25} max={100} colorClass="bg-red-500" />);
    const fill = container.querySelector(".bg-red-500") as HTMLElement;
    expect(fill.style.width).toBe("25%");
  });

  it("clamps to 100% when cur > max", () => {
    const { container } = render(<HpMpBar label="HP" cur={500} max={100} colorClass="bg-red-500" />);
    const fill = container.querySelector(".bg-red-500") as HTMLElement;
    expect(fill.style.width).toBe("100%");
  });

  it("renders 0% when max is 0", () => {
    const { container } = render(<HpMpBar label="MP" cur={10} max={0} colorClass="bg-blue-500" />);
    const fill = container.querySelector(".bg-blue-500") as HTMLElement;
    expect(fill.style.width).toBe("0%");
  });
});
