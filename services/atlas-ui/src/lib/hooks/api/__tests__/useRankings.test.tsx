import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { rankingsKeys, useRankings } from "@/lib/hooks/api/useRankings";
import type { RankingPage } from "@/services/api/rankings.service";

vi.mock("@/services/api/rankings.service", () => ({
  rankingsService: { leaderboard: vi.fn() },
}));

import { rankingsService } from "@/services/api/rankings.service";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  const TestWrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  TestWrapper.displayName = "TestWrapper";
  return TestWrapper;
}

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

describe("useRankings", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("calls rankingsService.leaderboard with worldId and filter", async () => {
    const mockPage: RankingPage = { entries: [], total: 0, lastPage: 1 };
    (rankingsService.leaderboard as ReturnType<typeof vi.fn>).mockResolvedValue(
      mockPage,
    );

    const filter = { jobCategory: 1, page: 0, pageSize: 25 };
    const { result } = renderHook(() => useRankings("t1", 2, filter), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
      expect(result.current.data).toEqual(mockPage);
    });

    expect(rankingsService.leaderboard).toHaveBeenCalledWith(2, filter);
  });

  it("does not call rankingsService.leaderboard when enabled=false", () => {
    const filter = { jobCategory: 1, page: 0, pageSize: 25 };
    const { result } = renderHook(() => useRankings("t1", 2, filter, false), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(false);
    expect(result.current.data).toBeUndefined();
    expect(rankingsService.leaderboard).not.toHaveBeenCalled();
  });
});
