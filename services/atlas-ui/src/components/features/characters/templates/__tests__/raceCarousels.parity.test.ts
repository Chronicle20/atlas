import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, it, expect } from "vitest";
import { classesForVersion } from "../jobNames";

const FIXTURE = path.resolve(
  __dirname,
  "..",
  "..",
  "..",
  "..",
  "..",
  "..",
  "..",
  "..",
  "docs",
  "packets",
  "race-carousels.json",
);

interface FixtureSlot {
  raceIndex: number;
  subJobIndex: number;
  jobId: number | null;
  class: string;
  verified: boolean;
  note?: string;
}

interface FixtureVersion {
  region: string;
  majorVersion: number;
  verified: boolean;
  slots: FixtureSlot[];
}

interface Fixture {
  versions: Record<string, FixtureVersion>;
}

const fixture: Fixture = JSON.parse(readFileSync(FIXTURE, "utf-8"));

describe("classesForVersion / race-carousels.json parity", () => {
  for (const [key, version] of Object.entries(fixture.versions)) {
    it(`covers exactly the ${key} fixture slots`, () => {
      const actual = classesForVersion(version.region, version.majorVersion)
        .map((c) => [c.jobIndex, c.subJobIndex, c.label] as const)
        .sort();
      const expected = version.slots
        .map((s) => [s.raceIndex, s.subJobIndex, s.class] as const)
        .sort();
      expect(actual).toEqual(expected);
    });
  }
});
