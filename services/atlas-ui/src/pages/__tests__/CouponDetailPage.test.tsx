import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { CouponDetailPage } from "@/pages/CouponDetailPage";
import { couponsService } from "@/services/api/coupons.service";
import type { Coupon, CouponRedemption } from "@/services/api/coupons.service";
import { accountsService } from "@/services/api/accounts.service";
import { charactersService } from "@/services/api/characters.service";
import { commoditiesService } from "@/services/api/commodities.service";
import { itemStringsService } from "@/services/api/item-strings.service";

vi.mock("@/services/api/coupons.service", () => {
  class CouponConflictError extends Error {
    readonly status = 409;
    constructor(message: string) {
      super(message);
      this.name = "CouponConflictError";
    }
  }
  return {
    CouponConflictError,
    couponsService: {
      list: vi.fn(),
      getOne: vi.fn(),
      create: vi.fn(),
      update: vi.fn(),
      remove: vi.fn(),
      listRedemptions: vi.fn(),
      listBatches: vi.fn(),
      getBatch: vi.fn(),
      generateBatch: vi.fn(),
    },
  };
});

// The redemption rows carry numeric ids; the page resolves them to names.
vi.mock("@/services/api/accounts.service", () => ({
  accountsService: { getAccountById: vi.fn() },
}));

vi.mock("@/services/api/characters.service", () => ({
  charactersService: { getById: vi.fn() },
}));

// A CASH_ITEM reward names a serial; the page resolves it to the item name
// through the commodity it sells.
vi.mock("@/services/api/commodities.service", () => ({
  commoditiesService: { getBySerialNumber: vi.fn(), drainAll: vi.fn() },
}));

