/**
 * React Query hooks for cash-shop coupon codes.
 *
 * Mirrors useAccounts.ts / useRewardPools.ts: a query-key factory, useQuery
 * readers, and useMutation writers that invalidate the list (and, for
 * update/delete, the touched detail) on success.
 *
 * Every query hook is gated on `!!activeTenant` (via `useTenant()`), matching
 * useRewardPools.ts. Coupons are a tenant-scoped resource read through the
 * singleton `apiClient`, which carries whatever tenant it was last set to —
 * without the gate, a query mounted before an active tenant is set (or
 * during a tenant switch) would fire against the wrong tenant's headers,
 * reading another tenant's coupons or 400ing.
 */

import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationResult,
  type UseQueryResult,
} from "@tanstack/react-query";
import {
  couponsService,
  type Coupon,
  type CouponBatch,
  type CouponFilters,
  type CouponRedemption,
  type CouponRedemptionQuery,
  type CreateCouponInput,
  type GenerateCouponBatchInput,
  type UpdateCouponInput,
} from "@/services/api/coupons.service";
import type { PagedResult } from "@/services/api/pagination";
import type { ServiceOptions } from "@/lib/api/query-params";
import { useTenant } from "@/context/tenant-context";

export const couponKeys = {
  all: ["coupons"] as const,
  lists: () => [...couponKeys.all, "list"] as const,
  list: (page: { number: number; size: number }, filters?: CouponFilters) =>
    [...couponKeys.lists(), page.number, page.size, filters] as const,
  details: () => [...couponKeys.all, "detail"] as const,
  detail: (id: string) => [...couponKeys.details(), id] as const,
  redemptions: () => [...couponKeys.all, "redemptions"] as const,
  redemptionsFor: (
    query: CouponRedemptionQuery,
    page: { number: number; size: number },
  ) =>
    [
      ...couponKeys.redemptions(),
      "couponId" in query ? query.couponId : `account:${query.accountId}`,
      page.number,
      page.size,
    ] as const,
  batches: () => [...couponKeys.all, "batches"] as const,
  batchList: (page: { number: number; size: number }) =>
    [...couponKeys.batches(), "list", page.number, page.size] as const,
  batchDetail: (id: string) => [...couponKeys.batches(), "detail", id] as const,
};

/** GET /coupons — one page, matching `filters`. */
export function useCoupons(
  page: { number: number; size: number },
  filters?: CouponFilters,
  options?: ServiceOptions,
): UseQueryResult<PagedResult<Coupon>, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: couponKeys.list(page, filters),
    queryFn: () => couponsService.list(page, filters, options),
    enabled: !!activeTenant,
    gcTime: 5 * 60 * 1000,
  });
}

/** GET /coupons/{id} */
export function useCoupon(
  id: string,
  options?: ServiceOptions,
): UseQueryResult<Coupon, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: couponKeys.detail(id),
    queryFn: () => couponsService.getOne(id, options),
    enabled: !!activeTenant && !!id,
    gcTime: 5 * 60 * 1000,
  });
}

/**
 * GET /coupons/{id}/redemptions or GET /coupon-redemptions?filter[accountId]=,
 * depending on which key `query` carries.
 */
export function useCouponRedemptions(
  query: CouponRedemptionQuery,
  page: { number: number; size: number },
  options?: ServiceOptions,
): UseQueryResult<PagedResult<CouponRedemption>, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: couponKeys.redemptionsFor(query, page),
    queryFn: () => couponsService.listRedemptions(query, page, options),
    enabled: !!activeTenant,
    gcTime: 5 * 60 * 1000,
  });
}

/** GET /coupon-batches */
export function useCouponBatches(
  page: { number: number; size: number },
  options?: ServiceOptions,
): UseQueryResult<PagedResult<CouponBatch>, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: couponKeys.batchList(page),
    queryFn: () => couponsService.listBatches(page, options),
    enabled: !!activeTenant,
    gcTime: 5 * 60 * 1000,
  });
}

/** GET /coupon-batches/{id} */
export function useCouponBatch(
  id: string,
  options?: ServiceOptions,
): UseQueryResult<CouponBatch, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: couponKeys.batchDetail(id),
    queryFn: () => couponsService.getBatch(id, options),
    enabled: !!activeTenant && !!id,
    gcTime: 5 * 60 * 1000,
  });
}

/** POST /coupons */
export function useCreateCoupon(): UseMutationResult<
  Coupon,
  Error,
  { input: CreateCouponInput; options?: ServiceOptions }
> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ input, options }) => couponsService.create(input, options),
    onSuccess: () => qc.invalidateQueries({ queryKey: couponKeys.lists() }),
  });
}

/** PATCH /coupons/{id} — genuinely partial, see UpdateCouponInput. */
export function useUpdateCoupon(): UseMutationResult<
  Coupon,
  Error,
  { id: string; patch: UpdateCouponInput; options?: ServiceOptions }
> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, patch, options }) =>
      couponsService.update(id, patch, options),
    onSuccess: (_data, { id }) => {
      qc.invalidateQueries({ queryKey: couponKeys.lists() });
      qc.invalidateQueries({ queryKey: couponKeys.detail(id) });
    },
  });
}

/**
 * DELETE /coupons/{id}. Rejects with CouponConflictError on 409 — the caller
 * (task 27) is expected to disable the delete affordance on
 * `redemptionCount > 0` and handle this as the race-lost case when it still
 * happens.
 */
export function useDeleteCoupon(): UseMutationResult<
  void,
  Error,
  { id: string; options?: ServiceOptions }
> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, options }) => couponsService.remove(id, options),
    onSuccess: (_data, { id }) => {
      qc.invalidateQueries({ queryKey: couponKeys.lists() });
      qc.invalidateQueries({ queryKey: couponKeys.detail(id) });
    },
  });
}

/** POST /coupon-batches */
export function useGenerateCouponBatch(): UseMutationResult<
  CouponBatch,
  Error,
  { input: GenerateCouponBatchInput; options?: ServiceOptions }
> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ input, options }) =>
      couponsService.generateBatch(input, options),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: couponKeys.batches() });
      qc.invalidateQueries({ queryKey: couponKeys.lists() });
    },
  });
}
