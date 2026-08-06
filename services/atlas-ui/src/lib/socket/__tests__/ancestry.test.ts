import { describe, expect, it } from "vitest";
import {
  classifyAgainstAncestor,
  inferAncestor,
  missingFromTenant,
} from "@/lib/socket/ancestry";
import type { Binding, SocketObject } from "@/lib/socket/model";
import { parseOpcode } from "@/lib/socket/opcode";

function binding(opCode: string, extra: Partial<Binding> = {}): Binding {
  return {
    opCode,
    opCodeValue: parseOpcode(opCode),
    validator: "LoggedInValidator",
    services: ["channel"],
    index: 0,
    ...extra,
  };
}

function makeObject(
  key: string,
  source: SocketObject["source"],
  major: number,
  minor: number,
  handlers: Record<string, Binding[]> = {},
  unsupportedHandlers: string[] = [],
  region = "GMS",
): SocketObject {
  return {
    key,
    label: `${region} v${major}.${minor}`,
    source,
    region,
    majorVersion: major,
    minorVersion: minor,
    handlers: new Map(Object.entries(handlers)),
    writers: new Map(),
    unsupportedHandlers: new Set(unsupportedHandlers),
    unsupportedWriters: new Set(),
  };
}

describe("inferAncestor", () => {
  const t83 = makeObject("t83", "template", 83, 1);
  const t95 = makeObject("t95", "template", 95, 1);
  const jms = makeObject("jms", "template", 185, 1, {}, [], "JMS");

  it("matches on exact region, major and minor version", () => {
    const tenant = makeObject("tnt", "tenant", 95, 1);
    expect(inferAncestor(tenant, [t83, t95, jms])?.key).toBe("t95");
  });

  it("does not match across regions", () => {
    const tenant = makeObject("tnt", "tenant", 185, 1, {}, [], "GMS");
    expect(inferAncestor(tenant, [t83, t95, jms])).toBeNull();
  });

  it("does not match on major version alone", () => {
    const tenant = makeObject("tnt", "tenant", 95, 2);
    expect(inferAncestor(tenant, [t83, t95])).toBeNull();
  });

  it("returns null when there is no template at all", () => {
    expect(inferAncestor(makeObject("tnt", "tenant", 95, 1), [])).toBeNull();
  });

  it("resolves several exact matches to the FIRST in array order", () => {
    // Every seed template is expected to hold a distinct (region, major,
    // minor) triple, so this is not expected to occur against real data.
    // Pinned anyway so the resolution rule is a documented decision -
    // Array.find's first-match semantics - rather than an accident of
    // whatever order the caller happened to pass templates in.
    const dupeA = makeObject("dupeA", "template", 95, 1);
    const dupeB = makeObject("dupeB", "template", 95, 1);
    const tenant = makeObject("tnt", "tenant", 95, 1);
    expect(inferAncestor(tenant, [dupeA, dupeB])?.key).toBe("dupeA");
    expect(inferAncestor(tenant, [dupeB, dupeA])?.key).toBe("dupeB");
  });
});

