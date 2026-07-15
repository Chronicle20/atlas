/**
 * Incubator Rewards Validation Schema
 *
 * Zod schema for incubator reward configuration. Validates the individual
 * reward rows that compose an incubator reward table.
 */

import { z } from 'zod';

/**
 * Schema for an incubator reward entry.
 *
 * Validates:
 * - itemId: the reward item (positive integer)
 * - quantity: amount of the item to reward (positive integer)
 * - weight: relative probability weight (positive integer, rejects zero)
 */
export const incubatorRewardSchema = z.object({
  itemId: z
    .number()
    .int('Item ID must be an integer')
    .positive('Item ID must be positive'),
  quantity: z
    .number()
    .int('Quantity must be an integer')
    .positive('Quantity must be positive'),
  weight: z
    .number()
    .int('Weight must be an integer')
    .positive('Weight must be positive'),
});

/**
 * TypeScript type inferred from the incubator reward schema. Matches the
 * `IncubatorReward` shape exposed by the service module.
 */
export type IncubatorRewardFormData = z.infer<typeof incubatorRewardSchema>;
