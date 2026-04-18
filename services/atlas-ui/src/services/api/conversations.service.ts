import { api } from "@/lib/api/client";
import {
  buildQueryString,
  runBatch,
  type QueryOptions,
  type ServiceOptions,
  type BatchOptions,
  type BatchResult,
  type ValidationError,
} from "@/lib/api/query-params";
import type {
  Conversation,
  ConversationAttributes,
} from "@/types/models/conversation";

const BASE_PATH = "/api/npcs/conversations";

export interface ConversationCreateRequest {
  data: { type: "conversations"; attributes: ConversationAttributes };
}

export interface ConversationUpdateRequest {
  data: { type: "conversations"; id: string; attributes: Partial<ConversationAttributes> };
}

export interface ConversationResponse {
  data: Conversation;
}

export interface ConversationsResponse {
  data: Conversation[];
}

function validateConversation(data: unknown): ValidationError[] {
  const errors: ValidationError[] = [];

  if (!data || typeof data !== "object") {
    errors.push({ field: "root", message: "Conversation data is required" });
    return errors;
  }

  const conversation = data as Partial<ConversationAttributes>;

  if (!conversation.npcId) {
    errors.push({ field: "npcId", message: "NPC ID is required", value: conversation.npcId });
  } else if (typeof conversation.npcId !== "number" || conversation.npcId <= 0) {
    errors.push({ field: "npcId", message: "NPC ID must be a positive number", value: conversation.npcId });
  }

  if (!conversation.startState) {
    errors.push({ field: "startState", message: "Start state is required", value: conversation.startState });
  }

  if (!Array.isArray(conversation.states) || conversation.states.length === 0) {
    errors.push({ field: "states", message: "At least one conversation state is required", value: conversation.states });
  } else {
    conversation.states.forEach((state, index) => {
      if (!state.id) {
        errors.push({ field: `states[${index}].id`, message: "State ID is required", value: state.id });
      }
      if (!state.type || !["dialogue", "genericAction", "craftAction", "listSelection"].includes(state.type)) {
        errors.push({
          field: `states[${index}].type`,
          message: "State type must be dialogue, genericAction, craftAction, or listSelection",
          value: state.type,
        });
      }
    });

    const stateIds = conversation.states.map(s => s.id);
    if (conversation.startState && !stateIds.includes(conversation.startState)) {
      errors.push({
        field: "startState",
        message: "Start state must exist in the states array",
        value: conversation.startState,
      });
    }
  }

  return errors;
}

function throwIfInvalid(data: unknown, shouldValidate: boolean): void {
  if (!shouldValidate) return;
  const errors = validateConversation(data);
  if (errors.length > 0) {
    throw new Error(`Conversation validation failed: ${errors.map(e => e.message).join(", ")}`);
  }
}

function wrapConversation(attributes: ConversationAttributes, id?: string): ConversationCreateRequest | ConversationUpdateRequest {
  return {
    data: { type: "conversations" as const, attributes, ...(id ? { id } : {}) },
  } as ConversationCreateRequest | ConversationUpdateRequest;
}

