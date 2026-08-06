import { describe, expect, it } from "vitest";
import { fromTemplate, fromTenantConfig } from "@/lib/socket/normalize";
import { nameOfEntry, stateOf } from "@/lib/socket/model";
import type { Template } from "@/types/models/template";
import type { TenantConfig } from "@/services/api/tenants.service";

function template(overrides: Partial<Template["attributes"]> = {}): Template {
  return {
    id: "tpl-1",
    attributes: {
      region: "GMS",
      majorVersion: 95,
      minorVersion: 1,
      usesPin: false,
      characters: { templates: [], presets: [] },
      npcs: [],
      worlds: [],
      socket: { handlers: [], writers: [] },
      ...overrides,
    } as Template["attributes"],
  };
}

describe("fromTemplate", () => {
  it("groups several bindings of one name under that name", () => {
    const obj = fromTemplate(
      template({
        socket: {
          handlers: [
            {
              opCode: "0x17",
              validator: "LoggedInValidator",
              handler: "NoOpHandler",
            },
            {
              opCode: "0x19",
              validator: "LoggedInValidator",
              handler: "NoOpHandler",
            },
            {
              opCode: "0x22",
              validator: "LoggedInValidator",
              handler: "NoOpHandler",
            },
            {
              opCode: "0x24",
              validator: "LoggedInValidator",
              handler: "NoOpHandler",
            },
            {
              opCode: "0x01",
              validator: "NoOpValidator",
              handler: "LoginHandle",
            },
          ],
          writers: [],
        },
      }),
    );
    expect(obj.handlers.get("NoOpHandler")).toHaveLength(4);
    expect(obj.handlers.get("NoOpHandler")!.map((b) => b.opCodeValue)).toEqual([
      0x17, 0x19, 0x22, 0x24,
    ]);
    expect(obj.handlers.get("LoginHandle")).toHaveLength(1);
  });

  it("keeps numerically-equal bindings as two separate bindings", () => {
    const obj = fromTemplate(
      template({
        socket: {
          handlers: [],
          writers: [
            { opCode: "0xB8", writer: "MiniRoom", options: {} },
            { opCode: "0x0B8", writer: "MiniRoom" },
          ],
        },
      }),
    );
    const bindings = obj.writers.get("MiniRoom")!;
    expect(bindings).toHaveLength(2);
    expect(bindings[0]!.opCodeValue).toBe(bindings[1]!.opCodeValue);
    // The stored strings are preserved verbatim; canonicalization is display-only.
    expect(bindings.map((b) => b.opCode)).toEqual(["0xB8", "0x0B8"]);
    // options is preserved when present, absent (undefined) when the key was absent.
    expect(bindings[0]!.options).toEqual({});
    expect(bindings[1]!.options).toBeUndefined();
  });

  it("treats an absent unsupported key as two empty sets", () => {
    const obj = fromTemplate(template());
    expect(obj.unsupportedHandlers.size).toBe(0);
    expect(obj.unsupportedWriters.size).toBe(0);
  });

  it("reads the three definition states, including defined-and-unsupported", () => {
    const obj = fromTemplate(
      template({
        socket: {
          handlers: [
            {
              opCode: "0x01",
              validator: "NoOpValidator",
              handler: "LoginHandle",
            },
            {
              opCode: "0x02",
              validator: "NoOpValidator",
              handler: "BothDefinedAndUnsupported",
            },
          ],
          writers: [],
          unsupported: {
            handlers: ["GuestLoginHandle", "BothDefinedAndUnsupported"],
            writers: [],
          },
        },
      }),
    );
    expect(stateOf(obj, "handler", "LoginHandle")).toBe("defined");
    expect(stateOf(obj, "handler", "GuestLoginHandle")).toBe("unsupported");
    expect(stateOf(obj, "handler", "NeverHeardOfIt")).toBe("undefined");
    // A name that is BOTH bound and listed unsupported: "defined" wins,
    // because an actual binding is a stronger fact than an audit-time
    // assertion the packet is absent. See the DefinitionState doc comment.
    expect(stateOf(obj, "handler", "BothDefinedAndUnsupported")).toBe(
      "defined",
    );
  });

  it("records a malformed opcode as a null value rather than throwing", () => {
    const obj = fromTemplate(
      template({
        socket: {
          handlers: [],
          writers: [{ opCode: "nonsense", writer: "Broken" }],
        },
      }),
    );
    expect(obj.writers.get("Broken")![0]!.opCodeValue).toBeNull();
  });

  it("carries fname through when present and leaves it undefined when absent", () => {
    const obj = fromTemplate(
      template({
        socket: {
          handlers: [
            {
              opCode: "0x01",
              validator: "NoOpValidator",
              handler: "LoginHandle",
              fname: "CLogin::SendCheckPasswordPacket",
            },
          ],
          writers: [{ opCode: "0x00", writer: "AuthLoginFailed" }],
        },
      }),
    );
    expect(obj.handlers.get("LoginHandle")![0]!.fname).toBe(
      "CLogin::SendCheckPasswordPacket",
    );
    expect(obj.writers.get("AuthLoginFailed")![0]!.fname).toBeUndefined();
  });

  it("labels the object from its region and version", () => {
    expect(fromTemplate(template()).label).toBe("GMS v95.1");
    expect(fromTemplate(template()).source).toBe("template");
  });
});

