import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";

// Mock sonner
vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

// Mock the heavy CharacterRenderer so preset tiles don't pull in MapleStory.io.
vi.mock("@/components/features/characters/CharacterRenderer", () => ({
  CharacterRenderer: ({
    character,
  }: {
    character: { id: string; attributes: { name: string } };
  }) => <div data-testid="renderer" data-name={character.attributes.name} />,
}));

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

const useServicesMock = vi.fn();
vi.mock("@/lib/hooks/api/useServices", () => ({
  useServices: (...a: unknown[]) => useServicesMock(...a),
}));

import { ApplyPresetDialog } from "../ApplyPresetDialog";

const fakeTenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as never;

const presetAttrs = {
  description: "",
  tags: [],
  gender: 0 as const,
  face: 20000,
  hair: 30000,
  hairColor: 0,
  skinColor: 0,
  mapId: 100000000,
  level: 1,
  meso: 0,
  gm: 0,
  stats: { str: 0, dex: 0, int: 0, luk: 0, hp: 50, mp: 5, ap: 0, sp: "" },
  defaultName: "",
  equipment: [],
  inventory: [],
  skills: [],
};

const twoPresets = [
  {
    id: "preset-1",
    attributes: { ...presetAttrs, name: "Warrior", jobId: 100 },
  },
  { id: "preset-2", attributes: { ...presetAttrs, name: "Mage", jobId: 200 } },
];

const channelServiceWithBothWorlds = {
  id: "svc-channel-1",
  attributes: {
    type: "channel-service" as const,
    tasks: [],
    tenants: [
      {
        id: "t1",
        ipAddress: "10.0.0.1",
        worlds: [
          { id: 0, channels: [{ id: 0, port: 7575 }] },
          { id: 1, channels: [{ id: 0, port: 7576 }] },
        ],
      },
    ],
  },
};

