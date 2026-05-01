import { z } from "zod";

export const createAccountSchema = z.object({
  name: z.string().min(4, "Name must be at least 4 characters"),
  password: z.string().min(6, "Password must be at least 6 characters"),
});

export type CreateAccountFormValues = z.infer<typeof createAccountSchema>;
