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
  browse: (worldId: number, filter: BrowseListingsFilter) =>
    [...mtsListingsKeys.all, "browse", worldId, filter] as const,
};

/**
 * Browse active listings for a world. Pass `enabled: false` to defer until a
 * tenant/world is selected.
 */
export function useMtsListings(
  worldId: number,
  filter: BrowseListingsFilter,
  enabled = true,
): UseQueryResult<MtsListingPage, Error> {
  return useQuery({
    queryKey: mtsListingsKeys.browse(worldId, filter),
    queryFn: () => mtsListingsService.browse(worldId, filter),
    enabled,
  });
}
