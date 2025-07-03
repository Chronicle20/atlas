"use client"

import { useTenant } from "@/context/tenant-context";
import { useEffect, useState, useCallback, useMemo, useRef, SetStateAction} from "react";
import {useParams} from "next/navigation";
import {Conversation, fetchNPCConversations} from "@/lib/npc-conversations";
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
import {Button} from "@/components/ui/button";
import {RefreshCw, ZoomIn, ZoomOut, Info, Edit} from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle
} from "@/components/ui/card";
import {Alert, AlertDescription, AlertTitle} from "@/components/ui/alert";
import {Badge} from "@/components/ui/badge";
import {Separator} from "@/components/ui/separator";
import {toast} from "sonner";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";

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
// Define types for the node data
interface NodeData {
  label: string;
  type: string;
  text?: string;
  operations?: string[];
  outcomes?: { nextState: string; conditions: string[] }[];
  choices?: { text: string; nextState: string | null }[];
  craftAction?: {
    successState?: string;
    failureState?: string;
    missingMaterialsState?: string;
  };
  id: string; // Add id to NodeData
}

// Extend NodeProps to include style
interface CustomNodeProps extends NodeProps {
  style?: React.CSSProperties;
  onNodeEdit?: (nodeId: string) => void; // Add callback for node editing
}

const CustomNode = ({data, isConnectable, onNodeEdit, ...props}: CustomNodeProps) => {
  const {label, type, text, operations, outcomes, choices, craftAction, id} = data as NodeData;

  return (
      <div style={props.style} className="custom-node">
        {/* Target handle on the left side */}
        <Handle
            type="target"
            position={Position.Left}
            style={{background: '#555', width: 8, height: 8}}
            isConnectable={isConnectable}
        />

        <div className="p-2">
          <div className="flex justify-between items-center">
            <div className="font-bold">{label}</div>
            <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => onNodeEdit && onNodeEdit(id)}>
              <Edit className="h-4 w-4" />
            </Button>
          </div>
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
                  {operations.map((op: string, index: number) => (
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
                  {outcomes.map((outcome: { nextState: string; conditions: string[] }, index: number) => (
                      <div key={index} className="bg-black/10 rounded p-2 relative">
                        {outcome.conditions.length > 0 ? (
                            <div>
                              {outcome.conditions.map((condition: string, condIndex: number) => (
                                  <div key={condIndex}>{condition}</div>
                              ))}
                            </div>
                        ) : (
                            <div className="italic">No conditions (default outcome)</div>
                        )}
                        {/* Source handle for this outcome */}
                        <Handle
                          id={`outcome-${index}`}
                          type="source"
                          position={Position.Right}
                          style={{
                            background: '#555',
                            width: 8,
                            height: 8,
                            right: -24,
                            top: '50%',
                            transform: 'translateY(-50%)'
                          }}
                          isConnectable={isConnectable}
                        />
                      </div>
                  ))}
                </div>
              </div>
          )}

          {/* CraftAction specific UI with handles */}
          {type === 'craftAction' && craftAction && (
              <div className="text-xs mt-2">
                <div className="font-medium">Craft Action:</div>
                <div className="space-y-2 mt-1">
                  <div className="bg-black/10 rounded p-2 border border-white/20 relative">
                    <div className="flex items-center">
                      <div className="w-3 h-3 rounded-full bg-green-500 mr-2"></div>
                      <span>Success: {craftAction.successState || 'Not set'}</span>
                    </div>
                    <Handle
                      id="success"
                      type="source"
                      position={Position.Right}
                      style={{
                        background: '#22c55e',
                        width: 8,
                        height: 8,
                        right: -24,
                        top: '50%',
                        transform: 'translateY(-50%)'
                      }}
                      isConnectable={isConnectable}
                    />
                  </div>

                  <div className="bg-black/10 rounded p-2 border border-white/20 relative">
                    <div className="flex items-center">
                      <div className="w-3 h-3 rounded-full bg-red-500 mr-2"></div>
                      <span>Failure: {craftAction.failureState || 'Not set'}</span>
                    </div>
                    <Handle
                      id="failure"
                      type="source"
                      position={Position.Right}
                      style={{
                        background: '#ef4444',
                        width: 8,
                        height: 8,
                        right: -24,
                        top: '50%',
                        transform: 'translateY(-50%)'
                      }}
                      isConnectable={isConnectable}
                    />
                  </div>

                  <div className="bg-black/10 rounded p-2 border border-white/20 relative">
                    <div className="flex items-center">
                      <div className="w-3 h-3 rounded-full bg-amber-500 mr-2"></div>
                      <span>Missing Materials: {craftAction.missingMaterialsState || 'Not set'}</span>
                    </div>
                    <Handle
                      id="missing"
                      type="source"
                      position={Position.Right}
                      style={{
                        background: '#f59e0b',
                        width: 8,
                        height: 8,
                        right: -24,
                        top: '50%',
                        transform: 'translateY(-50%)'
                      }}
                      isConnectable={isConnectable}
                    />
                  </div>
                </div>
              </div>
          )}

          {choices && choices.length > 0 && (
              <div className="text-xs mt-2">
                <div className="font-medium">Choices:</div>
                <div className="grid grid-cols-1 gap-2 mt-1">
                  {choices.map((choice: { text: string; nextState: string | null }, index: number) => (
                      <div key={index} className="bg-black/10 rounded p-2 border border-white/20 relative">
                        {choice.text}
                        {/* Source handle for this choice */}
                        <Handle
                          id={`choice-${index}`}
                          type="source"
                          position={Position.Right}
                          style={{
                            background: '#555',
                            width: 8,
                            height: 8,
                            right: -24,
                            top: '50%',
                            transform: 'translateY(-50%)'
                          }}
                          isConnectable={isConnectable}
                        />
                      </div>
                  ))}
                </div>
              </div>
          )}
        </div>
      </div>
  );
};

