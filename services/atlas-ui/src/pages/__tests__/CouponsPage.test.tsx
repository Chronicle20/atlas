import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { CouponsPage } from "@/pages/CouponsPage";
import { couponsService } from "@/services/api/coupons.service";
import type {
  Coupon,
  CouponAttributes,
  CouponBatch,
} from "@/services/api/coupons.service";

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

function makeCoupon(
  id: string,
  overrides: Partial<CouponAttributes> = {},
): Coupon {
  return {
    id,
    attributes: {
      code: "SUMMER2026",
      active: true,
      redemptionCount: 0,
      rewards: [{ type: "CURRENCY", currency: 1, amount: 1000 }],
      createdAt: "2026-01-01T00:00:00Z",
      updatedAt: "2026-01-01T00:00:00Z",
      ...overrides,
    },
  };
}

function renderPage(initial = "/coupons") {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initial]}>
        <Routes>
          <Route path="/coupons" element={<CouponsPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function pagedCoupons(data: Coupon[]) {
  return {
    data,
    meta: { total: data.length, page: { number: 1, size: 50, last: 1 } },
  };
}

describe("CouponsPage", () => {
  beforeEach(() => {
    vi.mocked(couponsService.list).mockReset();
    vi.mocked(couponsService.create).mockReset();
    vi.mocked(couponsService.update).mockReset();
    vi.mocked(couponsService.remove).mockReset();
    vi.mocked(couponsService.generateBatch).mockReset();
    vi.mocked(couponsService.list).mockResolvedValue(pagedCoupons([]));
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders code, status, reward summary, uses and window for each coupon", async () => {
    vi.mocked(couponsService.list).mockResolvedValue(
      pagedCoupons([
        makeCoupon("c1", {
          code: "SUMMER2026",
          active: true,
          redemptionCount: 3,
          maxUses: 10,
          startsAt: "2026-06-01T00:00:00Z",
          expiresAt: "2026-07-01T00:00:00Z",
          rewards: [{ type: "CURRENCY", currency: 1, amount: 1000 }],
        }),
        makeCoupon("c2", {
          code: "WINTER2026",
          active: false,
          redemptionCount: 0,
          rewards: [{ type: "CASH_ITEM", serialNumber: 50200004, quantity: 2 }],
        }),
      ]),
    );

    renderPage();

    expect(await screen.findByText("SUMMER2026")).toBeInTheDocument();
    expect(screen.getByText("WINTER2026")).toBeInTheDocument();

    expect(screen.getByText("Active")).toBeInTheDocument();
    expect(screen.getByText("Inactive")).toBeInTheDocument();

    expect(screen.getByText("1000 NX")).toBeInTheDocument();
    expect(screen.getByText("Cash item 50200004 ×2")).toBeInTheDocument();

    // maxUses present -> "n / max"; maxUses absent -> unlimited marker.
    expect(screen.getByText("3 / 10")).toBeInTheDocument();
    expect(screen.getByText("0 / ∞")).toBeInTheDocument();

    expect(screen.getByText("2026-06-01 → 2026-07-01")).toBeInTheDocument();
    expect(screen.getByText("Always")).toBeInTheDocument();
  });

  it("narrows the list to active coupons when the active filter is chosen", async () => {
    vi.mocked(couponsService.list).mockResolvedValue(
      pagedCoupons([makeCoupon("c1")]),
    );

    renderPage();
    await screen.findByText("SUMMER2026");

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Active only" }));

    await waitFor(() => {
      const lastCall = vi.mocked(couponsService.list).mock.calls.at(-1)!;
      expect(lastCall[0]).toEqual({ number: 1, size: 50 });
      expect(lastCall[1]).toEqual({ active: true });
    });
  });

  it("blocks an invalid reward row before any create request is issued", async () => {
    renderPage();
    await screen.findByText(/No coupons/i);

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "New Coupon" }));

    const dialog = await screen.findByRole("dialog");
    // The default reward row is a CURRENCY row with a blank amount.
    await user.click(within(dialog).getByRole("button", { name: "Create" }));

    expect(
      await within(dialog).findByText(/positive whole number/i),
    ).toBeInTheDocument();
    expect(couponsService.create).not.toHaveBeenCalled();
  });

  it("omits the code attribute entirely when the code field is left blank", async () => {
    vi.mocked(couponsService.create).mockResolvedValue(makeCoupon("c9"));

    renderPage();
    await screen.findByText(/No coupons/i);

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "New Coupon" }));

    const dialog = await screen.findByRole("dialog");
    await user.type(within(dialog).getByLabelText("Amount"), "500");
    await user.click(within(dialog).getByRole("button", { name: "Create" }));

    await waitFor(() => expect(couponsService.create).toHaveBeenCalled());

    const input = vi.mocked(couponsService.create).mock.calls[0]![0];
    expect(Object.keys(input)).not.toContain("code");
    expect(input.rewards).toEqual([
      { type: "CURRENCY", currency: 1, amount: 500 },
    ]);
  });

  it("offers a CSV download built from the generated batch codes", async () => {
    const batch: CouponBatch = {
      id: "b1",
      attributes: {
        requestedCount: 2,
        generatedCount: 2,
        redeemedCount: 0,
        createdAt: "2026-01-01T00:00:00Z",
        codes: ["AAAA-1111", "BBBB-2222"],
      },
    };
    vi.mocked(couponsService.generateBatch).mockResolvedValue(batch);

    const createObjectURL = vi.fn(() => "blob:coupon-codes");
    const revokeObjectURL = vi.fn();
    vi.stubGlobal("URL", {
      ...URL,
      createObjectURL,
      revokeObjectURL,
    });

    renderPage();
    await screen.findByText(/No coupons/i);

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Generate Batch" }));

    const dialog = await screen.findByRole("dialog");
    await user.clear(within(dialog).getByLabelText("Count"));
    await user.type(within(dialog).getByLabelText("Count"), "2");
    await user.type(within(dialog).getByLabelText("Amount"), "500");
    await user.click(within(dialog).getByRole("button", { name: "Generate" }));

    const download = await within(dialog).findByRole("button", {
      name: "Download CSV",
    });
    expect(within(dialog).getByText("AAAA-1111")).toBeInTheDocument();

    await user.click(download);
    expect(createObjectURL).toHaveBeenCalledTimes(1);

    vi.unstubAllGlobals();
  });

  it("disables delete once a coupon has been redeemed", async () => {
    vi.mocked(couponsService.list).mockResolvedValue(
      pagedCoupons([
        makeCoupon("c1", { code: "USED", redemptionCount: 4 }),
        makeCoupon("c2", { code: "UNUSED", redemptionCount: 0 }),
      ]),
    );

    renderPage();
    await screen.findByText("USED");

    expect(screen.getByRole("button", { name: "Delete USED" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Delete UNUSED" })).toBeEnabled();
  });

  it("reports the 409 conflict when a delete loses the race against a redemption", async () => {
    const { CouponConflictError } =
      await import("@/services/api/coupons.service");
    vi.mocked(couponsService.list).mockResolvedValue(
      pagedCoupons([makeCoupon("c2", { code: "UNUSED", redemptionCount: 0 })]),
    );
    vi.mocked(couponsService.remove).mockRejectedValue(
      new CouponConflictError("coupon has redemptions and cannot be deleted"),
    );

    renderPage();
    await screen.findByText("UNUSED");

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Delete UNUSED" }));
    const confirm = await screen.findByRole("alertdialog");
    await user.click(within(confirm).getByRole("button", { name: "Delete" }));

    await waitFor(() =>
      expect(couponsService.remove).toHaveBeenCalledWith("c2", undefined),
    );
    expect(
      await screen.findByText(/was redeemed while you were deleting it/i),
    ).toBeInTheDocument();
  });

  it("PATCHes only the active field when the status toggle is flipped", async () => {
    vi.mocked(couponsService.list).mockResolvedValue(
      pagedCoupons([
        makeCoupon("c1", {
          code: "SUMMER2026",
          active: true,
          maxUses: 10,
          description: "summer promo",
          startsAt: "2026-06-01T00:00:00Z",
        }),
      ]),
    );
    vi.mocked(couponsService.update).mockResolvedValue(makeCoupon("c1"));

    renderPage();
    await screen.findByText("SUMMER2026");

    const user = userEvent.setup();
    await user.click(
      screen.getByRole("switch", { name: "Deactivate SUMMER2026" }),
    );

    await waitFor(() => expect(couponsService.update).toHaveBeenCalled());
    const call = vi.mocked(couponsService.update).mock.calls[0]!;
    expect(call[0]).toBe("c1");
    // Partial PATCH: omitted keys preserve stored values, so the toggle must
    // send `active` and nothing else.
    expect(call[1]).toEqual({ active: false });
    expect(Object.keys(call[1])).toEqual(["active"]);
  });
});
