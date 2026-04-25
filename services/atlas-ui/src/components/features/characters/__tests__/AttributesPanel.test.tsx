import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AttributesPanel } from "../AttributesPanel";

const fakeTenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as never;

const mockTenantConfig = {
  attributes: { worlds: [{ name: "Bera" }, { name: "Scania" }] },
} as never;

const useCharacterGuildMock = vi.fn();
vi.mock("@/lib/hooks/api/useCharacterGuild", () => ({
  useCharacterGuild: (...a: unknown[]) => useCharacterGuildMock(...a),
}));

function renderPanel(character: never) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <AttributesPanel
          character={character}
          tenantConfig={mockTenantConfig}
          tenant={fakeTenant}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

const baseCharacter = (overrides: Record<string, unknown> = {}) =>
  ({
    id: "42",
    type: "characters",
    attributes: {
      name: "Test",
      level: 70,
      experience: 12345,
      mapId: "100000000",
      worldId: 0,
      gender: 0,
      jobId: 112,
      strength: 4,
      dexterity: 25,
      intelligence: 4,
      luck: 4,
      hp: 1500,
      maxHp: 2000,
      mp: 100,
      maxMp: 200,
      meso: 1234567,
      fame: 50,
      gm: 0,
      ...overrides,
    },
  }) as never;

describe("AttributesPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useCharacterGuildMock.mockReturnValue({ guild: null, isLoading: false, error: null });
  });

  it("labels gender 0 as Male and 1 as Female", () => {
    const { rerender } = renderPanel(baseCharacter({ gender: 0 }));
    expect(screen.getByText("Male")).toBeInTheDocument();
    rerender(
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <MemoryRouter>
          <AttributesPanel
            character={baseCharacter({ gender: 1 })}
            tenantConfig={mockTenantConfig}
            tenant={fakeTenant}
          />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    expect(screen.getByText("Female")).toBeInTheDocument();
  });

  it("renders the world badge using tenantConfig.worlds", () => {
    renderPanel(baseCharacter());
    expect(screen.getByText("Bera")).toBeInTheDocument();
  });

  it("formats mesos with thousands separators", () => {
    renderPanel(baseCharacter({ meso: 1234567 }));
    expect(screen.getByText(/1,234,567/)).toBeInTheDocument();
  });

  it("renders STR/DEX/INT/LUK on a single row", () => {
    const { container } = renderPanel(baseCharacter());
    expect(container.querySelector(".grid-cols-4")).toBeTruthy();
  });

  it('renders "None" for guild when not in a guild', () => {
    useCharacterGuildMock.mockReturnValue({ guild: null, isLoading: false, error: null });
    renderPanel(baseCharacter());
    expect(screen.getByText(/^None$/i)).toBeInTheDocument();
  });

  it("renders guild name when present", () => {
    useCharacterGuildMock.mockReturnValue({
      guild: { id: "5", attributes: { name: "Heroes" } },
      isLoading: false,
      error: null,
    });
    renderPanel(baseCharacter());
    expect(screen.getByText("Heroes")).toBeInTheDocument();
  });

  it('renders "Unknown" when guild fetch errors', () => {
    useCharacterGuildMock.mockReturnValue({
      guild: null,
      isLoading: false,
      error: new Error("x"),
    });
    renderPanel(baseCharacter());
    expect(screen.getByText(/Unknown/i)).toBeInTheDocument();
  });

  it("shows alliance placeholder", () => {
    renderPanel(baseCharacter());
    expect(screen.getByText(/Not available/i)).toBeInTheDocument();
  });

  it("renders the world badge as a focusable tooltip trigger for copyable id reveal", () => {
    renderPanel(baseCharacter({ worldId: 1 }));
    const badge = screen.getByText("Scania");
    expect(badge).toHaveAttribute("tabIndex", "0");
    expect(badge.className).toMatch(/cursor-help/);
    // Tooltip content (the raw worldId) renders into a Radix portal on
    // hover/focus; jsdom doesn't drive that reliably, so we verify the
    // trigger is wired and rely on Radix coverage for the open behavior.
  });
});