// Function to process conversation data into nodes and edges
const processConversationData = (conversation: Conversation) => {
  const nodes: Node[] = [];
  const edges: Edge[] = [];

  if (!conversation.attributes.states || conversation.attributes.states.length === 0) {
    return {nodes, edges};
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
    let operations: string[] = [];
    let outcomes: { nextState: string; conditions: string[]; }[] = [];
    let choices: { text: string; nextState: string | null; }[] = [];
    let craftActionData = null;

    if (state.type === 'genericAction' && state.genericAction) {
      // Extract operation types and convert to pascal case with spaces
      if (state.genericAction.operations && state.genericAction.operations.length > 0) {
        operations = state.genericAction.operations.map((op: { type: string }) =>
            camelToPascalWithSpaces(op.type)
        );
      }

      // Process outcomes with their conditions
      if (state.genericAction.outcomes && state.genericAction.outcomes.length > 0) {
        outcomes = state.genericAction.outcomes.map((outcome: { nextState: string; conditions: { type: string; operator: string; value: string }[] }) => {
          // Create an object with the nextState and a formatted conditions array
          return {
            nextState: outcome.nextState,
            conditions: outcome.conditions.map((condition: { type: string; operator: string; value: string }) =>
                `${camelToPascalWithSpaces(condition.type)} ${condition.operator} ${condition.value}`
            )
          };
        });
      }
    } else if (state.type === 'dialogue' && state.dialogue?.choices) {
      // Include choices for dialogue nodes
      choices = state.dialogue.choices.map((choice: { text: string; nextState: string | null }) => ({
        text: choice.text,
        nextState: choice.nextState
      }));
    } else if (state.type === 'listSelection' && state.listSelection?.choices) {
      // Include choices for listSelection nodes
      choices = state.listSelection.choices.map((choice: { text: string; nextState: string | null }) => ({
        text: choice.text,
        nextState: choice.nextState
      }));
    } else if (state.type === 'craftAction' && state.craftAction) {
      // Include craftAction data
      craftActionData = {
        successState: state.craftAction.successState,
        failureState: state.craftAction.failureState,
        missingMaterialsState: state.craftAction.missingMaterialsState
      };
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
        craftAction: craftActionData,
        id: state.id, // Add the node ID to the data
      },
      position: {x: 100 + (index % 3) * 200, y: 100 + Math.floor(index / 3) * 100},
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
    if (state.type === 'dialogue' && state.dialogue?.choices) {
      state.dialogue.choices.forEach((choice: { nextState: string | null }, index: number) => {
        if (choice.nextState) {
          // Use the choice-specific handle ID
          const sourceHandleId = `choice-${index}`;

          edges.push({
            id: `${state.id}-to-${choice.nextState}-${index}`,
            source: state.id,
            target: choice.nextState,
            sourceHandle: sourceHandleId,
            type: 'smoothstep',
            animated: false,
            style: {stroke: '#64748b'},
          });
        }
      });
    } else if (state.type === 'genericAction' && state.genericAction?.outcomes) {
      state.genericAction.outcomes.forEach((outcome: { nextState: string | null; conditions: { type: string; operator: string; value: string }[] }, index: number) => {
        if (outcome.nextState) {
          // Use the outcome-specific handle ID
          const sourceHandleId = `outcome-${index}`;

          edges.push({
            id: `${state.id}-to-${outcome.nextState}-${index}`,
            source: state.id,
            target: outcome.nextState,
            sourceHandle: sourceHandleId,
            type: 'smoothstep',
            animated: false,
            style: {stroke: '#64748b'},
          });
        }
      });
    } else if (state.type === 'craftAction' && state.craftAction) {
      // Use the specific handle IDs for craftAction outcomes
      if (state.craftAction.successState) {
        edges.push({
          id: `${state.id}-to-${state.craftAction.successState}-success`,
          source: state.id,
          target: state.craftAction.successState,
          sourceHandle: 'success',
          type: 'smoothstep',
          animated: false,
          style: {stroke: '#22c55e'},
        });
      }

      if (state.craftAction.failureState) {
        edges.push({
          id: `${state.id}-to-${state.craftAction.failureState}-failure`,
          source: state.id,
          target: state.craftAction.failureState,
          sourceHandle: 'failure',
          type: 'smoothstep',
          animated: false,
          style: {stroke: '#ef4444'},
        });
      }

      if (state.craftAction.missingMaterialsState) {
        edges.push({
          id: `${state.id}-to-${state.craftAction.missingMaterialsState}-missing`,
          source: state.id,
          target: state.craftAction.missingMaterialsState,
          sourceHandle: 'missing',
          type: 'smoothstep',
          animated: false,
          style: {stroke: '#f59e0b'},
        });
      }
    } else if (state.type === 'listSelection' && state.listSelection?.choices) {
      state.listSelection.choices.forEach((choice: { text: string; nextState: string | null }, index: number) => {
        if (choice.nextState) {
          // Use the choice-specific handle ID
          const sourceHandleId = `choice-${index}`;

          edges.push({
            id: `${state.id}-to-${choice.nextState}-${index}`,
            source: state.id,
            target: choice.nextState,
            sourceHandle: sourceHandleId,
            type: 'smoothstep',
            animated: false,
            style: {stroke: '#64748b'},
          });
        }
      });
    }
  });

  return {nodes, edges};
};

