import { describe, it, expect } from "vitest";
import {
  BEGINNER_SKILL_NAMES,
  resolveSkillName,
} from "@/lib/skills/beginner-skill-names";

describe("resolveSkillName", () => {
  it("prefers a non-blank server name", () => {
    expect(resolveSkillName(1000, "Improved HP Recovery")).toBe(
      "Improved HP Recovery",
    );
  });

  it("falls back to the curated name when the server name is blank", () => {
    expect(resolveSkillName(1000, "")).toBe("Three Snails");
    expect(resolveSkillName(1004, "")).toBe("Monster Riding");
    expect(resolveSkillName(1012, "")).toBe("Bless of Nymph");
  });

  it("treats whitespace-only as blank", () => {
    expect(resolveSkillName(1001, "   ")).toBe("Recovery");
  });

  it("falls back to Skill <id> for an unknown blank-name id", () => {
    expect(resolveSkillName(99999, "")).toBe("Skill 99999");
    expect(resolveSkillName(99999, undefined)).toBe("Skill 99999");
  });

  it("covers the 13 Beginner skills", () => {
    expect(Object.keys(BEGINNER_SKILL_NAMES)).toHaveLength(13);
  });
});
