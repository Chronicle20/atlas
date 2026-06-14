import { describe, it, expect } from "vitest";
import { JOB_HIERARCHY, filterHierarchy, jobNodeName } from "@/lib/jobs-hierarchy";

describe("jobs-hierarchy", () => {
  it("resolves leaf names via getJobNameById", () => {
    expect(jobNodeName({ jobId: 112, minMajorVersion: 83 })).toBe("Hero");
  });

  it("v83 keeps Adventurer but drops Cygnus, Legend, and Evan", () => {
    const tree = filterHierarchy(JOB_HIERARCHY, 83);
    const names = tree.map((a) => a.name);
    expect(names).toContain("Adventurer");
    expect(names).not.toContain("Cygnus");
    expect(names).not.toContain("Legend");
  });

  it("removes a class with no surviving jobs and an archetype with no surviving classes", () => {
    const tree = filterHierarchy(JOB_HIERARCHY, 83);
    expect(tree.find((a) => a.name === "Cygnus")).toBeUndefined();
    for (const arch of tree) {
      for (const cls of arch.classes) {
        expect(cls.jobs.length).toBeGreaterThan(0);
      }
    }
  });

  it("a high version keeps Cygnus and Legend archetypes", () => {
    const tree = filterHierarchy(JOB_HIERARCHY, 95);
    const names = tree.map((a) => a.name);
    expect(names).toContain("Cygnus");
    expect(names).toContain("Legend");
  });

  it("does not mutate the source tree", () => {
    const before = JOB_HIERARCHY.length;
    filterHierarchy(JOB_HIERARCHY, 83);
    expect(JOB_HIERARCHY.length).toBe(before);
  });
});
