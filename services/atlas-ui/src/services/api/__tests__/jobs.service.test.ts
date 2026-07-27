import { describe, it, expect, vi, beforeEach } from "vitest";

const getOneMock = vi.fn();
const getListDocumentMock = vi.fn();
vi.mock("@/lib/api/client", () => ({
  api: {
    getOne: (...args: unknown[]) => getOneMock(...args),
    getListDocument: (...args: unknown[]) => getListDocumentMock(...args),
  },
}));

import { jobsService } from "@/services/api/jobs.service";

describe("jobsService.getSkillsByJobId", () => {
  beforeEach(() => vi.clearAllMocks());

  it("returns the skills attribute of the jobs resource", async () => {
    getOneMock.mockResolvedValue({
      id: "112",
      type: "jobs",
      attributes: { skills: [1121000, 1121001] },
    });
    await expect(jobsService.getSkillsByJobId(112)).resolves.toEqual([
      1121000, 1121001,
    ]);
    expect(getOneMock).toHaveBeenCalledWith("/api/data/jobs/112/skills");
  });
});

describe("jobsService.getJobs", () => {
  beforeEach(() => vi.clearAllMocks());

  it("requests a full page and returns the jobs without include by default", async () => {
    getListDocumentMock.mockResolvedValue({
      data: [
        { id: "100", type: "jobs", attributes: { skills: [1001000] } },
        { id: "112", type: "jobs", attributes: { skills: [1121000] } },
      ],
    });

    const result = await jobsService.getJobs();

    expect(getListDocumentMock).toHaveBeenCalledTimes(1);
    const url = getListDocumentMock.mock.calls[0]?.[0] as string;
    expect(url).toContain("/api/data/jobs");
    expect(url).toContain("page%5Bsize%5D=250");
    expect(url).not.toContain("include");
    expect(result.jobs.map((j) => j.id)).toEqual(["100", "112"]);
    expect(result.skillsById.size).toBe(0);
  });

  it("follows links.next until exhausted", async () => {
    getListDocumentMock
      .mockResolvedValueOnce({
        data: [{ id: "100", type: "jobs", attributes: { skills: [] } }],
        links: { next: "/api/data/jobs?page%5Bnumber%5D=2&page%5Bsize%5D=250" },
      })
      .mockResolvedValueOnce({
        data: [{ id: "112", type: "jobs", attributes: { skills: [] } }],
        links: {},
      });

    const result = await jobsService.getJobs();

    expect(getListDocumentMock).toHaveBeenCalledTimes(2);
    expect(getListDocumentMock.mock.calls[1]?.[0]).toBe(
      "/api/data/jobs?page%5Bnumber%5D=2&page%5Bsize%5D=250",
    );
    expect(result.jobs.map((j) => j.id)).toEqual(["100", "112"]);
  });

  it("indexes included skills by numeric id when includeSkills is set", async () => {
    getListDocumentMock.mockResolvedValue({
      data: [{ id: "100", type: "jobs", attributes: { skills: [1001000] } }],
      included: [
        { id: "1001000", type: "skills", attributes: { name: "Power Strike" } },
      ],
    });

    const result = await jobsService.getJobs({ includeSkills: true });

    expect(getListDocumentMock.mock.calls[0]?.[0]).toContain("include=skills");
    expect(result.skillsById.get(1001000)?.attributes?.name).toBe(
      "Power Strike",
    );
  });

  it("ignores non-skills members of included", async () => {
    getListDocumentMock.mockResolvedValue({
      data: [{ id: "100", type: "jobs", attributes: { skills: [] } }],
      included: [{ id: "9", type: "somethingelse", attributes: {} }],
    });

    const result = await jobsService.getJobs({ includeSkills: true });
    expect(result.skillsById.size).toBe(0);
  });

  it("aborts instead of looping forever on a self-referential links.next", async () => {
    // A malformed backend response that points `next` back at the URL that
    // was just fetched. Without a visited-URL guard this would fetch forever.
    getListDocumentMock.mockImplementation((url: string) =>
      Promise.resolve({
        data: [{ id: "100", type: "jobs", attributes: { skills: [] } }],
        links: { next: url },
      }),
    );

    await expect(jobsService.getJobs()).rejects.toThrow(/aborting pagination/);
    // Caught on the second fetch (the repeat of the first URL), not looped.
    expect(getListDocumentMock).toHaveBeenCalledTimes(1);
  });

  it("aborts once the page-count backstop is reached even without an exact repeat", async () => {
    // Every page advances to a distinct URL (so the visited-Set repeat check
    // never fires) but never terminates. The MAX_PAGES ceiling must still
    // stop it rather than requesting forever.
    let page = 0;
    getListDocumentMock.mockImplementation(() => {
      page += 1;
      return Promise.resolve({
        data: [{ id: String(page), type: "jobs", attributes: { skills: [] } }],
        links: { next: `/api/data/jobs?page%5Bnumber%5D=${page + 1}` },
      });
    });

    await expect(jobsService.getJobs()).rejects.toThrow(/aborting pagination/);
    expect(getListDocumentMock.mock.calls.length).toBeLessThan(100);
  });
});
