/**
 * Character management service
 * Handles all character-related API operations with tenant support
 */
import type { ServiceOptions } from '@/lib/api/query-params';
import type { Character, UpdateCharacterData } from '@/types/models/character';
import { api } from '@/lib/api/client';

class CharactersService {
  private basePath = '/api/characters';

  /**
   * Get all characters for a tenant
   */
  async getAll(options?: ServiceOptions): Promise<Character[]> {
    // Set tenant for this request
    // Use the API client to fetch characters
    return api.getList<Character>(this.basePath, options);
  }

  /**
   * Get character by ID for a tenant
   */
  async getById(characterId: string, options?: ServiceOptions): Promise<Character> {
    // Set tenant for this request
    // Use the API client to fetch a single character
    return api.getOne<Character>(`${this.basePath}/${characterId}`, options);
  }

  /**
   * Delete a character permanently
   */
  async deleteCharacter(characterId: string, options?: ServiceOptions): Promise<void> {
    return api.delete(`${this.basePath}/${characterId}`, options);
  }

  /**
   * Update existing character with JSON:API format
   */
  async update(characterId: string, data: UpdateCharacterData, options?: ServiceOptions): Promise<void> {
    // Set tenant for this request
    // Prepare the JSON:API formatted request body
    const requestBody = {
      data: {
        type: "characters",
        id: characterId,
        attributes: data,
      },
    };
    
    // Use the centralized API client to update the character
    // The API client handles all error cases and status codes automatically
    return api.patch<void>(`/api/characters/${characterId}`, requestBody, options);
  }

}

export const charactersService = new CharactersService();

// Export the service class for potential extension
export { CharactersService };