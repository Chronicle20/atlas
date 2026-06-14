import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import type { Tenant } from "@/services/api/tenants.service";

const useTenantMock = vi.fn();
vi.mock("@/context/tenant-context", () => ({ useTenant: () => useTenantMock() }));

import { JobsPage } from "@/pages/JobsPage";

const v83 = { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } as unknown as Tenant;

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
    expect(screen.queryByText("Adventurer")).not.toBeInTheDocument();
  });

  it("renders Adventurer branches but not Cygnus/Legend on a v83 tenant", () => {
    useTenantMock.mockReturnValue({ activeTenant: v83 });
    renderPage();
    expect(screen.getByText("Adventurer")).toBeInTheDocument();
    expect(screen.getByText("Warrior")).toBeInTheDocument();
    expect(screen.queryByText("Cygnus")).not.toBeInTheDocument();
    expect(screen.queryByText("Dawn Warrior")).not.toBeInTheDocument();
  });
});
