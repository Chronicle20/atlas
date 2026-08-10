/**
 * Presentation helpers shared by the coupon list, detail page and dialogs.
 *
 * Nothing here talks to the API — these turn `CouponAttributes` into the
 * strings the tables render, and turn a generated batch's `codes` array into
 * the CSV the operator downloads (built client-side; the backend exposes no
 * CSV endpoint).
 */

import type {
  CouponAttributes,
  CouponReward,
} from "@/services/api/coupons.service";

/**
 * `currency` is an unconstrained `uint32` on the Go side; the convention it
 * reuses (wallet.Model.Balance) is 1 = credit/NX, 2 = Maple Points, else
 * prepaid — see the CouponReward doc comment in coupons.service.ts.
 */
export function formatCurrency(currency: number): string {
  if (currency === 1) return "NX";
  if (currency === 2) return "Maple Points";
  return "Prepaid";
}

export function formatReward(reward: CouponReward): string {
  if (reward.type === "CURRENCY") {
    return `${reward.amount} ${formatCurrency(reward.currency)}`;
  }
  return `Cash item ${reward.serialNumber} ×${reward.quantity}`;
}

export function formatRewards(rewards: CouponReward[]): string {
  if (rewards.length === 0) return "No rewards";
  return rewards.map(formatReward).join(", ");
}

/**
 * `redemptionCount / maxUses`, with `∞` standing in for an absent (unlimited)
 * `maxUses`.
 */
export function formatUses(attributes: {
  redemptionCount: number;
  maxUses?: number;
}): string {
  const limit =
    attributes.maxUses === undefined || attributes.maxUses === null
      ? "∞"
      : String(attributes.maxUses);
  return `${attributes.redemptionCount} / ${limit}`;
}

/** ISO date portion only — stable across locales, unlike toLocaleDateString. */
export function formatDate(iso: string | undefined): string | null {
  if (!iso) return null;
  const parsed = new Date(iso);
  if (Number.isNaN(parsed.getTime())) return iso;
  return parsed.toISOString().slice(0, 10);
}

/** Full timestamp for audit rows, where the time of day matters. */
export function formatTimestamp(iso: string): string {
  const parsed = new Date(iso);
  if (Number.isNaN(parsed.getTime())) return iso;
  return parsed.toISOString().replace("T", " ").slice(0, 19);
}

/** "Always" when unbounded on both ends, else `start → end` with an open side. */
export function formatWindow(attributes: {
  startsAt?: string;
  expiresAt?: string;
}): string {
  const from = formatDate(attributes.startsAt);
  const to = formatDate(attributes.expiresAt);
  if (!from && !to) return "Always";
  return `${from ?? "—"} → ${to ?? "—"}`;
}

export type CouponStatus = "Active" | "Inactive";

/**
 * The badge label. This reports the stored `active` flag and nothing else —
 * no expiry/exhaustion is folded in, because whether the server treats an
 * expired-but-active coupon as redeemable is its decision, not this table's.
 * The window and uses columns show those facts separately.
 */
export function couponStatus(attributes: CouponAttributes): CouponStatus {
  return attributes.active ? "Active" : "Inactive";
}

/** One header row plus one code per line; RFC-4180 line endings. */
export function buildCodesCsv(codes: string[]): string {
  return ["code", ...codes].join("\r\n") + "\r\n";
}

/**
 * Hand the CSV to the browser. Mirrors downloadJson's teardown discipline:
 * the body is built before createObjectURL, and the anchor removal plus the
 * revoke live in a `finally` so a throw cannot leak an object URL.
 */
export function downloadCodesCsv(filename: string, codes: string[]): void {
  const body = buildCodesCsv(codes);
  const url = URL.createObjectURL(new Blob([body], { type: "text/csv" }));
  const anchor = document.createElement("a");
  try {
    anchor.href = url;
    anchor.download = filename;
    document.body.appendChild(anchor);
    anchor.click();
  } finally {
    anchor.remove();
    URL.revokeObjectURL(url);
  }
}
