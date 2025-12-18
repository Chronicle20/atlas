/**
 * Guilds Service
 * 
 * Provides comprehensive guild management functionality including:
 * - Basic guild CRUD operations matching the original API
 * - Guild member management
 * - Enhanced error handling with tenant support
 * - Guild ranking and search capabilities
 */

import type { ServiceOptions } from './base.service';
import type { Guild, GuildAttributes, GuildMember } from '@/types/models/guild';
import type { Tenant } from '@/types/models/tenant';
import { api } from '@/lib/api/client';

// Guild search filters
interface GuildFilters {
  worldId?: number;
  name?: string;
  minLevel?: number;
  maxLevel?: number;
  hasSpace?: boolean;
  leaderId?: number;
}

/**
 * Guilds service class with tenant-aware API operations
 */
class GuildsService {
  private basePath = '/api/guilds';

  /**
   * Get all guilds for a tenant
   */
  async getAll(tenant: Tenant, options?: ServiceOptions): Promise<Guild[]> {
    // Set tenant for this request
    api.setTenant(tenant);
    
    // Use the API client to fetch guilds
    const guilds = await api.getList<Guild>(this.basePath, options);
    
    // Sort guilds by points (descending) and then by name
    return this.sortGuilds(guilds);
  }

  /**
   * Get guild by ID for a tenant
   */
  async getById(tenant: Tenant, guildId: string, options?: ServiceOptions): Promise<Guild> {
    // Set tenant for this request
    api.setTenant(tenant);
    
    // Use the API client to fetch a single guild
    const guild = await api.getOne<Guild>(`${this.basePath}/${guildId}`, options);
    
    // Transform response to sort members and titles
    return this.transformGuildResponse(guild);
  }

  /**
   * Get guilds by world ID
   */
  async getByWorld(tenant: Tenant, worldId: number, options?: ServiceOptions): Promise<Guild[]> {
    const guilds = await this.getAll(tenant, options);
    return guilds.filter(guild => guild.attributes.worldId === worldId);
  }

  /**
   * Search guilds by name
   */
  async search(tenant: Tenant, searchTerm: string, worldId?: number, options?: ServiceOptions): Promise<Guild[]> {
    const guilds = await this.getAll(tenant, options);
    
    let filtered = guilds.filter(guild => 
      guild.attributes.name.toLowerCase().includes(searchTerm.toLowerCase())
    );
    
    if (worldId !== undefined) {
      filtered = filtered.filter(guild => guild.attributes.worldId === worldId);
    }
    
    return filtered;
  }

  /**
   * Get guilds with available space
   */
  async getWithSpace(tenant: Tenant, worldId?: number, options?: ServiceOptions): Promise<Guild[]> {
    const guilds = await this.getAll(tenant, options);
    
    let filtered = guilds.filter(guild => 
      guild.attributes.members.length < guild.attributes.capacity
    );
    
    if (worldId !== undefined) {
      filtered = filtered.filter(guild => guild.attributes.worldId === worldId);
    }
    
    return filtered;
  }

  /**
   * Get guild rankings (top guilds by points)
   */
  async getRankings(tenant: Tenant, worldId?: number, limit = 50, options?: ServiceOptions): Promise<Guild[]> {
    let guilds = await this.getAll(tenant, options);
    
    if (worldId !== undefined) {
      guilds = guilds.filter(guild => guild.attributes.worldId === worldId);
    }
    
    // Already sorted by points, just take the top N
    return guilds.slice(0, limit);
  }

  /**
   * Create a new guild
   */
  async create(tenant: Tenant, attributes: GuildAttributes, options?: ServiceOptions): Promise<Guild> {
    // Set tenant for this request
    api.setTenant(tenant);
    
    // Prepare the JSON:API formatted request body
    const requestBody = {
      data: {
        type: 'guilds',
        attributes,
      },
    };

    const response = await api.post<{ data: Guild }>(this.basePath, requestBody, options);
    return this.transformGuildResponse(response.data);
  }

