import { describe, expect, it } from "vitest";
import {
  buildRows,
  emptyFilters,
  filterRows,
  sortRows,
} from "@/lib/socket/matrix";
import type { Binding, SocketObject } from "@/lib/socket/model";
import { parseOpcode } from "@/lib/socket/opcode";

function binding(opCode: string, extra: Partial<Binding> = {}): Binding {
  return {
    opCode,
    opCodeValue: parseOpcode(opCode),
    services: ["channel"],
    index: 0,
    ...extra,
  };
}

function obj(
  key: string,
  major: number,
  writers: Record<string, Binding[]>,
  unsupportedWriters: string[] = [],
): SocketObject {
  return {
    key,
    label: `GMS v${major}.1`,
    source: "template",
    region: "GMS",
    majorVersion: major,
    minorVersion: 1,
    handlers: new Map(),
    writers: new Map(Object.entries(writers)),
    unsupportedHandlers: new Set(),
    unsupportedWriters: new Set(unsupportedWriters),
  };
}

describe("buildRows", () => {
  it("unions defined and unsupported names across every object", () => {
    const a = obj("a", 83, { AuthSuccess: [binding("0x00")] }, [
      "MonsterCarnival",
    ]);
    const b = obj("b", 95, { PetActivated: [binding("0x9A")] });
    const rows = buildRows({ objects: [a, b], kind: "writer", baselineKey: "b" });
    expect(rows.map((r) => r.name).sort()).toEqual([
      "AuthSuccess",
      "MonsterCarnival",
      "PetActivated",
    ]);
  });

  it("gives each row one cell per object, with the right state", () => {
    const a = obj("a", 83, { AuthSuccess: [binding("0x00")] }, [
      "MonsterCarnival",
    ]);
    const b = obj("b", 95, {});
    const rows = buildRows({ objects: [a, b], kind: "writer", baselineKey: "b" });

    const auth = rows.find((r) => r.name === "AuthSuccess")!;
    expect(auth.cells.get("a")!.state).toBe("defined");
    expect(auth.cells.get("b")!.state).toBe("undefined");

    // FR-2.5: MonsterCarnival appears purely because it is Unsupported in "a"
    // and absent from "b" - neither object defines it.
    const carnival = rows.find((r) => r.name === "MonsterCarnival")!;
    expect(carnival.cells.get("a")!.state).toBe("unsupported");
    expect(carnival.cells.get("b")!.state).toBe("undefined");
  });

  it("carries every binding of a multi-binding definition into one cell", () => {
    const a = obj("a", 95, {
      CharacterEffect: [binding("0xE0"), binding("0xE9")],
    });
    const rows = buildRows({ objects: [a], kind: "writer", baselineKey: "a" });
    expect(rows[0]!.cells.get("a")!.bindings.map((x) => x.opCodeValue)).toEqual([
      0xe0, 0xe9,
    ]);
    expect(rows[0]!.cells.get("a")!.hasDuplicateOpcode).toBe(false);
  });

  // NoOpHandler is bound to four distinct opcodes in gms_95_1 (0x17, 0x19,
  // 0x22, 0x24) - a real corpus fact, not a hypothetical. One Definition, one
  // row, one Cell holding all four bindings.
  it("carries all four bindings of a NoOpHandler-style definition into one cell, with no duplicate flag", () => {
    const a = obj("a", 95, {
      NoOpHandler: [
        binding("0x17"),
        binding("0x19"),
        binding("0x22"),
        binding("0x24"),
      ],
    });
    const rows = buildRows({ objects: [a], kind: "writer", baselineKey: "a" });
    expect(rows).toHaveLength(1);
    const cell = rows[0]!.cells.get("a")!;
    expect(cell.bindings.map((x) => x.opCodeValue)).toEqual([
      0x17, 0x19, 0x22, 0x24,
    ]);
    expect(cell.hasDuplicateOpcode).toBe(false);
  });

  it("flags a cell whose bindings collide numerically", () => {
    const a = obj("a", 95, { MiniRoom: [binding("0xB8"), binding("0x0B8")] });
    const rows = buildRows({ objects: [a], kind: "writer", baselineKey: "a" });
    expect(rows[0]!.cells.get("a")!.hasDuplicateOpcode).toBe(true);
  });

  it("does not flag distinct opcodes as duplicates", () => {
    const a = obj("a", 95, {
      NoOpHandler: [binding("0x17"), binding("0x19")],
    });
    const rows = buildRows({ objects: [a], kind: "writer", baselineKey: "a" });
    expect(rows[0]!.cells.get("a")!.hasDuplicateOpcode).toBe(false);
  });

  it("takes the row fname from the first object that supplies one", () => {
    const a = obj("a", 83, { AuthSuccess: [binding("0x00")] });
    const b = obj("b", 95, {
      AuthSuccess: [binding("0x00", { fname: "CLogin::OnCheckPasswordResult" })],
    });
    const rows = buildRows({ objects: [a, b], kind: "writer", baselineKey: "a" });
    expect(rows[0]!.fname).toBe("CLogin::OnCheckPasswordResult");
  });

  it("leaves fname undefined when no object supplies one", () => {
    const a = obj("a", 83, { AuthSuccess: [binding("0x00")] });
    const rows = buildRows({ objects: [a], kind: "writer", baselineKey: "a" });
    expect(rows[0]!.fname).toBeUndefined();
    expect("fname" in rows[0]!).toBe(false);
  });

  // FR-3.2: the ONLY options signal in the grid. It fires on ABSENCE where a
  // sibling supplies options - never on structural divergence, which is the
  // expected state between versions (FR-3.1).
  it("marks a cell that supplies no options where a sibling does", () => {
    const a = obj("a", 83, {
      CharacterMovement: [binding("0xB9", { options: { types: ["A", "B"] } })],
    });
    const b = obj("b", 95, { CharacterMovement: [binding("0xC0")] });
    const c = obj("c", 87, {
      CharacterMovement: [binding("0xBC", { options: {} })],
    });
    const rows = buildRows({ objects: [a, b, c], kind: "writer", baselineKey: "a" });
    const row = rows[0]!;
    expect(row.cells.get("a")!.optionsMissing).toBe(false);
    expect(row.cells.get("b")!.optionsMissing).toBe(true); // absent
    expect(row.cells.get("c")!.optionsMissing).toBe(true); // explicit {}
  });

  it("marks nothing when no object supplies options for that definition", () => {
    const a = obj("a", 83, { AuthSuccess: [binding("0x00")] });
    const b = obj("b", 95, { AuthSuccess: [binding("0x00")] });
    const rows = buildRows({ objects: [a, b], kind: "writer", baselineKey: "a" });
    expect(rows[0]!.cells.get("a")!.optionsMissing).toBe(false);
    expect(rows[0]!.cells.get("b")!.optionsMissing).toBe(false);
  });

  it("marks nothing on an undefined cell - absence of a definition is not an options omission", () => {
    const a = obj("a", 83, {
      CharacterMovement: [binding("0xB9", { options: { types: ["A"] } })],
    });
    const b = obj("b", 95, {});
    const rows = buildRows({ objects: [a, b], kind: "writer", baselineKey: "a" });
    expect(rows[0]!.cells.get("b")!.optionsMissing).toBe(false);
  });

  // FR-3.1: structural divergence between two objects that BOTH supply
  // options is the expected, unremarkable state - gms_12_1 has 9 movement
  // types, jms_185_1 has 33. Neither side is "missing" anything; marking
  // either would be noise, not signal.
  it("marks neither side when two objects both supply options but with different content", () => {
    const a = obj("a", 12, {
      CharacterMovement: [
        binding("0xB9", {
          options: { types: ["WALK", "JUMP", "FLY"] },
        }),
      ],
    });
    const b = obj("b", 185, {
      CharacterMovement: [
        binding("0x09", {
          options: { types: ["WALK", "JUMP", "FLY", "SIT", "FLASH_JUMP"] },
        }),
      ],
    });
    const rows = buildRows({ objects: [a, b], kind: "writer", baselineKey: "a" });
    expect(rows[0]!.cells.get("a")!.optionsMissing).toBe(false);
    expect(rows[0]!.cells.get("b")!.optionsMissing).toBe(false);
  });

  // The empty gms_92_1/gms_95_1 CharacterMovement and gms_87_1/gms_95_1/
  // jms_185_1 PetMovement tables: the "types" key is present, its array just
  // happens to have zero entries. That counts as SUPPLYING options, not
  // omitting them - it must not trip the FR-3.2 marker on the sibling that
  // has a populated table, nor be flagged missing itself.
  it("treats an empty types array as supplying options, not omitting them", () => {
    const a = obj("a", 83, {
      CharacterMovement: [
        binding("0xB9", { options: { types: ["WALK", "JUMP"] } }),
      ],
    });
    const b = obj("b", 95, {
      CharacterMovement: [binding("0xC0", { options: { types: [] } })],
    });
    const rows = buildRows({ objects: [a, b], kind: "writer", baselineKey: "a" });
    const row = rows[0]!;
    expect(row.cells.get("a")!.optionsMissing).toBe(false);
    expect(row.cells.get("b")!.optionsMissing).toBe(false);
  });

  it("records baseline membership and the baseline opcode", () => {
    const a = obj("a", 83, { AuthSuccess: [binding("0x00")] });
    const b = obj("b", 95, { PetActivated: [binding("0x9A")] });
    const rows = buildRows({ objects: [a, b], kind: "writer", baselineKey: "b" });
    const pet = rows.find((r) => r.name === "PetActivated")!;
    const auth = rows.find((r) => r.name === "AuthSuccess")!;
    expect(pet.inBaseline).toBe(true);
    expect(pet.baselineOpCodeValue).toBe(0x9a);
    expect(auth.inBaseline).toBe(false);
    expect(auth.baselineOpCodeValue).toBeNull();
  });
});

