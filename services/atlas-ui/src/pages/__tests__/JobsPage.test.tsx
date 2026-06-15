import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import type { Tenant } from "@/services/api/tenants.service";

const useTenantMock = vi.fn();
vi.mock("@/context/tenant-context", () => ({ useTenant: () => useTenantMock() }));

import { JobsPage } from "@/pages/JobsPage";

const tenant = (major: number) =>
  ({ id: "t1", attributes: { region: "GMS", majorVersion: major, minorVersion: 1 } } as unknown as Tenant);

function renderPage() {
  return render(
    <MemoryRouter>
      <JobsPage />
    </MemoryRouter>,
  );
}

describe("JobsPage", () => {
  beforeEach(() => vi.clearAllMocks());

  it("shows a select-a-tenant empty state when no tenant is active", () => {
    useTenantMock.mockReturnValue({ activeTenant: null });
    renderPage();
    expect(screen.getByText(/select a tenant/i)).toBeInTheDocument();
    expect(screen.queryByText("Beginner")).not.toBeInTheDocument();
  });

  it("renders Cygnus + Aran roots on a v83 tenant, Evan hidden", () => {
    useTenantMock.mockReturnValue({ activeTenant: tenant(83) });
    renderPage();
    expect(screen.getByText("Beginner")).toBeInTheDocument();
    expect(screen.getByText("Warrior")).toBeInTheDocument();
    expect(screen.getByText("Noblesse")).toBeInTheDocument(); // Cygnus root visible on v83
    expect(screen.getByText("Legend")).toBeInTheDocument();   // Aran root visible on v83
    expect(screen.queryByText("Evan")).not.toBeInTheDocument();
  });

  it("reveals the Evan root on a v84 tenant", () => {
    useTenantMock.mockReturnValue({ activeTenant: tenant(84) });
    renderPage();
    expect(screen.getByText("Evan")).toBeInTheDocument();
  });

  it("gives branch nodes a toggle affordance and links job names to detail pages", () => {
    useTenantMock.mockReturnValue({ activeTenant: tenant(83) });
    renderPage();
    expect(screen.getByLabelText(/toggle beginner/i)).toBeInTheDocument();
    expect(screen.getByText("Warrior").closest("a")).toHaveAttribute("href", "/jobs/100");
  });

  it("scrolls in-page via a local overflow container (no app-shell change)", () => {
    useTenantMock.mockReturnValue({ activeTenant: tenant(83) });
    const { container } = renderPage();
    expect(container.firstChild).toHaveClass("overflow-y-auto");
  });
});
