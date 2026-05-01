// services/atlas-ui/src/components/features/accounts/__tests__/CreateAccountDialog.test.tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const navigateMock = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>(
    "react-router-dom"
  );
  return { ...actual, useNavigate: () => navigateMock };
});

const machineRef: { current: ReturnType<typeof makeMachine> } = {
  current: makeMachine(),
};

function makeMachine(overrides: Partial<ReturnType<typeof defaults>> = {}) {
  return { ...defaults(), ...overrides };
}
function defaults() {
  return {
    status: "idle" as
      | "idle"
      | "submitting"
      | "polling"
      | "success"
      | "timeout"
      | "error",
    accountId: null as number | null,
    errorMessage: null as string | null,
    errorKind: null as "duplicate-name" | "generic" | null,
    submit: vi.fn(async () => {}),
    retry: vi.fn(async () => {}),
    reset: vi.fn(),
  };
}

vi.mock("@/lib/hooks/api/useCreateAndPollAccount", () => ({
  useCreateAndPollAccount: () => machineRef.current,
}));

import { CreateAccountDialog } from "../CreateAccountDialog";
import { toast } from "sonner";

const tenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as never;

function renderDialog(props: Partial<React.ComponentProps<typeof CreateAccountDialog>> = {}) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={qc}>
        <CreateAccountDialog
          tenant={tenant}
          open
          onOpenChange={vi.fn()}
          {...props}
        />
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("CreateAccountDialog", () => {
  beforeEach(() => {
    machineRef.current = makeMachine();
    navigateMock.mockReset();
    vi.mocked(toast.success).mockReset();
    vi.mocked(toast.error).mockReset();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("blocks submit when name is shorter than 4", async () => {
    renderDialog();
    await userEvent.type(screen.getByLabelText(/account name/i), "abc");
    await userEvent.type(screen.getByLabelText(/password/i), "secret1");
    await userEvent.click(screen.getByRole("button", { name: /create/i }));
    expect(machineRef.current.submit).not.toHaveBeenCalled();
    expect(
      await screen.findByText(/Name must be at least 4 characters/i)
    ).toBeInTheDocument();
  });

  it("blocks submit when password is shorter than 6", async () => {
    renderDialog();
    await userEvent.type(screen.getByLabelText(/account name/i), "alice");
    await userEvent.type(screen.getByLabelText(/password/i), "abc");
    await userEvent.click(screen.getByRole("button", { name: /create/i }));
    expect(machineRef.current.submit).not.toHaveBeenCalled();
    expect(
      await screen.findByText(/Password must be at least 6 characters/i)
    ).toBeInTheDocument();
  });

  it("calls submit with valid input", async () => {
    renderDialog();
    await userEvent.type(screen.getByLabelText(/account name/i), "alice");
    await userEvent.type(screen.getByLabelText(/password/i), "secret1");
    await userEvent.click(screen.getByRole("button", { name: /create/i }));
    expect(machineRef.current.submit).toHaveBeenCalledWith({
      name: "alice",
      password: "secret1",
    });
  });

  it("on success: closes dialog, fires success toast, navigates", async () => {
    machineRef.current = makeMachine({ status: "success", accountId: 99 });
    const onOpenChange = vi.fn();
    renderDialog({ onOpenChange });
    expect(onOpenChange).toHaveBeenCalledWith(false);
    expect(toast.success).toHaveBeenCalled();
    expect(navigateMock).toHaveBeenCalledWith("/accounts/99");
  });

  it("on duplicate-name: shows inline name error", () => {
    machineRef.current = makeMachine({
      status: "error",
      errorKind: "duplicate-name",
      errorMessage: "already exists",
    });
    renderDialog();
    expect(screen.getByText(/already exists|name already taken/i)).toBeInTheDocument();
  });

  it("on generic error: fires error toast", () => {
    machineRef.current = makeMachine({
      status: "error",
      errorKind: "generic",
      errorMessage: "boom",
    });
    renderDialog();
    expect(toast.error).toHaveBeenCalledWith("boom");
  });

  it("on timeout: shows error and retry button", async () => {
    machineRef.current = makeMachine({ status: "timeout" });
    renderDialog();
    expect(
      screen.getByText(/timed out waiting for account to appear/i)
    ).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: /retry/i }));
    expect(machineRef.current.retry).toHaveBeenCalled();
  });

  it("cancel during polling closes dialog and resets the machine", async () => {
    machineRef.current = makeMachine({ status: "polling" });
    const onOpenChange = vi.fn();
    renderDialog({ onOpenChange });
    await userEvent.click(screen.getByRole("button", { name: /cancel/i }));
    expect(machineRef.current.reset).toHaveBeenCalled();
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
