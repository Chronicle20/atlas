import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import { MAPLE_LIFE_HANDLER } from "@/components/features/characters/maple-life/mapleLifeSupport";

vi.mock("@/components/features/config/ConfigExportButton", () => ({
  ConfigExportButton: ({ kind, id }: { kind: string; id?: string }) => (
    <button type="button" data-kind={kind} data-id={id}>
      Export
    </button>
  ),
}));

vi.mock("@/lib/hooks/api/useTenants", () => ({
  useTenantConfiguration: vi.fn(),
}));

function renderAt(id: string, path = "properties") {
  render(
    <MemoryRouter initialEntries={[`/tenants/${id}/${path}`]}>
      <Routes>
        <Route
          path={`/tenants/:id/${path}`}
          element={
            <TenantDetailLayout>
              <div>child</div>
            </TenantDetailLayout>
          }
        />
      </Routes>
    </MemoryRouter>,
  );
}

describe("TenantDetailLayout", () => {
  it("renders an Export control in the header for the routed tenant", () => {
    vi.mocked(useTenantConfiguration).mockReturnValue({
      data: undefined,
    } as never);
    renderAt("tnt-1");

    expect(screen.getByText("Tenant Details")).toBeInTheDocument();
    const button = screen.getByRole("button", { name: "Export" });
    expect(button).toHaveAttribute("data-kind", "tenant");
    expect(button).toHaveAttribute("data-id", "tnt-1");
  });

  it("hides the Maple Life nav item when the handler is absent", () => {
    vi.mocked(useTenantConfiguration).mockReturnValue({
      data: {
        id: "tnt-1",
        attributes: { socket: { handlers: [], writers: [] } },
      },
    } as never);
    renderAt("tnt-1");

    expect(
      screen.queryByRole("link", { name: "Maple Life" }),
    ).not.toBeInTheDocument();
  });

  it("shows the Maple Life nav item when the handler is present", () => {
    vi.mocked(useTenantConfiguration).mockReturnValue({
      data: {
        id: "tnt-1",
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
    renderAt("tnt-1");

    expect(
      screen.getByRole("link", { name: "Maple Life" }),
    ).toBeInTheDocument();
  });

  it("shows the Diagnostics nav item", () => {
    vi.mocked(useTenantConfiguration).mockReturnValue({
      data: {
        id: "tnt-1",
        attributes: { socket: { handlers: [], writers: [] } },
      },
    } as never);
    renderAt("tnt-1");

    const link = screen.getByRole("link", { name: "Diagnostics" });
    expect(link).toHaveAttribute("href", "/tenants/tnt-1/diagnostics");
  });
});
