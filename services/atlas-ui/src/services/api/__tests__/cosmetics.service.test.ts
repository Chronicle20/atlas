import { describe, it, expect, vi, beforeEach } from "vitest";

const fetchAllMock = vi.fn();
vi.mock("@/services/api/pagination", () => ({
  fetchAll: (...a: unknown[]) => fetchAllMock(...a),
}));

import { cosmeticsService } from "@/services/api/cosmetics.service";

beforeEach(() => fetchAllMock.mockReset());

describe("cosmeticsService", () => {
  it("enumerates faces via fetchAll and returns sorted numeric ids", async () => {
    fetchAllMock.mockResolvedValue([
      { id: "20001", attributes: { cash: false } },
      { id: "20000", attributes: { cash: false } },
      { id: "21000", attributes: { cash: true } },
    ]);
    await expect(cosmeticsService.getAllFaceIds()).resolves.toEqual([
      20000, 20001, 21000,
    ]);
    expect(fetchAllMock).toHaveBeenCalledWith("/api/data/cosmetics/faces");
  });

  it("enumerates hairs and drops non-numeric ids", async () => {
    fetchAllMock.mockResolvedValue([
      { id: "30030", attributes: { cash: false } },
      { id: "bogus", attributes: { cash: false } },
    ]);
    await expect(cosmeticsService.getAllHairIds()).resolves.toEqual([30030]);
    expect(fetchAllMock).toHaveBeenCalledWith("/api/data/cosmetics/hairs");
  });
});
