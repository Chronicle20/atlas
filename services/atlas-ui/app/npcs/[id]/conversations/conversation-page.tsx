"use client"

import { useTenant } from "@/context/tenant-context";
import { useEffect, useState, useCallback, useMemo, useRef } from "react";
import { useParams } from "next/navigation";
import { Conversation, fetchNPCConversations } from "@/lib/npc-conversations";
import ReactFlow, {
  Node,
  Edge,
  Background,
  useNodesState,
  useEdgesState,
  Panel,
  useReactFlow,
  ConnectionLineType,
  Handle,
  Position,
  NodeProps,
} from 'reactflow';
import 'reactflow/dist/style.css';
import { Button } from "@/components/ui/button";
import { RefreshCw, ZoomIn, ZoomOut, Info } from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle
} from "@/components/ui/card";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { toast } from "sonner";

// Node types and colors
const NODE_TYPES = {
  START: 'start',
  TERMINAL: 'terminal',
  NORMAL: 'normal',
};

// Utility function to convert camel case or snake case to pascal case with spaces
const camelToPascalWithSpaces = (str: string): string => {
  // Handle null or undefined input
  if (!str) return '';

  // First, replace underscores with spaces for snake_case
  const withSpaces = str.replace(/_/g, ' ');

  // Capitalize each word
  const pascalCase = withSpaces
    .split(' ')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');

  // Then add spaces before capital letters (for camelCase parts)
  return pascalCase.replace(/([A-Z])/g, ' $1').trim();
};

const NODE_COLORS = {
  [NODE_TYPES.START]: '#a3be8c', // Green
  [NODE_TYPES.TERMINAL]: '#bf616a', // Red
  [NODE_TYPES.NORMAL]: '#81a1c1', // Blue
};

