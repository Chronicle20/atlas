import { describe, it, expect } from "vitest";
import { incubatorRewardSchema } from "../incubator-rewards.schema";

describe("incubatorRewardSchema", () => {
  it("accepts positive integers", () => {
    expect(incubatorRewardSchema.safeParse({ itemId: 2000000, quantity: 1, weight: 50 }).success).toBe(true);
  });
  it("rejects zero weight", () => {
    expect(incubatorRewardSchema.safeParse({ itemId: 2000000, quantity: 1, weight: 0 }).success).toBe(false);
  });
  it("rejects non-integer itemId", () => {
    expect(incubatorRewardSchema.safeParse({ itemId: 1.5, quantity: 1, weight: 5 }).success).toBe(false);
  });
});