export const conversationsService = {
  async getAll(options?: QueryOptions): Promise<Conversation[]> {
    return api.getList<Conversation>(`${BASE_PATH}${buildQueryString(options)}`, options);
  },

  async getById(id: string, options?: ServiceOptions): Promise<Conversation> {
    return api.getOne<Conversation>(`${BASE_PATH}/${id}`, options);
  },

  async exists(id: string, options?: ServiceOptions): Promise<boolean> {
    try {
      await conversationsService.getById(id, options);
      return true;
    } catch (error) {
      if (error && typeof error === "object" && "status" in error && (error as { status: number }).status === 404) {
        return false;
      }
      throw error;
    }
  },

  async create(data: ConversationAttributes, options?: ServiceOptions): Promise<Conversation> {
    throwIfInvalid(data, options?.validate !== false);
    const response = await api.post<ConversationResponse>(BASE_PATH, wrapConversation(data), options);
    return response.data;
  },

  async update(id: string, data: Partial<ConversationAttributes>, options?: ServiceOptions): Promise<Conversation> {
    throwIfInvalid(data, options?.validate !== false);
    const response = await api.put<ConversationResponse>(`${BASE_PATH}/${id}`, wrapConversation(data as ConversationAttributes, id), options);
    return response.data;
  },

  async patch(id: string, data: Partial<ConversationAttributes>, options?: ServiceOptions): Promise<Conversation> {
    const response = await api.patch<ConversationResponse>(`${BASE_PATH}/${id}`, wrapConversation(data as ConversationAttributes, id), options);
    return response.data;
  },

  async delete(id: string, options?: ServiceOptions): Promise<void> {
    return api.delete(`${BASE_PATH}/${id}`, options);
  },

  async getByNpcId(npcId: number, options?: QueryOptions): Promise<Conversation | null> {
    try {
      const data = await api.getList<Conversation>(
        `/api/npcs/${npcId}/conversations${buildQueryString(options)}`,
        options,
      );
      return data.length > 0 ? data[0]! : null;
    } catch (error) {
      if (error && typeof error === "object" && "status" in error && (error as { status: number }).status === 404) {
        return null;
      }
      throw error;
    }
  },

  async createBatch(
    items: ConversationAttributes[],
    options?: ServiceOptions,
    batchOptions?: BatchOptions,
  ): Promise<BatchResult<Conversation>> {
    return runBatch(items, item => conversationsService.create(item, options), batchOptions);
  },

  async updateBatch(
    updates: Array<{ id: string; data: Partial<ConversationAttributes> }>,
    options?: ServiceOptions,
    batchOptions?: BatchOptions,
  ): Promise<BatchResult<Conversation>> {
    return runBatch(updates, ({ id, data }) => conversationsService.update(id, data, options), batchOptions);
  },

  async deleteBatch(
    ids: string[],
    options?: ServiceOptions,
    batchOptions?: BatchOptions,
  ): Promise<BatchResult<string>> {
    return runBatch(ids, async id => {
      await conversationsService.delete(id, options);
      return id;
    }, batchOptions);
  },

  async searchByText(searchText: string, options?: QueryOptions): Promise<Conversation[]> {
    return conversationsService.getAll({
      ...options,
      search: searchText,
      filters: {
        ...options?.filters,
        searchFields: "states.dialogue.text,states.listSelection.title",
      },
    });
  },

  async getConversationsByNpc(npcId: number, options?: QueryOptions): Promise<Conversation[]> {
    return conversationsService.getAll({
      ...options,
      filters: { ...options?.filters, npcId },
    });
  },

  async export(format: "json" | "csv" = "json", options?: QueryOptions): Promise<Blob> {
    const conversations = await conversationsService.getAll(options);

    if (format === "csv") {
      const headers = ["ID", "NPC ID", "Start State", "States Count", "Created At"];
      const rows = conversations.map(conv => [
        conv.id,
        conv.attributes.npcId.toString(),
        conv.attributes.startState,
        conv.attributes.states.length.toString(),
        new Date().toISOString(),
      ]);
      const content = [headers, ...rows].map(row => row.join(",")).join("\n");
      return new Blob([content], { type: "text/csv" });
    }

    return new Blob([JSON.stringify(conversations, null, 2)], { type: "application/json" });
  },

  async validateStateConsistency(conversationId: string): Promise<{ isValid: boolean; errors: string[] }> {
    const conversation = await conversationsService.getById(conversationId);
    const errors: string[] = [];
    const stateIds = new Set(conversation.attributes.states.map(s => s.id));
    const reachableStates = new Set<string>();

    if (!stateIds.has(conversation.attributes.startState)) {
      errors.push(`Start state '${conversation.attributes.startState}' does not exist`);
    } else {
      reachableStates.add(conversation.attributes.startState);
    }

    conversation.attributes.states.forEach(state => {
      if (state.dialogue) {
        state.dialogue.choices.forEach(choice => {
          if (choice.nextState && !stateIds.has(choice.nextState)) {
            errors.push(`State '${state.id}' references non-existent state '${choice.nextState}'`);
          } else if (choice.nextState) {
            reachableStates.add(choice.nextState);
          }
        });
      }
      if (state.genericAction) {
        state.genericAction.outcomes.forEach(outcome => {
          if (!stateIds.has(outcome.nextState)) {
            errors.push(`State '${state.id}' references non-existent state '${outcome.nextState}'`);
          } else {
            reachableStates.add(outcome.nextState);
          }
        });
      }
      if (state.craftAction) {
        const craft = state.craftAction;
        if (!stateIds.has(craft.successState)) {
          errors.push(`Craft action in state '${state.id}' references non-existent success state '${craft.successState}'`);
        } else reachableStates.add(craft.successState);
        if (!stateIds.has(craft.failureState)) {
          errors.push(`Craft action in state '${state.id}' references non-existent failure state '${craft.failureState}'`);
        } else reachableStates.add(craft.failureState);
        if (!stateIds.has(craft.missingMaterialsState)) {
          errors.push(`Craft action in state '${state.id}' references non-existent missing materials state '${craft.missingMaterialsState}'`);
        } else reachableStates.add(craft.missingMaterialsState);
      }
      if (state.listSelection) {
        state.listSelection.choices.forEach(choice => {
          if (choice.nextState && !stateIds.has(choice.nextState)) {
            errors.push(`List selection in state '${state.id}' references non-existent state '${choice.nextState}'`);
          } else if (choice.nextState) {
            reachableStates.add(choice.nextState);
          }
        });
      }
    });

    stateIds.forEach(stateId => {
      if (!reachableStates.has(stateId)) {
        errors.push(`State '${stateId}' is unreachable`);
      }
    });

    return { isValid: errors.length === 0, errors };
  },

  async validateConversation(conversation: ConversationAttributes): Promise<void> {
    await api.post<void>(`${BASE_PATH}/validate`, {
      data: { type: "conversations", attributes: conversation },
    });
  },

  async seedConversations(): Promise<void> {
    await api.post<void>(`${BASE_PATH}/seed`, {});
  },
};
