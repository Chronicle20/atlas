import { api } from "@/lib/api/client";
import type { ApiPagedResponse, JsonApiResource } from "@/types/api/responses";

export interface JobResource {
  id: string;
  type: string;
  attributes: { skills: number[] };
}

export interface JobsResult {
  jobs: JobResource[];
  /** Populated only when getJobs was called with includeSkills. */
  skillsById: Map<number, JsonApiResource>;
}

const BASE_PATH = "/api/data/jobs";

// The backend caps page[size] at 250 and there are ~82 jobs, so one request
// normally suffices — but links.next is followed anyway so the ceiling is not
// load-bearing.
const PAGE_SIZE = 250;

// Defensive ceiling on links.next pagination. At PAGE_SIZE=250 with ~82 jobs
// a real response is one page; this is only a backstop against a malformed
// or self-referential `next` link from the backend.
const MAX_PAGES = 50;

export const jobsService = {
  async getSkillsByJobId(jobId: number): Promise<number[]> {
    const job = await api.getOne<JobResource>(`${BASE_PATH}/${jobId}/skills`);
    return job.attributes.skills;
  },

  /**
   * Every job present for the active tenant, as ingested from that tenant's
   * Skill.wz. `includeSkills` side-loads the full skill resources into
   * `skillsById`; it defaults to false and has no production caller today —
   * per-skill definitions are fetched through useJobSkillDefinitions, which
   * caches per skill id across jobs.
   */
  async getJobs(opts?: { includeSkills?: boolean }): Promise<JobsResult> {
    const params = new URLSearchParams({ "page[size]": String(PAGE_SIZE) });
    if (opts?.includeSkills) params.set("include", "skills");

    let url: string | undefined = `${BASE_PATH}?${params.toString()}`;
    const jobs: JobResource[] = [];
    const skillsById = new Map<number, JsonApiResource>();
    const visited = new Set<string>();

    while (url) {
      if (visited.has(url) || visited.size >= MAX_PAGES) {
        throw new Error(
          `jobsService.getJobs: aborting pagination after ${visited.size} page(s) — ` +
            `links.next did not advance (url: ${url}). The backend is misbehaving.`,
        );
      }
      visited.add(url);

      const doc: ApiPagedResponse<JobResource> & {
        included?: JsonApiResource[];
      } = await api.getListDocument<JobResource>(url);
      jobs.push(...(doc.data ?? []));
      for (const inc of doc.included ?? []) {
        if (inc.type !== "skills") continue;
        const id = Number(inc.id);
        if (Number.isInteger(id)) skillsById.set(id, inc);
      }
      url = doc.links?.next;
    }

    return { jobs, skillsById };
  },
};
