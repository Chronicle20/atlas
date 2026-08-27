import { z } from "zod";
import type {
  MapleLifeClassEntry,
  MapleLifeConfig,
} from "@/types/models/template";
import { parseSpPool } from "@/components/features/characters/maple-life/mapleLifeEditorState";

export const MSG = {
  hairNotNormalised:
    "Hair style ids are normalised to (v/10)*10 (task-246 design.md §A3); this value is not a multiple of 10.",
  hairColorRange: "Hair colour is a bare digit 0..9 (task-246 design.md §A3).",
  spNotTenBooks:
    "The SP pool must be exactly ten comma-separated integers, the shape atlas-character persists.",
  spSkillOnHighOrdinal:
    "The client skips the SP step for class ordinal >= 2 (processor.go:424-427), so a player submitting sp != 0 is rejected outright. Clear this value.",
  spPoolTooSmall:
    "Book 0 must be at least 6: the server needs sp + 5 for the prerequisite (processor.go:428-437), so a pool below 6 makes even a level-1 investment unsatisfiable.",
  emptyPool:
    "This pool is empty, so every player submission for this gender is rejected with ErrLookInvalid (processor.go:405-422).",
  missingLookRow:
    "This gender has configured class rows but no looks row, which fails with ErrMapleLifeNotConfigured (processor.go:397-405).",
} as const;

const lookOptions = z.object({
  gender: z.number().int().nonnegative(),
  faces: z.array(z.number().int().nonnegative()),
  hairs: z.array(
    z
      .number()
      .int()
      .nonnegative()
      .refine((v) => v % 10 === 0, {
        message: MSG.hairNotNormalised,
      }),
  ),
  hairColors: z.array(
    z.number().int().min(0, MSG.hairColorRange).max(9, MSG.hairColorRange),
  ),
  skinColors: z.array(z.number().int().nonnegative()),
});

const statBlock = z.object({
  str: z.number().int().nonnegative(),
  dex: z.number().int().nonnegative(),
  int: z.number().int().nonnegative(),
  luk: z.number().int().nonnegative(),
  hp: z.number().int().nonnegative(),
  mp: z.number().int().nonnegative(),
});

const equipmentEntry = z.object({
  templateId: z.number().int().nonnegative(),
  useAverageStats: z.boolean(),
});

const inventoryEntry = z.object({
  templateId: z.number().int().nonnegative(),
  quantity: z.number().int().min(1),
});

const classEntry = z
  .object({
    ordinal: z.number().int().nonnegative(),
    gender: z.number().int().nonnegative(),
    jobId: z.number().int().nonnegative(),
    level: z.number().int().nonnegative(),
    mapId: z.number().int().nonnegative(),
    stats: statBlock,
    ap: z.number().int().nonnegative(),
    sp: z.string().refine((sp) => parseSpPool(sp).length === 10, {
      message: MSG.spNotTenBooks,
    }),
    spSkillId: z.number().int().optional(),
    meso: z.number().int().nonnegative(),
    equipment: z.array(equipmentEntry),
    inventory: z.array(inventoryEntry),
  })
  .superRefine((entry, ctx) => {
    const hasSkill = entry.spSkillId !== undefined && entry.spSkillId !== 0;

    if (hasSkill && entry.ordinal >= 2) {
      ctx.addIssue({
        code: "custom",
        path: ["spSkillId"],
        message: MSG.spSkillOnHighOrdinal,
      });
    }

    if (hasSkill) {
      const pool = parseSpPool(entry.sp);
      if (pool.length === 10 && (pool[0] ?? 0) < 6) {
        ctx.addIssue({
          code: "custom",
          path: ["sp"],
          message: MSG.spPoolTooSmall,
        });
      }
    }
  })
  // `exactOptionalPropertyTypes` forbids an explicit `spSkillId: undefined`
  // on `MapleLifeClassEntry` — omit the key entirely when it is unset,
  // rather than carry zod's `| undefined` optional-output convention.
  .transform((entry): MapleLifeClassEntry => {
    const { spSkillId, ...rest } = entry;
    return spSkillId === undefined ? rest : { ...rest, spSkillId };
  });

const LOOK_DIMENSIONS = ["faces", "hairs", "hairColors", "skinColors"] as const;

export const mapleLifeSchema: z.ZodType<MapleLifeConfig> = z
  .object({
    looks: z.array(lookOptions),
    classes: z.array(classEntry),
  })
  .superRefine((config, ctx) => {
    const gendersWithClasses = new Set(config.classes.map((c) => c.gender));

    for (const gender of gendersWithClasses) {
      const lookIndex = config.looks.findIndex((l) => l.gender === gender);
      if (lookIndex === -1) {
        ctx.addIssue({
          code: "custom",
          path: ["looks"],
          message: MSG.missingLookRow,
        });
        continue;
      }

      const look = config.looks[lookIndex]!;
      for (const dimension of LOOK_DIMENSIONS) {
        if (look[dimension].length === 0) {
          ctx.addIssue({
            code: "custom",
            path: ["looks", lookIndex, dimension],
            message: MSG.emptyPool,
          });
        }
      }
    }
  });

/** Dotted path -> messages, e.g. "classes.0.spSkillId" -> [...]. */
export type IssueMap = Map<string, string[]>;

/** Runs mapleLifeSchema.safeParse and folds issues into a path-keyed map. */
export function validateMapleLife(config: MapleLifeConfig): IssueMap {
  const result = mapleLifeSchema.safeParse(config);
  const issues: IssueMap = new Map();
  if (result.success) return issues;

  for (const issue of result.error.issues) {
    const path = issue.path.join(".");
    const messages = issues.get(path) ?? [];
    messages.push(issue.message);
    issues.set(path, messages);
  }
  return issues;
}
