import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BansPage } from "@/pages/BansPage";
import { bansService } from "@/services/api/bans.service";
import { BanType } from "@/types/models/ban";
import type { Ban } from "@/types/models/ban";

vi.mock("@/services/api/bans.service", () => ({
  bansService: {
    getBansPage: vi.fn(),
    getAllBans: vi.fn(async () => []),
  },
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "test-tenant",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

function makeBan(id: string): Ban {
  return {
    id,
    attributes: {
      banType: BanType.IP,
      value: "1.2.3.4",
      reason: "test",
      reasonCode: 0,
      permanent: true,
      expiresAt: "0001-01-01T00:00:00Z",
      issuedBy: "admin",
    },
  };
}

function renderAt(initial: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initial]}>
        <Routes>
          <Route path="/bans" element={<BansPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("BansPage", () => {
  beforeEach(() => {
    vi.mocked(bansService.getBansPage).mockReset();
  });

  it("requests page 1 at the default page size with no type filter on mount", async () => {
    vi.mocked(bansService.getBansPage).mockResolvedValue({
      data: [makeBan("1")],
      meta: { total: 1, page: { number: 1, size: 50, last: 1 } },
    });

    renderAt("/bans");

    await waitFor(() => {
      expect(bansService.getBansPage).toHaveBeenCalledWith(
        { number: 1, size: 50 },
        undefined,
      );
    });
  });

  it("renders the pager off meta and requests the next page on click", async () => {
    vi.mocked(bansService.getBansPage).mockResolvedValue({
      data: [makeBan("1")],
      meta: { total: 75, page: { number: 1, size: 50, last: 2 } },
    });

    renderAt("/bans");

    await screen.findByText(/Page 1 of 2/i);

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: /next page/i }));

    await waitFor(() => {
      const lastCall = vi.mocked(bansService.getBansPage).mock.calls.at(-1)!;
      expect(lastCall[0]).toEqual({ number: 2, size: 50 });
    });
  });

  it("hydrates the page number from ?page= in the URL", async () => {
    vi.mocked(bansService.getBansPage).mockResolvedValue({
      data: [makeBan("1")],
      meta: { total: 75, page: { number: 2, size: 50, last: 2 } },
    });

    renderAt("/bans?page=2");

    await waitFor(() => {
      expect(bansService.getBansPage).toHaveBeenCalledWith(
        { number: 2, size: 50 },
        undefined,
      );
    });
  });
});
