import type {
  Conversation,
  ConversationState,
  ConversationStateType,
  DialogueChoice,
  DialogueType,
} from "@/types/models/conversation";
import { getTransitions, buildStateIndex, type Transition } from "./transitions";

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

export function deriveUniqueId(
  existing: Set<string> | Conversation,
  seed = "newState",
): string {
  const taken =
    existing instanceof Set
      ? existing
      : new Set(existing.attributes.states.map(s => s.id));
  if (!taken.has(seed)) return seed;
  for (let i = 2; i < 10_000; i++) {
    const candidate = `${seed}${i}`;
    if (!taken.has(candidate)) return candidate;
  }
  return `${seed}${Date.now()}`;
}

export function emptyStateOfType(
  type: ConversationStateType,
  id: string,
): ConversationState {
  const base: ConversationState = { id, type };
  switch (type) {
    case "dialogue":
      base.dialogue = { dialogueType: "sendOk", text: "", choices: [] };
      break;
    case "listSelection":
      base.listSelection = { title: "", choices: [] };
      break;
    case "askSlideMenu":
      base.askSlideMenu = { title: "", menuType: 0, choices: [] };
      break;
    case "askNumber":
      base.askNumber = {
        text: "",
        defaultValue: 0,
        minValue: 0,
        maxValue: 0,
        nextState: "",
      };
      break;
    case "askStyle":
      base.askStyle = { text: "", nextState: "" };
      break;
    case "craftAction":
      base.craftAction = {
        itemId: 0,
        materials: [],
        quantities: [],
        mesoCost: 0,
        successState: "",
        failureState: "",
        missingMaterialsState: "",
      };
      break;
    case "transportAction":
      base.transportAction = { routeName: "", failureState: "" };
      break;
    case "partyQuestAction":
      base.partyQuestAction = { questId: "", failureState: "" };
      break;
    case "partyQuestBonusAction":
      base.partyQuestBonusAction = { failureState: "" };
      break;
    case "gachaponAction":
      base.gachaponAction = {
        gachaponId: "",
        ticketItemId: 0,
        failureState: "",
      };
      break;
    case "genericAction":
      base.genericAction = { operations: [], outcomes: [] };
      break;
  }
  return base;
}

export function switchStateType(
  conversation: Conversation,
  stateId: string,
  nextType: ConversationStateType,
): Conversation {
  const state = conversation.attributes.states.find(s => s.id === stateId);
  if (!state || state.type === nextType) return conversation;
  const replacement = emptyStateOfType(nextType, state.id);
  return replaceState(conversation, stateId, replacement);
}

export interface AddChildResult {
  conversation: Conversation;
  newStateId: string;
}

export function addChildState(
  conversation: Conversation,
  sourceId: string,
): AddChildResult | null {
  const source = conversation.attributes.states.find(s => s.id === sourceId);
  if (!source) return null;

  const newId = deriveUniqueId(conversation);
  const newState = emptyStateOfType("dialogue", newId);

  let updatedSource: ConversationState | null = null;
  const newChoice: DialogueChoice = { text: "", nextState: newId };

  switch (source.type) {
    case "dialogue":
      if (!source.dialogue) return null;
      updatedSource = {
        ...source,
        dialogue: {
          ...source.dialogue,
          choices: [...(source.dialogue.choices ?? []), newChoice],
        },
      };
      break;
    case "listSelection":
      if (!source.listSelection) return null;
      updatedSource = {
        ...source,
        listSelection: {
          ...source.listSelection,
          choices: [...(source.listSelection.choices ?? []), newChoice],
        },
      };
      break;
    case "askSlideMenu":
      if (!source.askSlideMenu) return null;
      updatedSource = {
        ...source,
        askSlideMenu: {
          ...source.askSlideMenu,
          choices: [...(source.askSlideMenu.choices ?? []), newChoice],
        },
      };
      break;
    default:
      return null;
  }

  const states = conversation.attributes.states.map(s =>
    s.id === sourceId ? updatedSource! : s,
  );
  states.push(newState);

  return {
    conversation: {
      ...conversation,
      attributes: { ...conversation.attributes, states },
    },
    newStateId: newId,
  };
}

export function canAddChild(stateType: ConversationStateType): boolean {
  return (
    stateType === "dialogue" ||
    stateType === "listSelection" ||
    stateType === "askSlideMenu"
  );
}

