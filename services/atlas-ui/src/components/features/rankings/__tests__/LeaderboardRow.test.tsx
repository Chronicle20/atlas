import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { LeaderboardRow } from "@/components/features/rankings/LeaderboardRow";
import type { RankingEntry } from "@/services/api/rankings.service";

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
vi.mock("@/components/features/characters/OptimizedCharacterRenderer", () => ({
  OptimizedCharacterRenderer: () => <div data-testid="renderer" />,
}));

function row(over: number, move: number): RankingEntry {
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
      jobRank: 1,
      jobRankMove: 0,
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
});
