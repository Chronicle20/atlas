import ELK from "elkjs/lib/elk.bundled.js";

export interface LayoutInput {
  nodeIds: string[];
  edges: Array<{ source: string; target: string; id?: string }>;
  nodeWidth: number;
  nodeHeight: number;
}

export interface LayoutResult {
  positions: Map<string, { x: number; y: number }>;
  width: number;
  height: number;
}

const elk = new ELK();

const DEFAULT_OPTIONS: Record<string, string> = {
  "elk.algorithm": "layered",
  "elk.direction": "RIGHT",
  "elk.layered.layering.strategy": "NETWORK_SIMPLEX",
  "elk.layered.nodePlacement.strategy": "BRANDES_KOEPF",
  "elk.spacing.nodeNode": "48",
  "elk.layered.spacing.nodeNodeBetweenLayers": "200",
  "elk.layered.spacing.edgeNodeBetweenLayers": "40",
  "elk.layered.spacing.edgeEdgeBetweenLayers": "24",
  "elk.layered.crossingMinimization.strategy": "LAYER_SWEEP",
  "elk.layered.cycleBreaking.strategy": "GREEDY",
  "elk.layered.considerModelOrder.strategy": "NODES_AND_EDGES",
};

export async function layoutGraph(input: LayoutInput): Promise<LayoutResult> {
  if (input.nodeIds.length === 0) {
    return { positions: new Map(), width: 0, height: 0 };
  }

  const graph = {
    id: "root",
    layoutOptions: DEFAULT_OPTIONS,
    children: input.nodeIds.map(id => ({
      id,
      width: input.nodeWidth,
      height: input.nodeHeight,
    })),
    edges: input.edges.map((e, i) => ({
      id: e.id ?? `e${i}`,
      sources: [e.source],
      targets: [e.target],
    })),
  };

  const laidOut = (await elk.layout(graph)) as {
    children?: Array<{ id: string; x?: number; y?: number }>;
    width?: number;
    height?: number;
  };
  const positions = new Map<string, { x: number; y: number }>();
  for (const child of laidOut.children ?? []) {
    positions.set(child.id, { x: child.x ?? 0, y: child.y ?? 0 });
  }
  return {
    positions,
    width: laidOut.width ?? 0,
    height: laidOut.height ?? 0,
  };
}
