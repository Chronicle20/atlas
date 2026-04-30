import { z } from "zod";

const equipmentEntry = z.object({
    templateId: z.number().int().nonnegative(),
    useAverageStats: z.boolean(),
});

const inventoryEntry = z.object({
    templateId: z.number().int().nonnegative(),
    quantity: z.number().int().min(1),
});

const skillEntry = z.object({
    skillId: z.number().int().nonnegative(),
    level: z.number().int().min(1),
});

const stats = z.object({
    str: z.number().int().nonnegative(),
    dex: z.number().int().nonnegative(),
    int: z.number().int().nonnegative(),
    luk: z.number().int().nonnegative(),
    hp: z.number().int().nonnegative(),
    mp: z.number().int().nonnegative(),
});

export const presetSchema = z.object({
    id: z.string().optional(),
    attributes: z.object({
        name: z.string().min(1).max(64),
        description: z.string().max(512),
        tags: z.array(z.string()),
        jobId: z.number().int().nonnegative(),
        gender: z.union([z.literal(0), z.literal(1)]),
        face: z.number().int().nonnegative(),
        hair: z.number().int().nonnegative(),
        hairColor: z.number().int().nonnegative(),
        skinColor: z.number().int().nonnegative(),
        mapId: z.number().int().nonnegative(),
        level: z.number().int().min(1).max(250),
        meso: z.number().int().nonnegative(),
        gm: z.number().int().min(0),
        stats,
        defaultName: z.string(),
        equipment: z.array(equipmentEntry),
        inventory: z.array(inventoryEntry),
        skills: z.array(skillEntry),
    }),
});

export const presetsFormSchema = z.object({
    presets: z.array(presetSchema),
});

export type PresetsFormValues = z.infer<typeof presetsFormSchema>;
