import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Template } from "@/types/models/template";

const onboardTenantMock = vi.fn();
const toastSuccess = vi.fn();
const toastError = vi.fn();
const useTemplatesPageMock = vi.fn();

vi.mock("@/lib/hooks/api/useTemplates", () => ({
  useTemplatesPage: (...args: unknown[]) => useTemplatesPageMock(...args),
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

function pagedResult(
  data: Template[],
  total = data.length,
  last = 1,
  number = 1,
  size = 50,
) {
  return {
    data: { data, meta: { total, page: { number, size, last } } },
    isLoading: false,
    error: null,
  };
}

import { TemplatesPage } from "@/pages/TemplatesPage";

function renderPage(initial = "/templates") {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initial]}>
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
    useTemplatesPageMock.mockReturnValue(pagedResult(mockTemplatesList));
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
      expect(onboardTenantMock).toHaveBeenCalledWith(
        "My Tenant",
        sampleTemplate,
      );
    });
  });
});

describe("TemplatesPage server-side paging (task-117)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("requests page 1 at the default page size on mount", () => {
    useTemplatesPageMock.mockReturnValue(pagedResult(mockTemplatesList));
    renderPage();
    expect(useTemplatesPageMock).toHaveBeenCalledWith({ number: 1, size: 50 });
  });

  it("renders the pager off meta.total / meta.page.last and requests the next page on click", async () => {
    useTemplatesPageMock.mockReturnValue(
      pagedResult(mockTemplatesList, 120, 3, 1, 50),
    );
    renderPage();

    await screen.findByText(/Page 1 of 3/i);
    expect(screen.getByText(/120 results/i)).toBeInTheDocument();

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: /next page/i }));

    await waitFor(() => {
      expect(useTemplatesPageMock).toHaveBeenLastCalledWith({
        number: 2,
        size: 50,
      });
    });
  });

  it("hydrates the page number from ?page= in the URL", () => {
    useTemplatesPageMock.mockReturnValue(
      pagedResult(mockTemplatesList, 120, 3, 3, 50),
    );
    renderPage("/templates?page=3");
    expect(useTemplatesPageMock).toHaveBeenCalledWith({ number: 3, size: 50 });
  });
});
