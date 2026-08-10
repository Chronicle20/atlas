/**
 * Cash-shop coupon validation schemas.
 *
 * The reward editor is a field array whose rows are edited as plain strings
 * (that is what an `<input>` yields) but which must reach the API as the
 * `CouponReward` discriminated union and nothing else. Each row is therefore a
 * string object that TRANSFORMS into the branch its `type` selects and then
 * PIPES into `couponRewardSchema` — the discriminated union is the last gate
 * before the mutation, so an invalid combination (a CASH_ITEM row with only a
 * currency amount filled in, say) cannot be submitted.
 *
 * `z.input<typeof couponFormSchema>` is consequently the string-shaped form
 * state react-hook-form owns, and `z.output<...>` is the validated payload
 * `handleSubmit` hands back.
 */

import { z } from "zod";
import type { CouponReward } from "@/services/api/coupons.service";

export const REWARD_TYPES = ["CURRENCY", "CASH_ITEM"] as const;
export type RewardType = (typeof REWARD_TYPES)[number];

/**
 * The three currency codes a CURRENCY reward may name, in wire order. Shared
 * by the schema's refinement below and the editor's dropdown so the two can
 * never disagree about what is selectable; `formatCurrency` (lib/utils/coupons)
 * owns their labels.
 */
export const CURRENCY_VALUES = [1, 2, 3] as const;

/** One reward row exactly as the form holds it — every field a string. */
export interface RewardRowInput {
  type: RewardType;
  currency: string;
  amount: string;
  serialNumber: string;
  quantity: string;
}

/** A fresh CURRENCY row; `1` is the credit/NX currency (see CouponReward). */
export function emptyRewardRow(): RewardRowInput {
  return {
    type: "CURRENCY",
    currency: "1",
    amount: "",
    serialNumber: "",
    quantity: "",
  };
}

/**
 * Blank or non-numeric text becomes -1 rather than NaN so the positive-integer
 * refinement below is what reports the problem, with a message a user can act
 * on, instead of zod's "expected number, received NaN".
 */
function toNumber(raw: string): number {
  const trimmed = raw.trim();
  if (trimmed === "") return -1;
  const parsed = Number(trimmed);
  return Number.isFinite(parsed) ? parsed : -1;
}

function positiveInt(label: string) {
  return z.number().refine((n) => Number.isInteger(n) && n > 0, {
    message: `${label} must be a positive whole number`,
  });
}

/**
 * The wire shape. Deliberately identical to `CouponReward` in
 * coupons.service.ts — the assignability check below fails the build if the
 * two ever drift.
 *
 * `currency` is a refined `number` rather than a `z.literal(1|2|3)` union
 * because the row transform above produces a plain `number`, and `.pipe()`
 * requires the upstream output to be assignable to this input. The refinement
 * enforces the same three values.
 */
export const couponRewardSchema = z.discriminatedUnion("type", [
  z.object({
    type: z.literal("CURRENCY"),
    currency: z
      .number()
      .refine((n) => (CURRENCY_VALUES as readonly number[]).includes(n), {
        message: "Currency must be 1 (NX), 2 (Maple Points) or 3 (Prepaid)",
      }),
    amount: positiveInt("Amount"),
  }),
  z.object({
    type: z.literal("CASH_ITEM"),
    serialNumber: positiveInt("Serial number"),
    quantity: positiveInt("Quantity"),
  }),
]);

// Compile-time proof that the schema's output is exactly the service's type.
type _RewardMatchesService =
  z.output<typeof couponRewardSchema> extends CouponReward ? true : never;
const _rewardMatchesService: _RewardMatchesService = true;
void _rewardMatchesService;

export const rewardRowSchema = z
  .object({
    type: z.enum(REWARD_TYPES),
    currency: z.string(),
    amount: z.string(),
    serialNumber: z.string(),
    quantity: z.string(),
  })
  .transform((row) =>
    row.type === "CURRENCY"
      ? {
          type: "CURRENCY" as const,
          currency: toNumber(row.currency),
          amount: toNumber(row.amount),
        }
      : {
          type: "CASH_ITEM" as const,
          serialNumber: toNumber(row.serialNumber),
          quantity: toNumber(row.quantity),
        },
  )
  .pipe(couponRewardSchema);

/** Blank means "no limit" (the API's `maxUses` is optional/nullable). */
const optionalPositiveInt = (label: string) =>
  z
    .string()
    .trim()
    .transform((raw) => (raw === "" ? null : toNumber(raw)))
    .pipe(positiveInt(label).nullable());

function expiryAfterStart(v: { startsAt: string; expiresAt: string }): boolean {
  if (!v.startsAt || !v.expiresAt) return true;
  return new Date(v.expiresAt) > new Date(v.startsAt);
}

/** Fresh object per call: zod types `path` as a mutable `PropertyKey[]`. */
const expiryIssue = () => ({
  path: ["expiresAt"],
  message: "Expiry must be after activation",
});

export const couponFormSchema = z
  .object({
    // Blank means "generate one" — the server picks the code, and the caller
    // must then omit the attribute entirely rather than send "".
    code: z.string().trim().max(32, "Code must be at most 32 characters"),
    description: z.string().trim(),
    active: z.boolean(),
    startsAt: z.string(),
    expiresAt: z.string(),
    maxUses: optionalPositiveInt("Max uses"),
    rewards: z
      .array(rewardRowSchema)
      .min(1, "A coupon must grant at least one reward"),
  })
  .refine(expiryAfterStart, expiryIssue());

export type CouponFormInput = z.input<typeof couponFormSchema>;
export type CouponFormOutput = z.output<typeof couponFormSchema>;

export function emptyCouponForm(): CouponFormInput {
  return {
    code: "",
    description: "",
    active: true,
    startsAt: "",
    expiresAt: "",
    maxUses: "",
    rewards: [emptyRewardRow()],
  };
}

export const couponBatchFormSchema = z
  .object({
    count: z.string().trim().transform(toNumber).pipe(positiveInt("Count")),
    prefix: z.string().trim().max(16, "Prefix must be at most 16 characters"),
    length: optionalPositiveInt("Length"),
    description: z.string().trim(),
    startsAt: z.string(),
    expiresAt: z.string(),
    rewards: z
      .array(rewardRowSchema)
      .min(1, "A batch must grant at least one reward"),
  })
  .refine(expiryAfterStart, expiryIssue());

export type CouponBatchFormInput = z.input<typeof couponBatchFormSchema>;
export type CouponBatchFormOutput = z.output<typeof couponBatchFormSchema>;

export function emptyCouponBatchForm(): CouponBatchFormInput {
  return {
    count: "10",
    prefix: "",
    length: "",
    description: "",
    startsAt: "",
    expiresAt: "",
    rewards: [emptyRewardRow()],
  };
}