describe("sortRows", () => {
  const a = obj("a", 83, { Zebra: [binding("0x02")] });
  const b = obj("b", 95, {
    Alpha: [binding("0x10")],
    Beta: [binding("0x05")],
  });
  const rows = buildRows({ objects: [a, b], kind: "writer", baselineKey: "b" });

  // FR-4.1 default sort, FR-2.11 non-baseline entries last.
  it("sorts by ascending baseline opcode and puts non-baseline rows last", () => {
    expect(sortRows(rows, "opcode", "asc").map((r) => r.name)).toEqual([
      "Beta",
      "Alpha",
      "Zebra",
    ]);
  });

  it("keeps non-baseline rows last when the direction is reversed", () => {
    expect(sortRows(rows, "opcode", "desc").map((r) => r.name)).toEqual([
      "Alpha",
      "Beta",
      "Zebra",
    ]);
  });

  it("sorts by name ascending", () => {
    expect(sortRows(rows, "name", "asc").map((r) => r.name)).toEqual([
      "Alpha",
      "Beta",
      "Zebra",
    ]);
  });

  it("sorts by name descending, non-baseline rows still last", () => {
    // Zebra ("a") is not in the "b" baseline, so it stays last even though
    // "Z" would otherwise sort first in a plain descending name compare.
    expect(sortRows(rows, "name", "desc").map((r) => r.name)).toEqual([
      "Beta",
      "Alpha",
      "Zebra",
    ]);
  });

  describe("by state", () => {
    // State-sort compares each row's cell for the FIRST selected object ("c"
    // here). Using a baselineKey that matches none of the selected objects
    // keeps every row out of the baseline group uniformly, so the ordering
    // below is driven purely by the state comparator, not by FR-2.11
    // baseline-first grouping (which the opcode-sort tests above already
    // cover).
    const c = obj("c", 92, { Def1: [binding("0x01")] }, ["Def2"]);
    const d = obj("d", 95, { Def3: [binding("0x03")] });
    const rowsNoBaseline = buildRows({
      objects: [c, d],
      kind: "writer",
      baselineKey: "no-such-object",
    });

    it("orders defined before unsupported before undefined, ascending", () => {
      expect(rowsNoBaseline.every((r) => !r.inBaseline)).toBe(true);
      const sorted = sortRows(rowsNoBaseline, "state", "asc").map(
        (r) => r.name,
      );
      expect(sorted).toEqual(["Def1", "Def2", "Def3"]);
    });

    it("reverses to undefined before unsupported before defined, descending", () => {
      const sorted = sortRows(rowsNoBaseline, "state", "desc").map(
        (r) => r.name,
      );
      expect(sorted).toEqual(["Def3", "Def2", "Def1"]);
    });
  });
});

