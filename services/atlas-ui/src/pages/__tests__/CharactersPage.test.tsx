import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { CharactersPage } from "@/pages/CharactersPage";
import { charactersService } from "@/services/api/characters.service";
import { accountsService } from "@/services/api/accounts.service";
import type { Character } from "@/types/models/character";

vi.mock("@/services/api/characters.service", () => ({
  charactersService: {
    getPage: vi.fn(),
    getAll: vi.fn(async () => []),
    deleteCharacter: vi.fn(),
    update: vi.fn(),
    checkNameValidity: vi.fn(),
  },
}));

vi.mock("@/services/api/accounts.service", () => ({
  accountsService: {
    getAllAccounts: vi.fn(async () => []),
  },
}));

vi.mock("@/lib/hooks/api/useTenants", () => ({
  useTenantConfiguration: () => ({ data: null, isLoading: false, error: null, isFetching: false, refetch: vi.fn() }),
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: { id: "test-tenant", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } },
  }),
}));

function makeCharacter(id: string, name: string): Character {
  return {
    id,
    attributes: {
      accountId: 1,
      worldId: 0,
      name,
      level: 1,
      experience: 0,
      gachaponExperience: 0,
      strength: 4,
      dexterity: 4,
      intelligence: 4,
      luck: 4,
      hp: 50,
      maxHp: 50,
      mp: 5,
      maxMp: 5,
      meso: 0,
      hpMpUsed: 0,
      jobId: 0,
      skinColor: 0,
      gender: 0,
      fame: 0,
      hair: 30000,
      face: 20000,
      ap: 0,
      sp: "0",
      spawnPoint: 0,
      gm: 0,
      x: 0,
      y: 0,
      stance: 0,
    },
  };
}

function renderAt(initial: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initial]}>
        <Routes>
          <Route path="/characters" element={<CharactersPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("CharactersPage", () => {
  beforeEach(() => {
    vi.mocked(charactersService.getPage).mockReset();
    vi.mocked(accountsService.getAllAccounts).mockReset().mockResolvedValue([]);
  });

  it("requests page 1 at the default page size on mount", async () => {
    vi.mocked(charactersService.getPage).mockResolvedValue({
      data: [makeCharacter("1", "Hero")],
      meta: { total: 1, page: { number: 1, size: 50, last: 1 } },
    });

    renderAt("/characters");

    await waitFor(() => {
      expect(charactersService.getPage).toHaveBeenCalledWith(
        { number: 1, size: 50 },
        expect.anything(),
      );
    });
  });

  it("renders the pager off meta.total / meta.page.last and requests the next page on click", async () => {
    vi.mocked(charactersService.getPage).mockResolvedValue({
      data: [makeCharacter("1", "Hero")],
      meta: { total: 120, page: { number: 1, size: 50, last: 3 } },
    });

    renderAt("/characters");

    await screen.findByText(/Page 1 of 3/i);
    expect(screen.getByText(/120 results/i)).toBeInTheDocument();

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: /next page/i }));

    await waitFor(() => {
      const lastCall = vi.mocked(charactersService.getPage).mock.calls.at(-1)!;
      expect(lastCall[0]).toEqual({ number: 2, size: 50 });
    });
  });

  it("hydrates the page number from ?page= in the URL", async () => {
    vi.mocked(charactersService.getPage).mockResolvedValue({
      data: [makeCharacter("1", "Hero")],
      meta: { total: 120, page: { number: 3, size: 50, last: 3 } },
    });

    renderAt("/characters?page=3");

    await waitFor(() => {
      expect(charactersService.getPage).toHaveBeenCalledWith(
        { number: 3, size: 50 },
        expect.anything(),
      );
    });
  });
});
