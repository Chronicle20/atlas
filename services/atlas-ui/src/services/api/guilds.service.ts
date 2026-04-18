/**
 * Guilds Service
 *
 * Provides read-only guild data access:
 * - Guild listing and detail retrieval
 * - Guild search and filtering (client-side)
 * - Guild ranking and statistics
 *
 * Note: The backend guild API is read-only. Guild mutations
 * (create, update, delete, member management) are handled
 * through in-game systems via Kafka events.
 */

import type { ServiceOptions } from './base.service';
import type { Guild, GuildAttributes, GuildMember } from '@/types/models/guild';
import type { Tenant } from '@/types/models/tenant';
import { api } from '@/lib/api/client';

/**
 * Guilds service class with tenant-aware read-only API operations
 */
class GuildsService {
  private basePath = '/api/guilds';

  /**
   * Get all guilds for a tenant
   */
  async getAll(tenant: Tenant, options?: ServiceOptions): Promise<Guild[]> {
    api.setTenant(tenant);
    const guilds = await api.getList<Guild>(this.basePath, options);
    return this.sortGuilds(guilds);
  }

  /**
   * Get guild by ID for a tenant
   */
  async getById(tenant: Tenant, guildId: string, options?: ServiceOptions): Promise<Guild> {
    api.setTenant(tenant);
    const guild = await api.getOne<Guild>(`${this.basePath}/${guildId}`, options);
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

    return guilds.slice(0, limit);
  }

  /**
   * Check if guild exists
   */
  async exists(tenant: Tenant, guildId: string, options?: ServiceOptions): Promise<boolean> {
    try {
      await this.getById(tenant, guildId, options);
      return true;
    } catch (error) {
      if (error && typeof error === 'object' && 'status' in error && error.status === 404) {
        return false;
      }
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

  private sortGuilds(guilds: Guild[]): Guild[] {
    return guilds.sort((a, b) => {
      if (a.attributes.points !== b.attributes.points) {
        return b.attributes.points - a.attributes.points;
      }
      return a.attributes.name.localeCompare(b.attributes.name);
    });
  }

  private transformGuildResponse(guild: Guild): Guild {
    const transformed = { ...guild };

    if (transformed.attributes.members) {
      transformed.attributes.members = [...transformed.attributes.members].sort((a, b) => {
        if (a.title !== b.title) {
          return a.title - b.title;
        }
        return b.level - a.level;
      });
    }

    if (transformed.attributes.titles) {
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
export type { Guild, GuildAttributes, GuildMember };