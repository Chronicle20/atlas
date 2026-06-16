import { describe, expect, it } from "vitest";
import { getItemType } from "../item";

describe("getItemType", () => {
  it("classifies equipment, consumable, setup, and etc by prefix", () => {
    expect(getItemType("1302000")).toBe("Equipment");
    expect(getItemType("2000000")).toBe("Consumable");
    expect(getItemType("3010000")).toBe("Setup");
    expect(getItemType("4000000")).toBe("Etc");
  });

  it("classifies pet items (classification 500) as Pet", () => {
    expect(getItemType("5000029")).toBe("Pet");
    expect(getItemType("5000000")).toBe("Pet");
    expect(getItemType("5009999")).toBe("Pet");
  });

  it("classifies non-pet cash items (classification 501+) as Cash", () => {
    expect(getItemType("5010000")).toBe("Cash"); // character effect
    expect(getItemType("5040000")).toBe("Cash"); // megaphone
  });

  it("returns Unknown for non-numeric or out-of-range ids", () => {
    expect(getItemType("abc")).toBe("Unknown");
    expect(getItemType("9000000")).toBe("Unknown");
  });
});
