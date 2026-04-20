import type { Conversation } from "@/types/models/conversation";
import {
  allTransitions,
  buildStateIndex,
  getTransitions,
  type Transition,
} from "./transitions";

export interface GraphAnalysis {
  backEdges: Set<string>;
  reachable: Set<string>;
  unreachable: string[];
  brokenRefs: Array<{ source: string; target: string }>;
  duplicateIds: string[];
  terminals: Set<string>;
  inboundCount: Map<string, number>;
  outboundCount: Map<string, number>;
}

function edgeKey(sourceId: string, targetId: string): string {
  return `${sourceId}->${targetId}`;
}

function computeReachable(conversation: Conversation): Set<string> {
  const visited = new Set<string>();
  const byId = buildStateIndex(conversation);
  const start = conversation.attributes.startState;
  if (!byId.has(start)) return visited;
  const queue = [start];
  while (queue.length > 0) {
    const id = queue.shift()!;
    if (visited.has(id)) continue;
    visited.add(id);
    const state = byId.get(id);
    if (!state) continue;
    for (const t of getTransitions(state)) {
      if (t.target && byId.has(t.target) && !visited.has(t.target)) {
        queue.push(t.target);
      }
    }
  }
  return visited;
}

function detectBackEdges(conversation: Conversation): Set<string> {
  const byId = buildStateIndex(conversation);
  const visited = new Set<string>();
  const stack = new Set<string>();
  const backEdges = new Set<string>();

  function dfs(stateId: string) {
    if (visited.has(stateId) || stack.has(stateId)) return;
    stack.add(stateId);
    const state = byId.get(stateId);
    if (state) {
      for (const t of getTransitions(state)) {
        if (!t.target || !byId.has(t.target)) continue;
        if (stack.has(t.target)) {
          backEdges.add(edgeKey(stateId, t.target));
        } else if (!visited.has(stateId)) {
          dfs(t.target);
        }
      }
    }
    stack.delete(stateId);
    visited.add(stateId);
  }

  const start = conversation.attributes.startState;
  if (byId.has(start)) dfs(start);
  for (const s of conversation.attributes.states) {
    if (!visited.has(s.id)) dfs(s.id);
  }
  return backEdges;
}

export function analyze(conversation: Conversation): GraphAnalysis {
  const byId = buildStateIndex(conversation);
  const backEdges = detectBackEdges(conversation);
  const reachable = computeReachable(conversation);
  const unreachable: string[] = [];
  for (const s of conversation.attributes.states) {
    if (!reachable.has(s.id)) unreachable.push(s.id);
  }

  const seen = new Set<string>();
  const dupes = new Set<string>();
  for (const s of conversation.attributes.states) {
    if (seen.has(s.id)) dupes.add(s.id);
    seen.add(s.id);
  }

  const inboundCount = new Map<string, number>();
  const outboundCount = new Map<string, number>();
  const brokenRefs: Array<{ source: string; target: string }> = [];
  const terminals = new Set<string>();

  const transitions = allTransitions(conversation);
  const sourceHasNonNull = new Map<string, boolean>();
  for (const t of transitions) {
    if (t.target === null) continue;
    if (!byId.has(t.target)) {
      brokenRefs.push({ source: t.source, target: t.target });
    } else {
      inboundCount.set(t.target, (inboundCount.get(t.target) ?? 0) + 1);
    }
    outboundCount.set(t.source, (outboundCount.get(t.source) ?? 0) + 1);
    sourceHasNonNull.set(t.source, true);
  }

  for (const s of conversation.attributes.states) {
    if (!sourceHasNonNull.has(s.id)) terminals.add(s.id);
  }

  return {
    backEdges,
    reachable,
    unreachable,
    brokenRefs,
    duplicateIds: Array.from(dupes),
    terminals,
    inboundCount,
    outboundCount,
  };
}

export function makeEdgeKey(t: Transition): string | null {
  if (!t.target) return null;
  return edgeKey(t.source, t.target);
}
