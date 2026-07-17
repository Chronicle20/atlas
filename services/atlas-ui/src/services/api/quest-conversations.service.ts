import { api } from "@/lib/api/client";
import { type ServiceOptions } from "@/lib/api/query-params";
import { fetchAll } from "@/services/api/pagination";
import type {
  QuestConversation,
  QuestConversationAttributes,
  QuestConversationResponse,
} from "@/types/models/conversation";

const BASE_PATH = "/api/quests/conversations";

export interface QuestConversationUpdateRequest {
  data: {
    type: "quest-conversations";
    id: string;
    attributes: Partial<QuestConversationAttributes>;
  };
}

function wrap(
  attributes: Partial<QuestConversationAttributes>,
  id: string,
): QuestConversationUpdateRequest {
  return {
    data: { type: "quest-conversations", id, attributes },
  };
}

export const questConversationsService = {
  /**
   * Get every quest conversation, draining all pages (task-117).
   */
  async getAll(options?: ServiceOptions): Promise<QuestConversation[]> {
    return fetchAll<QuestConversation>(BASE_PATH, undefined, options);
  },

  async getById(id: string, options?: ServiceOptions): Promise<QuestConversation> {
    return api.getOne<QuestConversation>(`${BASE_PATH}/${id}`, options);
  },

  async getByQuestId(
    questId: number,
    options?: ServiceOptions,
  ): Promise<QuestConversation | null> {
    try {
      return await api.getOne<QuestConversation>(
        `/api/quests/${questId}/conversation`,
        options,
      );
    } catch (error) {
      if (
        error &&
        typeof error === "object" &&
        "statusCode" in error &&
        (error as { statusCode: number }).statusCode === 404
      ) {
        return null;
      }
      throw error;
    }
  },

  async update(
    id: string,
    attributes: Partial<QuestConversationAttributes>,
    options?: ServiceOptions,
  ): Promise<QuestConversation> {
    const response = await api.patch<QuestConversationResponse>(
      `${BASE_PATH}/${id}`,
      wrap(attributes, id),
      options,
    );
    return response.data;
  },

  async delete(id: string, options?: ServiceOptions): Promise<void> {
    return api.delete(`${BASE_PATH}/${id}`, options);
  },
};
