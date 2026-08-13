import { useMemo } from "react";
import { useTenant } from "@/context/tenant-context";
import { useJobAvailability } from "@/lib/hooks/api/useJobAvailability";

export interface PresetJobOption {
  id: number;
  name: string;
}

export interface PresetJobOptionsResult {
  options: PresetJobOption[];
  isPending: boolean;
  isError: boolean;
}

/**
 * The job options the active tenant can actually select, ascending by id.
 *
 * Sourced from GET /api/data/job-availability -- the tenant version's
 * RELEASED job identities -- rather than the former static-table/useJobs
 * pairing (WZ-presence gating), because WZ presence includes unreleased
 * STUB jobs (e.g. the pre-v0.61 Pirate placeholder at wire id 500, which at
 * that version is actually Gm). Availability also supplies the
 * version-correct display name directly (Set.Name, e.g. "Gm" not "GM"), so
 * this supersedes the static-table name lookup too (task-187).
 *
 * useJobAvailability's pending/error state means "unknown", not "empty"
 * (TenantProvider clears the query cache on every tenant switch). Until
 * availability is known, `options` is an EMPTY list -- callers MUST branch
 * on `isPending`/`isError`, not on `options.length`, to tell "still
 * loading"/"failed" apart from "loaded, tenant has no jobs" and render the
 * matching affordance themselves. It used to fall back to a static v83 job
 * table on the reasoning that "a picker must never be blank" -- but that
 * offered v83 job names to a v0.48 tenant, which is exactly the bug task-202
 * removed. An empty list plus explicit loading/error state is honest; a
 * wrong list is not.
 */
export function usePresetJobOptions(): PresetJobOptionsResult {
  const { activeTenant } = useTenant();
  const availabilityQuery = useJobAvailability(activeTenant);

  const options = useMemo(() => {
    if (!availabilityQuery.isSuccess) return [];
    return availabilityQuery.data.jobs
      .map((j) => ({ id: j.id, name: j.name }))
      .sort((a, b) => a.id - b.id);
  }, [availabilityQuery.isSuccess, availabilityQuery.data]);

  return {
    options,
    isPending: availabilityQuery.isPending,
    isError: availabilityQuery.isError,
  };
}
