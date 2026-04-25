import { describe, it, expect } from "vitest";
import { jobTreePath } from "../job-tree";

describe("jobTreePath", () => {
  it("Hero (112) -> [Beginner, Warrior, Fighter, Crusader, Hero]", () => {
    expect(jobTreePath(112).map((j) => j.name)).toEqual([
      "Beginner",
      "Warrior",
      "Fighter",
      "Crusader",
      "Hero",
    ]);
  });

  it("Bishop (232) -> [Beginner, Magician, Cleric, Priest, Bishop]", () => {
    expect(jobTreePath(232).map((j) => j.name)).toEqual([
      "Beginner",
      "Magician",
      "Cleric",
      "Priest",
      "Bishop",
    ]);
  });

  it("Beginner (0) -> [Beginner]", () => {
    expect(jobTreePath(0).map((j) => j.name)).toEqual(["Beginner"]);
  });

  it("Aran 4th (2112) -> 5-step path starting at Legend", () => {
    const path = jobTreePath(2112).map((j) => j.name);
    expect(path[0]).toBe("Legend");
    expect(path).toHaveLength(5);
  });

  it("unknown jobId -> []", () => {
    expect(jobTreePath(99999)).toEqual([]);
  });
});
