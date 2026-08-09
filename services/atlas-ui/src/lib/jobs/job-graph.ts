import type { JobAvailabilityEntry } from "@/services/api/availability.service";

/**
 * One node of the tenant version's job advancement graph.
 *
 * `id` is the version's WIRE id (the key the API round-trips). `identity` is
 * the version-blind canonical token — key version-stable curation (rail
 * grouping, accent colours) on THIS, never on `id`: wire id 500 is Gm at
 * gms 48.1 and Pirate at gms 72.1, so a wire-keyed rail would render Gm
 * inside the Explorers group in pirate colours.
 */
export interface JobNode {
  id: number;
  identity: number;
  name: string;
  parent: number | null;
}

export type JobGraph = ReadonlyMap<number, JobNode>;

/**
 * The tenant's job graph: release availability (GET /api/data/job-availability)
 * INTERSECTED with WZ presence (GET /api/data/jobs), per FR-4.1. Names and
 * parent edges come from availability, which is version-correct; presence
 * only gates visibility.
 *
 * Re-rooting is applied here and ONLY here: a parent dropped by the
 * intersection becomes null, so no downstream helper ever has to handle a
 * dangling edge. This is the same rule the backend's job.Set.ParentWire
 * applies (an unavailable parent makes the entry a root, rather than
 * synthesising a grandparent edge) — applied a second time because the
 * intersection can drop a parent whose child survives.
 */
export function buildJobGraph(
  availability: readonly JobAvailabilityEntry[],
  present: ReadonlySet<number>,
): JobGraph {
  const kept = availability.filter((e) => present.has(e.id));
  const keptIds = new Set(kept.map((e) => e.id));
  const graph = new Map<number, JobNode>();
  for (const e of kept) {
    graph.set(e.id, {
      id: e.id,
      identity: e.identity,
      name: e.name,
      parent: e.parent !== null && keptIds.has(e.parent) ? e.parent : null,
    });
  }
  return graph;
}

/** Display name for a wire id, or `Job <id>` when the graph does not carry it. */
export function jobNodeName(graph: JobGraph, id: number): string {
  return graph.get(id)?.name ?? `Job ${id}`;
}

/** Direct children of a node, ascending by wire id. */
export function childrenOf(graph: JobGraph, id: number): number[] {
  return [...graph.values()]
    .filter((n) => n.parent === id)
    .map((n) => n.id)
    .sort((a, b) => a - b);
}

/** Walk parent edges to the branch root. Returns the id itself if it is a root or absent. */
export function rootOf(graph: JobGraph, id: number): number {
  let cur = graph.get(id);
  if (!cur) return id;
  // Cycle guard: the graph is built from API data, so a malformed parent
  // edge (self-parent, or a mutual cycle) must degrade to the last node
  // visited before the repeat rather than loop forever — this is a plain
  // synchronous while loop, which nothing can preempt once it's spinning.
  const visited = new Set<number>([cur.id]);
  while (cur.parent !== null) {
    const next = graph.get(cur.parent);
    if (!next || visited.has(next.id)) break;
    visited.add(next.id);
    cur = next;
  }
  return cur.id;
}

/** Root -> node advancement path (inclusive). Empty when the node is absent. */
export function jobTreePath(graph: JobGraph, id: number): JobNode[] {
  const path: JobNode[] = [];
  // Cycle guard: same reasoning as rootOf — a malformed parent edge must
  // truncate the path at the repeat instead of growing it without bound.
  const visited = new Set<number>();
  let cur = graph.get(id);
  while (cur && !visited.has(cur.id)) {
    visited.add(cur.id);
    path.unshift(cur);
    cur = cur.parent !== null ? graph.get(cur.parent) : undefined;
  }
  return path;
}

function ordinal(n: number): string {
  if (n === 1) return "1st";
  if (n === 2) return "2nd";
  if (n === 3) return "3rd";
  return `${n}th`;
}

/**
 * Tier tag for a flow chip: "Base" for a graph root with children, "" for a
 * childless root or an absent id, else the ordinal advancement depth
 * ("1st" … "10th") measured from the root.
 */
export function tierLabel(graph: JobGraph, id: number): string {
  const depth = jobTreePath(graph, id).length - 1;
  if (depth < 0) return "";
  if (depth === 0) return childrenOf(graph, id).length > 0 ? "Base" : "";
  return ordinal(depth);
}

/**
 * Every advancement chain below entryId: one array per root-to-leaf path of
 * the subtree, EXCLUDING entryId itself, DFS in ascending child order. A leaf
 * entry yields []. No availability filter is needed — the graph IS the
 * available set (buildJobGraph already intersected it).
 */
export function advancementChains(
  graph: JobGraph,
  entryId: number,
): number[][] {
  const walk = (id: number, ancestors: ReadonlySet<number>): number[][] => {
    const kids = childrenOf(graph, id);
    if (kids.length === 0) return [[]];
    const out: number[][] = [];
    for (const k of kids) {
      // Cycle guard: the graph is built from API data, so a child that
      // reappears in its own ancestor chain must terminate this branch as a
      // leaf rather than recurse forever (a stack-overflow RangeError, not a
      // hang, but still not something a rendering component should catch).
      if (ancestors.has(k)) {
        out.push([k]);
        continue;
      }
      const nextAncestors = new Set(ancestors);
      nextAncestors.add(k);
      for (const rest of walk(k, nextAncestors)) out.push([k, ...rest]);
    }
    return out;
  };
  return walk(entryId, new Set([entryId])).filter((chain) => chain.length > 0);
}

/** Count of nodes in entryId's subtree, entry included (0 if the entry is absent). */
export function subtreeCount(graph: JobGraph, entryId: number): number {
  if (!graph.has(entryId)) return 0;
  const walk = (id: number, ancestors: ReadonlySet<number>): number => {
    return (
      1 +
      childrenOf(graph, id).reduce((n, k) => {
        // Cycle guard: mirrors advancementChains' ancestor-chain check — a
        // child that reappears in its own ancestor chain stops recursing
        // there instead of overflowing the call stack.
        if (ancestors.has(k)) return n;
        const nextAncestors = new Set(ancestors);
        nextAncestors.add(k);
        return n + walk(k, nextAncestors);
      }, 0)
    );
  };
  return walk(entryId, new Set([entryId]));
}
