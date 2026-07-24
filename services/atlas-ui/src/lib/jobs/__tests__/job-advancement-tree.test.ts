import { describe, it, expect } from "vitest";
import {
  JOB_GRAPH,
  BRANCH_FLOORS,
  JOB_ROOTS,
  childrenOf,
  rootOf,
  floorOf,
  visibleRoots,
  visibleChildrenOf,
  jobTreePath,
  advancementChains,
  tierLabel,
  subtreeCount,
} from "@/lib/jobs/job-advancement-tree";

describe("job-advancement-tree", () => {
  it("exposes the five branch roots ascending (GM line is no longer a root)", () => {
    expect(JOB_ROOTS).toEqual([0, 800, 1000, 2000, 2001]);
  });

  it("derives children from parent edges, ascending", () => {
    expect(childrenOf(0)).toEqual([100, 200, 300, 400, 500, 900]);
    expect(childrenOf(100)).toEqual([110, 120, 130]);
    expect(childrenOf(900)).toEqual([910]); // GM advances to Super GM
    expect(childrenOf(112)).toEqual([]); // 4th job is a leaf
  });

  it("walks to the branch root", () => {
    expect(rootOf(112)).toBe(0); // Hero -> Beginner
    expect(rootOf(1112)).toBe(1000); // Dawn Warrior 4 -> Noblesse
    expect(rootOf(2112)).toBe(2000); // Aran 4 -> Legend
    expect(rootOf(2218)).toBe(2001); // Evan 10 -> Evan root
    expect(rootOf(910)).toBe(0); // Super GM -> GM -> Beginner
    expect(rootOf(99999)).toBe(99999); // unknown id returns itself
  });

  it("uses the corrected per-branch floors, inherited from the root", () => {
    expect(BRANCH_FLOORS).toEqual({
      0: 1,
      800: 83,
      1000: 83,
      2000: 80,
      2001: 84,
    });
    expect(floorOf(112)).toBe(1); // Adventurer — present since launch
    expect(floorOf(900)).toBe(1); // GM — inherits Beginner's floor as a non-root
    expect(floorOf(910)).toBe(1); // Super GM — likewise
    expect(floorOf(1112)).toBe(83); // Cygnus corrected 92 -> 83
    expect(floorOf(2112)).toBe(80); // Aran corrected 88 -> 80
    expect(floorOf(2218)).toBe(84); // Evan
  });

  it("shows Cygnus + Aran on v83 and hides Evan until v84", () => {
    const r83 = visibleRoots(83);
    expect(r83).toContain(0);
    expect(r83).toContain(1000); // Cygnus visible on v83
    expect(r83).toContain(2000); // Aran visible on v83
    expect(r83).not.toContain(2001); // Evan hidden on v83
    expect(visibleRoots(84)).toContain(2001); // Evan visible on v84
  });

  it("shows base Adventurers + the GM line on legacy sub-83 versions (GMS v12/v48)", () => {
    const r12 = visibleRoots(12);
    expect(r12).toContain(0); // Adventurers — the jobs page was empty before this
    expect(r12).not.toContain(1000); // Cygnus (v83) hidden
    expect(r12).not.toContain(2000); // Aran (v80) hidden
    expect(r12).not.toContain(800); // Maple Leaf Brigadier (special) hidden
    // GM/Super GM are children of Beginner now — visible via the tree, not as roots.
    expect(visibleChildrenOf(0, 12)).toContain(900);
    expect(visibleChildrenOf(900, 12)).toContain(910);
  });

  it("gates the Pirate subtree at v62 while its launch-era siblings stay visible", () => {
    // Pirate (500) was added in GMS v62; the other four explorer classes existed at launch.
    expect(floorOf(500)).toBe(62);
    expect(floorOf(100)).toBe(1); // Warrior — launch-era
    // On a sub-62 tenant Pirate is hidden from the Beginner tree; the others show.
    expect(visibleChildrenOf(0, 12)).toEqual(
      expect.arrayContaining([100, 200, 300, 400]),
    );
    expect(visibleChildrenOf(0, 12)).not.toContain(500);
    // On v83 Pirate is visible again.
    expect(visibleChildrenOf(0, 83)).toContain(500);
  });

  it("jobTreePath returns root->node inclusive", () => {
    expect(jobTreePath(112).map((j) => j.name)).toEqual([
      "Beginner",
      "Warrior",
      "Fighter",
      "Crusader",
      "Hero",
    ]);
    expect(jobTreePath(0).map((j) => j.name)).toEqual(["Beginner"]);
    expect(jobTreePath(99999)).toEqual([]);
    expect(jobTreePath(910).map((j) => j.id)).toEqual([0, 900, 910]);
    expect(jobTreePath(910).map((j) => j.name)).toEqual([
      "Beginner",
      "GM",
      "Super GM",
    ]);
  });

  it("has no orphan parent references", () => {
    for (const e of Object.values(JOB_GRAPH)) {
      if (e.parent !== null) {
        expect(JOB_GRAPH[e.parent]).toBeDefined();
      }
    }
  });
});

