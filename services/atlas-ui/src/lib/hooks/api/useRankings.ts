/**
 * React Query hook for browsing character leaderboard rankings (read-only).
 */

import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import {
  rankingsService,
  type LeaderboardFilter,
  type RankingPage,
} from "@/services/api/rankings.service";

export const rankingsKeys = {
  all: ["rankings"] as const,
  // The tenant id is the FIRST key segment: rankings are tenant-scoped only via
  // the mutable global apiClient tenant header, so without the tenant in the key
  // two tenants sharing the same (worldId, filter) collide — switching tenants
  // while RankingsPage stays mounted would serve tenant A's cached rankings
  // under tenant B with no refetch. Mirrors the mtsListingsKeys tenant-first pattern.
  leaderboard: (tenantId: string, worldId: number, filter: LeaderboardFilter) =>
    [...rankingsKeys.all, "leaderboard", tenantId, worldId, filter] as const,
};

/**
 * Browse character rankings for a world leaderboard. Pass `enabled: false` to defer until a
 * tenant/world is selected. `tenantId` scopes the cache entry (the active
 * tenant's id) so the query refetches when the tenant changes.
 */
export function useRankings(
  tenantId: string,
  worldId: number,
  filter: LeaderboardFilter,
  enabled = true,
): UseQueryResult<RankingPage, Error> {
  return useQuery({
    queryKey: rankingsKeys.leaderboard(tenantId, worldId, filter),
    queryFn: () => rankingsService.leaderboard(worldId, filter),
    enabled,
  });
}
