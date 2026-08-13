import { describe, expect, it } from "vitest";
import {
  buildOptionsMatrix,
  classifyOptions,
  deepEqual,
} from "@/lib/socket/options";
import type { Binding, SocketObject } from "@/lib/socket/model";

function obj(
  key: string,
  options: unknown,
  writerName = "CharacterMovement",
): SocketObject {
  const binding: Binding = {
    opCode: "0xB9",
    opCodeValue: 0xb9,
    services: ["channel"],
    options,
    index: 0,
  };
  return {
    key,
    label: key,
    source: "template",
    region: "GMS",
    majorVersion: 95,
    minorVersion: 1,
    handlers: new Map(),
    writers: new Map([[writerName, [binding]]]),
    unsupportedHandlers: new Set(),
    unsupportedWriters: new Set(),
  };
}

describe("classifyOptions", () => {
  it("classifies absence", () => {
    expect(classifyOptions(undefined)).toBe("absent");
    expect(classifyOptions(null)).toBe("absent");
  });

  it("classifies an explicit empty object", () => {
    // Measured: 9 of 543 options objects across the eleven templates are the
    // literal `{}`.
    expect(classifyOptions({})).toBe("empty");
  });

  it("classifies a single-key array value as a list", () => {
    expect(classifyOptions({ types: ["WALK", "STAND", "UNKNOWN"] })).toBe(
      "list",
    );
  });

  it("classifies a flat name to number object as a map", () => {
    expect(classifyOptions({ operations: 1, failedReasonCodes: 2 })).toBe(
      "map",
    );
    expect(classifyOptions({ codes: 7 })).toBe("map");
  });

  it("falls back to map for anything else", () => {
    expect(classifyOptions({ a: { nested: true }, b: 1 })).toBe("map");
    expect(classifyOptions("not an object")).toBe("map");
  });

  it("classifies an empty list as a list, not as empty", () => {
    // Hypothetical case, not observed in the corpus: per the measured table,
    // gms_87_1 PetMovement, gms_92_1 CharacterMovement, gms_95_1
    // CharacterMovement + PetMovement, and jms_185_1 PetMovement all have the
    // `types` key ABSENT entirely, not present-but-empty. If a version DID
    // store `{ types: [] }`, it must still classify as a list with zero
    // entries - visibly different from supplying no options at all - so this
    // exercises that defensive case without claiming it occurs.
    expect(classifyOptions({ types: [] })).toBe("list");
  });

  it("does not throw on primitives, arrays, or malformed shapes", () => {
    expect(() => classifyOptions(42)).not.toThrow();
    expect(() => classifyOptions(true)).not.toThrow();
    expect(() => classifyOptions(["bare", "array"])).not.toThrow();
    expect(() => classifyOptions(() => undefined)).not.toThrow();
    expect(classifyOptions(42)).toBe("map");
    expect(classifyOptions(true)).toBe("map");
  });

  it("does not throw on a circular-ish structure", () => {
    const circular: Record<string, unknown> = { types: [] };
    circular.self = circular;
    expect(() => classifyOptions(circular)).not.toThrow();
  });
});

