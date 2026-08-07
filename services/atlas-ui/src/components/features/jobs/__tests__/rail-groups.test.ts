import { describe, it, expect } from "vitest";
import { JOB_GRAPH } from "@/lib/jobs/job-advancement-tree";
import {
  RAIL_GROUPS,
  branchEntryOf,
  visibleRailGroups,
} from "@/components/features/jobs/rail-groups";

/** Every id in the graph — the "modern tenant has everything" case. */
const ALL_IDS: ReadonlySet<number> = new Set(
  Object.values(JOB_GRAPH).map((e) => e.id),
);

/** Everything except the Evan branch — a tenant whose ingest predates Evan. */
const NO_EVAN: ReadonlySet<number> = new Set(
  [...ALL_IDS].filter((id) => id !== 2001 && !(id >= 2200 && id <= 2218)),
);

/** A legacy tenant: the four launch-era explorer branches + the GM line only. */
const LAUNCH_ERA: ReadonlySet<number> = new Set([
  0, 100, 110, 111, 112, 120, 121, 122, 130, 131, 132, 200, 210, 211, 212, 220,
  221, 222, 230, 231, 232, 300, 310, 311, 312, 320, 321, 322, 400, 410, 411,
  412, 420, 421, 422, 900, 910,
]);

/** LAUNCH_ERA plus the Pirate branch. */
const WITH_PIRATE: ReadonlySet<number> = new Set([
  ...LAUNCH_ERA,
  500,
  510,
  511,
  512,
  520,
  521,
  522,
]);

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
  it("gates entries by the tenant job set and drops empty groups (legacy tenant)", () => {
    const groups = visibleRailGroups(LAUNCH_ERA);
    expect(groups.map((g) => g.label)).toEqual(["Explorers", "Special"]);
    expect(groups[0]!.entries.map((e) => e.id)).toEqual([100, 200, 300, 400]); // no Pirate
    expect(groups[1]!.entries.map((e) => e.id)).toEqual([900]); // no Brigadier
  });

  it("adds Pirate, Cygnus/Aran/Brigadier, and Evan as the tenant's job set grows", () => {
    expect(
      visibleRailGroups(WITH_PIRATE)[0]!.entries.map((e) => e.id),
    ).toContain(500);
    const noEvan = visibleRailGroups(NO_EVAN);
    expect(noEvan.map((g) => g.label)).toEqual([
      "Explorers",
      "Cygnus Knights",
      "Legends",
      "Special",
    ]);
    expect(noEvan[2]!.entries.map((e) => e.id)).toEqual([2000]); // Evan absent
    expect(noEvan[3]!.entries.map((e) => e.id)).toEqual([800, 900]);
    expect(visibleRailGroups(ALL_IDS)[2]!.entries.map((e) => e.id)).toEqual([
      2000, 2001,
    ]);
  });

  it("decorates entries with display name and visible subtree count", () => {
    const noEvan = visibleRailGroups(NO_EVAN);
    const warrior = noEvan[0]!.entries[0]!;
    expect(warrior.name).toBe("Warrior");
    expect(warrior.count).toBe(10);
    const gm = noEvan[3]!.entries.find((e) => e.id === 900);
    expect(gm?.name).toBe("GM");
    expect(gm?.count).toBe(2);
  });
});
