import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { KindBadge } from "../KindBadge";

describe("KindBadge", () => {
  it("renders a distinct badge for cash-surprise pools", () => {
    render(<KindBadge kind="cash-surprise" />);
    expect(screen.getByText("Cash Surprise")).toBeInTheDocument();
  });

  it("still renders the existing kinds unchanged", () => {
    const { rerender } = render(<KindBadge kind="gachapon" />);
    expect(screen.getByText("Gachapon")).toBeInTheDocument();
    rerender(<KindBadge kind="incubator" />);
    expect(screen.getByText("Incubator")).toBeInTheDocument();
  });
});
