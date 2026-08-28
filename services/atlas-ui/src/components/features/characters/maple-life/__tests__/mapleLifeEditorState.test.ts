import { describe, expect, it } from "vitest";

import type { MapleLifeConfig } from "@/types/models/template";
import {
  draftIndex,
  initialMapleLifeState,
  isDirty,
  isEmptyConfig,
  mapleLifeReducer,
  parseSpPool,
  picksFor,
  projectForSave,
} from "@/components/features/characters/maple-life/mapleLifeEditorState";

// Shipped seed row, reproduced verbatim from
// services/atlas-configurations/seed-data/templates/template_gms_83_1.json
// `mapleLife.looks[0]` and `mapleLife.classes[0]`, extended per the brief
// with a second (synthetic) class row exercising the omitempty `spSkillId`
// case and a second look row.
const SEED: MapleLifeConfig = {
  looks: [
    {
      gender: 0,
      faces: [20000, 20001, 20002],
      hairs: [30030, 30020, 30000],
      hairColors: [0, 7, 3, 2],
      skinColors: [0, 1, 2, 3],
    },
    {
      gender: 1,
      faces: [21000],
      hairs: [31050],
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
      equipment: [
        { templateId: 1040021, useAverageStats: true },
        { templateId: 1060016, useAverageStats: true },
        { templateId: 1072039, useAverageStats: true },
        { templateId: 1302008, useAverageStats: true },
        { templateId: 1442001, useAverageStats: true },
        { templateId: 1422001, useAverageStats: true },
        { templateId: 1312005, useAverageStats: true },
      ],
      inventory: [
        { templateId: 2000002, quantity: 100 },
        { templateId: 2000006, quantity: 100 },
        { templateId: 3010000, quantity: 1 },
      ],
    },
    {
      ordinal: 3,
      gender: 1,
      jobId: 400,
      level: 30,
      mapId: 103000000,
      stats: { str: 4, dex: 25, int: 4, luk: 20, hp: 520, mp: 130 },
      ap: 121,
      sp: "61,0,0,0,0,0,0,0,0,0",
      meso: 100000,
      equipment: [],
      inventory: [],
    },
  ],
};

describe("mapleLifeReducer", () => {
  it("load expands a sparse config into ten ordinal-major slots", () => {
    const s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config: SEED,
    });
    expect(s.drafts).toHaveLength(10);
    expect(s.drafts[0]?.ordinal).toBe(0);
    expect(s.drafts[0]?.gender).toBe(0);
    expect(s.drafts[9]?.ordinal).toBe(4);
    expect(s.drafts[9]?.gender).toBe(1);
  });

  it("load marks presence only for rows the config carries", () => {
    const s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config: SEED,
    });
    expect(
      s.drafts.filter((d) => d.present).map((d) => [d.ordinal, d.gender]),
    ).toEqual([
      [0, 0],
      [3, 1],
    ]);
  });

  it("load parses sp into ten books", () => {
    const s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config: SEED,
    });
    expect(s.drafts[0]?.spBooks).toEqual([61, 0, 0, 0, 0, 0, 0, 0, 0, 0]);
    expect(s.drafts[0]?.spRaw).toBe("61,0,0,0,0,0,0,0,0,0");
  });

  it("load leaves spBooks empty for an unparseable sp", () => {
    const config: MapleLifeConfig = {
      looks: [],
      classes: [{ ...SEED.classes[0]!, sp: "61,0" }],
    };
    const s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config,
    });
    expect(s.drafts[0]?.spBooks).toEqual([]);
    expect(s.drafts[0]?.spRaw).toBe("61,0");
  });

  it("load with undefined yields ten absent slots and two absent looks", () => {
    const s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config: undefined,
    });
    expect(s.drafts.every((d) => d.present === false)).toBe(true);
    expect(s.looks[0]?.present).toBe(false);
    expect(s.looks[1]?.present).toBe(false);
    expect(s.loaded).toBe(true);
  });

  it("load tolerates null looks/classes (the shape the API sends for an unconfigured version)", () => {
    // services/atlas-configurations/tenants/rest.go and templates/rest.go
    // always emit a `mapleLife` object; a version with no block still
    // carries `{"looks":null,"classes":null}`, cast through the wire shape
    // here since MapleLifeConfig's TS type says non-null arrays.
    const nullConfig = {
      looks: null,
      classes: null,
    } as unknown as MapleLifeConfig;
    expect(() =>
      mapleLifeReducer(initialMapleLifeState(), {
        type: "load",
        config: nullConfig,
      }),
    ).not.toThrow();
    const s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config: nullConfig,
    });
    expect(s.drafts).toHaveLength(10);
    expect(s.drafts.every((d) => d.present === false)).toBe(true);
    expect(isEmptyConfig(nullConfig)).toBe(true);
  });

  it("load marks look presence per gender", () => {
    const s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config: SEED,
    });
    expect(s.looks[0]?.present).toBe(true);
    expect(s.looks[1]?.present).toBe(true);

    const genderZeroOnly: MapleLifeConfig = {
      looks: [SEED.looks[0]!],
      classes: SEED.classes,
    };
    const s2 = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config: genderZeroOnly,
    });
    expect(s2.looks[1]?.present).toBe(false);
  });
});

