import {
  JOB_GRAPH,
  floorOf,
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

// Rail entries per PRD FR-3.1; accents are scoped via style={{ "--acc": `var(${accent})` }}.
export const RAIL_GROUPS: RailGroup[] = [
  {
    label: "Explorers",
    entries: [
      { id: 100, accent: "--c-warrior" },
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
  return RAIL_GROUPS[0].entries[0];
}

export interface VisibleRailEntry extends RailEntry {
  name: string;
  count: number;
}

export interface VisibleRailGroup {
  label: string;
  entries: VisibleRailEntry[];
}

/** Version-gated rail groups with display name + visible-subtree count; empty groups dropped. */
export function visibleRailGroups(major: number): VisibleRailGroup[] {
  return RAIL_GROUPS.map((g) => ({
    label: g.label,
    entries: g.entries
      .filter((e) => floorOf(e.id) <= major)
      .map((e) => ({
        ...e,
        name: JOB_GRAPH[e.id]?.name ?? `Job ${e.id}`,
        count: subtreeCount(e.id, major),
      })),
  })).filter((g) => g.entries.length > 0);
}
