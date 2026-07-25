import { describe, it, expect } from "vitest";
import { PRESET_JOBS, jobLabel } from "../presetJobs";

describe("presetJobs", () => {
  it("maps known job ids to names", () => {
    expect(jobLabel(0)).toBe("Beginner");
    expect(jobLabel(100)).toBe("Warrior");
    expect(jobLabel(900)).toBe("GM");
  });

  it("falls back to Job <id> for unknown ids", () => {
    expect(jobLabel(123456)).toBe("Job 123456");
  });

  it("exposes an ascending, de-duplicated curated list", () => {
    const ids = PRESET_JOBS.map((j) => j.id);
    expect(ids).toEqual([...ids].sort((a, b) => a - b));
    expect(new Set(ids).size).toBe(ids.length);
    expect(PRESET_JOBS.find((j) => j.id === 0)?.name).toBe("Beginner");
  });
});
