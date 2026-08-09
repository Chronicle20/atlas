import { describe, it, expect } from "vitest";
import {
  buildJobGraph,
  advancementChains,
  subtreeCount,
  tierLabel,
} from "@/lib/jobs/job-graph";
import type { JobAvailabilityEntry } from "@/services/api/availability.service";
import {
  branchEntryOf,
  visibleRailGroups,
} from "@/components/features/jobs/rail-groups";

/**
 * The Thief branch of a gms 92/95/jms 185 tenant, including the Dual Blade
 * line (task-204). Wire id === identity throughout, as it is for every job
 * on those columns.
 *
 * Dual Blade is the first Explorer branch with FIVE advancements rather than
 * three, and the only one where a third child hangs off the branch root — so
 * it is worth pinning that the rail curation and the flow helpers, none of
 * which know anything about it, carry it correctly. Nothing in the UI needed
 * a change for this; these tests exist so that stays true.
 */
const THIEF_BRANCH: JobAvailabilityEntry[] = [
  { id: 0, name: "Beginner", parent: null, identity: 0 },
  { id: 400, name: "Rogue", parent: 0, identity: 400 },
  { id: 410, name: "Assassin", parent: 400, identity: 410 },
  { id: 411, name: "Hermit", parent: 410, identity: 411 },
  { id: 412, name: "Night Lord", parent: 411, identity: 412 },
  { id: 420, name: "Bandit", parent: 400, identity: 420 },
  { id: 421, name: "Chief Bandit", parent: 420, identity: 421 },
  { id: 422, name: "Shadower", parent: 421, identity: 422 },
  { id: 430, name: "Blade Recruit", parent: 400, identity: 430 },
  { id: 431, name: "Blade Acolyte", parent: 430, identity: 431 },
  { id: 432, name: "Blade Specialist", parent: 431, identity: 432 },
  { id: 433, name: "Blade Lord", parent: 432, identity: 433 },
  { id: 434, name: "Blade Master", parent: 433, identity: 434 },
];

const ALL_IDS = new Set(THIEF_BRANCH.map((e) => e.id));
const graph = buildJobGraph(THIEF_BRANCH, ALL_IDS);

/** The same tenant before v0.88 — Dual Blade absent from availability. */
const preDualBlade = buildJobGraph(
  THIEF_BRANCH.filter((e) => e.id < 430 || e.id > 434),
  ALL_IDS,
);

describe("Dual Blade in the job rail", () => {
  it("files every Dual Blade job under the Explorers rail's Rogue entry", () => {
    for (const id of [430, 431, 432, 433, 434]) {
      expect(branchEntryOf(graph, id).identity).toBe(400);
    }
  });

  it("counts the Dual Blade line into the Rogue rail entry's subtree", () => {
    const explorers = visibleRailGroups(graph).find(
      (g) => g.label === "Explorers",
    );
    const rogue = explorers?.entries.find((e) => e.identity === 400);
    // Rogue + Assassin/Hermit/NightLord + Bandit/ChiefBandit/Shadower + the
    // five Dual Blade jobs.
    expect(rogue?.count).toBe(12);
    expect(subtreeCount(preDualBlade, 400)).toBe(7);
  });

  it("renders Dual Blade as a third chain off Rogue, five deep", () => {
    const chains = advancementChains(graph, 400);
    expect(chains).toContainEqual([430, 431, 432, 433, 434]);
    expect(chains).toHaveLength(3);
    expect(advancementChains(preDualBlade, 400)).toHaveLength(2);
  });

  it("labels the chain by graph depth, continuing past the Explorer tiers", () => {
    // Depth-from-root labelling, the same rule Assassin (2nd) / Hermit (3rd)
    // / Night Lord (4th) already get — NOT the in-game advancement number.
    expect(tierLabel(graph, 430)).toBe("2nd");
    expect(tierLabel(graph, 434)).toBe("6th");
  });
});
