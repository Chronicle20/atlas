import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach, type Mock } from "vitest";
import { MemoryRouter } from "react-router-dom";
import type { Tenant } from "@/types/models/tenant";

const refreshTenantsMock = vi.fn();
const useTenantMock = vi.fn();
const updateTenantMock = vi.fn();
const deleteTenantMock = vi.fn();
const toastSuccess = vi.fn();
const toastError = vi.fn();

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => useTenantMock(),
}));

vi.mock("@/services/api", () => ({
  tenantsService: {
    updateTenant: (...args: unknown[]) => updateTenantMock(...args),
    deleteTenant: (...args: unknown[]) => deleteTenantMock(...args),
  },
}));

vi.mock("sonner", () => ({
  toast: {
    success: (...args: unknown[]) => toastSuccess(...args),
    error: (...args: unknown[]) => toastError(...args),
  },
}));

vi.mock("@/lib/utils/toast", () => ({ success: vi.fn(), error: vi.fn() }));

import { TenantsPage } from "@/pages/TenantsPage";
import * as gridToast from "@/lib/utils/toast";

function makeTenant(id: string, name: string): Tenant {
  return {
    id,
    attributes: {
      name,
      region: "GMS",
      majorVersion: 83,
      minorVersion: 1,
    },
  } as unknown as Tenant;
}

const tenantA = makeTenant("aaa", "Acme");
const tenantB = makeTenant("bbb", "Beta");

function defaultUseTenantValue() {
  return {
    tenants: [tenantA, tenantB],
    loading: false,
    tenantsUpdatedAt: 0,
    refreshTenants: refreshTenantsMock,
    activeTenant: tenantA,
    setActiveTenant: vi.fn(),
    refreshAndSelectTenant: vi.fn(),
    fetchTenantConfiguration: vi.fn(),
  };
}

async function openRenameDialogFor(tenantId: string) {
  const user = userEvent.setup();
  const row = screen
    .getByText(tenantId === "aaa" ? "Acme" : "Beta")
    .closest("tr");
  if (!row) throw new Error("row not found");

  const menuButton = within(row as HTMLElement).getByRole("button", {
    name: /open menu/i,
  });
  await user.click(menuButton);
  await user.click(await screen.findByText("Rename"));
  return user;
}

describe("TenantsPage rename flow", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    refreshTenantsMock.mockResolvedValue({ ok: true });
    useTenantMock.mockReturnValue(defaultUseTenantValue());
  });

  function renderPage() {
    return render(
      <MemoryRouter>
        <TenantsPage />
      </MemoryRouter>,
    );
  }

  it("opens dialog prefilled with the tenant's current name", async () => {
    renderPage();
    await openRenameDialogFor("aaa");

    const dialog = await screen.findByRole("dialog");
    const input = within(dialog).getByLabelText("Name") as HTMLInputElement;
    expect(input.value).toBe("Acme");
  });

  it("submit disabled when trimmed input equals current name", async () => {
    renderPage();
    const user = await openRenameDialogFor("aaa");

    const dialog = await screen.findByRole("dialog");
    const save = within(dialog).getByRole("button", { name: /^save$/i });
    expect(save).toBeDisabled();

    const input = within(dialog).getByLabelText("Name");
    await user.clear(input);
    await user.type(input, "   Acme   ");
    expect(save).toBeDisabled();
  });

  it("rejects empty / whitespace-only name with inline error and does not call network", async () => {
    renderPage();
    const user = await openRenameDialogFor("aaa");

    const dialog = await screen.findByRole("dialog");
    const input = within(dialog).getByLabelText("Name");
    await user.clear(input);
    await user.type(input, "   ");

    const save = within(dialog).getByRole("button", { name: /^save$/i });
    expect(save).toBeDisabled();
    await waitFor(() => {
      expect(
        within(dialog).getByText(/tenant name is required/i),
      ).toBeInTheDocument();
    });
    expect(updateTenantMock).not.toHaveBeenCalled();
  });

  it("rejects name longer than 100 chars with inline error", async () => {
    renderPage();
    const user = await openRenameDialogFor("aaa");

    const dialog = await screen.findByRole("dialog");
    const input = within(dialog).getByLabelText("Name");
    await user.clear(input);
    await user.type(input, "x".repeat(101));

    await waitFor(() => {
      expect(
        within(dialog).getByText(/100 characters or less/i),
      ).toBeInTheDocument();
    });
    expect(updateTenantMock).not.toHaveBeenCalled();
  });

  it("submits valid new name, closes dialog, shows success toast, calls refreshTenants", async () => {
    updateTenantMock.mockResolvedValueOnce({
      ...tenantA,
      attributes: { ...tenantA.attributes, name: "Acme Renamed" },
    });

    renderPage();
    const user = await openRenameDialogFor("aaa");

    const dialog = await screen.findByRole("dialog");
    const input = within(dialog).getByLabelText("Name");
    await user.clear(input);
    await user.type(input, "Acme Renamed");

    const save = within(dialog).getByRole("button", { name: /^save$/i });
    await waitFor(() => expect(save).not.toBeDisabled());
    await user.click(save);

    await waitFor(() => {
      expect(updateTenantMock).toHaveBeenCalledWith(tenantA, {
        name: "Acme Renamed",
      });
    });
    expect(refreshTenantsMock).toHaveBeenCalled();
    await waitFor(() => {
      expect(toastSuccess).toHaveBeenCalledWith("Tenant renamed");
    });
    await waitFor(() => {
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });
  });

  it("keeps dialog open, shows error toast, logs to console.error on PATCH failure", async () => {
    const consoleErrorSpy = vi
      .spyOn(console, "error")
      .mockImplementation(() => {});
    (updateTenantMock as Mock).mockRejectedValueOnce(new Error("boom"));

    renderPage();
    const user = await openRenameDialogFor("aaa");

    const dialog = await screen.findByRole("dialog");
    const input = within(dialog).getByLabelText("Name");
    await user.clear(input);
    await user.type(input, "Something New");

    const save = within(dialog).getByRole("button", { name: /^save$/i });
    await waitFor(() => expect(save).not.toBeDisabled());
    await user.click(save);

    await waitFor(() => {
      expect(toastError).toHaveBeenCalledWith("Failed to rename tenant");
    });
    expect(consoleErrorSpy).toHaveBeenCalled();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    consoleErrorSpy.mockRestore();
  });
});

