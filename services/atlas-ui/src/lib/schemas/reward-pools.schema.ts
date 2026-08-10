/**
 * Reward Pools Validation Schemas
 *
 * Zod schemas for the gachapon and incubator reward-pool dialogs (see
 * atlas-reward-pools `gachapon-pools` / `incubator-pools` resources).
 */

import { z } from "zod";

export const gachaponPoolSchema = z
  .object({
    name: z.string().min(1, "Name is required"),
    npcIds: z.array(z.number().int().positive()),
    commonWeight: z.number().int().min(0),
    uncommonWeight: z.number().int().min(0),
    rareWeight: z.number().int().min(0),
  })
  .refine((v) => v.commonWeight + v.uncommonWeight + v.rareWeight > 0, {
    message: "Tier weights must sum to more than zero",
    path: ["commonWeight"],
  });
export type GachaponPoolFormData = z.infer<typeof gachaponPoolSchema>;

export const incubatorPoolSchema = z.object({
  eggItemId: z.number().int().positive("Egg item id is required"),
  name: z.string().min(1, "Name is required"),
  successNpcId: z.number().int().positive("Success NPC id is required"),
});
export type IncubatorPoolFormData = z.infer<typeof incubatorPoolSchema>;

// The item field is a picker that starts unset, so the value RHF submits when
// nothing was chosen is `undefined` — that trips z.number()'s TYPE check, not
// .positive(), and would surface zod's raw "expected number, received
// undefined". The `error` option is what puts the authored message on that
// path.
export const tierItemSchema = z.object({
  itemId: z
    .number({ error: "Item id is required" })
    .int()
    .positive("Item id is required"),
  quantity: z.number().int().positive(),
  tier: z.enum(["common", "uncommon", "rare"]),
});
export type TierItemFormData = z.infer<typeof tierItemSchema>;

export const weightItemSchema = z.object({
  itemId: z
    .number({ error: "Item id is required" })
    .int()
    .positive("Item id is required"),
  quantity: z.number().int().positive(),
  weight: z.number().int().positive("Weight must be at least 1"),
});
export type WeightItemFormData = z.infer<typeof weightItemSchema>;
