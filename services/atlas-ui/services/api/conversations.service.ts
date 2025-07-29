/**
 * NPC conversation service
 * Handles all NPC conversation-related API operations
 */
import { BaseService } from './base.service';
import type { Conversation } from '@/types/models/conversation';

class ConversationsService extends BaseService {
  protected basePath = '/conversations';

  /**
   * Get all conversations
   */
  async getAll(signal?: AbortSignal): Promise<Conversation[]> {
    return super.getAll<Conversation>(signal);
  }

  /**
   * Get conversation by ID
   */
  async getById(id: string, signal?: AbortSignal): Promise<Conversation> {
    return super.getById<Conversation>(id, signal);
  }

  /**
   * Create new conversation
   */
  async create(data: Partial<Conversation>, signal?: AbortSignal): Promise<Conversation> {
    return super.create<Conversation, Partial<Conversation>>(data, signal);
  }

  /**
   * Update existing conversation
   */
  async update(id: string, data: Partial<Conversation>, signal?: AbortSignal): Promise<Conversation> {
    return super.update<Conversation, Partial<Conversation>>(id, data, signal);
  }

  /**
   * Delete conversation
   */
  async delete(id: string, signal?: AbortSignal): Promise<void> {
    return super.delete(id, signal);
  }
}

export const conversationsService = new ConversationsService();