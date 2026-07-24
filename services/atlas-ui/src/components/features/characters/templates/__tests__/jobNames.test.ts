import { describe, it, expect } from "vitest";
import {
  worldNameFromJobIndex,
  genderLabel,
  templateLabels,
  KNOWN_CLASSES,
} from "../jobNames";

describe("worldNameFromJobIndex", () => {
  it("maps the four known job indexes (mirrors JobFromIndex)", () => {
    expect(worldNameFromJobIndex(0)).toBe("Cygnus Knights");
    expect(worldNameFromJobIndex(1)).toBe("Adventurer");
    expect(worldNameFromJobIndex(2)).toBe("Aran");
    expect(worldNameFromJobIndex(3)).toBe("Evan");
  });

  it("falls back to Job N for unknown indexes", () => {
    expect(worldNameFromJobIndex(7)).toBe("Job 7");
  });
});

describe("genderLabel", () => {
  it("maps 0 to M and 1 to F", () => {
    expect(genderLabel(0)).toBe("M");
    expect(genderLabel(1)).toBe("F");
  });
});

describe("templateLabels", () => {
  it("labels as <World> · <M|F>", () => {
    expect(templateLabels([{ jobIndex: 1, gender: 0 }])).toEqual([
      "Adventurer · M",
    ]);
  });

  it("suffixes ordinals only on duplicate labels, starting at (2)", () => {
    expect(
      templateLabels([
        { jobIndex: 1, gender: 0 },
        { jobIndex: 1, gender: 1 },
        { jobIndex: 1, gender: 0 },
        { jobIndex: 1, gender: 0 },
      ]),
    ).toEqual([
      "Adventurer · M",
      "Adventurer · F",
      "Adventurer · M (2)",
      "Adventurer · M (3)",
    ]);
  });
});

describe("KNOWN_CLASSES", () => {
  it("lists the four factory-mapped classes with jobIndex.subJobIndex labels", () => {
    expect(KNOWN_CLASSES).toEqual([
      { jobIndex: 0, subJobIndex: 0, label: "Cygnus Knights (0.0)" },
      { jobIndex: 1, subJobIndex: 0, label: "Adventurer (1.0)" },
      { jobIndex: 2, subJobIndex: 0, label: "Aran (2.0)" },
      { jobIndex: 3, subJobIndex: 0, label: "Evan (3.0)" },
    ]);
  });
});
