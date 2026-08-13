import { describe, expect, it, vi, beforeEach } from "vitest";

const getMock = vi.fn();
const getOneMock = vi.fn();
const postMock = vi.fn();
const patchMock = vi.fn();
const deleteMock = vi.fn();

vi.mock("@/lib/api/client", () => ({
  api: {
    get: (...args: unknown[]) => getMock(...args),
    getOne: (...args: unknown[]) => getOneMock(...args),
    post: (...args: unknown[]) => postMock(...args),
    patch: (...args: unknown[]) => patchMock(...args),
    delete: (...args: unknown[]) => deleteMock(...args),
  },
}));

import {
  couponsService,
  CouponConflictError,
  type Coupon,
} from "@/services/api/coupons.service";

function makeCoupon(
  id: string,
  overrides?: Partial<Coupon["attributes"]>,
): Coupon {
  return {
    id,
    attributes: {
      code: "SUMMER2026",
      active: true,
      redemptionCount: 0,
      rewards: [{ type: "CURRENCY", currency: 1, amount: 5000 }],
      createdAt: "2026-01-01T00:00:00Z",
      updatedAt: "2026-01-01T00:00:00Z",
      ...overrides,
    },
  };
}

describe("couponsService.create", () => {
  beforeEach(() => vi.clearAllMocks());

  it("sends a JSON:API envelope with an exact body", async () => {
    postMock.mockResolvedValue({ data: makeCoupon("1") });

    await couponsService.create({
      code: "SUMMER2026",
      active: true,
      rewards: [{ type: "CURRENCY", currency: 1, amount: 5000 }],
    });

    expect(postMock).toHaveBeenCalledWith(
      "/api/coupons",
      {
        data: {
          type: "coupons",
          attributes: {
            code: "SUMMER2026",
            active: true,
            rewards: [{ type: "CURRENCY", currency: 1, amount: 5000 }],
          },
        },
      },
      undefined,
    );
  });
});

describe("couponsService.update", () => {
  beforeEach(() => vi.clearAllMocks());

  it("PATCHes with {data:{type,id,attributes}}", async () => {
    patchMock.mockResolvedValue({ data: makeCoupon("1", { active: false }) });

    await couponsService.update("1", { active: false });

    expect(patchMock).toHaveBeenCalledWith(
      "/api/coupons/1",
      {
        data: {
          type: "coupons",
          id: "1",
          attributes: { active: false },
        },
      },
      undefined,
    );
  });

  it("omits an untouched field entirely (absent -> preserve)", async () => {
    patchMock.mockResolvedValue({ data: makeCoupon("1") });

    await couponsService.update("1", { active: true });

    const [, body] = patchMock.mock.calls[0] as [
      string,
      { data: { attributes: object } },
    ];
    expect(body.data.attributes).not.toHaveProperty("description");
    expect(body.data.attributes).not.toHaveProperty("expiresAt");
  });

  it("sends an explicit null to clear a nullable field", async () => {
    patchMock.mockResolvedValue({ data: makeCoupon("1") });

    await couponsService.update("1", { expiresAt: null, maxUses: null });

    expect(patchMock).toHaveBeenCalledWith(
      "/api/coupons/1",
      {
        data: {
          type: "coupons",
          id: "1",
          attributes: { expiresAt: null, maxUses: null },
        },
      },
      undefined,
    );
  });
});

describe("couponsService.generateBatch", () => {
  beforeEach(() => vi.clearAllMocks());

  it("posts to /api/coupon-batches and returns the generated codes", async () => {
    postMock.mockResolvedValue({
      data: {
        id: "batch-1",
        attributes: {
          requestedCount: 3,
          generatedCount: 3,
          redeemedCount: 0,
          createdAt: "2026-01-01T00:00:00Z",
          codes: ["ABC123", "DEF456", "GHI789"],
        },
      },
    });

    const result = await couponsService.generateBatch({
      count: 3,
      prefix: "SUM",
      length: 8,
      startsAt: "2026-06-01T00:00:00Z",
      expiresAt: "2026-08-31T00:00:00Z",
      rewards: [{ type: "CURRENCY", currency: 1, amount: 1000 }],
      description: "Summer batch",
    });

    expect(postMock).toHaveBeenCalledWith(
      "/api/coupon-batches",
      {
        data: {
          type: "coupon-batches",
          attributes: {
            count: 3,
            prefix: "SUM",
            length: 8,
            startsAt: "2026-06-01T00:00:00Z",
            expiresAt: "2026-08-31T00:00:00Z",
            rewards: [{ type: "CURRENCY", currency: 1, amount: 1000 }],
            description: "Summer batch",
          },
        },
      },
      undefined,
    );
    expect(result.attributes.codes).toEqual(["ABC123", "DEF456", "GHI789"]);
  });
});

describe("couponsService.list", () => {
  beforeEach(() => vi.clearAllMocks());

  it("passes filter[active], filter[code] and filter[batchId] through as query params", async () => {
    getMock.mockResolvedValue({
      data: [makeCoupon("1")],
      meta: { total: 1, page: { number: 1, size: 20, last: 1 } },
    });

    await couponsService.list(
      { number: 1, size: 20 },
      { active: true, code: "SUM", batchId: "batch-1" },
    );

    const [calledUrl] = getMock.mock.calls[0] as [string];
    const params = new URL(calledUrl, "http://example.test").searchParams;
    expect(params.get("filter[active]")).toBe("true");
    expect(params.get("filter[code]")).toBe("SUM");
    expect(params.get("filter[batchId]")).toBe("batch-1");
    expect(params.get("page[number]")).toBe("1");
    expect(params.get("page[size]")).toBe("20");
  });
});

describe("couponsService.remove", () => {
  beforeEach(() => vi.clearAllMocks());

  it("surfaces a 409 as a CouponConflictError", async () => {
    deleteMock.mockRejectedValue({
      message: "coupon has redemptions and cannot be deleted",
      statusCode: 409,
      code: "NETWORK_ERROR",
    });

    await expect(couponsService.remove("1")).rejects.toBeInstanceOf(
      CouponConflictError,
    );
  });

  it("rethrows a non-409 failure unchanged", async () => {
    const notFound = {
      message: "no such coupon",
      statusCode: 404,
      code: "NOT_FOUND",
    };
    deleteMock.mockRejectedValue(notFound);

    await expect(couponsService.remove("1")).rejects.toBe(notFound);
  });
});
