import { describe, it, expect } from "vitest";
import { incubatorChances, gachaponChances, tierHasMixedWeights } from "../reward-pool-chance";

describe("incubatorChances", () => {
  it("divides weight by total", () => {
    const m = incubatorChances([{ id: "a", weight: 75 }, { id: "b", weight: 25 }]);
    expect(m.get("a")).toBeCloseTo(0.75);
    expect(m.get("b")).toBeCloseTo(0.25);
  });
  it("zero-total pool falls back to uniform 1/N (mirrors server selectItem)", () => {
    const m = incubatorChances([{ id: "a", weight: 0 }, { id: "b", weight: 0 }]);
    expect(m.get("a")).toBeCloseTo(0.5);
    expect(m.get("b")).toBeCloseTo(0.5);
  });
  it("empty pool yields empty map", () => {
    expect(incubatorChances([]).size).toBe(0);
  });
  it("weighted pool gives a zero-weight item exactly 0", () => {
    const m = incubatorChances([{ id: "a", weight: 10 }, { id: "b", weight: 0 }]);
    expect(m.get("a")).toBeCloseTo(1);
    expect(m.get("b")).toBe(0);
  });
});

describe("gachaponChances", () => {
  const tw = { common: 70, uncommon: 25, rare: 5 };

  it("uniform within an unweighted tier, scaled by tier chance", () => {
    const m = gachaponChances(tw, [
      { key: "a", tier: "common", weight: 0 },
      { key: "b", tier: "common", weight: 0 },
    ]);
    expect(m.get("a")!.chance).toBeCloseTo(0.7 / 2);
    expect(m.get("a")!.excluded).toBe(false);
  });

  it("weight-proportional when any row in the tier is weighted; zero-weight rows excluded", () => {
    const m = gachaponChances(tw, [
      { key: "w", tier: "rare", weight: 10 },
      { key: "z", tier: "rare", weight: 0 },
    ]);
    expect(m.get("w")!.chance).toBeCloseTo(0.05);
    expect(m.get("z")!.chance).toBe(0);
    expect(m.get("z")!.excluded).toBe(true);
  });

  it("zero tier-weight sum yields zeros", () => {
    const m = gachaponChances({ common: 0, uncommon: 0, rare: 0 }, [{ key: "a", tier: "common", weight: 0 }]);
    expect(m.get("a")!.chance).toBe(0);
  });
});

describe("tierHasMixedWeights", () => {
  it("detects a mixed tier", () => {
    const rows = [
      { tier: "rare", weight: 10 },
      { tier: "rare", weight: 0 },
      { tier: "common", weight: 0 },
    ];
    expect(tierHasMixedWeights(rows, "rare")).toBe(true);
    expect(tierHasMixedWeights(rows, "common")).toBe(false);
  });
});