vi.mock("@/services/api/item-strings.service", () => ({
  itemStringsService: { getItemString: vi.fn() },
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "test-tenant",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

const coupon: Coupon = {
  id: "c1",
  attributes: {
    code: "SUMMER2026",
    description: "summer promo",
    active: true,
    redemptionCount: 2,
    maxUses: 10,
    startsAt: "2026-06-01T00:00:00Z",
    expiresAt: "2026-07-01T00:00:00Z",
    rewards: [{ type: "CURRENCY", currency: 1, amount: 1000 }],
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
  },
};

function makeRedemption(
  id: string,
  accountId: number,
  characterId: number,
): CouponRedemption {
  return {
    id,
    attributes: {
      couponId: "c1",
      accountId,
      characterId,
      transactionId: `tx-${id}`,
      rewardsGranted: [{ type: "CURRENCY", currency: 1, amount: 1000 }],
      redeemedAt: "2026-06-05T12:00:00Z",
    },
  };
}

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={["/coupons/c1"]}>
        <Routes>
          <Route path="/coupons/:couponId" element={<CouponDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("CouponDetailPage", () => {
  beforeEach(() => {
    vi.mocked(couponsService.getOne).mockReset();
    vi.mocked(couponsService.listRedemptions).mockReset();
    vi.mocked(accountsService.getAccountById).mockReset();
    vi.mocked(charactersService.getById).mockReset();
    vi.mocked(accountsService.getAccountById).mockRejectedValue(
      new Error("not resolved"),
    );
    vi.mocked(charactersService.getById).mockRejectedValue(
      new Error("not resolved"),
    );
    vi.mocked(commoditiesService.getBySerialNumber).mockReset();
    vi.mocked(itemStringsService.getItemString).mockReset();
    vi.mocked(commoditiesService.getBySerialNumber).mockRejectedValue(
      new Error("no commodity"),
    );
    vi.mocked(itemStringsService.getItemString).mockRejectedValue(
      new Error("no name"),
    );
  });

  it("renders the redemption history for that coupon", async () => {
    vi.mocked(couponsService.getOne).mockResolvedValue(coupon);
    vi.mocked(couponsService.listRedemptions).mockResolvedValue({
      data: [makeRedemption("r1", 42, 4200), makeRedemption("r2", 43, 4300)],
      meta: { total: 2, page: { number: 1, size: 50, last: 1 } },
    });

    renderPage();

    expect(await screen.findByText("SUMMER2026")).toBeInTheDocument();
    expect(screen.getByText("2 / 10")).toBeInTheDocument();

    // History is fetched per-code, never as a global list.
    expect(couponsService.listRedemptions).toHaveBeenCalledWith(
      { couponId: "c1" },
      { number: 1, size: 50 },
      undefined,
    );

    expect(await screen.findByText("tx-r1")).toBeInTheDocument();
    expect(screen.getByText("tx-r2")).toBeInTheDocument();
    // Names could not be resolved here, so the rows degrade to the raw ids.
    expect(screen.getByText("42")).toBeInTheDocument();
    expect(screen.getByText("4300")).toBeInTheDocument();

    // Navigation back to the list is the breadcrumb bar's job, not a button.
    expect(
      screen.queryByRole("link", { name: /back to coupons/i }),
    ).not.toBeInTheDocument();
  });

  it("resolves the redemption account and character ids to names", async () => {
    vi.mocked(couponsService.getOne).mockResolvedValue(coupon);
    vi.mocked(couponsService.listRedemptions).mockResolvedValue({
      data: [makeRedemption("r1", 42, 4200)],
      meta: { total: 1, page: { number: 1, size: 50, last: 1 } },
    });
    vi.mocked(accountsService.getAccountById).mockResolvedValue({
      id: "42",
      attributes: { name: "operator" },
    } as Awaited<ReturnType<typeof accountsService.getAccountById>>);
    vi.mocked(charactersService.getById).mockResolvedValue({
      id: "4200",
      attributes: { name: "Sharpie" },
    } as Awaited<ReturnType<typeof charactersService.getById>>);

    renderPage();

    expect(await screen.findByText("operator")).toBeInTheDocument();
    expect(await screen.findByText("Sharpie")).toBeInTheDocument();
    // The name replaces the id rather than sitting beside it.
    expect(screen.queryByText("42")).not.toBeInTheDocument();
    expect(screen.queryByText("4200")).not.toBeInTheDocument();
  });

  it("names the cash item a reward grants, in both the card and the history", async () => {
    const cashReward = {
      type: "CASH_ITEM" as const,
      serialNumber: 20000036,
      quantity: 2,
    };
    vi.mocked(couponsService.getOne).mockResolvedValue({
      ...coupon,
      attributes: { ...coupon.attributes, rewards: [cashReward] },
    });
    vi.mocked(couponsService.listRedemptions).mockResolvedValue({
      data: [
        {
          ...makeRedemption("r1", 42, 4200),
          attributes: {
            ...makeRedemption("r1", 42, 4200).attributes,
            rewardsGranted: [cashReward],
          },
        },
      ],
      meta: { total: 1, page: { number: 1, size: 50, last: 1 } },
    });
    // Serial 20000036 sells item 1002077 — a hat, i.e. EQUIPMENT, which is
    // what most of the cash shop actually is.
    vi.mocked(commoditiesService.getBySerialNumber).mockResolvedValue({
      id: "20000036",
      itemId: 1002077,
      count: 1,
      price: 3700,
      period: 90,
      priority: 5,
      gender: 2,
      onSale: true,
    });
    vi.mocked(itemStringsService.getItemString).mockResolvedValue({
      id: "1002077",
      attributes: { name: "Zeta Nova Hat" },
    } as Awaited<ReturnType<typeof itemStringsService.getItemString>>);

    renderPage();

    // Both the Rewards card and the Rewards Granted cell name the item.
    const named = await screen.findAllByText("Zeta Nova Hat ×2");
    expect(named).toHaveLength(2);
    expect(screen.queryByText(/Cash item 20000036/)).not.toBeInTheDocument();
  });

  it("falls back to the serial when the cash item cannot be resolved", async () => {
    vi.mocked(couponsService.getOne).mockResolvedValue({
      ...coupon,
      attributes: {
        ...coupon.attributes,
        rewards: [{ type: "CASH_ITEM", serialNumber: 20000036, quantity: 2 }],
      },
    });
    vi.mocked(couponsService.listRedemptions).mockResolvedValue({
      data: [],
      meta: { total: 0, page: { number: 1, size: 50, last: 1 } },
    });

    renderPage();

    expect(
      await screen.findByText("Cash item 20000036 ×2"),
    ).toBeInTheDocument();
  });

  it("shows an empty history rather than an error when the code was never redeemed", async () => {
    vi.mocked(couponsService.getOne).mockResolvedValue({
      ...coupon,
      attributes: { ...coupon.attributes, redemptionCount: 0 },
    });
    vi.mocked(couponsService.listRedemptions).mockResolvedValue({
      data: [],
      meta: { total: 0, page: { number: 1, size: 50, last: 1 } },
    });

    renderPage();

    expect(
      await screen.findByText("This code has not been redeemed yet."),
    ).toBeInTheDocument();
  });
});
