import { describe, it, expect } from "vitest";
import {
  worldNameFromJobIndex,
  genderLabel,
  templateLabels,
} from "../jobNames";

describe("worldNameFromJobIndex", () => {
  it("pre-Big-Bang slot 0: GMS 83 jobIndex 0 -> Cygnus Knight", () => {
    expect(worldNameFromJobIndex(0, "GMS", 83)).toBe("Cygnus Knight");
  });

  it("pre-Big-Bang slot 1: GMS 83 jobIndex 1 -> Explorer", () => {
    expect(worldNameFromJobIndex(1, "GMS", 83)).toBe("Explorer");
  });

  it("pre-Big-Bang slot 2: GMS 83 jobIndex 2 -> Aran", () => {
    expect(worldNameFromJobIndex(2, "GMS", 83)).toBe("Aran");
  });

  it("v95 slot 1: GMS 95 jobIndex 1 -> Explorer", () => {
    expect(worldNameFromJobIndex(1, "GMS", 95)).toBe("Explorer");
  });

  it("v95 slot 2: GMS 95 jobIndex 2 -> Cygnus Knight", () => {
    expect(worldNameFromJobIndex(2, "GMS", 95)).toBe("Cygnus Knight");
  });

  it("falls back to Job N for an unknown ordinal (FR-23)", () => {
    expect(worldNameFromJobIndex(42, "GMS", 95)).toBe("Job 42");
  });

  it("falls back to Job N for an unknown ordinal on a pre-Big-Bang version (FR-23)", () => {
    expect(worldNameFromJobIndex(42, "GMS", 83)).toBe("Job 42");
  });

  it("falls back to Job N when no tenant is selected", () => {
    expect(worldNameFromJobIndex(1, undefined, undefined)).toBe("Job 1");
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
    expect(templateLabels([{ jobIndex: 1, gender: 0 }], "GMS", 95)).toEqual([
      "Explorer · M",
    ]);
  });

  it("suffixes ordinals only on duplicate labels, starting at (2)", () => {
    expect(
      templateLabels(
        [
          { jobIndex: 1, gender: 0 },
          { jobIndex: 1, gender: 1 },
          { jobIndex: 1, gender: 0 },
          { jobIndex: 1, gender: 0 },
        ],
        "GMS",
        95,
      ),
    ).toEqual([
      "Explorer · M",
      "Explorer · F",
      "Explorer · M (2)",
      "Explorer · M (3)",
    ]);
  });
});
