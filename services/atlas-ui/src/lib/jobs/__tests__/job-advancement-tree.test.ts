import { describe, it, expect } from "vitest";
import {
  JOB_GRAPH,
  JOB_LIST,
  jobName,
  jobTreePath,
} from "@/lib/jobs/job-advancement-tree";

describe("job-advancement-tree", () => {
  it("keeps topology helpers version-independent", () => {
    expect(jobTreePath(112).map((e) => e.id)).toEqual([0, 100, 110, 111, 112]);
  });

  // Restored (task-10 fix round 1): Task 9's set-driven rewrite dropped this
  // structural invariant along with the floor-based assertions that used to
  // share its `it()` block. It is version-independent — no `available` set
  // involved — so it ports unchanged from the pre-branch file.
  it("has no orphan parent references", () => {
    for (const e of Object.values(JOB_GRAPH)) {
      if (e.parent !== null) {
        expect(JOB_GRAPH[e.parent]).toBeDefined();
      }
    }
  });
});

describe("jobName", () => {
  it("names explorer classes", () => {
    expect(jobName(0)).toBe("Beginner");
    expect(jobName(100)).toBe("Warrior");
    expect(jobName(900)).toBe("GM");
  });

  it("names Aran and Evan (the ids the old curated list omitted)", () => {
    expect(jobName(2000)).toBe("Legend");
    expect(jobName(2100)).toBe("Aran 1");
    expect(jobName(2110)).toBe("Aran 2");
    expect(jobName(2001)).toBe("Evan");
    expect(jobName(2200)).toBe("Evan 1");
  });

  it("falls back to Job <id> for an id the graph does not cover", () => {
    expect(jobName(4321)).toBe("Job 4321");
    expect(jobName(123456)).toBe("Job 123456");
  });
});

describe("JOB_LIST", () => {
  it("is every graph entry, ascending by id, including Aran/Evan/Cygnus", () => {
    expect(JOB_LIST).toHaveLength(Object.keys(JOB_GRAPH).length);
    const ids = JOB_LIST.map((j) => j.id);
    expect([...ids]).toEqual([...ids].sort((a, b) => a - b));
    expect(ids).toContain(2100); // Aran 1
    expect(ids).toContain(2001); // Evan
    expect(ids).toContain(1000); // Noblesse (Cygnus)
  });
});
