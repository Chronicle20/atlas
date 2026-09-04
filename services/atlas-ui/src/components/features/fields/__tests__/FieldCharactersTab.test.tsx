import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi, beforeEach } from "vitest";

import { FieldCharactersTab } from "@/components/features/fields/FieldCharactersTab";
import type { Character } from "@/types/models/character";

const mockTenant = {
  id: "tenant-1",
  attributes: {
    name: "Test Tenant",
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
  },
};

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: mockTenant,
    tenants: [mockTenant],
    loading: false,
    setActiveTenant: vi.fn(),
    refreshTenants: vi.fn(),
    refreshAndSelectTenant: vi.fn(),
    fetchTenantConfiguration: vi.fn(),
  }),
}));

const useCharacterMock = vi.fn();
vi.mock("@/lib/hooks/api/useCharacters", () => ({
  useCharacter: (...args: unknown[]) => useCharacterMock(...args),
}));

const useJobNameLookupMock = vi.fn();
vi.mock("@/lib/hooks/api/useJobGraph", () => ({
  useJobNameLookup: () => useJobNameLookupMock(),
}));

function makeCharacter(
  id: string,
  attrs: Partial<Character["attributes"]>,
): Character {
  return {
    id,
    attributes: {
      accountId: 1,
      worldId: 0,
      name: "Unknown",
      level: 1,
      experience: 0,
      gachaponExperience: 0,
      strength: 0,
      dexterity: 0,
      intelligence: 0,
      luck: 0,
      hp: 0,
      maxHp: 0,
      mp: 0,
      maxMp: 0,
      meso: 0,
      hpMpUsed: 0,
      jobId: 0,
      skinColor: 0,
      gender: 0,
      fame: 0,
      hair: 0,
      x: 0,
      y: 0,
      stance: 0,
      ...attrs,
    } as Character["attributes"],
  };
}

function queryResult(overrides: Record<string, unknown>) {
  return {
    data: undefined,
    isLoading: false,
    isError: false,
    error: null,
    ...overrides,
  };
}

function renderTab(characterIds: string[]) {
  return render(
    <MemoryRouter>
      <FieldCharactersTab characterIds={characterIds} />
    </MemoryRouter>,
  );
}

describe("FieldCharactersTab", () => {
  beforeEach(() => {
    useCharacterMock.mockReset();
    useJobNameLookupMock.mockReset();
    useJobNameLookupMock.mockReturnValue((id: number) => `Job ${id}`);
  });

  it("renders one row per character id", () => {
    useCharacterMock.mockImplementation((_tenant: unknown, id: string) =>
      queryResult({ data: makeCharacter(id, { name: `Char ${id}` }) }),
    );

    renderTab(["100", "200"]);

    expect(screen.getAllByRole("row")).toHaveLength(3); // header + 2 rows
  });

  it("renders the settled columns", () => {
    useJobNameLookupMock.mockReturnValue((id: number) =>
      id === 110 ? "Beginner" : `Job ${id}`,
    );
    useCharacterMock.mockImplementation((_tenant: unknown, id: string) =>
      queryResult({
        data: makeCharacter(id, {
          name: "Bob",
          level: 42,
          jobId: 110,
          x: 640,
          y: 120,
        }),
      }),
    );

    renderTab(["100"]);

    expect(screen.getByText("Bob")).toBeInTheDocument();
    expect(screen.getByText("42")).toBeInTheDocument();
    expect(screen.getByText("Beginner")).toBeInTheDocument();
    expect(screen.getByText(/640/)).toBeInTheDocument();
    expect(screen.getByText(/120/)).toBeInTheDocument();
  });

  it("has no Character ID or State column", () => {
    useCharacterMock.mockImplementation((_tenant: unknown, id: string) =>
      queryResult({ data: makeCharacter(id, { name: "Bob" }) }),
    );

    renderTab(["100"]);

    expect(
      screen.queryByRole("columnheader", { name: "Character ID" }),
    ).not.toBeInTheDocument();
    expect(screen.queryByText("State")).not.toBeInTheDocument();
  });

  it("the Name cell exposes the copyable id via a tooltip (bug-fields-ui item 10)", async () => {
    useCharacterMock.mockImplementation((_tenant: unknown, id: string) =>
      queryResult({ data: makeCharacter(id, { name: "Bob" }) }),
    );
    const user = userEvent.setup();

    renderTab(["100"]);

    expect(screen.queryByText("100")).not.toBeInTheDocument();
    await user.hover(screen.getByRole("link", { name: "Bob" }));
    expect(await screen.findByText("100")).toBeInTheDocument();
  });

  it("pending enrichment shows the raw id", () => {
    useCharacterMock.mockImplementation(() => queryResult({ isLoading: true }));

    renderTab(["100"]);

    expect(screen.getAllByText("100").length).toBeGreaterThan(0);
    expect(screen.getAllByRole("row")).toHaveLength(2); // header + 1 row, not unmounted
  });

  it("failed enrichment shows the raw id", () => {
    useCharacterMock.mockImplementation(() =>
      queryResult({ isError: true, error: new Error("boom") }),
    );

    renderTab(["100"]);

    expect(screen.getAllByText("100").length).toBeGreaterThan(0);
    expect(screen.getAllByRole("row")).toHaveLength(2);
  });

  it("one failure does not block others", () => {
    useCharacterMock.mockImplementation((_tenant: unknown, id: string) => {
      if (id === "100") {
        return queryResult({ isError: true, error: new Error("boom") });
      }
      return queryResult({ data: makeCharacter(id, { name: "Alice" }) });
    });

    renderTab(["100", "200"]);

    expect(screen.getAllByRole("row")).toHaveLength(3);
    expect(screen.getByText("Alice")).toBeInTheDocument();
  });

  it("name links to Character Detail", () => {
    useCharacterMock.mockImplementation((_tenant: unknown, id: string) =>
      queryResult({ data: makeCharacter(id, { name: "Bob" }) }),
    );

    renderTab(["100"]);

    expect(screen.getByRole("link", { name: "Bob" })).toHaveAttribute(
      "href",
      "/characters/100",
    );
  });

  it("empty tab copy", () => {
    renderTab([]);

    expect(
      screen.getByText("No characters are currently in this field."),
    ).toBeInTheDocument();
  });
});
