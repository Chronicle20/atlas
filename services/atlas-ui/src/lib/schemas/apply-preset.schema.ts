import { z } from "zod";

export const applyPresetSchema = z.object({
  presetId: z.string().min(1, "Select a preset"),
  worldId: z.number().int().min(0),
  name: z
    .string()
    .min(3, "Name must be at least 3 characters")
    .max(12, "Name must be at most 12 characters"),
});

export type ApplyPresetFormValues = z.infer<typeof applyPresetSchema>;