const DIALOGUE_DEFAULT_LABELS: Record<DialogueType, string[]> = {
  sendOk: ["Ok", "Exit"],
  sendYesNo: ["Yes", "No", "Exit"],
  sendNext: ["Next", "Exit"],
  sendNextPrev: ["Previous", "Next", "Exit"],
  sendPrev: ["Previous", "Ok", "Exit"],
  sendAcceptDecline: ["Accept", "Decline", "Exit"],
};

export function defaultChoicesForDialogueType(
  type: DialogueType,
): DialogueChoice[] {
  const labels = DIALOGUE_DEFAULT_LABELS[type] ?? ["Ok", "Exit"];
  return labels.map((text, i) => ({
    text,
    nextState: i === labels.length - 1 ? null : null,
  }));
}

export function resizeChoicesForDialogueType(
  type: DialogueType,
  existing: DialogueChoice[],
): DialogueChoice[] {
  const labels = DIALOGUE_DEFAULT_LABELS[type] ?? ["Ok", "Exit"];
  return labels.map((label, i) => {
    if (existing[i]) {
      // Preserve existing text + nextState when in range.
      return existing[i]!;
    }
    return { text: label, nextState: null };
  });
}

export function dialogueChoiceCount(type: DialogueType): number {
  return (DIALOGUE_DEFAULT_LABELS[type] ?? []).length;
}

export function clearTransition(
  conversation: Conversation,
  sourceId: string,
  kind: Transition["kind"],
  ordinal: number,
): { conversation: Conversation; cascadedDeletedIds: string[] } | null {
  const source = conversation.attributes.states.find(s => s.id === sourceId);
  if (!source) return null;

  const updatedSource = setTransitionTarget(source, kind, ordinal, null);
  const intermediate: Conversation = {
    ...conversation,
    attributes: {
      ...conversation.attributes,
      states: conversation.attributes.states.map(s =>
        s.id === sourceId ? updatedSource : s,
      ),
    },
  };

  const reachable = new Set<string>();
  const byId = new Map<string, ConversationState>();
  for (const s of intermediate.attributes.states) byId.set(s.id, s);
  const start = intermediate.attributes.startState;
  if (byId.has(start)) {
    const queue = [start];
    while (queue.length > 0) {
      const id = queue.shift()!;
      if (reachable.has(id)) continue;
      reachable.add(id);
      const s = byId.get(id);
      if (!s) continue;
      for (const t of getTransitions(s)) {
        if (t.target && byId.has(t.target) && !reachable.has(t.target)) {
          queue.push(t.target);
        }
      }
    }
  }

  const toRemove = new Set<string>();
  for (const state of intermediate.attributes.states) {
    if (state.id === start) continue;
    if (!reachable.has(state.id)) toRemove.add(state.id);
  }
  if (toRemove.size === 0) {
    return { conversation: intermediate, cascadedDeletedIds: [] };
  }

  const rewireNull = (id: string | null | undefined): string | null =>
    id && toRemove.has(id) ? null : (id ?? null);
  const states = intermediate.attributes.states
    .filter(s => !toRemove.has(s.id))
    .map(s => rewireStateRefs(s, rewireNull));

  return {
    conversation: {
      ...intermediate,
      attributes: { ...intermediate.attributes, states },
    },
    cascadedDeletedIds: Array.from(toRemove),
  };
}

export function moveChoiceUp<T>(arr: T[], i: number): T[] {
  if (i <= 0 || i >= arr.length) return arr;
  const next = arr.slice();
  const tmp = next[i - 1]!;
  next[i - 1] = next[i]!;
  next[i] = tmp;
  return next;
}

export function moveChoiceDown<T>(arr: T[], i: number): T[] {
  if (i < 0 || i >= arr.length - 1) return arr;
  const next = arr.slice();
  const tmp = next[i + 1]!;
  next[i + 1] = next[i]!;
  next[i] = tmp;
  return next;
}

