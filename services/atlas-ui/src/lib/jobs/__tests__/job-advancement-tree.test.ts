import { describe, it, expect } from "vitest";
import {
  JOB_GRAPH,
  JOB_ROOTS,
  childrenOf,
  rootOf,
  visibleRoots,
  visibleChildrenOf,
  jobTreePath,
  advancementChains,
  tierLabel,
  subtreeCount,
} from "@/lib/jobs/job-advancement-tree";

/** Every id in the graph — the "modern tenant has everything" case. */
const ALL: ReadonlySet<number> = new Set(
  Object.values(JOB_GRAPH).map((e) => e.id),
);

/** A legacy tenant: explorers only, no Pirate, no GM line, no Cygnus/Legends. */
const LEGACY: ReadonlySet<number> = new Set([
  0, 100, 110, 111, 112, 120, 121, 122, 130, 131, 132, 200, 210, 211, 212, 220,
  221, 222, 230, 231, 232, 300, 310, 311, 312, 320, 321, 322, 400, 410, 411,
  412, 420, 421, 422,
]);

describe("job-advancement-tree", () => {
  it("exposes the five branch roots ascending (GM line is not a root)", () => {
    expect(JOB_ROOTS).toEqual([0, 800, 1000, 2000, 2001]);
  });

  it("derives children from parent edges, ascending", () => {
    expect(childrenOf(0)).toEqual([100, 200, 300, 400, 500, 900]);
    expect(childrenOf(100)).toEqual([110, 120, 130]);
    expect(childrenOf(900)).toEqual([910]);
    expect(childrenOf(112)).toEqual([]);
  });

  it("walks to the branch root", () => {
    expect(rootOf(112)).toBe(0);
    expect(rootOf(1112)).toBe(1000);
    expect(rootOf(2112)).toBe(2000);
    expect(rootOf(2218)).toBe(2001);
    expect(rootOf(910)).toBe(0);
    expect(rootOf(99999)).toBe(99999);
  });

  it("shows every root when the tenant has every job", () => {
    expect(visibleRoots(ALL)).toEqual([0, 800, 1000, 2000, 2001]);
  });

  it("hides roots the tenant has no job document for", () => {
    const roots = visibleRoots(LEGACY);
    expect(roots).toEqual([0]);
    expect(roots).not.toContain(1000);
    expect(roots).not.toContain(2000);
    expect(roots).not.toContain(2001);
    expect(roots).not.toContain(800);
  });

  it("hides an absent subtree while keeping its present siblings", () => {
    // LEGACY has no Pirate (500) and no GM (900).
    expect(visibleChildrenOf(0, LEGACY)).toEqual([100, 200, 300, 400]);
    expect(visibleChildrenOf(0, ALL)).toEqual([100, 200, 300, 400, 500, 900]);
  });

  it("returns nothing when the tenant set is empty", () => {
    const none: ReadonlySet<number> = new Set();
    expect(visibleRoots(none)).toEqual([]);
    expect(visibleChildrenOf(0, none)).toEqual([]);
    expect(subtreeCount(0, none)).toBe(0);
    expect(advancementChains(0, none)).toEqual([]);
  });

  it("drops an advancement chain containing any absent node", () => {
    // Warrior's chains are [110,111,112], [120,121,122], [130,131,132].
    const noHero: ReadonlySet<number> = new Set(
      [...LEGACY].filter((id) => id !== 112),
    );
    const chains = advancementChains(100, noHero);
    expect(chains).not.toContainEqual([110, 111, 112]);
    expect(chains).toContainEqual([120, 121, 122]);
    expect(chains).toContainEqual([130, 131, 132]);
  });

  it("counts only the jobs the tenant actually has", () => {
    // Warrior subtree: 100 + 110,111,112 + 120,121,122 + 130,131,132 = 10.
    expect(subtreeCount(100, LEGACY)).toBe(10);
    expect(subtreeCount(100, new Set([100, 110]))).toBe(2);
    expect(subtreeCount(100, new Set([110]))).toBe(0);
    expect(subtreeCount(99999, ALL)).toBe(0);
  });

  it("keeps topology helpers version-independent", () => {
    expect(jobTreePath(112).map((e) => e.id)).toEqual([0, 100, 110, 111, 112]);
    expect(tierLabel(0)).toBe("Base");
    expect(tierLabel(112)).toBe("4th");
    expect(tierLabel(99999)).toBe("");
  });
});