const channelServiceWithOnlyWorldZero = {
  id: "svc-channel-1",
  attributes: {
    type: "channel-service" as const,
    tasks: [],
    tenants: [
      {
        id: "t1",
        ipAddress: "10.0.0.1",
        worlds: [{ id: 0, channels: [{ id: 0, port: 7575 }] }],
      },
    ],
  },
};

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
      data: {
        attributes: {
          characters: { presets: twoPresets },
          worlds: [
            { name: "Scania", flag: "" },
            { name: "Bera", flag: "" },
          ],
        },
      },
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

    useServicesMock.mockReturnValue({
      data: [channelServiceWithBothWorlds],
      isLoading: false,
    });
  });

  it("renders without crashing and shows title", () => {
    render(<ApplyPresetDialog {...defaultProps()} />);
    expect(screen.getByText("Add character from preset")).toBeInTheDocument();
  });

  it("does not render when closed", () => {
    render(<ApplyPresetDialog {...defaultProps({ open: false })} />);
    expect(
      screen.queryByText("Add character from preset"),
    ).not.toBeInTheDocument();
  });

  it("renders one preset tile per configured preset", () => {
    render(<ApplyPresetDialog {...defaultProps()} />);
    const tiles = screen.getAllByRole("radio");
    expect(tiles).toHaveLength(2);
    expect(screen.getByText("Warrior")).toBeInTheDocument();
    expect(screen.getByText("Mage")).toBeInTheDocument();
  });

  it("marks the clicked preset tile as aria-checked", async () => {
    render(<ApplyPresetDialog {...defaultProps()} />);
    const warriorTile = screen
      .getAllByRole("radio")
      .find((el) => el.textContent?.includes("Warrior"))!;
    expect(warriorTile).toHaveAttribute("aria-checked", "false");
    await userEvent.click(warriorTile);
    await waitFor(() =>
      expect(warriorTile).toHaveAttribute("aria-checked", "true"),
    );
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

  it("Apply button enables after preset + world + valid name are chosen (mode: onChange)", async () => {
    useNameValidityMock.mockReturnValue({
      data: { valid: true },
      isLoading: false,
    });
    render(<ApplyPresetDialog {...defaultProps()} />);

    const warriorTile = screen
      .getAllByRole("radio")
      .find((el) => el.textContent?.includes("Warrior"))!;
    await userEvent.click(warriorTile);

    // Set worldId via the hidden native select rendered by Radix.
    const nativeSelect = document.querySelector(
      'select[aria-hidden="true"]',
    ) as HTMLSelectElement | null;
    if (nativeSelect) {
      fireEvent.change(nativeSelect, { target: { value: "0" } });
    }

    const nameInput = screen.getByPlaceholderText("3-12 characters");
    await userEvent.type(nameInput, "Foobar");

    await waitFor(() => {
      expect(screen.getByRole("button", { name: /apply/i })).not.toBeDisabled();
    });
  });

  it("Apply button enables when the name check resolves AFTER the name is typed (async timing)", async () => {
    // Mimic real usage: useNameValidity starts undefined (debounce + query
    // in-flight) and only resolves to { valid: true } after the user has
    // already filled the form. The synchronous-validity test above can't catch
    // the regression because returning a value from the first render subscribes
    // RHF's formState.isValid from mount. Here isValid is never read while
    // validity is undefined, so without the unconditional read it stays a stale
    // false and Apply stays disabled forever.
    useNameValidityMock.mockReturnValue({ data: undefined, isLoading: false });

    const { rerender } = render(<ApplyPresetDialog {...defaultProps()} />);

    const warriorTile = screen
      .getAllByRole("radio")
      .find((el) => el.textContent?.includes("Warrior"))!;
    await userEvent.click(warriorTile);

    const nativeSelect = document.querySelector(
      'select[aria-hidden="true"]',
    ) as HTMLSelectElement | null;
    if (nativeSelect) {
      fireEvent.change(nativeSelect, { target: { value: "0" } });
    }

    const nameInput = screen.getByPlaceholderText("3-12 characters");
    await userEvent.type(nameInput, "Foobar");

    // Still in-flight: Apply is correctly disabled while validity is undefined.
    expect(screen.getByRole("button", { name: /apply/i })).toBeDisabled();

    // Name check resolves -> available. Re-render with the resolved value.
    useNameValidityMock.mockReturnValue({
      data: { valid: true },
      isLoading: false,
    });
    rerender(<ApplyPresetDialog {...defaultProps()} />);

    await waitFor(() => {
      expect(screen.getByRole("button", { name: /apply/i })).not.toBeDisabled();
    });
  });

  it("calls onOpenChange(false) when Cancel is clicked", () => {
    const onOpenChange = vi.fn();
    render(<ApplyPresetDialog {...defaultProps({ onOpenChange })} />);
    const cancelBtn = screen.getByRole("button", { name: /cancel/i });
    fireEvent.click(cancelBtn);
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("pre-selects initialPresetId when provided and it matches a saved preset", async () => {
    render(
      <ApplyPresetDialog
        {...defaultProps({ initialPresetId: "preset-2" })}
      />,
    );
    const mageTile = await screen.findByRole("radio", { name: /Mage/i });
    await waitFor(() =>
      expect(mageTile).toHaveAttribute("aria-checked", "true"),
    );
  });

  it("does not pre-select any preset when initialPresetId does not match a saved preset", async () => {
    useNameValidityMock.mockReturnValue({
      data: { valid: true },
      isLoading: false,
    });
    render(
      <ApplyPresetDialog
        {...defaultProps({ initialPresetId: "does-not-exist" })}
      />,
    );
    for (const tile of screen.getAllByRole("radio")) {
      expect(tile).toHaveAttribute("aria-checked", "false");
    }

    // Strengthen beyond the aria-checked check: a naive implementation could
    // set the form's `presetId` to the literal invalid string
    // "does-not-exist" (rather than "") and every tile would still correctly
    // render aria-checked="false" (none of them equals that literal string),
    // masking the bug. Fill in world + a valid name and confirm Apply stays
    // disabled because `presetId` is actually "" and fails the "select a
    // preset" schema validation — proving the resolved value, not just the
    // rendered tiles.
    const nativeSelect = document.querySelector(
      'select[aria-hidden="true"]',
    ) as HTMLSelectElement | null;
    if (nativeSelect) {
      fireEvent.change(nativeSelect, { target: { value: "0" } });
    }
    const nameInput = screen.getByPlaceholderText("3-12 characters");
    await userEvent.type(nameInput, "Foobar");

    await waitFor(() => {
      expect(screen.getByRole("button", { name: /apply/i })).toBeDisabled();
    });
  });

  it("does not clobber user edits when presets changes identity while the dialog stays open", async () => {
    useNameValidityMock.mockReturnValue({
      data: { valid: true },
      isLoading: false,
    });

    const initialConfig = {
      attributes: {
        characters: { presets: twoPresets },
        worlds: [
          { name: "Scania", flag: "" },
          { name: "Bera", flag: "" },
        ],
      },
    };
    useTenantConfigurationMock.mockReturnValue({
      data: initialConfig,
      isLoading: false,
    });

    const { rerender } = render(
      <ApplyPresetDialog
        {...defaultProps({ initialPresetId: "preset-1" })}
      />,
    );

    // Confirm the initial seed landed (Warrior pre-selected).
    const warriorTile = await screen.findByRole("radio", {
      name: /Warrior/i,
    });
    await waitFor(() =>
      expect(warriorTile).toHaveAttribute("aria-checked", "true"),
    );

    // User changes state: types a name and picks a DIFFERENT preset.
    const nameInput = screen.getByPlaceholderText("3-12 characters");
    await userEvent.type(nameInput, "MyHero");

    const mageTile = screen
      .getAllByRole("radio")
      .find((el) => el.textContent?.includes("Mage"))!;
    await userEvent.click(mageTile);
    await waitFor(() =>
      expect(mageTile).toHaveAttribute("aria-checked", "true"),
    );

    // Simulate the tenant-config query being invalidated/refetched with a
    // NEW reference but equal content (as happens when the sibling
    // character-presets editor saves while this dialog is open).
    const newConfigSameContent = {
      attributes: {
        characters: { presets: [...twoPresets] },
        worlds: [
          { name: "Scania", flag: "" },
          { name: "Bera", flag: "" },
        ],
      },
    };
    useTenantConfigurationMock.mockReturnValue({
      data: newConfigSameContent,
      isLoading: false,
    });
    rerender(
      <ApplyPresetDialog
        {...defaultProps({ initialPresetId: "preset-1" })}
      />,
    );

    // The user's typed name and manually-picked preset must survive — NOT
    // be reset back to initialPresetId="preset-1"/"".
    expect(screen.getByPlaceholderText("3-12 characters")).toHaveValue(
      "MyHero",
    );
    expect(mageTile).toHaveAttribute("aria-checked", "true");
    expect(warriorTile).toHaveAttribute("aria-checked", "false");
  });
});

describe("ApplyPresetDialog — world filtering", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useTenantConfigurationMock.mockReturnValue({
      data: {
        attributes: {
          characters: { presets: twoPresets },
          worlds: [
            { name: "Scania", flag: "" },
            { name: "Bera", flag: "" },
          ],
        },
      },
      isLoading: false,
    });
    useNameValidityMock.mockReturnValue({ data: undefined, isLoading: false });
    useCreateCharacterFromPresetMock.mockReturnValue({
      mutate,
      isPending: false,
    });
  });

  it("only lists worlds the tenant's channel-service configs cover", () => {
    useServicesMock.mockReturnValue({
      data: [channelServiceWithOnlyWorldZero],
      isLoading: false,
    });
    render(<ApplyPresetDialog {...defaultProps()} />);

    const nativeSelect = document.querySelector(
      'select[aria-hidden="true"]',
    ) as HTMLSelectElement;
    expect(nativeSelect).toBeTruthy();
    const optionTexts = Array.from(nativeSelect.options).map((o) => o.text);
    expect(optionTexts).toContain("Scania");
    expect(optionTexts).not.toContain("Bera");
  });

  it("disables the world Select with channel-service helper text when no channel service serves the tenant", () => {
    useServicesMock.mockReturnValue({
      data: [],
      isLoading: false,
    });
    render(<ApplyPresetDialog {...defaultProps()} />);
    const trigger = screen.getByLabelText(/^world$/i);
    expect(trigger).toBeDisabled();
    expect(screen.getByText(/channel service/i)).toBeInTheDocument();
  });

  it("ignores channel-service entries for other tenants", () => {
    useServicesMock.mockReturnValue({
      data: [
        {
          id: "svc-other",
          attributes: {
            type: "channel-service" as const,
            tasks: [],
            tenants: [
              {
                id: "other-tenant",
                ipAddress: "10.0.0.99",
                worlds: [
                  { id: 0, channels: [{ id: 0, port: 7575 }] },
                  { id: 1, channels: [{ id: 0, port: 7576 }] },
                ],
              },
            ],
          },
        },
      ],
      isLoading: false,
    });
    render(<ApplyPresetDialog {...defaultProps()} />);
    const trigger = screen.getByLabelText(/^world$/i);
    expect(trigger).toBeDisabled();
  });
});
