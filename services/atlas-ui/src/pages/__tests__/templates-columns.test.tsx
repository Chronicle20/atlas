import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { getColumns } from "@/pages/templates-columns";
import type { Template } from "@/types/models/template";

function driftCell(attributes: Partial<Template["attributes"]>) {
  const columns = getColumns({});
  const column = columns.find((c) => c.id === "seedDrift");
  if (!column?.cell || typeof column.cell !== "function") {
    throw new Error("seedDrift column is missing or has no cell renderer");
  }
  const row = {
    original: { id: "abc-123", attributes } as Template,
  };
  return column.cell({ row } as never);
}

describe("templates-columns seedDrift", () => {
  it("renders the badge when the template has drifted", () => {
    render(<MemoryRouter>{driftCell({ seedDrift: true })}</MemoryRouter>);
    expect(screen.getByText("Differs from image")).toBeInTheDocument();
  });

  it("renders nothing when the template has not drifted", () => {
    const { container } = render(
      <MemoryRouter>{driftCell({ seedDrift: false })}</MemoryRouter>,
    );
    expect(container).not.toHaveTextContent("Differs from image");
  });

  it("renders nothing when no seed file ships", () => {
    const { container } = render(
      <MemoryRouter>
        {driftCell({ shippedRevision: "", seedDrift: false })}
      </MemoryRouter>,
    );
    expect(container).not.toHaveTextContent("Differs from image");
  });

  it("renders nothing when the attribute is absent", () => {
    const { container } = render(<MemoryRouter>{driftCell({})}</MemoryRouter>);
    expect(container).not.toHaveTextContent("Differs from image");
  });
});
