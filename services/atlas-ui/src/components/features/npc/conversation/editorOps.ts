import type {
  Conversation,
  ConversationState,
} from "@/types/models/conversation";
import { getTransitions, buildStateIndex } from "./transitions";

export function replaceState(
  conversation: Conversation,
  id: string,
  next: ConversationState,
): Conversation {
  return {
    ...conversation,
    attributes: {
      ...conversation.attributes,
      states: conversation.attributes.states.map(s =>
        s.id === id ? next : s,
      ),
    },
  };
}

function rewireStateRefs(
  state: ConversationState,
  rewire: (id: string | null | undefined) => string | null,
): ConversationState {
  const next: ConversationState = JSON.parse(JSON.stringify(state));
  if (next.dialogue) {
    next.dialogue.choices = (next.dialogue.choices ?? []).map(c => ({
      ...c,
      nextState: rewire(c.nextState) ?? null,
    }));
  }
  if (next.listSelection) {
    next.listSelection.choices = (next.listSelection.choices ?? []).map(c => ({
      ...c,
      nextState: rewire(c.nextState) ?? null,
    }));
  }
  if (next.askSlideMenu) {
    next.askSlideMenu.choices = (next.askSlideMenu.choices ?? []).map(c => ({
      ...c,
      nextState: rewire(c.nextState) ?? null,
    }));
  }
  if (next.askNumber) {
    next.askNumber.nextState = rewire(next.askNumber.nextState) ?? next.askNumber.nextState;
  }
  if (next.askStyle) {
    next.askStyle.nextState = rewire(next.askStyle.nextState) ?? next.askStyle.nextState;
  }
  if (next.craftAction) {
    const ca = next.craftAction;
    ca.successState = rewire(ca.successState) ?? ca.successState;
    ca.failureState = rewire(ca.failureState) ?? ca.failureState;
    ca.missingMaterialsState = rewire(ca.missingMaterialsState) ?? ca.missingMaterialsState;
  }
  if (next.transportAction) {
    const ta = next.transportAction;
    ta.failureState = rewire(ta.failureState) ?? ta.failureState;
    if (ta.capacityFullState !== undefined) {
      ta.capacityFullState = rewire(ta.capacityFullState) ?? ta.capacityFullState;
    }
    if (ta.alreadyInTransitState !== undefined) {
      ta.alreadyInTransitState = rewire(ta.alreadyInTransitState) ?? ta.alreadyInTransitState;
    }
    if (ta.routeNotFoundState !== undefined) {
      ta.routeNotFoundState = rewire(ta.routeNotFoundState) ?? ta.routeNotFoundState;
    }
    if (ta.serviceErrorState !== undefined) {
      ta.serviceErrorState = rewire(ta.serviceErrorState) ?? ta.serviceErrorState;
    }
  }
  if (next.partyQuestAction) {
    const p = next.partyQuestAction;
    p.failureState = rewire(p.failureState) ?? p.failureState;
    if (p.notInPartyState !== undefined) {
      p.notInPartyState = rewire(p.notInPartyState) ?? p.notInPartyState;
    }
    if (p.notLeaderState !== undefined) {
      p.notLeaderState = rewire(p.notLeaderState) ?? p.notLeaderState;
    }
  }
  if (next.partyQuestBonusAction) {
    next.partyQuestBonusAction.failureState =
      rewire(next.partyQuestBonusAction.failureState) ??
      next.partyQuestBonusAction.failureState;
  }
  if (next.gachaponAction) {
    next.gachaponAction.failureState =
      rewire(next.gachaponAction.failureState) ?? next.gachaponAction.failureState;
  }
  if (next.genericAction?.outcomes) {
    next.genericAction.outcomes = next.genericAction.outcomes.map(o => ({
      ...o,
      nextState: rewire(o.nextState) ?? o.nextState,
    }));
  }
  return next;
}

