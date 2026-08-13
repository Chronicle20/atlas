import { describe, expect, it } from "vitest";
import {
  MutationError,
  addBinding,
  clearUnsupported,
  copyBindings,
  copyMissingFromAncestor,
  deleteBinding,
  editBinding,
  fillMissingValidators,
  markUnsupported,
} from "@/lib/socket/mutate";
import type { SocketConfig } from "@/types/models/socket";

function config(): SocketConfig {
  return {
    handlers: [
      {
        opCode: "0x01",
        validator: "NoOpValidator",
        handler: "LoginHandle",
        services: ["login"],
      },
      {
        opCode: "0x17",
        validator: "LoggedInValidator",
        handler: "NoOpHandler",
        services: ["channel"],
      },
      {
        opCode: "0x19",
        validator: "LoggedInValidator",
        handler: "NoOpHandler",
        services: ["channel"],
      },
      {
        opCode: "0x22",
        validator: "LoggedInValidator",
        handler: "NoOpHandler",
        services: ["channel"],
      },
    ],
    writers: [{ opCode: "0x00", writer: "AuthSuccess", services: ["login"] }],
    unsupported: { handlers: ["GuestLoginHandle"], writers: [] },
  };
}

/** Recursively freezes a value so any accidental mutation throws (strict mode). Used to prove purity. */
function deepFreeze<T>(value: T): T {
  if (value !== null && typeof value === "object" && !Object.isFrozen(value)) {
    for (const key of Object.getOwnPropertyNames(value)) {
      deepFreeze((value as Record<string, unknown>)[key]);
    }
    Object.freeze(value);
  }
  return value;
}

describe("addBinding", () => {
  it("appends the binding", () => {
    const out = addBinding(config(), "writer", "PetActivated", {
      opCode: "0x9A",
      services: ["channel"],
    });
    expect(out.writers).toHaveLength(2);
    expect(out.writers[1]).toMatchObject({
      opCode: "0x9A",
      writer: "PetActivated",
    });
  });

  // FR-1.2
  it("clears any unsupported marker for that name", () => {
    const out = addBinding(config(), "handler", "GuestLoginHandle", {
      opCode: "0x02",
      validator: "NoOpValidator",
      services: ["login"],
    });
    expect(out.unsupported!.handlers).toEqual([]);
  });

  it("adds a second binding to an existing name without touching the first", () => {
    const out = addBinding(config(), "handler", "NoOpHandler", {
      opCode: "0x24",
      validator: "LoggedInValidator",
      services: ["channel"],
    });
    expect(
      out.handlers.filter((h) => h.handler === "NoOpHandler"),
    ).toHaveLength(4);
  });

  it("does not mutate the input", () => {
    const input = config();
    addBinding(input, "writer", "PetActivated", {
      opCode: "0x9A",
      services: ["channel"],
    });
    expect(input.writers).toHaveLength(1);
  });

  it("rejects a binding at an opcode the name already uses", () => {
    expect(() =>
      addBinding(config(), "handler", "NoOpHandler", {
        opCode: "0x017",
        validator: "LoggedInValidator",
        services: ["channel"],
      }),
    ).toThrow(MutationError);
  });

  // Corpus fact: the 25 entries with no services omit the `services` key
  // entirely (measured: services present+non-empty 2834, absent 25, present
  // as `[]` 0). An empty BindingInput.services must round-trip the same way
  // rather than writing a shape ("services": []) that never occurs.
  it("omits the services key entirely when the input has none", () => {
    const out = addBinding(config(), "writer", "PetActivated", {
      opCode: "0x9A",
      services: [],
    });
    const added = out.writers[1]!;
    expect("services" in added).toBe(false);
  });

  // formatOpcode zero-pads to at least 2 digits; reusing it (rather than a
  // re-derived hex formatter) keeps error-message opcodes consistent with
  // every other display path in the domain layer.
  it("formats a single-hex-digit opcode zero-padded in the collision error", () => {
    const single: SocketConfig = {
      handlers: [],
      writers: [
        { opCode: "0x9", writer: "PetActivated", services: ["channel"] },
      ],
    };
    expect(() =>
      addBinding(single, "writer", "PetActivated", {
        opCode: "0x09",
        services: ["channel"],
      }),
    ).toThrow(/0x09/);
  });
});

