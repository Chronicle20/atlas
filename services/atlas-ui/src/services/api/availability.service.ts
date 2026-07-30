import { api } from "@/lib/api/client";
import type { ApiPagedResponse } from "@/types/api/responses";

/**
 * A job-availability or skill-availability JSON:API resource. The resource
 * `id` IS the version-appropriate wire id (not a version-blind identity
 * token) -- atlas-data's RestModel.GetID() returns strconv.Itoa(wireId) --
 * so it round-trips straight back into whatever job/skill id field the
 * tenant's version expects. `attributes.name` is the version's display name
 * (e.g. wire id 500 is "Gm" pre-v0.61, "Pirate" at v0.61+).
 */
export interface JobAvailabilityResource {
  id: string;
  type: string;
  attributes: { name: string };
}

export interface SkillAvailabilityResource {
  id: string;
  type: string;
  attributes: { name: string };
}

export interface AvailabilityEntry {
  id: number;
  name: string;
}

const JOB_BASE_PATH = "/api/data/job-availability?page[size]=250";
const SKILL_BASE_PATH = "/api/data/skill-availability?page[size]=250";

// The backend returns the tenant version's full RELEASED set in one
// unpaginated response (jobavailability/skillavailability resource.go calls
// GetAvailable() once and marshals the whole slice -- no links.next). The
// page[size] param is sent for parity with other list endpoints but is not
// load-bearing here.

function toEntries(
  resources: Array<{ id: string; attributes: { name: string } }>,
): AvailabilityEntry[] {
  return resources.map((r) => ({ id: Number(r.id), name: r.attributes.name }));
}

export const availabilityService = {
  /** The tenant version's RELEASED job identities: wire id + version-correct name. */
  async getJobAvailability(): Promise<AvailabilityEntry[]> {
    const doc: ApiPagedResponse<JobAvailabilityResource> =
      await api.getListDocument<JobAvailabilityResource>(JOB_BASE_PATH);
    return toEntries(doc.data ?? []);
  },

  /** The tenant version's RELEASED skill identities: wire id + version-correct name. */
  async getSkillAvailability(): Promise<AvailabilityEntry[]> {
    const doc: ApiPagedResponse<SkillAvailabilityResource> =
      await api.getListDocument<SkillAvailabilityResource>(SKILL_BASE_PATH);
    return toEntries(doc.data ?? []);
  },
};
