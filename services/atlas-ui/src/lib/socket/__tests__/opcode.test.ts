import { describe, expect, it } from "vitest";
import {
  formatOpcode,
  matchesOpcodeQuery,
  parseOpcode,
} from "@/lib/socket/opcode";

describe("parseOpcode", () => {
  it("parses the stored wire forms", () => {
    expect(parseOpcode("0x2A")).toBe(42);
    expect(parseOpcode("0x2a")).toBe(42);
    expect(parseOpcode("0X2A")).toBe(42);
    // jms_185_1 stores a single-digit opcode.
    expect(parseOpcode("0x9")).toBe(9);
    // gms_84_1 stores a three-digit padded opcode.
    expect(parseOpcode("0x0A5")).toBe(165);
    expect(parseOpcode("0xFFFF")).toBe(65535);
  });

  it("returns null for anything that is not a stored opcode", () => {
    expect(parseOpcode("2A")).toBeNull();
    expect(parseOpcode("42")).toBeNull();
    expect(parseOpcode("")).toBeNull();
    expect(parseOpcode("0x")).toBeNull();
    expect(parseOpcode("0xZZ")).toBeNull();
    expect(parseOpcode("0x10000")).toBeNull();
  });

  it("treats 0xB8 and 0x0B8 as the same value", () => {
    expect(parseOpcode("0xB8")).toBe(parseOpcode("0x0B8"));
  });
});

describe("formatOpcode", () => {
  it("renders the canonical two-digit-minimum upper-case form", () => {
    expect(formatOpcode(42)).toBe("0x2A");
    expect(formatOpcode(9)).toBe("0x09");
    expect(formatOpcode(165)).toBe("0xA5");
    expect(formatOpcode(65535)).toBe("0xFFFF");
  });

  it("round-trips with parseOpcode", () => {
    for (const n of [0, 9, 42, 165, 184, 65535]) {
      expect(parseOpcode(formatOpcode(n))).toBe(n);
    }
  });
});

// FR-4.3: searching 0x2A, 2A and 42 must all match the same cell. A bare
// numeric query is ambiguous, so it matches under BOTH a hex and a decimal
// reading - "42" therefore matches 0x42 (66) as well as 0x2A (42).
describe("matchesOpcodeQuery", () => {
  it("matches the prefixed form exactly", () => {
    expect(matchesOpcodeQuery("0x2A", 42)).toBe(true);
    expect(matchesOpcodeQuery("0x2a", 42)).toBe(true);
    expect(matchesOpcodeQuery("0x2A", 66)).toBe(false);
  });

  it("matches an unprefixed hex query", () => {
    expect(matchesOpcodeQuery("2A", 42)).toBe(true);
    expect(matchesOpcodeQuery("2a", 42)).toBe(true);
  });

  it("matches a bare number under both hex and decimal readings", () => {
    expect(matchesOpcodeQuery("42", 42)).toBe(true); // decimal reading
    expect(matchesOpcodeQuery("42", 66)).toBe(true); // hex reading, 0x42
  });

  it("ignores surrounding whitespace and does not match non-numeric text", () => {
    expect(matchesOpcodeQuery("  0x2A  ", 42)).toBe(true);
    expect(matchesOpcodeQuery("LoginHandle", 42)).toBe(false);
    expect(matchesOpcodeQuery("", 42)).toBe(false);
  });
});
