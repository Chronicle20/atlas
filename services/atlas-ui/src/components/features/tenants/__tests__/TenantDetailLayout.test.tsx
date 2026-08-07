import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";

vi.mock("@/components/features/config/ConfigExportButton", () => ({
  ConfigExportButton: ({ kind, id }: { kind: string; id?: string }) => (
    <button type="button" data-kind={kind} data-id={id}>
      Export
    </button>
  ),
}));

describe("TenantDetailLayout", () => {
  it("renders an Export control in the header for the routed tenant", () => {
    render(
      <MemoryRouter initialEntries={["/tenants/tnt-1/properties"]}>
        <Routes>
          <Route
            path="/tenants/:id/properties"
            element={
              <TenantDetailLayout>
                <div>child</div>
              </TenantDetailLayout>
            }
          />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.getByText("Tenant Details")).toBeInTheDocument();
    const button = screen.getByRole("button", { name: "Export" });
    expect(button).toHaveAttribute("data-kind", "tenant");
    expect(button).toHaveAttribute("data-id", "tnt-1");
  });
});
