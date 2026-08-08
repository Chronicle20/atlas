import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TemplateReseedButton } from "@/components/features/templates/TemplateReseedButton";
import { useTemplate, useReseedTemplate } from "@/lib/hooks/api/useTemplates";

vi.mock("@/lib/hooks/api/useTemplates", () => ({
  useTemplate: vi.fn(),
  useReseedTemplate: vi.fn(),
}));

const toastError = vi.fn();
const toastSuccess = vi.fn();
vi.mock("sonner", () => ({
  toast: {
    error: (...args: unknown[]) => toastError(...args),
    success: (...args: unknown[]) => toastSuccess(...args),
  },
}));

function template(attrs: Record<string, unknown>) {
  return {
    id: "abc-123",
    attributes: {
      region: "GMS",
      majorVersion: 83,
      minorVersion: 1,
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
  vi.mocked(useTemplate).mockReturnValue({ data } as never);
  vi.mocked(useReseedTemplate).mockReturnValue({
    mutateAsync: mutateAsync ?? vi.fn().mockResolvedValue(undefined),
  } as never);
}

describe("TemplateReseedButton", () => {
  beforeEach(() => vi.clearAllMocks());

  it("is disabled when the template ships no seed file", () => {
    mockHooks({ data: template({ shippedRevision: "" }) });
    render(<TemplateReseedButton id="abc-123" />);
    expect(
      screen.getByRole("button", { name: /reset to shipped defaults/i }),
    ).toBeDisabled();
  });

  it("is disabled when shippedRevision is absent", () => {
    mockHooks({ data: template({}) });
    render(<TemplateReseedButton id="abc-123" />);
    expect(
      screen.getByRole("button", { name: /reset to shipped defaults/i }),
    ).toBeDisabled();
  });

  it("is enabled when a seed file ships", () => {
    mockHooks({ data: template({ shippedRevision: "aa" }) });
    render(<TemplateReseedButton id="abc-123" />);
    expect(
      screen.getByRole("button", { name: /reset to shipped defaults/i }),
    ).toBeEnabled();
  });

  it("issues no request when the dialog is dismissed", async () => {
    const mutateAsync = vi.fn().mockResolvedValue(undefined);
    mockHooks({ data: template({ shippedRevision: "aa" }), mutateAsync });
    const user = userEvent.setup();

    render(<TemplateReseedButton id="abc-123" />);
    await user.click(
      screen.getByRole("button", { name: /reset to shipped defaults/i }),
    );
    await screen.findByRole("alertdialog");
    await user.click(screen.getByRole("button", { name: /^cancel$/i }));

    expect(mutateAsync).not.toHaveBeenCalled();
  });

  it("posts the re-seed on confirmation", async () => {
    const mutateAsync = vi.fn().mockResolvedValue(undefined);
    mockHooks({ data: template({ shippedRevision: "aa" }), mutateAsync });
    const user = userEvent.setup();

    render(<TemplateReseedButton id="abc-123" />);
    await user.click(
      screen.getByRole("button", { name: /reset to shipped defaults/i }),
    );
    await screen.findByRole("alertdialog");
    await user.click(screen.getByRole("button", { name: /^reset template$/i }));

    await waitFor(() =>
      expect(mutateAsync).toHaveBeenCalledWith({ id: "abc-123" }),
    );
    await waitFor(() => expect(toastSuccess).toHaveBeenCalled());
  });

  it("surfaces an error toast when the re-seed fails", async () => {
    const mutateAsync = vi.fn().mockRejectedValue(new Error("409 conflict"));
    mockHooks({ data: template({ shippedRevision: "aa" }), mutateAsync });
    const user = userEvent.setup();

    render(<TemplateReseedButton id="abc-123" />);
    await user.click(
      screen.getByRole("button", { name: /reset to shipped defaults/i }),
    );
    await screen.findByRole("alertdialog");
    await user.click(screen.getByRole("button", { name: /^reset template$/i }));

    await waitFor(() => expect(toastError).toHaveBeenCalled());
    expect(toastSuccess).not.toHaveBeenCalled();
  });

  it("names the image comparison in the confirm dialog", async () => {
    mockHooks({ data: template({ shippedRevision: "aa" }) });
    const user = userEvent.setup();

    render(<TemplateReseedButton id="abc-123" />);
    await user.click(
      screen.getByRole("button", { name: /reset to shipped defaults/i }),
    );
    const dialog = await screen.findByRole("alertdialog");
    expect(dialog).toHaveTextContent(/shipped in this image/i);
    expect(dialog).toHaveTextContent(/edits made through the UI will be lost/i);
  });
});
