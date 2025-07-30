"use client"

import { useTenant } from "@/context/tenant-context";
import { useEffect, useState, useCallback, useMemo, useRef, SetStateAction } from "react";
import {useParams} from "next/navigation";
import { conversationsService } from "@/services/api/conversations.service";
import type {
  Conversation,
  ConversationState,
  DialogueChoice,
} from "@/types/models/conversation";
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
  addEdge,
  Connection,
  EdgeChange,
  OnEdgeUpdateFunc,

} from 'reactflow';
import 'reactflow/dist/style.css';
import {Button} from "@/components/ui/button";
import {RefreshCw, ZoomIn, ZoomOut, Info, Edit, Trash, MessageSquare, Cog, Hammer, List, X} from "lucide-react";
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
  onNodeDelete?: (nodeId: string) => void; // Add callback for node deletion
  onEdgeRemove?: (edgeId: string) => void; // Add callback for edge removal
  onTextEdit?: (nodeId: string, text: string) => void; // Add callback for text editing
}

const CustomNode = ({data, isConnectable, onNodeEdit, onNodeDelete, onEdgeRemove, onTextEdit, ...props}: CustomNodeProps) => {
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

        <div>
          <div className="flex justify-between items-center">
            <div className="flex items-center">
              {type === 'dialogue' && <MessageSquare className="h-4 w-4 mr-1" />}
              {type === 'genericAction' && <Cog className="h-4 w-4 mr-1" />}
              {type === 'craftAction' && <Hammer className="h-4 w-4 mr-1" />}
              {type === 'listSelection' && <List className="h-4 w-4 mr-1" />}
              <div className="font-bold">{label}</div>
            </div>
            <div className="flex">
              <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => onNodeEdit && onNodeEdit(id)}>
                <Edit className="h-4 w-4" />
              </Button>
              <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => onNodeDelete && onNodeDelete(id)}>
                <Trash className="h-4 w-4" />
              </Button>
            </div>
          </div>
          {text && (
              <div 
                className="text-xs mt-2 p-1 bg-black/10 rounded cursor-pointer hover:bg-black/20"
                onClick={() => onTextEdit && onTextEdit(id, text)}
                title="Click to edit"
              >
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
                      <div key={index} className="bg-black/10 rounded p-2 relative min-h-[32px]">
                        <div className="flex justify-between items-start">
                          <div className="flex-1">
                            {outcome.conditions.length > 0 ? (
                                <div>
                                  {outcome.conditions.map((condition: string, condIndex: number) => (
                                      <div key={condIndex}>{condition}</div>
                                  ))}
                                </div>
                            ) : (
                                <div className="italic">Otherwise</div>
                            )}
                          </div>
                          <div className="w-5 h-5 ml-2 flex-shrink-0">
                            {outcome.nextState && (
                              <Button 
                                variant="ghost" 
                                size="icon" 
                                className="h-5 w-5" 
                                onClick={(e) => {
                                  e.stopPropagation();
                                  // Create a change object to remove the edge
                                  const edgeId = `${id}-to-${outcome.nextState}-${index}`;
                                  if (onEdgeRemove) {
                                    onEdgeRemove(edgeId);
                                  }
                                }}
                              >
                                <X className="h-3 w-3" />
                              </Button>
                            )}
                          </div>
                        </div>
                        {/* Source handle for this outcome */}
                        <Handle
                          id={`outcome-${index}`}
                          type="source"
                          position={Position.Right}
                          style={{
                            background: '#555',
                            width: 8,
                            height: 8,
                            right: -14,
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
                  <div className="bg-black/10 rounded p-2 border border-white/20 relative min-h-[32px]">
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
                        right: -14,
                        top: '50%',
                        transform: 'translateY(-50%)'
                      }}
                      isConnectable={isConnectable}
                    />
                  </div>

                  <div className="bg-black/10 rounded p-2 border border-white/20 relative min-h-[32px]">
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
                        right: -14,
                        top: '50%',
                        transform: 'translateY(-50%)'
                      }}
                      isConnectable={isConnectable}
                    />
                  </div>

                  <div className="bg-black/10 rounded p-2 border border-white/20 relative min-h-[32px]">
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
                        right: -14,
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
                      <div key={index} className="bg-black/10 rounded p-2 border border-white/20 relative min-h-[32px]">
                        <div className="flex justify-between items-start">
                          <div className="flex-1">
                            {choice.text}
                          </div>
                          <div className="w-5 h-5 ml-2 flex-shrink-0">
                            {choice.nextState && (
                              <Button 
                                variant="ghost" 
                                size="icon" 
                                className="h-5 w-5" 
                                onClick={(e) => {
                                  e.stopPropagation();
                                  // Create a change object to remove the edge
                                  const edgeId = `${id}-to-${choice.nextState}-${index}`;
                                  if (onEdgeRemove) {
                                    onEdgeRemove(edgeId);
                                  }
                                }}
                              >
                                <X className="h-3 w-3" />
                              </Button>
                            )}
                          </div>
                        </div>
                        {/* Source handle for this choice */}
                        <Handle
                          id={`choice-${index}`}
                          type="source"
                          position={Position.Right}
                          style={{
                            background: '#555',
                            width: 8,
                            height: 8,
                            right: -14,
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

// Define node types factory function outside the component
const createNodeTypes = (
  onNodeEdit: (nodeId: string) => void, 
  onNodeDelete: (nodeId: string) => void,
  onEdgeRemove: (edgeId: string) => void,
  onTextEdit: (nodeId: string, text: string) => void
) => {
  return {
    customNode: (props: CustomNodeProps) => (
      <CustomNode 
        {...props} 
        onNodeEdit={onNodeEdit} 
        onNodeDelete={onNodeDelete} 
        onEdgeRemove={onEdgeRemove} 
        onTextEdit={onTextEdit} 
      />
    )
  };
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
        itemId: state.craftAction.itemId,
        materials: state.craftAction.materials,
        quantities: state.craftAction.quantities,
        mesoCost: state.craftAction.mesoCost,
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

  // Custom edge change handler to update conversation state when edges are removed
  const handleEdgesChange = useCallback((changes: EdgeChange[]) => {
    // First, apply the changes to the edges state using the default handler
    onEdgesChange(changes);

    // Check if any of the changes are of type 'remove'
    const removedEdges = changes.filter(change => change.type === 'remove');

    if (removedEdges.length > 0 && conversation) {
      // Create a copy of the conversation to modify
      const updatedConversation = { ...conversation };

      // For each removed edge, find the corresponding connection in the conversation state and update it
      removedEdges.forEach(removedEdge => {
        const edgeId = removedEdge.id;
        const edge = edges.find(e => e.id === edgeId);

        if (edge) {
          const { source: sourceId, target: targetId, sourceHandle } = edge;

          // Find the source state in the conversation data
          const sourceState = updatedConversation.attributes.states.find(
            state => state.id === sourceId
          );

          if (sourceState) {
            if (sourceHandle?.startsWith('choice-')) {
              // This is a choice connection
              const choiceIndex = parseInt(sourceHandle.split('-')[1] || '0');

              if (sourceState.type === 'dialogue' && sourceState.dialogue?.choices) {
                // Update the nextState for this choice
                if (choiceIndex >= 0 && choiceIndex < sourceState.dialogue.choices.length) {
                  // Only update if this choice points to the target node
                  const choice = sourceState.dialogue.choices[choiceIndex];
                  if (choice && choice.nextState === targetId) {
                    choice.nextState = null;
                  }
                }
              } else if (sourceState.type === 'listSelection' && sourceState.listSelection?.choices) {
                // Update the nextState for this choice in listSelection
                if (choiceIndex >= 0 && choiceIndex < sourceState.listSelection.choices.length) {
                  // Only update if this choice points to the target node
                  const choice = sourceState.listSelection.choices[choiceIndex];
                  if (choice && choice.nextState === targetId) {
                    choice.nextState = null;
                  }
                }
              }
            } else if (sourceHandle?.startsWith('outcome-')) {
              // This is an outcome connection
              const outcomeIndex = parseInt(sourceHandle.split('-')[1] || '0');

              if (sourceState.type === 'genericAction' && sourceState.genericAction?.outcomes) {
                // Update the nextState for this outcome
                if (outcomeIndex >= 0 && outcomeIndex < sourceState.genericAction.outcomes.length) {
                  // Only update if this outcome points to the target node
                  const outcome = sourceState.genericAction.outcomes[outcomeIndex];
                  if (outcome && outcome.nextState === targetId) {
                    outcome.nextState = '';
                  }
                }
              }
            } else if (sourceHandle === 'success' || sourceHandle === 'failure' || sourceHandle === 'missing') {
              // This is a craftAction connection
              if (sourceState.type === 'craftAction' && sourceState.craftAction) {
                if (sourceHandle === 'success' && sourceState.craftAction.successState === targetId) {
                  sourceState.craftAction.successState = '';
                } else if (sourceHandle === 'failure' && sourceState.craftAction.failureState === targetId) {
                  sourceState.craftAction.failureState = '';
                } else if (sourceHandle === 'missing' && sourceState.craftAction.missingMaterialsState === targetId) {
                  sourceState.craftAction.missingMaterialsState = '';
                }
              }
            }
          }
        }
      });

      // Update the conversation state
      setConversation(updatedConversation);
    }
  }, [onEdgesChange, edges, conversation, setConversation]);

  // State for node editing dialog
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [isSaveDialogOpen, setIsSaveDialogOpen] = useState(false);
  const [isAddNodeDialogOpen, setIsAddNodeDialogOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
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
  const [editDialogueType, setEditDialogueType] = useState<"sendOk" | "sendYesNo" | "sendSimple" | "sendNext">("sendOk");

  // State for tracking edge updates
  const [, setEdgeUpdateSuccessful] = useState(false);

  // State for temporary edit choices
  const [editChoices, setEditChoices] = useState<DialogueChoice[]>([]);

  // State for backup of conversation before editing
  const [conversationBackup, setConversationBackup] = useState<Conversation | null>(null);

  // State for text editing dialog
  const [isTextEditDialogOpen, setIsTextEditDialogOpen] = useState(false);
  const [editingNodeId, setEditingNodeId] = useState<string>('');
  const [editingText, setEditingText] = useState<string>('');

  // Handle dialogue type change - defined first to avoid circular dependency
  const handleDialogueTypeChange = useCallback((newDialogueType: "sendOk" | "sendYesNo" | "sendSimple" | "sendNext") => {
    setEditDialogueType(newDialogueType);

    // Create blank choices based on dialogue type
    let blankChoices: DialogueChoice[] = [];

    switch (newDialogueType) {
      case "sendOk":
        blankChoices = [
          { text: "Ok", nextState: null },
          { text: "Exit", nextState: null }
        ];
        break;
      case "sendYesNo":
        blankChoices = [
          { text: "Yes", nextState: null },
          { text: "No", nextState: null },
          { text: "Exit", nextState: null }
        ];
        break;
      case "sendSimple":
        blankChoices = [
          { text: "Ok", nextState: null },
          { text: "Exit", nextState: null }
        ];
        break;
      case "sendNext":
        blankChoices = [
          { text: "Next", nextState: null },
          { text: "Exit", nextState: null }
        ];
        break;
    }

    // Update the edit choices state
    setEditChoices(blankChoices);
  }, []);

  // Handle node type change - defined after handleDialogueTypeChange to avoid circular dependency
  const handleNodeTypeChange = useCallback((newType: string) => {
    setEditNodeType(newType);

    // Clear fields that are not relevant to the new node type
    if (newType !== 'dialogue') {
      setEditNodeText('');
    }

    if (newType !== 'listSelection') {
      setEditNodeTitle('');
    }

    // If changing to dialogue type, set up choices based on the current dialogue type
    if (newType === 'dialogue') {
      // Use the current dialogue type to set up choices
      handleDialogueTypeChange(editDialogueType);
    } else {
      // Clear edit choices if not dialogue
      setEditChoices([]);
    }
  }, [editDialogueType, handleDialogueTypeChange]);

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
        ...(currentState.dialogue || { choices: [] }),
        dialogueType: editDialogueType,
        text: editNodeText,
        choices: editChoices.length > 0 ? editChoices : (currentState.dialogue?.choices || [])
      };
    } else if (updatedState.type === 'listSelection') {
      // Ensure there's always an Exit choice
      let choices = currentState.listSelection?.choices || [];
      if (!choices.some(choice => choice.text === "Exit")) {
        choices = [...choices, { text: "Exit", nextState: null }];
      }

      updatedState.listSelection = {
        ...(currentState.listSelection || { choices: [] }),
        title: editNodeTitle,
        choices: choices
      };
    } else if (updatedState.type === 'genericAction') {
      updatedState.genericAction = currentState.genericAction || { operations: [], outcomes: [] };
    } else if (updatedState.type === 'craftAction') {
      updatedState.craftAction = currentState.craftAction || {
        itemId: 0,
        materials: [],
        quantities: [],
        mesoCost: 0,
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

    // Clear the backup
    setConversationBackup(null);

    // Show success message
    toast.success("Node updated successfully");
  }, [conversation, selectedNodeId, editNodeId, editNodeType, editNodeText, editNodeTitle, setNodes, setEdges]);

  // Handle adding a new node
  const handleAddNode = useCallback(() => {
    // Initialize with default values for a new node
    setSelectedNodeId(null);
    setEditNodeId(`state_${Date.now()}`); // Generate a unique ID
    setEditNodeType('dialogue'); // Default type
    setEditNodeText(''); // Empty text
    setEditNodeTitle(''); // Empty title
    setEditDialogueType('sendOk'); // Default dialogue type

    // Initialize with default choices based on dialogue type
    handleDialogueTypeChange('sendOk');

    // Create a backup of the conversation state
    if (conversation) {
      setConversationBackup(JSON.parse(JSON.stringify(conversation)));
    }

    // Open the add node dialog
    setIsAddNodeDialogOpen(true);
  }, [conversation, handleDialogueTypeChange]);

  // Handle saving a new node
  const handleSaveNewNode = useCallback(() => {
    if (!conversation) return;

    // Create a copy of the conversation
    const updatedConversation = { ...conversation };

    // Create a new state object
    const newState: ConversationState = {
      id: editNodeId,
      type: editNodeType as "dialogue" | "genericAction" | "craftAction" | "listSelection",
    };

    // Add type-specific properties
    if (newState.type === 'dialogue') {
      newState.dialogue = {
        dialogueType: editDialogueType,
        text: editNodeText,
        choices: editChoices
      };
    } else if (newState.type === 'listSelection') {
      newState.listSelection = {
        title: editNodeTitle,
        choices: [
          { text: "Exit", nextState: null }
        ]
      };
    } else if (newState.type === 'genericAction') {
      newState.genericAction = { operations: [], outcomes: [] };
    } else if (newState.type === 'craftAction') {
      newState.craftAction = {
        itemId: 0,
        materials: [],
        quantities: [],
        mesoCost: 0,
        successState: '', 
        failureState: '', 
        missingMaterialsState: '' 
      };
    }

    // Add the new state to the conversation
    updatedConversation.attributes.states.push(newState);

    // If this is the first state, set it as the start state
    if (updatedConversation.attributes.states.length === 1) {
      updatedConversation.attributes.startState = newState.id;
    }

    // Update the conversation in state
    setConversation(updatedConversation);

    // Process the updated conversation to update nodes and edges
    const { nodes: processedNodes, edges: processedEdges } = processConversationData(updatedConversation);
    setNodes(processedNodes);
    setEdges(processedEdges);

    // Close the dialog
    setIsAddNodeDialogOpen(false);

    // Clear the backup
    setConversationBackup(null);

    // Show success message
    toast.success("Node added successfully");
  }, [conversation, editNodeId, editNodeType, editNodeText, editNodeTitle, setNodes, setEdges]);

  // Handle node edit
  const handleNodeEdit = useCallback((nodeId: string) => {
    setSelectedNodeId(nodeId);

    // Create a backup of the conversation state
    if (conversation) {
      setConversationBackup(JSON.parse(JSON.stringify(conversation)));
    }

    // Find the node state in the conversation
    const nodeState = conversation?.attributes.states.find(state => state.id === nodeId);

    if (nodeState) {
      // Initialize edit state with current values
      setEditNodeId(nodeState.id);
      setEditNodeType(nodeState.type);

      // Set text for dialogue nodes
      if (nodeState.type === 'dialogue' && nodeState.dialogue) {
        setEditNodeText(nodeState.dialogue.text || '');
        // Set dialogueType for dialogue nodes
        if (nodeState.dialogue?.dialogueType) {
          setEditDialogueType(nodeState.dialogue.dialogueType);
        } else {
          setEditDialogueType("sendOk");
        }
        // Set choices for dialogue nodes
        if (nodeState.dialogue?.choices) {
          setEditChoices(nodeState.dialogue.choices);
        } else {
          // Create default choices based on dialogue type
          handleDialogueTypeChange(nodeState.dialogue?.dialogueType || "sendOk");
        }
      } else {
        setEditNodeText('');
        setEditChoices([]);
      }

      // Set title for listSelection nodes
      if (nodeState.type === 'listSelection' && nodeState.listSelection?.title) {
        setEditNodeTitle(nodeState.listSelection.title);
      } else {
        setEditNodeTitle('');
      }
    }

    setIsDialogOpen(true);
  }, [conversation, handleDialogueTypeChange]);

  // Handle node delete
  const handleNodeDelete = useCallback((nodeId: string) => {
    setSelectedNodeId(nodeId);
    setIsDeleteDialogOpen(true);
  }, []);

  // Handle text edit
  const handleTextEdit = useCallback((nodeId: string, text: string) => {
    // Create a backup of the conversation state
    if (conversation) {
      setConversationBackup(JSON.parse(JSON.stringify(conversation)));
    }

    setEditingNodeId(nodeId);
    setEditingText(text);
    setIsTextEditDialogOpen(true);
  }, [conversation]);

  // Handle save text changes
  const handleSaveTextChanges = useCallback(() => {
    if (!conversation || !editingNodeId) return;

    // Create a copy of the conversation
    const updatedConversation = { ...conversation };

    // Find the node state in the conversation
    const nodeState = updatedConversation.attributes.states.find(state => state.id === editingNodeId);

    if (nodeState) {
      // Update the text based on node type
      if (nodeState.type === 'dialogue' && nodeState.dialogue) {
        nodeState.dialogue.text = editingText;
      } else if (nodeState.type === 'listSelection' && nodeState.listSelection) {
        nodeState.listSelection.title = editingText;
      }

      // Update the conversation in state
      setConversation(updatedConversation);

      // Process the updated conversation to update nodes and edges
      const { nodes: processedNodes, edges: processedEdges } = processConversationData(updatedConversation);
      setNodes(processedNodes);
      setEdges(processedEdges);

      // Close the dialog
      setIsTextEditDialogOpen(false);

      // Clear the backup
      setConversationBackup(null);

      // Show success message
      toast.success("Text updated successfully");
    }
  }, [conversation, editingNodeId, editingText, setNodes, setEdges]);

  // Handle confirm delete
  const handleConfirmDelete = useCallback(() => {
    if (!conversation || !selectedNodeId) return;

    // Create a copy of the conversation
    const updatedConversation = { ...conversation };

    // Find the node state index in the conversation
    const nodeIndex = updatedConversation.attributes.states.findIndex(state => state.id === selectedNodeId);

    if (nodeIndex === -1) return;

    // Remove the node from the conversation
    updatedConversation.attributes.states.splice(nodeIndex, 1);

    // If the deleted node was the start state, update the start state
    if (selectedNodeId && updatedConversation.attributes.startState === selectedNodeId) {
      // If there are other states, set the first one as the start state
      if (updatedConversation.attributes.states.length > 0) {
        const firstState = updatedConversation.attributes.states[0];
        updatedConversation.attributes.startState = firstState?.id || '';
      } else {
        // If no states left, clear the start state
        updatedConversation.attributes.startState = '';
      }
    }

    // Clear any references to the deleted node in other states
    updatedConversation.attributes.states.forEach(state => {
      // Update dialogue choices
      if (state.type === 'dialogue' && state.dialogue?.choices) {
        state.dialogue.choices.forEach(choice => {
          if (choice.nextState === selectedNodeId) {
            choice.nextState = null;
          }
        });
      }

      // Update genericAction outcomes
      if (state.type === 'genericAction' && state.genericAction?.outcomes) {
        state.genericAction.outcomes.forEach(outcome => {
          if (outcome.nextState === selectedNodeId) {
            outcome.nextState = '';
          }
        });
      }

      // Update craftAction state transitions
      if (state.type === 'craftAction' && state.craftAction) {
        if (state.craftAction.successState === selectedNodeId) {
          state.craftAction.successState = '';
        }
        if (state.craftAction.failureState === selectedNodeId) {
          state.craftAction.failureState = '';
        }
        if (state.craftAction.missingMaterialsState === selectedNodeId) {
          state.craftAction.missingMaterialsState = '';
        }
      }

      // Update listSelection choices
      if (state.type === 'listSelection' && state.listSelection?.choices) {
        state.listSelection.choices.forEach(choice => {
          if (choice.nextState === selectedNodeId) {
            choice.nextState = null;
          }
        });
      }
    });

    // Update the conversation in state
    setConversation(updatedConversation);

    // Process the updated conversation to update nodes and edges
    const { nodes: processedNodes, edges: processedEdges } = processConversationData(updatedConversation);
    setNodes(processedNodes);
    setEdges(processedEdges);

    // Close the dialog
    setIsDeleteDialogOpen(false);

    // Clear the selected node
    setSelectedNodeId(null);

    // Show success message
    toast.success("Node deleted successfully");
  }, [conversation, selectedNodeId, setNodes, setEdges]);

  // Use the factory function to create nodeTypes with the edit and delete handlers
  const nodeTypes = useMemo(
    () => createNodeTypes(
      handleNodeEdit, 
      handleNodeDelete, 
      (edgeId) => {
        const edgeRemoveChange = {
          id: edgeId,
          type: 'remove' as const,
        };
        handleEdgesChange([edgeRemoveChange]);
      },
      handleTextEdit
    ),
    [handleNodeEdit, handleNodeDelete, handleEdgesChange, handleTextEdit]
  );

  const reactFlowInstance = useReactFlow();

  // Function to check if a source handle already has a connection
  const isHandleConnected = useCallback((sourceId: string | null, sourceHandle: string | null) => {
    return edges.some(edge => 
      edge.source === sourceId && 
      (sourceHandle === null || edge.sourceHandle === sourceHandle)
    );
  }, [edges]);

  // Handle new connections
  const onConnect = useCallback((params: Connection) => {
    // Check if the source handle already has a connection
    if (isHandleConnected(params.source, params.sourceHandle)) {
      // If it does, don't create a new connection
      return;
    }

    // Find source and target nodes to log information
    const sourceNode = nodes.find(node => node.id === params.source);
    const targetNode = nodes.find(node => node.id === params.target);

    if (sourceNode && targetNode) {
      const sourceHandleId = params.sourceHandle;

      // Determine what type of connection this is based on the handle ID
      const connectionInfo = {
        sourceStateId: sourceNode.id,
        targetStateId: targetNode.id,
        connectionType: '',
        itemLabel: ''
      };

      // Update the conversation state based on the connection type
      if (conversation) {
        // Create a copy of the conversation to modify
        const updatedConversation = { ...conversation };

        // Find the source state in the conversation data
        const sourceState = updatedConversation.attributes.states.find(
          state => state.id === sourceNode.id
        );

        if (sourceState) {
          if (sourceHandleId?.startsWith('choice-')) {
            // This is a choice connection
            const choiceIndex = parseInt(sourceHandleId.split('-')[1] || '0');

            if (sourceState.type === 'dialogue' && sourceState.dialogue?.choices) {
              // Update the nextState for this choice
              if (choiceIndex >= 0 && choiceIndex < sourceState.dialogue.choices.length) {
                const choice = sourceState.dialogue.choices[choiceIndex];
                if (choice) {
                  choice.nextState = targetNode.id;
                  connectionInfo.connectionType = 'choice';
                  connectionInfo.itemLabel = choice.text || 'Unknown choice';
                }
              }
            } else if (sourceState.type === 'listSelection' && sourceState.listSelection?.choices) {
              // Update the nextState for this choice in listSelection
              if (choiceIndex >= 0 && choiceIndex < sourceState.listSelection.choices.length) {
                const choice = sourceState.listSelection.choices[choiceIndex];
                if (choice) {
                  choice.nextState = targetNode.id;
                  connectionInfo.connectionType = 'choice';
                  connectionInfo.itemLabel = choice.text || 'Unknown choice';
                }
              }
            }
          } else if (sourceHandleId?.startsWith('outcome-')) {
            // This is an outcome connection
            const outcomeIndex = parseInt(sourceHandleId.split('-')[1] || '0');

            if (sourceState.type === 'genericAction' && sourceState.genericAction?.outcomes) {
              // Update the nextState for this outcome
              if (outcomeIndex >= 0 && outcomeIndex < sourceState.genericAction.outcomes.length) {
                const outcome = sourceState.genericAction.outcomes[outcomeIndex];
                if (outcome) {
                  outcome.nextState = targetNode.id;
                  connectionInfo.connectionType = 'outcome';
                  const conditions = outcome.conditions;
                  connectionInfo.itemLabel = conditions.length > 0 ? 
                    `Outcome with ${conditions.length} condition(s)` : 
                    'Default outcome';
                }
              }
            }
          } else if (sourceHandleId === 'success' || sourceHandleId === 'failure' || sourceHandleId === 'missing') {
            // This is a craftAction connection
            if (sourceState.type === 'craftAction' && sourceState.craftAction) {
              if (sourceHandleId === 'success') {
                sourceState.craftAction.successState = targetNode.id;
              } else if (sourceHandleId === 'failure') {
                sourceState.craftAction.failureState = targetNode.id;
              } else if (sourceHandleId === 'missing') {
                sourceState.craftAction.missingMaterialsState = targetNode.id;
              }

              connectionInfo.connectionType = 'craftAction';
              connectionInfo.itemLabel = sourceHandleId;
            }
          }

          // Update the conversation state
          setConversation(updatedConversation);
        }
      }

      // Log the connection information
      console.log('New connection created:', connectionInfo);
    }

    // Create the new connection with a specific ID format
    let edgeId = '';
    let edgeStyle = { stroke: '#64748b' };

    // Generate the edge ID based on the connection type
    if (sourceNode && targetNode && params.sourceHandle) {
      if (params.sourceHandle.startsWith('choice-')) {
        const choiceIndex = parseInt(params.sourceHandle.split('-')[1] || '0');
        edgeId = `${sourceNode.id}-to-${targetNode.id}-${choiceIndex}`;
      } else if (params.sourceHandle.startsWith('outcome-')) {
        const outcomeIndex = parseInt(params.sourceHandle.split('-')[1] || '0');
        edgeId = `${sourceNode.id}-to-${targetNode.id}-${outcomeIndex}`;
      } else if (params.sourceHandle === 'success') {
        edgeId = `${sourceNode.id}-to-${targetNode.id}-success`;
        edgeStyle = { stroke: '#22c55e' };
      } else if (params.sourceHandle === 'failure') {
        edgeId = `${sourceNode.id}-to-${targetNode.id}-failure`;
        edgeStyle = { stroke: '#ef4444' };
      } else if (params.sourceHandle === 'missing') {
        edgeId = `${sourceNode.id}-to-${targetNode.id}-missing`;
        edgeStyle = { stroke: '#f59e0b' };
      }
    }

    // If we couldn't generate a specific ID, use a default format
    if (!edgeId && sourceNode && targetNode) {
      edgeId = `${sourceNode.id}-to-${targetNode.id}-${Date.now()}`;
    }

    setEdges(eds => addEdge({
      ...params,
      id: edgeId,
      type: 'smoothstep',
      animated: false,
      style: edgeStyle,
    }, eds));
  }, [isHandleConnected, setEdges, setNodes, nodes, conversation, setConversation, processConversationData]);

  // Edge update handlers for drag-and-drop deletion
  const onEdgeUpdateStart = useCallback(() => {
    // Reset the edge update successful flag when starting to drag an edge
    setEdgeUpdateSuccessful(false);
  }, []);

  const onEdgeUpdate = useCallback<OnEdgeUpdateFunc>((oldEdge, newConnection) => {
    // Mark the update as successful if we have a valid connection
    setEdgeUpdateSuccessful(true);

    // Update the edges with the new connection
    setEdges((els) => updateEdge(oldEdge, newConnection, els));

    // Update the conversation data with the new connection
    if (conversation) {
      // Create a copy of the conversation to modify
      const updatedConversation = { ...conversation };

      // Find the source state in the conversation data
      const sourceState = updatedConversation.attributes.states.find(
        state => state.id === oldEdge.source
      );

      if (sourceState) {
        const sourceHandleId = oldEdge.sourceHandle;

        // Update the appropriate connection in the conversation data
        if (sourceHandleId?.startsWith('choice-')) {
          // This is a choice connection
          const choiceIndex = parseInt(sourceHandleId.split('-')[1] || '0');

          if (sourceState.type === 'dialogue' && sourceState.dialogue?.choices) {
            // Update the nextState for this choice
            if (choiceIndex >= 0 && choiceIndex < sourceState.dialogue.choices.length) {
              const choice = sourceState.dialogue.choices[choiceIndex];
              // Only update if this choice points to the old target node
              if (choice && choice.nextState === oldEdge.target) {
                if (newConnection.target != null) {
                  choice.nextState = newConnection.target;
                }
              }
            }
          } else if (sourceState.type === 'listSelection' && sourceState.listSelection?.choices) {
            // Update the nextState for this choice in listSelection
            if (choiceIndex >= 0 && choiceIndex < sourceState.listSelection.choices.length) {
              const choice = sourceState.listSelection.choices[choiceIndex];
              // Only update if this choice points to the old target node
              if (choice && choice.nextState === oldEdge.target) {
                if (newConnection.target != null) {
                  choice.nextState = newConnection.target;
                }
              }
            }
          }
        } else if (sourceHandleId?.startsWith('outcome-')) {
          // This is an outcome connection
          const outcomeIndex = parseInt(sourceHandleId.split('-')[1] || '0');

          if (sourceState.type === 'genericAction' && sourceState.genericAction?.outcomes) {
            // Update the nextState for this outcome
            if (outcomeIndex >= 0 && outcomeIndex < sourceState.genericAction.outcomes.length) {
              const outcome = sourceState.genericAction.outcomes[outcomeIndex];
              // Only update if this outcome points to the old target node
              if (outcome && outcome.nextState === oldEdge.target) {
                if (newConnection.target != null) {
                  outcome.nextState = newConnection.target;
                }
              }
            }
          }
        } else if (sourceHandleId === 'success' || sourceHandleId === 'failure' || sourceHandleId === 'missing') {
          // This is a craftAction connection
          if (sourceState.type === 'craftAction' && sourceState.craftAction) {
            if (sourceHandleId === 'success' && sourceState.craftAction.successState === oldEdge.target) {
              if (newConnection.target != null) {
                sourceState.craftAction.successState = newConnection.target;
              }
            } else if (sourceHandleId === 'failure' && sourceState.craftAction.failureState === oldEdge.target) {
              if (newConnection.target != null) {
                sourceState.craftAction.failureState = newConnection.target;
              }
            } else if (sourceHandleId === 'missing' && sourceState.craftAction.missingMaterialsState === oldEdge.target) {
              if (newConnection.target != null) {
                sourceState.craftAction.missingMaterialsState = newConnection.target;
              }
            }
          }
        }

        // Update the conversation state
        setConversation(updatedConversation);
      }
    }

    return true;
  }, [conversation, setConversation, setEdges]);

  // Helper function to update an edge
  const updateEdge = (oldEdge: Edge, newConnection: Connection, edges: Edge[]): Edge[] => {
    // Remove the old edge
    const updatedEdges = edges.filter((e) => e.id !== oldEdge.id);

    // Add the new edge with the same id and style, being careful about exact optional properties
    const newEdge: Edge = {
      id: oldEdge.id,
      source: newConnection.source || oldEdge.source,
      target: newConnection.target || oldEdge.target,
      sourceHandle: newConnection.sourceHandle !== undefined ? newConnection.sourceHandle : oldEdge.sourceHandle,
      targetHandle: newConnection.targetHandle !== undefined ? newConnection.targetHandle : oldEdge.targetHandle,
      // Copy other properties from oldEdge that might be optional
      ...(oldEdge.type && { type: oldEdge.type }),
      ...(oldEdge.style && { style: oldEdge.style }),
      ...(oldEdge.animated !== undefined && { animated: oldEdge.animated }),
      ...(oldEdge.hidden !== undefined && { hidden: oldEdge.hidden }),
      ...(oldEdge.deletable !== undefined && { deletable: oldEdge.deletable }),
      ...(oldEdge.data && { data: oldEdge.data }),
      ...(oldEdge.className && { className: oldEdge.className }),
      ...(oldEdge.selected !== undefined && { selected: oldEdge.selected }),
      ...(oldEdge.markerStart && { markerStart: oldEdge.markerStart }),
      ...(oldEdge.markerEnd && { markerEnd: oldEdge.markerEnd }),
      ...(oldEdge.zIndex !== undefined && { zIndex: oldEdge.zIndex }),
      ...(oldEdge.ariaLabel && { ariaLabel: oldEdge.ariaLabel }),
      ...(oldEdge.interactionWidth !== undefined && { interactionWidth: oldEdge.interactionWidth }),
    };

    return [...updatedEdges, newEdge];
  };

  const fetchConversationData = useCallback(async () => {
    if (!activeTenant) return;

    setLoading(true);
    setError(null);

    try {
      const conversationData = await conversationsService.getByNpcId(npcId);

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

    // Calculate the average position of the connected nodes to use as a reference point
    let avgX = 0;
    let maxLevel = 0;

    if (nodePositions.length > 0) {
      // Calculate the average X position of all positioned nodes
      avgX = nodePositions.reduce((sum, node) => sum + node.position.x, 0) / nodePositions.length;

      // Find the maximum level to place disconnected nodes after it
      maxLevel = Math.max(...Array.from(nodeLevels.values()));
    }

    // Position for disconnected nodes - place them at the next level after the max level
    const disconnectedX = maxLevel > 0 ? (maxLevel + 1) * horizontalSpacing + 100 : avgX + horizontalSpacing;
    let disconnectedY = 100; // Starting Y position for disconnected nodes

    // Add any nodes that weren't visited in our traversal
    nodes.forEach(node => {
      if (!nodePositions.some(n => n.id === node.id)) {
        // Use the estimated height for this node, or a default of 250
        const nodeHeight = nodeHeights.get(node.id) || 250;
        nodePositions.push({
          ...node,
          position: { x: disconnectedX, y: disconnectedY }
        });

        // Update Y position for the next disconnected node
        disconnectedY += nodeHeight + verticalSpacing;
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

  // Update nodes and edges when conversation changes
  const prevConversationRef = useRef<Conversation | null>(null);
  useEffect(() => {
    // Only update if the conversation has actually changed
    if (conversation && conversation !== prevConversationRef.current) {
      prevConversationRef.current = conversation;

      // Process the conversation to get updated nodes and edges
      const { nodes: processedNodes, edges: processedEdges } = processConversationData(conversation);

      // Create a map of existing node positions
      const nodePositions = new Map<string, { x: number, y: number }>();
      nodes.forEach(node => {
        nodePositions.set(node.id, { ...node.position });
      });

      // Update nodes with new data but preserve positions
      const updatedNodes = processedNodes.map(node => {
        return {
          ...node,
          position: nodePositions.get(node.id) || node.position
        };
      });

      // Update the nodes and edges state
      setNodes(updatedNodes);
      setEdges(processedEdges);
    }
  }, [conversation, nodes, processConversationData, setNodes, setEdges]);

  const handleZoomIn = () => {
    reactFlowInstance.zoomIn();
  };

  const handleZoomOut = () => {
    reactFlowInstance.zoomOut();
  };

  const handleFitView = () => {
    reactFlowInstance.fitView({ padding: 0.2 });
  };

  // Handle saving the conversation to the database
  const handleSaveConversation = useCallback(async () => {
    if (!conversation) return;

    try {
      // Save the current conversation state to the database
      await conversationsService.update(conversation.id, conversation.attributes);
      toast.success("Conversation saved successfully");
      setIsSaveDialogOpen(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to save conversation");
    }
  }, [conversation]);

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
      {/* Save Confirmation Dialog */}
      <Dialog open={isSaveDialogOpen} onOpenChange={setIsSaveDialogOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Save Conversation</DialogTitle>
          </DialogHeader>
          <div className="py-4">
            <p>Are you sure you want to save the current conversation? This will update the conversation in the database.</p>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsSaveDialogOpen(false)}>Cancel</Button>
            <Button onClick={handleSaveConversation}>Yes, Save</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Node Edit Dialog */}
      <Dialog 
        open={isDialogOpen} 
        onOpenChange={(open) => {
          if (!open) {
            // Dialog is being closed without saving, restore the backup
            if (conversationBackup) {
              setConversation(conversationBackup);
              // Clear the backup
              setConversationBackup(null);
            }
            setIsDialogOpen(false);
          } else {
            setIsDialogOpen(true);
          }
        }}
      >
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
                  <h3 className="text-lg font-semibold">Dialogue Type</h3>
                  <select 
                    value={editDialogueType} 
                    onChange={(e) => handleDialogueTypeChange(e.target.value as "sendOk" | "sendYesNo" | "sendSimple" | "sendNext")}
                    className="w-full p-2 border rounded-md"
                  >
                    <option value="sendOk">sendOk</option>
                    <option value="sendYesNo">sendYesNo</option>
                    <option value="sendSimple">sendSimple</option>
                    <option value="sendNext">sendNext</option>
                  </select>
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

      {/* Add Node Dialog */}
      <Dialog 
        open={isAddNodeDialogOpen} 
        onOpenChange={(open) => {
          if (!open) {
            // Dialog is being closed without saving, restore the backup
            if (conversationBackup) {
              setConversation(conversationBackup);
              // Clear the backup
              setConversationBackup(null);
            }
            setIsAddNodeDialogOpen(false);
          } else {
            setIsAddNodeDialogOpen(true);
          }
        }}
      >
        <DialogContent className="max-w-3xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Add New Node</DialogTitle>
          </DialogHeader>

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
                <h3 className="text-lg font-semibold">Dialogue Type</h3>
                <select 
                  value={editDialogueType} 
                  onChange={(e) => handleDialogueTypeChange(e.target.value as "sendOk" | "sendYesNo" | "sendSimple" | "sendNext")}
                  className="w-full p-2 border rounded-md"
                >
                  <option value="sendOk">sendOk</option>
                  <option value="sendYesNo">sendYesNo</option>
                  <option value="sendSimple">sendSimple</option>
                  <option value="sendNext">sendNext</option>
                </select>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setIsAddNodeDialogOpen(false)}>Cancel</Button>
            <Button onClick={handleSaveNewNode}>Create Node</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={isDeleteDialogOpen} onOpenChange={setIsDeleteDialogOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Delete Node</DialogTitle>
          </DialogHeader>
          <div className="py-4">
            <p>Are you sure you want to delete this node? This action cannot be undone and will remove any connections to this node.</p>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsDeleteDialogOpen(false)}>Cancel</Button>
            <Button variant="destructive" onClick={handleConfirmDelete}>Yes, Delete</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Text Edit Dialog */}
      <Dialog 
        open={isTextEditDialogOpen} 
        onOpenChange={(open) => {
          if (!open) {
            // Dialog is being closed without saving, restore the backup
            if (conversationBackup) {
              setConversation(conversationBackup);
              // Clear the backup
              setConversationBackup(null);
            }
            setIsTextEditDialogOpen(false);
          } else {
            setIsTextEditDialogOpen(true);
          }
        }}
      >
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Edit Text</DialogTitle>
          </DialogHeader>
          <div className="py-4">
            <textarea 
              value={editingText} 
              onChange={(e) => setEditingText(e.target.value)}
              className="w-full p-2 border rounded-md min-h-[150px]"
              placeholder="Enter text here..."
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsTextEditDialogOpen(false)}>Cancel</Button>
            <Button onClick={handleSaveTextChanges}>Apply</Button>
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
          <Button variant="default" size="sm" onClick={() => setIsSaveDialogOpen(true)}>
            Save
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
          onEdgesChange={handleEdgesChange}
          onConnect={onConnect}
          onEdgeUpdate={onEdgeUpdate}
          onEdgeUpdateStart={onEdgeUpdateStart}
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
            <Separator className="my-1" />
            <Button variant="default" size="sm" onClick={handleAddNode}>
              Add Node
            </Button>
          </Panel>
        </ReactFlow>
      </div>
    </div>
  );
}
