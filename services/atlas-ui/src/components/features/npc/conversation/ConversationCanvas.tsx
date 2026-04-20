import { useCallback, useEffect, useMemo, useState } from "react";
import ReactFlow, {
  Background,
  Controls,
  Handle,
  MarkerType,
  MiniMap,
  Position,
  ReactFlowProvider,
  useReactFlow,
  type Edge,
  type Node,
  type NodeMouseHandler,
  type NodeProps,
} from "reactflow";
import "reactflow/dist/style.css";
import {
  AlertTriangle,
  CornerUpLeft,
  Play,
  Share2,
  Square,
} from "lucide-react";
import type { Conversation } from "@/types/models/conversation";
import {
  allTransitions,
  buildStateIndex,
  type Transition,
} from "./transitions";
import { analyze, type GraphAnalysis } from "./graphAnalysis";
import { layoutGraph } from "./layout";
import { STATE_TYPE_META, describeState } from "./stateMeta";

interface ConversationCanvasProps {
  conversation: Conversation;
  selectedStateId: string | null;
  onSelect: (stateId: string) => void;
  showFullLoopEdges: boolean;
  height: number;
}

const NODE_WIDTH = 260;
const NODE_HEIGHT_BASE = 136;
const NODE_HEIGHT_PER_JUMP = 20;
const SHARED_TARGET_THRESHOLD = 3;
const DENSE_GRAPH_THRESHOLD = 60;

interface NodeData {
  conversation: Conversation;
  analysis: GraphAnalysis;
  selectedStateId: string | null;
  jumpBacks: Transition[];
  onSelect: (stateId: string) => void;
}

function StateNode({ id, data }: NodeProps<NodeData>) {
  const { conversation, analysis, selectedStateId, jumpBacks, onSelect } = data;
  const state = buildStateIndex(conversation).get(id);
  if (!state) return null;

  const meta = STATE_TYPE_META[state.type];
  const description = describeState(state);
  const isStart = conversation.attributes.startState === id;
  const isTerminal = analysis.terminals.has(id);
  const isSelected = selectedStateId === id;
  const inbound = analysis.inboundCount.get(id) ?? 0;
  const isShared = inbound >= SHARED_TARGET_THRESHOLD;
  const hasIssue =
    analysis.duplicateIds.includes(id) ||
    analysis.brokenRefs.some(r => r.source === id);
  const isUnreachable = !analysis.reachable.has(id) && !isStart;

  const ring = isSelected
    ? "ring-2 ring-primary shadow-md"
    : isUnreachable
      ? "ring-1 ring-destructive/50 hover:ring-destructive"
      : "ring-1 ring-border hover:ring-foreground/40";

  return (
    <div
      style={{ width: NODE_WIDTH }}
      className={`flex flex-col rounded-md bg-card text-left cursor-pointer ${ring} transition-shadow`}
    >
      <Handle type="target" position={Position.Left} style={{ opacity: 0 }} />
      <div className="flex items-center gap-1.5 border-b px-3 py-1.5">
        <span
          className={`text-[10px] px-1.5 py-[1px] rounded-sm border ${meta.accent}`}
        >
          {meta.label}
        </span>
        {state.type === "dialogue" && state.dialogue?.dialogueType && (
          <span className="text-[10px] px-1.5 py-[1px] rounded-sm border border-border/60 bg-muted/40 font-mono text-muted-foreground">
            {state.dialogue.dialogueType}
          </span>
        )}
        {isStart && (
          <Play
            className="h-2.5 w-2.5 fill-emerald-500 text-emerald-500 shrink-0"
            aria-label="start"
          />
        )}
        {isTerminal && (
          <Square
            className="h-2.5 w-2.5 fill-orange-500 text-orange-500 shrink-0"
            aria-label="terminal"
          />
        )}
        {isShared && (
          <Share2
            className="h-2.5 w-2.5 text-blue-500 shrink-0"
            aria-label={`shared target (${inbound} inbound)`}
          />
        )}
        {hasIssue && (
          <AlertTriangle
            className="h-2.5 w-2.5 text-destructive shrink-0"
            aria-label="validation issue"
          />
        )}
        <span className="font-mono text-[11px] truncate flex-1" title={id}>
          {id}
        </span>
      </div>
      {description && (
        <p className="px-3 py-1.5 text-[11px] text-muted-foreground line-clamp-4 whitespace-pre-wrap">
          {description}
        </p>
      )}
      {jumpBacks.length > 0 && (
        <div className="border-t px-3 py-1.5 flex flex-col gap-0.5 bg-muted/30">
          {jumpBacks.map((t, i) => (
            <button
              key={i}
              type="button"
              onClick={e => {
                e.stopPropagation();
                if (t.target) onSelect(t.target);
              }}
              className="flex items-center gap-1 text-[10px] text-foreground/80 hover:text-primary text-left"
              title={`Back to ${t.target}`}
            >
              <CornerUpLeft className="h-2.5 w-2.5 shrink-0 text-muted-foreground" />
              <span className="truncate">{t.label}</span>
              <span className="text-muted-foreground">→</span>
              <span className="font-mono truncate">{t.target}</span>
            </button>
          ))}
        </div>
      )}
      <Handle type="source" position={Position.Right} style={{ opacity: 0 }} />
    </div>
  );
}

const nodeTypes = { state: StateNode };