export function renameState(
  conversation: Conversation,
  oldId: string,
  newId: string,
): Conversation {
  if (oldId === newId) return conversation;
  const rewire = (id: string | null | undefined) =>
    id === oldId ? newId : (id ?? null);
  const states = conversation.attributes.states.map(state => {
    const rewired = rewireStateRefs(state, rewire);
    if (rewired.id === oldId) return { ...rewired, id: newId };
    return rewired;
  });
  return {
    ...conversation,
    attributes: {
      ...conversation.attributes,
      states,
      startState:
        conversation.attributes.startState === oldId
          ? newId
          : conversation.attributes.startState,
    },
  };
}

export interface DeleteImpact {
  targetId: string;
  incomingFromKept: Array<{ source: string; label: string }>;
  wouldBecomeUnreachable: string[];
  isStart: boolean;
}

export function previewDelete(
  conversation: Conversation,
  targetId: string,
): DeleteImpact {
  const incoming: Array<{ source: string; label: string }> = [];
  for (const state of conversation.attributes.states) {
    if (state.id === targetId) continue;
    for (const t of getTransitions(state)) {
      if (t.target === targetId) incoming.push({ source: state.id, label: t.label });
    }
  }
  const byId = new Map<string, ConversationState>();
  for (const s of conversation.attributes.states) byId.set(s.id, s);

  const reach = new Set<string>();
  const startId = conversation.attributes.startState;
  if (byId.has(startId) && startId !== targetId) {
    const queue = [startId];
    while (queue.length > 0) {
      const id = queue.shift()!;
      if (reach.has(id)) continue;
      reach.add(id);
      const state = byId.get(id);
      if (!state) continue;
      for (const t of getTransitions(state)) {
        if (!t.target) continue;
        if (t.target === targetId) continue;
        if (reach.has(t.target)) continue;
        if (!byId.has(t.target)) continue;
        queue.push(t.target);
      }
    }
  }

  const wouldBecomeUnreachable: string[] = [];
  for (const s of conversation.attributes.states) {
    if (s.id === targetId) continue;
    if (!reach.has(s.id)) {
      const prevReachable = computeCurrentlyReachable(conversation).has(s.id);
      if (prevReachable) wouldBecomeUnreachable.push(s.id);
    }
  }

  return {
    targetId,
    incomingFromKept: incoming,
    wouldBecomeUnreachable,
    isStart: startId === targetId,
  };
}

function computeCurrentlyReachable(conversation: Conversation): Set<string> {
  const byId = buildStateIndex(conversation);
  const reach = new Set<string>();
  const start = conversation.attributes.startState;
  if (!byId.has(start)) return reach;
  const queue = [start];
  while (queue.length > 0) {
    const id = queue.shift()!;
    if (reach.has(id)) continue;
    reach.add(id);
    const state = byId.get(id);
    if (!state) continue;
    for (const t of getTransitions(state)) {
      if (t.target && byId.has(t.target) && !reach.has(t.target)) {
        queue.push(t.target);
      }
    }
  }
  return reach;
}

export function deleteState(
  conversation: Conversation,
  targetId: string,
  options: { cascade: boolean },
): Conversation {
  const toRemove = new Set<string>([targetId]);
  if (options.cascade) {
    const impact = previewDelete(conversation, targetId);
    for (const id of impact.wouldBecomeUnreachable) toRemove.add(id);
  }
  const rewireNull = (id: string | null | undefined): string | null =>
    id && toRemove.has(id) ? null : (id ?? null);

  const states = conversation.attributes.states
    .filter(s => !toRemove.has(s.id))
    .map(s => rewireStateRefs(s, rewireNull));

  return {
    ...conversation,
    attributes: {
      ...conversation.attributes,
      states,
    },
  };
}

export function idIsTaken(
  conversation: Conversation,
  id: string,
  exceptId?: string,
): boolean {
  return conversation.attributes.states.some(
    s => s.id === id && s.id !== exceptId,
  );
}