describe("editBinding", () => {
  it("edits exactly the addressed binding of a multi-binding name", () => {
    const out = editBinding(config(), "handler", "NoOpHandler", 0x19, {
      opCode: "0x1A",
      validator: "NoOpValidator",
      services: ["channel", "login"],
    });
    const noop = out.handlers.filter((h) => h.handler === "NoOpHandler");
    expect(noop.map((h) => h.opCode)).toEqual(["0x17", "0x1A", "0x22"]);
    expect(noop[1]!.validator).toBe("NoOpValidator");
    expect(noop[0]!.validator).toBe("LoggedInValidator");
  });

  it("throws when the binding does not resolve", () => {
    expect(() =>
      editBinding(config(), "handler", "NoOpHandler", 0xff, {
        opCode: "0xFF",
        validator: "NoOpValidator",
        services: [],
      }),
    ).toThrow(MutationError);
  });

  it("throws when the binding resolves more than once", () => {
    const dup: SocketConfig = {
      handlers: [],
      writers: [
        { opCode: "0xB8", writer: "MiniRoom", services: ["channel"] },
        { opCode: "0x0B8", writer: "MiniRoom", services: ["channel"] },
      ],
    };
    expect(() =>
      editBinding(dup, "writer", "MiniRoom", 0xb8, {
        opCode: "0xB8",
        services: [],
      }),
    ).toThrow(MutationError);
  });

  // Resolution is by NORMALIZED opcode value: "0x0B8" in the request must
  // still resolve to a binding stored as "0xB8".
  it("resolves across zero-padding when addressing the binding to edit", () => {
    const cfg: SocketConfig = {
      handlers: [],
      writers: [{ opCode: "0xB8", writer: "MiniRoom", services: ["channel"] }],
    };
    const out = editBinding(cfg, "writer", "MiniRoom", 0x0b8, {
      opCode: "0xB8",
      services: ["channel", "login"],
    });
    expect(out.writers).toHaveLength(1);
    expect(out.writers[0]!.services).toEqual(["channel", "login"]);
  });

  it("rejects moving to an opcode another binding of the same name already uses", () => {
    expect(() =>
      editBinding(config(), "handler", "NoOpHandler", 0x19, {
        opCode: "0x22",
        validator: "LoggedInValidator",
        services: ["channel"],
      }),
    ).toThrow(MutationError);
  });
});

describe("deleteBinding", () => {
  it("removes exactly one binding of a multi-binding name", () => {
    const out = deleteBinding(config(), "handler", "NoOpHandler", 0x19);
    expect(
      out.handlers
        .filter((h) => h.handler === "NoOpHandler")
        .map((h) => h.opCode),
    ).toEqual(["0x17", "0x22"]);
  });

  // Resolution is by NORMALIZED opcode value, never the stored string form.
  it("resolves 0x0B8 against a binding stored as 0xB8", () => {
    const cfg: SocketConfig = {
      handlers: [],
      writers: [{ opCode: "0xB8", writer: "MiniRoom", services: ["channel"] }],
    };
    const out = deleteBinding(cfg, "writer", "MiniRoom", 0x0b8);
    expect(out.writers).toHaveLength(0);
  });

  // FR-1.4: deleting leaves the definition Undefined; it does NOT mark it
  // unsupported. That is a separate, explicit choice in the dialog.
  it("does not add an unsupported marker", () => {
    const out = deleteBinding(config(), "writer", "AuthSuccess", 0x00);
    expect(out.writers).toHaveLength(0);
    expect(out.unsupported!.writers).toEqual([]);
  });

  it("throws when the binding does not resolve", () => {
    expect(() =>
      deleteBinding(config(), "writer", "AuthSuccess", 0x99),
    ).toThrow(MutationError);
  });

  it("throws when the binding resolves more than once, leaving both intact", () => {
    const dup: SocketConfig = {
      handlers: [],
      writers: [
        { opCode: "0xB8", writer: "MiniRoom", services: ["channel"] },
        { opCode: "0x0B8", writer: "MiniRoom", services: ["channel"] },
      ],
    };
    expect(() => deleteBinding(dup, "writer", "MiniRoom", 0xb8)).toThrow(
      MutationError,
    );
  });
});

describe("markUnsupported", () => {
  // The plural matters: unsupported is NAME-scoped while bindings are
  // OPCODE-scoped, so marking a name necessarily removes all four NoOpHandler
  // routes. The dialog states this before confirming.
  it("removes EVERY binding of the name", () => {
    const out = markUnsupported(config(), "handler", "NoOpHandler");
    expect(
      out.handlers.filter((h) => h.handler === "NoOpHandler"),
    ).toHaveLength(0);
    expect(out.unsupported!.handlers).toContain("NoOpHandler");
  });

  it("is idempotent", () => {
    const once = markUnsupported(config(), "handler", "NoOpHandler");
    const twice = markUnsupported(once, "handler", "NoOpHandler");
    expect(
      twice.unsupported!.handlers.filter((n) => n === "NoOpHandler"),
    ).toHaveLength(1);
  });

  it("works on a name that was never defined", () => {
    const out = markUnsupported(config(), "writer", "MonsterCarnival");
    expect(out.unsupported!.writers).toEqual(["MonsterCarnival"]);
  });
});