describe("classifyAgainstAncestor", () => {
  it("returns same for an identical binding set", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, {
      LoginHandle: [binding("0x01")],
    });
    const anc = makeObject("t83", "template", 83, 1, {
      LoginHandle: [binding("0x01")],
    });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "LoginHandle")).toBe(
      "same",
    );
  });

  it("normalises opcodes numerically before comparing", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, {
      MiniRoom: [binding("0x0B8")],
    });
    const anc = makeObject("t83", "template", 83, 1, {
      MiniRoom: [binding("0xB8")],
    });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "MiniRoom")).toBe(
      "same",
    );
  });

  it("ignores fname entirely", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, {
      LoginHandle: [
        binding("0x01", { fname: "CLogin::SendCheckPasswordPacket" }),
      ],
    });
    const anc = makeObject("t83", "template", 83, 1, {
      LoginHandle: [binding("0x01")],
    });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "LoginHandle")).toBe(
      "same",
    );
  });

  it("returns modified when the opcode differs", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, {
      LoginHandle: [binding("0x02")],
    });
    const anc = makeObject("t83", "template", 83, 1, {
      LoginHandle: [binding("0x01")],
    });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "LoginHandle")).toBe(
      "modified",
    );
  });

  it("returns modified when the validator differs", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, {
      LoginHandle: [binding("0x01", { validator: "NoOpValidator" })],
    });
    const anc = makeObject("t83", "template", 83, 1, {
      LoginHandle: [binding("0x01")],
    });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "LoginHandle")).toBe(
      "modified",
    );
  });

  it("returns modified when the services differ", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, {
      LoginHandle: [binding("0x01", { services: ["login", "channel"] })],
    });
    const anc = makeObject("t83", "template", 83, 1, {
      LoginHandle: [binding("0x01")],
    });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "LoginHandle")).toBe(
      "modified",
    );
  });

  it("treats the services list as a SET - order does not matter", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, {
      LoginHandle: [binding("0x01", { services: ["channel", "login"] })],
    });
    const anc = makeObject("t83", "template", 83, 1, {
      LoginHandle: [binding("0x01", { services: ["login", "channel"] })],
    });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "LoginHandle")).toBe(
      "same",
    );
  });

  it("returns modified when the options differ structurally", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, {
      Move: [binding("0x01", { options: { types: ["WALK"] } })],
    });
    const anc = makeObject("t83", "template", 83, 1, {
      Move: [binding("0x01", { options: { types: ["WALK", "STAND"] } })],
    });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "Move")).toBe(
      "modified",
    );
  });

  it("treats an absent options key and an explicit {} as equal (no content)", () => {
    // Real corpus fact (matrix.ts's suppliesOptions doc comment): gms_95_1's
    // MiniRoom writer stores options as an explicit {}, while gms_87_1's
    // PetMovement omits the options key entirely - both mean "no options
    // supplied". A PATCH round-trip materialising {} must not read as
    // `modified` against an ancestor that simply omitted the key.
    const tenant = makeObject("tnt", "tenant", 83, 1, {
      MiniRoom: [binding("0xB8", { options: {} })],
    });
    const anc = makeObject("t83", "template", 83, 1, {
      MiniRoom: [binding("0xB8")],
    });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "MiniRoom")).toBe(
      "same",
    );
  });

  it("is NOT fooled by options key order - CharacterInteraction, seed order vs alphabetized", () => {
    // Real corpus fact: CharacterInteraction's five option groups are
    // `operations, enterError, resultType, putStoneError, leaveReason` in
    // seed-template order (verified directly against the seed templates).
    // Go's encoding/json marshals map keys SORTED, so a tenant configuration
    // written back through the REST model comes out alphabetized - a tenant
    // byte-identical in meaning to its ancestor must still classify `same`.
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
    const anc2 = makeObject("t83", "template", 83, 1, {
      CharacterInteraction: [binding("0x01", { options: seedOrder })],
    });
    const tenant2 = makeObject("tnt", "tenant", 83, 1, {
      CharacterInteraction: [binding("0x01", { options: alphabetized })],
    });
    expect(
      classifyAgainstAncestor(tenant2, anc2, "handler", "CharacterInteraction"),
    ).toBe("same");
  });

  it("is NOT fooled by options key order - NoteOperation, 2-key shape", () => {
    const anc = makeObject("t83", "template", 83, 1, {
      NoteOperation: [
        binding("0x02", {
          options: { operations: { A: 1 }, errors: { B: 2 } },
        }),
      ],
    });
    const tenant = makeObject("tnt", "tenant", 83, 1, {
      NoteOperation: [
        binding("0x02", {
          options: { errors: { B: 2 }, operations: { A: 1 } },
        }),
      ],
    });
    expect(
      classifyAgainstAncestor(tenant, anc, "handler", "NoteOperation"),
    ).toBe("same");
  });

  it("keeps a list's ARRAY order significant - same elements reordered is still modified", () => {
    // Guards against a fix that over-normalizes by sorting arrays too:
    // FR-3.5's list semantics depend on the array INDEX being the wire
    // value, so this must NOT collapse to "same" the way the key-order fix
    // does for maps.
    const tenant = makeObject("tnt", "tenant", 83, 1, {
      Move: [binding("0x01", { options: { types: ["STAND", "WALK"] } })],
    });
    const anc = makeObject("t83", "template", 83, 1, {
      Move: [binding("0x01", { options: { types: ["WALK", "STAND"] } })],
    });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "Move")).toBe(
      "modified",
    );
  });

  it("is key-order-independent at a NESTED level, not just the top-level options object", () => {
    const anc = makeObject("t83", "template", 83, 1, {
      ServerIP: [
        binding("0x03", {
          options: { codes: { A: 1, B: 2 }, modes: { X: 9 } },
        }),
      ],
    });
    const tenant = makeObject("tnt", "tenant", 83, 1, {
      // Top-level order matches; the NESTED `codes` group's key order does not.
      ServerIP: [
        binding("0x03", {
          options: { codes: { B: 2, A: 1 }, modes: { X: 9 } },
        }),
      ],
    });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "ServerIP")).toBe(
      "same",
    );
  });

  it("returns modified when the binding COUNT differs for one name", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, {
      NoOpHandler: [binding("0x17"), binding("0x19")],
    });
    const anc = makeObject("t83", "template", 83, 1, {
      NoOpHandler: [binding("0x17"), binding("0x19"), binding("0x22")],
    });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "NoOpHandler")).toBe(
      "modified",
    );
  });

  it("compares multi-binding sets irrespective of stored order", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, {
      NoOpHandler: [binding("0x19"), binding("0x17")],
    });
    const anc = makeObject("t83", "template", 83, 1, {
      NoOpHandler: [binding("0x17"), binding("0x19")],
    });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "NoOpHandler")).toBe(
      "same",
    );
  });

  it("returns modified for a multi-binding set that agrees on some but not all bindings", () => {
    // A single-binding implementation comparing only the first stored entry
    // would report this pair as "same" - the third binding's opcode changed
    // (0x22 -> 0x23) but the first two entries are identical.
    const tenant = makeObject("tnt", "tenant", 83, 1, {
      NoOpHandler: [binding("0x17"), binding("0x19"), binding("0x23")],
    });
    const anc = makeObject("t83", "template", 83, 1, {
      NoOpHandler: [binding("0x17"), binding("0x19"), binding("0x22")],
    });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "NoOpHandler")).toBe(
      "modified",
    );
  });

  it("returns tenant-only when the ancestor has no such definition", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, {
      Custom: [binding("0x7F")],
    });
    const anc = makeObject("t83", "template", 83, 1, {});
    expect(classifyAgainstAncestor(tenant, anc, "handler", "Custom")).toBe(
      "tenant-only",
    );
  });

  it("returns missing when the ancestor defines it and the tenant does not", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, {});
    const anc = makeObject("t83", "template", 83, 1, {
      LoginHandle: [binding("0x01")],
    });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "LoginHandle")).toBe(
      "missing",
    );
  });

  it("returns unsupported when the tenant marked it so, whatever the ancestor says", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, {}, ["GuestLoginHandle"]);
    const anc = makeObject("t83", "template", 83, 1, {
      GuestLoginHandle: [binding("0x02")],
    });
    expect(
      classifyAgainstAncestor(tenant, anc, "handler", "GuestLoginHandle"),
    ).toBe("unsupported");
  });

  it("lets a live binding win over a stale Unsupported marking (stateOf's defined-wins rule)", () => {
    // model.ts's stateOf: a name can be BOTH defined and listed unsupported
    // at once (a handler re-added after being marked unsupported, where the
    // unsupported entry was never cleaned up) - "defined" wins. This proves
    // classifyAgainstAncestor inherits that priority rather than checking
    // the unsupported set first.
    const tenant = makeObject(
      "tnt",
      "tenant",
      83,
      1,
      { GuestLoginHandle: [binding("0x02")] },
      ["GuestLoginHandle"],
    );
    const anc = makeObject("t83", "template", 83, 1, {
      GuestLoginHandle: [binding("0x02")],
    });
    expect(
      classifyAgainstAncestor(tenant, anc, "handler", "GuestLoginHandle"),
    ).toBe("same");
  });
});

describe("missingFromTenant", () => {
  it("lists only names defined in the ancestor and undefined in the tenant", () => {
    const tenant = makeObject(
      "tnt",
      "tenant",
      83,
      1,
      { LoginHandle: [binding("0x01")] },
      ["GuestLoginHandle"],
    );
    const anc = makeObject("t83", "template", 83, 1, {
      LoginHandle: [binding("0x01")],
      PongHandle: [binding("0x18")],
      GuestLoginHandle: [binding("0x02")],
    });
    // LoginHandle is already defined; GuestLoginHandle is explicitly unsupported
    // and so is NOT undefined (FR-9.5 excludes it unless the user opts in).
    expect(missingFromTenant(tenant, anc, "handler")).toEqual(["PongHandle"]);
  });

  it("returns an empty list when the tenant defines everything", () => {
    const b = { LoginHandle: [binding("0x01")] };
    const tenant = makeObject("tnt", "tenant", 83, 1, b);
    const anc = makeObject("t83", "template", 83, 1, b);
    expect(missingFromTenant(tenant, anc, "handler")).toEqual([]);
  });
});