function CanvasInner({
  conversation,
  selectedStateId,
  onSelect,
  showFullLoopEdges,
}: Omit<ConversationCanvasProps, "height">) {
  const analysis = useMemo(() => analyze(conversation), [conversation]);
  const transitions = useMemo(() => allTransitions(conversation), [conversation]);

  const jumpBacksBySource = useMemo(() => {
    const m = new Map<string, Transition[]>();
    if (showFullLoopEdges) return m;
    for (const t of transitions) {
      if (!t.target) continue;
      if (analysis.backEdges.has(`${t.source}->${t.target}`)) {
        const list = m.get(t.source) ?? [];
        list.push(t);
        m.set(t.source, list);
      }
    }
    return m;
  }, [transitions, analysis, showFullLoopEdges]);

  const [positions, setPositions] = useState<Map<string, { x: number; y: number }>>(
    () => new Map(),
  );
  const [laying, setLaying] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLaying(true);
    const ids = conversation.attributes.states.map(s => s.id);
    const forwardEdges = transitions
      .filter(t => t.target && !analysis.backEdges.has(`${t.source}->${t.target}`))
      .map((t, i) => ({
        source: t.source,
        target: t.target!,
        id: `fwd${i}`,
      }));
    layoutGraph({
      nodeIds: ids,
      edges: forwardEdges,
      nodeWidth: NODE_WIDTH,
      nodeHeight: NODE_HEIGHT_BASE,
    })
      .then(result => {
        if (cancelled) return;
        setPositions(result.positions);
        setLaying(false);
      })
      .catch(err => {
        if (cancelled) return;
        console.error("ELK layout failed", err);
        setLaying(false);
      });
    return () => {
      cancelled = true;
    };
  }, [conversation, transitions, analysis]);

  const nodes = useMemo<Node<NodeData>[]>(() => {
    return conversation.attributes.states.map(state => {
      const pos = positions.get(state.id) ?? { x: 0, y: 0 };
      const jumpBacks = jumpBacksBySource.get(state.id) ?? [];
      const extraHeight = jumpBacks.length * NODE_HEIGHT_PER_JUMP;
      return {
        id: state.id,
        type: "state",
        position: pos,
        draggable: false,
        connectable: false,
        data: {
          conversation,
          analysis,
          selectedStateId,
          jumpBacks,
          onSelect,
        },
        style: { width: NODE_WIDTH, height: NODE_HEIGHT_BASE + extraHeight },
      };
    });
  }, [conversation, positions, analysis, selectedStateId, jumpBacksBySource, onSelect]);

  const edges = useMemo<Edge[]>(() => {
    const isDense =
      conversation.attributes.states.length > DENSE_GRAPH_THRESHOLD;
    const byId = buildStateIndex(conversation);
    const out: Edge[] = [];
    let i = 0;
    for (const t of transitions) {
      if (!t.target || !byId.has(t.target)) continue;
      const isBack = analysis.backEdges.has(`${t.source}->${t.target}`);
      if (isBack && !showFullLoopEdges) continue;
      const label = isDense ? undefined : t.label;
      out.push({
        id: `e${i++}-${t.source}-${t.target}`,
        source: t.source,
        target: t.target,
        ...(label !== undefined && { label }),
        ...(isBack && { type: "smoothstep" }),
        labelStyle: { fontSize: 10, fill: "hsl(var(--foreground))" },
        labelBgStyle: { fill: "hsl(var(--background))" },
        labelBgPadding: [4, 2],
        labelBgBorderRadius: 4,
        markerEnd: { type: MarkerType.ArrowClosed, width: 14, height: 14 },
        style: {
          stroke: "hsl(var(--muted-foreground))",
          strokeWidth: 1,
          opacity: isDense ? 0.35 : isBack ? 0.55 : 1,
          ...(isBack && { strokeDasharray: "4 3" }),
        },
      });
    }
    return out;
  }, [transitions, conversation, analysis, showFullLoopEdges]);

  const rf = useReactFlow();
  const handleNodeClick = useCallback<NodeMouseHandler>(
    (_event, node) => onSelect(node.id),
    [onSelect],
  );

  useEffect(() => {
    if (!selectedStateId) return;
    const pos = positions.get(selectedStateId);
    if (!pos) return;
    const t = setTimeout(() => {
      rf.setCenter(pos.x + NODE_WIDTH / 2, pos.y + NODE_HEIGHT_BASE / 2, {
        zoom: rf.getZoom(),
        duration: 300,
      });
    }, 40);
    return () => clearTimeout(t);
  }, [selectedStateId, positions, rf]);

  if (laying && positions.size === 0) {
    return (
      <div className="flex items-center justify-center h-full text-xs text-muted-foreground">
        Laying out…
      </div>
    );
  }

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      nodeTypes={nodeTypes}
      fitView
      fitViewOptions={{ padding: 0.2 }}
      minZoom={0.15}
      nodesDraggable={false}
      nodesConnectable={false}
      onNodeClick={handleNodeClick}
      panOnDrag
      zoomOnScroll
      proOptions={{ hideAttribution: true }}
    >
      <Background gap={16} />
      <Controls showInteractive={false} />
      <MiniMap
        pannable
        zoomable
        nodeStrokeWidth={3}
        nodeBorderRadius={4}
        style={{ background: "hsl(var(--background))" }}
      />
    </ReactFlow>
  );
}

export function ConversationCanvas({ height, ...props }: ConversationCanvasProps) {
  return (
    <div
      className="w-full rounded-md border bg-background"
      style={{ height }}
    >
      <ReactFlowProvider>
        <CanvasInner {...props} />
      </ReactFlowProvider>
    </div>
  );
}
