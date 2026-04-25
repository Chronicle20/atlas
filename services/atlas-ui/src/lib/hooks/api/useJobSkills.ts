import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import type { Tenant } from "@/services/api/tenants.service";
import { jobsService } from "@/services/api/jobs.service";

export const jobSkillsKeys = {
  all: ["job-skills"] as const,
  detail: (tenantId: string | undefined, jobId: number) =>
    ["job-skills", tenantId, jobId] as const,
};

export function useJobSkills(
  tenant: Tenant | null | undefined,
  jobId: number
): UseQueryResult<number[], Error> {
  return useQuery({
    queryKey: jobSkillsKeys.detail(tenant?.id, jobId),
    queryFn: () => jobsService.getSkillsByJobId(jobId),
    enabled: !!tenant?.id && jobId > 0,
    staleTime: 30 * 60 * 1000,
    gcTime: 24 * 60 * 60 * 1000,
  });
}
