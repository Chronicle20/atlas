// services/atlas-ui/src/components/features/accounts/__tests__/CharactersPanel.test.tsx
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";

vi.mock("@/components/features/characters/CharacterRenderer", () => ({
  CharacterRenderer: ({
    character,
  }: {
    character: { attributes: { name: string } };
  }) => <div data-testid="renderer">{character.attributes.name}</div>,
}));

vi.mock("../FilledSlotTile", () => ({
  FilledSlotTile: ({
    character,
  }: {
    character: { id: string; attributes: { name: string } };
  }) => <a href={`/characters/${character.id}`}>{character.attributes.name}</a>,
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

// worldId -> slot count returned by the per-world hook. Tests set this
// before rendering so each world's WorldCharactersSection can be given a
// distinct slot count (proving the panel groups by world instead of
// summing a single account-wide total).
let slotsByWorld: Record<number, number> = {};
const useCharacterSlotsMock = vi.fn(
  (_tenant: unknown, _accountId: string, worldId: number) => ({
    data: { attributes: { worldId, slots: slotsByWorld[worldId] ?? 0 } },
    isLoading: false,
    error: null,
  }),
);
vi.mock("@/lib/hooks/api/useCharacterSlots", () => ({
  useCharacterSlots: (...a: [unknown, string, number]) =>
    useCharacterSlotsMock(...a),
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

const account = () =>
  ({
    id: "1",
    type: "accounts",
    attributes: {
      name: "Acct",
      gender: 0,
      loggedIn: 0,
      lastLogin: 0,
      pinAttempts: 0,
      picAttempts: 0,
      tos: false,
    },
  }) as never;

const character = (
  id: string,
  accountId: number,
  worldId: number,
  name = "Foo",
) =>
  ({
    id,
    type: "characters",
    attributes: { accountId, worldId, name },
  }) as never;

const oneWorldConfig = [
  {
    name: "Scania",
    flag: "",
    serverMessage: "",
    eventMessage: "",
    whyAmIRecommended: "",
  },
];

const twoWorldConfig = [
  ...oneWorldConfig,
  {
    name: "Bera",
    flag: "",
    serverMessage: "",
    eventMessage: "",
    whyAmIRecommended: "",
  },
];

function renderPanel() {
  return render(
    <MemoryRouter>
      <CharactersPanel tenant={tenant} account={account()} />
    </MemoryRouter>,
  );
}

describe("CharactersPanel", () => {
  beforeEach(() => {
    slotsByWorld = {};
    useTenantConfigurationMock.mockReturnValue({
      data: {
        attributes: {
          characters: {
            presets: [{ id: "p1", attributes: { name: "Warrior" } }],
          },
          worlds: oneWorldConfig,
        },
      },
      isLoading: false,
    });
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("renders 3 filled + 2 empty tiles for 5 slots and 3 characters in the one configured world", () => {
    slotsByWorld = { 0: 5 };
    useCharactersMock.mockReturnValue({
      data: [
        character("10", 1, 0, "Alpha"),
        character("11", 1, 0, "Beta"),
        character("12", 1, 0, "Gamma"),
        character("13", 99, 0, "OtherTenantAccount"),
      ],
      isLoading: false,
      isFetching: false,
      error: null,
    });
    renderPanel();
    expect(screen.getAllByRole("link").length).toBe(3);
    expect(
      screen.getAllByRole("button", { name: /add character to slot/i }).length,
    ).toBe(2);
  });

  it("shows over-capacity hint and no empty tiles when characters exceed slots", () => {
    slotsByWorld = { 0: 2 };
    useCharactersMock.mockReturnValue({
      data: [
        character("10", 1, 0, "A"),
        character("11", 1, 0, "B"),
        character("12", 1, 0, "C"),
      ],
      isLoading: false,
      isFetching: false,
      error: null,
    });
    renderPanel();
    expect(screen.getByText(/over capacity/i)).toBeInTheDocument();
    expect(
      screen.queryAllByRole("button", { name: /add character to slot/i }),
    ).toHaveLength(0);
  });

  it("opens the apply preset dialog when an empty tile is clicked", async () => {
    slotsByWorld = { 0: 2 };
    useCharactersMock.mockReturnValue({
      data: [],
      isLoading: false,
      isFetching: false,
      error: null,
    });
    renderPanel();
    expect(screen.queryByTestId("apply-preset-dialog")).toBeNull();
    const addButtons = screen.getAllByRole("button", {
      name: /add character to slot/i,
    });
    await userEvent.click(addButtons[0]!);
    expect(screen.getByTestId("apply-preset-dialog")).toBeInTheDocument();
  });

  it("disables empty tile clicks when no presets are configured", async () => {
    slotsByWorld = { 0: 2 };
    useTenantConfigurationMock.mockReturnValue({
      data: {
        attributes: { characters: { presets: [] }, worlds: oneWorldConfig },
      },
      isLoading: false,
    });
    useCharactersMock.mockReturnValue({
      data: [],
      isLoading: false,
      isFetching: false,
      error: null,
    });
    renderPanel();
    const emptyBtn = screen.getAllByRole("button", {
      name: /add character to slot/i,
    })[0];
    expect(emptyBtn).toBeDisabled();
  });

  it("renders loading skeleton tiles while characters are loading", () => {
    slotsByWorld = { 0: 4 };
    useCharactersMock.mockReturnValue({
      data: undefined,
      isLoading: true,
      isFetching: true,
      error: null,
    });
    const { container } = renderPanel();
    const skeletons = container.querySelectorAll(".animate-pulse");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("groups characters by world: each world shows only its own slot count and its own characters, not a summed total", () => {
    useTenantConfigurationMock.mockReturnValue({
      data: {
        attributes: {
          characters: {
            presets: [{ id: "p1", attributes: { name: "Warrior" } }],
          },
          worlds: twoWorldConfig,
        },
      },
      isLoading: false,
    });
    // Scania (world 0): 3 slots, 1 character -> 2 empty tiles.
    // Bera (world 1): 2 slots, 2 characters -> 0 empty tiles, at capacity
    // (not over capacity, and NOT summed with Scania's count).
    slotsByWorld = { 0: 3, 1: 2 };
    useCharactersMock.mockReturnValue({
      data: [
        character("20", 1, 0, "ScaniaHero"),
        character("21", 1, 1, "BeraHero1"),
        character("22", 1, 1, "BeraHero2"),
      ],
      isLoading: false,
      isFetching: false,
      error: null,
    });
    renderPanel();

    const scaniaSection = screen.getByText("Scania").closest("section")!;
    const beraSection = screen.getByText("Bera").closest("section")!;

    expect(within(scaniaSection).getByText("ScaniaHero")).toBeInTheDocument();
    expect(
      within(scaniaSection).getAllByRole("button", {
        name: /add character to slot/i,
      }),
    ).toHaveLength(2);

    expect(within(beraSection).getByText("BeraHero1")).toBeInTheDocument();
    expect(within(beraSection).getByText("BeraHero2")).toBeInTheDocument();
    expect(
      within(beraSection).queryAllByRole("button", {
        name: /add character to slot/i,
      }),
    ).toHaveLength(0);
    // Bera is exactly at capacity (2 characters, 2 slots) — must not read
    // as over-capacity from a summed 5-slot/3-character total across both
    // worlds.
    expect(
      within(beraSection).queryByText(/over capacity/i),
    ).not.toBeInTheDocument();

    // Neither world's slot count is a sum of the two (3+2=5): each section
    // reports its own count independently.
    expect(within(scaniaSection).getByText(/1\/3 slots/)).toBeInTheDocument();
    expect(within(beraSection).getByText(/2\/2 slots/)).toBeInTheDocument();
  });
});