describe("fromTenantConfig", () => {
  function tenantConfig(
    overrides: Partial<TenantConfig["attributes"]> = {},
  ): TenantConfig {
    return {
      id: "tnt-1",
      attributes: {
        region: "GMS",
        majorVersion: 83,
        minorVersion: 1,
        usesPin: false,
        characters: { templates: [], presets: [] },
        npcs: [],
        worlds: [],
        socket: { handlers: [], writers: [] },
        ...overrides,
      } as TenantConfig["attributes"],
    };
  }

  it("produces the same shape from a tenant configuration with unsupported present", () => {
    const obj = fromTenantConfig(
      tenantConfig({
        socket: {
          handlers: [
            {
              opCode: "0x01",
              validator: "NoOpValidator",
              handler: "LoginHandle",
            },
          ],
          writers: [],
          unsupported: { handlers: [], writers: ["MonsterCarnival"] },
        },
      }),
    );

    expect(obj.source).toBe("tenant");
    expect(obj.key).toBe("tnt-1");
    expect(obj.label).toBe("GMS v83.1");
    expect(stateOf(obj, "writer", "MonsterCarnival")).toBe("unsupported");
    expect(stateOf(obj, "handler", "LoginHandle")).toBe("defined");
  });

  it("treats an absent unsupported key (older data) as two empty sets", () => {
    const obj = fromTenantConfig(
      tenantConfig({
        socket: {
          handlers: [
            {
              opCode: "0x01",
              validator: "NoOpValidator",
              handler: "LoginHandle",
            },
          ],
          writers: [{ opCode: "0x00", writer: "AuthLoginFailed" }],
        },
      }),
    );
    expect(obj.unsupportedHandlers.size).toBe(0);
    expect(obj.unsupportedWriters.size).toBe(0);
    expect(obj.handlers.get("LoginHandle")![0]!.options).toBeUndefined();
    expect(obj.writers.get("AuthLoginFailed")![0]!.options).toBeUndefined();
  });
});

describe("nameOfEntry", () => {
  it("reads the name from the handler key for a handler entry", () => {
    const entry = {
      opCode: "0x01",
      validator: "NoOpValidator",
      handler: "LoginHandle",
    };
    expect(nameOfEntry(entry, "handler")).toBe("LoginHandle");
  });

  it("reads the name from the writer key for a writer entry", () => {
    const entry = { opCode: "0x00", writer: "AuthLoginFailed" };
    expect(nameOfEntry(entry, "writer")).toBe("AuthLoginFailed");
  });
});