describe("TenantsPage empty-state refresh", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    refreshTenantsMock.mockResolvedValue({ ok: true });
  });

  function renderPage() {
    return render(
      <MemoryRouter>
        <TenantsPage />
      </MemoryRouter>,
    );
  }

  it("offers a refresh control when no tenants exist", () => {
    useTenantMock.mockReturnValue({ ...defaultUseTenantValue(), tenants: [] });
    renderPage();

    expect(screen.getByText("No tenants found")).toBeInTheDocument();
    expect(screen.getByTestId("empty-state-refresh")).toBeInTheDocument();
  });

  it("clicking refresh calls refreshTenants and toasts success", async () => {
    useTenantMock.mockReturnValue({ ...defaultUseTenantValue(), tenants: [] });
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByTestId("empty-state-refresh"));

    await waitFor(() => {
      expect(refreshTenantsMock).toHaveBeenCalledTimes(1);
    });
    await waitFor(() => {
      expect(gridToast.success).toHaveBeenCalledWith("Data refreshed");
    });
    expect(gridToast.error).not.toHaveBeenCalled();
  });

  it("toasts an error when the refresh reports failure", async () => {
    useTenantMock.mockReturnValue({ ...defaultUseTenantValue(), tenants: [] });
    const failure = new Error("network down");
    refreshTenantsMock.mockResolvedValue({ ok: false, error: failure });
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByTestId("empty-state-refresh"));

    await waitFor(() => {
      expect(gridToast.error).toHaveBeenCalledTimes(1);
    });
    expect(gridToast.error).toHaveBeenCalledWith(failure, {
      context: { action: "refresh" },
    });
    expect(gridToast.success).not.toHaveBeenCalled();
  });

  it("treats an undefined refresh result as success", async () => {
    useTenantMock.mockReturnValue({ ...defaultUseTenantValue(), tenants: [] });
    refreshTenantsMock.mockResolvedValue(undefined);
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByTestId("empty-state-refresh"));

    await waitFor(() => {
      expect(gridToast.success).toHaveBeenCalledTimes(1);
    });
  });

  it("shows the grid, not the skeleton, while refreshing", async () => {
    let resolveRefresh: (value: { ok: true }) => void;
    refreshTenantsMock.mockReturnValue(
      new Promise((resolve) => {
        resolveRefresh = resolve;
      }),
    );
    const value = { ...defaultUseTenantValue(), tenants: [tenantA, tenantB] };
    useTenantMock.mockReturnValue(value);
    const user = userEvent.setup();
    const { rerender } = renderPage();

    const refreshButtons = screen.getAllByRole("button", { name: /refresh/i });
    await user.click(refreshButtons[0] as HTMLElement);

    useTenantMock.mockReturnValue({ ...value, loading: true });
    rerender(
      <MemoryRouter>
        <TenantsPage />
      </MemoryRouter>,
    );

    expect(
      screen.getByRole("heading", { name: "Tenants" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Acme")).toBeInTheDocument();

    resolveRefresh!({ ok: true });
  });

  it("renders the skeleton on first load", () => {
    useTenantMock.mockReturnValue({
      ...defaultUseTenantValue(),
      loading: true,
    });
    renderPage();

    expect(
      screen.queryByRole("heading", { name: "Tenants" }),
    ).not.toBeInTheDocument();
    expect(screen.queryByText("No tenants found")).not.toBeInTheDocument();
  });
});
