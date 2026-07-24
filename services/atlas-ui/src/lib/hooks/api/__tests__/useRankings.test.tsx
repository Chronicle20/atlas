import { describe, expect, it, vi } from "vitest";
import { rankingsKeys } from "@/lib/hooks/api/useRankings";

vi.mock("@/services/api/rankings.service", () => ({
  rankingsService: { leaderboard: vi.fn() },
}));

describe("rankingsKeys", () => {
  it("scopes the cache by tenant, world and filter", () => {
    const key = rankingsKeys.leaderboard("t1", 0, {
      jobCategory: 1,
      page: 0,
      pageSize: 25,
    });
    expect(key[0]).toBe("rankings");
    expect(key).toContain("t1");
    expect(key).toContain(0);
  });
});