describe("projectForSave", () => {
  it("emits only present rows, ordinal-major", () => {
    const s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config: SEED,
    });
    expect(projectForSave(s).classes.map((c) => [c.ordinal, c.gender])).toEqual(
      [
        [0, 0],
        [3, 1],
      ],
    );
  });

  it("an untouched load round-trips to the loaded value", () => {
    const s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config: SEED,
    });
    expect(projectForSave(s)).toEqual(SEED);
  });

  it("omits spSkillId entirely when the draft has none", () => {
    const s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config: SEED,
    });
    expect(Object.hasOwn(projectForSave(s).classes[1]!, "spSkillId")).toBe(
      false,
    );
  });

  it("emits an unparseable sp verbatim", () => {
    const config: MapleLifeConfig = {
      looks: [],
      classes: [{ ...SEED.classes[0]!, sp: "61,0" }],
    };
    const s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config,
    });
    expect(projectForSave(s).classes[0]?.sp).toBe("61,0");
  });

  it("re-serialises sp from the books after an edit", () => {
    let s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config: SEED,
    });
    s = mapleLifeReducer(s, { type: "select", ordinal: 0, gender: 0 });
    s = mapleLifeReducer(s, { type: "setSpBook", index: 0, value: 75 });
    expect(projectForSave(s).classes[0]?.sp).toBe("75,0,0,0,0,0,0,0,0,0");
  });

  it("emits only present look rows", () => {
    const config: MapleLifeConfig = {
      looks: [SEED.looks[0]!],
      classes: SEED.classes,
    };
    const s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config,
    });
    expect(projectForSave(s).looks).toHaveLength(1);
    expect(projectForSave(s).looks[0]?.gender).toBe(0);
  });
});

describe("materialisation", () => {
  it("editing an absent row materialises it", () => {
    let s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config: SEED,
    });
    s = mapleLifeReducer(s, { type: "select", ordinal: 2, gender: 0 });
    s = mapleLifeReducer(s, {
      type: "setIdentity",
      field: "jobId",
      value: 300,
    });
    expect(s.drafts[draftIndex(2, 0)]?.present).toBe(true);
    const classes = projectForSave(s).classes;
    expect(classes).toHaveLength(3);
    const row = classes.find((c) => c.ordinal === 2 && c.gender === 0);
    expect(row?.jobId).toBe(300);
  });

  it("adding a look entry materialises that gender's look row", () => {
    const config: MapleLifeConfig = {
      looks: [SEED.looks[0]!],
      classes: SEED.classes,
    };
    let s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config,
    });
    s = mapleLifeReducer(s, { type: "select", ordinal: 0, gender: 1 });
    s = mapleLifeReducer(s, {
      type: "addLookEntry",
      dimension: "faces",
      id: 21000,
    });
    expect(s.looks[1]?.present).toBe(true);
    expect(projectForSave(s).looks).toHaveLength(2);
  });

  it("materialiseAll marks all ten rows and both looks present", () => {
    let s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config: undefined,
    });
    s = mapleLifeReducer(s, { type: "materialiseAll" });
    expect(projectForSave(s).classes).toHaveLength(10);
    expect(projectForSave(s).looks).toHaveLength(2);
  });

  it("selecting a row does NOT materialise it", () => {
    let s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config: SEED,
    });
    s = mapleLifeReducer(s, { type: "select", ordinal: 2, gender: 0 });
    expect(projectForSave(s).classes).toHaveLength(2);
  });
});

