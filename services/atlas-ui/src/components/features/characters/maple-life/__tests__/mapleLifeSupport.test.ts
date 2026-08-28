import { describe, expect, it } from "vitest";

import type { SocketConfig } from "@/types/models/socket";
import { MAPLE_LIFE_HANDLER, supportsMapleLife } from "../mapleLifeSupport";

function socket(handlers: SocketConfig["handlers"]): SocketConfig {
  return { handlers, writers: [] };
}

describe("supportsMapleLife", () => {
  it("is false for an absent socket", () => {
    expect(supportsMapleLife(undefined)).toBe(false);
  });

  it("is false when the handler is absent (e.g. gms_84_1)", () => {
    expect(
      supportsMapleLife(
        socket([
          {
            opCode: "0x100",
            validator: "SomeValidator",
            handler: "SomeOtherHandle",
          },
        ]),
      ),
    ).toBe(false);
  });

  it("is true when the handler is present, regardless of opcode", () => {
    for (const opCode of ["0x100", "0x10E", "0x12D", "0x137"]) {
      expect(
        supportsMapleLife(
          socket([
            {
              opCode,
              validator: "MapleLifeValidator",
              handler: MAPLE_LIFE_HANDLER,
            },
          ]),
        ),
      ).toBe(true);
    }
  });

  it("is false for an empty handlers list", () => {
    expect(supportsMapleLife(socket([]))).toBe(false);
  });
});
