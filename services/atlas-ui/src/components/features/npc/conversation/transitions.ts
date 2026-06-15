import type {
  Conversation,
  ConversationState,
} from "@/types/models/conversation";

export interface Transition {
  source: string;
  target: string | null;
  label: string;
  kind:
    | "choice"
    | "outcome"
    | "answer"
    | "selection"
    | "success"
    | "failure"
    | "missing"
    | "capacityFull"
    | "alreadyInTransit"
    | "routeNotFound"
    | "serviceError"
    | "notInParty"
    | "notLeader"
    | "bonus";
  ordinal: number;
  context?: Record<string, string>;
  conditionSummary?: string;
}

const MAX_LABEL = 40;

function truncate(text: string, max = MAX_LABEL): string {
  if (!text) return text;
  if (text.length <= max) return text;
  return text.slice(0, max - 1).trimEnd() + "…";
}

function normalizeTarget(target: string | null | undefined): string | null {
  if (target === undefined || target === null) return null;
  if (typeof target === "string" && target.trim() === "") return null;
  return target;
}

export function getTransitions(state: ConversationState): Transition[] {
  const out: Transition[] = [];
  const push = (
    partial: Omit<Transition, "source"> & { source?: string },
  ) => {
    out.push({
      ...partial,
      source: state.id,
    } as Transition);
  };

  switch (state.type) {
    case "dialogue": {
      const choices = state.dialogue?.choices ?? [];
      choices.forEach((c, i) => {
        push({
          target: normalizeTarget(c.nextState),
          label: c.text ? truncate(c.text) : `Choice ${i + 1}`,
          kind: "choice",
          ordinal: i,
          ...(c.context && Object.keys(c.context).length > 0 && {
            context: c.context,
          }),
        });
      });
      break;
    }
    case "listSelection": {
      const choices = state.listSelection?.choices ?? [];
      choices.forEach((c, i) => {
        push({
          target: normalizeTarget(c.nextState),
          label: c.text ? truncate(c.text) : `Item ${i + 1}`,
          kind: "choice",
          ordinal: i,
          ...(c.context && Object.keys(c.context).length > 0 && {
            context: c.context,
          }),
        });
      });
      break;
    }
    case "askSlideMenu": {
      const choices = state.askSlideMenu?.choices ?? [];
      choices.forEach((c, i) => {
        push({
          target: normalizeTarget(c.nextState),
          label: c.text ? truncate(c.text) : `Option ${i + 1}`,
          kind: "choice",
          ordinal: i,
          ...(c.context && Object.keys(c.context).length > 0 && {
            context: c.context,
          }),
        });
      });
      break;
    }
    case "askNumber":
      if (state.askNumber) {
        push({
          target: normalizeTarget(state.askNumber.nextState),
          label: "answer",
          kind: "answer",
          ordinal: 0,
        });
      }
      break;
    case "askStyle":
      if (state.askStyle) {
        push({
          target: normalizeTarget(state.askStyle.nextState),
          label: "selection",
          kind: "selection",
          ordinal: 0,
        });
      }
      break;
    case "craftAction":
      if (state.craftAction) {
        push({
          target: normalizeTarget(state.craftAction.successState),
          label: "success",
          kind: "success",
          ordinal: 0,
        });
        push({
          target: normalizeTarget(state.craftAction.failureState),
          label: "failure",
          kind: "failure",
          ordinal: 1,
        });
        push({
          target: normalizeTarget(state.craftAction.missingMaterialsState),
          label: "missing mats",
          kind: "missing",
          ordinal: 2,
        });
      }
      break;
    case "transportAction":
      if (state.transportAction) {
        const t = state.transportAction;
        push({
          target: normalizeTarget(t.failureState),
          label: "failure",
          kind: "failure",
          ordinal: 0,
        });
        if (t.capacityFullState) {
          push({
            target: normalizeTarget(t.capacityFullState),
            label: "capacity full",
            kind: "capacityFull",
            ordinal: 1,
          });
        }
        if (t.alreadyInTransitState) {
          push({
            target: normalizeTarget(t.alreadyInTransitState),
            label: "in transit",
            kind: "alreadyInTransit",
            ordinal: 2,
          });
        }
        if (t.routeNotFoundState) {
          push({
            target: normalizeTarget(t.routeNotFoundState),
            label: "route missing",
            kind: "routeNotFound",
            ordinal: 3,
          });
        }
        if (t.serviceErrorState) {
          push({
            target: normalizeTarget(t.serviceErrorState),
            label: "service error",
            kind: "serviceError",
            ordinal: 4,
          });
        }
      }
      break;
    case "partyQuestAction":
      if (state.partyQuestAction) {
        const p = state.partyQuestAction;
        push({
          target: normalizeTarget(p.failureState),
          label: "failure",
          kind: "failure",
          ordinal: 0,
        });
        if (p.notInPartyState) {
          push({
            target: normalizeTarget(p.notInPartyState),
            label: "not in party",
            kind: "notInParty",
            ordinal: 1,
          });
        }
        if (p.notLeaderState) {
          push({
            target: normalizeTarget(p.notLeaderState),
            label: "not leader",
            kind: "notLeader",
            ordinal: 2,
          });
        }
      }
      break;
    case "partyQuestBonusAction":
      if (state.partyQuestBonusAction) {
        push({
          target: normalizeTarget(state.partyQuestBonusAction.failureState),
          label: "failure",
          kind: "failure",
          ordinal: 0,
        });
      }
      break;
    case "gachaponAction":
      if (state.gachaponAction) {
        push({
          target: normalizeTarget(state.gachaponAction.failureState),
          label: "failure",
          kind: "failure",
          ordinal: 0,
        });
      }
      break;
    case "genericAction":
      if (state.genericAction) {
        const outcomes = state.genericAction.outcomes ?? [];
        outcomes.forEach((o, i) => {
          const condCount = o.conditions?.length ?? 0;
          const condSummary =
            condCount > 0
              ? (o.conditions ?? [])
                  .map(c => c.type)
                  .slice(0, 2)
                  .join(", ") +
                (condCount > 2 ? `, +${condCount - 2}` : "")
              : undefined;
          const target = normalizeTarget(o.nextState);
          push({
            target,
            label: condCount
              ? condSummary || `if ${condCount} cond${condCount === 1 ? "" : "s"}`
              : `outcome ${i + 1}`,
            kind: "outcome",
            ordinal: i,
            ...(condSummary && { conditionSummary: condSummary }),
          });
        });
      }
      break;
  }
  return out;
}

export function buildStateIndex(
  conversation: Conversation,
): Map<string, ConversationState> {
  const m = new Map<string, ConversationState>();
  for (const s of conversation.attributes.states) m.set(s.id, s);
  return m;
}

export function allTransitions(conversation: Conversation): Transition[] {
  const out: Transition[] = [];
  for (const state of conversation.attributes.states) {
    for (const t of getTransitions(state)) out.push(t);
  }
  return out;
}
