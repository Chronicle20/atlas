import { useMemo } from "react";
import { useTenant } from "@/context/tenant-context";
import { useJobAvailability } from "@/lib/hooks/api/useJobAvailability";
import { JOB_LIST } from "@/lib/jobs/job-advancement-tree";

export interface PresetJobOption {
  id: number;
  name: string;
}

/**
 * The job options the active tenant can actually select, ascending by id.
 *
 * Sourced from GET /api/data/job-availability -- the tenant version's
 * RELEASED job identities -- rather than the former JOB_LIST/useJobs pairing
 * (WZ-presence gating), because WZ presence includes unreleased STUB jobs
 * (e.g. the pre-v0.61 Pirate placeholder at wire id 500, which at that
 * version is actually Gm). Availability also supplies the version-correct
 * display name directly (Set.Name, e.g. "Gm" not "GM"), so this supersedes
 * the static JOB_LIST/jobName lookup as the name source too (task-187).
 *
 * useJobAvailability's pending/error state means "unknown", not "empty"
 * (TenantProvider clears the query cache on every tenant switch). So until
 * availability is known this returns the full advancement graph (JOB_LIST)
 * rather than an empty list: a picker must never be blank, and the backend
 * validates the chosen id regardless. Once availability loads it narrows to
 * exactly what the tenant's version has released, named for that version.
 */
export function usePresetJobOptions(): PresetJobOption[] {
  const { activeTenant } = useTenant();
  const availabilityQuery = useJobAvailability(activeTenant);

  return useMemo(() => {
    if (!availabilityQuery.isSuccess) return JOB_LIST;
    return availabilityQuery.data.jobs
      .map((j) => ({ id: j.id, name: j.name }))
      .sort((a, b) => a.id - b.id);
  }, [availabilityQuery.isSuccess, availabilityQuery.data]);
}
