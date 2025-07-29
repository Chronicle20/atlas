/**
 * Character management service
 * Handles all character-related API operations
 */
import { BaseService } from './base.service';
import type { Character, CreateCharacterDto } from '@/types/models/character';

class CharactersService extends BaseService {
  protected basePath = '/characters';

  /**
   * Get all characters
   */
  async getAll(signal?: AbortSignal): Promise<Character[]> {
    return super.getAll<Character>(signal);
  }

  /**
   * Get character by ID
   */
  async getById(id: string, signal?: AbortSignal): Promise<Character> {
    return super.getById<Character>(id, signal);
  }

  /**
   * Create new character
   */
  async create(data: CreateCharacterDto, signal?: AbortSignal): Promise<Character> {
    return super.create<Character, CreateCharacterDto>(data, signal);
  }

  /**
   * Update existing character
   */
  async update(id: string, data: Partial<Character>, signal?: AbortSignal): Promise<Character> {
    return super.update<Character, Partial<Character>>(id, data, signal);
  }

  /**
   * Delete character
   */
  async delete(id: string, signal?: AbortSignal): Promise<void> {
    return super.delete(id, signal);
  }
}

export const charactersService = new CharactersService();