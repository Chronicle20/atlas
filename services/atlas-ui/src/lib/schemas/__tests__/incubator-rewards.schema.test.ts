import { describe, it, expect } from "vitest";
import { incubatorRewardSchema } from "../incubator-rewards.schema";

describe("incubatorRewardSchema", () => {
  it("accepts positive integers", () => {
    expect(
      incubatorRewardSchema.safeParse({ eggId: 4170005, itemId: 2000000, quantity: 1, weight: 50 }).success,
    ).toBe(true);
  });
  it("rejects zero weight", () => {
    expect(
      incubatorRewardSchema.safeParse({ eggId: 4170005, itemId: 2000000, quantity: 1, weight: 0 }).success,
    ).toBe(false);
  });
  it("rejects non-integer itemId", () => {
    expect(
      incubatorRewardSchema.safeParse({ eggId: 4170005, itemId: 1.5, quantity: 1, weight: 5 }).success,
    ).toBe(false);
  });
  it("rejects an eggId out of the 4170000-4170009 range", () => {
    expect(
      incubatorRewardSchema.safeParse({ eggId: 4169999, itemId: 2000000, quantity: 1, weight: 5 }).success,
    ).toBe(false);
    expect(
      incubatorRewardSchema.safeParse({ eggId: 4170010, itemId: 2000000, quantity: 1, weight: 5 }).success,
    ).toBe(false);
  });
  it("accepts eggId at both ends of the valid range", () => {
    expect(
      incubatorRewardSchema.safeParse({ eggId: 4170000, itemId: 2000000, quantity: 1, weight: 5 }).success,
    ).toBe(true);
    expect(
      incubatorRewardSchema.safeParse({ eggId: 4170009, itemId: 2000000, quantity: 1, weight: 5 }).success,
    ).toBe(true);
  });
});
