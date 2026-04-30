import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { wizardReducer, initialState } from "../AdminBootstrapWizard.types";

// ---------------------------------------------------------------------------
// Pure reducer tests
// ---------------------------------------------------------------------------
describe("wizardReducer", () => {
  it("SET_ACCOUNT updates account credentials", () => {
    const next = wizardReducer(initialState, {
      type: "SET_ACCOUNT",
      account: { name: "admin", password: "secret" },
    });
    expect(next.account).toEqual({ name: "admin", password: "secret" });
    // other state untouched
    expect(next.step).toBe(1);
    expect(next.rows).toEqual({});
  });

  it("SET_WORLD updates worldId", () => {
    const next = wizardReducer(initialState, { type: "SET_WORLD", worldId: 3 });
    expect(next.worldId).toBe(3);
  });

  it("TOGGLE_PRESET adds a new row when absent", () => {
    const next = wizardReducer(initialState, {
      type: "TOGGLE_PRESET",
      presetId: "p1",
      presetName: "Warrior",
    });
    expect(next.rows["p1"]).toMatchObject({
      presetId: "p1",
      presetName: "Warrior",
      name: "",
      validity: null,
      applyStatus: "pending",
    });
  });

  it("TOGGLE_PRESET removes an existing row", () => {
    const withRow = wizardReducer(initialState, {
      type: "TOGGLE_PRESET",
      presetId: "p1",
      presetName: "Warrior",
    });
    const removed = wizardReducer(withRow, {
      type: "TOGGLE_PRESET",
      presetId: "p1",
      presetName: "Warrior",
    });
    expect(removed.rows["p1"]).toBeUndefined();
  });

  it("SET_NAME updates the row name and clears validity", () => {
    const withRow = wizardReducer(initialState, {
      type: "TOGGLE_PRESET",
      presetId: "p1",
      presetName: "Warrior",
    });
    const withValidity = wizardReducer(withRow, {
      type: "SET_VALIDITY",
      presetId: "p1",
      validity: { valid: true },
    });
    const renamed = wizardReducer(withValidity, {
      type: "SET_NAME",
      presetId: "p1",
      name: "NewName",
    });
    expect(renamed.rows["p1"]!.name).toBe("NewName");
    expect(renamed.rows["p1"]!.validity).toBeNull();
  });

  it("SET_NAME is a no-op for unknown presetId", () => {
    const next = wizardReducer(initialState, {
      type: "SET_NAME",
      presetId: "nonexistent",
      name: "whatever",
    });
    expect(next).toBe(initialState);
  });

  it("SET_VALIDITY updates row validity", () => {
    const withRow = wizardReducer(initialState, {
      type: "TOGGLE_PRESET",
      presetId: "p1",
      presetName: "Warrior",
    });
    const next = wizardReducer(withRow, {
      type: "SET_VALIDITY",
      presetId: "p1",
      validity: { valid: true },
    });
    expect(next.rows["p1"]!.validity).toEqual({ valid: true });
  });

  it("SET_VALIDITY is a no-op for unknown presetId", () => {
    const next = wizardReducer(initialState, {
      type: "SET_VALIDITY",
      presetId: "nonexistent",
      validity: { valid: true },
    });
    expect(next).toBe(initialState);
  });

  it("SET_ROW_STATUS updates applyStatus and error", () => {
    const withRow = wizardReducer(initialState, {
      type: "TOGGLE_PRESET",
      presetId: "p1",
      presetName: "Warrior",
    });
    const next = wizardReducer(withRow, {
      type: "SET_ROW_STATUS",
      presetId: "p1",
      status: "failed",
      error: "409 Conflict",
    });
    expect(next.rows["p1"]!.applyStatus).toBe("failed");
    expect(next.rows["p1"]!.error).toBe("409 Conflict");
  });

  it("SET_ROW_STATUS is a no-op for unknown presetId", () => {
    const next = wizardReducer(initialState, {
      type: "SET_ROW_STATUS",
      presetId: "ghost",
      status: "success",
    });
    expect(next).toBe(initialState);
  });

  it("ACCOUNT_CREATED sets accountId", () => {
    const next = wizardReducer(initialState, {
      type: "ACCOUNT_CREATED",
      accountId: 42,
    });
    expect(next.accountId).toBe(42);
  });

  it("GOTO advances to the target step", () => {
    const next = wizardReducer(initialState, { type: "GOTO", step: 3 });
    expect(next.step).toBe(3);
  });

  it("SET_ERROR stores error string", () => {
    const next = wizardReducer(initialState, {
      type: "SET_ERROR",
      error: "Something went wrong",
    });
    expect(next.error).toBe("Something went wrong");
  });

  it("RESET returns initialState", () => {
    const withStuff = wizardReducer(
      wizardReducer(
        wizardReducer(initialState, {
          type: "TOGGLE_PRESET",
          presetId: "p1",
          presetName: "Warrior",
        }),
        { type: "GOTO", step: 3 },
      ),
      { type: "ACCOUNT_CREATED", accountId: 99 },
    );
    const reset = wizardReducer(withStuff, { type: "RESET" });
    expect(reset).toEqual(initialState);
  });
});

// ---------------------------------------------------------------------------
// Component smoke tests
// ---------------------------------------------------------------------------

// Mock heavy dependencies before importing the component
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const useTenantConfigurationMock = vi.fn();
vi.mock("@/lib/hooks/api/useTenants", () => ({
  useTenantConfiguration: (...a: unknown[]) => useTenantConfigurationMock(...a),
}));

