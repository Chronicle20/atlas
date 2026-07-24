import { describe, it, expect } from "vitest";
import {
  RAIL_GROUPS,
  branchEntryOf,
  visibleRailGroups,
} from "@/components/features/jobs/rail-groups";

describe("RAIL_GROUPS", () => {
  it("defines the four labeled groups with the PRD's entries in order", () => {
    expect(RAIL_GROUPS.map((g) => g.label)).toEqual([
      "Explorers",
      "Cygnus Knights",
      "Legends",
      "Special",
    ]);
    expect(RAIL_GROUPS[0]!.entries.map((e) => e.id)).toEqual([
      100, 200, 300, 400, 500,
    ]);
    expect(RAIL_GROUPS[1]!.entries.map((e) => e.id)).toEqual([1000]);
    expect(RAIL_GROUPS[2]!.entries.map((e) => e.id)).toEqual([2000, 2001]);
    expect(RAIL_GROUPS[3]!.entries.map((e) => e.id)).toEqual([800, 900]);
  });

  it("maps every entry to a --c-* accent token name", () => {
    for (const g of RAIL_GROUPS) {
      for (const e of g.entries) {
        expect(e.accent).toMatch(/^--c-[a-z]+$/);
      }
    }
    expect(RAIL_GROUPS[0]!.entries[0]!.accent).toBe("--c-warrior");
    expect(RAIL_GROUPS[3]!.entries[1]!.accent).toBe("--c-special");
  });
});

describe("branchEntryOf", () => {
  it("resolves any node to the rail entry on its path", () => {
    expect(branchEntryOf(112).id).toBe(100); // Hero -> Warrior
    expect(branchEntryOf(910).id).toBe(900); // Super GM -> GM entry
    expect(branchEntryOf(1512).id).toBe(1000); // Thunder Breaker 4 -> Noblesse
    expect(branchEntryOf(2218).id).toBe(2001); // Evan 10 -> Evan
    expect(branchEntryOf(800).id).toBe(800);
  });

  it("falls back to the Warrior entry for Beginner and unknown ids", () => {
    expect(branchEntryOf(0).id).toBe(100);
    expect(branchEntryOf(99999).id).toBe(100);
  });
});

describe("visibleRailGroups", () => {
  it("gates entries by version floor and drops empty groups (GMS v12)", () => {
    const groups = visibleRailGroups(12);
    expect(groups.map((g) => g.label)).toEqual(["Explorers", "Special"]);
    expect(groups[0]!.entries.map((e) => e.id)).toEqual([100, 200, 300, 400]); // no Pirate
    expect(groups[1]!.entries.map((e) => e.id)).toEqual([900]); // no Brigadier (v83)
  });

  it("adds Pirate at v62, Cygnus/Aran/Brigadier at v83, Evan at v84", () => {
    expect(
      visibleRailGroups(62)[0]!.entries.map((e) => e.id),
    ).toContain(500);
    const v83 = visibleRailGroups(83);
    expect(v83.map((g) => g.label)).toEqual([
      "Explorers",
      "Cygnus Knights",
      "Legends",
      "Special",
    ]);
    expect(v83[2]!.entries.map((e) => e.id)).toEqual([2000]); // Evan hidden
    expect(v83[3]!.entries.map((e) => e.id)).toEqual([800, 900]);
    expect(visibleRailGroups(84)[2]!.entries.map((e) => e.id)).toEqual([
      2000, 2001,
    ]);
  });

  it("decorates entries with display name and visible subtree count", () => {
    const v83 = visibleRailGroups(83);
    const warrior = v83[0]!.entries[0]!;
    expect(warrior.name).toBe("Warrior");
    expect(warrior.count).toBe(10);
    const gm = v83[3]!.entries.find((e) => e.id === 900);
    expect(gm?.name).toBe("GM");
    expect(gm?.count).toBe(2);
  });
});
