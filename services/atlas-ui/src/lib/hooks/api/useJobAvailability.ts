import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import type { Tenant } from "@/services/api/tenants.service";
import {
  availabilityService,
  type JobAvailabilityEntry,
} from "@/services/api/availability.service";

export interface JobAvailabilityResult {
  jobs: JobAvailabilityEntry[];
}

export const jobAvailabilityKeys = {
  all: ["job-availability"] as const,
  lists: () => [...jobAvailabilityKeys.all, "list"] as const,
  list: (tenantId: string | undefined) =>
    [...jobAvailabilityKeys.lists(), tenantId ?? "no-tenant"] as const,
};

/**
 * The tenant version's RELEASED job identities (wire id + version-correct
 * name), sourced from GET /api/data/job-availability. Unlike useJobs (which
 * reflects whatever the tenant's ingested Skill.wz happens to contain,
 * including unreleased STUB jobs like the pre-v0.61 Pirate placeholder),
 * this is gated on the version's actual release status -- see task-187.
 *
 * TenantProvider calls queryClient.clear() on every tenant switch, so
 * callers MUST treat the pending state as "unknown", not "empty".
 */
export function useJobAvailability(
  tenant: Tenant | null | undefined,
): UseQueryResult<JobAvailabilityResult, Error> {
  return useQuery({
    queryKey: jobAvailabilityKeys.list(tenant?.id),
    queryFn: async () => ({
      jobs: await availabilityService.getJobAvailability(),
    }),
    enabled: !!tenant?.id,
    staleTime: 30 * 60 * 1000,
    gcTime: 24 * 60 * 60 * 1000,
  });
}
