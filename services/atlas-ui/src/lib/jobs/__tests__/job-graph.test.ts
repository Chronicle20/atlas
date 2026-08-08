import { describe, expect, it } from "vitest";
import {
  advancementChains,
  buildJobGraph,
  childrenOf,
  jobNodeName,
  jobTreePath,
  rootOf,
  subtreeCount,
  tierLabel,
  type JobGraph,
  type JobNode,
} from "@/lib/jobs/job-graph";
import type { JobAvailabilityEntry } from "@/services/api/availability.service";

// A v0.48-shaped fixture: wire id 500 is Gm (canonical identity 900) with
// Super Gm beneath it, and there is no Pirate branch at all.
const V48_AVAILABILITY: JobAvailabilityEntry[] = [
  { id: 0, name: "Beginner", parent: null, identity: 0 },
  { id: 100, name: "Warrior", parent: 0, identity: 100 },
  { id: 110, name: "Fighter", parent: 100, identity: 110 },
  { id: 111, name: "Crusader", parent: 110, identity: 111 },
  { id: 500, name: "Gm", parent: 0, identity: 900 },
  { id: 510, name: "Super Gm", parent: 500, identity: 910 },
];
const V48_PRESENT = new Set([0, 100, 110, 111, 500, 510]);

function v48(): JobGraph {
  return buildJobGraph(V48_AVAILABILITY, V48_PRESENT);
}

describe("buildJobGraph", () => {
  it("keeps only ids present in BOTH availability and the WZ job set", () => {
    const graph = buildJobGraph(V48_AVAILABILITY, new Set([0, 100, 110]));
    expect([...graph.keys()].sort((a, b) => a - b)).toEqual([0, 100, 110]);
  });

  it("drops an id the WZ has but availability does not", () => {
    const graph = buildJobGraph(V48_AVAILABILITY, new Set([0, 100, 1000]));
    expect(graph.has(1000)).toBe(false);
  });

  it("re-roots a surviving child whose parent the intersection dropped", () => {
    // 110 survives, its parent 100 does not.
    const graph = buildJobGraph(V48_AVAILABILITY, new Set([0, 110, 111]));
    expect(graph.get(110)?.parent).toBeNull();
    // 111's parent 110 DID survive, so its edge is untouched.
    expect(graph.get(111)?.parent).toBe(110);
  });

  it("carries the canonical identity, which can differ from the wire id", () => {
    expect(v48().get(500)?.identity).toBe(900);
    expect(v48().get(500)?.name).toBe("Gm");
  });
});

describe("graph helpers", () => {
  it("childrenOf returns direct children ascending", () => {
    expect(childrenOf(v48(), 0)).toEqual([100, 500]);
    expect(childrenOf(v48(), 111)).toEqual([]);
  });

  it("rootOf walks to the branch root and returns the id itself if unknown", () => {
    expect(rootOf(v48(), 510)).toBe(0);
    expect(rootOf(v48(), 0)).toBe(0);
    expect(rootOf(v48(), 9999)).toBe(9999);
  });

  it("jobTreePath is root -> node inclusive", () => {
    expect(jobTreePath(v48(), 111).map((n) => n.id)).toEqual([
      0, 100, 110, 111,
    ]);
    expect(jobTreePath(v48(), 9999)).toEqual([]);
  });

  it("tierLabel is Base for a root with children, else the ordinal depth", () => {
    expect(tierLabel(v48(), 0)).toBe("Base");
    expect(tierLabel(v48(), 100)).toBe("1st");
    expect(tierLabel(v48(), 111)).toBe("3rd");
    expect(tierLabel(v48(), 9999)).toBe("");
  });

  it("advancementChains lists every root-to-leaf path below the entry, entry excluded", () => {
    expect(advancementChains(v48(), 100)).toEqual([[110, 111]]);
    expect(advancementChains(v48(), 111)).toEqual([]);
  });

  it("subtreeCount counts the entry and everything below it", () => {
    expect(subtreeCount(v48(), 100)).toBe(3);
    expect(subtreeCount(v48(), 111)).toBe(1);
    expect(subtreeCount(v48(), 9999)).toBe(0);
  });

  it("jobNodeName falls back to `Job <id>` for an id the graph does not carry", () => {
    expect(jobNodeName(v48(), 500)).toBe("Gm");
    expect(jobNodeName(v48(), 9999)).toBe("Job 9999");
  });
});

// buildJobGraph cannot produce a cycle from acyclic input (re-rooting only
// ever nulls a dangling parent), so these malformed graphs are constructed
// directly as Map<number, JobNode> — the shape an API response would have
// to be in to trigger the bug this guards against.
function selfParentGraph(): JobGraph {
  const g = new Map<number, JobNode>();
  g.set(5, { id: 5, identity: 5, name: "Self", parent: 5 });
  return g;
}

function mutualCycleGraph(): JobGraph {
  const g = new Map<number, JobNode>();
  g.set(1, { id: 1, identity: 1, name: "A", parent: 2 });
  g.set(2, { id: 2, identity: 2, name: "B", parent: 1 });
  return g;
}

describe("cycle guards on malformed (API-sourced) graphs", () => {
  it("rootOf terminates on a self-parenting node", () => {
    expect(rootOf(selfParentGraph(), 5)).toBe(5);
  });

  it("rootOf terminates on a two-node mutual cycle, returning the last node before the repeat", () => {
    expect(rootOf(mutualCycleGraph(), 1)).toBe(2);
  });

  it("jobTreePath terminates on a self-parenting node", () => {
    expect(jobTreePath(selfParentGraph(), 5).map((n) => n.id)).toEqual([5]);
  });

  it("jobTreePath terminates on a two-node mutual cycle, truncated at the repeat", () => {
    expect(jobTreePath(mutualCycleGraph(), 1).map((n) => n.id)).toEqual([2, 1]);
  });

  it("tierLabel terminates on a cyclic graph (via jobTreePath)", () => {
    expect(tierLabel(mutualCycleGraph(), 1)).not.toBeUndefined();
  });

  it("advancementChains terminates on a self-parenting node", () => {
    expect(advancementChains(selfParentGraph(), 5)).toEqual([[5]]);
  });

  it("advancementChains terminates on a two-node mutual cycle", () => {
    expect(advancementChains(mutualCycleGraph(), 1)).toEqual([[2, 1]]);
  });

  it("subtreeCount terminates on a self-parenting node", () => {
    expect(subtreeCount(selfParentGraph(), 5)).toBe(1);
  });

  it("subtreeCount terminates on a two-node mutual cycle, counting each node once", () => {
    expect(subtreeCount(mutualCycleGraph(), 1)).toBe(2);
  });
});
