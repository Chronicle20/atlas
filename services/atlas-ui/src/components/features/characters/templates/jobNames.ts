import type { CharacterTemplate } from "@/types/models/template";

export type RaceClass = {
  jobIndex: number;
  subJobIndex: number;
  label: string;
};

// Per-version race carousels, transcribed from docs/packets/race-carousels.json, which is
// the machine-readable projection of the IDA findings in
// docs/tasks/task-283-race-index-job-mapping/findings.md. raceCarousels.parity.test.ts
// asserts this table against that file; a drift here fails that test rather than silently
// mislabelling a tenant admin's editor.

// gms_12: no IDA export, no IDB. Lone (1,0) slot, present in every candidate mapping.
const gms12Classes: readonly RaceClass[] = [
  { jobIndex: 1, subJobIndex: 0, label: "Explorer" },
];

// gms_v48, gms_v61, gms_v72: CLogin::SendNewCharPacket encodes no race member on any of
// these versions, so the only reachable slot is (1,0).
const noRaceClasses: readonly RaceClass[] = [
  {
    jobIndex: 1,
    subJobIndex: 0,
    label: "Explorer only; no race index exists on this version",
  },
];

// gms_v79: three-arm switch. Ordinal 2's class is unverified by geometry only (no class
// symbol in this IDB).
const race3V79Classes: readonly RaceClass[] = [
  { jobIndex: 0, subJobIndex: 0, label: "Cygnus Knight" },
  { jobIndex: 1, subJobIndex: 0, label: "Explorer" },
  { jobIndex: 2, subJobIndex: 0, label: "Aran-family dialog" },
];

// gms_v83: three-arm switch, anchor column — class symbols named directly in this IDB.
const race3V83Classes: readonly RaceClass[] = [
  { jobIndex: 0, subJobIndex: 0, label: "Cygnus Knight" },
  { jobIndex: 1, subJobIndex: 0, label: "Explorer" },
  { jobIndex: 2, subJobIndex: 0, label: "Aran" },
];

// gms_v84, gms_v87: four-arm switch. Ordinal 3's class is unverified (distinct singleton,
// no symbol or RTTI string to pin which class it is).
const race4V84V87Classes: readonly RaceClass[] = [
  { jobIndex: 0, subJobIndex: 0, label: "Cygnus Knight" },
  { jobIndex: 1, subJobIndex: 0, label: "Explorer" },
  { jobIndex: 2, subJobIndex: 0, label: "Aran" },
  { jobIndex: 3, subJobIndex: 0, label: "fourth race (Evan slot)" },
];

// gms_v92: same four-arm shape, but which of ordinals 2/3 is Aran vs. the fourth race was
// not established within budget.
const race4V92Classes: readonly RaceClass[] = [
  { jobIndex: 0, subJobIndex: 0, label: "Cygnus Knight" },
  { jobIndex: 1, subJobIndex: 0, label: "Explorer" },
  { jobIndex: 2, subJobIndex: 0, label: "Aran or Evan (one of the two)" },
  { jobIndex: 3, subJobIndex: 0, label: "Aran or Evan (the other)" },
  {
    jobIndex: 1,
    subJobIndex: 1,
    label: "—",
  },
];

// gms_v95: five-arm jump table, the only version with Resistance/Citizen and a live
// sub-job field.
const race5V95Classes: readonly RaceClass[] = [
  { jobIndex: 0, subJobIndex: 0, label: "Resistance / Citizen" },
  { jobIndex: 1, subJobIndex: 0, label: "Explorer" },
  {
    jobIndex: 1,
    subJobIndex: 1,
    label: "Explorer, sub-job 1 (Dual Blade)",
  },
  { jobIndex: 2, subJobIndex: 0, label: "Cygnus Knight" },
  { jobIndex: 3, subJobIndex: 0, label: "Aran" },
  { jobIndex: 4, subJobIndex: 0, label: "Evan" },
];

// gms_jms_185: same four-ordinal shape as race4, but every class identity is unverified in
// this IDB — do not reuse the race4 class labels here.
const race4Jms185Classes: readonly RaceClass[] = [
  { jobIndex: 0, subJobIndex: 0, label: "unverified" },
  { jobIndex: 1, subJobIndex: 0, label: "unverified" },
  { jobIndex: 2, subJobIndex: 0, label: "unverified" },
  { jobIndex: 3, subJobIndex: 0, label: "unverified" },
  { jobIndex: 1, subJobIndex: 1, label: "—" },
];

export function classesForVersion(
  region: string | undefined,
  majorVersion: number | undefined,
): readonly RaceClass[] {
  if (region === undefined || majorVersion === undefined) {
    // No tenant selected: nothing to resolve against, so every ordinal falls through to
    // worldNameFromJobIndex's `Job ${jobIndex}` fallback (FR-23) rather than guessing a
    // version.
    return [];
  }
  if (region === "JMS" && majorVersion >= 185) {
    return race4Jms185Classes;
  }
  if (region === "GMS" && majorVersion >= 95) {
    return race5V95Classes;
  }
  if (region === "GMS" && majorVersion === 92) {
    return race4V92Classes;
  }
  if (region === "GMS" && majorVersion >= 84 && majorVersion <= 92) {
    return race4V84V87Classes;
  }
  if (region === "GMS" && majorVersion === 83) {
    return race3V83Classes;
  }
  if (region === "GMS" && majorVersion === 79) {
    return race3V79Classes;
  }
  if (region === "GMS" && majorVersion >= 79 && majorVersion <= 83) {
    // gms_v80/81/82: no fixture entries of their own. Go's carouselFor
    // (character-factory/job/carousel.go:96, MajorInRange(79, 83)) routes them to the same
    // race3Carousel as 79 and 83, which assigns identical job ids either way -- this is a
    // label-only choice. race3V83Classes is the anchor column (carousel.go:38, "class
    // symbols named directly in this IDB"), so the interpolated band inherits its
    // confirmed labels rather than v79's geometry-only "Aran-family dialog" guess.
    return race3V83Classes;
  }
  if (region === "GMS" && majorVersion >= 48 && majorVersion <= 72) {
    return noRaceClasses;
  }
  // gms_12 has no IDA export and cannot be verified; its lone seeded slot is (1,0) ->
  // Explorer, present in every candidate mapping. Also the fallback for any tenant version
  // that matches none of the arms above.
  return gms12Classes;
}

export function worldNameFromJobIndex(
  jobIndex: number,
  region: string | undefined,
  majorVersion: number | undefined,
): string {
  const found = classesForVersion(region, majorVersion).find(
    (c) => c.jobIndex === jobIndex,
  );
  return found?.label ?? `Job ${jobIndex}`;
}

export function genderLabel(gender: number): "M" | "F" {
  return gender === 1 ? "F" : "M";
}

/**
 * Segmented-control labels: "<World> · <M|F>", with " (2)", " (3)" ordinals
 * appended to the second and later occurrences of a duplicate label.
 */
export function templateLabels(
  templates: Pick<CharacterTemplate, "jobIndex" | "gender">[],
  region: string | undefined,
  majorVersion: number | undefined,
): string[] {
  const seen = new Map<string, number>();
  return templates.map((t) => {
    const base = `${worldNameFromJobIndex(t.jobIndex, region, majorVersion)} · ${genderLabel(t.gender)}`;
    const n = (seen.get(base) ?? 0) + 1;
    seen.set(base, n);
    return n === 1 ? base : `${base} (${n})`;
  });
}
