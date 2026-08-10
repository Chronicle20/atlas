/**
 * Cash-shop coupon codes: admin CRUD over `coupons`, generation of
 * `coupon-batches`, and read-only `coupon-redemptions` audit views.
 *
 * Backed by atlas-cashshop (task-206 backend, task-23):
 *   GET/POST   /api/coupons
 *   GET/PATCH/DELETE /api/coupons/{id}
 *   GET/POST   /api/coupon-batches
 *   GET        /api/coupon-batches/{id}
 *   GET        /api/coupons/{id}/redemptions
 *   GET        /api/coupon-redemptions?filter[accountId]=
 *
 * There is deliberately no redeem endpoint on the backend (a REST redeem
 * would be an unauthenticated reward faucet — see coupon/resource.go) and
 * no client method for one here.
 *
 * Writes use the JSON:API envelope `{data:{type, attributes}}` (create) /
 * `{data:{type, id, attributes}}` (update) — a bare body 400s.
 */

import { api } from "@/lib/api/client";
import type { ServiceOptions } from "@/lib/api/query-params";
import { fetchPaged, type PagedResult } from "@/services/api/pagination";
import type { ApiSingleResponse } from "@/types/api/responses";

const BASE_PATH = "/api/coupons";
const BATCH_PATH = "/api/coupon-batches";
const REDEMPTIONS_PATH = "/api/coupon-redemptions";

export const COUPON_RESOURCE_TYPE = "coupons";
export const COUPON_BATCH_RESOURCE_TYPE = "coupon-batches";
export const COUPON_REDEMPTION_RESOURCE_TYPE = "coupon-redemptions";

/**
 * Mirrors the Go coupon/reward.Reward jsonb/REST document exactly
 * (services/atlas-cashshop/atlas.com/cashshop/coupon/reward/reward.go). It is
 * the same document in the jsonb column and the REST attribute, and is
 * pinned by an exact-bytes Go test — do not invent a different shape here.
 *
 * `currency` is `uint32` in Go (no enum) — the doc comment there notes it
 * reuses wallet.Model.Balance's convention (1 = credit/NX, 2 = Maple Points,
 * else = prepaid) but the type itself is unconstrained, so this is typed as
 * `number` rather than a `1 | 2 | 3` literal union.
 */
export type CouponReward =
  | { type: "CURRENCY"; currency: number; amount: number }
  | { type: "CASH_ITEM"; serialNumber: number; quantity: number };

/** Mirrors coupon.RestModel (coupon/rest.go). */
export interface CouponAttributes {
  batchId?: string;
  code: string;
  description?: string;
  active: boolean;
  startsAt?: string;
  expiresAt?: string;
  maxUses?: number;
  redemptionCount: number;
  rewards: CouponReward[];
  createdAt: string;
  updatedAt: string;
}

export interface Coupon {
  id: string;
  attributes: CouponAttributes;
}

/**
 * Mirrors batch.RestModel (coupon/batch/rest.go). `codes` is OUTPUT-ONLY —
 * present only on the response of `generateBatch` (the one moment the
 * plaintext codes are handed back); a later `getBatch`/`listBatches` will not
 * carry it.
 */
export interface CouponBatchAttributes {
  description?: string;
  requestedCount: number;
  generatedCount: number;
  redeemedCount: number;
  createdAt: string;
  codes?: string[];
}

export interface CouponBatch {
  id: string;
  attributes: CouponBatchAttributes;
}

/** Mirrors redemption.RestModel (coupon/redemption/rest.go). Read-only. */
export interface CouponRedemptionAttributes {
  couponId: string;
  accountId: number;
  characterId: number;
  transactionId: string;
  rewardsGranted: CouponReward[];
  redeemedAt: string;
}

export interface CouponRedemption {
  id: string;
  attributes: CouponRedemptionAttributes;
}

/** GET /coupons query narrowings (coupon/resource.go#parseFilters). */
export interface CouponFilters {
  active?: boolean;
  code?: string;
  batchId?: string;
  expiresBefore?: string;
  expiresAfter?: string;
}

/** POST /coupons body. A blank/omitted `code` asks the server to generate one. */
export interface CreateCouponInput {
  code?: string;
  description?: string;
  active: boolean;
  startsAt?: string;
  expiresAt?: string;
  maxUses?: number;
  rewards: CouponReward[];
}

/**
 * PATCH /coupons/{id} body. Mirrors coupon.PatchRestModel (coupon/patch.go)
 * FIELD SEMANTICS exactly:
 *
 *   - a key OMITTED from this object preserves the stored value.
 *   - `description`/`startsAt`/`expiresAt`/`maxUses` set to `null` CLEARS it.
 *   - `active`/`rewards`, present, REPLACE the stored value (neither is
 *     nullable on the Go side: `active` has no third state, and an explicit
 *     null for `rewards` reaches Rewards.Validate and is refused because a
 *     coupon must grant at least one reward).
 *
 * There is no wrapper type here (unlike Go's `Nullable[T]`) because
 * `JSON.stringify` already distinguishes "key absent" (dropped) from
 * "key: null" (serialized) — callers just omit a field to leave it alone.
 * `code` and `batchId` are not included: both are server-owned and ignored
 * by the backend wherever they appear in a PATCH body.
 */
export interface UpdateCouponInput {
  description?: string | null;
  active?: boolean;
  startsAt?: string | null;
  expiresAt?: string | null;
  maxUses?: number | null;
  rewards?: CouponReward[];
}

