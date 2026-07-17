import { describe, it, expect } from "vitest";
import {
  JOB_GRAPH, BRANCH_FLOORS, JOB_ROOTS,
  childrenOf, rootOf, floorOf, visibleRoots, jobTreePath,
} from "@/lib/jobs/job-advancement-tree";

describe("job-advancement-tree", () => {
  it("exposes the seven branch roots ascending", () => {
    expect(JOB_ROOTS).toEqual([0, 800, 900, 910, 1000, 2000, 2001]);
  });

  it("derives children from parent edges, ascending", () => {
    expect(childrenOf(0)).toEqual([100, 200, 300, 400, 500]);
    expect(childrenOf(100)).toEqual([110, 120, 130]);
    expect(childrenOf(112)).toEqual([]); // 4th job is a leaf
  });

  it("walks to the branch root", () => {
    expect(rootOf(112)).toBe(0);    // Hero -> Beginner
    expect(rootOf(1112)).toBe(1000); // Dawn Warrior 4 -> Noblesse
    expect(rootOf(2112)).toBe(2000); // Aran 4 -> Legend
    expect(rootOf(2218)).toBe(2001); // Evan 10 -> Evan root
    expect(rootOf(99999)).toBe(99999); // unknown id returns itself
  });

  it("uses the corrected per-branch floors, inherited from the root", () => {
    expect(BRANCH_FLOORS).toEqual({ 0: 1, 800: 83, 900: 1, 910: 1, 1000: 83, 2000: 80, 2001: 84 });
    expect(floorOf(112)).toBe(1);   // Adventurer — present since launch
    expect(floorOf(1112)).toBe(83); // Cygnus corrected 92 -> 83
    expect(floorOf(2112)).toBe(80); // Aran corrected 88 -> 80
    expect(floorOf(2218)).toBe(84); // Evan
  });

  it("shows Cygnus + Aran on v83 and hides Evan until v84", () => {
    const r83 = visibleRoots(83);
    expect(r83).toContain(0);
    expect(r83).toContain(1000);     // Cygnus visible on v83
    expect(r83).toContain(2000);     // Aran visible on v83
    expect(r83).not.toContain(2001); // Evan hidden on v83
    expect(visibleRoots(84)).toContain(2001); // Evan visible on v84
  });

  it("shows base Adventurers + admin jobs on legacy sub-83 versions (GMS v12/v48)", () => {
    const r12 = visibleRoots(12);
    expect(r12).toContain(0);        // Adventurers — the jobs page was empty before this
    expect(r12).toContain(900);      // GM — always present
    expect(r12).toContain(910);      // Super GM — always present
    expect(r12).not.toContain(1000); // Cygnus (v83) hidden
    expect(r12).not.toContain(2000); // Aran (v80) hidden
    expect(r12).not.toContain(800);  // Maple Leaf Brigadier (special) hidden
  });

  it("jobTreePath returns root->node inclusive", () => {
    expect(jobTreePath(112).map((j) => j.name)).toEqual([
      "Beginner", "Warrior", "Fighter", "Crusader", "Hero",
    ]);
    expect(jobTreePath(0).map((j) => j.name)).toEqual(["Beginner"]);
    expect(jobTreePath(99999)).toEqual([]);
  });

  it("has no orphan parent references", () => {
    for (const e of Object.values(JOB_GRAPH)) {
      if (e.parent !== null) {
        expect(JOB_GRAPH[e.parent]).toBeDefined();
      }
    }
  });
});
