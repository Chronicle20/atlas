import { describe, it, expect } from "vitest";
import { formatSkillDescription } from "@/lib/skills/format-skill-description";

describe("formatSkillDescription", () => {
  it("returns an empty model for blank input", () => {
    expect(formatSkillDescription(undefined)).toEqual({ lines: [] });
    expect(formatSkillDescription("")).toEqual({ lines: [] });
    expect(formatSkillDescription("   ")).toEqual({ lines: [] });
  });

  it("splits on newlines into lines of segments", () => {
    const r = formatSkillDescription("Line one\nLine two");
    expect(r.lines).toHaveLength(2);
    expect(r.lines[0]).toEqual([{ text: "Line one" }]);
    expect(r.lines[1]).toEqual([{ text: "Line two" }]);
  });

  it("captures and suppresses a leading [Master Level : N] header", () => {
    const r = formatSkillDescription(
      "[Master Level : 16]\nRecover additional HP",
    );
    expect(r.masterLevelHeader).toBe(16);
    expect(r.lines).toEqual([[{ text: "Recover additional HP" }]]);
  });

  it("strips #c...# color markers and marks the segment color", () => {
    const r = formatSkillDescription("#cAt least Level 3 on Sacrifice#");
    expect(r.lines).toEqual([
      [{ text: "At least Level 3 on Sacrifice", color: "highlight" }],
    ]);
  });

  it("strips unknown #x...# directives keeping the inner text", () => {
    const r = formatSkillDescription("see #ebold text# here");
    expect(r.lines).toEqual([
      [{ text: "see " }, { text: "bold text" }, { text: " here" }],
    ]);
  });

  it("consumes bare # reset markers, leaking no '#'", () => {
    const r = formatSkillDescription("#cred#then plain");
    const flat = r.lines
      .flat()
      .map((s) => s.text)
      .join("");
    expect(flat).not.toContain("#");
    expect(flat).toBe("red" + "then plain");
  });

  it("keeps the master-level header even if the body is empty", () => {
    expect(formatSkillDescription("[Master Level : 30]")).toEqual({
      lines: [],
      masterLevelHeader: 30,
    });
  });
});
