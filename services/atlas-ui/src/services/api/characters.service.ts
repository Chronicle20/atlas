/**
 * Character management service
 * Handles all character-related API operations with tenant support
 */
import type { ServiceOptions } from '@/lib/api/query-params';
import type { Character, UpdateCharacterData } from '@/types/models/character';
import { api } from '@/lib/api/client';
import { tenantHeaders } from '@/lib/headers';
import type { Tenant } from '@/types/models/tenant';

export interface NameValidityResponse {
  valid: boolean;
  reason?: 'regex' | 'length' | 'blocked' | 'duplicate';
  detail?: string;
}

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

  /**
   * GET /api/characters/name-validity?name=&worldId=
   *
   * atlas-character is the authority on character names (per task-037 design D-6).
   * Returns plain JSON {valid, reason?, detail?} — not JSON:API. Tenant headers
   * are passed explicitly so the call works regardless of singleton client state.
   */
  async checkNameValidity(
    tenant: Tenant,
    name: string,
    worldId: number,
  ): Promise<NameValidityResponse> {
    const params = new URLSearchParams({ name, worldId: String(worldId) });
    return api.get<NameValidityResponse>(
      `${this.basePath}/name-validity?${params.toString()}`,
      { headers: tenantHeaders(tenant) },
    );
  }

}

export const charactersService = new CharactersService();

// Export the service class for potential extension
export { CharactersService };