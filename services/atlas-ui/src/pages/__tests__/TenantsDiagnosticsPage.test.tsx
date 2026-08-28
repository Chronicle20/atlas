import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import type { TenantConfig } from "@/services/api/tenants.service";

const sampleTenant: TenantConfig = {
  id: "tnt-1",
  attributes: {
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
    usesPin: false,
    characters: { templates: [], presets: [] },
    npcs: [],
    socket: { handlers: [], writers: [] },
    worlds: [],
  },
} as unknown as TenantConfig;

const { tenantHolder, mutateMock } = vi.hoisted(() => ({
  tenantHolder: {
    data: undefined as TenantConfig | undefined,
    isLoading: false,
    error: null as Error | null,
  },
  mutateMock: vi.fn(),
}));

vi.mock("@/lib/hooks/api/useTenants", () => ({
  useTenantConfiguration: () => tenantHolder,
  useUpdateTenantConfiguration: () => ({
    mutate: mutateMock,
    isPending: false,
  }),
}));

vi.mock("@/components/features/tenants/TenantDetailLayout", () => ({
  TenantDetailLayout: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

import { TenantsDiagnosticsPage } from "@/pages/TenantsDiagnosticsPage";

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/tenants/tnt-1/diagnostics"]}>
      <Routes>
        <Route
          path="/tenants/:id/diagnostics"
          element={<TenantsDiagnosticsPage />}
        />
      </Routes>
    </MemoryRouter>,
  );
}

describe("TenantsDiagnosticsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    tenantHolder.data = sampleTenant;
    tenantHolder.isLoading = false;
    tenantHolder.error = null;
  });

  it("renders the switch off for a tenant with no diagnostics object", () => {
    renderPage();
    expect(screen.getByRole("switch")).not.toBeChecked();
  });

  it("renders the switch on for a tenant with tracePackets true", () => {
    tenantHolder.data = {
      ...sampleTenant,
      attributes: {
        ...sampleTenant.attributes,
        diagnostics: { tracePackets: true },
      },
    } as unknown as TenantConfig;
    renderPage();
    expect(screen.getByRole("switch")).toBeChecked();
  });

  it("renders the credential warning", () => {
    renderPage();
    const alert = screen.getByRole("alert");
    expect(alert.textContent).toContain("account passwords");
    expect(alert.textContent).toContain("very large volumes of log output");
    expect(alert.textContent).toMatch(/LOG_LEVEL=Debug or Trace/);
  });

  it("submits only the diagnostics object", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("switch"));
    await user.click(screen.getByRole("button", { name: /save/i }));

    expect(mutateMock).toHaveBeenCalledTimes(1);
    const [vars] = mutateMock.mock.calls[0]!;
    expect(vars.updates).toEqual({ diagnostics: { tracePackets: true } });
  });
});