describe("buildOptionsMatrix - lists compare positionally", () => {
  const a = obj("a", { types: ["WALK", "STAND", "JUMP"] });
  const b = obj("b", { types: ["WALK", "JUMP", "STAND"] });
  const c = obj("c", { types: ["WALK"] });

  const m = buildOptionsMatrix({
    objects: [a, b, c],
    kind: "writer",
    name: "CharacterMovement",
    baselineKey: "a",
  });

  it("keys rows by array index, because the index IS the wire value", () => {
    expect(m.shape).toBe("list");
    expect(m.rows.map((r) => r.key)).toEqual(["0", "1", "2"]);
    expect(m.rows[0]!.label).toBe("0");
  });

  it("treats a name that shifted index as a difference, not a match", () => {
    // A name-keyed comparison would look up "STAND" in both objects, find it
    // present in both, and report "same" - hiding the fact that it moved from
    // wire slot 1 to wire slot 2. The correct, positional answer is that slot
    // 1 differs (a says STAND, b says JUMP).
    const idx1 = m.rows[1]!;
    expect(idx1.cells.get("a")!.value).toBe("STAND");
    expect(idx1.cells.get("b")!.value).toBe("JUMP");
    expect(idx1.cells.get("b")!.state).toBe("differs");
  });

  it("marks positions the baseline has and an object does not as missing", () => {
    expect(m.rows[1]!.cells.get("c")!.state).toBe("missing");
    expect(m.rows[2]!.cells.get("c")!.state).toBe("missing");
  });

  it("marks matching positions as same", () => {
    expect(m.rows[0]!.cells.get("b")!.state).toBe("same");
    expect(m.rows[0]!.cells.get("c")!.state).toBe("same");
  });

  it("marks positions past the baseline's extent as extra", () => {
    const long = obj("d", { types: ["WALK", "STAND", "JUMP", "FLY"] });
    const m2 = buildOptionsMatrix({
      objects: [a, long],
      kind: "writer",
      name: "CharacterMovement",
      baselineKey: "a",
    });
    expect(m2.rows).toHaveLength(4);
    expect(m2.rows[3]!.cells.get("d")!.state).toBe("extra");
    expect(m2.rows[3]!.cells.get("a")!.state).toBe("missing");
  });

  it("keeps a repeated name at several indices as separate rows (the duplicate-name trap)", () => {
    // Per the measured corpus, gms_95_1 CharacterMovement carries UNKNOWN at
    // six separate indices. A name-keyed implementation would collapse all
    // occurrences of "UNKNOWN" into a single row (or merge them against each
    // other across objects), losing five of the six wire slots. The correct,
    // positional answer keeps every occurrence as its own row.
    const dup = obj("e", { types: ["UNKNOWN", "UNKNOWN", "UNKNOWN"] });
    const m3 = buildOptionsMatrix({
      objects: [dup],
      kind: "writer",
      name: "CharacterMovement",
      baselineKey: "e",
    });
    expect(m3.rows).toHaveLength(3);
    expect(m3.rows.map((r) => r.cells.get("e")!.value)).toEqual([
      "UNKNOWN",
      "UNKNOWN",
      "UNKNOWN",
    ]);
  });

  it("proves positional comparison catches what name-keyed comparison would miss", () => {
    // Two objects that both carry UNKNOWN at six indices, but with ONE index
    // (3) swapped for a real name in object f. A name-keyed comparison
    // grouping by label would see "UNKNOWN" present in both objects (multiple
    // times) and "REAL_NAME" present only in f, but it could not say WHICH
    // index differs - it has no positional identity to anchor the diff to.
    // The positional comparison correctly isolates index 3 as the only
    // difference and leaves every other index "same".
    const six = () => [
      "UNKNOWN",
      "UNKNOWN",
      "UNKNOWN",
      "UNKNOWN",
      "UNKNOWN",
      "UNKNOWN",
    ];
    const base = six();
    const shifted = six();
    shifted[3] = "REAL_NAME";

    const e1 = obj("e1", { types: base });
    const e2 = obj("e2", { types: shifted });
    const m4 = buildOptionsMatrix({
      objects: [e1, e2],
      kind: "writer",
      name: "CharacterMovement",
      baselineKey: "e1",
    });

    expect(m4.rows).toHaveLength(6);
    m4.rows.forEach((row, i) => {
      if (i === 3) {
        expect(row.cells.get("e2")!.state).toBe("differs");
      } else {
        expect(row.cells.get("e2")!.state).toBe("same");
      }
    });
  });
});

