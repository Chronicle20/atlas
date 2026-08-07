import { z } from "zod";

/**
 * Mirrors the blocking rules in
 * services/atlas-configurations/atlas.com/configurations/socket/validate.go.
 * The server 400s on every one of these, so catching them inline is the only
 * way the dialogs stay usable. If you change a rule here, change it there.
 */

/** 0x/0X followed by 1-4 hex digits. jms_185_1 stores "0x9"; gms_84_1 "0x0A5". */
export const OPCODE_PATTERN = /^0[xX][0-9A-Fa-f]{1,4}$/;

/** The closed set from libs/atlas-opcodes/config.go. */
export const KNOWN_SERVICES = ["login", "channel"] as const;

export type KnownService = (typeof KNOWN_SERVICES)[number];

export const definitionFormSchema = z.object({
  name: z.string().trim().min(1, "Definition name is required"),
  opCode: z
    .string()
    .trim()
    .regex(OPCODE_PATTERN, "Use 0x followed by 1-4 hex digits, e.g. 0x2A"),
  validator: z.string().trim(),
  services: z.array(z.enum(KNOWN_SERVICES)),
  fname: z.string().trim().optional(),
  options: z.unknown().optional(),
});

export type DefinitionFormValues = z.infer<typeof definitionFormSchema>;

/** Handlers require a validator; writers have none (validate.go needsValidator). */
export function validatorRequiredFor(kind: "handler" | "writer"): boolean {
  return kind === "handler";
}

/**
 * Layers the per-kind validator rule on top of the shared shape, so a form
 * for a `kind` blocks exactly what the server blocks - a non-blank validator
 * for handlers, nothing extra for writers (they carry no validator field at
 * all). Never adds a validator ALLOW-list: validate.go's rule is
 * presence-only, and the corpus carries validator strings (e.g.
 * "NoOpValidator") this schema must accept unmodified.
 */
export function definitionFormSchemaFor(kind: "handler" | "writer") {
  return definitionFormSchema.superRefine((values, ctx) => {
    if (validatorRequiredFor(kind) && values.validator.trim() === "") {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["validator"],
        message: "Validator is required for handlers.",
      });
    }
  });
}
