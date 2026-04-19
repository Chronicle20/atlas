import { useMemo } from "react";
import { ArrowRight, CornerDownRight } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import type {
  Conversation,
  ConversationState,
} from "@/types/models/conversation";

interface NpcConversationTreePreviewProps {
  conversation: Conversation;
  maxNodes?: number;
}

interface RenderedNode {
  id: string;
  depth: number;
  state: ConversationState | null;
  transitions: StateTransition[];
}

interface StateTransition {
  label: string;
  target: string | null;
}

const TYPE_LABEL: Record<ConversationState["type"], string> = {
  dialogue: "Dialogue",
  genericAction: "Action",
  craftAction: "Craft",
  listSelection: "List",
  askNumber: "Number",
  askStyle: "Style",
};

function truncate(text: string, max: number): string {
  if (text.length <= max) return text;
  return text.slice(0, max - 1).trimEnd() + "…";
}

function getStateTransitions(state: ConversationState): StateTransition[] {
  switch (state.type) {
    case "dialogue":
      return (
        state.dialogue?.choices.map((choice, i) => ({
          label: choice.text
            ? truncate(choice.text, 24)
            : `Choice ${i + 1}`,
          target: choice.nextState,
        })) ?? []
      );
    case "genericAction":
      return (
        state.genericAction?.outcomes.map((outcome, i) => ({
          label: outcome.conditions.length
            ? `if ${outcome.conditions.length} cond${outcome.conditions.length === 1 ? "" : "s"}`
            : `outcome ${i + 1}`,
          target: outcome.nextState,
        })) ?? []
      );
    case "craftAction":
      if (!state.craftAction) return [];
      return [
        { label: "success", target: state.craftAction.successState },
        { label: "failure", target: state.craftAction.failureState },
        {
          label: "missing mats",
          target: state.craftAction.missingMaterialsState,
        },
      ];
    case "listSelection":
      return (
        state.listSelection?.choices.map((choice, i) => ({
          label: choice.text
            ? truncate(choice.text, 24)
            : `Choice ${i + 1}`,
          target: choice.nextState,
        })) ?? []
      );
    case "askNumber":
      return state.askNumber
        ? [{ label: "answer", target: state.askNumber.nextState }]
        : [];
    case "askStyle":
      return state.askStyle
        ? [{ label: "selection", target: state.askStyle.nextState }]
        : [];
    default:
      return [];
  }
}

function describeState(state: ConversationState): string {
  switch (state.type) {
    case "dialogue":
      return state.dialogue?.text ? truncate(state.dialogue.text, 80) : "";
    case "genericAction": {
      const opCount = state.genericAction?.operations.length ?? 0;
      const outcomeCount = state.genericAction?.outcomes.length ?? 0;
      return `${opCount} op${opCount === 1 ? "" : "s"} · ${outcomeCount} outcome${outcomeCount === 1 ? "" : "s"}`;
    }
    case "craftAction":
      return state.craftAction
        ? `Craft item ${state.craftAction.itemId}`
        : "";
    case "listSelection":
      return state.listSelection?.title
        ? truncate(state.listSelection.title, 80)
        : "";
    case "askNumber":
      return state.askNumber?.text ? truncate(state.askNumber.text, 80) : "";
    case "askStyle":
      return state.askStyle?.text ? truncate(state.askStyle.text, 80) : "";
    default:
      return "";
  }
}

function buildPreview(
  conversation: Conversation,
  maxNodes: number,
): RenderedNode[] {
  const stateMap = new Map<string, ConversationState>();
  for (const state of conversation.attributes.states) {
    stateMap.set(state.id, state);
  }

  const visited = new Set<string>();
  const out: RenderedNode[] = [];
  const startState = conversation.attributes.startState;

  function walk(id: string, depth: number) {
    if (out.length >= maxNodes) return;
    if (visited.has(id)) {
      out.push({ id, depth, state: null, transitions: [] });
      return;
    }
    visited.add(id);
    const state = stateMap.get(id);
    if (!state) {
      out.push({ id, depth, state: null, transitions: [] });
      return;
    }
    const transitions = getStateTransitions(state);
    out.push({ id, depth, state, transitions });
    for (const t of transitions) {
      if (!t.target) continue;
      walk(t.target, depth + 1);
      if (out.length >= maxNodes) return;
    }
  }

  walk(startState, 0);
  return out;
}

export function NpcConversationTreePreview({
  conversation,
  maxNodes = 12,
}: NpcConversationTreePreviewProps) {
  const nodes = useMemo(
    () => buildPreview(conversation, maxNodes),
    [conversation, maxNodes],
  );

  if (nodes.length === 0) return null;

  const truncated =
    nodes.length >= maxNodes &&
    conversation.attributes.states.length > nodes.filter(n => n.state !== null).length;

  return (
    <div className="flex flex-col gap-1.5">
      {nodes.map((node, idx) => (
        <ConversationTreeNode key={`${node.id}-${idx}`} node={node} />
      ))}
      {truncated && (
        <p className="text-xs text-muted-foreground pl-6">
          … {conversation.attributes.states.length - nodes.filter(n => n.state !== null).length} more state(s) not shown
        </p>
      )}
    </div>
  );
}

interface ConversationTreeNodeProps {
  node: RenderedNode;
}

function ConversationTreeNode({ node }: ConversationTreeNodeProps) {
  const indent = Math.min(node.depth, 6);
  const isCycle = node.state === null;
  const description = node.state ? describeState(node.state) : "cycle or missing";
  return (
    <div
      className="flex items-start gap-2 text-sm"
      style={{ paddingLeft: `${indent * 12}px` }}
    >
      {node.depth > 0 && (
        <CornerDownRight className="h-3.5 w-3.5 mt-1 text-muted-foreground shrink-0" />
      )}
      <div className="flex flex-col gap-0.5 min-w-0 flex-1">
        <div className="flex items-center gap-2 flex-wrap">
          {node.state ? (
            <Badge variant="outline" className="text-[10px] px-1.5 py-0">
              {TYPE_LABEL[node.state.type]}
            </Badge>
          ) : (
            <Badge variant="secondary" className="text-[10px] px-1.5 py-0">
              ref
            </Badge>
          )}
          <span className="font-mono text-xs truncate">{node.id}</span>
        </div>
        {description && !isCycle && (
          <p className="text-xs text-muted-foreground truncate">{description}</p>
        )}
        {node.transitions.length > 0 && (
          <div className="flex flex-col gap-0.5">
            {node.transitions.map((t, i) => (
              <div
                key={i}
                className="flex items-center gap-1 text-xs text-muted-foreground"
              >
                <ArrowRight className="h-3 w-3 shrink-0" />
                <span className="truncate">{t.label}</span>
                {t.target && (
                  <span className="font-mono text-[11px] text-foreground/70 truncate">
                    → {t.target}
                  </span>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
