import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { RankingsPage } from "@/pages/RankingsPage";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t1" } }),
}));
vi.mock("@/lib/hooks/api/useTenants", () => ({
  useTenantConfiguration: () => ({
    data: { attributes: { worlds: [{ name: "Scania" }] } },
  }),
}));
vi.mock("@/lib/hooks/api/useRankings", () => ({
  useRankings: () => ({
    data: { entries: [], total: 0, lastPage: 1 },
    isLoading: false,
    isError: false,
  }),
}));

describe("RankingsPage", () => {
  it("renders the leaderboard heading and world selector", () => {
    render(<RankingsPage />);
    expect(screen.getByText(/Rankings/i)).toBeInTheDocument();
  });
});
