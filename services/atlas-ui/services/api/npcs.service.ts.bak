/**
 * NPC shop service
 * Handles all NPC shop-related API operations
 */
import { BaseService } from './base.service';
import type { Npc } from '@/types/models/npc';

class NpcsService extends BaseService {
  protected basePath = '/npcs';

  /**
   * Get all NPCs
   */
  async getAll(signal?: AbortSignal): Promise<Npc[]> {
    return super.getAll<Npc>(signal);
  }

  /**
   * Get NPC by ID
   */
  async getById(id: string, signal?: AbortSignal): Promise<Npc> {
    return super.getById<Npc>(id, signal);
  }

  /**
   * Create new NPC
   */
  async create(data: Partial<Npc>, signal?: AbortSignal): Promise<Npc> {
    return super.create<Npc, Partial<Npc>>(data, signal);
  }

  /**
   * Update existing NPC
   */
  async update(id: string, data: Partial<Npc>, signal?: AbortSignal): Promise<Npc> {
    return super.update<Npc, Partial<Npc>>(id, data, signal);
  }

  /**
   * Delete NPC
   */
  async delete(id: string, signal?: AbortSignal): Promise<void> {
    return super.delete(id, signal);
  }
}

export const npcsService = new NpcsService();