import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { SurfaceKindBadge } from "@/components/features/maps/SurfaceKindBadge";

describe("SurfaceKindBadge", () => {
  it("renders the word Definition", () => {
    render(<SurfaceKindBadge kind="definition" />);
    expect(screen.getByText("Definition")).toBeInTheDocument();
  });

  it("renders the word Runtime", () => {
    render(<SurfaceKindBadge kind="runtime" />);
    expect(screen.getByText("Runtime")).toBeInTheDocument();
  });

  it("never relies on colour alone", () => {
    const { rerender } = render(<SurfaceKindBadge kind="definition" />);
    expect(screen.getByText("Definition").textContent).toBeTruthy();

    rerender(<SurfaceKindBadge kind="runtime" />);
    expect(screen.getByText("Runtime").textContent).toBeTruthy();
  });
});
