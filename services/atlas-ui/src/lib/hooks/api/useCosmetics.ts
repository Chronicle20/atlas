import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { cosmeticsService } from "@/services/api/cosmetics.service";
import { useTenant } from "@/context/tenant-context";

export const cosmeticsKeys = {
  all: ["cosmetics"] as const,
  faces: () => [...cosmeticsKeys.all, "faces"] as const,
  hairs: () => [...cosmeticsKeys.all, "hairs"] as const,
};

// WZ data changes only with re-ingest; TenantProvider clears all caches on
// tenant switch, so a long staleTime is safe.
const WZ_STALE_TIME = 60 * 60 * 1000;

export function useFaceIds(): UseQueryResult<number[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: cosmeticsKeys.faces(),
    queryFn: () => cosmeticsService.getAllFaceIds(),
    enabled: !!activeTenant,
    staleTime: WZ_STALE_TIME,
    gcTime: WZ_STALE_TIME,
  });
}

export function useHairIds(): UseQueryResult<number[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: cosmeticsKeys.hairs(),
    queryFn: () => cosmeticsService.getAllHairIds(),
    enabled: !!activeTenant,
    staleTime: WZ_STALE_TIME,
    gcTime: WZ_STALE_TIME,
  });
}
