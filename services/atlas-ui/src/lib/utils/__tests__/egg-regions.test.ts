import { describe, it, expect } from "vitest";
import { eggRegionLabel, formatIncubatorName } from "../egg-regions";

describe("eggRegionLabel", () => {
  it("resolves a known egg id (number)", () => {
    expect(eggRegionLabel(4170000)).toBe("Henesys");
  });

  it("resolves a known egg id passed as a string", () => {
    expect(eggRegionLabel("4170005")).toBe("Ludibrium");
  });

  it("returns null for the intentionally-unconfirmed egg id", () => {
    expect(eggRegionLabel(4170008)).toBeNull();
  });

  it("returns null for a non-egg id", () => {
    expect(eggRegionLabel(9999999)).toBeNull();
  });
});

describe("formatIncubatorName", () => {
  it("appends the region when known", () => {
    expect(formatIncubatorName("Pigmy Egg", 4170001)).toBe(
      "Pigmy Egg (Ellinia)",
    );
  });

  it("leaves the name unchanged when the region is unknown", () => {
    expect(formatIncubatorName("Pigmy Egg", 4170008)).toBe("Pigmy Egg");
  });
});
