import type {Tenant} from "@/types/models/tenant";
import { api } from "@/lib/api/client";

// Define interfaces based on the NPC conversation schema
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

// API client functions for NPC conversations
export async function fetchConversations(tenant: Tenant): Promise<Conversation[]> {
  api.setTenant(tenant);
  const response = await api.get<ConversationsResponse>('/api/npcs/conversations');
  return response.data;
}

export async function fetchConversation(tenant: Tenant, conversationId: string): Promise<Conversation> {
  api.setTenant(tenant);
  const response = await api.get<ConversationResponse>(`/api/npcs/conversations/${conversationId}`);
  return response.data;
}

export async function createConversation(tenant: Tenant, conversationAttributes: ConversationAttributes): Promise<Conversation> {
  api.setTenant(tenant);
  const response = await api.post<ConversationResponse>('/api/npcs/conversations', {
    data: {
      type: "conversations",
      attributes: conversationAttributes
    }
  });
  return response.data;
}

export async function updateConversation(tenant: Tenant, conversationId: string, conversationAttributes: Partial<ConversationAttributes>): Promise<Conversation> {
  api.setTenant(tenant);
  const response = await api.patch<ConversationResponse>(`/api/npcs/conversations/${conversationId}`, {
    data: {
      type: "conversations",
      id: conversationId,
      attributes: conversationAttributes
    }
  });
  return response.data;
}

export async function deleteConversation(tenant: Tenant, conversationId: string): Promise<void> {
  api.setTenant(tenant);
  return api.delete<void>(`/api/npcs/conversations/${conversationId}`);
}

export async function fetchNPCConversations(tenant: Tenant, npcId: number): Promise<Conversation | null> {
  api.setTenant(tenant);
  const response = await api.get<ConversationsResponse>(`/api/npcs/${npcId}/conversations`);
  // Return the first conversation if one exists, otherwise return null
  return response.data.length > 0 ? (response.data[0] || null) : null;
}
