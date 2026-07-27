import {
  JOB_GRAPH,
  jobTreePath,
  subtreeCount,
} from "@/lib/jobs/job-advancement-tree";

export interface RailEntry {
  /** JOB_GRAPH node whose advancement flow the entry shows. */
  id: number;
  /** Theme token name for the branch accent, e.g. "--c-warrior" (src/index.css). */
  accent: string;
}

export interface RailGroup {
  label: string;
  entries: RailEntry[];
}

/** The Warrior rail entry; also branchEntryOf's fallback for Beginner/unknown ids. */
const WARRIOR_ENTRY: RailEntry = { id: 100, accent: "--c-warrior" };

// Rail entries per PRD FR-3.1; accents are scoped via style={{ "--acc": `var(${accent})` }}.
export const RAIL_GROUPS: RailGroup[] = [
  {
    label: "Explorers",
    entries: [
      WARRIOR_ENTRY,
      { id: 200, accent: "--c-magician" },
      { id: 300, accent: "--c-bowman" },
      { id: 400, accent: "--c-thief" },
      { id: 500, accent: "--c-pirate" },
    ],
  },
  { label: "Cygnus Knights", entries: [{ id: 1000, accent: "--c-cygnus" }] },
  {
    label: "Legends",
    entries: [
      { id: 2000, accent: "--c-aran" },
      { id: 2001, accent: "--c-evan" },
    ],
  },
  {
    label: "Special",
    entries: [
      { id: 800, accent: "--c-special" },
      { id: 900, accent: "--c-special" },
    ],
  },
];

/**
 * The rail entry whose node lies on jobId's advancement path. Beginner (0) and
 * unknown ids fall back to the first entry (Warrior) — the caller keeps the
 * job selection itself.
 */
export function branchEntryOf(jobId: number): RailEntry {
  const path = jobTreePath(jobId).map((e) => e.id);
  for (const g of RAIL_GROUPS) {
    for (const e of g.entries) {
      if (path.includes(e.id)) return e;
    }
  }
  return WARRIOR_ENTRY;
}

export interface VisibleRailEntry extends RailEntry {
  name: string;
  count: number;
}

export interface VisibleRailGroup {
  label: string;
  entries: VisibleRailEntry[];
}

/** Rail groups for the tenant's job set, with display name + subtree count; empty groups dropped. */
export function visibleRailGroups(
  available: ReadonlySet<number>,
): VisibleRailGroup[] {
  return RAIL_GROUPS.map((g) => ({
    label: g.label,
    entries: g.entries
      .filter((e) => available.has(e.id))
      .map((e) => ({
        ...e,
        name: JOB_GRAPH[e.id]?.name ?? `Job ${e.id}`,
        count: subtreeCount(e.id, available),
      })),
  })).filter((g) => g.entries.length > 0);
}
