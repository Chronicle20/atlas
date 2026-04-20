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

import { TenantsPage } from "@/pages/TenantsPage";

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
    refreshTenants: refreshTenantsMock,
    activeTenant: tenantA,
    setActiveTenant: vi.fn(),
    refreshAndSelectTenant: vi.fn(),
    fetchTenantConfiguration: vi.fn(),
  };
}

async function openRenameDialogFor(tenantId: string) {
  const user = userEvent.setup();
  const row = screen.getByText(
    tenantId === "aaa" ? "Acme" : "Beta",
  ).closest("tr");
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
      expect(within(dialog).getByText(/tenant name is required/i)).toBeInTheDocument();
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
      expect(updateTenantMock).toHaveBeenCalledWith(tenantA, { name: "Acme Renamed" });
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
    const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
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
