import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { TemplateDetailLayout } from "@/components/features/templates/TemplateDetailLayout";

vi.mock("@/components/features/config/ConfigExportButton", () => ({
  ConfigExportButton: ({ kind, id }: { kind: string; id?: string }) => (
    <button type="button" data-kind={kind} data-id={id}>
      Export
    </button>
  ),
}));

vi.mock("@/components/features/templates/TemplateReseedButton", () => ({
  TemplateReseedButton: ({ id }: { id?: string }) => (
    <button type="button" data-id={id}>
      Reset to shipped defaults
    </button>
  ),
}));

describe("TemplateDetailLayout", () => {
  it("renders an Export control in the header for the routed template", () => {
    render(
      <MemoryRouter initialEntries={["/templates/tpl-1/properties"]}>
        <Routes>
          <Route
            path="/templates/:id/properties"
            element={
              <TemplateDetailLayout>
                <div>child</div>
              </TemplateDetailLayout>
            }
          />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.getByText("Template Details")).toBeInTheDocument();
    const button = screen.getByRole("button", { name: "Export" });
    expect(button).toHaveAttribute("data-kind", "template");
    expect(button).toHaveAttribute("data-id", "tpl-1");
  });
});
