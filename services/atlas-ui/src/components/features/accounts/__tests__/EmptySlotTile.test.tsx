// services/atlas-ui/src/components/features/accounts/__tests__/EmptySlotTile.test.tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { EmptySlotTile } from "../EmptySlotTile";

describe("EmptySlotTile", () => {
  it("renders an accessible button with the silhouette image", () => {
    render(<EmptySlotTile onClick={vi.fn()} />);
    const btn = screen.getByRole("button", { name: /add character to slot/i });
    expect(btn).toBeInTheDocument();
    expect(screen.getByText(/add character/i)).toBeInTheDocument();
    const img = btn.querySelector("img");
    expect(img).not.toBeNull();
    expect(img!.getAttribute("src")).toContain("default-character-avatar.svg");
  });

  it("invokes onClick when clicked", async () => {
    const onClick = vi.fn();
    render(<EmptySlotTile onClick={onClick} />);
    await userEvent.click(screen.getByRole("button", { name: /add character to slot/i }));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("blocks click when disabled", async () => {
    const onClick = vi.fn();
    render(<EmptySlotTile onClick={onClick} disabled />);
    await userEvent.click(screen.getByRole("button", { name: /add character to slot/i }));
    expect(onClick).not.toHaveBeenCalled();
  });
});
