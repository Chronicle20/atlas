import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";

import { ConfigExportButton } from "@/components/features/config/ConfigExportButton";
import { DetailActionBarProvider } from "@/components/DetailActionBarContext";
import { templateKeys } from "@/lib/hooks/api/useTemplates";
import { tenantKeys } from "@/lib/hooks/api/useTenants";
import { downloadJson } from "@/lib/utils/download-json";
import { templatesService } from "@/services/api/templates.service";
import { tenantsService } from "@/services/api/tenants.service";
import { toast } from "sonner";

vi.mock("@/lib/utils/download-json", () => ({ downloadJson: vi.fn() }));
vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));
vi.mock("@/services/api/templates.service", () => ({
  templatesService: { getById: vi.fn() },
}));
vi.mock("@/services/api/tenants.service", () => ({
  tenantsService: { getTenantConfigurationById: vi.fn() },
}));

const ID = "8b1d4c4e-0000-4000-8000-000000000000";

const attributes = {
  region: "GMS",
  majorVersion: 83,
  minorVersion: 1,
  usesPin: false,
  socket: { handlers: [], writers: [] },
  characters: { templates: [], presets: [] },
  npcs: null,
  worlds: null,
};

function makeClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
}

function wrapper(client: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={client}>
        <DetailActionBarProvider>{children}</DetailActionBarProvider>
      </QueryClientProvider>
    );
  };
}

describe("ConfigExportButton", () => {
  beforeEach(() => {
    vi.mocked(templatesService.getById).mockResolvedValue({
      id: ID,
      attributes,
    } as never);
    vi.mocked(tenantsService.getTenantConfigurationById).mockResolvedValue({
      id: ID,
      attributes,
    } as never);
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("is disabled until the query has data", async () => {
    const client = makeClient();
    render(<ConfigExportButton kind="template" id={ID} />, {
      wrapper: wrapper(client),
    });

    const button = screen.getByRole("button", { name: "Export" });
    expect(button).toBeDisabled();
    await waitFor(() => expect(button).toBeEnabled());
  });

  it("stays disabled when there is no id", () => {
    const client = makeClient();
    render(<ConfigExportButton kind="template" id={undefined} />, {
      wrapper: wrapper(client),
    });

    expect(screen.getByRole("button", { name: "Export" })).toBeDisabled();
  });

  it("stays disabled when the query errors", async () => {
    vi.mocked(templatesService.getById).mockRejectedValue(new Error("boom"));
    const client = makeClient();
    render(<ConfigExportButton kind="template" id={ID} />, {
      wrapper: wrapper(client),
    });

    const button = screen.getByRole("button", { name: "Export" });
    await waitFor(() =>
      expect(vi.mocked(templatesService.getById)).toHaveBeenCalled(),
    );
    expect(button).toBeDisabled();
  });

  it("downloads the projected attributes under the seed-style filename", async () => {
    const client = makeClient();
    client.setQueryData(templateKeys.detail(ID), { id: ID, attributes });
    render(<ConfigExportButton kind="template" id={ID} />, {
      wrapper: wrapper(client),
    });

    const button = screen.getByRole("button", { name: "Export" });
    await waitFor(() => expect(button).toBeEnabled());
    await userEvent.click(button);

    expect(vi.mocked(downloadJson)).toHaveBeenCalledTimes(1);
    const call = vi.mocked(downloadJson).mock.calls[0];
    expect(call).toBeDefined();
    const [filename, payload] = call!;
    expect(filename).toBe("template_gms_83_1.json");
    expect(payload).not.toHaveProperty("id");
    expect(payload).toMatchObject({ region: "GMS", npcs: [], worlds: [] });
    expect(vi.mocked(toast.success)).toHaveBeenCalledWith("Template exported");
  });

  it("fires no additional request when clicked with the resource cached", async () => {
    const client = makeClient();
    client.setQueryData(templateKeys.detail(ID), { id: ID, attributes });
    render(<ConfigExportButton kind="template" id={ID} />, {
      wrapper: wrapper(client),
    });

    const button = screen.getByRole("button", { name: "Export" });
    await waitFor(() => expect(button).toBeEnabled());
    await waitFor(() =>
      expect(vi.mocked(templatesService.getById)).toHaveBeenCalledTimes(1),
    );
    const before = vi.mocked(templatesService.getById).mock.calls.length;

    await userEvent.click(button);

    expect(vi.mocked(templatesService.getById).mock.calls.length).toBe(before);
  });

  it("exports a tenant configuration with the tenant_ prefix and toast", async () => {
    const client = makeClient();
    client.setQueryData(tenantKeys.configDetail(ID), { id: ID, attributes });
    render(<ConfigExportButton kind="tenant" id={ID} />, {
      wrapper: wrapper(client),
    });

    const button = screen.getByRole("button", { name: "Export" });
    await waitFor(() => expect(button).toBeEnabled());
    await userEvent.click(button);

    expect(vi.mocked(downloadJson).mock.calls[0]?.[0]).toBe(
      "tenant_gms_83_1.json",
    );
    expect(vi.mocked(toast.success)).toHaveBeenCalledWith("Tenant exported");
    expect(vi.mocked(templatesService.getById)).not.toHaveBeenCalled();
  });

  it("shows an error toast when the download throws", async () => {
    vi.mocked(downloadJson).mockImplementation(() => {
      throw new Error("nope");
    });
    const client = makeClient();
    client.setQueryData(templateKeys.detail(ID), { id: ID, attributes });
    render(<ConfigExportButton kind="template" id={ID} />, {
      wrapper: wrapper(client),
    });

    const button = screen.getByRole("button", { name: "Export" });
    await waitFor(() => expect(button).toBeEnabled());
    await userEvent.click(button);

    // The thrown error's own message must survive into the toast - a bare
    // "Export failed" would discard the only diagnostic the user ever sees.
    expect(vi.mocked(toast.error)).toHaveBeenCalledWith("Export failed: nope");
    expect(vi.mocked(toast.success)).not.toHaveBeenCalled();
  });

  it("marks the icon aria-hidden so the accessible name is just Export", async () => {
    const client = makeClient();
    client.setQueryData(templateKeys.detail(ID), { id: ID, attributes });
    render(<ConfigExportButton kind="template" id={ID} />, {
      wrapper: wrapper(client),
    });

    const button = screen.getByRole("button", { name: "Export" });
    expect(button.querySelector("svg")).toHaveAttribute("aria-hidden", "true");
  });
});