describe("filterRows", () => {
  const a = obj(
    "a",
    83,
    {
      AuthSuccess: [binding("0x00", { fname: "CLogin::OnCheckPasswordResult" })],
      CharacterMovement: [binding("0x2A", { options: { types: ["A"] } })],
    },
    ["MonsterCarnival"],
  );
  const b = obj("b", 95, { CharacterMovement: [binding("0xC0")] });
  const rows = buildRows({ objects: [a, b], kind: "writer", baselineKey: "a" });

  it("returns everything when no filter is set", () => {
    expect(filterRows(rows, emptyFilters(), "a")).toHaveLength(3);
  });

  it("searches the definition name, case-insensitively", () => {
    const got = filterRows(rows, { ...emptyFilters(), query: "movement" }, "a");
    expect(got.map((r) => r.name)).toEqual(["CharacterMovement"]);
  });

  it("searches fname", () => {
    const got = filterRows(rows, { ...emptyFilters(), query: "CheckPassword" }, "a");
    expect(got.map((r) => r.name)).toEqual(["AuthSuccess"]);
  });

  // FR-4.3, end to end through the filter.
  it("matches 0x2A, 2A and 42 against the same row", () => {
    for (const q of ["0x2A", "2A", "42"]) {
      const got = filterRows(rows, { ...emptyFilters(), query: q }, "a");
      expect(got.map((r) => r.name), `query ${q}`).toContain("CharacterMovement");
    }
  });

  it("filters by state within the baseline object", () => {
    const got = filterRows(rows, { ...emptyFilters(), states: ["unsupported"] }, "a");
    expect(got.map((r) => r.name)).toEqual(["MonsterCarnival"]);
  });

  it("filters to rows carrying the options-omission marker", () => {
    const got = filterRows(rows, { ...emptyFilters(), optionsMissingOnly: true }, "a");
    expect(got.map((r) => r.name)).toEqual(["CharacterMovement"]);
  });

  it("filters by hasOptions=true to rows whose baseline cell supplies options", () => {
    const got = filterRows(rows, { ...emptyFilters(), hasOptions: true }, "a");
    expect(got.map((r) => r.name)).toEqual(["CharacterMovement"]);
  });

  it("filters by hasOptions=false to rows whose baseline cell supplies none", () => {
    const got = filterRows(rows, { ...emptyFilters(), hasOptions: false }, "a");
    expect(got.map((r) => r.name).sort()).toEqual([
      "AuthSuccess",
      "MonsterCarnival",
    ]);
  });

  it("filters by service", () => {
    const got = filterRows(rows, { ...emptyFilters(), services: ["login"] }, "a");
    expect(got).toHaveLength(0);
  });

  it("combines a name search with a state filter", () => {
    const got = filterRows(
      rows,
      { ...emptyFilters(), query: "monster", states: ["unsupported"] },
      "a",
    );
    expect(got.map((r) => r.name)).toEqual(["MonsterCarnival"]);
  });

  it("returns nothing when a combination of filters matches no row", () => {
    const got = filterRows(
      rows,
      { ...emptyFilters(), query: "monster", states: ["defined"] },
      "a",
    );
    expect(got).toHaveLength(0);
  });
});
