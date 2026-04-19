import { useMemo } from "react";
import ReactFlow, {
  Background,
  Controls,
  Handle,
  Position,
  ReactFlowProvider,
  type Edge,
  type Node,
  type NodeProps,
} from "reactflow";
import "reactflow/dist/style.css";

import { Badge } from "@/components/ui/badge";
import type {
  Conversation,
  ConversationState,
} from "@/types/models/conversation";

interface NpcConversationStateMachineProps {
  conversation: Conversation;
  height?: number;
}

const NODE_WIDTH = 240;
const NODE_HEIGHT = 120;
const LEVEL_GAP_X = 80;
const NODE_GAP_Y = 24;

const TYPE_LABEL: Record<ConversationState["type"], string> = {
  dialogue: "Dialogue",
  genericAction: "Action",
  craftAction: "Craft",
  listSelection: "List",
  askNumber: "Number",
  askStyle: "Style",
};

interface StateTransition {
  label: string;
  target: string | null;
}

interface StateNodeData {
  stateId: string;
  stateType: ConversationState["type"];
  description: string;
  isStart: boolean;
  isTerminal: boolean;
}

function truncate(text: string, max: number): string {
  if (text.length <= max) return text;
  return text.slice(0, max - 1).trimEnd() + "…";
}

function getStateTransitions(state: ConversationState): StateTransition[] {
  switch (state.type) {
    case "dialogue":
      return (
        state.dialogue?.choices?.map((choice, i) => ({
          label: choice.text ? truncate(choice.text, 32) : `Choice ${i + 1}`,
          target: choice.nextState,
        })) ?? []
      );
    case "genericAction":
      return (
        state.genericAction?.outcomes?.map((outcome, i) => {
          const condCount = outcome.conditions?.length ?? 0;
          return {
            label: condCount
              ? `if ${condCount} cond${condCount === 1 ? "" : "s"}`
              : `outcome ${i + 1}`,
            target: outcome.nextState,
          };
        }) ?? []
      );
    case "craftAction":
      if (!state.craftAction) return [];
      return [
        { label: "success", target: state.craftAction.successState },
        { label: "failure", target: state.craftAction.failureState },
        { label: "missing mats", target: state.craftAction.missingMaterialsState },
      ];
    case "listSelection":
      return (
        state.listSelection?.choices?.map((choice, i) => ({
          label: choice.text ? truncate(choice.text, 32) : `Choice ${i + 1}`,
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
      return state.dialogue?.text ? truncate(state.dialogue.text, 120) : "";
    case "genericAction": {
      const opCount = state.genericAction?.operations?.length ?? 0;
      const outcomeCount = state.genericAction?.outcomes?.length ?? 0;
      return `${opCount} op${opCount === 1 ? "" : "s"} · ${outcomeCount} outcome${outcomeCount === 1 ? "" : "s"}`;
    }
    case "craftAction":
      return state.craftAction ? `Craft item ${state.craftAction.itemId}` : "";
    case "listSelection":
      return state.listSelection?.title
        ? truncate(state.listSelection.title, 120)
        : "";
    case "askNumber":
      return state.askNumber?.text ? truncate(state.askNumber.text, 120) : "";
    case "askStyle":
      return state.askStyle?.text ? truncate(state.askStyle.text, 120) : "";
    default:
      return "";
  }
}

function layoutByBFS(
  states: ConversationState[],
  startId: string,
  transitions: Map<string, StateTransition[]>,
): Map<string, { level: number; indexInLevel: number }> {
  const levels = new Map<string, number>();
  const queue: string[] = [];
  const visited = new Set<string>();

  if (states.some(s => s.id === startId)) {
    levels.set(startId, 0);
    visited.add(startId);
    queue.push(startId);
  }

  while (queue.length > 0) {
    const id = queue.shift()!;
    const level = levels.get(id) ?? 0;
    const outs = transitions.get(id) ?? [];
    for (const t of outs) {
      if (!t.target) continue;
      if (visited.has(t.target)) continue;
      if (!states.some(s => s.id === t.target)) continue;
      visited.add(t.target);
      levels.set(t.target, level + 1);
      queue.push(t.target);
    }
  }

  const maxLevel = levels.size > 0 ? Math.max(...levels.values()) : 0;
  const orphanLevel = maxLevel + 1;
  for (const s of states) {
    if (!levels.has(s.id)) levels.set(s.id, orphanLevel);
  }

  const byLevel = new Map<number, string[]>();
  for (const [id, level] of levels) {
    const row = byLevel.get(level) ?? [];
    row.push(id);
    byLevel.set(level, row);
  }

  const result = new Map<string, { level: number; indexInLevel: number }>();
  for (const [level, ids] of byLevel) {
    ids.forEach((id, i) => {
      result.set(id, { level, indexInLevel: i });
    });
  }
  return result;
}

function StateNode({ data }: NodeProps<StateNodeData>) {
  const { stateId, stateType, description, isStart, isTerminal } = data;
  const borderClass = isStart
    ? "border-primary"
    : isTerminal
      ? "border-destructive/50"
      : "border-border";
  return (
    <div
      className={`rounded-md border bg-card shadow-sm ${borderClass}`}
      style={{ width: NODE_WIDTH }}
    >
      <Handle type="target" position={Position.Left} style={{ opacity: 0 }} />
      <div className="flex items-center gap-2 border-b px-3 py-2">
        <Badge variant="outline" className="text-[10px] px-1.5 py-0">
          {TYPE_LABEL[stateType]}
        </Badge>
        {isStart && (
          <Badge variant="default" className="text-[10px] px-1.5 py-0">
            start
          </Badge>
        )}
        {isTerminal && (
          <Badge variant="secondary" className="text-[10px] px-1.5 py-0">
            end
          </Badge>
        )}
        <span className="font-mono text-xs truncate" title={stateId}>
          {stateId}
        </span>
      </div>
      {description && (
        <p className="px-3 py-2 text-xs text-muted-foreground line-clamp-3">
          {description}
        </p>
      )}
      <Handle type="source" position={Position.Right} style={{ opacity: 0 }} />
    </div>
  );
}

const nodeTypes = { state: StateNode };

function buildGraph(conversation: Conversation): {
  nodes: Node<StateNodeData>[];
  edges: Edge[];
} {
  const states = conversation.attributes.states;
  const startId = conversation.attributes.startState;

  const transitions = new Map<string, StateTransition[]>();
  const hasOutgoing = new Set<string>();
  for (const s of states) {
    const outs = getStateTransitions(s);
    transitions.set(s.id, outs);
    if (outs.some(t => t.target)) hasOutgoing.add(s.id);
  }

  const positions = layoutByBFS(states, startId, transitions);

  const rowCount = new Map<number, number>();
  for (const { level } of positions.values()) {
    rowCount.set(level, (rowCount.get(level) ?? 0) + 1);
  }

  const nodes: Node<StateNodeData>[] = states.map(state => {
    const pos = positions.get(state.id) ?? { level: 0, indexInLevel: 0 };
    const rows = rowCount.get(pos.level) ?? 1;
    const columnHeight = rows * NODE_HEIGHT + (rows - 1) * NODE_GAP_Y;
    const yOffset = (NODE_HEIGHT + NODE_GAP_Y) * pos.indexInLevel - columnHeight / 2;
    return {
      id: state.id,
      type: "state",
      position: {
        x: pos.level * (NODE_WIDTH + LEVEL_GAP_X),
        y: yOffset,
      },
      data: {
        stateId: state.id,
        stateType: state.type,
        description: describeState(state),
        isStart: state.id === startId,
        isTerminal: !hasOutgoing.has(state.id),
      },
      draggable: false,
      selectable: false,
      connectable: false,
    };
  });

  const edges: Edge[] = [];
  for (const state of states) {
    const outs = transitions.get(state.id) ?? [];
    outs.forEach((t, i) => {
      if (!t.target) return;
      if (!states.some(s => s.id === t.target)) return;
      edges.push({
        id: `${state.id}-${i}-${t.target}`,
        source: state.id,
        target: t.target,
        label: t.label,
        labelStyle: { fontSize: 10, fill: "hsl(var(--foreground))" },
        labelBgStyle: { fill: "hsl(var(--background))" },
        labelBgPadding: [4, 2],
        labelBgBorderRadius: 4,
        style: { stroke: "hsl(var(--muted-foreground))", strokeWidth: 1 },
      });
    });
  }

  return { nodes, edges };
}

export function NpcConversationStateMachine({
  conversation,
  height = 480,
}: NpcConversationStateMachineProps) {
  const { nodes, edges } = useMemo(() => buildGraph(conversation), [conversation]);

  if (nodes.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        Conversation has no states to visualize.
      </p>
    );
  }

  return (
    <div
      className="w-full rounded-md border bg-background"
      style={{ height }}
    >
      <ReactFlowProvider>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          nodeTypes={nodeTypes}
          fitView
          fitViewOptions={{ padding: 0.2 }}
          nodesDraggable={false}
          nodesConnectable={false}
          elementsSelectable={false}
          panOnDrag
          zoomOnScroll
          proOptions={{ hideAttribution: true }}
        >
          <Background gap={16} />
          <Controls showInteractive={false} />
        </ReactFlow>
      </ReactFlowProvider>
    </div>
  );
}
