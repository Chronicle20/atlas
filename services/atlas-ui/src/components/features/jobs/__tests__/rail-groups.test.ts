import { describe, it, expect } from "vitest";
import {
  FIXTURE_JOB_TREE,
  FIXTURE_JOBS_SORTED,
} from "@/lib/jobs/__tests__/job-graph-fixtures";
import { buildJobGraph, type JobGraph } from "@/lib/jobs/job-graph";
import type { JobAvailabilityEntry } from "@/services/api/availability.service";
import {
  RAIL_GROUPS,
  branchEntryOf,
  visibleRailGroups,
} from "@/components/features/jobs/rail-groups";

// The structural fixture source predates identity/wire-id divergence, so its
// wire id doubles as the canonical identity here — every modern-tenant
// fixture below is identity === id, which is the common case. The
// v0.48/v0.72/v0.79 fixtures further down are the ones that actually
// exercise a wire id != identity.
const FULL_AVAILABILITY: JobAvailabilityEntry[] = FIXTURE_JOBS_SORTED.map(
  (e) => ({
    id: e.id,
    name: e.name,
    parent: e.parent,
    identity: e.id,
  }),
);

function graphOf(present: ReadonlySet<number>): JobGraph {
  return buildJobGraph(FULL_AVAILABILITY, present);
}

/** Every id in the graph — the "modern tenant has everything" case. */
const ALL_IDS: ReadonlySet<number> = new Set(
  Object.values(FIXTURE_JOB_TREE).map((e) => e.id),
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
    expect(RAIL_GROUPS[0]!.entries.map((e) => e.identity)).toEqual([
      100, 200, 300, 400, 500,
    ]);
    expect(RAIL_GROUPS[1]!.entries.map((e) => e.identity)).toEqual([1000]);
    expect(RAIL_GROUPS[2]!.entries.map((e) => e.identity)).toEqual([
      2000, 2001,
    ]);
    expect(RAIL_GROUPS[3]!.entries.map((e) => e.identity)).toEqual([800, 900]);
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
  it("resolves any node to the rail entry (by identity) on its path", () => {
    expect(branchEntryOf(graphOf(ALL_IDS), 112).identity).toBe(100); // Hero -> Warrior
    expect(branchEntryOf(graphOf(ALL_IDS), 910).identity).toBe(900); // Super GM -> GM entry
    expect(branchEntryOf(graphOf(ALL_IDS), 1512).identity).toBe(1000); // Thunder Breaker 4 -> Noblesse
    expect(branchEntryOf(graphOf(ALL_IDS), 2218).identity).toBe(2001); // Evan 10 -> Evan
    expect(branchEntryOf(graphOf(ALL_IDS), 800).identity).toBe(800);
  });

  it("falls back to the Warrior entry for Beginner and unknown ids", () => {
    expect(branchEntryOf(graphOf(ALL_IDS), 0).identity).toBe(100);
    expect(branchEntryOf(graphOf(ALL_IDS), 99999).identity).toBe(100);
  });
});

describe("visibleRailGroups", () => {
  it("gates entries by the tenant job set and drops empty groups (legacy tenant)", () => {
    const groups = visibleRailGroups(graphOf(LAUNCH_ERA));
    expect(groups.map((g) => g.label)).toEqual(["Explorers", "Special"]);
    expect(groups[0]!.entries.map((e) => e.id)).toEqual([100, 200, 300, 400]); // no Pirate
    expect(groups[1]!.entries.map((e) => e.id)).toEqual([900]); // no Brigadier
  });

  it("adds Pirate, Cygnus/Aran/Brigadier, and Evan as the tenant's job set grows", () => {
    expect(
      visibleRailGroups(graphOf(WITH_PIRATE))[0]!.entries.map((e) => e.id),
    ).toContain(500);
    const noEvan = visibleRailGroups(graphOf(NO_EVAN));
    expect(noEvan.map((g) => g.label)).toEqual([
      "Explorers",
      "Cygnus Knights",
      "Legends",
      "Special",
    ]);
    expect(noEvan[2]!.entries.map((e) => e.id)).toEqual([2000]); // Evan absent
    expect(noEvan[3]!.entries.map((e) => e.id)).toEqual([800, 900]);
    expect(
      visibleRailGroups(graphOf(ALL_IDS))[2]!.entries.map((e) => e.id),
    ).toEqual([2000, 2001]);
  });

  it("decorates entries with display name and visible subtree count", () => {
    const noEvan = visibleRailGroups(graphOf(NO_EVAN));
    const warrior = noEvan[0]!.entries[0]!;
    expect(warrior.name).toBe("Warrior");
    expect(warrior.count).toBe(10);
    const gm = noEvan[3]!.entries.find((e) => e.id === 900);
    expect(gm?.name).toBe("GM");
    expect(gm?.count).toBe(2);
  });
});

describe("visibleRailGroups — identity keying across wire-id divergence", () => {
  // A v0.48 fixture: wire 500 is Gm (identity 900), 510 is Super Gm (910), and
  // there is no Pirate branch. This is the case a wire-keyed rail gets wrong —
  // it would file "Gm" under Explorers in pirate colours.
  const v48 = buildJobGraph(
    [
      { id: 0, name: "Beginner", parent: null, identity: 0 },
      { id: 100, name: "Warrior", parent: 0, identity: 100 },
      { id: 500, name: "Gm", parent: 0, identity: 900 },
      { id: 510, name: "Super Gm", parent: 500, identity: 910 },
    ],
    new Set([0, 100, 500, 510]),
  );

  it("v0.48: the Explorers rail has no Pirate entry", () => {
    const explorers = visibleRailGroups(v48).find(
      (g) => g.label === "Explorers",
    );
    expect(explorers?.entries.map((e) => e.identity)).toEqual([100]);
  });

  it("v0.48: the Special group shows Gm, with Super Gm counted beneath it", () => {
    const special = visibleRailGroups(v48).find((g) => g.label === "Special");
    expect(special?.entries.map((e) => e.name)).toEqual(["Gm"]);
    expect(special?.entries[0]?.id).toBe(500);
    expect(special?.entries[0]?.count).toBe(2);
  });

  it("v0.72: the Cygnus Knights group is absent entirely", () => {
    const v72 = buildJobGraph(
      [
        { id: 0, name: "Beginner", parent: null, identity: 0 },
        { id: 100, name: "Warrior", parent: 0, identity: 100 },
        { id: 500, name: "Pirate", parent: 0, identity: 500 },
        { id: 900, name: "Gm", parent: 0, identity: 900 },
      ],
      new Set([0, 100, 500, 900]),
    );
    expect(visibleRailGroups(v72).map((g) => g.label)).not.toContain(
      "Cygnus Knights",
    );
  });

  it("v0.79: the Legends group is absent entirely", () => {
    const v79 = buildJobGraph(
      [
        { id: 0, name: "Beginner", parent: null, identity: 0 },
        { id: 1000, name: "Noblesse", parent: null, identity: 1000 },
        { id: 1100, name: "Dawn Warrior 1", parent: 1000, identity: 1100 },
      ],
      new Set([0, 1000, 1100]),
    );
    expect(visibleRailGroups(v79).map((g) => g.label)).not.toContain("Legends");
  });
});
