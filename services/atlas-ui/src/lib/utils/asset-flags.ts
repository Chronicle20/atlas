import type { Asset } from "@/services/api/inventory.service";

/** Asset flag bit for a sealing lock. Mirrors libs/atlas-constants/asset/flag.go:6 (FlagLock = 0x01). */
export const FLAG_LOCK = 0x01;

/** Sentinel the backend emits for "no expiration". */
export const ZERO_DATE = "0001-01-01T00:00:00Z";

export function isSealed(a: Asset): boolean {
  return (a.attributes.flag & FLAG_LOCK) !== 0;
}

export function isTagged(a: Asset): boolean {
  return a.attributes.owner.trim() !== "";
}
