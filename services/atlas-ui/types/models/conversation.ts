// NPC Conversation domain model types
// Complete type definitions for conversation management

export interface DialogueChoice {
  text: string;
  nextState: string | null;
  context?: Record<string, string>;
}

export interface DialogueState {
  dialogueType: "sendOk" | "sendYesNo" | "sendSimple" | "sendNext";
  text: string;
  choices: DialogueChoice[];
  exit?: boolean;
}

export interface GenericActionOperation {
  type: string;
  params?: Record<string, never>;
}

export interface Condition {
  type: string;
  operator: string;
  value: string;
}

export interface GenericActionOutcome {
  conditions: Condition[];
  nextState: string;
}

export interface GenericActionState {
  operations: GenericActionOperation[];
  outcomes: GenericActionOutcome[];
}

export interface CraftActionState {
  itemId: number;
  materials: number[];
  quantities: number[];
  mesoCost: number;
  stimulatorId?: number;
  stimulatorFailChance?: number;
  successState: string;
  failureState: string;
  missingMaterialsState: string;
}

export interface ListSelectionState {
  title: string;
  choices: DialogueChoice[];
}

export interface ConversationState {
  id: string;
  type: "dialogue" | "genericAction" | "craftAction" | "listSelection";
  dialogue?: DialogueState;
  genericAction?: GenericActionState;
  craftAction?: CraftActionState;
  listSelection?: ListSelectionState;
}

export interface ConversationAttributes {
  npcId: number;
  startState: string;
  states: ConversationState[];
}

export interface Conversation {
  id: string;
  type: string;
  attributes: ConversationAttributes;
}

export interface ConversationResponse {
  data: Conversation;
}

export interface ConversationsResponse {
  data: Conversation[];
}