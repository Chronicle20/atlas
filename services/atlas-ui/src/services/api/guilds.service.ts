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

import type { ServiceOptions } from '@/lib/api/query-params';
import type { Guild, GuildAttributes, GuildMember } from '@/types/models/guild';
import { api } from '@/lib/api/client';

/**
 * Guilds service class with tenant-aware read-only API operations.
 *
 * Tenant is handled centrally by TenantProvider's effect
 * (`api.setTenant(activeTenant)`), so these methods no longer take a
 * tenant argument.
 */
class GuildsService {
  private basePath = '/api/guilds';

  async getAll(options?: ServiceOptions): Promise<Guild[]> {
    const guilds = await api.getList<Guild>(this.basePath, options);
    return this.sortGuilds(guilds);
  }

  async getById(guildId: string, options?: ServiceOptions): Promise<Guild> {
    const guild = await api.getOne<Guild>(`${this.basePath}/${guildId}`, options);
    return this.transformGuildResponse(guild);
  }

  async getByWorld(worldId: number, options?: ServiceOptions): Promise<Guild[]> {
    const guilds = await this.getAll(options);
    return guilds.filter(guild => guild.attributes.worldId === worldId);
  }

  async search(searchTerm: string, worldId?: number, options?: ServiceOptions): Promise<Guild[]> {
    const guilds = await this.getAll(options);
    let filtered = guilds.filter(guild =>
      guild.attributes.name.toLowerCase().includes(searchTerm.toLowerCase())
    );
    if (worldId !== undefined) {
      filtered = filtered.filter(guild => guild.attributes.worldId === worldId);
    }
    return filtered;
  }

  async getWithSpace(worldId?: number, options?: ServiceOptions): Promise<Guild[]> {
    const guilds = await this.getAll(options);
    let filtered = guilds.filter(guild =>
      guild.attributes.members.length < guild.attributes.capacity
    );
    if (worldId !== undefined) {
      filtered = filtered.filter(guild => guild.attributes.worldId === worldId);
    }
    return filtered;
  }

  async getRankings(worldId?: number, limit = 50, options?: ServiceOptions): Promise<Guild[]> {
    let guilds = await this.getAll(options);
    if (worldId !== undefined) {
      guilds = guilds.filter(guild => guild.attributes.worldId === worldId);
    }
    return guilds.slice(0, limit);
  }

  async exists(guildId: string, options?: ServiceOptions): Promise<boolean> {
    try {
      await this.getById(guildId, options);
      return true;
    } catch (error) {
      if (error && typeof error === 'object' && 'status' in error && error.status === 404) {
        return false;
      }
      throw error;
    }
  }

  async getMemberCount(guildId: string, options?: ServiceOptions): Promise<number> {
    const guild = await this.getById(guildId, options);
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

export const guildsService = new GuildsService();

export type { Guild, GuildAttributes, GuildMember };
