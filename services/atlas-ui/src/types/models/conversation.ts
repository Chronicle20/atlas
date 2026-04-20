// NPC Conversation domain model types
// Complete type definitions for conversation management

export interface DialogueChoice {
  text: string;
  nextState: string | null;
  context?: Record<string, string>;
}

export interface DialogueState {
  dialogueType: "sendOk" | "sendYesNo" | "sendNext";
  text: string;
  choices: DialogueChoice[];
  exit?: boolean;
}

export interface GenericActionOperation {
  type: string;
  params?: Record<string, string>;
}

export interface Condition {
  type: string;
  operator: string;
  value: string;
  referenceId?: string;
  step?: string;
  worldId?: number | string;
  channelId?: number | string;
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

export interface AskNumberState {
  text: string;
  defaultValue: number;
  minValue: number;
  maxValue: number;
  contextKey?: string;
  nextState: string;
}

export interface AskStyleState {
  text: string;
  styles?: number[];
  stylesContextKey?: string;
  contextKey?: string;
  nextState: string;
}

export interface AskSlideMenuState {
  title: string;
  menuType?: number;
  choices: DialogueChoice[];
}

export interface TransportActionState {
  routeName: string;
  failureState: string;
  capacityFullState?: string;
  alreadyInTransitState?: string;
  routeNotFoundState?: string;
  serviceErrorState?: string;
}

export interface PartyQuestActionState {
  questId: string;
  failureState: string;
  notInPartyState?: string;
  notLeaderState?: string;
}

export interface PartyQuestBonusActionState {
  failureState: string;
}

export interface GachaponActionState {
  gachaponId: string;
  ticketItemId: number;
  failureState: string;
}

export type ConversationStateType =
  | "dialogue"
  | "genericAction"
  | "craftAction"
  | "listSelection"
  | "askNumber"
  | "askStyle"
  | "askSlideMenu"
  | "transportAction"
  | "partyQuestAction"
  | "partyQuestBonusAction"
  | "gachaponAction";

export interface ConversationState {
  id: string;
  type: ConversationStateType;
  dialogue?: DialogueState;
  genericAction?: GenericActionState;
  craftAction?: CraftActionState;
  listSelection?: ListSelectionState;
  askNumber?: AskNumberState;
  askStyle?: AskStyleState;
  askSlideMenu?: AskSlideMenuState;
  transportAction?: TransportActionState;
  partyQuestAction?: PartyQuestActionState;
  partyQuestBonusAction?: PartyQuestBonusActionState;
  gachaponAction?: GachaponActionState;
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