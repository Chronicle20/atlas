import { useMemo } from "react";
import { useTenant } from "@/context/tenant-context";
import { useJobs } from "@/lib/hooks/api/useJobs";
import { JOB_LIST, type JobEntry } from "@/lib/jobs/job-advancement-tree";

/**
 * The job-graph entries the active tenant can actually select, ascending by id.
 *
 * Sourced from the advancement graph (which already knows Aran, Evan and the
 * Cygnus Knights) and gated by the tenant's ingested job set from
 * GET /api/data/jobs — the same signal JobsPage uses. On a version that
 * shipped Aran/Evan those roots are present in the job set and appear here; on
 * one that did not they are absent and stay hidden. This replaces the former
 * hand-maintained curated list that stopped at Super GM and could never show
 * Aran/Evan on any version.
 *
 * useJobs' pending/error state means "unknown", not "empty" (TenantProvider
 * clears the query cache on every tenant switch). So until the job set is
 * known this returns the full graph rather than an empty list: a picker must
 * never be blank, and the backend validates the chosen id regardless. Once the
 * set loads it narrows to what the tenant has.
 */
export function usePresetJobOptions(): JobEntry[] {
  const { activeTenant } = useTenant();
  const jobsQuery = useJobs(activeTenant);

  return useMemo(() => {
    if (!jobsQuery.isSuccess) return JOB_LIST;
    const available = new Set(jobsQuery.data.jobs.map((j) => Number(j.id)));
    return JOB_LIST.filter((j) => available.has(j.id));
  }, [jobsQuery.isSuccess, jobsQuery.data]);
}
