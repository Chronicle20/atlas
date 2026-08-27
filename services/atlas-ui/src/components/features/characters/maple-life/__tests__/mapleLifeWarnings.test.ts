import { describe, expect, it } from "vitest";

import type { MapleLifeConfig } from "@/types/models/template";
import {
  initialMapleLifeState,
  mapleLifeReducer,
} from "@/components/features/characters/maple-life/mapleLifeEditorState";
import {
  WARN,
  warningMap,
} from "@/components/features/characters/maple-life/mapleLifeWarnings";

function classRow(
  overrides: Partial<MapleLifeConfig["classes"][number]> = {},
): MapleLifeConfig["classes"][number] {
  return {
    ordinal: 0,
    gender: 0,
    jobId: 100,
    level: 30,
    mapId: 102000000,
    stats: { str: 35, dex: 4, int: 4, luk: 4, hp: 804, mp: 150 },
    ap: 123,
    sp: "61,0,0,0,0,0,0,0,0,0",
    meso: 100000,
    equipment: [],
    inventory: [],
    ...overrides,
  };
}

function load(config: MapleLifeConfig | undefined) {
  return mapleLifeReducer(initialMapleLifeState(), { type: "load", config });
}

describe("warningMap", () => {
  it("warns on every present ordinal >= 2", () => {
    const s = load({
      looks: [],
      classes: [
        classRow({ ordinal: 1, gender: 0 }),
        classRow({ ordinal: 2, gender: 0 }),
        classRow({ ordinal: 4, gender: 0 }),
      ],
    });
    const map = warningMap(s);
    expect(map.get("classes.2.0")).toContain(WARN.unconfirmedOrdinal);
    expect(map.get("classes.4.0")).toContain(WARN.unconfirmedOrdinal);
    expect(map.get("classes.1.0")).toBeUndefined();
  });

  it("does not warn about an unconfirmed ordinal that is absent", () => {
    const s = load({
      looks: [],
      classes: [classRow({ ordinal: 0, gender: 0 })],
    });
    const messages = [...warningMap(s).values()].flat();
    expect(messages).not.toContain(WARN.unconfirmedOrdinal);
  });

  it("warns on an spSkillId outside the two known ids", () => {
    const s = load({
      looks: [],
      classes: [classRow({ spSkillId: 9999999 })],
    });
    expect(warningMap(s).get("classes.0.0.spSkillId")).toContain(
      WARN.unknownSpSkill,
    );
  });

  it("does not warn for either known id", () => {
    for (const spSkillId of [1000001, 2000001]) {
      const s = load({ looks: [], classes: [classRow({ spSkillId })] });
      const messages = [...warningMap(s).values()].flat();
      expect(messages).not.toContain(WARN.unknownSpSkill);
    }
  });

  it("does not warn when spSkillId is absent", () => {
    const s = load({ looks: [], classes: [classRow()] });
    const messages = [...warningMap(s).values()].flat();
    expect(messages).not.toContain(WARN.unknownSpSkill);
  });

  it("warns for every absent row", () => {
    const s = load(undefined);
    const absentMessages = [...warningMap(s).values()]
      .flat()
      .filter((m) => m === WARN.absentRow);
    expect(absentMessages).toHaveLength(10);
  });

  it("stops warning about a row once it is materialised", () => {
    let s = load(undefined);
    s = mapleLifeReducer(s, { type: "select", ordinal: 2, gender: 0 });
    s = mapleLifeReducer(s, {
      type: "setIdentity",
      field: "jobId",
      value: 300,
    });
    const messages = warningMap(s).get("classes.2.0") ?? [];
    expect(messages).not.toContain(WARN.absentRow);
    expect(messages).toContain(WARN.unconfirmedOrdinal);
  });
});
