/**
 * NPC conversation service
 * Handles all NPC conversation-related API operations with full API client feature support
 */
import { BaseService, type QueryOptions } from './base.service';
import type { 
  Conversation, 
  ConversationAttributes,
  DialogueChoice,
  DialogueState,
  GenericActionOperation,
  Condition,
  GenericActionOutcome,
  GenericActionState,
  CraftActionState,
  ListSelectionState,
  ConversationState
} from '@/types/models/conversation';

/**
 * Request/response interfaces for API communication
 */
export interface ConversationCreateRequest {
  data: {
    type: "conversations";
    attributes: ConversationAttributes;
  };
}

export interface ConversationUpdateRequest {
  data: {
    type: "conversations";
    id: string;
    attributes: Partial<ConversationAttributes>;
  };
}

export interface ConversationResponse {
  data: Conversation;
}

export interface ConversationsResponse {
  data: Conversation[];
}

/**
 * Conversation service class extending BaseService
 * Provides comprehensive conversation management with specialized NPC operations
 */
class ConversationsService extends BaseService {
  protected basePath = '/api/npcs/conversations';

  /**
   * Transform request data to API format
   */
  protected override transformRequest<T>(data: T): T {
    // For create/update operations, wrap data in the expected format
    if (data && typeof data === 'object' && 'npcId' in data) {
      return {
        data: {
          type: "conversations",
          attributes: data,
          ...(('id' in data && data.id) ? { id: data.id } : {})
        }
      } as T;
    }
    return data;
  }

  /**
   * Transform API response to domain model format
   */
  protected override transformResponse<T>(data: T): T {
    // Extract data from API response wrapper if present
    if (data && typeof data === 'object' && 'data' in data) {
      return (data as any).data;
    }
    return data;
  }

  /**
   * Validate conversation data before API calls
   */
  protected override validate(data: unknown): Array<{ field: string; message: string; value?: unknown }> {
    const errors: Array<{ field: string; message: string; value?: unknown }> = [];
    
    if (!data || typeof data !== 'object') {
      errors.push({ field: 'root', message: 'Conversation data is required' });
      return errors;
    }

    const conversation = data as Partial<ConversationAttributes>;

    // Validate required fields
    if (!conversation.npcId) {
      errors.push({ field: 'npcId', message: 'NPC ID is required', value: conversation.npcId });
    } else if (typeof conversation.npcId !== 'number' || conversation.npcId <= 0) {
      errors.push({ field: 'npcId', message: 'NPC ID must be a positive number', value: conversation.npcId });
    }

    if (!conversation.startState) {
      errors.push({ field: 'startState', message: 'Start state is required', value: conversation.startState });
    }

    if (!Array.isArray(conversation.states) || conversation.states.length === 0) {
      errors.push({ field: 'states', message: 'At least one conversation state is required', value: conversation.states });
    } else {
      // Validate each state
      conversation.states.forEach((state, index) => {
        if (!state.id) {
          errors.push({ field: `states[${index}].id`, message: 'State ID is required', value: state.id });
        }
        if (!state.type || !['dialogue', 'genericAction', 'craftAction', 'listSelection'].includes(state.type)) {
          errors.push({ 
            field: `states[${index}].type`, 
            message: 'State type must be dialogue, genericAction, craftAction, or listSelection', 
            value: state.type 
          });
        }
      });

      // Validate that startState exists in states
      const stateIds = conversation.states.map(s => s.id);
      if (conversation.startState && !stateIds.includes(conversation.startState)) {
        errors.push({ 
          field: 'startState', 
          message: 'Start state must exist in the states array', 
          value: conversation.startState 
        });
      }
    }

    return errors;
  }

  /**
   * Get all conversations with advanced query support
   */
  override async getAll<T = Conversation>(options?: QueryOptions): Promise<T[]> {
    return super.getAll<T>(options);
  }

  /**
   * Get conversation by ID
   */
  override async getById<T = Conversation>(id: string, options?: Parameters<BaseService['getById']>[1]): Promise<T> {
    return super.getById<T>(id, options);
  }

  /**
   * Check if conversation exists
   */
  override async exists(id: string, options?: Parameters<BaseService['exists']>[1]): Promise<boolean> {
    return super.exists(id, options);
  }

  /**
   * Create new conversation with validation
   */
  override async create<T = Conversation, D = ConversationAttributes>(data: D, options?: Parameters<BaseService['create']>[1]): Promise<T> {
    return super.create<T, D>(data, { validate: true, ...options });
  }

  /**
   * Update existing conversation with validation
   */
  async update(id: string, conversationAttributes: Partial<ConversationAttributes>, options?: Parameters<BaseService['update']>[2]): Promise<Conversation> {
    return super.update<Conversation, Partial<ConversationAttributes>>(id, conversationAttributes, { validate: true, ...options });
  }

  /**
   * Partially update conversation (PATCH)
   */
  async patch(id: string, updates: Partial<ConversationAttributes>, options?: Parameters<BaseService['patch']>[2]): Promise<Conversation> {
    return super.patch<Conversation, Partial<ConversationAttributes>>(id, updates, options);
  }

  /**
   * Delete conversation
   */
  async delete(id: string, options?: Parameters<BaseService['delete']>[1]): Promise<void> {
    return super.delete(id, options);
  }

