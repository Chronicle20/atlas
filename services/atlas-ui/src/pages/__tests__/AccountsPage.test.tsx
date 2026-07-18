import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AccountsPage } from "@/pages/AccountsPage";
import { accountsService } from "@/services/api/accounts.service";
import type { Account } from "@/types/models/account";

vi.mock("@/services/api/accounts.service", () => ({
  accountsService: {
    getAccountsPage: vi.fn(),
    getAllAccounts: vi.fn(async () => []),
  },
}));

vi.mock("@/services/api/bans.service", () => ({
  bansService: {
    checkBan: vi.fn(async () => ({ id: "0", attributes: { banned: false } })),
    getBansByType: vi.fn(async () => []),
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

function makeAccount(id: string, name: string): Account {
  return {
    id,
    attributes: {
      name,
      pin: "",
      pic: "",
      pinAttempts: 0,
      picAttempts: 0,
      loggedIn: 0,
      lastLogin: 0,
      gender: 0,
      tos: true,
      language: "en",
      country: "US",
      characterSlots: 6,
    },
  };
}

function renderAt(initial: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initial]}>
        <Routes>
          <Route path="/accounts" element={<AccountsPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AccountsPage", () => {
  beforeEach(() => {
    vi.mocked(accountsService.getAccountsPage).mockReset();
  });

  it("requests page 1 at the default page size on mount", async () => {
    vi.mocked(accountsService.getAccountsPage).mockResolvedValue({
      data: [makeAccount("1", "alpha")],
      meta: { total: 1, page: { number: 1, size: 50, last: 1 } },
    });

    renderAt("/accounts");

    await waitFor(() => {
      expect(accountsService.getAccountsPage).toHaveBeenCalledWith(
        { number: 1, size: 50 },
        expect.anything(),
      );
    });
  });

  it("renders the pager off meta and requests the next page on click", async () => {
    vi.mocked(accountsService.getAccountsPage).mockResolvedValue({
      data: [makeAccount("1", "alpha")],
      meta: { total: 60, page: { number: 1, size: 50, last: 2 } },
    });

    renderAt("/accounts");

    await screen.findByText(/Page 1 of 2/i);

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: /next page/i }));

    await waitFor(() => {
      const lastCall = vi
        .mocked(accountsService.getAccountsPage)
        .mock.calls.at(-1)!;
      expect(lastCall[0]).toEqual({ number: 2, size: 50 });
    });
  });
});
