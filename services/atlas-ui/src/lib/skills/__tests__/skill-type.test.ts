import { describe, it, expect } from "vitest";
import { deriveSkillType } from "@/lib/skills/skill-type";

describe("deriveSkillType", () => {
  it("classifies a stat-up effect as Buff", () => {
    expect(
      deriveSkillType({ action: true, effects: [{ statups: [{ type: "WeaponDefense", amount: 16 }] }] }),
    ).toBe("Buff");
  });

  it("classifies an overTime effect with no statups as Buff", () => {
    expect(deriveSkillType({ action: false, effects: [{ overTime: true }] })).toBe("Buff");
  });

  it("classifies an action skill with no statups/overTime as Active", () => {
    expect(
      deriveSkillType({ action: true, effects: [{ damage: 120, attackCount: 1 }] }),
    ).toBe("Active");
  });

  it("classifies a non-action skill as Passive", () => {
    expect(deriveSkillType({ action: false, effects: [{ accuracy: 10 }] })).toBe("Passive");
  });

  it("degrades safely with missing fields", () => {
    expect(deriveSkillType({ action: false, effects: [] })).toBe("Passive");
    expect(deriveSkillType({ action: true } as never)).toBe("Active");
    expect(deriveSkillType({} as never)).toBe("Passive");
  });
});
