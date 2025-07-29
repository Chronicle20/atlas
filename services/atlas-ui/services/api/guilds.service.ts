/**
 * Guild management service
 * Handles all guild-related API operations
 */
import { BaseService } from './base.service';
import type { Guild } from '@/types/models/guild';

class GuildsService extends BaseService {
  protected basePath = '/guilds';

  /**
   * Get all guilds
   */
  async getAll(signal?: AbortSignal): Promise<Guild[]> {
    return super.getAll<Guild>(signal);
  }

  /**
   * Get guild by ID
   */
  async getById(id: string, signal?: AbortSignal): Promise<Guild> {
    return super.getById<Guild>(id, signal);
  }

  /**
   * Create new guild
   */
  async create(data: Partial<Guild>, signal?: AbortSignal): Promise<Guild> {
    return super.create<Guild, Partial<Guild>>(data, signal);
  }

  /**
   * Update existing guild
   */
  async update(id: string, data: Partial<Guild>, signal?: AbortSignal): Promise<Guild> {
    return super.update<Guild, Partial<Guild>>(id, data, signal);
  }

  /**
   * Delete guild
   */
  async delete(id: string, signal?: AbortSignal): Promise<void> {
    return super.delete(id, signal);
  }
}

export const guildsService = new GuildsService();