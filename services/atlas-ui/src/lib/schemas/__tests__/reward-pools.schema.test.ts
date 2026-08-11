import { describe, it, expect } from "vitest";
import {
  gachaponPoolSchema,
  incubatorPoolSchema,
  tierItemSchema,
  weightItemSchema,
  cashSurprisePoolSchema,
  cashSurpriseItemSchema,
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

describe("cashSurprisePoolSchema", () => {
  it("accepts a valid pool", () => {
    expect(
      cashSurprisePoolSchema.safeParse({
        boxItemId: 5222000,
        name: "Surprise Box",
      }).success,
    ).toBe(true);
  });
  it("requires a positive box item id", () => {
    expect(
      cashSurprisePoolSchema.safeParse({ boxItemId: 0, name: "X" }).success,
    ).toBe(false);
    expect(
      cashSurprisePoolSchema.safeParse({ boxItemId: -1, name: "X" }).success,
    ).toBe(false);
  });
  it("requires a non-empty name", () => {
    expect(
      cashSurprisePoolSchema.safeParse({ boxItemId: 5222000, name: "" })
        .success,
    ).toBe(false);
  });
});

describe("cashSurpriseItemSchema", () => {
  it("accepts a valid entry", () => {
    expect(
      cashSurpriseItemSchema.safeParse({
        itemId: 2000000,
        quantity: 1,
        weight: 50,
        commodityId: 100200300,
      }).success,
    ).toBe(true);
  });
  it("requires a positive, non-zero commodityId", () => {
    expect(
      cashSurpriseItemSchema.safeParse({
        itemId: 2000000,
        quantity: 1,
        weight: 50,
        commodityId: 0,
      }).success,
    ).toBe(false);
    expect(
      cashSurpriseItemSchema.safeParse({
        itemId: 2000000,
        quantity: 1,
        weight: 50,
        commodityId: -1,
      }).success,
    ).toBe(false);
  });
  it("requires weight >= 1", () => {
    expect(
      cashSurpriseItemSchema.safeParse({
        itemId: 2000000,
        quantity: 1,
        weight: 0,
        commodityId: 100200300,
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
