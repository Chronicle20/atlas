import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { CouponDetailPage } from "@/pages/CouponDetailPage";
import { couponsService } from "@/services/api/coupons.service";
import type { Coupon, CouponRedemption } from "@/services/api/coupons.service";

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
    expect(screen.getByText("42")).toBeInTheDocument();
    expect(screen.getByText("4300")).toBeInTheDocument();
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
