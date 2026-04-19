import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Template } from "@/types/models/template";

const onboardTenantMock = vi.fn();
const toastSuccess = vi.fn();
const toastError = vi.fn();

vi.mock("@/lib/hooks/api/useTemplates", () => ({
  useTemplates: () => ({ data: mockTemplatesList, isLoading: false, error: null }),
  useInvalidateTemplates: () => ({ invalidateAll: vi.fn() }),
  useCreateTemplate: () => ({ mutate: vi.fn(), isPending: false }),
  useDeleteTemplate: () => ({ mutate: vi.fn(), isPending: false }),
}));

vi.mock("@/services/api", () => ({
  templatesService: {
    cloneTemplate: vi.fn(),
  },
  onboardingService: {
    onboardTenant: (...args: unknown[]) => onboardTenantMock(...args),
  },
  ConfigurationCreationError: class ConfigurationCreationError extends Error {
    tenantId: string;
    constructor(message: string, tenantId: string) {
      super(message);
      this.tenantId = tenantId;
    }
  },
}));

vi.mock("sonner", () => ({
  toast: {
    success: (...args: unknown[]) => toastSuccess(...args),
    error: (...args: unknown[]) => toastError(...args),
  },
}));

const sampleTemplate: Template = {
  id: "tpl-1",
  attributes: {
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
  },
} as unknown as Template;

const mockTemplatesList: Template[] = [sampleTemplate];

import { TemplatesPage } from "@/pages/TemplatesPage";

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <TemplatesPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

async function openCreateTenantDialog() {
  const user = userEvent.setup();
  const row = screen.getByText("tpl-1").closest("tr");
  if (!row) throw new Error("row not found");
  await user.click(
    within(row as HTMLElement).getByRole("button", { name: /open menu/i }),
  );
  await user.click(await screen.findByText(/create tenant/i));
  return user;
}

describe("TemplatesPage create-tenant-from-template dialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Stub navigation side effect
    Object.defineProperty(window, "location", {
      writable: true,
      value: { ...window.location, replace: vi.fn() },
    });
  });

  it("keeps Create button disabled until a valid name is entered", async () => {
    renderPage();
    const user = await openCreateTenantDialog();

    const dialog = await screen.findByRole("dialog");
    const createButton = within(dialog).getByRole("button", {
      name: /create tenant/i,
    });
    expect(createButton).toBeDisabled();

    const input = within(dialog).getByLabelText("Name");
    await user.type(input, "My Tenant");

    await waitFor(() => expect(createButton).not.toBeDisabled());
  });

  it("submits with the user-entered name (not a hardcoded placeholder)", async () => {
    onboardTenantMock.mockResolvedValueOnce({ tenant: { id: "new-id" } });

    renderPage();
    const user = await openCreateTenantDialog();

    const dialog = await screen.findByRole("dialog");
    const input = within(dialog).getByLabelText("Name");
    await user.type(input, "My Tenant");

    const createButton = within(dialog).getByRole("button", {
      name: /create tenant/i,
    });
    await waitFor(() => expect(createButton).not.toBeDisabled());
    await user.click(createButton);

    await waitFor(() => {
      expect(onboardTenantMock).toHaveBeenCalledWith("My Tenant", sampleTemplate);
    });
  });
});
