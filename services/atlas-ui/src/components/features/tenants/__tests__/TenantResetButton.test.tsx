import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TenantResetButton } from "@/components/features/tenants/TenantResetButton";
import {
  useTenantConfiguration,
  useResetTenantConfiguration,
} from "@/lib/hooks/api/useTenants";

vi.mock("@/lib/hooks/api/useTenants", () => ({
  useTenantConfiguration: vi.fn(),
  useResetTenantConfiguration: vi.fn(),
}));

const toastError = vi.fn();
const toastSuccess = vi.fn();
vi.mock("sonner", () => ({
  toast: {
    error: (...args: unknown[]) => toastError(...args),
    success: (...args: unknown[]) => toastSuccess(...args),
  },
}));

function tenantConfig(attrs: Record<string, unknown>) {
  return {
    id: "t1",
    type: "tenants",
    attributes: {
      region: "GMS",
      majorVersion: 83,
      minorVersion: 1,
      usesPin: true,
      ...attrs,
    },
  };
}

function mockHooks({
  data,
  mutateAsync,
}: {
  data: unknown;
  mutateAsync?: ReturnType<typeof vi.fn>;
}) {
  vi.mocked(useTenantConfiguration).mockReturnValue({ data } as never);
  vi.mocked(useResetTenantConfiguration).mockReturnValue({
    mutateAsync: mutateAsync ?? vi.fn().mockResolvedValue(undefined),
  } as never);
}

