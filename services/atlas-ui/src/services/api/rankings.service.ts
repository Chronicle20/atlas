import { apiClient } from "@/lib/api/client";
import type { ServiceOptions } from "@/lib/api/query-params";
import type { ApiPagedResponse } from "@/types/api/responses";

/**
 * Read-only per-world character leaderboard, backed by atlas-rankings:
 *   GET /api/rankings?filter[worldId]=&filter[jobCategory]=&page[number]=&page[size]=
 *
 * The response is a JSON:API list of `rankings` resources with a pagination
 * `meta` block (`meta.total`, `meta.page.last`), so total/lastPage are
 * authoritative — never inferred from the returned length.
 */

export interface RankingEntryAttributes {
  characterId: number;
  name: string;
  worldId: number;
  level: number;
  jobId: number;
  jobCategory: number;
  rank: number;
  rankMove: number;
  jobRank: number;
  jobRankMove: number;
  computedAt: string;
}

export interface RankingEntry {
  id: string;
  attributes: RankingEntryAttributes;
}

export interface RankingPage {
  entries: RankingEntry[];
  total: number;
  lastPage: number;
}

export interface LeaderboardFilter {
  /** Overall leaderboard when undefined; otherwise restricts to jobId/100. */
  jobCategory?: number | undefined;
  /** ZERO-BASED caller page (page=0 is the first page). */
  page?: number | undefined;
  pageSize?: number | undefined;
}

export function buildLeaderboardQuery(worldId: number, filter: LeaderboardFilter): string {
  const params = new URLSearchParams();
  params.set("filter[worldId]", String(worldId));
  if (filter.jobCategory !== undefined)
    params.set("filter[jobCategory]", String(filter.jobCategory));
  if (filter.page !== undefined)
    params.set("page[number]", String(filter.page + 1));
  if (filter.pageSize !== undefined)
    params.set("page[size]", String(filter.pageSize));
  return `?${params.toString()}`;
}

export const rankingsService = {
  async leaderboard(
    worldId: number,
    filter: LeaderboardFilter,
    options?: ServiceOptions,
  ): Promise<RankingPage> {
    const query = buildLeaderboardQuery(worldId, filter);
    const resp = await apiClient.get<ApiPagedResponse<RankingEntry>>(
      `/api/rankings${query}`,
      options,
    );
    const total = resp.meta?.total ?? resp.data.length;
    const lastPage = resp.meta?.page?.last ?? 1;
    return { entries: resp.data, total, lastPage };
  },
};
