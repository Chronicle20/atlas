import { api } from "@/lib/api/client";

interface JobResource {
  id: string;
  type: string;
  attributes: { skills: number[] };
}

const BASE_PATH = "/api/data/jobs";

export const jobsService = {
  async getSkillsByJobId(jobId: number): Promise<number[]> {
    const job = await api.getOne<JobResource>(`${BASE_PATH}/${jobId}/skills`);
    return job.attributes.skills;
  },
};