describe("clearUnsupported", () => {
  // FR-1.3
  it("returns the definition to Undefined", () => {
    const out = clearUnsupported(config(), "handler", "GuestLoginHandle");
    expect(out.unsupported!.handlers).toEqual([]);
    expect(out.handlers.some((h) => h.handler === "GuestLoginHandle")).toBe(
      false,
    );
  });

  it("is a no-op for a name that is not marked", () => {
    const out = clearUnsupported(config(), "handler", "LoginHandle");
    expect(out.unsupported!.handlers).toEqual(["GuestLoginHandle"]);
  });
});

describe("copyBindings", () => {
  it("adds every supplied binding and clears the unsupported marker", () => {
    const out = copyBindings(config(), "handler", "GuestLoginHandle", [
      { opCode: "0x02", validator: "NoOpValidator", services: ["login"] },
      { opCode: "0x03", validator: "NoOpValidator", services: ["login"] },
    ]);
    expect(
      out.handlers.filter((h) => h.handler === "GuestLoginHandle"),
    ).toHaveLength(2);
    expect(out.unsupported!.handlers).toEqual([]);
  });

  it("produces a result independent of the source", () => {
    const options = { types: ["WALK"] };
    const out = copyBindings(config(), "writer", "CharacterMovement", [
      { opCode: "0xB9", services: ["channel"], options },
    ]);
    (options.types as string[]).push("STAND");
    const copied = out.writers.find((w) => w.writer === "CharacterMovement")!;
    expect((copied.options as { types: string[] }).types).toEqual(["WALK"]);
  });
});

describe("copyMissingFromAncestor", () => {
  const additions = [
    {
      name: "PongHandle",
      bindings: [
        { opCode: "0x18", validator: "NoOpValidator", services: ["channel"] },
      ],
    },
    {
      name: "LoginHandle",
      bindings: [
        { opCode: "0xFF", validator: "NoOpValidator", services: ["login"] },
      ],
    },
  ];

  // FR-9.4
  it("never overwrites an already-defined definition", () => {
    const out = copyMissingFromAncestor(config(), "handler", additions);
    const login = out.handlers.filter((h) => h.handler === "LoginHandle");
    expect(login).toHaveLength(1);
    expect(login[0]!.opCode).toBe("0x01");
  });

  it("adds the definitions that were undefined", () => {
    const out = copyMissingFromAncestor(config(), "handler", additions);
    expect(out.handlers.filter((h) => h.handler === "PongHandle")).toHaveLength(
      1,
    );
  });

  // FR-9.6
  it("applies the whole selection as one document", () => {
    const out = copyMissingFromAncestor(config(), "handler", [
      {
        name: "A",
        bindings: [
          { opCode: "0x30", validator: "NoOpValidator", services: ["channel"] },
        ],
      },
      {
        name: "B",
        bindings: [
          { opCode: "0x31", validator: "NoOpValidator", services: ["channel"] },
        ],
      },
      {
        name: "C",
        bindings: [
          { opCode: "0x32", validator: "NoOpValidator", services: ["channel"] },
        ],
      },
    ]);
    expect(out.handlers).toHaveLength(config().handlers.length + 3);
  });

  it("clears the unsupported marker for any name it adds", () => {
    const out = copyMissingFromAncestor(config(), "handler", [
      {
        name: "GuestLoginHandle",
        bindings: [
          { opCode: "0x02", validator: "NoOpValidator", services: ["login"] },
        ],
      },
    ]);
    expect(out.unsupported!.handlers).toEqual([]);
  });
});

