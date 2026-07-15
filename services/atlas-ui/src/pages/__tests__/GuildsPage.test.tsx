import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { GuildsPage } from "@/pages/GuildsPage";
import { guildsService } from "@/services/api/guilds.service";
import type { Guild, GuildMember } from "@/types/models/guild";

vi.mock("@/services/api/guilds.service", () => ({
  guildsService: {
    getPage: vi.fn(),
    search: vi.fn(),
    getById: vi.fn(),
  },
}));

vi.mock("@/lib/hooks/api/useTenants", () => ({
  useTenantConfiguration: () => ({ data: null, isLoading: false, error: null, isFetching: false, refetch: vi.fn() }),
}));

vi.mock("@/lib/hooks/api/useCharacters", () => ({
  useCharacters: () => ({ data: [], isLoading: false, error: null, isFetching: false, refetch: vi.fn() }),
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: { id: "test-tenant", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } },
  }),
}));

function makeGuild(id: string, name: string): Guild {
  const member: GuildMember = {
    characterId: 1,
    name: "Leader",
    jobId: 100,
    level: 50,
    title: 0,
    online: true,
    allianceTitle: 0,
  };
  return {
    id,
    attributes: {
      worldId: 0,
      name,
      notice: "",
      points: 100,
      capacity: 50,
      logo: 0,
      logoColor: 0,
      logoBackground: 0,
      logoBackgroundColor: 0,
      leaderId: 1,
      members: [member],
      titles: [],
    },
  };
}

function renderAt(initial: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initial]}>
        <Routes>
          <Route path="/guilds" element={<GuildsPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("GuildsPage", () => {
  beforeEach(() => {
    vi.mocked(guildsService.getPage).mockReset();
    vi.mocked(guildsService.search).mockReset();
  });

  it("requests page 1 at the default page size on mount (browse, no search term)", async () => {
    vi.mocked(guildsService.getPage).mockResolvedValue({
      data: [makeGuild("1", "Alpha Guild")],
      meta: { total: 1, page: { number: 1, size: 50, last: 1 } },
    });

    renderAt("/guilds");

    await waitFor(() => {
      expect(guildsService.getPage).toHaveBeenCalledWith(
        { number: 1, size: 50 },
        expect.anything(),
      );
    });
    expect(guildsService.search).not.toHaveBeenCalled();
  });

  it("renders the pager off meta.total / meta.page.last and requests the next page on click", async () => {
    vi.mocked(guildsService.getPage).mockResolvedValue({
      data: [makeGuild("1", "Alpha Guild")],
      meta: { total: 120, page: { number: 1, size: 50, last: 3 } },
    });

    renderAt("/guilds");

    await screen.findByText(/Page 1 of 3/i);
    expect(screen.getByText(/120 results/i)).toBeInTheDocument();

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: /next page/i }));

    await waitFor(() => {
      const lastCall = vi.mocked(guildsService.getPage).mock.calls.at(-1)!;
      expect(lastCall[0]).toEqual({ number: 2, size: 50 });
    });
  });

  it("hydrates the page number from ?page= in the URL", async () => {
    vi.mocked(guildsService.getPage).mockResolvedValue({
      data: [makeGuild("1", "Alpha Guild")],
      meta: { total: 120, page: { number: 3, size: 50, last: 3 } },
    });

    renderAt("/guilds?page=3");

    await waitFor(() => {
      expect(guildsService.getPage).toHaveBeenCalledWith(
        { number: 3, size: 50 },
        expect.anything(),
      );
    });
  });

  it("typing in the search box drives a server-side filter[name] request, not a client-side filter over a full fetch", async () => {
    vi.mocked(guildsService.getPage).mockResolvedValue({
      data: [makeGuild("1", "Alpha Guild")],
      meta: { total: 1, page: { number: 1, size: 50, last: 1 } },
    });
    vi.mocked(guildsService.search).mockResolvedValue({
      data: [makeGuild("2", "Bravo Guild")],
      meta: { total: 1, page: { number: 1, size: 50, last: 1 } },
    });

    renderAt("/guilds");

    const searchInput = await screen.findByLabelText(/search guilds/i);

    const user = userEvent.setup();
    await user.type(searchInput, "Bravo");

    await waitFor(() => {
      expect(guildsService.search).toHaveBeenCalledWith(
        "Bravo",
        { number: 1, size: 50 },
        expect.anything(),
      );
    });

    await screen.findByText("Bravo Guild");
  });

  it("hydrates the search term from ?q= in the URL and calls search directly", async () => {
    vi.mocked(guildsService.search).mockResolvedValue({
      data: [makeGuild("2", "Bravo Guild")],
      meta: { total: 1, page: { number: 1, size: 50, last: 1 } },
    });

    renderAt("/guilds?q=Bravo");

    await waitFor(() => {
      expect(guildsService.search).toHaveBeenCalledWith(
        "Bravo",
        { number: 1, size: 50 },
        expect.anything(),
      );
    });
    expect(guildsService.getPage).not.toHaveBeenCalled();
  });
});
