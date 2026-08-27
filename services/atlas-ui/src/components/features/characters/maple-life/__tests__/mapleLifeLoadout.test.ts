import { describe, expect, it } from "vitest";

import type {
  LookDimension,
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
import type { CharacterLoadout } from "@/services/api/characterRender.service";

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

interface VariantDimensionCase {
  dimension: LookDimension;
  field: keyof CharacterLoadout;
  look: MapleLifeLookDraft;
  picks: PreviewPicks;
  candidateId: number;
  expectedValue: number;
}

const VARIANT_DIMENSION_CASES: VariantDimensionCase[] = [
  {
    dimension: "faces",
    field: "face",
    look: look(),
    picks: picks(),
    candidateId: 21000,
    expectedValue: 21000,
  },
  {
    dimension: "hairs",
    field: "hair",
    look: look({ hairColors: [0, 7] }),
    picks: picks({ hairColorIdx: 1 }),
    candidateId: 30020,
    expectedValue: composeHair(30020, 7),
  },
  {
    dimension: "hairColors",
    field: "hair",
    look: look({ hairs: [30030] }),
    picks: picks({ hairIdx: 0 }),
    candidateId: 3,
    expectedValue: composeHair(30030, 3),
  },
  {
    dimension: "skinColors",
    field: "skin",
    look: look(),
    picks: picks(),
    candidateId: 2,
    expectedValue: 2,
  },
];

describe("buildMapleLifeVariantLoadout", () => {
  it.each(VARIANT_DIMENSION_CASES)(
    "$dimension substitutes the candidate on $field and preserves every other field from buildMapleLifeLoadout",
    ({
      dimension,
      field,
      look: lookDraft,
      picks: pickSet,
      candidateId,
      expectedValue,
    }) => {
      const classDraft = draft();
      const base = buildMapleLifeLoadout(classDraft, lookDraft, pickSet);
      const variant = buildMapleLifeVariantLoadout(
        classDraft,
        lookDraft,
        pickSet,
        dimension,
        candidateId,
      );

      expect(variant[field]).toBe(expectedValue);

      const otherFields = (
        Object.keys(base) as (keyof CharacterLoadout)[]
      ).filter((key) => key !== field);
      for (const key of otherFields) {
        expect(variant[key]).toEqual(base[key]);
      }
    },
  );
});

describe("combinationCount", () => {
  it("multiplies the four pools", () => {
    expect(combinationCount(look())).toBe(144);
  });

  it("is zero when any pool is empty", () => {
    expect(combinationCount(look({ faces: [] }))).toBe(0);
  });
});
