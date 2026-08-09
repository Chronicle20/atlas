import { jobTreePath, subtreeCount, type JobGraph } from "@/lib/jobs/job-graph";

export interface RailEntry {
  /**
   * The CANONICAL job identity whose advancement flow the entry shows —
   * never the wire id. Rail curation ("Explorers", "Special", the accent
   * colours) is an editorial grouping that cannot be derived from graph
   * shape: at gms 48.1 Warrior, Magician, Bowman, Rogue AND Gm are all
   * depth-1 children of Beginner, and nothing structural says which are
   * Explorers. So it needs a version-stable key for a job CONCEPT, and the
   * wire id is not one — wire id 500 is Pirate at gms 72.1 but Gm at
   * gms 48.1, which a wire-keyed rail would file under Explorers in pirate
   * colours (task-202 design D9).
   */
  identity: number;
  /** Theme token name for the branch accent, e.g. "--c-warrior" (src/index.css). */
  accent: string;
}

export interface RailGroup {
  label: string;
  entries: RailEntry[];
}

/** The Warrior rail entry; also branchEntryOf's fallback for Beginner/unknown ids. */
const WARRIOR_ENTRY: RailEntry = { identity: 100, accent: "--c-warrior" };

// Rail entries per PRD FR-3.1; accents are scoped via style={{ "--acc": `var(${accent})` }}.
export const RAIL_GROUPS: RailGroup[] = [
  {
    label: "Explorers",
    entries: [
      WARRIOR_ENTRY,
      { identity: 200, accent: "--c-magician" },
      { identity: 300, accent: "--c-bowman" },
      { identity: 400, accent: "--c-thief" },
      { identity: 500, accent: "--c-pirate" },
    ],
  },
  {
    label: "Cygnus Knights",
    entries: [{ identity: 1000, accent: "--c-cygnus" }],
  },
  {
    label: "Legends",
    entries: [
      { identity: 2000, accent: "--c-aran" },
      { identity: 2001, accent: "--c-evan" },
    ],
  },
  {
    label: "Special",
    entries: [
      { identity: 800, accent: "--c-special" },
      { identity: 900, accent: "--c-special" },
    ],
  },
];

/** The wire id this version binds a canonical identity to, or undefined. */
function wireIdOf(graph: JobGraph, identity: number): number | undefined {
  for (const n of graph.values()) {
    if (n.identity === identity) return n.id;
  }
  return undefined;
}

/**
 * The rail entry whose node lies on jobId's advancement path. Beginner and
 * unknown ids fall back to the first entry (Warrior) — the caller keeps the
 * job selection itself.
 */
export function branchEntryOf(graph: JobGraph, jobId: number): RailEntry {
  const pathIdentities = new Set(
    jobTreePath(graph, jobId).map((n) => n.identity),
  );
  for (const g of RAIL_GROUPS) {
    for (const e of g.entries) {
      if (pathIdentities.has(e.identity)) return e;
    }
  }
  return WARRIOR_ENTRY;
}

export interface VisibleRailEntry extends RailEntry {
  /** This version's wire id for the entry — what selection and routing use. */
  id: number;
  name: string;
  count: number;
}

export interface VisibleRailGroup {
  label: string;
  entries: VisibleRailEntry[];
}

/**
 * Rail groups for the tenant's job graph, with display name + subtree count;
 * empty groups dropped (FR-4.6). The graph IS the intersected available set,
 * so a class the version never released — or never ingested — takes its whole
 * group with it.
 */
export function visibleRailGroups(graph: JobGraph): VisibleRailGroup[] {
  return RAIL_GROUPS.map((g) => ({
    label: g.label,
    entries: g.entries.flatMap((e) => {
      const id = wireIdOf(graph, e.identity);
      if (id === undefined) return [];
      return [
        {
          ...e,
          id,
          name: graph.get(id)?.name ?? `Job ${id}`,
          count: subtreeCount(graph, id),
        },
      ];
    }),
  })).filter((g) => g.entries.length > 0);
}