describe("buildOptionsMatrix - maps compare by key", () => {
  // Both objects here carry a single group ("operations"), so per the
  // always-qualify rule `key` is still the fully-qualified "operations.X"
  // form, but `label` collapses to the bare entry name since the whole
  // compared set agrees on one group.
  const a = obj("a", { operations: { INVITE: 1, JOIN: 2 } });
  const b = obj("b", { operations: { INVITE: 1, JOIN: 5, LEAVE: 9 } });

  const m = buildOptionsMatrix({
    objects: [a, b],
    kind: "writer",
    name: "CharacterMovement",
    baselineKey: "a",
  });

  it("keys rows by fully-qualified group.entry, but labels the bare entry name for a single-group Definition", () => {
    expect(m.shape).toBe("map");
    expect(m.rows.map((r) => r.key).sort()).toEqual([
      "operations.INVITE",
      "operations.JOIN",
      "operations.LEAVE",
    ]);
    expect(m.rows.map((r) => r.label).sort()).toEqual([
      "INVITE",
      "JOIN",
      "LEAVE",
    ]);
  });

  it("classifies equal, differing and extra values", () => {
    const invite = m.rows.find((r) => r.key === "operations.INVITE")!;
    const join = m.rows.find((r) => r.key === "operations.JOIN")!;
    const leave = m.rows.find((r) => r.key === "operations.LEAVE")!;
    expect(invite.cells.get("b")!.state).toBe("same");
    expect(join.cells.get("b")!.state).toBe("differs");
    expect(leave.cells.get("b")!.state).toBe("extra");
    expect(leave.cells.get("a")!.state).toBe("missing");
  });

  it("classifies a key present in one object and absent in another regardless of which side is the baseline", () => {
    const c = obj("c", { failedReasonCodes: { NOT_ENOUGH_MONEY: 1 } });
    const d = obj("d", {
      failedReasonCodes: { NOT_ENOUGH_MONEY: 1, ALREADY_OWNED: 2 },
    });
    const cd = buildOptionsMatrix({
      objects: [c, d],
      kind: "writer",
      name: "CharacterMovement",
      baselineKey: "d",
    });
    const alreadyOwned = cd.rows.find(
      (r) => r.key === "failedReasonCodes.ALREADY_OWNED",
    )!;
    expect(alreadyOwned.cells.get("d")!.state).toBe("same");
    expect(alreadyOwned.cells.get("c")!.state).toBe("missing");
  });

  it("a single-group map: key is qualified, label is bare", () => {
    const single = obj("single", { operations: { OPEN: 5 } });
    const m2 = buildOptionsMatrix({
      objects: [single],
      kind: "writer",
      name: "CharacterMovement",
      baselineKey: "single",
    });
    expect(m2.rows).toHaveLength(1);
    expect(m2.rows[0]!.key).toBe("operations.OPEN");
    expect(m2.rows[0]!.label).toBe("OPEN");
  });
});

