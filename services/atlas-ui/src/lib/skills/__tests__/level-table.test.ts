import { describe, it, expect } from "vitest";
import { buildLevelTable } from "@/lib/skills/level-table";
import type { SkillEffect } from "@/services/api/skills.service";

describe("buildLevelTable", () => {
  it("returns just a Level column for empty effects", () => {
    const t = buildLevelTable([]);
    expect(t.columns.map((c) => c.key)).toEqual(["level"]);
    expect(t.rows).toEqual([]);
  });

  it("emits one row per level with level numbers 1..n", () => {
    const effects: SkillEffect[] = [{ MPConsume: 10 }, { MPConsume: 12 }];
    const t = buildLevelTable(effects);
    expect(t.rows).toHaveLength(2);
    expect(t.rows[0]?.level).toBe("1");
    expect(t.rows[1]?.level).toBe("2");
  });

  it("omits a column that is zero/absent across every level", () => {
    const effects: SkillEffect[] = [
      { MPConsume: 10, weaponAttack: 0 },
      { MPConsume: 12 },
    ];
    const t = buildLevelTable(effects);
    const keys = t.columns.map((c) => c.key);
    expect(keys).toContain("MPConsume");
    expect(keys).not.toContain("weaponAttack");
  });

  it("includes a column with at least one non-zero level", () => {
    const effects: SkillEffect[] = [{ weaponAttack: 0 }, { weaponAttack: 5 }];
    const t = buildLevelTable(effects);
    const col = t.columns.find((c) => c.key === "weaponAttack");
    expect(col?.label).toBe("Weapon Atk");
    expect(t.rows[0]?.weaponAttack).toBe("");
    expect(t.rows[1]?.weaponAttack).toBe("5");
  });

  it("derives one column per distinct statup type, labelled and valued per level", () => {
    const effects: SkillEffect[] = [
      { statups: [{ type: "WeaponAttack", amount: 10 }] },
      {
        statups: [
          { type: "WeaponAttack", amount: 12 },
          { type: "Accuracy", amount: 3 },
        ],
      },
    ];
    const t = buildLevelTable(effects);
    const watk = t.columns.find((c) => c.key === "statup:WeaponAttack");
    const acc = t.columns.find((c) => c.key === "statup:Accuracy");
    expect(watk?.label).toBe("Weapon Attack"); // from reused STATUP_LABELS
    expect(acc?.label).toBe("Accuracy");
    expect(t.rows[0]?.["statup:WeaponAttack"]).toBe("10");
    expect(t.rows[0]?.["statup:Accuracy"]).toBe(""); // absent on level 1
    expect(t.rows[1]?.["statup:Accuracy"]).toBe("3");
  });

  it("falls back to the raw key for an unlabeled field", () => {
    const effects: SkillEffect[] = [{ morphId: 1000 }];
    const t = buildLevelTable(effects);
    const col = t.columns.find((c) => c.key === "morphId");
    expect(col?.label).toBe("Morph ID");
    const unknownStatup = buildLevelTable([
      { statups: [{ type: "Zzz", amount: 1 }] },
    ]);
    expect(
      unknownStatup.columns.find((c) => c.key === "statup:Zzz")?.label,
    ).toBe("Zzz");
  });
});
