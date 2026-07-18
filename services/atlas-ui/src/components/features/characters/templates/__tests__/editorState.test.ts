import { describe, it, expect } from "vitest";
import {
  editorReducer,
  initialEditorState,
  normalizeTemplate,
  blankTemplate,
  isDirty,
  picksFor,
  emptyPoolWarnings,
  DEFAULT_PICKS,
  type EditorState,
} from "../editorState";

function loaded(
  templates: Parameters<typeof normalizeTemplate>[0][],
): EditorState {
  return editorReducer(initialEditorState(), {
    type: "load",
    templates: templates.map(normalizeTemplate),
  });
}

const tpl = (over: Partial<ReturnType<typeof blankTemplate>> = {}) =>
  normalizeTemplate({ ...blankTemplate(), ...over });

describe("load", () => {
  it("normalizes missing arrays, snapshots baseline, sets loaded", () => {
    const s = loaded([{ jobIndex: 1 }]);
    expect(s.loaded).toBe(true);
    expect(s.templates[0]!.faces).toEqual([]);
    expect(s.baseline).toEqual(s.templates);
    expect(isDirty(s)).toBe(false);
  });
});

describe("template lifecycle", () => {
  it("addTemplate appends a blank template and selects it", () => {
    const s = editorReducer(loaded([tpl()]), { type: "addTemplate" });
    expect(s.templates).toHaveLength(2);
    expect(s.selectedIndex).toBe(1);
    expect(s.templates[1]!).toEqual(blankTemplate());
  });

  it("duplicateTemplate deep-copies the selected template and selects the copy", () => {
    const s0 = loaded([tpl({ faces: [20000] })]);
    const s = editorReducer(s0, { type: "duplicateTemplate" });
    expect(s.templates).toHaveLength(2);
    expect(s.selectedIndex).toBe(1);
    expect(s.templates[1]!).toEqual(s.templates[0]);
    // deep copy: mutating the copy's array must not touch the original
    s.templates[1]!.faces.push(21000);
    expect(s.templates[0]!.faces).toEqual([20000]);
  });

  it("removeTemplate selects the nearest remaining index and remaps picks", () => {
    let s = loaded([tpl(), tpl(), tpl()]);
    s = editorReducer(s, { type: "select", index: 2 });
    s = editorReducer(s, { type: "setPreviewPick", pick: "faceIdx", value: 3 });
    s = editorReducer(s, { type: "select", index: 1 });
    s = editorReducer(s, { type: "removeTemplate" }); // removes index 1
    expect(s.templates).toHaveLength(2);
    expect(s.selectedIndex).toBe(1); // nearest remaining
    // picks for old index 2 shifted down to index 1
    expect(picksFor(s, 1).faceIdx).toBe(3);
  });

  it("removing the last remaining template clamps selection to 0", () => {
    const s = editorReducer(loaded([tpl()]), { type: "removeTemplate" });
    expect(s.templates).toHaveLength(0);
    expect(s.selectedIndex).toBe(0);
  });
});

describe("edits", () => {
  it("setIdentity updates the selected template and marks dirty", () => {
    const s = editorReducer(loaded([tpl()]), {
      type: "setIdentity",
      field: "mapId",
      value: 100000000,
    });
    expect(s.templates[0]!.mapId).toBe(100000000);
    expect(isDirty(s)).toBe(true);
  });

  it("addPoolEntry appends and prevents double-add", () => {
    let s = editorReducer(loaded([tpl()]), {
      type: "addPoolEntry",
      pool: "faces",
      id: 20000,
    });
    s = editorReducer(s, { type: "addPoolEntry", pool: "faces", id: 20000 });
    expect(s.templates[0]!.faces).toEqual([20000]);
  });

  it("removePoolEntry clamps the matching preview pick", () => {
    let s = loaded([tpl({ faces: [20000, 20001] })]);
    s = editorReducer(s, { type: "setPreviewPick", pick: "faceIdx", value: 1 });
    s = editorReducer(s, {
      type: "removePoolEntry",
      pool: "faces",
      entryIndex: 1,
    });
    expect(s.templates[0]!.faces).toEqual([20000]);
    expect(picksFor(s, 0).faceIdx).toBe(0);
  });

  it("select never touches templates (free switching keeps edits)", () => {
    let s = loaded([tpl(), tpl()]);
    s = editorReducer(s, { type: "addPoolEntry", pool: "hairs", id: 30030 });
    s = editorReducer(s, { type: "select", index: 1 });
    expect(s.templates[0]!.hairs).toEqual([30030]);
    expect(isDirty(s)).toBe(true);
  });
});

describe("discard / savedOk", () => {
  it("discard restores baseline and resets picks", () => {
    let s = loaded([tpl()]);
    s = editorReducer(s, { type: "setPreviewPick", pick: "hairIdx", value: 2 });
    s = editorReducer(s, { type: "addPoolEntry", pool: "faces", id: 20000 });
    s = editorReducer(s, { type: "discard" });
    expect(isDirty(s)).toBe(false);
    expect(s.templates[0]!.faces).toEqual([]);
    expect(picksFor(s, 0)).toEqual(DEFAULT_PICKS);
  });

  it("savedOk re-baselines the working copy", () => {
    let s = loaded([tpl()]);
    s = editorReducer(s, { type: "addPoolEntry", pool: "faces", id: 20000 });
    s = editorReducer(s, { type: "savedOk" });
    expect(isDirty(s)).toBe(false);
    expect(s.baseline[0]!.faces).toEqual([20000]);
  });
});

describe("emptyPoolWarnings", () => {
  it("flags the four appearance pools the factory rejects when empty", () => {
    expect(emptyPoolWarnings(blankTemplate())).toEqual([
      "faces",
      "hairs",
      "hairColors",
      "skinColors",
    ]);
    expect(
      emptyPoolWarnings(
        tpl({ faces: [1], hairs: [1], hairColors: [0], skinColors: [0] }),
      ),
    ).toEqual([]);
  });
});
