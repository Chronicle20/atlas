import { describe, it, expect } from "vitest";
import {
  buildPresetLoadout,
  buildPresetVariantLoadout,
  wornTemplateIds,
} from "../presetLoadout";
import { DEFAULT_PRESET_ATTRIBUTES } from "../presetEditorState";
import {
  RENDER_DEFAULT_HAIR,
  RENDER_DEFAULT_FACE,
  RENDER_DEFAULT_SKIN,
} from "../../templates/previewLoadout";

const attrs = (o: Partial<typeof DEFAULT_PRESET_ATTRIBUTES>) => ({
  ...DEFAULT_PRESET_ATTRIBUTES,
  ...o,
});

describe("presetLoadout", () => {
  it("composites hair as base + color and passes gender/face/skin", () => {
    const l = buildPresetLoadout(
      attrs({ hair: 30030, hairColor: 2, face: 20001, skinColor: 3, gender: 1 }),
    );
    expect(l.hair).toBe(30032);
    expect(l.face).toBe(20001);
    expect(l.skin).toBe(3);
    expect(l.gender).toBe(1);
  });

  it("places worn equipment on canonical slots and drops unplaceable ids", () => {
    const l = buildPresetLoadout(
      attrs({
        equipment: [
          { templateId: 1040002, useAverageStats: true }, // top -> -5
          { templateId: 1302000, useAverageStats: true }, // weapon -> -11
          { templateId: 2000000, useAverageStats: true }, // use item -> null, skipped
        ],
      }),
    );
    expect(l.equipment["-5"]).toBe(1040002);
    expect(l.equipment["-11"]).toBe(1302000);
    expect(Object.values(l.equipment)).not.toContain(2000000);
  });

  it("later same-slot item overwrites earlier (map semantics)", () => {
    const l = buildPresetLoadout(
      attrs({
        equipment: [
          { templateId: 1040002, useAverageStats: true }, // top -5
          { templateId: 1040010, useAverageStats: true }, // top -5 (wins)
        ],
      }),
    );
    expect(l.equipment["-5"]).toBe(1040010);
  });

  it("empty loadout falls back to render defaults", () => {
    const l = buildPresetLoadout(attrs({ hair: 0, hairColor: 0, face: 0, skinColor: 0 }));
    expect(l.hair).toBe(RENDER_DEFAULT_HAIR);
    expect(l.face).toBe(RENDER_DEFAULT_FACE);
    expect(l.skin).toBe(RENDER_DEFAULT_SKIN);
  });

  it("variant loadout substitutes one dimension", () => {
    const base = attrs({ hair: 30030, hairColor: 1, face: 20000, skinColor: 0 });
    expect(buildPresetVariantLoadout(base, "faces", 21000).face).toBe(21000);
    expect(buildPresetVariantLoadout(base, "hairs", 30040).hair).toBe(30041);
    expect(buildPresetVariantLoadout(base, "hairColors", 5).hair).toBe(30035);
    expect(buildPresetVariantLoadout(base, "skinColors", 4).skin).toBe(4);
  });

  it("wornTemplateIds lists placeable ids only", () => {
    expect(
      wornTemplateIds(
        attrs({
          equipment: [
            { templateId: 1040002, useAverageStats: true },
            { templateId: 2000000, useAverageStats: true },
          ],
        }),
      ),
    ).toEqual([1040002]);
  });
});