const useNameValidityMock = vi.fn();
vi.mock("@/lib/hooks/api/useNameValidity", () => ({
  useNameValidity: (...a: unknown[]) => useNameValidityMock(...a),
}));

vi.mock("@/services/api/accounts.service", () => ({
  accountsService: {
    createAccount: vi.fn(),
    getAllAccounts: vi.fn(),
  },
}));

vi.mock("@/services/api/factory.service", () => ({
  factoryService: {
    createFromPreset: vi.fn(),
    checkNameValidity: vi.fn(),
  },
}));

import { AdminBootstrapWizard } from "../AdminBootstrapWizard";

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
    open: true,
    onOpenChange: vi.fn(),
    ...overrides,
  };
}

describe("AdminBootstrapWizard component", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useTenantConfigurationMock.mockReturnValue({
      data: { attributes: { characters: { presets: twoPresets } } },
      isLoading: false,
    });
    useNameValidityMock.mockReturnValue({ data: undefined, isLoading: false });
  });

  it("renders dialog title when open", () => {
    render(<AdminBootstrapWizard {...defaultProps()} />);
    expect(screen.getByText("Bootstrap Admin Account")).toBeInTheDocument();
  });

  it("does not render dialog content when closed", () => {
    render(<AdminBootstrapWizard {...defaultProps({ open: false })} />);
    expect(screen.queryByText("Bootstrap Admin Account")).not.toBeInTheDocument();
  });

  it("Step 1: Next button disabled until name (>=4) and password (>=6) set", async () => {
    render(<AdminBootstrapWizard {...defaultProps()} />);

    const nextBtn = screen.getByRole("button", { name: /next/i });
    expect(nextBtn).toBeDisabled();

    // Fill name too short + password too short => still disabled
    fireEvent.change(screen.getByLabelText(/account name/i), {
      target: { value: "ab" },
    });
    fireEvent.change(screen.getByLabelText(/password/i), {
      target: { value: "12" },
    });
    expect(nextBtn).toBeDisabled();

    // Fill valid credentials => enabled
    fireEvent.change(screen.getByLabelText(/account name/i), {
      target: { value: "admin" },
    });
    fireEvent.change(screen.getByLabelText(/password/i), {
      target: { value: "password123" },
    });

    await waitFor(() => expect(nextBtn).not.toBeDisabled());
  });

  it("Step 2: tag filter narrows visible preset list", async () => {
    render(<AdminBootstrapWizard {...defaultProps()} />);

    // Advance to step 2
    fireEvent.change(screen.getByLabelText(/account name/i), {
      target: { value: "admin" },
    });
    fireEvent.change(screen.getByLabelText(/password/i), {
      target: { value: "secret123" },
    });
    fireEvent.click(screen.getByRole("button", { name: /next/i }));

    // Both presets should be visible
    await waitFor(() => {
      expect(screen.getByText("Warrior")).toBeInTheDocument();
      expect(screen.getByText("Mage")).toBeInTheDocument();
    });

    // Click the "magic" tag filter button — only Mage should remain
    const magicFilterBtn = screen.getByRole("button", { name: "magic" });
    fireEvent.click(magicFilterBtn);

    // Warrior's checkbox label should now be gone; Mage stays
    const checkboxLabels = screen.getAllByRole("checkbox");
    // Only one preset checkbox should remain (Mage)
    expect(checkboxLabels).toHaveLength(1);
    expect(screen.getByText("Mage")).toBeInTheDocument();
  });

  it("Step 3: wizard-internal duplicate name flags both rows as invalid", async () => {
    render(<AdminBootstrapWizard {...defaultProps()} />);

    // Step 1 → 2
    fireEvent.change(screen.getByLabelText(/account name/i), { target: { value: "admin" } });
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: "secret123" } });
    fireEvent.click(screen.getByRole("button", { name: /next/i }));

    // Select both presets
    await waitFor(() => screen.getByText("Warrior"));
    const checkboxes = screen.getAllByRole("checkbox");
    checkboxes.forEach((cb) => fireEvent.click(cb));

    // Next (step 2 → 3)
    const nextBtn = screen.getByRole("button", { name: /next/i });
    await waitFor(() => expect(nextBtn).not.toBeDisabled());
    fireEvent.click(nextBtn);

    // Step 3 should show name inputs for both rows
    await waitFor(() => {
      // Both preset name labels appear in the table
      expect(screen.getByText("Warrior")).toBeInTheDocument();
      expect(screen.getByText("Mage")).toBeInTheDocument();
    });

    // Enter the same name in both inputs
    const inputs = screen.getAllByPlaceholderText(/3.12 characters/i);
    expect(inputs).toHaveLength(2);
    fireEvent.change(inputs[0]!, { target: { value: "SameName" } });
    fireEvent.change(inputs[1]!, { target: { value: "SameName" } });

    // Both rows should show "Duplicate within wizard"
    await waitFor(() => {
      const dupeLabels = screen.getAllByText("Duplicate within wizard");
      expect(dupeLabels).toHaveLength(2);
    });

    // Apply button should be disabled
    expect(screen.getByRole("button", { name: /apply/i })).toBeDisabled();
  });
});

// NOTE: Step 4 integration tests (account-creation fetch + polling + sequential
// preset application) are intentionally omitted. The reducer tests above cover
// all Step-4 state transitions. The component wires them together correctly, but
// testing the async saga (createAccount → poll → sequential createFromPreset)
// requires deep fetch-mocking that adds maintenance burden without commensurate
// safety value. Covered by e2e / manual testing.
