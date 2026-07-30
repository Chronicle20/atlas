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

// Defensive ceiling on links.next pagination, mirroring jobsService.getJobs.
// A backstop against a malformed or self-referential `next` link from the
// backend, not a real expected page count.
const MAX_PAGES = 50;

function toEntries(
  resources: Array<{ id: string; attributes: { name: string } }>,
): AvailabilityEntry[] {
  return resources.map((r) => ({ id: Number(r.id), name: r.attributes.name }));
}

/**
 * Follows links.next until exhausted, collecting every resource across all
 * pages. Job/skill availability is a version's RELEASED set, which for
 * skills can exceed a single page[size]=250 response, so links.next MUST be
 * followed rather than trusting the first page alone.
 */
async function fetchAllPages<
  T extends { id: string; attributes: { name: string } },
>(startUrl: string): Promise<AvailabilityEntry[]> {
  let url: string | undefined = startUrl;
  const resources: T[] = [];
  const visited = new Set<string>();

  while (url) {
    if (visited.has(url) || visited.size >= MAX_PAGES) {
      throw new Error(
        `availabilityService: aborting pagination after ${visited.size} page(s) — ` +
          `links.next did not advance (url: ${url}). The backend is misbehaving.`,
      );
    }
    visited.add(url);

    const doc: ApiPagedResponse<T> = await api.getListDocument<T>(url);
    resources.push(...(doc.data ?? []));
    url = doc.links?.next;
  }

  return toEntries(resources);
}

export const availabilityService = {
  /** The tenant version's RELEASED job identities: wire id + version-correct name. */
  async getJobAvailability(): Promise<AvailabilityEntry[]> {
    return fetchAllPages<JobAvailabilityResource>(JOB_BASE_PATH);
  },

  /** The tenant version's RELEASED skill identities: wire id + version-correct name. */
  async getSkillAvailability(): Promise<AvailabilityEntry[]> {
    return fetchAllPages<SkillAvailabilityResource>(SKILL_BASE_PATH);
  },
};