// Custom node component with multiple source handles
const CustomNode = ({ data, isConnectable, ...props }: NodeProps) => {
  const { label, type, text, operations, outcomes, choices, outgoingEdgesCount = 0 } = data;

  // Create n+1 source handles for outgoing edges
  const sourceHandles = [];
  const handleCount = outgoingEdgesCount > 0 ? outgoingEdgesCount : 1;

  for (let i = 0; i < handleCount; i++) {
    // Calculate position along the right side of the node
    const handlePosition = (i + 1) / (handleCount + 1);

    sourceHandles.push(
      <Handle
        key={`source-${i}`}
        id={`source-${i}`}
        type="source"
        position={Position.Right}
        style={{
          top: `${handlePosition * 100}%`,
          background: '#555',
          width: 8,
          height: 8,
        }}
        isConnectable={isConnectable}
      />
    );
  }

  return (
    <div style={props.style} className="custom-node">
      {/* Target handle on the left side */}
      <Handle
        type="target"
        position={Position.Left}
        style={{ background: '#555', width: 8, height: 8 }}
        isConnectable={isConnectable}
      />

      <div className="p-2">
        <div className="font-bold">{label}</div>
        <div className="text-xs">{type}</div>
        {text && (
          <div className="text-xs mt-2 p-1 bg-black/10 rounded">
            {text}
          </div>
        )}

        {operations && operations.length > 0 && (
          <div className="text-xs mt-2">
            <div className="font-medium">Operations:</div>
            <div className="grid grid-cols-1 gap-2 mt-1">
              {operations.map((op, index) => (
                <div key={index} className="bg-black/10 rounded p-2 border border-white/20">
                  {op}
                </div>
              ))}
            </div>
          </div>
        )}

        {outcomes && outcomes.length > 0 && (
          <div className="text-xs mt-2">
            <div className="font-medium">Outcomes:</div>
            <div className="space-y-2 mt-1">
              {outcomes.map((outcome, index) => (
                <div key={index} className="bg-black/10 rounded p-2">
                  {outcome.conditions.length > 0 ? (
                    <div>
                      {outcome.conditions.map((condition, condIndex) => (
                        <div key={condIndex}>{condition}</div>
                      ))}
                    </div>
                  ) : (
                    <div className="italic">No conditions (default outcome)</div>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        {choices && choices.length > 0 && (
          <div className="text-xs mt-2">
            <div className="font-medium">Choices:</div>
            <div className="grid grid-cols-1 gap-2 mt-1">
              {choices.map((choice, index) => (
                <div key={index} className="bg-black/10 rounded p-2 border border-white/20">
                  {choice.text}
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Source handles on the right side */}
      {sourceHandles}
    </div>
  );
};

// Function to process conversation data into nodes and edges
const processConversationData = (conversation: Conversation) => {
  const nodes: Node[] = [];
  const edges: Edge[] = [];

  if (!conversation.attributes.states || conversation.attributes.states.length === 0) {
    return { nodes, edges };
  }

  // Track terminal states (states with no outgoing transitions)
  const statesWithOutgoingTransitions = new Set<string>();

  // First pass: identify states with outgoing transitions
  conversation.attributes.states.forEach(state => {
    if (state.type === 'dialogue' && state.dialogue?.choices) {
      state.dialogue.choices.forEach(choice => {
        if (choice.nextState) {
          statesWithOutgoingTransitions.add(state.id);
        }
      });
    } else if (state.type === 'genericAction' && state.genericAction?.outcomes) {
      state.genericAction.outcomes.forEach(outcome => {
        if (outcome.nextState) {
          statesWithOutgoingTransitions.add(state.id);
        }
      });
    } else if (state.type === 'craftAction' && state.craftAction) {
      if (state.craftAction.successState) statesWithOutgoingTransitions.add(state.id);
      if (state.craftAction.failureState) statesWithOutgoingTransitions.add(state.id);
      if (state.craftAction.missingMaterialsState) statesWithOutgoingTransitions.add(state.id);
    } else if (state.type === 'listSelection' && state.listSelection?.choices) {
      state.listSelection.choices.forEach(choice => {
        if (choice.nextState) {
          statesWithOutgoingTransitions.add(state.id);
        }
      });
    }
  });

  // Create nodes
  conversation.attributes.states.forEach((state, index) => {
    let nodeType = NODE_TYPES.NORMAL;

    // Start state
    if (state.id === conversation.attributes.startState) {
      nodeType = NODE_TYPES.START;
    } 
    // Terminal state (no outgoing transitions)
    else if (!statesWithOutgoingTransitions.has(state.id)) {
      nodeType = NODE_TYPES.TERMINAL;
    }

    // Count outgoing edges for this state to determine number of handles
    let outgoingEdgesCount = 0;
    if (state.type === 'dialogue' && state.dialogue?.choices) {
      outgoingEdgesCount = state.dialogue.choices.filter(choice => choice.nextState).length;
    } else if (state.type === 'genericAction' && state.genericAction?.outcomes) {
      outgoingEdgesCount = state.genericAction.outcomes.filter(outcome => outcome.nextState).length;
    } else if (state.type === 'craftAction' && state.craftAction) {
      if (state.craftAction.successState) outgoingEdgesCount++;
      if (state.craftAction.failureState) outgoingEdgesCount++;
      if (state.craftAction.missingMaterialsState) outgoingEdgesCount++;
    } else if (state.type === 'listSelection' && state.listSelection?.choices) {
      outgoingEdgesCount = state.listSelection.choices.filter(choice => choice.nextState).length;
    }

    // Prepare text content based on node type
    let textContent = null;
    if (state.type === 'dialogue' && state.dialogue?.text) {
      textContent = state.dialogue.text;
    } else if (state.type === 'listSelection' && state.listSelection?.title) {
      textContent = state.listSelection.title;
    }

    // Prepare operations and outcomes for genericAction nodes
    let operations = [];
    let outcomes = [];
    let choices = [];

    if (state.type === 'genericAction' && state.genericAction) {
      // Extract operation types and convert to pascal case with spaces
      if (state.genericAction.operations && state.genericAction.operations.length > 0) {
        operations = state.genericAction.operations.map(op => 
          camelToPascalWithSpaces(op.type)
        );
      }

      // Process outcomes with their conditions
      if (state.genericAction.outcomes && state.genericAction.outcomes.length > 0) {
        outcomes = state.genericAction.outcomes.map(outcome => {
          // Create an object with the nextState and a formatted conditions array
          return {
            nextState: outcome.nextState,
            conditions: outcome.conditions.map(condition => 
              `${camelToPascalWithSpaces(condition.type)} ${condition.operator} ${condition.value}`
            )
          };
        });
      }
    } else if (state.type === 'listSelection' && state.listSelection?.choices) {
      // Include choices for listSelection nodes
      choices = state.listSelection.choices.map(choice => ({
        text: choice.text,
        nextState: choice.nextState
      }));
    }

    nodes.push({
      id: state.id,
      type: 'customNode', // Use our custom node type
      data: { 
        label: camelToPascalWithSpaces(state.id),
        type: state.type,
        text: textContent,
        operations: operations,
        outcomes: outcomes,
        choices: choices,
        outgoingEdgesCount: outgoingEdgesCount, // Store count for handle creation
      },
      position: { x: 100 + (index % 3) * 200, y: 100 + Math.floor(index / 3) * 100 },
      style: {
        background: NODE_COLORS[nodeType],
        color: '#ffffff',
        border: '1px solid #ffffff',
        borderRadius: '8px',
        padding: '10px',
        width: 250,
      },
    });
  });

  // Create edges
  conversation.attributes.states.forEach(state => {
    // Track the current edge index for this state to assign source handles
    let currentEdgeIndex = 0;

    if (state.type === 'dialogue' && state.dialogue?.choices) {
      state.dialogue.choices.forEach((choice, index) => {
        if (choice.nextState) {
          // Calculate the source handle ID based on the current edge index
          const sourceHandleId = `source-${currentEdgeIndex}`;

          edges.push({
            id: `${state.id}-to-${choice.nextState}-${index}`,
            source: state.id,
            target: choice.nextState,
            sourceHandle: sourceHandleId,
            type: 'smoothstep',
            animated: false,
            style: { stroke: '#64748b' },
          });

          currentEdgeIndex++;
        }
      });
    } else if (state.type === 'genericAction' && state.genericAction?.outcomes) {
      state.genericAction.outcomes.forEach((outcome, index) => {
        if (outcome.nextState) {
          // Create a simplified condition summary
          const conditionSummary = `Outcome ${index + 1}`;
          // Calculate the source handle ID based on the current edge index
          const sourceHandleId = `source-${currentEdgeIndex}`;

          edges.push({
            id: `${state.id}-to-${outcome.nextState}-${index}`,
            source: state.id,
            target: outcome.nextState,
            sourceHandle: sourceHandleId,
            type: 'smoothstep',
            animated: false,
            style: { stroke: '#64748b' },
          });

          currentEdgeIndex++;
        }
      });
    } else if (state.type === 'craftAction' && state.craftAction) {
      if (state.craftAction.successState) {
        // Calculate the source handle ID based on the current edge index
        const sourceHandleId = `source-${currentEdgeIndex}`;

        edges.push({
          id: `${state.id}-to-${state.craftAction.successState}-success`,
          source: state.id,
          target: state.craftAction.successState,
          sourceHandle: sourceHandleId,
          type: 'smoothstep',
          animated: false,
          style: { stroke: '#22c55e' },
        });

        currentEdgeIndex++;
      }

      if (state.craftAction.failureState) {
        // Calculate the source handle ID based on the current edge index
        const sourceHandleId = `source-${currentEdgeIndex}`;

        edges.push({
          id: `${state.id}-to-${state.craftAction.failureState}-failure`,
          source: state.id,
          target: state.craftAction.failureState,
          sourceHandle: sourceHandleId,
          type: 'smoothstep',
          animated: false,
          style: { stroke: '#ef4444' },
        });

        currentEdgeIndex++;
      }

      if (state.craftAction.missingMaterialsState) {
        // Calculate the source handle ID based on the current edge index
        const sourceHandleId = `source-${currentEdgeIndex}`;

        edges.push({
          id: `${state.id}-to-${state.craftAction.missingMaterialsState}-missing`,
          source: state.id,
          target: state.craftAction.missingMaterialsState,
          sourceHandle: sourceHandleId,
          type: 'smoothstep',
          animated: false,
          style: { stroke: '#f59e0b' },
        });

        currentEdgeIndex++;
      }
    } else if (state.type === 'listSelection' && state.listSelection?.choices) {
      state.listSelection.choices.forEach((choice, index) => {
        if (choice.nextState) {
          // Calculate the source handle ID based on the current edge index
          const sourceHandleId = `source-${currentEdgeIndex}`;

          edges.push({
            id: `${state.id}-to-${choice.nextState}-${index}`,
            source: state.id,
            target: choice.nextState,
            sourceHandle: sourceHandleId,
            type: 'smoothstep',
            animated: false,
            style: { stroke: '#64748b' },
          });

          currentEdgeIndex++;
        }
      });
    }
  });

  return { nodes, edges };
};

export default function ConversationPage() {
  const { activeTenant } = useTenant();
  const params = useParams();
  const npcId = Number(params.id);

  const [, setConversation] = useState<Conversation | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [showLegend, setShowLegend] = useState(false);

  // Define node types
  const nodeTypes = useMemo(() => ({ customNode: CustomNode }), []);

  const reactFlowInstance = useReactFlow();

  const fetchConversationData = useCallback(async () => {
    if (!activeTenant) return;

    setLoading(true);
    setError(null);

    try {
      const conversationData = await fetchNPCConversations(activeTenant, npcId);

      if (!conversationData) {
        setError("No conversation found for this NPC.");
        setConversation(null);
        setNodes([]);
        setEdges([]);
        return;
      }

      setConversation(conversationData);

      // Process conversation data into nodes and edges
      const { nodes: processedNodes, edges: processedEdges } = processConversationData(conversationData);
      setNodes(processedNodes);
      setEdges(processedEdges);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch conversation data.");
      setConversation(null);
      setNodes([]);
      setEdges([]);
    } finally {
      setLoading(false);
    }
  }, [activeTenant, npcId, setNodes, setEdges]);

  useEffect(() => {
    fetchConversationData();
  }, [fetchConversationData]);

  const handleReorganize = useCallback(() => {
    if (!reactFlowInstance) return;

    // Create a horizontal tree layout
    // First, find the start node and build a tree structure
    const startNodeId = nodes.find(node => 
      node.style?.background === NODE_COLORS[NODE_TYPES.START]
    )?.id;

    if (!startNodeId) {
      toast.error("Could not find start node");
      return;
    }

    // Create a map of node IDs to their outgoing edges
    const nodeOutgoingEdges = new Map();
    edges.forEach(edge => {
      if (!nodeOutgoingEdges.has(edge.source)) {
        nodeOutgoingEdges.set(edge.source, []);
      }
      nodeOutgoingEdges.get(edge.source).push(edge);
    });

    // Create a map to track the level (depth) of each node
    const nodeLevels = new Map();
    nodeLevels.set(startNodeId, 0);

    // Breadth-first traversal to assign levels to nodes
    const queue = [startNodeId];
    const visited = new Set([startNodeId]);

    while (queue.length > 0) {
      const currentNodeId = queue.shift();
      const currentLevel = nodeLevels.get(currentNodeId);

      // Get all outgoing edges from this node
      const outgoingEdges = nodeOutgoingEdges.get(currentNodeId) || [];

      outgoingEdges.forEach(edge => {
        const targetNodeId = edge.target;

        if (!visited.has(targetNodeId)) {
          visited.add(targetNodeId);
          nodeLevels.set(targetNodeId, currentLevel + 1);
          queue.push(targetNodeId);
        }
      });
    }

    // Group nodes by level
    const nodesByLevel = new Map();
    nodeLevels.forEach((level, nodeId) => {
      if (!nodesByLevel.has(level)) {
        nodesByLevel.set(level, []);
      }
      nodesByLevel.get(level).push(nodeId);
    });

    // Position nodes based on their level
    const horizontalSpacing = 300; // Space between levels
    const verticalSpacing = 150;   // Minimum space between nodes in the same level
    const safetyMargin = 75;       // Additional safety margin to prevent overlapping

    // Estimate node heights based on content
    const nodeHeights = new Map();
    nodes.forEach(node => {
      // Base height for all nodes
      let height = 150; // Increased base height

      // Add height for text content (no max limit since we want to show all text)
      if (node.data.text) {
        // Estimate 30px per line of text
        const textLines = node.data.text.split('\n').length;
        height += textLines * 30;
      }

      // Add height for operations (no max limit since we want to show all operations)
      if (node.data.operations && node.data.operations.length > 0) {
        height += 30 + node.data.operations.length * 30;
      }

      // Add height for outcomes (no max limit since we want to show all outcomes)
      if (node.data.outcomes && node.data.outcomes.length > 0) {
        height += 30 + node.data.outcomes.length * 40;
      }

      // Add height for choices (no max limit since we want to show all choices)
      if (node.data.choices && node.data.choices.length > 0) {
        height += 30 + node.data.choices.length * 40; // 30px for the header, 40px per choice
      }

      // Add safety margin to all height calculations
      height += safetyMargin;

      nodeHeights.set(node.id, height);
    });

    // Position nodes with proper vertical spacing
    const nodePositions = [];

    // Process each level
    for (let level = 0; level <= Math.max(...Array.from(nodeLevels.values())); level++) {
      const nodesInLevel = nodesByLevel.get(level) || [];
      let currentY = 100; // Starting Y position for this level

      // Position each node in this level
      for (let i = 0; i < nodesInLevel.length; i++) {
        const nodeId = nodesInLevel[i];
        const node = nodes.find(n => n.id === nodeId);

        if (node) {
          const x = level * horizontalSpacing + 100;

          // Create a new node with updated position
          nodePositions.push({
            ...node,
            position: { x, y: currentY }
          });

          // Update Y position for the next node, adding the current node's height plus spacing
          currentY += (nodeHeights.get(nodeId) || 250) + verticalSpacing;
        }
      }
    }

    // Add any nodes that weren't visited in our traversal at the end
    nodes.forEach(node => {
      if (!nodePositions.some(n => n.id === node.id)) {
        // Use the estimated height for this node, or a default of 250 (increased from 150)
        const nodeHeight = nodeHeights.get(node.id) || 250;
        nodePositions.push({
          ...node,
          position: { x: 100, y: 100 + nodePositions.length * (nodeHeight + verticalSpacing) }
        });
      }
    });

    setNodes(nodePositions);

    // Fit the view to show all nodes
    setTimeout(() => {
      reactFlowInstance.fitView({ padding: 0.2 });
    }, 50);

    toast.success("Graph reorganized horizontally");
  }, [nodes, edges, reactFlowInstance, setNodes]);

  // Auto-organize on initial load - using a ref to track if we've already organized
  const hasOrganized = useRef(false);
  useEffect(() => {
    if (nodes.length > 0 && !loading && !hasOrganized.current) {
      hasOrganized.current = true;
      handleReorganize();
    }
  }, [loading, handleReorganize]);

  const handleZoomIn = () => {
    reactFlowInstance.zoomIn();
  };

  const handleZoomOut = () => {
    reactFlowInstance.zoomOut();
  };

  const handleFitView = () => {
    reactFlowInstance.fitView({ padding: 0.2 });
  };

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center h-[calc(100vh-10rem)] p-10">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary"></div>
        <p className="mt-4 text-lg">Loading conversation data...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col space-y-6 p-10 pb-16">
        <h2 className="text-2xl font-bold tracking-tight">NPC #{npcId} Conversation</h2>
        <Alert variant="destructive">
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      </div>
    );
  }

  return (
    <div className="flex flex-col space-y-6 p-10 pb-16 h-[calc(100vh-4rem)]">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">NPC #{npcId} Conversation</h2>
        </div>
        <div className="flex space-x-2">
          <Button variant="outline" size="sm" onClick={() => setShowLegend(!showLegend)}>
            <Info className="h-4 w-4 mr-2" />
            {showLegend ? 'Hide Legend' : 'Show Legend'}
          </Button>
          <Button variant="outline" size="sm" onClick={fetchConversationData}>
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
        </div>
      </div>

      {showLegend && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle>Legend</CardTitle>
            <CardDescription>Understanding the conversation graph</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <h3 className="font-semibold mb-2">Node Colors</h3>
                <div className="space-y-2">
                  <div className="flex items-center">
                    <div className="w-4 h-4 rounded mr-2" style={{ backgroundColor: NODE_COLORS[NODE_TYPES.START] }}></div>
                    <span>Start State</span>
                  </div>
                  <div className="flex items-center">
                    <div className="w-4 h-4 rounded mr-2" style={{ backgroundColor: NODE_COLORS[NODE_TYPES.TERMINAL] }}></div>
                    <span>Terminal State (no outgoing transitions)</span>
                  </div>
                  <div className="flex items-center">
                    <div className="w-4 h-4 rounded mr-2" style={{ backgroundColor: NODE_COLORS[NODE_TYPES.NORMAL] }}></div>
                    <span>Normal State</span>
                  </div>
                </div>
              </div>

              <div>
                <h3 className="font-semibold mb-2">State Types</h3>
                <div className="space-y-2">
                  <Badge variant="outline">dialogue</Badge>
                  <Badge variant="outline">genericAction</Badge>
                  <Badge variant="outline">craftAction</Badge>
                  <Badge variant="outline">listSelection</Badge>
                </div>
              </div>

            </div>
          </CardContent>
        </Card>
      )}

      <div className="flex-1 border rounded-md bg-background">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          nodeTypes={nodeTypes}
          fitView
          proOptions={{ hideAttribution: true }}
          connectionLineType={ConnectionLineType.SmoothStep}
        >
          <Background />
          <Panel position="top-right" className="bg-background border rounded-md p-2 flex flex-col space-y-2">
            <Button variant="outline" size="sm" onClick={handleZoomIn}>
              <ZoomIn className="h-4 w-4" />
            </Button>
            <Button variant="outline" size="sm" onClick={handleZoomOut}>
              <ZoomOut className="h-4 w-4" />
            </Button>
            <Separator className="my-1" />
            <Button variant="outline" size="sm" onClick={handleFitView}>
              Fit View
            </Button>
            <Button variant="default" size="sm" onClick={handleReorganize}>
              Reorganize
            </Button>
          </Panel>
        </ReactFlow>
      </div>
    </div>
  );
}
