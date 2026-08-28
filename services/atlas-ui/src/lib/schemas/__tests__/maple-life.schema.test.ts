import { describe, it, expect } from "vitest";
import { validateMapleLife } from "@/lib/schemas/maple-life.schema";
import type { MapleLifeConfig } from "@/types/models/template";

/**
 * Derived from the canonical shipped-seed row (`SEED_ML` in
 * `src/services/api/__tests__/tenants.service.test.ts`), trimmed to a single
 * element per look dimension and with equipment/inventory emptied for
 * brevity.
 */
function valid(): MapleLifeConfig {
  return {
    looks: [
      {
        gender: 0,
        faces: [20000],
        hairs: [30030],
        hairColors: [0],
        skinColors: [0],
      },
    ],
    classes: [
      {
        ordinal: 0,
        gender: 0,
        jobId: 100,
        level: 30,
        mapId: 102000000,
        stats: { str: 35, dex: 4, int: 4, luk: 4, hp: 804, mp: 150 },
        ap: 123,
        sp: "61,0,0,0,0,0,0,0,0,0",
        spSkillId: 1000001,
        meso: 100000,
        equipment: [],
        inventory: [],
      },
    ],
  };
}

describe("mapleLifeSchema / validateMapleLife", () => {
  it("the seed fixture validates clean", () => {
    expect(validateMapleLife(valid()).size).toBe(0);
  });

  it("an empty configuration validates clean", () => {
    expect(validateMapleLife({ looks: [], classes: [] }).size).toBe(0);
  });

  it("issues accumulate per path", () => {
    const cfg = valid();
    cfg.looks[0]!.hairs = [30035, 30047];
    const issues = validateMapleLife(cfg);
    expect(issues.get("looks.0.hairs.0")).toHaveLength(1);
    expect(issues.get("looks.0.hairs.1")).toHaveLength(1);
  });

  describe("FR-11.1 hair style not divisible by 10", () => {
    it("flags a non-multiple-of-10 hair id", () => {
      const cfg = valid();
      cfg.looks[0]!.hairs = [30035];
      expect([...validateMapleLife(cfg).keys()]).toContain("looks.0.hairs.0");
    });

    it("passes a multiple-of-10 hair id", () => {
      const cfg = valid();
      cfg.looks[0]!.hairs = [30030];
      expect([...validateMapleLife(cfg).keys()]).not.toContain(
        "looks.0.hairs.0",
      );
    });
  });

  describe("FR-11.2 hair colour outside 0..9", () => {
    it("flags a colour above 9", () => {
      const cfg = valid();
      cfg.looks[0]!.hairColors = [10];
      expect([...validateMapleLife(cfg).keys()]).toContain(
        "looks.0.hairColors.0",
      );
    });

    it("passes the upper bound 9", () => {
      const cfg = valid();
      cfg.looks[0]!.hairColors = [9];
      expect([...validateMapleLife(cfg).keys()]).not.toContain(
        "looks.0.hairColors.0",
      );
    });

    it("flags a colour below 0", () => {
      const cfg = valid();
      cfg.looks[0]!.hairColors = [-1];
      expect([...validateMapleLife(cfg).keys()]).toContain(
        "looks.0.hairColors.0",
      );
    });

    it("passes the lower bound 0", () => {
      const cfg = valid();
      cfg.looks[0]!.hairColors = [0];
      expect([...validateMapleLife(cfg).keys()]).not.toContain(
        "looks.0.hairColors.0",
      );
    });
  });

  describe("FR-11.5 sp not ten integers", () => {
    it("flags a pool that is too short", () => {
      const cfg = valid();
      cfg.classes[0]!.sp = "61,0";
      expect([...validateMapleLife(cfg).keys()]).toContain("classes.0.sp");
    });

    it("flags a pool that is too long", () => {
      const cfg = valid();
      cfg.classes[0]!.sp = "0,0,0,0,0,0,0,0,0,0,0";
      expect([...validateMapleLife(cfg).keys()]).toContain("classes.0.sp");
    });

    it("flags a pool with a non-numeric entry", () => {
      const cfg = valid();
      cfg.classes[0]!.sp = "61,x,0,0,0,0,0,0,0,0";
      expect([...validateMapleLife(cfg).keys()]).toContain("classes.0.sp");
    });

    it("passes a well-formed ten-book pool", () => {
      const cfg = valid();
      cfg.classes[0]!.sp = "61,0,0,0,0,0,0,0,0,0";
      expect([...validateMapleLife(cfg).keys()]).not.toContain("classes.0.sp");
    });
  });

  describe("FR-11.4 non-zero spSkillId on ordinal >= 2", () => {
    it("flags a set spSkillId on ordinal 2", () => {
      const cfg = valid();
      cfg.classes[0]!.ordinal = 2;
      cfg.classes[0]!.gender = 0;
      cfg.classes[0]!.spSkillId = 1000001;
      expect([...validateMapleLife(cfg).keys()]).toContain(
        "classes.0.spSkillId",
      );
    });

    it("passes ordinal 2 with spSkillId absent", () => {
      const cfg = valid();
      cfg.classes[0]!.ordinal = 2;
      delete cfg.classes[0]!.spSkillId;
      expect([...validateMapleLife(cfg).keys()]).not.toContain(
        "classes.0.spSkillId",
      );
    });
  });

  describe("FR-11.6 spPool[0] < 6 while spSkillId is set", () => {
    it("flags a book-0 investment below 6", () => {
      const cfg = valid();
      cfg.classes[0]!.sp = "5,0,0,0,0,0,0,0,0,0";
      expect([...validateMapleLife(cfg).keys()]).toContain("classes.0.sp");
    });

    it("passes a book-0 investment of exactly 6", () => {
      const cfg = valid();
      cfg.classes[0]!.sp = "6,0,0,0,0,0,0,0,0,0";
      expect([...validateMapleLife(cfg).keys()]).not.toContain("classes.0.sp");
    });

    it("does not fire when no skill is set", () => {
      const cfg = valid();
      cfg.classes[0]!.sp = "5,0,0,0,0,0,0,0,0,0";
      delete cfg.classes[0]!.spSkillId;
      expect([...validateMapleLife(cfg).keys()]).not.toContain("classes.0.sp");
    });
  });

  describe("FR-11.3 empty pool for a gender with class rows", () => {
    it("flags an emptied faces pool", () => {
      const cfg = valid();
      cfg.looks[0]!.faces = [];
      expect([...validateMapleLife(cfg).keys()]).toContain("looks.0.faces");
    });

    it("passes a populated faces pool", () => {
      const cfg = valid();
      cfg.looks[0]!.faces = [20000];
      expect([...validateMapleLife(cfg).keys()]).not.toContain("looks.0.faces");
    });

    it("flags each of the four dimensions emptied in turn", () => {
      const dims = ["faces", "hairs", "hairColors", "skinColors"] as const;
      for (const dim of dims) {
        const cfg = valid();
        cfg.looks[0]![dim] = [];
        expect([...validateMapleLife(cfg).keys()]).toContain(`looks.0.${dim}`);
      }
    });

    it("is harmless when there is no class row for that gender", () => {
      const cfg = valid();
      cfg.classes = [];
      cfg.looks[0]!.faces = [];
      expect([...validateMapleLife(cfg).keys()]).not.toContain("looks.0.faces");
    });
  });

  describe("FR-11.7 gender with class rows but no looks row", () => {
    it("flags a gender-1 class row with no gender-1 look row", () => {
      const cfg = valid();
      cfg.classes.push({ ...cfg.classes[0]!, ordinal: 1, gender: 1 });
      expect([...validateMapleLife(cfg).keys()]).toContain("looks");
    });

    it("passes once a gender-1 look row is added", () => {
      const cfg = valid();
      cfg.classes.push({ ...cfg.classes[0]!, ordinal: 1, gender: 1 });
      cfg.looks.push({
        gender: 1,
        faces: [20100],
        hairs: [30000],
        hairColors: [0],
        skinColors: [0],
      });
      expect([...validateMapleLife(cfg).keys()]).not.toContain("looks");
    });
  });
});
