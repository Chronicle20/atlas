// services/atlas-ui/src/components/common/__tests__/CopyableIdHeader.test.tsx
import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { CopyableIdHeader } from "../CopyableIdHeader";

describe("CopyableIdHeader", () => {
  it("renders the title as a focusable h2 with focus-ring class", () => {
    render(<CopyableIdHeader title="Aran4th" id="42" />);
    const heading = screen.getByText("Aran4th");
    expect(heading.tagName).toBe("H2");
    expect(heading).toHaveAttribute("tabIndex", "0");
    expect(heading.className).toMatch(/focus-visible:ring/);
  });

  it("renders supplied actions", () => {
    render(
      <CopyableIdHeader
        title="Aran4th"
        id="42"
        actions={<button>Promote</button>}
      />
    );
    expect(screen.getByRole("button", { name: /promote/i })).toBeInTheDocument();
  });

  it("renders without actions when none provided", () => {
    render(<CopyableIdHeader title="Aran4th" id="42" />);
    expect(screen.queryByRole("button")).toBeNull();
  });
});
