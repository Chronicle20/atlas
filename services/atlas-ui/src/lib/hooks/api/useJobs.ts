import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import type { Tenant } from "@/services/api/tenants.service";
import { jobsService, type JobsResult } from "@/services/api/jobs.service";

export const jobsKeys = {
  all: ["jobs"] as const,
  lists: () => [...jobsKeys.all, "list"] as const,
  list: (tenantId: string | undefined) =>
    [...jobsKeys.lists(), tenantId ?? "no-tenant"] as const,
};

/**
 * The tenant's job set, as ingested from its Skill.wz. This is the backend
 * replacement for the retired hand-maintained version-floor tables that used
 * to gate job visibility off majorVersion: a job is visible if and only if
 * the tenant has a JOB document for it.
 *
 * TenantProvider calls queryClient.clear() on every tenant switch, so callers
 * MUST treat the pending state as "unknown", not "empty" — see JobsPage.
 */
export function useJobs(
  tenant: Tenant | null | undefined,
): UseQueryResult<JobsResult, Error> {
  return useQuery({
    queryKey: jobsKeys.list(tenant?.id),
    queryFn: () => jobsService.getJobs(),
    enabled: !!tenant?.id,
    staleTime: 30 * 60 * 1000,
    gcTime: 24 * 60 * 60 * 1000,
  });
}
