import { describe, it, expect } from "vitest";
import {
  gachaponPoolSchema,
  incubatorPoolSchema,
  tierItemSchema,
  weightItemSchema,
} from "../reward-pools.schema";

describe("gachaponPoolSchema", () => {
  it("accepts a valid pool", () => {
    expect(
      gachaponPoolSchema.safeParse({
        name: "Henesys",
        npcIds: [9100100],
        commonWeight: 70,
        uncommonWeight: 25,
        rareWeight: 5,
      }).success,
    ).toBe(true);
  });
  it("rejects an all-zero tier-weight sum", () => {
    expect(
      gachaponPoolSchema.safeParse({
        name: "X",
        npcIds: [],
        commonWeight: 0,
        uncommonWeight: 0,
        rareWeight: 0,
      }).success,
    ).toBe(false);
  });
});

describe("incubatorPoolSchema", () => {
  it("requires a positive egg item id", () => {
    expect(
      incubatorPoolSchema.safeParse({
        eggItemId: 4170001,
        name: "Pigmy Egg (Victoria)",
        successNpcId: 1012004,
      }).success,
    ).toBe(true);
    expect(
      incubatorPoolSchema.safeParse({
        eggItemId: 0,
        name: "X",
        successNpcId: 1,
      }).success,
    ).toBe(false);
  });
});

describe("item schemas", () => {
  it("tierItemSchema enforces the tier enum", () => {
    expect(
      tierItemSchema.safeParse({ itemId: 2000000, quantity: 1, tier: "common" })
        .success,
    ).toBe(true);
    expect(
      tierItemSchema.safeParse({ itemId: 2000000, quantity: 1, tier: "epic" })
        .success,
    ).toBe(false);
  });
  it("weightItemSchema requires weight ≥ 1", () => {
    expect(
      weightItemSchema.safeParse({ itemId: 2000000, quantity: 1, weight: 50 })
        .success,
    ).toBe(true);
    expect(
      weightItemSchema.safeParse({ itemId: 2000000, quantity: 1, weight: 0 })
        .success,
    ).toBe(false);
  });
});
