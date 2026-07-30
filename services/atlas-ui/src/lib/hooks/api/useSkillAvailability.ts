import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import type { Tenant } from "@/services/api/tenants.service";
import {
  availabilityService,
  type AvailabilityEntry,
} from "@/services/api/availability.service";

export interface SkillAvailabilityResult {
  skills: AvailabilityEntry[];
}

export const skillAvailabilityKeys = {
  all: ["skill-availability"] as const,
  lists: () => [...skillAvailabilityKeys.all, "list"] as const,
  list: (tenantId: string | undefined) =>
    [...skillAvailabilityKeys.lists(), tenantId ?? "no-tenant"] as const,
};

/**
 * The tenant version's RELEASED skill identities (wire id + version-correct
 * name), sourced from GET /api/data/skill-availability. Mirrors
 * useJobAvailability -- see its doc comment for the availability-vs-WZ-
 * presence distinction (task-187).
 *
 * TenantProvider calls queryClient.clear() on every tenant switch, so
 * callers MUST treat the pending state as "unknown", not "empty".
 */
export function useSkillAvailability(
  tenant: Tenant | null | undefined,
): UseQueryResult<SkillAvailabilityResult, Error> {
  return useQuery({
    queryKey: skillAvailabilityKeys.list(tenant?.id),
    queryFn: async () => ({
      skills: await availabilityService.getSkillAvailability(),
    }),
    enabled: !!tenant?.id,
    staleTime: 30 * 60 * 1000,
    gcTime: 24 * 60 * 60 * 1000,
  });
}