describe("buildOptionsMatrix - multi-group maps flatten to group.entry rows", () => {
  // Measured: 39 of 543 options objects across the eleven templates carry
  // more than one top-level key, and every one of them is entirely
  // map-shaped groups (no group is ever a list). ServerIP (gms_12_1,
  // gms_48_1, gms_61_1, ...) is a real two-group example: `codes` + `modes`.
  it("compares a ServerIP-style codes+modes object by group.entry, not by opaque group", () => {
    const a = obj("a", {
      codes: { OK: 0, INCORRECT_PASSWORD: 4 },
      modes: { OK: 0, INCORRECT_LOGIN_ID: 1 },
    });
    const b = obj("b", {
      codes: { OK: 0, INCORRECT_PASSWORD: 99 },
      modes: { OK: 0, INCORRECT_LOGIN_ID: 1 },
    });

    const m = buildOptionsMatrix({
      objects: [a, b],
      kind: "writer",
      name: "CharacterMovement",
      baselineKey: "a",
    });

    expect(m.shape).toBe("map");
    expect(m.rows.map((r) => r.key).sort()).toEqual([
      "codes.INCORRECT_PASSWORD",
      "codes.OK",
      "modes.INCORRECT_LOGIN_ID",
      "modes.OK",
    ]);

    const diverged = m.rows.find((r) => r.key === "codes.INCORRECT_PASSWORD")!;
    expect(diverged.cells.get("b")!.state).toBe("differs");
    // Multi-group Definition: label is qualified too, not just key.
    expect(diverged.label).toBe("codes.INCORRECT_PASSWORD");

    // Everything else - including the OTHER group entirely - is unaffected.
    for (const row of m.rows) {
      if (row.key === "codes.INCORRECT_PASSWORD") continue;
      expect(row.cells.get("b")!.state).toBe("same");
    }
  });

  it("produces one row per entry across all groups, not one row per group (CharacterInteraction-style)", () => {
    // Measured: CharacterInteraction in gms_48_1 carries five groups -
    // operations, enterError, leaveReason, putStoneError, resultType. This
    // uses a smaller synthetic stand-in with the same five-group shape and
    // deliberately uneven entry counts per group, to prove the row count is
    // the SUM of entry counts, not the group count (5).
    const c = obj("c", {
      operations: { INVITE: 2, ENTER: 4, LEAVE: 10 },
      enterError: { FULL: 2, UNABLE: 6 },
      leaveReason: { KICKED: 1 },
      putStoneError: { WRONG_STONE: 1, NOT_ENOUGH: 2 },
      resultType: { SUCCESS: 0 },
    });

    const m = buildOptionsMatrix({
      objects: [c],
      kind: "writer",
      name: "CharacterMovement",
      baselineKey: "c",
    });

    expect(m.shape).toBe("map");
    // 3 + 2 + 1 + 2 + 1 = 9 entries, not 5 groups.
    expect(m.rows).toHaveLength(9);
    expect(m.rows.map((r) => r.key).sort()).toEqual(
      [
        "operations.INVITE",
        "operations.ENTER",
        "operations.LEAVE",
        "enterError.FULL",
        "enterError.UNABLE",
        "leaveReason.KICKED",
        "putStoneError.WRONG_STONE",
        "putStoneError.NOT_ENOUGH",
        "resultType.SUCCESS",
      ].sort(),
    );
  });

  it("classifies a whole missing group as per-entry missing/extra rows, not one coarse group-level row", () => {
    const a = obj("a", {
      codes: { OK: 0, ERROR: 1 },
      modes: { OK: 0, INCORRECT_LOGIN_ID: 1 },
    });
    // b never supplies the "modes" group at all.
    const b = obj("b", { codes: { OK: 0, ERROR: 1 } });

    const m = buildOptionsMatrix({
      objects: [a, b],
      kind: "writer",
      name: "CharacterMovement",
      baselineKey: "a",
    });

    const modeRows = m.rows.filter((r) => r.key.startsWith("modes."));
    expect(modeRows).toHaveLength(2);
    for (const row of modeRows) {
      expect(row.cells.get("b")!.state).toBe("missing");
      expect(row.cells.get("a")!.state).toBe("same");
    }
    // There is no single coarse "modes" row standing in for the group.
    expect(m.rows.some((r) => r.key === "modes")).toBe(false);

    // And the reverse direction: b as baseline, a supplies the extra group.
    const m2 = buildOptionsMatrix({
      objects: [a, b],
      kind: "writer",
      name: "CharacterMovement",
      baselineKey: "b",
    });
    const modeRows2 = m2.rows.filter((r) => r.key.startsWith("modes."));
    expect(modeRows2).toHaveLength(2);
    for (const row of modeRows2) {
      expect(row.cells.get("a")!.state).toBe("extra");
      expect(row.cells.get("b")!.state).toBe("missing");
    }
  });

  it("compares the SAME entry as 'same' even when the two objects carry different GROUP COUNTS for the Definition (real CharacterInteraction cross-cardinality)", () => {
    // Measured directly against the seed templates: CharacterInteraction
    // carries 1 group (operations) in gms_92_1, but 5 groups (operations,
    // enterError, leaveReason, putStoneError, resultType) in gms_48_1,
    // gms_61_1, gms_72_1, gms_79_1, gms_83_1, gms_84_1, gms_87_1, gms_95_1.
    // A per-object flatten decision (flatten only when THAT object has >1
    // key) would key operations.INVITE as "INVITE" in the 1-group object
    // and "operations.INVITE" in the 5-group object - two different row
    // keys for the identical wire value, which could never compare "same".
    const oneGroup = obj(
      "gms_92_1",
      { operations: { INVITE: 2 } },
      "CharacterInteraction",
    );
    const fiveGroup = obj(
      "gms_83_1",
      {
        operations: { INVITE: 2, ENTER: 4 },
        enterError: { FULL: 2 },
        leaveReason: { KICKED: 1 },
        putStoneError: { WRONG_STONE: 1 },
        resultType: { SUCCESS: 0 },
      },
      "CharacterInteraction",
    );

    const m = buildOptionsMatrix({
      objects: [oneGroup, fiveGroup],
      kind: "writer",
      name: "CharacterInteraction",
      baselineKey: "gms_92_1",
    });

    const invite = m.rows.find((r) => r.key === "operations.INVITE");
    expect(invite).toBeDefined();
    expect(invite!.cells.get("gms_92_1")!.state).toBe("same");
    expect(invite!.cells.get("gms_83_1")!.state).toBe("same");
    // There is no separate, uncorrelated "INVITE"-only row left over from a
    // per-object flatten decision.
    expect(m.rows.some((r) => r.key === "INVITE")).toBe(false);
  });

  it("a group missing from one object's cardinality (jms_185_1-style) still classifies per-entry, not per-group", () => {
    // Measured: jms_185_1 CharacterInteraction has 4 groups (missing
    // `enterError`) against the 5-group shape gms_48_1/61_1/.../95_1 all
    // carry.
    const jms185 = obj(
      "jms_185_1",
      {
        operations: { INVITE: 2 },
        leaveReason: { KICKED: 1 },
        putStoneError: { WRONG_STONE: 1 },
        resultType: { SUCCESS: 0 },
      },
      "CharacterInteraction",
    );
    const gms83 = obj(
      "gms_83_1",
      {
        operations: { INVITE: 2 },
        enterError: { FULL: 2, UNABLE: 6 },
        leaveReason: { KICKED: 1 },
        putStoneError: { WRONG_STONE: 1 },
        resultType: { SUCCESS: 0 },
      },
      "CharacterInteraction",
    );

    const m = buildOptionsMatrix({
      objects: [jms185, gms83],
      kind: "writer",
      name: "CharacterInteraction",
      baselineKey: "gms_83_1",
    });

    const enterErrorRows = m.rows.filter((r) =>
      r.key.startsWith("enterError."),
    );
    expect(enterErrorRows).toHaveLength(2);
    for (const row of enterErrorRows) {
      expect(row.cells.get("jms_185_1")!.state).toBe("missing");
      expect(row.cells.get("gms_83_1")!.state).toBe("same");
    }
    // No coarse group-level row stands in for the missing group.
    expect(m.rows.some((r) => r.key === "enterError")).toBe(false);

    // Every other group's entries are unaffected.
    const operationsInvite = m.rows.find((r) => r.key === "operations.INVITE")!;
    expect(operationsInvite.cells.get("jms_185_1")!.state).toBe("same");
  });

  it("throws on an ambiguous flattened key instead of silently dropping a row", () => {
    // Defensive only - not a corpus shape. Group "a.b" entry "c" and group
    // "a" entry "b.c" both flatten to the string "a.b.c".
    const ambiguous = obj("x", {
      "a.b": { c: 1 },
      a: { "b.c": 2 },
    });
    expect(() =>
      buildOptionsMatrix({
        objects: [ambiguous],
        kind: "writer",
        name: "CharacterMovement",
        baselineKey: "x",
      }),
    ).toThrow(/ambiguous flattened options row key/);
  });
});

