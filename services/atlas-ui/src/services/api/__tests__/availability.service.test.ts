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
        { id: "100", type: "job-availability", attributes: { name: "Warrior" } },
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
});
