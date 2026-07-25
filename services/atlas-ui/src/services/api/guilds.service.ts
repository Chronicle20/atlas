/**
 * Guilds Service
 *
 * Provides read-only guild data access:
 * - Guild listing (paged) and detail retrieval
 * - Guild search by name via the server-side `filter[name]` substring match
 * - Guild ranking and statistics
 *
 * Note: The backend guild API is read-only. Guild mutations
 * (create, update, delete, member management) are handled
 * through in-game systems via Kafka events.
 *
 * task-117: the previous fetch-all-then-filter pattern (`getAll()` +
 * client-side `.filter()`) is gone. `getPage`/`search` page server-side
 * against `GET /guilds` / `GET /guilds?filter[name]=`. `getByWorld`,
 * `getWithSpace`, and `getRankings` still drain the full collection via
 * `fetchAll` because atlas-guilds has no `filter[worldId]` route
 * (verified: `services/atlas-guilds/atlas.com/guilds/guild/resource.go`
 * only registers `filter[members.id]` and `filter[name]`) — a future
 * `filter[worldId]` would let those page server-side too.
 */

import type { ServiceOptions } from "@/lib/api/query-params";
import type { Guild, GuildAttributes, GuildMember } from "@/types/models/guild";
import { api } from "@/lib/api/client";
import {
  fetchAll,
  fetchPaged,
  type PagedResult,
} from "@/services/api/pagination";

/**
 * Guilds service class with tenant-aware read-only API operations.
 *
 * Tenant is handled centrally by TenantProvider's effect
 * (`api.setTenant(activeTenant)`), so these methods no longer take a
 * tenant argument.
 */
class GuildsService {
  private basePath = "/api/guilds";

  /**
   * Get a single page of guilds. Used by the Guilds list view (task-117),
   * which pages server-side.
   */
  async getPage(
    page: { number: number; size: number },
    options?: ServiceOptions,
  ): Promise<PagedResult<Guild>> {
    const result = await fetchPaged<Guild>(this.basePath, page, options);
    return { data: this.sortGuilds(result.data), meta: result.meta };
  }

  async getById(guildId: string, options?: ServiceOptions): Promise<Guild> {
    const guild = await api.getOne<Guild>(
      `${this.basePath}/${guildId}`,
      options,
    );
    return this.transformGuildResponse(guild);
  }

  /**
   * Fetch guilds by name substring via the backend's `filter[name]` query
   * (task-117 / Task 11's server-side contract), a single page at a time.
   */
  async search(
    searchTerm: string,
    page: { number: number; size: number },
    options?: ServiceOptions,
  ): Promise<PagedResult<Guild>> {
    const qs = new URLSearchParams({ "filter[name]": searchTerm }).toString();
    const result = await fetchPaged<Guild>(
      `${this.basePath}?${qs}`,
      page,
      options,
    );
    return { data: this.sortGuilds(result.data), meta: result.meta };
  }

  /**
   * Fetch guilds containing a specific member, using the backend's
   * `filter[members.id]` query string. Returns an array (possibly empty).
   */
  async getByMemberId(
    memberId: string,
    options?: ServiceOptions,
  ): Promise<Guild[]> {
    const qs = new URLSearchParams({
      "filter[members.id]": memberId,
    }).toString();
    return api.getList<Guild>(`${this.basePath}?${qs}`, options);
  }

  /**
   * Fetch guilds on a given world. No `filter[worldId]` route exists on
   * atlas-guilds yet, so this drains the full collection and filters
   * in-memory (correct, just not page-bounded).
   */
  async getByWorld(
    worldId: number,
    options?: ServiceOptions,
  ): Promise<Guild[]> {
    const guilds = await fetchAll<Guild>(this.basePath, undefined, options);
    return this.sortGuilds(
      guilds.filter((guild) => guild.attributes.worldId === worldId),
    );
  }

  /**
   * Fetch guilds with open member slots, optionally scoped to a world. No
   * server-side support for either filter, so this drains + filters
   * in-memory (correct, just not page-bounded).
   */
  async getWithSpace(
    worldId?: number,
    options?: ServiceOptions,
  ): Promise<Guild[]> {
    const guilds = await fetchAll<Guild>(this.basePath, undefined, options);
    let filtered = guilds.filter(
      (guild) => guild.attributes.members.length < guild.attributes.capacity,
    );
    if (worldId !== undefined) {
      filtered = filtered.filter(
        (guild) => guild.attributes.worldId === worldId,
      );
    }
    return this.sortGuilds(filtered);
  }

  /**
   * Fetch the top `limit` guilds by points, optionally scoped to a world.
   * Ranking requires the whole collection to sort correctly, so this
   * drains + filters + sorts in-memory (correct, just not page-bounded).
   */
  async getRankings(
    worldId?: number,
    limit = 50,
    options?: ServiceOptions,
  ): Promise<Guild[]> {
    let guilds = await fetchAll<Guild>(this.basePath, undefined, options);
    if (worldId !== undefined) {
      guilds = guilds.filter((guild) => guild.attributes.worldId === worldId);
    }
    return this.sortGuilds(guilds).slice(0, limit);
  }

  async exists(guildId: string, options?: ServiceOptions): Promise<boolean> {
    try {
      await this.getById(guildId, options);
      return true;
    } catch (error) {
      if (
        error &&
        typeof error === "object" &&
        "status" in error &&
        error.status === 404
      ) {
        return false;
      }
      throw error;
    }
  }

  async getMemberCount(
    guildId: string,
    options?: ServiceOptions,
  ): Promise<number> {
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
      transformed.attributes.members = [...transformed.attributes.members].sort(
        (a, b) => {
          if (a.title !== b.title) {
            return a.title - b.title;
          }
          return b.level - a.level;
        },
      );
    }
    if (transformed.attributes.titles) {
      transformed.attributes.titles = [...transformed.attributes.titles].sort(
        (a, b) => a.index - b.index,
      );
    }
    return transformed;
  }
}

export const guildsService = new GuildsService();

export type { Guild, GuildAttributes, GuildMember };