describe("fillMissingValidators", () => {
  // Strict FR-11.4 blocks any save of a document carrying an empty validator,
  // and saves are whole-document, so a single-definition edit can never be the
  // fix. This repairs every offender in one write.
  it("fills every empty handler validator in one pass", () => {
    const broken: SocketConfig = {
      handlers: [
        { opCode: "0x01", validator: "", handler: "A", services: ["channel"] },
        {
          opCode: "0x02",
          validator: "   ",
          handler: "B",
          services: ["channel"],
        },
        {
          opCode: "0x03",
          validator: "LoggedInValidator",
          handler: "C",
          services: ["channel"],
        },
      ],
      writers: [],
    };
    const out = fillMissingValidators(broken, "NoOpValidator");
    expect(out.handlers.map((h) => h.validator)).toEqual([
      "NoOpValidator",
      "NoOpValidator",
      "LoggedInValidator",
    ]);
  });

  it("leaves writers alone - they carry no validator", () => {
    const cfg = config();
    const out = fillMissingValidators(cfg, "NoOpValidator");
    expect(out.writers).toEqual(cfg.writers);
  });

  // The seed corpus has zero handler entries with an absent or null
  // `validator`, so no fixture built from real seed data can exercise this.
  // The 32 malformed validators this function exists to repair live in a
  // live gms_95 tenant, not the seed templates - the SocketHandlerEntry type
  // declares `validator: string` (required), but runtime data is not
  // guaranteed to honor that, which is exactly why the escape hatch exists.
  // Built by hand, not derived from any fixture: one entry with the key
  // absent entirely, one with an explicit `null`.
  it("fills a handler validator that is absent or null at runtime, not merely empty", () => {
    const broken = {
      handlers: [
        { opCode: "0x01", handler: "A", services: ["channel"] },
        {
          opCode: "0x02",
          handler: "B",
          services: ["channel"],
          validator: null,
        },
        {
          opCode: "0x03",
          handler: "C",
          services: ["channel"],
          validator: "LoggedInValidator",
        },
      ],
      writers: [],
    } as unknown as SocketConfig;

    expect(() => fillMissingValidators(broken, "NoOpValidator")).not.toThrow();
    const out = fillMissingValidators(broken, "NoOpValidator");
    expect(out.handlers.map((h) => h.validator)).toEqual([
      "NoOpValidator",
      "NoOpValidator",
      "LoggedInValidator",
    ]);
  });
});

// Every mutating function returns a NEW object graph and shares no mutable
// structure with its input. Deep-freezing the input turns any accidental
// in-place write into a thrown TypeError (strict mode), so "did not throw"
// is direct proof the function never wrote through the input, on top of the
// "did not mutate" assertions above. One representative per function group.
describe("purity (deep-frozen input)", () => {
  it("addBinding does not throw against a frozen config and leaves it unchanged", () => {
    const frozen = deepFreeze(config());
    const before = JSON.stringify(frozen);
    expect(() =>
      addBinding(frozen, "writer", "PetActivated", {
        opCode: "0x9A",
        services: ["channel"],
      }),
    ).not.toThrow();
    expect(JSON.stringify(frozen)).toBe(before);
  });

  it("editBinding does not throw against a frozen config and leaves it unchanged", () => {
    const frozen = deepFreeze(config());
    const before = JSON.stringify(frozen);
    expect(() =>
      editBinding(frozen, "handler", "NoOpHandler", 0x19, {
        opCode: "0x1A",
        validator: "NoOpValidator",
        services: ["channel"],
      }),
    ).not.toThrow();
    expect(JSON.stringify(frozen)).toBe(before);
  });

  it("deleteBinding does not throw against a frozen config and leaves it unchanged", () => {
    const frozen = deepFreeze(config());
    const before = JSON.stringify(frozen);
    expect(() =>
      deleteBinding(frozen, "handler", "NoOpHandler", 0x19),
    ).not.toThrow();
    expect(JSON.stringify(frozen)).toBe(before);
  });

  it("markUnsupported / clearUnsupported do not throw against a frozen config and leave it unchanged", () => {
    const frozen = deepFreeze(config());
    const before = JSON.stringify(frozen);
    expect(() =>
      markUnsupported(frozen, "handler", "NoOpHandler"),
    ).not.toThrow();
    expect(() =>
      clearUnsupported(frozen, "handler", "GuestLoginHandle"),
    ).not.toThrow();
    expect(JSON.stringify(frozen)).toBe(before);
  });

  it("copyBindings / copyMissingFromAncestor do not throw against a frozen config and leave it unchanged", () => {
    const frozen = deepFreeze(config());
    const before = JSON.stringify(frozen);
    expect(() =>
      copyBindings(frozen, "handler", "GuestLoginHandle", [
        { opCode: "0x02", validator: "NoOpValidator", services: ["login"] },
      ]),
    ).not.toThrow();
    expect(() =>
      copyMissingFromAncestor(frozen, "handler", [
        {
          name: "PongHandle",
          bindings: [
            {
              opCode: "0x18",
              validator: "NoOpValidator",
              services: ["channel"],
            },
          ],
        },
      ]),
    ).not.toThrow();
    expect(JSON.stringify(frozen)).toBe(before);
  });

  it("fillMissingValidators does not throw against a frozen config and leaves it unchanged", () => {
    const frozen = deepFreeze<SocketConfig>({
      handlers: [
        { opCode: "0x01", validator: "", handler: "A", services: ["channel"] },
      ],
      writers: [],
    });
    const before = JSON.stringify(frozen);
    expect(() => fillMissingValidators(frozen, "NoOpValidator")).not.toThrow();
    expect(JSON.stringify(frozen)).toBe(before);
  });
});
