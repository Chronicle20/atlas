import { useCallback, useMemo } from "react";
import { useTenant } from "@/context/tenant-context";
import { useJobs } from "@/lib/hooks/api/useJobs";
import { useJobAvailability } from "@/lib/hooks/api/useJobAvailability";
import {
  buildJobGraph,
  jobNodeName,
  type JobGraph,
} from "@/lib/jobs/job-graph";

export interface JobGraphResult {
  /** The tenant version's job graph: availability ∩ WZ presence, re-rooted. Empty unless isSuccess. */
  graph: JobGraph;
  /** Both queries succeeded. Only then may a caller treat an absent id as absent. */
  isSuccess: boolean;
  /** Either query is still pending. Means "unknown", NOT "empty". */
  isPending: boolean;
  /** Either query failed. */
  isError: boolean;
}

/**
 * The single source of the Jobs page's hierarchy: release availability
 * (version-correct names + advancement parents) intersected with the
 * tenant's ingested WZ job set (FR-4.1/4.2/4.3).
 *
 * Gating discipline, generalised from task-182 design D10 across two
 * queries: TenantProvider calls queryClient.clear() on every tenant switch,
 * so "empty graph" is the NORMAL state immediately after a switch. Callers
 * MUST gate destructive behaviour (redirecting an invalid /jobs/{id},
 * treating an id as absent) on isSuccess, never on graph.size. Both
 * underlying queries are keyed by tenant id, so no cross-tenant bleed is
 * possible even without the cache clear.
 */
export function useJobGraph(): JobGraphResult {
  const { activeTenant } = useTenant();
  const availabilityQuery = useJobAvailability(activeTenant);
  const jobsQuery = useJobs(activeTenant);

  const isSuccess = availabilityQuery.isSuccess && jobsQuery.isSuccess;
  const isPending = availabilityQuery.isPending || jobsQuery.isPending;
  const isError = availabilityQuery.isError || jobsQuery.isError;

  const graph = useMemo<JobGraph>(() => {
    if (!isSuccess) return new Map();
    const present = new Set(
      (jobsQuery.data?.jobs ?? []).map((j) => Number(j.id)),
    );
    return buildJobGraph(availabilityQuery.data?.jobs ?? [], present);
  }, [isSuccess, availabilityQuery.data, jobsQuery.data]);

  return { graph, isSuccess, isPending, isError };
}

/**
 * A version-correct job-name resolver for call sites that are not React
 * components and cannot hold the graph themselves (breadcrumb resolvers,
 * table column builders). The nearest component calls this hook and passes
 * the returned function down.
 *
 * Falls back to `Job <id>` while the graph is unknown. That is deliberate:
 * it is honest about not knowing, whereas the static table it replaces
 * asserted a v83 name to a v0.48 tenant.
 */
export function useJobNameLookup(): (id: number) => string {
  const { graph } = useJobGraph();
  return useCallback((id: number) => jobNodeName(graph, id), [graph]);
}
