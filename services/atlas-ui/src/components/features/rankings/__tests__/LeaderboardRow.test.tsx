import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { LeaderboardRow } from "@/components/features/rankings/LeaderboardRow";
import type { RankingEntry } from "@/services/api/rankings.service";
import { FIXTURE_JOB_TREE } from "@/lib/jobs/__tests__/job-graph-fixtures";

// The job badge's name comes from the tenant's job graph via
// useJobNameLookup; mock it to the structural fixture rather than standing
// up a QueryClientProvider for a component test that isn't about the graph.
vi.mock("@/lib/hooks/api/useJobGraph", () => ({
  useJobNameLookup: () => (id: number) =>
    FIXTURE_JOB_TREE[id]?.name ?? `Job ${id}`,
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));
vi.mock("@/lib/hooks/api/useCharacters", () => ({
  useCharacter: () => ({ data: undefined, isLoading: false, isError: true }),
}));
vi.mock("@/lib/hooks/api/useInventory", () => ({
  useInventory: () => ({ data: undefined, isLoading: false, isError: false }),
}));
vi.mock("@/components/features/characters/OptimizedCharacterRenderer", () => ({
  OptimizedCharacterRenderer: () => <div data-testid="renderer" />,
}));

function row(
  over: number,
  move: number,
  jobOver = 1,
  jobMove = 0,
): RankingEntry {
  return {
    id: String(over),
    attributes: {
      characterId: 2,
      name: "B",
      worldId: 0,
      level: 50,
      jobId: 110,
      jobCategory: 1,
      rank: over,
      rankMove: move,
      jobRank: jobOver,
      jobRankMove: jobMove,
      computedAt: "",
    },
  };
}

describe("LeaderboardRow", () => {
  it("renders rank, name and level even when the character render is unavailable", () => {
    render(
      <table>
        <tbody>
          <LeaderboardRow entry={row(1, 0)} view="overall" />
        </tbody>
      </table>,
    );
    expect(screen.getByText("B")).toBeInTheDocument();
    expect(screen.getByText("50")).toBeInTheDocument();
    expect(screen.getByText("#1")).toBeInTheDocument();
  });

  it("shows an up arrow when rankMove is positive", () => {
    render(
      <table>
        <tbody>
          <LeaderboardRow entry={row(1, 3)} view="overall" />
        </tbody>
      </table>,
    );
    expect(screen.getByLabelText("moved up")).toBeInTheDocument();
  });

  it("shows a down arrow when rankMove is negative", () => {
    render(
      <table>
        <tbody>
          <LeaderboardRow entry={row(1, -2)} view="overall" />
        </tbody>
      </table>,
    );
    expect(screen.getByLabelText("moved down")).toBeInTheDocument();
  });

  it("shows a no-change indicator when rankMove is zero", () => {
    render(
      <table>
        <tbody>
          <LeaderboardRow entry={row(1, 0)} view="overall" />
        </tbody>
      </table>,
    );
    expect(screen.getByLabelText("no change")).toBeInTheDocument();
  });

  it("renders the job as a human-readable name badge, not the raw id", () => {
    render(
      <table>
        <tbody>
          <LeaderboardRow entry={row(1, 0)} view="overall" />
        </tbody>
      </table>,
    );
    // jobId 110 -> "Fighter" via the job graph; the raw id must not be shown.
    expect(screen.getByText("Fighter")).toBeInTheDocument();
    expect(screen.queryByText("110")).not.toBeInTheDocument();
  });

  it("uses jobRank/jobRankMove instead of rank/rankMove when view is job", () => {
    render(
      <table>
        <tbody>
          <LeaderboardRow entry={row(1, 5, 7, -4)} view="job" />
        </tbody>
      </table>,
    );
    expect(screen.getByText("#7")).toBeInTheDocument();
    expect(screen.getByLabelText("moved down")).toBeInTheDocument();
  });
});