/** POST /coupon-batches body (batch.RestModel's input-only fields). */
export interface GenerateCouponBatchInput {
  count: number;
  prefix?: string;
  length?: number;
  startsAt?: string;
  expiresAt?: string;
  rewards: CouponReward[];
  description?: string;
}

/** Either audit route: by coupon (GET /coupons/{id}/redemptions) or by account (GET /coupon-redemptions?filter[accountId]=). */
export type CouponRedemptionQuery =
  { couponId: string } | { accountId: number };

/**
 * Thrown by `remove` on a 409 — either a duplicate normalized code (not
 * applicable to delete) or, the real case here, a coupon that still has
 * redemptions (coupon/resource.go#handleDeleteCoupon, ErrHasRedemptions).
 * Lets callers distinguish "this delete lost a race against a redemption"
 * from a generic failure without string-matching the message.
 */
export class CouponConflictError extends Error {
  readonly status = 409;

  constructor(message: string) {
    super(message);
    this.name = "CouponConflictError";
  }
}

function isConflict(error: unknown): boolean {
  if (!error || typeof error !== "object") return false;
  const withStatus = error as { status?: number; statusCode?: number };
  return withStatus.status === 409 || withStatus.statusCode === 409;
}

function errorMessage(error: unknown, fallback: string): string {
  if (error && typeof error === "object" && "message" in error) {
    const message = (error as { message?: unknown }).message;
    if (typeof message === "string" && message) return message;
  }
  return fallback;
}

function couponQueryString(filters?: CouponFilters): string {
  const params = new URLSearchParams();
  if (filters?.code) params.append("filter[code]", filters.code);
  if (filters?.active !== undefined)
    params.append("filter[active]", String(filters.active));
  if (filters?.batchId) params.append("filter[batchId]", filters.batchId);
  if (filters?.expiresBefore)
    params.append("filter[expiresBefore]", filters.expiresBefore);
  if (filters?.expiresAfter)
    params.append("filter[expiresAfter]", filters.expiresAfter);
  const qs = params.toString();
  return qs ? `?${qs}` : "";
}

export const couponsService = {
  /** GET /coupons — one page, matching `filters`. */
  async list(
    page: { number: number; size: number },
    filters?: CouponFilters,
    options?: ServiceOptions,
  ): Promise<PagedResult<Coupon>> {
    return fetchPaged<Coupon>(
      `${BASE_PATH}${couponQueryString(filters)}`,
      page,
      options,
    );
  },

  /** GET /coupons/{id} */
  async getOne(id: string, options?: ServiceOptions): Promise<Coupon> {
    return api.getOne<Coupon>(`${BASE_PATH}/${id}`, options);
  },

  /** POST /coupons */
  async create(
    input: CreateCouponInput,
    options?: ServiceOptions,
  ): Promise<Coupon> {
    const response = await api.post<ApiSingleResponse<Coupon>>(
      BASE_PATH,
      { data: { type: COUPON_RESOURCE_TYPE, attributes: input } },
      options,
    );
    return response.data;
  },

  /** PATCH /coupons/{id} — genuinely partial, see UpdateCouponInput. */
  async update(
    id: string,
    patch: UpdateCouponInput,
    options?: ServiceOptions,
  ): Promise<Coupon> {
    const response = await api.patch<ApiSingleResponse<Coupon>>(
      `${BASE_PATH}/${id}`,
      { data: { type: COUPON_RESOURCE_TYPE, id, attributes: patch } },
      options,
    );
    return response.data;
  },

  /**
   * DELETE /coupons/{id}. Rejects with a CouponConflictError on 409 (the
   * coupon still has redemptions); any other failure is rethrown unchanged.
   */
  async remove(id: string, options?: ServiceOptions): Promise<void> {
    try {
      await api.delete(`${BASE_PATH}/${id}`, options);
    } catch (error) {
      if (isConflict(error)) {
        throw new CouponConflictError(
          errorMessage(error, "coupon has redemptions and cannot be deleted"),
        );
      }
      throw error;
    }
  },

  /**
   * GET /coupons/{id}/redemptions or GET /coupon-redemptions?filter[accountId]=,
   * depending on which key `query` carries.
   */
  async listRedemptions(
    query: CouponRedemptionQuery,
    page: { number: number; size: number },
    options?: ServiceOptions,
  ): Promise<PagedResult<CouponRedemption>> {
    const url =
      "couponId" in query
        ? `${BASE_PATH}/${query.couponId}/redemptions`
        : `${REDEMPTIONS_PATH}?filter[accountId]=${query.accountId}`;
    return fetchPaged<CouponRedemption>(url, page, options);
  },

  /** GET /coupon-batches */
  async listBatches(
    page: { number: number; size: number },
    options?: ServiceOptions,
  ): Promise<PagedResult<CouponBatch>> {
    return fetchPaged<CouponBatch>(BATCH_PATH, page, options);
  },

  /** GET /coupon-batches/{id} */
  async getBatch(id: string, options?: ServiceOptions): Promise<CouponBatch> {
    return api.getOne<CouponBatch>(`${BATCH_PATH}/${id}`, options);
  },

  /**
   * POST /coupon-batches. The returned batch's `attributes.codes` carries
   * the generated plaintext codes — the one response that ever does.
   */
  async generateBatch(
    input: GenerateCouponBatchInput,
    options?: ServiceOptions,
  ): Promise<CouponBatch> {
    const response = await api.post<ApiSingleResponse<CouponBatch>>(
      BATCH_PATH,
      { data: { type: COUPON_BATCH_RESOURCE_TYPE, attributes: input } },
      options,
    );
    return response.data;
  },
};
