// services/atlas-ui/src/components/features/accounts/__tests__/CharactersPanel.test.tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";

vi.mock("@/components/features/characters/CharacterRenderer", () => ({
  CharacterRenderer: ({ character }: { character: { attributes: { name: string } } }) => (
    <div data-testid="renderer">{character.attributes.name}</div>
  ),
}));

const useCharactersMock = vi.fn();
vi.mock("@/lib/hooks/api/useCharacters", () => ({
  useCharacters: (...a: unknown[]) => useCharactersMock(...a),
  characterKeys: { lists: () => ["characters", "list"] },
}));

const useTenantConfigurationMock = vi.fn();
vi.mock("@/lib/hooks/api/useTenants", () => ({
  useTenantConfiguration: (...a: unknown[]) => useTenantConfigurationMock(...a),
}));

vi.mock("@/components/features/characters/ApplyPresetDialog", () => ({
  ApplyPresetDialog: ({ open }: { open: boolean }) =>
    open ? <div data-testid="apply-preset-dialog">apply</div> : null,
}));

import { CharactersPanel } from "../CharactersPanel";

const tenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as never;

const account = (slots: number) => ({
  id: "1",
  type: "accounts",
  attributes: {
    name: "Acct",
    characterSlots: slots,
    gender: 0,
    loggedIn: 0,
    lastLogin: 0,
    pinAttempts: 0,
    picAttempts: 0,
    tos: false,
  },
}) as never;

const character = (id: string, accountId: number, name = "Foo") => ({
  id,
  type: "characters",
  attributes: { accountId, worldId: 0, name },
}) as never;

const worldsConfig = [
  { name: "Scania", flag: "", serverMessage: "", eventMessage: "", whyAmIRecommended: "" },
];

function renderPanel(slots: number) {
  return render(
    <MemoryRouter>
      <CharactersPanel tenant={tenant} account={account(slots)} />
    </MemoryRouter>
  );
}

describe("CharactersPanel", () => {
  beforeEach(() => {
    useTenantConfigurationMock.mockReturnValue({
      data: {
        attributes: {
          characters: { presets: [{ id: "p1", attributes: { name: "Warrior" } }] },
          worlds: worldsConfig,
        },
      },
      isLoading: false,
    });
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("renders 3 filled + 2 empty tiles for 5 slots and 3 characters", () => {
    useCharactersMock.mockReturnValue({
      data: [
        character("10", 1, "Alpha"),
        character("11", 1, "Beta"),
        character("12", 1, "Gamma"),
        character("13", 99, "OtherTenantAccount"),
      ],
      isLoading: false,
      isFetching: false,
      error: null,
    });
    renderPanel(5);
    expect(screen.getAllByRole("link").length).toBe(3);
    expect(screen.getAllByRole("button", { name: /add character to slot/i }).length).toBe(2);
  });

  it("shows over-capacity hint and no empty tiles when characters exceed slots", () => {
    useCharactersMock.mockReturnValue({
      data: [
        character("10", 1, "A"),
        character("11", 1, "B"),
        character("12", 1, "C"),
      ],
      isLoading: false,
      isFetching: false,
      error: null,
    });
    renderPanel(2);
    expect(screen.getByText(/over capacity/i)).toBeInTheDocument();
    expect(screen.queryAllByRole("button", { name: /add character to slot/i })).toHaveLength(0);
  });

  it("opens the apply preset dialog when an empty tile is clicked", async () => {
    useCharactersMock.mockReturnValue({
      data: [],
      isLoading: false,
      isFetching: false,
      error: null,
    });
    renderPanel(2);
    expect(screen.queryByTestId("apply-preset-dialog")).toBeNull();
    await userEvent.click(
      screen.getAllByRole("button", { name: /add character to slot/i })[0]
    );
    expect(screen.getByTestId("apply-preset-dialog")).toBeInTheDocument();
  });

  it("disables empty tile clicks when no presets are configured", async () => {
    useTenantConfigurationMock.mockReturnValue({
      data: {
        attributes: { characters: { presets: [] }, worlds: worldsConfig },
      },
      isLoading: false,
    });
    useCharactersMock.mockReturnValue({
      data: [],
      isLoading: false,
      isFetching: false,
      error: null,
    });
    renderPanel(2);
    const emptyBtn = screen.getAllByRole("button", { name: /add character to slot/i })[0];
    expect(emptyBtn).toBeDisabled();
  });

  it("renders skeleton tiles equal to characterSlots while loading", () => {
    useCharactersMock.mockReturnValue({
      data: undefined,
      isLoading: true,
      isFetching: true,
      error: null,
    });
    const { container } = renderPanel(4);
    // The Skeleton component renders a div with role=status or class .animate-pulse.
    const skeletons = container.querySelectorAll(".animate-pulse");
    expect(skeletons.length).toBe(4);
  });
});
