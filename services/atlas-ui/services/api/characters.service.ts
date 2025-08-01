/**
 * Character management service
 * Handles all character-related API operations with tenant support
 */
import type { ServiceOptions } from './base.service';
import type { Character, UpdateCharacterData } from '@/types/models/character';
import type { Tenant } from '@/types/models/tenant';
import { api } from '@/lib/api/client';

class CharactersService {
  private basePath = '/api/characters';

  /**
   * Get all characters for a tenant
   */
  async getAll(tenant: Tenant, options?: ServiceOptions): Promise<Character[]> {
    // Set tenant for this request
    api.setTenant(tenant);
    
    // Use the API client to fetch characters
    return api.getList<Character>(this.basePath, options);
  }

  /**
   * Get character by ID for a tenant
   */
  async getById(tenant: Tenant, characterId: string, options?: ServiceOptions): Promise<Character> {
    // Set tenant for this request
    api.setTenant(tenant);
    
    // Use the API client to fetch a single character
    return api.getOne<Character>(`${this.basePath}/${characterId}`, options);
  }

  /**
   * Update existing character with JSON:API format
   */
  async update(tenant: Tenant, characterId: string, data: UpdateCharacterData, options?: ServiceOptions): Promise<void> {
    // Set tenant for this request
    api.setTenant(tenant);
    
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