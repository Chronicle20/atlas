import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MerchantsPage } from "@/pages/MerchantsPage";
import { merchantsService } from "@/services/api/merchants.service";
import type { MerchantShop } from "@/types/models/merchant";

vi.mock("@/services/api/merchants.service", () => ({
  merchantsService: {
    getShopsPage: vi.fn(),
    getAllShops: vi.fn(async () => []),
    searchListings: vi.fn(async () => []),
    getShopListings: vi.fn(async () => []),
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

function makeShop(id: string): MerchantShop {
  return {
    id,
    attributes: {
      characterId: 1,
      shopType: 1,
      state: 2,
      title: `Shop ${id}`,
      worldId: 0,
      channelId: 0,
      mapId: 100000000,
      instanceId: "",
      x: 0,
      y: 0,
      permitItemId: 0,
      closeReason: 0,
      mesoBalance: 0,
      listingCount: 0,
    },
  };
}

function renderAt(initial: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initial]}>
        <Routes>
          <Route path="/merchants" element={<MerchantsPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("MerchantsPage shops tab", () => {
  beforeEach(() => {
    vi.mocked(merchantsService.getShopsPage).mockReset();
  });

  it("requests page 1 of shops at the default page size on mount", async () => {
    vi.mocked(merchantsService.getShopsPage).mockResolvedValue({
      data: [makeShop("1")],
      meta: { total: 1, page: { number: 1, size: 50, last: 1 } },
    });

    renderAt("/merchants");

    await waitFor(() => {
      expect(merchantsService.getShopsPage).toHaveBeenCalledWith({ number: 1, size: 50 });
    });
  });

  it("renders the pager off meta and requests the next page on click", async () => {
    vi.mocked(merchantsService.getShopsPage).mockResolvedValue({
      data: [makeShop("1")],
      meta: { total: 80, page: { number: 1, size: 50, last: 2 } },
    });

    renderAt("/merchants");

    await screen.findByText(/Page 1 of 2/i);

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: /next page/i }));

    await waitFor(() => {
      const lastCall = vi.mocked(merchantsService.getShopsPage).mock.calls.at(-1)!;
      expect(lastCall[0]).toEqual({ number: 2, size: 50 });
    });
  });
});
