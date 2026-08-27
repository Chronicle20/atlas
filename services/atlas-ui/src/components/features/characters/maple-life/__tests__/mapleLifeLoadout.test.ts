import { describe, expect, it } from "vitest";

import type {
  MapleLifeClassDraft,
  MapleLifeLookDraft,
  PreviewPicks,
} from "@/components/features/characters/maple-life/mapleLifeEditorState";
import { DEFAULT_PICKS } from "@/components/features/characters/maple-life/mapleLifeEditorState";
import {
  buildMapleLifeLoadout,
  buildMapleLifeVariantLoadout,
  combinationCount,
  composeHair,
} from "@/components/features/characters/maple-life/mapleLifeLoadout";

function draft(
  overrides: Partial<MapleLifeClassDraft> = {},
): MapleLifeClassDraft {
  return {
    ordinal: 0,
    gender: 0,
    jobId: 100,
    level: 30,
    mapId: 102000000,
    stats: { str: 35, dex: 4, int: 4, luk: 4, hp: 804, mp: 150 },
    ap: 123,
    spBooks: [61, 0, 0, 0, 0, 0, 0, 0, 0, 0],
    spRaw: "61,0,0,0,0,0,0,0,0,0",
    spSkillId: 1000001,
    meso: 100000,
    equipment: [],
    inventory: [],
    present: true,
    ...overrides,
  };
}

function look(overrides: Partial<MapleLifeLookDraft> = {}): MapleLifeLookDraft {
  return {
    gender: 0,
    faces: [20000, 20001, 20002],
    hairs: [30030, 30020, 30000],
    hairColors: [0, 7, 3, 2],
    skinColors: [0, 1, 2, 3],
    present: true,
    ...overrides,
  };
}

function picks(overrides: Partial<PreviewPicks> = {}): PreviewPicks {
  return { ...DEFAULT_PICKS, ...overrides };
}

describe("composeHair", () => {
  it.each([
    ["already normalised", 30030, 0, 30030],
    ["normalised + colour", 30030, 7, 30037],
    ["non-multiple-of-ten base normalises", 30035, 2, 30032],
    ["non-multiple-of-ten, colour 0", 30037, 0, 30030],
    ["low id", 0, 3, 3],
  ])("%s", (_label, hairStyle, hairColor, expected) => {
    expect(composeHair(hairStyle, hairColor)).toBe(expected);
  });
});

describe("buildMapleLifeLoadout", () => {
  it("passes gender explicitly from the look row", () => {
    const loadout = buildMapleLifeLoadout(
      draft(),
      look({ gender: 1 }),
      picks(),
    );
    expect(loadout.gender).toBe(1);
  });

  it("composes hair from the picked style and colour", () => {
    const loadout = buildMapleLifeLoadout(
      draft(),
      look({ hairs: [30030], hairColors: [0, 7] }),
      picks({ hairIdx: 0, hairColorIdx: 1 }),
    );
    expect(loadout.hair).toBe(30037);
  });

  it("places the first four equips on the canonical slots", () => {
    const loadout = buildMapleLifeLoadout(
      draft({
        equipment: [
          { templateId: 1040021, useAverageStats: true },
          { templateId: 1060016, useAverageStats: true },
          { templateId: 1072039, useAverageStats: true },
          { templateId: 1302008, useAverageStats: true },
          { templateId: 1442001, useAverageStats: true },
        ],
      }),
      look(),
      picks(),
    );
    expect(loadout.equipment).toEqual({
      "-5": 1040021,
      "-6": 1060016,
      "-7": 1072039,
      "-11": 1302008,
    });
  });

  it("falls back to render defaults for an empty pool", () => {
    const loadout = buildMapleLifeLoadout(
      draft(),
      look({ faces: [], hairs: [], hairColors: [], skinColors: [] }),
      picks(),
    );
    expect(loadout.skin).toBe(0);
    expect(loadout.hair).toBe(30030);
    expect(loadout.face).toBe(20000);
  });

  it("clamps an out-of-range pick", () => {
    const loadout = buildMapleLifeLoadout(
      draft(),
      look({ faces: [20000, 20001] }),
      picks({ faceIdx: 9 }),
    );
    expect(loadout.face).toBe(20001);
  });
});

describe("buildMapleLifeVariantLoadout", () => {
  it("faces substitutes the candidate", () => {
    const loadout = buildMapleLifeVariantLoadout(
      draft(),
      look(),
      picks(),
      "faces",
      21000,
    );
    expect(loadout.face).toBe(21000);
  });

  it("hairs composes the candidate with the picked colour", () => {
    const loadout = buildMapleLifeVariantLoadout(
      draft(),
      look({ hairColors: [0, 7] }),
      picks({ hairColorIdx: 1 }),
      "hairs",
      30020,
    );
    expect(loadout.hair).toBe(30027);
  });

  it("hairColors composes the picked style with the candidate", () => {
    const loadout = buildMapleLifeVariantLoadout(
      draft(),
      look({ hairs: [30030] }),
      picks({ hairIdx: 0 }),
      "hairColors",
      3,
    );
    expect(loadout.hair).toBe(30033);
  });

  it("skinColors substitutes the candidate", () => {
    const loadout = buildMapleLifeVariantLoadout(
      draft(),
      look(),
      picks(),
      "skinColors",
      2,
    );
    expect(loadout.skin).toBe(2);
  });
});

describe("combinationCount", () => {
  it("multiplies the four pools", () => {
    expect(combinationCount(look())).toBe(144);
  });

  it("is zero when any pool is empty", () => {
    expect(combinationCount(look({ faces: [] }))).toBe(0);
  });
});