export default function ConversationPage() {
  const {activeTenant} = useTenant();
  const params = useParams();
  const npcId = Number(params.id);

  const [conversation, setConversation] = useState<Conversation | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [showLegend, setShowLegend] = useState(false);

  // State for node editing dialog
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const selectedNode = useMemo(() => {
    if (!selectedNodeId) return null;
    return nodes.find(node => node.id === selectedNodeId);
  }, [selectedNodeId, nodes]);

  // State from the conversation
  const selectedNodeState = useMemo(() => {
    if (!selectedNodeId || !conversation) return null;
    return conversation.attributes.states.find(state => state.id === selectedNodeId);
  }, [selectedNodeId, conversation]);

  // State for editing node
  const [editNodeId, setEditNodeId] = useState<string>('');
  const [editNodeType, setEditNodeType] = useState<string>('');
  const [editNodeText, setEditNodeText] = useState<string>('');
  const [editNodeTitle, setEditNodeTitle] = useState<string>('');

  // Handle node type change
  const handleNodeTypeChange = useCallback((newType: string) => {
    setEditNodeType(newType);

    // Clear fields that are not relevant to the new node type
    if (newType !== 'dialogue') {
      setEditNodeText('');
    }

    if (newType !== 'listSelection') {
      setEditNodeTitle('');
    }
  }, []);

  // Handle save changes
  const handleSaveChanges = useCallback(() => {
    if (!conversation || !selectedNodeId) return;

    // Create a copy of the conversation
    const updatedConversation = { ...conversation };

    // Find the node state in the conversation
    const nodeIndex = updatedConversation.attributes.states.findIndex(state => state.id === selectedNodeId);

    if (nodeIndex === -1) return;

    // Get the current node state
    const currentState = { ...updatedConversation.attributes.states[nodeIndex] };

    // Check if the node ID has been changed
    const isNodeIdChanged = currentState.id !== editNodeId;
    const oldNodeId = currentState.id;

    // Create updated state with new values
    const updatedState: ConversationState = {
      id: editNodeId,
      type: editNodeType as "dialogue" | "genericAction" | "craftAction" | "listSelection",
    };

    // Add type-specific properties
    if (updatedState.type === 'dialogue') {
      updatedState.dialogue = {
        ...(currentState.dialogue || { dialogueType: "sendOk", choices: [] }),
        text: editNodeText,
      };
    } else if (updatedState.type === 'listSelection') {
      updatedState.listSelection = {
        ...(currentState.listSelection || { choices: [] }),
        title: editNodeTitle,
      };
    } else if (updatedState.type === 'genericAction') {
      updatedState.genericAction = currentState.genericAction || { operations: [], outcomes: [] };
    } else if (updatedState.type === 'craftAction') {
      updatedState.craftAction = currentState.craftAction || { 
        recipeId: 0, 
        quantity: 1, 
        successState: '', 
        failureState: '', 
        missingMaterialsState: '' 
      };
    }

    // Update the state in the conversation
    updatedConversation.attributes.states[nodeIndex] = updatedState;

    // If the node ID has been changed, update all references to it
    if (isNodeIdChanged) {
      // Update startState reference if needed
      if (updatedConversation.attributes.startState === oldNodeId) {
        updatedConversation.attributes.startState = editNodeId;
      }

      // Update references in all states
      updatedConversation.attributes.states.forEach(state => {
        // Update dialogue choices
        if (state.type === 'dialogue' && state.dialogue?.choices) {
          state.dialogue.choices.forEach(choice => {
            if (choice.nextState === oldNodeId) {
              choice.nextState = editNodeId;
            }
          });
        }

        // Update genericAction outcomes
        if (state.type === 'genericAction' && state.genericAction?.outcomes) {
          state.genericAction.outcomes.forEach(outcome => {
            if (outcome.nextState === oldNodeId) {
              outcome.nextState = editNodeId;
            }
          });
        }

        // Update craftAction state transitions
        if (state.type === 'craftAction' && state.craftAction) {
          if (state.craftAction.successState === oldNodeId) {
            state.craftAction.successState = editNodeId;
          }
          if (state.craftAction.failureState === oldNodeId) {
            state.craftAction.failureState = editNodeId;
          }
          if (state.craftAction.missingMaterialsState === oldNodeId) {
            state.craftAction.missingMaterialsState = editNodeId;
          }
        }

        // Update listSelection choices
        if (state.type === 'listSelection' && state.listSelection?.choices) {
          state.listSelection.choices.forEach(choice => {
            if (choice.nextState === oldNodeId) {
              choice.nextState = editNodeId;
            }
          });
        }
      });
    }

    // Update the conversation in state
    setConversation(updatedConversation);

    // Process the updated conversation to update nodes and edges
    const { nodes: processedNodes, edges: processedEdges } = processConversationData(updatedConversation);
    setNodes(processedNodes);
    setEdges(processedEdges);

    // Close the dialog
    setIsDialogOpen(false);

    // Show success message
    toast.success("Node updated successfully");
  }, [conversation, selectedNodeId, editNodeId, editNodeType, editNodeText, editNodeTitle, setNodes, setEdges]);

  // Handle node edit
  const handleNodeEdit = useCallback((nodeId: string) => {
    setSelectedNodeId(nodeId);

    // Find the node state in the conversation
    const nodeState = conversation?.attributes.states.find(state => state.id === nodeId);

    if (nodeState) {
      // Initialize edit state with current values
      setEditNodeId(nodeState.id);
      setEditNodeType(nodeState.type);

      // Set text for dialogue nodes
      if (nodeState.type === 'dialogue' && nodeState.dialogue?.text) {
        setEditNodeText(nodeState.dialogue.text);
      } else {
        setEditNodeText('');
      }

      // Set title for listSelection nodes
      if (nodeState.type === 'listSelection' && nodeState.listSelection?.title) {
        setEditNodeTitle(nodeState.listSelection.title);
      } else {
        setEditNodeTitle('');
      }
    }

    setIsDialogOpen(true);
  }, [conversation]);

  // Define node types with the edit handler
  const nodeTypes = useMemo(() => ({
    customNode: (props: CustomNodeProps) => (
      <CustomNode {...props} onNodeEdit={handleNodeEdit} />
    )
  }), [handleNodeEdit]);

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
      const {nodes: processedNodes, edges: processedEdges} = processConversationData(conversationData);
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

      outgoingEdges.forEach((edge: { target: string; source: string; id: string }) => {
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
    const nodePositions: SetStateAction<Node[]> = [];

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
      {/* Node Edit Dialog */}
      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent className="max-w-3xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>
              {selectedNode ? `Edit Node: ${selectedNode.data.label}` : 'Node Details'}
            </DialogTitle>
          </DialogHeader>

          {selectedNodeState && (
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <h3 className="text-lg font-semibold">Node ID</h3>
                  <input 
                    type="text" 
                    value={editNodeId} 
                    onChange={(e) => setEditNodeId(e.target.value)}
                    className="w-full p-2 border rounded-md"
                  />
                </div>
                <div>
                  <h3 className="text-lg font-semibold">Node Type</h3>
                  <select 
                    value={editNodeType} 
                    onChange={(e) => handleNodeTypeChange(e.target.value)}
                    className="w-full p-2 border rounded-md"
                  >
                    <option value="dialogue">dialogue</option>
                    <option value="genericAction">genericAction</option>
                    <option value="craftAction">craftAction</option>
                    <option value="listSelection">listSelection</option>
                  </select>
                </div>
              </div>

              {editNodeType === 'dialogue' && (
                <div>
                  <h3 className="text-lg font-semibold">Dialogue</h3>
                  <div className="border p-4 rounded-md">
                    <p className="font-medium">Text:</p>
                    <textarea 
                      value={editNodeText} 
                      onChange={(e) => setEditNodeText(e.target.value)}
                      className="w-full p-2 border rounded-md min-h-[100px]"
                    />

                    {selectedNodeState.type === 'dialogue' && selectedNodeState.dialogue?.choices && selectedNodeState.dialogue.choices.length > 0 && (
                      <div className="mt-4">
                        <p className="font-medium">Choices:</p>
                        <div className="space-y-2 mt-2">
                          {selectedNodeState.dialogue.choices.map((choice, index) => (
                            <div key={index} className="border p-2 rounded-md">
                              <p><span className="font-medium">Text:</span> {choice.text}</p>
                              <p><span className="font-medium">Next State:</span> {choice.nextState || 'None'}</p>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              )}

              {editNodeType === 'genericAction' && (
                <div>
                  <h3 className="text-lg font-semibold">Generic Action</h3>
                  <div className="border p-4 rounded-md">
                    {selectedNodeState.type === 'genericAction' && selectedNodeState.genericAction?.operations && selectedNodeState.genericAction.operations.length > 0 && (
                      <div>
                        <p className="font-medium">Operations:</p>
                        <div className="space-y-2 mt-2">
                          {selectedNodeState.genericAction.operations.map((op, index) => (
                            <div key={index} className="border p-2 rounded-md">
                              <p><span className="font-medium">Type:</span> {op.type}</p>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}

                    {selectedNodeState.type === 'genericAction' && selectedNodeState.genericAction?.outcomes && selectedNodeState.genericAction.outcomes.length > 0 && (
                      <div className="mt-4">
                        <p className="font-medium">Outcomes:</p>
                        <div className="space-y-2 mt-2">
                          {selectedNodeState.genericAction.outcomes.map((outcome, index) => (
                            <div key={index} className="border p-2 rounded-md">
                              <p><span className="font-medium">Next State:</span> {outcome.nextState}</p>
                              {outcome.conditions && outcome.conditions.length > 0 && (
                                <div>
                                  <p className="font-medium mt-2">Conditions:</p>
                                  <div className="space-y-1 mt-1">
                                    {outcome.conditions.map((condition, condIndex) => (
                                      <p key={condIndex}>{condition.type} {condition.operator} {condition.value}</p>
                                    ))}
                                  </div>
                                </div>
                              )}
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              )}

              {editNodeType === 'craftAction' && (
                <div>
                  <h3 className="text-lg font-semibold">Craft Action</h3>
                  <div className="border p-4 rounded-md">
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <p className="font-medium">Recipe ID:</p>
                        <p>{selectedNodeState.type === 'craftAction' && selectedNodeState.craftAction?.recipeId}</p>
                      </div>
                      <div>
                        <p className="font-medium">Quantity:</p>
                        <p>{selectedNodeState.type === 'craftAction' && selectedNodeState.craftAction?.quantity}</p>
                      </div>
                    </div>

                    <div className="grid grid-cols-3 gap-4 mt-4">
                      {selectedNodeState.type === 'craftAction' && selectedNodeState.craftAction?.successState && (
                        <div>
                          <p className="font-medium">Success State:</p>
                          <p>{selectedNodeState.craftAction.successState}</p>
                        </div>
                      )}

                      {selectedNodeState.type === 'craftAction' && selectedNodeState.craftAction?.failureState && (
                        <div>
                          <p className="font-medium">Failure State:</p>
                          <p>{selectedNodeState.craftAction.failureState}</p>
                        </div>
                      )}

                      {selectedNodeState.type === 'craftAction' && selectedNodeState.craftAction?.missingMaterialsState && (
                        <div>
                          <p className="font-medium">Missing Materials State:</p>
                          <p>{selectedNodeState.craftAction.missingMaterialsState}</p>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              )}

              {editNodeType === 'listSelection' && (
                <div>
                  <h3 className="text-lg font-semibold">List Selection</h3>
                  <div className="border p-4 rounded-md">
                    <div>
                      <p className="font-medium">Title:</p>
                      <input 
                        type="text" 
                        value={editNodeTitle} 
                        onChange={(e) => setEditNodeTitle(e.target.value)}
                        className="w-full p-2 border rounded-md"
                      />
                    </div>

                    {selectedNodeState.type === 'listSelection' && selectedNodeState.listSelection?.choices && selectedNodeState.listSelection.choices.length > 0 && (
                      <div className="mt-4">
                        <p className="font-medium">Choices:</p>
                        <div className="space-y-2 mt-2">
                          {selectedNodeState.listSelection.choices.map((choice, index) => (
                            <div key={index} className="border p-2 rounded-md">
                              <p><span className="font-medium">Text:</span> {choice.text}</p>
                              <p><span className="font-medium">Next State:</span> {choice.nextState || 'None'}</p>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          )}

          <DialogFooter>
            <Button variant="outline" onClick={() => setIsDialogOpen(false)}>Cancel</Button>
            <Button onClick={handleSaveChanges}>OK</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

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
