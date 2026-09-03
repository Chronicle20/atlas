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

vi.mock("@/components/features/tenants/TenantResetButton", () => ({
  TenantResetButton: ({ id }: { id?: string }) => (
    <button type="button" data-id={id}>
      Reset to template
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

  it("names the diverging sections in the header", () => {
    vi.mocked(useTenantConfiguration).mockReturnValue({
      data: {
        id: "tnt-1",
        attributes: {
          socket: { handlers: [], writers: [] },
          templateDrift: true,
          sectionDrift: {
            properties: false,
            socket: true,
            characters: true,
            npcs: false,
            cashShop: false,
            mapleLife: false,
          },
        },
      },
    } as never);
    renderAt("tnt-1");

    const badgeText = screen.getByText(/Differs from template:/);
    expect(badgeText).toHaveTextContent("socket");
    expect(badgeText).toHaveTextContent("characters");
    expect(badgeText).not.toHaveTextContent("npcs");
  });

  it("places the drift badge in the left header block, beneath the tenant id", () => {
    vi.mocked(useTenantConfiguration).mockReturnValue({
      data: {
        id: "tnt-1",
        attributes: {
          socket: { handlers: [], writers: [] },
          templateDrift: true,
          sectionDrift: {
            properties: false,
            socket: true,
            characters: false,
            npcs: false,
            cashShop: false,
            mapleLife: false,
          },
        },
      },
    } as never);
    renderAt("tnt-1");

    const badgeText = screen.getByText(/Differs from template:/);
    const headerBlock = screen.getByText("Tenant Details").parentElement;
    expect(headerBlock).toContainElement(badgeText);
    expect(headerBlock).not.toContainElement(
      screen.getByRole("button", { name: "Export" }),
    );
    expect(headerBlock).not.toContainElement(
      screen.getByRole("button", { name: /reset to template/i }),
    );
  });

  it("renders no drift summary when nothing has drifted", () => {
    vi.mocked(useTenantConfiguration).mockReturnValue({
      data: {
        id: "tnt-1",
        attributes: {
          socket: { handlers: [], writers: [] },
          templateDrift: false,
          sectionDrift: {
            properties: false,
            socket: false,
            characters: false,
            npcs: false,
            cashShop: false,
            mapleLife: false,
          },
        },
      },
    } as never);
    renderAt("tnt-1");

    expect(
      screen.queryByText(/Differs from template:/),
    ).not.toBeInTheDocument();
  });

  it("renders no drift summary when templateDrift is absent", () => {
    vi.mocked(useTenantConfiguration).mockReturnValue({
      data: {
        id: "tnt-1",
        attributes: {},
      },
    } as never);
    renderAt("tnt-1");

    expect(
      screen.queryByText(/Differs from template:/),
    ).not.toBeInTheDocument();
  });

  it("mounts the whole-document reset button", () => {
    vi.mocked(useTenantConfiguration).mockReturnValue({
      data: undefined,
    } as never);
    renderAt("tnt-1");

    expect(
      screen.getByRole("button", { name: /reset to template/i }),
    ).toBeInTheDocument();
  });
});
