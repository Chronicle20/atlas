import { describe, it, expect } from "vitest";
import { formatStatup, formatDurationMs } from "../skill-effect-format";

describe("formatStatup", () => {
  it("formats WeaponAttack with duration", () => {
    expect(formatStatup({ type: "WeaponAttack", amount: 10 }, 30000)).toBe("+10 Weapon Attack for 30s");
  });
  it("falls back for unknown types", () => {
    expect(formatStatup({ type: "UnknownStat", amount: 5 }, 0)).toBe("UnknownStat: +5");
  });
  it("omits duration block when 0", () => {
    expect(formatStatup({ type: "WeaponDefense", amount: 5 }, 0)).toBe("+5 Weapon Defense");
  });
});

describe("formatDurationMs", () => {
  it("converts ms to s", () => {
    expect(formatDurationMs(30000)).toBe(" for 30s");
  });
  it("returns empty for 0", () => {
    expect(formatDurationMs(0)).toBe("");
  });
});
