import { describe, it, expect } from "vitest";
import {
  DEFAULT_PRESET_ATTRIBUTES,
  initialPresetEditorState,
  presetReducer,
  isDirty,
  projectForSave,
  presetDirty,
  normalizePreset,
  type PresetEditorState,
} from "../presetEditorState";
import type { CharacterPreset } from "@/types/models/template";

const preset = (id: string | undefined, name: string): CharacterPreset => ({
  ...(id !== undefined ? { id } : {}),
  attributes: { ...DEFAULT_PRESET_ATTRIBUTES, name },
});

function loaded(presets: CharacterPreset[]): PresetEditorState {
  return presetReducer(initialPresetEditorState(), { type: "load", presets });
}

describe("presetEditorState", () => {
  it("load assigns a stable key (id when present, local-<n> otherwise) and is not dirty", () => {
    const s = loaded([preset("a1", "One"), preset(undefined, "Two")]);
    expect(s.presets[0]!.key).toBe("a1");
    expect(s.presets[1]!.key).toBe("local-0");
    expect(s.loaded).toBe(true);
    expect(s.selectedKey).toBeNull();
    expect(isDirty(s)).toBe(false);
  });

  it("dirty compare ignores the UI-only key", () => {
    let s = loaded([preset("a1", "One")]);
    // mutate key directly to prove it is excluded
    s = { ...s, presets: [{ ...s.presets[0]!, key: "different" }] };
    expect(isDirty(s)).toBe(false);
  });

  it("projectForSave emits { id?, attributes } with no key", () => {
    const s = loaded([preset("a1", "One"), preset(undefined, "Two")]);
    const out = projectForSave(s);
    expect(out).toEqual([
      { id: "a1", attributes: { ...DEFAULT_PRESET_ATTRIBUTES, name: "One" } },
      { attributes: { ...DEFAULT_PRESET_ATTRIBUTES, name: "Two" } },
    ]);
    expect(Object.keys(out[0]!)).not.toContain("key");
  });

  it("addPreset appends defaults, selects the new key, marks dirty", () => {
    let s = loaded([preset("a1", "One")]);
    s = presetReducer(s, { type: "addPreset" });
    expect(s.presets).toHaveLength(2);
    expect(s.presets[1]!.attributes.name).toBe("New preset");
    expect(s.selectedKey).toBe(s.presets[1]!.key);
    expect(s.presets[1]!.key).toMatch(/^local-/);
    expect(isDirty(s)).toBe(true);
  });

  it("duplicatePreset deep-copies, gives a fresh key/no id, selects it", () => {
    let s = loaded([preset("a1", "One")]);
    s = presetReducer(s, { type: "duplicatePreset", key: "a1" });
    expect(s.presets).toHaveLength(2);
    expect(s.presets[1]!.id).toBeUndefined();
    expect(s.presets[1]!.key).not.toBe("a1");
    expect(s.presets[1]!.attributes.name).toBe("One");
    // deep copy: mutating source arrays does not touch the copy
    s.presets[0]!.attributes.tags.push("x");
    expect(s.presets[1]!.attributes.tags).toEqual([]);
    expect(s.selectedKey).toBe(s.presets[1]!.key);
  });

  it("removePreset drops the row and clears selection to library", () => {
    let s = loaded([preset("a1", "One"), preset("b2", "Two")]);
    s = presetReducer(s, { type: "select", key: "a1" });
    s = presetReducer(s, { type: "removePreset", key: "a1" });
    expect(s.presets.map((p) => p.id)).toEqual(["b2"]);
    expect(s.selectedKey).toBeNull();
  });

  it("setField writes appearance/stat/identity directly to attributes", () => {
    let s = loaded([preset("a1", "One")]);
    s = presetReducer(s, { type: "select", key: "a1" });
    s = presetReducer(s, {
      type: "setField",
      key: "a1",
      path: "face",
      value: 21000,
    });
    s = presetReducer(s, {
      type: "setField",
      key: "a1",
      path: "gender",
      value: 1,
    });
    s = presetReducer(s, {
      type: "setField",
      key: "a1",
      path: "stats.str",
      value: 12,
    });
    expect(s.presets[0]!.attributes.face).toBe(21000);
    expect(s.presets[0]!.attributes.gender).toBe(1);
    expect(s.presets[0]!.attributes.stats.str).toBe(12);
    expect(isDirty(s)).toBe(true);
  });

  it("equipment add/remove/avg edits the working preset", () => {
    let s = loaded([preset("a1", "One")]);
    s = presetReducer(s, { type: "addEquip", key: "a1", templateId: 1040002 });
    expect(s.presets[0]!.attributes.equipment).toEqual([
      { templateId: 1040002, useAverageStats: true },
    ]);
    s = presetReducer(s, {
      type: "setEquipAvg",
      key: "a1",
      index: 0,
      value: false,
    });
    expect(s.presets[0]!.attributes.equipment[0]!.useAverageStats).toBe(false);
    s = presetReducer(s, { type: "removeEquip", key: "a1", index: 0 });
    expect(s.presets[0]!.attributes.equipment).toEqual([]);
  });

  it("inventory + skills add/remove/qty/level edit the working preset", () => {
    let s = loaded([preset("a1", "One")]);
    s = presetReducer(s, {
      type: "addInventory",
      key: "a1",
      templateId: 2000000,
    });
    s = presetReducer(s, {
      type: "setInventoryQty",
      key: "a1",
      index: 0,
      value: 5,
    });
    expect(s.presets[0]!.attributes.inventory).toEqual([
      { templateId: 2000000, quantity: 5 },
    ]);
    s = presetReducer(s, { type: "addSkill", key: "a1", skillId: 1001004 });
    s = presetReducer(s, {
      type: "setSkillLevel",
      key: "a1",
      index: 0,
      value: 3,
    });
    expect(s.presets[0]!.attributes.skills).toEqual([
      { skillId: 1001004, level: 3 },
    ]);
    s = presetReducer(s, { type: "removeSkill", key: "a1", index: 0 });
    s = presetReducer(s, { type: "removeInventory", key: "a1", index: 0 });
    expect(s.presets[0]!.attributes.skills).toEqual([]);
    expect(s.presets[0]!.attributes.inventory).toEqual([]);
  });

  it("tags add/remove is case-preserving and de-duplicated", () => {
    let s = loaded([preset("a1", "One")]);
    s = presetReducer(s, { type: "addTag", key: "a1", tag: "PvP" });
    s = presetReducer(s, { type: "addTag", key: "a1", tag: "PvP" });
    expect(s.presets[0]!.attributes.tags).toEqual(["PvP"]);
    s = presetReducer(s, { type: "removeTag", key: "a1", tag: "PvP" });
    expect(s.presets[0]!.attributes.tags).toEqual([]);
  });

  it("discard restores baseline and returns to library", () => {
    let s = loaded([preset("a1", "One")]);
    s = presetReducer(s, { type: "select", key: "a1" });
    s = presetReducer(s, {
      type: "setField",
      key: "a1",
      path: "level",
      value: 99,
    });
    expect(isDirty(s)).toBe(true);
    s = presetReducer(s, { type: "discard" });
    expect(isDirty(s)).toBe(false);
    expect(s.presets[0]!.attributes.level).toBe(1);
    expect(s.selectedKey).toBeNull();
  });

  it("savedOk rebaselines to the current working copy", () => {
    let s = loaded([preset("a1", "One")]);
    s = presetReducer(s, {
      type: "setField",
      key: "a1",
      path: "level",
      value: 50,
    });
    expect(isDirty(s)).toBe(true);
    s = presetReducer(s, { type: "savedOk" });
    expect(isDirty(s)).toBe(false);
    expect(presetDirty(s, "a1")).toBe(false);
  });

  it("savedOk with persisted backfills a missing id by position and rebaselines", () => {
    let s = loaded([preset("a1", "One")]);
    s = presetReducer(s, { type: "addPreset" }); // local-0, no id
    expect(s.presets[1]!.id).toBeUndefined();
    const persisted: CharacterPreset[] = [
      { id: "a1", attributes: s.presets[0]!.attributes },
      { id: "new-server-id", attributes: s.presets[1]!.attributes },
    ];
    s = presetReducer(s, { type: "savedOk", persisted });
    expect(s.presets[1]!.id).toBe("new-server-id");
    expect(s.presets[1]!.key).toBe("local-0"); // key untouched
    expect(isDirty(s)).toBe(false);
    expect(presetDirty(s, "local-0")).toBe(false);
  });

  it("savedOk with persisted never overwrites an already-set id", () => {
    let s = loaded([preset("a1", "One")]);
    const persisted: CharacterPreset[] = [
      { id: "different-id", attributes: s.presets[0]!.attributes },
    ];
    s = presetReducer(s, { type: "savedOk", persisted });
    expect(s.presets[0]!.id).toBe("a1");
  });

  it("presetDirty reflects a single preset's diff vs baseline", () => {
    let s = loaded([preset("a1", "One"), preset("b2", "Two")]);
    s = presetReducer(s, {
      type: "setField",
      key: "b2",
      path: "level",
      value: 7,
    });
    expect(presetDirty(s, "a1")).toBe(false);
    expect(presetDirty(s, "b2")).toBe(true);
  });

  it("presetDirty matches baseline by id, not position, after an earlier row is removed", () => {
    let s = loaded([
      preset("a1", "One"),
      preset("b2", "Two"),
      preset("c3", "Three"),
    ]);
    s = presetReducer(s, { type: "removePreset", key: "a1" });
    expect(presetDirty(s, "b2")).toBe(false);
    expect(presetDirty(s, "c3")).toBe(false);
  });

  it("setField coerces stringly-typed gender values", () => {
    let s = loaded([preset("a1", "One")]);
    s = presetReducer(s, {
      type: "setField",
      key: "a1",
      path: "gender",
      value: "1",
    });
    expect(s.presets[0]!.attributes.gender).toBe(1);
    s = presetReducer(s, {
      type: "setField",
      key: "a1",
      path: "gender",
      value: "0",
    });
    expect(s.presets[0]!.attributes.gender).toBe(0);
  });

  it("normalizePreset fills missing fields from defaults", () => {
    const attrs = normalizePreset({ name: "Partial" } as never);
    expect(attrs.jobId).toBe(0);
    expect(attrs.stats).toEqual(DEFAULT_PRESET_ATTRIBUTES.stats);
    expect(attrs.equipment).toEqual([]);
  });
});
