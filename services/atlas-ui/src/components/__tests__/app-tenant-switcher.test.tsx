import { describe, it, expect, vi, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { TenantSwitcher } from "@/components/app-tenant-switcher";
import { SidebarProvider } from "@/components/ui/sidebar";

const mockTenant = {
  id: "11111111-1111-1111-1111-111111111111",
  attributes: {
    name: "Test Tenant",
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
  },
};

const setActiveTenant = vi.fn();

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    tenants: [mockTenant],
    activeTenant: mockTenant,
    setActiveTenant,
    refreshTenants: vi.fn(),
  }),
}));

vi.mock("@/components/features/tenants/CreateTenantDialog", () => ({
  CreateTenantDialog: () => null,
}));

beforeAll(() => {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
});

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <SidebarProvider>
        <TenantSwitcher />
      </SidebarProvider>
    </MemoryRouter>,
  );
}

describe("TenantSwitcher", () => {
  it.each(["/templates", "/tenants/9f8e/writers", "/services", "/baselines"])(
    "renders the inert Deployment-wide state on %s",
    (path) => {
      renderAt(path);
      expect(screen.getByText("Deployment-wide")).toBeInTheDocument();
      expect(screen.getByText("tenant selection inactive")).toBeInTheDocument();
      // No dropdown affordance: the picker trigger must not exist.
      expect(screen.queryByRole("button")).not.toBeInTheDocument();
      expect(screen.queryByText("Test Tenant")).not.toBeInTheDocument();
    },
  );

  it.each(["/", "/accounts", "/setup", "/characters/42"])(
    "renders the interactive picker on %s",
    (path) => {
      renderAt(path);
      expect(screen.getByText("Test Tenant")).toBeInTheDocument();
      expect(screen.queryByText("Deployment-wide")).not.toBeInTheDocument();
    },
  );

  it("never writes tenant state from the inert branch", () => {
    renderAt("/templates");
    expect(setActiveTenant).not.toHaveBeenCalled();
  });
});
