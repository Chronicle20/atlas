import type { CharacterTemplate } from "@/types/models/template";

// World names mirror atlas-character-factory job/model.go JobFromIndex:
// 0 → Noblesse (Cygnus Knights), 1 → Beginner (Adventurer), 2 → Legend (Aran),
// 3 → Evan. Unknown indexes are permitted (backend is the validator of record).
const WORLD_NAMES: Record<number, string> = {
  0: "Cygnus Knights",
  1: "Adventurer",
  2: "Aran",
  3: "Evan",
};

export function worldNameFromJobIndex(jobIndex: number): string {
  return WORLD_NAMES[jobIndex] ?? `Job ${jobIndex}`;
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
): string[] {
  const seen = new Map<string, number>();
  return templates.map((t) => {
    const base = `${worldNameFromJobIndex(t.jobIndex)} · ${genderLabel(t.gender)}`;
    const n = (seen.get(base) ?? 0) + 1;
    seen.set(base, n);
    return n === 1 ? base : `${base} (${n})`;
  });
}

export const KNOWN_CLASSES: readonly {
  jobIndex: number;
  subJobIndex: number;
  label: string;
}[] = [
  { jobIndex: 0, subJobIndex: 0, label: "Cygnus Knights (0.0)" },
  { jobIndex: 1, subJobIndex: 0, label: "Adventurer (1.0)" },
  { jobIndex: 2, subJobIndex: 0, label: "Aran (2.0)" },
  { jobIndex: 3, subJobIndex: 0, label: "Evan (3.0)" },
];
