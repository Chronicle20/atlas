import { describe, it, expect } from "vitest";
import { isFemaleCosmeticId } from "@/services/api/characterRender.service";
import { blankTemplate, normalizeTemplate, DEFAULT_PICKS } from "../editorState";
import {
  buildPreviewLoadout,
  buildVariantLoadout,
  RENDER_DEFAULT_FACE,
  RENDER_DEFAULT_HAIR,
  RENDER_DEFAULT_SKIN,
  EQUIP_SLOT_BY_POOL,
} from "../previewLoadout";

const tpl = normalizeTemplate({
  gender: 1,
  faces: [21000, 21001],
  hairs: [31000, 31030],
  hairColors: [0, 7],
  skinColors: [0, 3],
  tops: [1041002],
  bottoms: [1061002],
  shoes: [1072001],
  weapons: [1302000],
});

describe("buildPreviewLoadout", () => {
  it("uses picked pool entries; hair = base hair + color digit", () => {
    const lo = buildPreviewLoadout(tpl, {
      faceIdx: 1,
      hairIdx: 1,
      hairColorIdx: 1,
      skinIdx: 1,
    });
    expect(lo.face).toBe(21001);
    expect(lo.hair).toBe(31030 + 7);
    expect(lo.skin).toBe(3);
    expect(lo.gender).toBe(1);
  });

  it("maps first-of-pool equipment onto the render slots", () => {
    const lo = buildPreviewLoadout(tpl, DEFAULT_PICKS);
    expect(lo.equipment).toEqual({
      [EQUIP_SLOT_BY_POOL.tops]: 1041002,
      [EQUIP_SLOT_BY_POOL.bottoms]: 1061002,
      [EQUIP_SLOT_BY_POOL.shoes]: 1072001,
      [EQUIP_SLOT_BY_POOL.weapons]: 1302000,
    });
  });

  it("falls back to render defaults for empty pools (fresh + New template)", () => {
    const lo = buildPreviewLoadout(blankTemplate(), DEFAULT_PICKS);
    expect(lo).toEqual({
      skin: RENDER_DEFAULT_SKIN,
      hair: RENDER_DEFAULT_HAIR,
      face: RENDER_DEFAULT_FACE,
      equipment: {},
      gender: 0,
    });
  });

  it("clamps out-of-range picks instead of reading past the pool", () => {
    const lo = buildPreviewLoadout(tpl, {
      faceIdx: 9,
      hairIdx: 9,
      hairColorIdx: 9,
      skinIdx: 9,
    });
    expect(lo.face).toBe(21001);
  });
});

describe("buildVariantLoadout", () => {
  it("substitutes only the varied dimension", () => {
    expect(buildVariantLoadout(tpl, DEFAULT_PICKS, "faces", 22000).face).toBe(
      22000,
    );
    // hair candidate keeps the picked color digit (pick 0 → digit 0)
    expect(buildVariantLoadout(tpl, DEFAULT_PICKS, "hairs", 32000).hair).toBe(
      32000,
    );
    // hairColor candidate applies to the picked base hair
    expect(
      buildVariantLoadout(tpl, DEFAULT_PICKS, "hairColors", 5).hair,
    ).toBe(31000 + 5);
    expect(
      buildVariantLoadout(tpl, DEFAULT_PICKS, "skinColors", 2).skin,
    ).toBe(2);
  });
});

describe("isFemaleCosmeticId", () => {
  it("implements the v83 (id/1000)%10 === 1 convention", () => {
    expect(isFemaleCosmeticId(21000)).toBe(true); // female face
    expect(isFemaleCosmeticId(20000)).toBe(false); // male face
    expect(isFemaleCosmeticId(31000)).toBe(true); // female hair
    expect(isFemaleCosmeticId(30030)).toBe(false); // male hair
    expect(isFemaleCosmeticId(0)).toBe(false);
  });
});