describe("advancementChains", () => {
  it("returns one chain per advancement path below the entry, ascending, entry excluded", () => {
    expect(advancementChains(100, 83)).toEqual([
      [110, 111, 112],
      [120, 121, 122],
      [130, 131, 132],
    ]);
  });

  it("handles the Evan 10-tier single chain", () => {
    expect(advancementChains(2001, 84)).toEqual([
      [2200, 2210, 2211, 2212, 2213, 2214, 2215, 2216, 2217, 2218],
    ]);
  });

  it("renders the GM line as a single chain below the GM entry", () => {
    expect(advancementChains(900, 1)).toEqual([[910]]);
  });

  it("includes the GM line among Beginner's chains", () => {
    const chains = advancementChains(0, 83);
    expect(chains).toContainEqual([900, 910]);
    expect(chains).toContainEqual([100, 110, 111, 112]);
    // Warrior/Magician have 3 paths each; Bowman/Rogue/Pirate 2 each; + GM line
    expect(chains).toHaveLength(13);
  });

  it("drops any chain containing a below-floor node (Pirate on v12)", () => {
    const chains = advancementChains(0, 12);
    expect(chains).toHaveLength(11); // 13 minus Pirate's 2 paths
    for (const chain of chains) {
      expect(chain).not.toContain(500);
    }
  });

  it("returns [] for a leaf entry", () => {
    expect(advancementChains(112, 83)).toEqual([]);
    expect(advancementChains(800, 83)).toEqual([]);
  });
});

describe("tierLabel", () => {
  it("labels tree roots with children as Base, childless roots empty", () => {
    expect(tierLabel(0)).toBe("Base");
    expect(tierLabel(1000)).toBe("Base");
    expect(tierLabel(800)).toBe("");
  });

  it("labels advancement depth ordinally, including the GM line and Evan depth 10", () => {
    expect(tierLabel(100)).toBe("1st");
    expect(tierLabel(110)).toBe("2nd");
    expect(tierLabel(111)).toBe("3rd");
    expect(tierLabel(112)).toBe("4th");
    expect(tierLabel(900)).toBe("1st");
    expect(tierLabel(910)).toBe("2nd");
    expect(tierLabel(2218)).toBe("10th");
  });

  it("returns empty for unknown ids", () => {
    expect(tierLabel(99999)).toBe("");
  });
});

describe("subtreeCount", () => {
  it("counts version-visible nodes including the entry", () => {
    expect(subtreeCount(100, 83)).toBe(10); // Warrior line
    expect(subtreeCount(900, 1)).toBe(2); // GM + Super GM
    expect(subtreeCount(1000, 83)).toBe(21); // Noblesse + 5 paths x 4
    expect(subtreeCount(800, 83)).toBe(1);
    expect(subtreeCount(2001, 84)).toBe(11); // Evan root + 10 tiers
  });

  it("excludes below-floor subtrees and below-floor entries", () => {
    // Beginner + Warrior 10 + Magician 10 + Bowman 7 + Rogue 7 + GM line 2
    expect(subtreeCount(0, 12)).toBe(37);
    expect(subtreeCount(0, 62)).toBe(44); // Pirate's 7 nodes join at v62
    expect(subtreeCount(500, 12)).toBe(0); // Pirate entry itself below floor
  });
});
