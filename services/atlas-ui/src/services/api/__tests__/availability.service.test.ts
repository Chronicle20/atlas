import { describe, it, expect, vi, beforeEach } from "vitest";

const getListDocumentMock = vi.fn();
vi.mock("@/lib/api/client", () => ({
  api: {
    getListDocument: (...args: unknown[]) => getListDocumentMock(...args),
  },
}));

import { availabilityService } from "@/services/api/availability.service";

beforeEach(() => vi.clearAllMocks());

describe("availabilityService.getJobAvailability", () => {
  it("requests job-availability and maps resource id (wire id) + attributes.name", async () => {
    getListDocumentMock.mockResolvedValue({
      data: [
        { id: "500", type: "job-availability", attributes: { name: "Gm" } },
        {
          id: "100",
          type: "job-availability",
          attributes: { name: "Warrior" },
        },
      ],
    });

    const result = await availabilityService.getJobAvailability();

    expect(getListDocumentMock).toHaveBeenCalledTimes(1);
    const url = getListDocumentMock.mock.calls[0]?.[0] as string;
    expect(url).toContain("/api/data/job-availability");
    expect(result).toEqual([
      { id: 500, name: "Gm" },
      { id: 100, name: "Warrior" },
    ]);
  });

  it("returns an empty array when data is absent", async () => {
    getListDocumentMock.mockResolvedValue({});
    const result = await availabilityService.getJobAvailability();
    expect(result).toEqual([]);
  });

  it("follows links.next until exhausted", async () => {
    getListDocumentMock
      .mockResolvedValueOnce({
        data: [
          { id: "500", type: "job-availability", attributes: { name: "Gm" } },
        ],
        links: {
          next: "/api/data/job-availability?page%5Bnumber%5D=2&page%5Bsize%5D=250",
        },
      })
      .mockResolvedValueOnce({
        data: [
          {
            id: "100",
            type: "job-availability",
            attributes: { name: "Warrior" },
          },
        ],
        links: {},
      });

    const result = await availabilityService.getJobAvailability();

    expect(getListDocumentMock).toHaveBeenCalledTimes(2);
    expect(getListDocumentMock.mock.calls[1]?.[0]).toBe(
      "/api/data/job-availability?page%5Bnumber%5D=2&page%5Bsize%5D=250",
    );
    expect(result).toEqual([
      { id: 500, name: "Gm" },
      { id: 100, name: "Warrior" },
    ]);
  });

  it("aborts instead of looping forever on a self-referential links.next", async () => {
    getListDocumentMock.mockImplementation((url: string) =>
      Promise.resolve({
        data: [
          { id: "500", type: "job-availability", attributes: { name: "Gm" } },
        ],
        links: { next: url },
      }),
    );

    await expect(availabilityService.getJobAvailability()).rejects.toThrow(
      /aborting pagination/,
    );
    expect(getListDocumentMock).toHaveBeenCalledTimes(1);
  });
});

describe("availabilityService.getSkillAvailability", () => {
  it("requests skill-availability and maps resource id (wire id) + attributes.name", async () => {
    getListDocumentMock.mockResolvedValue({
      data: [
        {
          id: "1121000",
          type: "skill-availability",
          attributes: { name: "Power Strike" },
        },
      ],
    });

    const result = await availabilityService.getSkillAvailability();

    expect(getListDocumentMock).toHaveBeenCalledTimes(1);
    const url = getListDocumentMock.mock.calls[0]?.[0] as string;
    expect(url).toContain("/api/data/skill-availability");
    expect(result).toEqual([{ id: 1121000, name: "Power Strike" }]);
  });

  it("follows links.next until exhausted (skills can exceed one page)", async () => {
    getListDocumentMock
      .mockResolvedValueOnce({
        data: [
          {
            id: "1121000",
            type: "skill-availability",
            attributes: { name: "Power Strike" },
          },
        ],
        links: {
          next: "/api/data/skill-availability?page%5Bnumber%5D=2&page%5Bsize%5D=250",
        },
      })
      .mockResolvedValueOnce({
        data: [
          {
            id: "1121001",
            type: "skill-availability",
            attributes: { name: "Slash Blast" },
          },
        ],
        links: {},
      });

    const result = await availabilityService.getSkillAvailability();

    expect(getListDocumentMock).toHaveBeenCalledTimes(2);
    expect(result).toEqual([
      { id: 1121000, name: "Power Strike" },
      { id: 1121001, name: "Slash Blast" },
    ]);
  });
});
