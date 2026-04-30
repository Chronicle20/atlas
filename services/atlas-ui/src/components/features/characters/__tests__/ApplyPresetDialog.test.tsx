import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { ApplyPresetDialog } from "../ApplyPresetDialog";

// Mock sonner
vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

// --- Hook mocks ---
const useTenantConfigurationMock = vi.fn();
vi.mock("@/lib/hooks/api/useTenants", () => ({
  useTenantConfiguration: (...a: unknown[]) => useTenantConfigurationMock(...a),
}));

const useNameValidityMock = vi.fn();
vi.mock("@/lib/hooks/api/useNameValidity", () => ({
  useNameValidity: (...a: unknown[]) => useNameValidityMock(...a),
}));

const mutate = vi.fn();
const useCreateCharacterFromPresetMock = vi.fn();
vi.mock("@/lib/hooks/api/useCharacterFromPresetMutation", () => ({
  useCreateCharacterFromPreset: (...a: unknown[]) =>
    useCreateCharacterFromPresetMock(...a),
}));

// --- Fixtures ---
const fakeTenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as never;

const twoPresets = [
  { id: "preset-1", attributes: { name: "Warrior", jobId: 100 } },
  { id: "preset-2", attributes: { name: "Mage", jobId: 200 } },
];

function defaultProps(overrides: Record<string, unknown> = {}) {
  return {
    tenant: fakeTenant,
    accountId: 42,
    open: true,
    onOpenChange: vi.fn(),
    ...overrides,
  };
}

describe("ApplyPresetDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();

    useTenantConfigurationMock.mockReturnValue({
      data: { attributes: { characters: { presets: twoPresets } } },
      isLoading: false,
    });

    useNameValidityMock.mockReturnValue({
      data: undefined,
      isLoading: false,
    });

    useCreateCharacterFromPresetMock.mockReturnValue({
      mutate,
      isPending: false,
    });
  });

  it("renders without crashing and shows title", () => {
    render(<ApplyPresetDialog {...defaultProps()} />);
    expect(screen.getByText("Add character from preset")).toBeInTheDocument();
  });

  it("does not render when closed", () => {
    render(<ApplyPresetDialog {...defaultProps({ open: false })} />);
    expect(screen.queryByText("Add character from preset")).not.toBeInTheDocument();
  });

  it("shows both presets as native select options (accessible via aria-hidden select)", () => {
    render(<ApplyPresetDialog {...defaultProps()} />);
    // Radix Select renders a hidden native <select> for accessibility; verify
    // both preset names appear as option text without triggering pointer events.
    const nativeSelect = document.querySelector(
      'select[aria-hidden="true"]',
    ) as HTMLSelectElement;
    expect(nativeSelect).toBeTruthy();
    const optionTexts = Array.from(nativeSelect.options).map((o) => o.text);
    expect(optionTexts).toContain("Warrior");
    expect(optionTexts).toContain("Mage");
  });

  it("shows 'Name is available.' when validity is valid", () => {
    useNameValidityMock.mockReturnValue({
      data: { valid: true },
      isLoading: false,
    });
    render(<ApplyPresetDialog {...defaultProps()} />);
    expect(screen.getByText("Name is available.")).toBeInTheDocument();
  });

  it("shows name-invalid message when validity is not valid", () => {
    useNameValidityMock.mockReturnValue({
      data: { valid: false, reason: "duplicate", detail: "That name is taken" },
      isLoading: false,
    });
    render(<ApplyPresetDialog {...defaultProps()} />);
    expect(screen.getByText("That name is taken")).toBeInTheDocument();
  });

  it("Apply button is disabled when validity is undefined", () => {
    useNameValidityMock.mockReturnValue({ data: undefined, isLoading: false });
    render(<ApplyPresetDialog {...defaultProps()} />);
    expect(screen.getByRole("button", { name: /apply/i })).toBeDisabled();
  });

  it("Apply button is disabled when validity is not valid", () => {
    useNameValidityMock.mockReturnValue({
      data: { valid: false, reason: "duplicate" },
      isLoading: false,
    });
    render(<ApplyPresetDialog {...defaultProps()} />);
    expect(screen.getByRole("button", { name: /apply/i })).toBeDisabled();
  });

  it("calls onOpenChange(false) when Cancel is clicked", async () => {
    const onOpenChange = vi.fn();
    render(<ApplyPresetDialog {...defaultProps({ onOpenChange })} />);
    const cancelBtn = screen.getByRole("button", { name: /cancel/i });
    fireEvent.click(cancelBtn);
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("on 409 error mutate callback: sets inline name field error", async () => {
    // Capture the callbacks passed to mutate so we can invoke them directly
    let capturedCallbacks: {
      onSuccess?: () => void;
      onError?: (e: unknown) => void;
    } = {};

    mutate.mockImplementation(
      (
        _payload: unknown,
        cbs: { onSuccess?: () => void; onError?: (e: unknown) => void },
      ) => {
        capturedCallbacks = cbs;
      },
    );

    useNameValidityMock.mockReturnValue({
      data: { valid: true },
      isLoading: false,
    });

    render(<ApplyPresetDialog {...defaultProps()} />);

    // Fill the name field so the form can pass zod validation for the name
    const nameInput = screen.getByPlaceholderText("3-12 characters");
    await userEvent.type(nameInput, "TestName");

    // Use the hidden native select (aria-hidden) to set presetId, bypassing
    // Radix UI's pointer-event-based portal which doesn't work in jsdom
    const nativeSelect = document.querySelector(
      'select[aria-hidden="true"]',
    ) as HTMLSelectElement | null;
    if (nativeSelect) {
      fireEvent.change(nativeSelect, { target: { value: "preset-1" } });
    }

    // Submit the form
    const form = document.querySelector("form") as HTMLFormElement;
    fireEvent.submit(form);

    // Wait until mutate was called (form passed zod validation)
    await waitFor(() => expect(mutate).toHaveBeenCalled());

    // Invoke the onError callback with a 409
    const err409 = Object.assign(new Error("Name already taken."), { status: 409 });
    capturedCallbacks.onError?.(err409);

    await waitFor(() => {
      expect(screen.getByText("Name already taken.")).toBeInTheDocument();
    });
  });

  it("on success mutate callback: fires toast and closes dialog", async () => {
    const { toast } = await import("sonner");
    const onOpenChange = vi.fn();

    let capturedCallbacks: {
      onSuccess?: () => void;
      onError?: (e: unknown) => void;
    } = {};

    mutate.mockImplementation(
      (
        _payload: unknown,
        cbs: { onSuccess?: () => void; onError?: (e: unknown) => void },
      ) => {
        capturedCallbacks = cbs;
      },
    );

    useNameValidityMock.mockReturnValue({
      data: { valid: true },
      isLoading: false,
    });

    render(<ApplyPresetDialog {...defaultProps({ onOpenChange })} />);

    const nameInput = screen.getByPlaceholderText("3-12 characters");
    await userEvent.type(nameInput, "TestName");

    const nativeSelect = document.querySelector(
      'select[aria-hidden="true"]',
    ) as HTMLSelectElement | null;
    if (nativeSelect) {
      fireEvent.change(nativeSelect, { target: { value: "preset-1" } });
    }

    const form = document.querySelector("form") as HTMLFormElement;
    fireEvent.submit(form);

    await waitFor(() => expect(mutate).toHaveBeenCalled());

    capturedCallbacks.onSuccess?.();

    await waitFor(() => {
      expect(toast.success).toHaveBeenCalledWith("Character creation started.");
      expect(onOpenChange).toHaveBeenCalledWith(false);
    });
  });
});
