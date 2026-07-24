import { describe, expect, it, vi, beforeEach } from "vitest";
import { apiClient } from "@/lib/api/client";
import { rankingsService } from "@/services/api/rankings.service";

vi.mock("@/lib/api/client", () => ({
  apiClient: { get: vi.fn() },
}));

describe("rankingsService.leaderboard", () => {
  beforeEach(() => vi.clearAllMocks());

  it("requests the leaderboard with world + paging params and returns total/lastPage from meta", async () => {
    (apiClient.get as ReturnType<typeof vi.fn>).mockResolvedValue({
      data: [{ id: "2", attributes: { characterId: 2, name: "B", worldId: 0, level: 50, jobId: 110, jobCategory: 1, rank: 1, rankMove: 0, jobRank: 1, jobRankMove: 0, computedAt: "" } }],
      meta: { total: 1, page: { last: 1 } },
    });
    const res = await rankingsService.leaderboard(0, { page: 0, pageSize: 25 });
    expect(apiClient.get).toHaveBeenCalledWith(
      "/api/rankings?filter%5BworldId%5D=0&page%5Bnumber%5D=1&page%5Bsize%5D=25",
      undefined,
    );
    expect(res.total).toBe(1);
    expect(res.entries[0]!.attributes.characterId).toBe(2);
  });

  it("includes the job category filter when set", async () => {
    (apiClient.get as ReturnType<typeof vi.fn>).mockResolvedValue({ data: [], meta: { total: 0, page: { last: 1 } } });
    await rankingsService.leaderboard(0, { jobCategory: 1, page: 0, pageSize: 25 });
    expect(apiClient.get).toHaveBeenCalledWith(
      expect.stringContaining("filter%5BjobCategory%5D=1"),
      undefined,
    );
  });
});
