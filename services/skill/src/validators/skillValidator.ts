import { z } from "zod";

export const matchSkillsSchema = z.object({
  skill_id: z.string().uuid({ message: "Invalid skill_id format" }),
  intent: z.enum(["learn", "teach"], {
    required_error: "Intent must be 'learn' or 'teach'",
  }),
});