describe("TenantResetButton", () => {
  beforeEach(() => vi.clearAllMocks());

  it("is disabled with an explanatory tooltip when no baseline resolves", async () => {
    mockHooks({ data: tenantConfig({ baselineTemplateId: "" }) });
    const user = userEvent.setup();
    render(<TenantResetButton id="t1" />);

    const button = screen.getByRole("button", { name: /reset to template/i });
    expect(button).toBeDisabled();

    await user.hover(button);
    expect(
      await screen.findByText(
        /no configuration template resolves for this tenant's region and version/i,
      ),
    ).toBeInTheDocument();
  });

  it("is disabled when baselineTemplateId is absent", () => {
    mockHooks({ data: tenantConfig({}) });
    render(<TenantResetButton id="t1" />);
    expect(
      screen.getByRole("button", { name: /reset to template/i }),
    ).toBeDisabled();
  });

  it("is enabled when a baseline resolves", () => {
    mockHooks({ data: tenantConfig({ baselineTemplateId: "b1" }) });
    render(<TenantResetButton id="t1" />);
    expect(
      screen.getByRole("button", { name: /reset to template/i }),
    ).toBeEnabled();
  });

  it("Cancel renders before the destructive action", async () => {
    mockHooks({ data: tenantConfig({ baselineTemplateId: "b1" }) });
    const user = userEvent.setup();
    render(<TenantResetButton id="t1" />);

    await user.click(
      screen.getByRole("button", { name: /reset to template/i }),
    );
    const dialog = await screen.findByRole("alertdialog");
    const buttons = within(dialog).getAllByRole("button");
    const cancelIndex = buttons.findIndex((b) =>
      /^cancel$/i.test(b.textContent ?? ""),
    );
    const confirmIndex = buttons.findIndex((b) =>
      /^reset tenant$/i.test(b.textContent ?? ""),
    );
    expect(cancelIndex).toBeGreaterThanOrEqual(0);
    expect(confirmIndex).toBeGreaterThanOrEqual(0);
    expect(cancelIndex).toBeLessThan(confirmIndex);
  });

  it("Cancel closes without calling the mutation", async () => {
    const mutateAsync = vi.fn().mockResolvedValue(undefined);
    mockHooks({
      data: tenantConfig({ baselineTemplateId: "b1" }),
      mutateAsync,
    });
    const user = userEvent.setup();
    render(<TenantResetButton id="t1" />);

    await user.click(
      screen.getByRole("button", { name: /reset to template/i }),
    );
    await screen.findByRole("alertdialog");
    await user.click(screen.getByRole("button", { name: /^cancel$/i }));

    expect(mutateAsync).not.toHaveBeenCalled();
    await waitFor(() =>
      expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument(),
    );
  });

  it("confirm calls the mutation with no sections for the whole document", async () => {
    const mutateAsync = vi.fn().mockResolvedValue(undefined);
    mockHooks({
      data: tenantConfig({ baselineTemplateId: "b1" }),
      mutateAsync,
    });
    const user = userEvent.setup();
    render(<TenantResetButton id="t1" />);

    await user.click(
      screen.getByRole("button", { name: /reset to template/i }),
    );
    await screen.findByRole("alertdialog");
    await user.click(screen.getByRole("button", { name: /^reset tenant$/i }));

    await waitFor(() =>
      expect(mutateAsync).toHaveBeenCalledWith({
        id: "t1",
      }),
    );
  });

  it("confirm calls the mutation with the scoped sections", async () => {
    const mutateAsync = vi.fn().mockResolvedValue(undefined);
    mockHooks({
      data: tenantConfig({ baselineTemplateId: "b1" }),
      mutateAsync,
    });
    const user = userEvent.setup();
    render(
      <TenantResetButton
        id="t1"
        sections={["socket"]}
        sectionLabel="socket handlers and writers"
      />,
    );

    await user.click(
      screen.getByRole("button", {
        name: /reset socket handlers and writers/i,
      }),
    );
    await screen.findByRole("alertdialog");
    await user.click(
      screen.getByRole("button", {
        name: /^reset socket handlers and writers$/i,
      }),
    );

    await waitFor(() =>
      expect(mutateAsync).toHaveBeenCalledWith({
        id: "t1",
        sections: ["socket"],
      }),
    );
  });

  it("a failure toasts the server detail and leaves the dialog open", async () => {
    const mutateAsync = vi
      .fn()
      .mockRejectedValue(new Error("baseline is unprocessable"));
    mockHooks({
      data: tenantConfig({ baselineTemplateId: "b1" }),
      mutateAsync,
    });
    const user = userEvent.setup();
    render(<TenantResetButton id="t1" />);

    await user.click(
      screen.getByRole("button", { name: /reset to template/i }),
    );
    await screen.findByRole("alertdialog");
    await user.click(screen.getByRole("button", { name: /^reset tenant$/i }));

    await waitFor(() =>
      expect(toastError).toHaveBeenCalledWith(
        expect.stringContaining("baseline is unprocessable"),
      ),
    );
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });

  it("a success toasts and closes", async () => {
    const mutateAsync = vi.fn().mockResolvedValue(undefined);
    mockHooks({
      data: tenantConfig({ baselineTemplateId: "b1" }),
      mutateAsync,
    });
    const user = userEvent.setup();
    render(<TenantResetButton id="t1" />);

    await user.click(
      screen.getByRole("button", { name: /reset to template/i }),
    );
    await screen.findByRole("alertdialog");
    await user.click(screen.getByRole("button", { name: /^reset tenant$/i }));

    await waitFor(() => expect(toastSuccess).toHaveBeenCalled());
    await waitFor(() =>
      expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument(),
    );
  });

  it("the whole-document dialog copy states all three facts", async () => {
    mockHooks({ data: tenantConfig({ baselineTemplateId: "b1" }) });
    const user = userEvent.setup();
    render(<TenantResetButton id="t1" />);

    await user.click(
      screen.getByRole("button", { name: /reset to template/i }),
    );
    const dialog = await screen.findByRole("alertdialog");
    expect(dialog).toHaveTextContent(
      /edits you have made through the ui.*will be lost/i,
    );
    expect(dialog).toHaveTextContent(
      /id, region, version, world configuration and diagnostics are unchanged/i,
    );
    expect(dialog).toHaveTextContent(/no game data.*is affected/i);
  });

  it("the scoped dialog copy names the section", async () => {
    mockHooks({ data: tenantConfig({ baselineTemplateId: "b1" }) });
    const user = userEvent.setup();
    render(
      <TenantResetButton
        id="t1"
        sections={["socket"]}
        sectionLabel="socket handlers and writers"
      />,
    );

    await user.click(
      screen.getByRole("button", {
        name: /reset socket handlers and writers/i,
      }),
    );
    const dialog = await screen.findByRole("alertdialog");
    expect(dialog).toHaveTextContent(/socket handlers and writers/i);
  });
});
