import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { TemplateDetailLayout } from "@/components/features/templates/TemplateDetailLayout";
import { useTemplate } from "@/lib/hooks/api/useTemplates";
import { MAPLE_LIFE_HANDLER } from "@/components/features/characters/maple-life/mapleLifeSupport";

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

vi.mock("@/lib/hooks/api/useTemplates", () => ({
  useTemplate: vi.fn(),
}));

function renderAt(id: string, path = "properties") {
  render(
    <MemoryRouter initialEntries={[`/templates/${id}/${path}`]}>
      <Routes>
        <Route
          path={`/templates/:id/${path}`}
          element={
            <TemplateDetailLayout>
              <div>child</div>
            </TemplateDetailLayout>
          }
        />
      </Routes>
    </MemoryRouter>,
  );
}

describe("TemplateDetailLayout", () => {
  it("renders an Export control in the header for the routed template", () => {
    vi.mocked(useTemplate).mockReturnValue({ data: undefined } as never);
    renderAt("tpl-1");

    expect(screen.getByText("Template Details")).toBeInTheDocument();
    const button = screen.getByRole("button", { name: "Export" });
    expect(button).toHaveAttribute("data-kind", "template");
    expect(button).toHaveAttribute("data-id", "tpl-1");
  });

  it("hides the Maple Life nav item when the handler is absent", () => {
    vi.mocked(useTemplate).mockReturnValue({
      data: {
        id: "tpl-1",
        attributes: { socket: { handlers: [], writers: [] } },
      },
    } as never);
    renderAt("tpl-1");

    expect(
      screen.queryByRole("link", { name: "Maple Life" }),
    ).not.toBeInTheDocument();
  });

  it("shows the Maple Life nav item when the handler is present", () => {
    vi.mocked(useTemplate).mockReturnValue({
      data: {
        id: "tpl-1",
        attributes: {
          socket: {
            handlers: [
              {
                opCode: "0x12D",
                validator: "V",
                handler: MAPLE_LIFE_HANDLER,
              },
            ],
            writers: [],
          },
        },
      },
    } as never);
    renderAt("tpl-1");

    expect(
      screen.getByRole("link", { name: "Maple Life" }),
    ).toBeInTheDocument();
  });
});