  /**
   * Get conversations for a specific NPC
   * Returns the first conversation if one exists, otherwise returns null
   */
  async getByNpcId(npcId: number, options?: Parameters<BaseService['getAll']>[0]): Promise<Conversation | null> {
    try {
      const url = `/api/npcs/${npcId}/conversations`;
      const processedOptions = options ? { ...options, useCache: options.useCache !== false } : { useCache: true };
      
      // Override the basePath temporarily for this specific request
      const originalBasePath = this.basePath;
      (this as any).basePath = '';
      
      const conversations = await super.getAll<Conversation>({ ...processedOptions });
      
      // Restore original basePath
      (this as any).basePath = originalBasePath;
      
      return conversations.length > 0 ? conversations[0] : null;
    } catch (error) {
      // If it's a 404 error, no conversations exist for this NPC
      if (error && typeof error === 'object' && 'status' in error && error.status === 404) {
        return null;
      }
      throw error;
    }
  }

  /**
   * Batch create multiple conversations
   */
  async createBatch(
    conversations: ConversationAttributes[], 
    options?: Parameters<BaseService['createBatch']>[1],
    batchOptions?: Parameters<BaseService['createBatch']>[2]
  ) {
    return super.createBatch<Conversation, ConversationAttributes>(
      conversations, 
      { validate: true, ...options },
      batchOptions
    );
  }

  /**
   * Batch update multiple conversations
   */
  async updateBatch(
    updates: Array<{ id: string; data: Partial<ConversationAttributes> }>,
    options?: Parameters<BaseService['updateBatch']>[1],
    batchOptions?: Parameters<BaseService['updateBatch']>[2]
  ) {
    return super.updateBatch<Conversation, Partial<ConversationAttributes>>(
      updates,
      { validate: true, ...options },
      batchOptions
    );
  }

  /**
   * Batch delete multiple conversations
   */
  async deleteBatch(
    ids: string[],
    options?: Parameters<BaseService['deleteBatch']>[1],
    batchOptions?: Parameters<BaseService['deleteBatch']>[2]
  ) {
    return super.deleteBatch(ids, options, batchOptions);
  }

  /**
   * Search conversations by text content in dialogue states
   */
  async searchByText(searchText: string, options?: Parameters<BaseService['getAll']>[0]): Promise<Conversation[]> {
    return this.getAll({
      ...options,
      search: searchText,
      filters: {
        ...options?.filters,
        searchFields: 'states.dialogue.text,states.listSelection.title'
      }
    });
  }

  /**
   * Get conversations by NPC ID with filtering
   */
  async getConversationsByNpc(npcId: number, options?: Parameters<BaseService['getAll']>[0]): Promise<Conversation[]> {
    return this.getAll({
      ...options,
      filters: {
        ...options?.filters,
        npcId: npcId
      }
    });
  }

  /**
   * Export conversation data (useful for backups or migrations)  
   */
  async export(format: 'json' | 'csv' = 'json', options?: Parameters<BaseService['getAll']>[0]): Promise<Blob> {
    const conversations = await this.getAll(options);
    
    let content: string;
    let mimeType: string;
    
    if (format === 'csv') {
      // Convert to CSV format
      const headers = ['ID', 'NPC ID', 'Start State', 'States Count', 'Created At'];
      const rows = conversations.map(conv => [
        conv.id,
        conv.attributes.npcId.toString(),
        conv.attributes.startState,
        conv.attributes.states.length.toString(),
        // Assuming there might be timestamps in the future
        new Date().toISOString()
      ]);
      
      content = [headers, ...rows].map(row => row.join(',')).join('\n');
      mimeType = 'text/csv';
    } else {
      content = JSON.stringify(conversations, null, 2);
      mimeType = 'application/json';
    }
    
    return new Blob([content], { type: mimeType });
  }

  /**
   * Validate conversation state consistency
   * Checks that all state references are valid and reachable
   */
  async validateStateConsistency(conversationId: string): Promise<{ isValid: boolean; errors: string[] }> {
    const conversation = await this.getById(conversationId);
    const errors: string[] = [];
    
    const stateIds = new Set(conversation.attributes.states.map(s => s.id));
    const reachableStates = new Set<string>();
    
    // Check if start state exists
    if (!stateIds.has(conversation.attributes.startState)) {
      errors.push(`Start state '${conversation.attributes.startState}' does not exist`);
    } else {
      reachableStates.add(conversation.attributes.startState);
    }
    
    // Check all state transitions
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
        if (!stateIds.has(state.craftAction.successState)) {
          errors.push(`Craft action in state '${state.id}' references non-existent success state '${state.craftAction.successState}'`);
        } else {
          reachableStates.add(state.craftAction.successState);
        }
        
        if (!stateIds.has(state.craftAction.failureState)) {
          errors.push(`Craft action in state '${state.id}' references non-existent failure state '${state.craftAction.failureState}'`);
        } else {
          reachableStates.add(state.craftAction.failureState);
        }
        
        if (!stateIds.has(state.craftAction.missingMaterialsState)) {
          errors.push(`Craft action in state '${state.id}' references non-existent missing materials state '${state.craftAction.missingMaterialsState}'`);
        } else {
          reachableStates.add(state.craftAction.missingMaterialsState);
        }
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
    
    // Check for unreachable states
    stateIds.forEach(stateId => {
      if (!reachableStates.has(stateId)) {
        errors.push(`State '${stateId}' is unreachable`);
      }
    });
    
    return {
      isValid: errors.length === 0,
      errors
    };
  }

  /**
   * Clear conversations cache
   */
  clearCache(): void {
    super.clearServiceCache();
  }

  /**
   * Get cache statistics for conversations
   */
  getCacheStats() {
    return super.getServiceCacheStats();
  }
}

// Export singleton instance
export const conversationsService = new ConversationsService();