describe("dirty / discard / savedOk", () => {
  it("a fresh load is not dirty", () => {
    const s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config: SEED,
    });
    expect(isDirty(s)).toBe(false);
  });

  it("any field edit is dirty", () => {
    let s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config: SEED,
    });
    s = mapleLifeReducer(s, { type: "select", ordinal: 0, gender: 0 });
    s = mapleLifeReducer(s, { type: "setScalar", field: "ap", value: 124 });
    expect(isDirty(s)).toBe(true);
  });

  it("a selection change alone is not dirty", () => {
    let s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config: SEED,
    });
    s = mapleLifeReducer(s, { type: "select", ordinal: 4, gender: 1 });
    s = mapleLifeReducer(s, {
      type: "setPreviewPick",
      pick: "faceIdx",
      value: 2,
    });
    expect(isDirty(s)).toBe(false);
  });

  it("discard restores the baseline", () => {
    let s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config: SEED,
    });
    s = mapleLifeReducer(s, { type: "select", ordinal: 0, gender: 0 });
    s = mapleLifeReducer(s, { type: "setScalar", field: "ap", value: 124 });
    s = mapleLifeReducer(s, { type: "discard" });
    expect(isDirty(s)).toBe(false);
    expect(projectForSave(s)).toEqual(SEED);
  });

  it("savedOk rebases so the edit is no longer dirty", () => {
    let s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config: SEED,
    });
    s = mapleLifeReducer(s, { type: "select", ordinal: 0, gender: 0 });
    s = mapleLifeReducer(s, { type: "setScalar", field: "ap", value: 124 });
    s = mapleLifeReducer(s, { type: "savedOk" });
    expect(isDirty(s)).toBe(false);
    expect(projectForSave(s).classes[0]?.ap).toBe(124);
  });

  it("seedFromTemplate replaces the working copy and reads dirty", () => {
    let s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config: undefined,
    });
    s = mapleLifeReducer(s, { type: "seedFromTemplate", config: SEED });
    expect(isDirty(s)).toBe(true);
    expect(projectForSave(s)).toEqual(SEED);
  });

  it("seedFromTemplate deep-clones the donor", () => {
    const originalLength = SEED.classes[0]!.equipment.length;
    let s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config: undefined,
    });
    s = mapleLifeReducer(s, { type: "seedFromTemplate", config: SEED });
    s = mapleLifeReducer(s, { type: "select", ordinal: 0, gender: 0 });
    s = mapleLifeReducer(s, { type: "addEquipment", templateId: 1302000 });
    expect(projectForSave(s).classes[0]?.equipment).toHaveLength(
      originalLength + 1,
    );
    expect(SEED.classes[0]!.equipment).toHaveLength(originalLength);
  });
});

describe("helpers", () => {
  it("draftIndex is ordinal-major", () => {
    expect(draftIndex(0, 0)).toBe(0);
    expect(draftIndex(0, 1)).toBe(1);
    expect(draftIndex(4, 1)).toBe(9);
  });

  it("parseSpPool accepts exactly ten integers", () => {
    expect(parseSpPool("61,0,0,0,0,0,0,0,0,0")).toEqual([
      61, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    ]);
  });

  it("parseSpPool rejects a nine-book pool", () => {
    expect(parseSpPool("1,2,3,4,5,6,7,8,9")).toEqual([]);
  });

  it("parseSpPool rejects an eleven-book pool", () => {
    expect(parseSpPool("0,0,0,0,0,0,0,0,0,0,0")).toEqual([]);
  });

  it("parseSpPool rejects a non-numeric book", () => {
    expect(parseSpPool("61,x,0,0,0,0,0,0,0,0")).toEqual([]);
  });

  it("isEmptyConfig", () => {
    expect(isEmptyConfig(undefined)).toBe(true);
    expect(isEmptyConfig({ looks: [], classes: [] })).toBe(true);
    expect(isEmptyConfig(SEED)).toBe(false);
  });

  it("picks are gender-split", () => {
    let s = mapleLifeReducer(initialMapleLifeState(), {
      type: "load",
      config: SEED,
    });
    s = mapleLifeReducer(s, { type: "select", ordinal: 0, gender: 0 });
    s = mapleLifeReducer(s, {
      type: "setPreviewPick",
      pick: "faceIdx",
      value: 2,
    });
    s = mapleLifeReducer(s, { type: "select", ordinal: 0, gender: 1 });
    expect(picksFor(s, 1).faceIdx).toBe(0);
    expect(picksFor(s, 0).faceIdx).toBe(2);
  });
});
