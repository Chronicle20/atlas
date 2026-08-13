/**
 * Reward Pools Validation Schemas
 *
 * Zod schemas for the gachapon, incubator, and cash-surprise reward-pool
 * dialogs (see atlas-reward-pools `gachapon-pools` / `incubator-pools` /
 * `cash-surprise-pools` resources).
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

// A cash-surprise pool's id IS the box template id, exactly as an incubator
// pool's id is the egg item id — there is no separate column. npcIds is
// unused for this kind (the box is opened from the Cash Shop, not from an
// NPC), so the form omits the field entirely rather than hiding it.
export const cashSurprisePoolSchema = z.object({
  boxItemId: z.number().int().positive("Box item id is required"),
  name: z.string().min(1, "Name is required"),
});
export type CashSurprisePoolFormData = z.infer<typeof cashSurprisePoolSchema>;

// A cash-surprise entry awards a cash shop COMMODITY (serial number), not a
// raw item id: the commodity catalog owns the reward's itemId, count and
// period, so rolling a commodity guarantees a self-consistent locker entry.
// itemId and quantity stay on the entry for operator display only — the open
// path grants `ci.ItemId()` × `ci.Count()` from the commodity itself
// (atlas-cashshop surprise/processor.go) — so the form DERIVES both from the
// chosen commodity rather than asking for them. Their messages are therefore
// unreachable through the UI and phrased for the one control that exists.
// Every derived field carries the `error` option as well as the .positive()
// message: unpicked, they are `undefined`, which trips z.number()'s TYPE check
// and would otherwise surface zod's raw "expected number, received undefined".
const cashItemRequired = { error: "Choose a cash item" } as const;
export const cashSurpriseItemSchema = z.object({
  itemId: z.number(cashItemRequired).int().positive("Choose a cash item"),
  quantity: z.number(cashItemRequired).int().positive("Choose a cash item"),
  weight: z.number().int().positive("Weight must be at least 1"),
  commodityId: z.number(cashItemRequired).int().positive("Choose a cash item"),
});
export type CashSurpriseItemFormData = z.infer<typeof cashSurpriseItemSchema>;