  /**
   * Update an existing guild
   */
  async update(
    tenant: Tenant,
    guildId: string,
    updatedAttributes: Partial<GuildAttributes>,
    options?: ServiceOptions
  ): Promise<Guild> {
    // Set tenant for this request
    api.setTenant(tenant);
    
    // Prepare the JSON:API formatted request body
    const requestBody = {
      data: {
        id: guildId,
        type: 'guilds',
        attributes: updatedAttributes,
      },
    };

    await api.patch(`${this.basePath}/${guildId}`, requestBody, options);
    
    // Return updated guild by fetching it
    return this.getById(tenant, guildId, options);
  }

  /**
   * Update guild notice
   */
  async updateNotice(tenant: Tenant, guildId: string, notice: string, options?: ServiceOptions): Promise<Guild> {
    return this.update(tenant, guildId, { notice }, options);
  }

  /**
   * Add member to guild
   */
  async addMember(tenant: Tenant, guildId: string, member: GuildMember, options?: ServiceOptions): Promise<Guild> {
    const guild = await this.getById(tenant, guildId, options);
    const updatedMembers = [...guild.attributes.members, member];
    return this.update(tenant, guildId, { members: updatedMembers }, options);
  }

  /**
   * Remove member from guild
   */
  async removeMember(tenant: Tenant, guildId: string, characterId: number, options?: ServiceOptions): Promise<Guild> {
    const guild = await this.getById(tenant, guildId, options);
    const updatedMembers = guild.attributes.members.filter(member => member.characterId !== characterId);
    return this.update(tenant, guildId, { members: updatedMembers }, options);
  }

  /**
   * Update member's title in guild
   */
  async updateMemberTitle(
    tenant: Tenant,
    guildId: string, 
    characterId: number, 
    newTitle: number, 
    options?: ServiceOptions
  ): Promise<Guild> {
    const guild = await this.getById(tenant, guildId, options);
    const updatedMembers = guild.attributes.members.map(member =>
      member.characterId === characterId ? { ...member, title: newTitle } : member
    );
    return this.update(tenant, guildId, { members: updatedMembers }, options);
  }

  /**
   * Delete a guild
   */
  async delete(tenant: Tenant, guildId: string, options?: ServiceOptions): Promise<void> {
    // Set tenant for this request
    api.setTenant(tenant);
    
    return api.delete(`${this.basePath}/${guildId}`, options);
  }

  /**
   * Check if guild exists
   */
  async exists(tenant: Tenant, guildId: string, options?: ServiceOptions): Promise<boolean> {
    try {
      await this.getById(tenant, guildId, options);
      return true;
    } catch (error) {
      // If it's a 404 error, the guild doesn't exist
      if (error && typeof error === 'object' && 'status' in error && error.status === 404) {
        return false;
      }
      // Re-throw other errors (network issues, etc.)
      throw error;
    }
  }

  /**
   * Get guild member count
   */
  async getMemberCount(tenant: Tenant, guildId: string, options?: ServiceOptions): Promise<number> {
    const guild = await this.getById(tenant, guildId, options);
    return guild.attributes.members.length;
  }

  // === PRIVATE HELPER METHODS ===

  /**
   * Sort guilds by points (descending) and then by name
   */
  private sortGuilds(guilds: Guild[]): Guild[] {
    return guilds.sort((a, b) => {
      if (a.attributes.points !== b.attributes.points) {
        return b.attributes.points - a.attributes.points; // Higher points first
      }
      return a.attributes.name.localeCompare(b.attributes.name);
    });
  }

  /**
   * Transform guild response data (sort members and titles)
   */
  private transformGuildResponse(guild: Guild): Guild {
    const transformed = { ...guild };
    
    if (transformed.attributes.members) {
      // Sort members by title (rank) first, then by level descending
      transformed.attributes.members = [...transformed.attributes.members].sort((a, b) => {
        if (a.title !== b.title) {
          return a.title - b.title; // Lower title numbers = higher rank
        }
        return b.level - a.level; // Higher level first within same rank
      });
    }
    
    if (transformed.attributes.titles) {
      // Sort titles by index
      transformed.attributes.titles = [...transformed.attributes.titles].sort(
        (a, b) => a.index - b.index
      );
    }
    
    return transformed;
  }
}

// Create and export a singleton instance
export const guildsService = new GuildsService();

// Export types for use in other files
export type { Guild, GuildAttributes, GuildMember, GuildFilters };