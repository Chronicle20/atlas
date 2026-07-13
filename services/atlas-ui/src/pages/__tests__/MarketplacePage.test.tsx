import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

// A mutable holder so individual tests can flip the active tenant (e.g. the
// "no tenant selected" empty state) without re-declaring the module mock.
const { tenantHolder } = vi.hoisted(() => ({
  tenantHolder: { activeTenant: { id: "test-tenant" } as { id: string } | null },
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => tenantHolder,
}));

// The world dropdown is populated from the tenant configuration; a single world
// is enough for these tests (world selection is a Radix Select, not exercised
// here — jsdom lacks the pointer-capture APIs Radix needs).
vi.mock("@/lib/hooks/api/useTenants", () => ({
  useTenantConfiguration: () => ({ data: { attributes: { worlds: [{ name: "Scania" }] } } }),
}));

// ItemNameCell resolves an item name over its own query/tenant; stub it so a
// rendered listing row has no side effects.
vi.mock("@/components/item-name-cell", () => ({
  ItemNameCell: ({ itemId }: { itemId: string }) => <span>item-{itemId}</span>,
}));

vi.mock("@/services/api/mts-listings.service", () => ({
  mtsListingsService: {
    browse: vi.fn(async () => ({ listings: [], total: 0, lastPage: 1 })),
  },
}));

import { MarketplacePage } from "@/pages/MarketplacePage";
import { mtsListingsService } from "@/services/api/mts-listings.service";

function lastBrowseFilter() {
  const calls = vi.mocked(mtsListingsService.browse).mock.calls;
  return calls.at(-1)![1];
}

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <MarketplacePage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("MarketplacePage", () => {
  beforeEach(() => {
    tenantHolder.activeTenant = { id: "test-tenant" };
    vi.mocked(mtsListingsService.browse).mockReset();
    vi.mocked(mtsListingsService.browse).mockResolvedValue({ listings: [], total: 0, lastPage: 1 });
  });

  it("browses on mount, converting the 1-based UI page to the 0-based wire page", async () => {
    renderPage();
    await waitFor(() => expect(mtsListingsService.browse).toHaveBeenCalled());
    const filter = lastBrowseFilter();
    // UI starts on page 1 → wire page 0; pageSize is the fixed browse size.
    expect(filter.page).toBe(0);
    expect(filter.pageSize).toBe(16);
  });

  it("does not browse until a tenant is active, showing the empty state instead", async () => {
    tenantHolder.activeTenant = null;
    renderPage();
    expect(screen.getByText(/select a tenant to browse listings/i)).toBeInTheDocument();
    expect(mtsListingsService.browse).not.toHaveBeenCalled();
  });

  it("applies the typed item id filter and resets to the first page on Search", async () => {
    const user = userEvent.setup();
    renderPage();
    await waitFor(() => expect(mtsListingsService.browse).toHaveBeenCalled());

    await user.type(screen.getByPlaceholderText("e.g. 1302000"), "1302000");
    await user.click(screen.getByRole("button", { name: /search/i }));

    await waitFor(() => {
      const filter = lastBrowseFilter();
      expect(filter.itemId).toBe(1302000);
      expect(filter.page).toBe(0);
    });
  });

  it("advancing the pager refires the query on the next 0-based wire page", async () => {
    vi.mocked(mtsListingsService.browse).mockResolvedValue({
      listings: [
        {
          id: "1",
          attributes: {
            worldId: 0,
            sellerId: 1,
            sellerName: "Seller",
            saleType: "fixed",
            state: "active",
            templateId: 1302000,
            quantity: 1,
            strength: 0,
            dexterity: 0,
            intelligence: 0,
            luck: 0,
            hp: 0,
            mp: 0,
            weaponAttack: 0,
            magicAttack: 0,
            weaponDefense: 0,
            magicDefense: 0,
            accuracy: 0,
            avoidability: 0,
            hands: 0,
            speed: 0,
            jump: 0,
            slots: 0,
            level: 0,
            itemLevel: 0,
            itemExp: 0,
            ringId: 0,
            viciousCount: 0,
            flags: 0,
            listValue: 1000,
            commissionRate: 0.07,
            category: "",
            subCategory: "",
            currentBid: 0,
            highBidderId: 0,
            minIncrement: 0,
            createdAt: "",
            updatedAt: "",
          },
        },
      ],
      total: 32,
      lastPage: 2,
    });
    const user = userEvent.setup();
    renderPage();

    const next = await screen.findByRole("button", { name: /next page/i });
    await user.click(next);

    await waitFor(() => {
      // UI page 2 → wire page 1.
      expect(lastBrowseFilter().page).toBe(1);
    });
  });
});
