/**
 * MTS Configuration Validation Schemas
 *
 * Zod schema for the per-tenant marketplace (MTS) economic knobs. These mirror
 * the atlas-tenants `mts-configs` JSON:API resource attributes exactly (the §8
 * economic-knob surface): listingFee, commissionRate, maxActiveListings,
 * minLevel, auctionMinHours, auctionMaxHours, priceFloor, pageSize,
 * minBidIncrement.
 */

import { z } from "zod";

/**
 * Schema for the per-tenant MTS configuration.
 *
 * Validates:
 * - listingFee: flat NX fee charged at list time (non-negative integer)
 * - commissionRate: fractional cut taken on sale, 0..1
 * - maxActiveListings: per-character active-listing cap (positive integer)
 * - minLevel: minimum character level to use the marketplace (non-negative integer)
 * - auctionMinHours / auctionMaxHours: auction duration bounds (positive integers)
 * - priceFloor: minimum NX list value (non-negative integer)
 * - pageSize: browse page size (positive integer)
 * - minBidIncrement: minimum bid step for auctions (positive integer)
 */
export const mtsConfigSchema = z
  .object({
    listingFee: z
      .number()
      .int("Listing fee must be an integer")
      .nonnegative("Listing fee must be non-negative"),
    commissionRate: z
      .number()
      .min(0, "Commission rate must be at least 0")
      .max(1, "Commission rate must be at most 1"),
    maxActiveListings: z
      .number()
      .int("Max active listings must be an integer")
      .positive("Max active listings must be greater than 0"),
    minLevel: z
      .number()
      .int("Minimum level must be an integer")
      .nonnegative("Minimum level must be non-negative"),
    auctionMinHours: z
      .number()
      .int("Auction minimum hours must be an integer")
      .positive("Auction minimum hours must be greater than 0"),
    auctionMaxHours: z
      .number()
      .int("Auction maximum hours must be an integer")
      .positive("Auction maximum hours must be greater than 0"),
    priceFloor: z
      .number()
      .int("Price floor must be an integer")
      .nonnegative("Price floor must be non-negative"),
    pageSize: z
      .number()
      .int("Page size must be an integer")
      .positive("Page size must be greater than 0"),
    minBidIncrement: z
      .number()
      .int("Minimum bid increment must be an integer")
      .positive("Minimum bid increment must be greater than 0"),
  })
  .refine((v) => v.auctionMaxHours >= v.auctionMinHours, {
    message:
      "Auction maximum hours must be greater than or equal to minimum hours",
    path: ["auctionMaxHours"],
  });

/**
 * TypeScript type inferred from the MTS config schema. Matches the
 * `MtsConfigAttributes` shape exposed by the service module.
 */
export type MtsConfigFormData = z.infer<typeof mtsConfigSchema>;