describe("buildOptionsMatrix - degenerate inputs", () => {
  it("returns no rows when nobody supplies options", () => {
    const m = buildOptionsMatrix({
      objects: [obj("a", undefined), obj("b", {})],
      kind: "writer",
      name: "CharacterMovement",
      baselineKey: "a",
    });
    expect(m.rows).toHaveLength(0);
  });

  it("renders nothing surprising when both sides are the literal empty object", () => {
    const m = buildOptionsMatrix({
      objects: [obj("a", {}), obj("b", {})],
      kind: "writer",
      name: "CharacterMovement",
      baselineKey: "a",
    });
    expect(m.shape).toBe("empty");
    expect(m.rows).toHaveLength(0);
  });

  it("uses a supplying object's shape when the baseline supplies none", () => {
    const m = buildOptionsMatrix({
      objects: [obj("a", undefined), obj("b", { types: ["WALK"] })],
      kind: "writer",
      name: "CharacterMovement",
      baselineKey: "a",
    });
    expect(m.shape).toBe("list");
    expect(m.rows).toHaveLength(1);
    expect(m.rows[0]!.cells.get("a")!.state).toBe("missing");
    expect(m.rows[0]!.cells.get("b")!.state).toBe("extra");
  });

  it("a Definition where one object omits options entirely and another supplies it", () => {
    const m = buildOptionsMatrix({
      objects: [obj("a", { operations: { INVITE: 1 } }), obj("b", undefined)],
      kind: "writer",
      name: "CharacterMovement",
      baselineKey: "a",
    });
    expect(m.shape).toBe("map");
    expect(m.rows).toHaveLength(1);
    expect(m.rows[0]!.cells.get("a")!.state).toBe("same");
    expect(m.rows[0]!.cells.get("b")!.state).toBe("missing");
  });

  it("classifies the baseline's own column as same against itself, never spuriously differs", () => {
    const a = obj("a", {
      types: ["WALK", "STAND", "JUMP"],
    });
    const b = obj("b", { types: ["FLY"] });
    const m = buildOptionsMatrix({
      objects: [a, b],
      kind: "writer",
      name: "CharacterMovement",
      baselineKey: "a",
    });
    for (const row of m.rows) {
      const baselineCell = row.cells.get("a")!;
      expect(baselineCell.state).not.toBe("differs");
      expect(baselineCell.state).not.toBe("missing");
    }

    const mapA = obj("mapA", { operations: { INVITE: 1, JOIN: 2 } });
    const mapB = obj("mapB", { operations: { INVITE: 9 } });
    const mapM = buildOptionsMatrix({
      objects: [mapA, mapB],
      kind: "writer",
      name: "CharacterMovement",
      baselineKey: "mapA",
    });
    for (const row of mapM.rows) {
      const baselineCell = row.cells.get("mapA")!;
      expect(baselineCell.state).not.toBe("differs");
      expect(baselineCell.state).not.toBe("missing");
    }
  });

  it("does not throw when options values are circular-ish", () => {
    const circA: Record<string, unknown> = { operations: { INVITE: 1 } };
    const circB: Record<string, unknown> = { operations: { INVITE: 1 } };
    (circA.operations as Record<string, unknown>).self = circA;
    (circB.operations as Record<string, unknown>).self = circB;

    expect(() =>
      buildOptionsMatrix({
        objects: [obj("a", circA), obj("b", circB)],
        kind: "writer",
        name: "CharacterMovement",
        baselineKey: "a",
      }),
    ).not.toThrow();
  });
});

