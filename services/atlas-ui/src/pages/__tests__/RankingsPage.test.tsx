import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RankingsPage } from "@/pages/RankingsPage";
import type { RankingEntry } from "@/services/api/rankings.service";

// Radix Select relies on DOM APIs jsdom does not implement.
Element.prototype.hasPointerCapture ||= () => false;
Element.prototype.scrollIntoView ||= () => {};

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t1" } }),
}));
vi.mock("@/lib/hooks/api/useTenants", () => ({
  useTenantConfiguration: () => ({
    data: { attributes: { worlds: [{ name: "Scania" }] } },
  }),
}));

const useRankingsMock = vi.fn();
vi.mock("@/lib/hooks/api/useRankings", () => ({
  useRankings: (...args: unknown[]) => useRankingsMock(...args),
}));

vi.mock("@/components/features/rankings/LeaderboardRow", () => ({
  LeaderboardRow: ({ entry, view }: { entry: RankingEntry; view: string }) => (
    <tr data-testid={`row-${entry.id}`}>
      <td>{entry.attributes.name}</td>
      <td>{view}</td>
    </tr>
  ),
}));

function entry(id: string, name: string): RankingEntry {
  return {
    id,
    attributes: {
      characterId: Number(id),
      name,
      worldId: 0,
      level: 50,
      jobId: 110,
      jobCategory: 1,
      rank: Number(id),
      rankMove: 0,
      jobRank: Number(id),
      jobRankMove: 0,
      computedAt: "",
    },
  };
}

describe("RankingsPage", () => {
  beforeEach(() => {
    useRankingsMock.mockReset();
    useRankingsMock.mockReturnValue({
      data: { entries: [], total: 0, lastPage: 1 },
      isLoading: false,
      isError: false,
    });
  });

  it("renders the leaderboard heading and world selector", () => {
    render(<RankingsPage />);
    expect(screen.getByText(/Rankings/i)).toBeInTheDocument();
  });

  it("renders a row per populated leaderboard entry", () => {
    useRankingsMock.mockReturnValue({
      data: {
        entries: [entry("1", "Alice"), entry("2", "Bob")],
        total: 2,
        lastPage: 1,
      },
      isLoading: false,
      isError: false,
    });

    render(<RankingsPage />);

    expect(screen.getByTestId("row-1")).toBeInTheDocument();
    expect(screen.getByTestId("row-2")).toBeInTheDocument();
    expect(screen.getByText("Alice")).toBeInTheDocument();
    expect(screen.getByText("Bob")).toBeInTheDocument();
  });

  it("shows the error text and no table when the query errors", () => {
    useRankingsMock.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
    });

    render(<RankingsPage />);

    expect(screen.getByText("Failed to load rankings.")).toBeInTheDocument();
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });

  it("shows a loading indicator while the query is loading", () => {
    useRankingsMock.mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
    });

    render(<RankingsPage />);

    expect(screen.getByText(/loading/i)).toBeInTheDocument();
  });

  it("disables Previous on the first page and Next when lastPage=1", () => {
    useRankingsMock.mockReturnValue({
      data: { entries: [], total: 0, lastPage: 1 },
      isLoading: false,
      isError: false,
    });

    render(<RankingsPage />);

    expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Next" })).toBeDisabled();
  });

  it("enables Next when more pages remain", () => {
    useRankingsMock.mockReturnValue({
      data: { entries: [], total: 100, lastPage: 4 },
      isLoading: false,
      isError: false,
    });

    render(<RankingsPage />);

    expect(screen.getByRole("button", { name: "Next" })).toBeEnabled();
  });

  it("selecting a job category flips the view and filter passed to useRankings", async () => {
    const user = userEvent.setup();
    render(<RankingsPage />);

    const [, jobTrigger] = screen.getAllByRole("combobox");
    await user.click(jobTrigger!);
    await user.click(await screen.findByRole("option", { name: "Warrior" }));

    const lastCall =
      useRankingsMock.mock.calls[useRankingsMock.mock.calls.length - 1]!;
    const [, worldId, filter] = lastCall as [
      string,
      number,
      { jobCategory?: number },
    ];
    expect(worldId).toBe(0);
    expect(filter.jobCategory).toBe(1);
  });
});
