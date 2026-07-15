/**
 * React Query hook for browsing atlas-mts marketplace listings (read-only).
 */

import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import {
  mtsListingsService,
  type BrowseListingsFilter,
  type MtsListingPage,
} from "@/services/api/mts-listings.service";

export const mtsListingsKeys = {
  all: ["mts-listings"] as const,
  // The tenant id is the FIRST key segment: listings are tenant-scoped only via
  // the mutable global apiClient tenant header, so without the tenant in the key
  // two tenants sharing the same (worldId, filter) collide — switching tenants
  // while MarketplacePage stays mounted would serve tenant A's cached listings
  // under tenant B with no refetch. Mirrors the guildKeys tenant-first pattern.
  browse: (tenantId: string, worldId: number, filter: BrowseListingsFilter) =>
    [...mtsListingsKeys.all, "browse", tenantId, worldId, filter] as const,
};

/**
 * Browse active listings for a world. Pass `enabled: false` to defer until a
 * tenant/world is selected. `tenantId` scopes the cache entry (the active
 * tenant's id) so the query refetches when the tenant changes.
 */
export function useMtsListings(
  tenantId: string,
  worldId: number,
  filter: BrowseListingsFilter,
  enabled = true,
): UseQueryResult<MtsListingPage, Error> {
  return useQuery({
    queryKey: mtsListingsKeys.browse(tenantId, worldId, filter),
    queryFn: () => mtsListingsService.browse(worldId, filter),
    enabled,
  });
}
