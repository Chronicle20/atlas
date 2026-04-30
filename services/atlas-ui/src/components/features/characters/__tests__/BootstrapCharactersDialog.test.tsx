import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";

// ---------------------------------------------------------------------------
// Mock heavy dependencies before importing the component
// ---------------------------------------------------------------------------
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const useTenantConfigurationMock = vi.fn();
vi.mock("@/lib/hooks/api/useTenants", () => ({
  useTenantConfiguration: (...a: unknown[]) => useTenantConfigurationMock(...a),
}));

const useNameValidityMock = vi.fn();
vi.mock("@/lib/hooks/api/useNameValidity", () => ({
  useNameValidity: (...a: unknown[]) => useNameValidityMock(...a),
}));

vi.mock("@/services/api/factory.service", () => ({
  factoryService: {
    createFromPreset: vi.fn(),
    checkNameValidity: vi.fn(),
  },
}));

import { BootstrapCharactersDialog } from "../BootstrapCharactersDialog";

const fakeTenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as never;

const twoPresets = [
  { id: "preset-warrior", attributes: { name: "Warrior", tags: ["beginner", "melee"] } },
  { id: "preset-mage", attributes: { name: "Mage", tags: ["magic"] } },
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

describe("BootstrapCharactersDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useTenantConfigurationMock.mockReturnValue({
      data: { attributes: { characters: { presets: twoPresets } } },
      isLoading: false,
    });
    useNameValidityMock.mockReturnValue({ data: undefined, isLoading: false });
  });

  it("renders dialog title when open", () => {
    render(<BootstrapCharactersDialog {...defaultProps()} />);
    expect(screen.getByText("Bootstrap characters")).toBeInTheDocument();
  });

  it("does not render dialog content when closed", () => {
    render(<BootstrapCharactersDialog {...defaultProps({ open: false })} />);
    expect(screen.queryByText("Bootstrap characters")).not.toBeInTheDocument();
  });

  it("renders the flow's first step label", () => {
    render(<BootstrapCharactersDialog {...defaultProps()} />);
    expect(screen.getByText(/Step 1 of 3/i)).toBeInTheDocument();
  });

  it("renders preset list from tenant config", async () => {
    render(<BootstrapCharactersDialog {...defaultProps()} />);
    await waitFor(() => {
      expect(screen.getByText("Warrior")).toBeInTheDocument();
      expect(screen.getByText("Mage")).toBeInTheDocument();
    });
  });

  it("Next button is disabled when no presets selected", () => {
    render(<BootstrapCharactersDialog {...defaultProps()} />);
    const nextBtn = screen.getByRole("button", { name: /next/i });
    expect(nextBtn).toBeDisabled();
  });

  it("Next button enables after selecting a preset", async () => {
    render(<BootstrapCharactersDialog {...defaultProps()} />);
    await waitFor(() => screen.getByText("Warrior"));

    const [firstCheckbox] = screen.getAllByRole("checkbox");
    fireEvent.click(firstCheckbox!);

    await waitFor(() => {
      expect(screen.getByRole("button", { name: /next/i })).not.toBeDisabled();
    });
  });

  it("tag filter narrows visible presets", async () => {
    render(<BootstrapCharactersDialog {...defaultProps()} />);
    await waitFor(() => screen.getByText("Warrior"));

    // Click the "magic" tag — only Mage should remain
    const magicFilterBtn = screen.getByRole("button", { name: "magic" });
    fireEvent.click(magicFilterBtn);

    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(1);
    expect(screen.getByText("Mage")).toBeInTheDocument();
    expect(screen.queryByText("Warrior")).not.toBeInTheDocument();
  });

  it("Cancel button calls onOpenChange(false)", async () => {
    const onOpenChange = vi.fn();
    render(<BootstrapCharactersDialog {...defaultProps({ onOpenChange })} />);

    const cancelBtn = screen.getByRole("button", { name: /cancel/i });
    fireEvent.click(cancelBtn);

    await waitFor(() => {
      expect(onOpenChange).toHaveBeenCalledWith(false);
    });
  });

  it("advances to name-override step after selecting presets and clicking Next", async () => {
    render(<BootstrapCharactersDialog {...defaultProps()} />);
    await waitFor(() => screen.getByText("Warrior"));

    // Select one preset
    const [firstCheckbox] = screen.getAllByRole("checkbox");
    fireEvent.click(firstCheckbox!);

    // Advance to step 2
    await waitFor(() => expect(screen.getByRole("button", { name: /next/i })).not.toBeDisabled());
    fireEvent.click(screen.getByRole("button", { name: /next/i }));

    await waitFor(() => {
      expect(screen.getByText(/Step 2 of 3/i)).toBeInTheDocument();
      expect(screen.getByText(/Enter a character name/i)).toBeInTheDocument();
    });
  });

  it("duplicate names within the selection are flagged invalid", async () => {
    render(<BootstrapCharactersDialog {...defaultProps()} />);
    await waitFor(() => screen.getByText("Warrior"));

    // Select both presets
    const checkboxes = screen.getAllByRole("checkbox");
    checkboxes.forEach((cb) => fireEvent.click(cb));

    await waitFor(() => expect(screen.getByRole("button", { name: /next/i })).not.toBeDisabled());
    fireEvent.click(screen.getByRole("button", { name: /next/i }));

    // Step 2 — name overrides
    await waitFor(() => screen.getByText(/Step 2 of 3/i));

    const inputs = screen.getAllByPlaceholderText(/3.12 characters/i);
    expect(inputs).toHaveLength(2);
    fireEvent.change(inputs[0]!, { target: { value: "SameName" } });
    fireEvent.change(inputs[1]!, { target: { value: "SameName" } });

    await waitFor(() => {
      const dupeLabels = screen.getAllByText("Duplicate within selection");
      expect(dupeLabels).toHaveLength(2);
    });

    // Apply button should be disabled due to duplicates
    expect(screen.getByRole("button", { name: /apply/i })).toBeDisabled();
  });
});
