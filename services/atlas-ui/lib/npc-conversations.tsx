import type {Tenant} from "@/types/models/tenant";
import {tenantHeaders} from "@/lib/headers";

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
  const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
  const response = await fetch(rootUrl + "/api/npcs/conversations", {
    method: "GET",
    headers: tenantHeaders(tenant),
  });
  if (!response.ok) {
    throw new Error("Failed to fetch NPC conversations.");
  }
  const responseData: ConversationsResponse = await response.json();
  return responseData.data;
}

export async function fetchConversation(tenant: Tenant, conversationId: string): Promise<Conversation> {
  const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
  const response = await fetch(rootUrl + "/api/npcs/conversations/" + conversationId, {
    method: "GET",
    headers: tenantHeaders(tenant),
  });
  if (!response.ok) {
    throw new Error("Failed to fetch NPC conversation.");
  }
  const responseData: ConversationResponse = await response.json();
  return responseData.data;
}

export async function createConversation(tenant: Tenant, conversationAttributes: ConversationAttributes): Promise<Conversation> {
  const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
  const response = await fetch(rootUrl + "/api/npcs/conversations", {
    method: "POST",
    headers: tenantHeaders(tenant),
    body: JSON.stringify({
      data: {
        type: "conversations",
        attributes: conversationAttributes
      }
    }),
  });
  if (!response.ok) {
    throw new Error("Failed to create NPC conversation.");
  }
  const responseData: ConversationResponse = await response.json();
  return responseData.data;
}

export async function updateConversation(tenant: Tenant, conversationId: string, conversationAttributes: Partial<ConversationAttributes>): Promise<Conversation> {
  const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
  const response = await fetch(rootUrl + "/api/npcs/conversations/" + conversationId, {
    method: "PATCH",
    headers: tenantHeaders(tenant),
    body: JSON.stringify({
      data: {
        type: "conversations",
        id: conversationId,
        attributes: conversationAttributes
      }
    }),
  });
  if (!response.ok) {
    throw new Error("Failed to update NPC conversation.");
  }
  const responseData: ConversationResponse = await response.json();
  return responseData.data;
}

export async function deleteConversation(tenant: Tenant, conversationId: string): Promise<void> {
  const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
  const response = await fetch(rootUrl + "/api/npcs/conversations/" + conversationId, {
    method: "DELETE",
    headers: tenantHeaders(tenant),
  });
  if (!response.ok) {
    throw new Error("Failed to delete NPC conversation.");
  }
}

export async function fetchNPCConversations(tenant: Tenant, npcId: number): Promise<Conversation | null> {
  const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
  const response = await fetch(rootUrl + "/api/npcs/" + npcId + "/conversations", {
    method: "GET",
    headers: tenantHeaders(tenant),
  });
  if (!response.ok) {
    throw new Error("Failed to fetch NPC conversations.");
  }
  const responseData: ConversationsResponse = await response.json();
  // Return the first conversation if one exists, otherwise return null
  return responseData.data.length > 0 ? responseData.data[0] : null;
}
