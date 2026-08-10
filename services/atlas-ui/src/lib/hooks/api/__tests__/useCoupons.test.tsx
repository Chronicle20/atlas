import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import {
  useCoupons,
  useCouponBatches,
  useCouponRedemptions,
} from "../useCoupons";
import { couponsService } from "@/services/api/coupons.service";

function emptyPage() {
  return {
    data: [],
    meta: { total: 0, page: { number: 1, size: 20, last: 1 } },
  };
}

vi.mock("@/services/api/coupons.service", () => ({
  couponsService: {
    list: vi.fn().mockResolvedValue(emptyPage()),
    listBatches: vi.fn().mockResolvedValue(emptyPage()),
    listRedemptions: vi.fn().mockResolvedValue(emptyPage()),
  },
}));

let activeTenant: { id: string } | null = {
  id: "t1",
};
vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant }),
}));

function wrapper(qc: QueryClient) {
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  );
}

describe("coupon list hooks — tenant-readiness gate", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    activeTenant = { id: "t1" };
  });

  it("useCoupons fetches once a tenant is active", async () => {
    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const { result } = renderHook(() => useCoupons({ number: 1, size: 20 }), {
      wrapper: wrapper(qc),
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(couponsService.list).toHaveBeenCalled();
  });

  it("useCoupons does not fire without an active tenant", async () => {
    activeTenant = null;
    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const { result } = renderHook(() => useCoupons({ number: 1, size: 20 }), {
      wrapper: wrapper(qc),
    });
    expect(result.current.fetchStatus).toBe("idle");
    expect(couponsService.list).not.toHaveBeenCalled();
  });

  it("useCouponBatches does not fire without an active tenant", async () => {
    activeTenant = null;
    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    renderHook(() => useCouponBatches({ number: 1, size: 20 }), {
      wrapper: wrapper(qc),
    });
    expect(couponsService.listBatches).not.toHaveBeenCalled();
  });

  it("useCouponRedemptions does not fire without an active tenant", async () => {
    activeTenant = null;
    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    renderHook(
      () => useCouponRedemptions({ couponId: "c1" }, { number: 1, size: 20 }),
      { wrapper: wrapper(qc) },
    );
    expect(couponsService.listRedemptions).not.toHaveBeenCalled();
  });
});