describe("deepEqual", () => {
  it("is key-order-independent for a whole multi-key options object", () => {
    // Real corpus fact: CharacterInteraction's five option groups are
    // `operations, enterError, resultType, putStoneError, leaveReason` in
    // seed-template order. Go's encoding/json marshals object/map keys
    // SORTED, so a tenant round-tripped through the REST model comes back
    // alphabetized. The two objects below carry identical values in each of
    // those two orders - this must compare equal.
    const seedOrder = {
      operations: { OPEN: 1 },
      enterError: { A: 1 },
      resultType: { B: 1 },
      putStoneError: { C: 1 },
      leaveReason: { D: 1 },
    };
    const alphabetized = {
      enterError: { A: 1 },
      leaveReason: { D: 1 },
      operations: { OPEN: 1 },
      putStoneError: { C: 1 },
      resultType: { B: 1 },
    };
    expect(deepEqual(seedOrder, alphabetized)).toBe(true);
  });

  it("is key-order-independent for a 2-key options object (NoteOperation shape)", () => {
    const seedOrder = { operations: { A: 1 }, errors: { B: 2 } };
    const alphabetized = { errors: { B: 2 }, operations: { A: 1 } };
    expect(deepEqual(seedOrder, alphabetized)).toBe(true);
  });

  it("is key-order-independent at a NESTED level too, not just the top level", () => {
    const a = { operations: { A: 1, B: 2 } };
    const b = { operations: { B: 2, A: 1 } };
    expect(deepEqual(a, b)).toBe(true);
  });

  it("keeps ARRAYS strictly positional - same elements, different order, not equal", () => {
    // Guards against a fix that over-normalizes by sorting arrays too:
    // FR-3.5's list semantics depend on the array INDEX being the wire
    // value, so reordering a `types` list is a real, meaningful change.
    expect(
      deepEqual({ types: ["WALK", "STAND"] }, { types: ["STAND", "WALK"] }),
    ).toBe(false);
  });

  it("still reports a genuine value difference under any key order", () => {
    const a = { operations: { A: 1 }, errors: { B: 2 } };
    const b = { errors: { B: 3 }, operations: { A: 1 } };
    expect(deepEqual(a, b)).toBe(false);
  });
});