function setTransitionTarget(
  state: ConversationState,
  kind: Transition["kind"],
  ordinal: number,
  newTarget: string | null,
): ConversationState {
  const clone = JSON.parse(JSON.stringify(state)) as ConversationState;
  const stringTarget = newTarget ?? "";
  switch (kind) {
    case "choice":
      if (clone.dialogue?.choices?.[ordinal]) {
        clone.dialogue.choices[ordinal].nextState = newTarget;
      } else if (clone.listSelection?.choices?.[ordinal]) {
        clone.listSelection.choices[ordinal].nextState = newTarget;
      } else if (clone.askSlideMenu?.choices?.[ordinal]) {
        clone.askSlideMenu.choices[ordinal].nextState = newTarget;
      }
      break;
    case "outcome":
      if (clone.genericAction?.outcomes?.[ordinal]) {
        clone.genericAction.outcomes[ordinal].nextState = stringTarget;
      }
      break;
    case "answer":
      if (clone.askNumber) clone.askNumber.nextState = stringTarget;
      break;
    case "selection":
      if (clone.askStyle) clone.askStyle.nextState = stringTarget;
      break;
    case "success":
      if (clone.craftAction) clone.craftAction.successState = stringTarget;
      break;
    case "failure":
      if (clone.craftAction) clone.craftAction.failureState = stringTarget;
      else if (clone.transportAction) clone.transportAction.failureState = stringTarget;
      else if (clone.partyQuestAction) clone.partyQuestAction.failureState = stringTarget;
      else if (clone.partyQuestBonusAction) clone.partyQuestBonusAction.failureState = stringTarget;
      else if (clone.gachaponAction) clone.gachaponAction.failureState = stringTarget;
      break;
    case "missing":
      if (clone.craftAction) clone.craftAction.missingMaterialsState = stringTarget;
      break;
    case "capacityFull":
      if (clone.transportAction) clone.transportAction.capacityFullState = stringTarget;
      break;
    case "alreadyInTransit":
      if (clone.transportAction) clone.transportAction.alreadyInTransitState = stringTarget;
      break;
    case "routeNotFound":
      if (clone.transportAction) clone.transportAction.routeNotFoundState = stringTarget;
      break;
    case "serviceError":
      if (clone.transportAction) clone.transportAction.serviceErrorState = stringTarget;
      break;
    case "notInParty":
      if (clone.partyQuestAction) clone.partyQuestAction.notInPartyState = stringTarget;
      break;
    case "notLeader":
      if (clone.partyQuestAction) clone.partyQuestAction.notLeaderState = stringTarget;
      break;
  }
  return clone;
}

export function insertBetween(
  conversation: Conversation,
  sourceId: string,
  kind: Transition["kind"],
  ordinal: number,
): AddChildResult | null {
  const source = conversation.attributes.states.find(s => s.id === sourceId);
  if (!source) return null;

  const transitions = getTransitions(source);
  const trans = transitions.find(
    t => t.kind === kind && t.ordinal === ordinal,
  );
  if (!trans) return null;

  const oldTarget = trans.target;
  const newId = deriveUniqueId(conversation);

  const newState: ConversationState = {
    id: newId,
    type: "dialogue",
    dialogue: {
      dialogueType: "sendOk",
      text: "",
      choices: [{ text: "", nextState: oldTarget }],
    },
  };

  const updatedSource = setTransitionTarget(source, kind, ordinal, newId);

  const states = conversation.attributes.states.map(s =>
    s.id === sourceId ? updatedSource : s,
  );
  states.push(newState);

  return {
    conversation: {
      ...conversation,
      attributes: { ...conversation.attributes, states },
    },
    newStateId: newId,
  };
}

export function insertBefore(
  conversation: Conversation,
  existingId: string,
): AddChildResult | null {
  const existing = conversation.attributes.states.find(s => s.id === existingId);
  if (!existing) return null;

  const newId = deriveUniqueId(conversation);
  const newState: ConversationState = {
    id: newId,
    type: "dialogue",
    dialogue: {
      dialogueType: "sendOk",
      text: "",
      choices: [{ text: "", nextState: existingId }],
    },
  };

  const rewire = (id: string | null | undefined) =>
    id === existingId ? newId : (id ?? null);

  const states = conversation.attributes.states.map(s =>
    rewireStateRefs(s, rewire),
  );
  states.push(newState);

  return {
    conversation: {
      ...conversation,
      attributes: {
        ...conversation.attributes,
        states,
        startState:
          conversation.attributes.startState === existingId
            ? newId
            : conversation.attributes.startState,
      },
    },
    newStateId: newId,
  };
}
