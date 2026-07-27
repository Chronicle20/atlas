import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { PossibleRewardsCard } from "../PossibleRewardsCard";
import type { RewardModel } from "@/types/models/item";

// Mock the data hook so no tenant/network is needed; name echoes the itemId.
vi.mock("@/lib/hooks/useItemData", () => ({
  useItemData: (itemId: number) => ({
    name: `Item ${itemId}`,
    iconUrl: undefined,
    isLoading: false,
  }),
}));

function mk(over: Partial<RewardModel>): RewardModel {
  return {
    itemId: 1000,
    count: 1,
    prob: 100,
    effect: "",
    worldMsg: "",
    period: 0,
    ...over,
  };
}

function wrap(children: React.ReactNode) {
  return <MemoryRouter>{children}</MemoryRouter>;
}

describe("PossibleRewardsCard", () => {
  it("renders nothing when there are no rewards", () => {
    const { container } = render(wrap(<PossibleRewardsCard rewards={[]} />));
    expect(container.firstChild).toBeNull();
  });

  it("computes chance as prob/total and shows the count in the title", () => {
    render(
      wrap(
        <PossibleRewardsCard
          rewards={[mk({ itemId: 1, prob: 30 }), mk({ itemId: 2, prob: 10 })]}
        />,
      ),
    );
    expect(screen.getByText("Possible Rewards (2)")).toBeInTheDocument();
    expect(screen.getByText("75.000%")).toBeInTheDocument(); // 30 / 40
    expect(screen.getByText("25.000%")).toBeInTheDocument(); // 10 / 40
  });

  it("sorts rows by chance descending", () => {
    render(
      wrap(
        <PossibleRewardsCard
          rewards={[mk({ itemId: 1, prob: 10 }), mk({ itemId: 2, prob: 90 })]}
        />,
      ),
    );
    const pcts = screen.getAllByText(/%$/).map((el) => el.textContent);
    expect(pcts).toEqual(["90.000%", "10.000%"]);
  });

  it("guards total=0 without producing NaN", () => {
    render(
      wrap(
        <PossibleRewardsCard
          rewards={[mk({ itemId: 1, prob: 0 }), mk({ itemId: 2, prob: 0 })]}
        />,
      ),
    );
    expect(screen.getAllByText("0.000%").length).toBe(2);
  });

  it("renders a rare (sub-0.01%) chance at 3-decimal fidelity, not rounded up to 0.01%", () => {
    // 1 / 20000 = 0.005%; at 2 decimals this would wrongly read 0.01% (2× overstated).
    render(
      wrap(
        <PossibleRewardsCard
          rewards={[mk({ itemId: 1, prob: 1 }), mk({ itemId: 2, prob: 19999 })]}
        />,
      ),
    );
    expect(screen.getByText("0.005%")).toBeInTheDocument();
  });

  it("shows ×count, time-limited and announces when applicable", () => {
    render(
      wrap(
        <PossibleRewardsCard
          rewards={[
            mk({
              itemId: 1,
              prob: 100,
              count: 3,
              period: 7200,
              worldMsg: "/name got /item",
            }),
          ]}
        />,
      ),
    );
    expect(screen.getByText("×3")).toBeInTheDocument();
    expect(screen.getByText("time-limited")).toBeInTheDocument();
    expect(screen.getByText("announces")).toBeInTheDocument();
  });

  it("omits ×count, time-limited and announces when not applicable", () => {
    render(
      wrap(
        <PossibleRewardsCard
          rewards={[mk({ itemId: 1, count: 1, period: 0, worldMsg: "" })]}
        />,
      ),
    );
    expect(screen.queryByText("time-limited")).toBeNull();
    expect(screen.queryByText("announces")).toBeNull();
    expect(screen.queryByText(/^×/)).toBeNull();
  });

  it("links each reward row to its item detail page", () => {
    render(
      wrap(
        <PossibleRewardsCard rewards={[mk({ itemId: 2041303, prob: 100 })]} />,
      ),
    );
    expect(screen.getByRole("link")).toHaveAttribute("href", "/items/2041303");
  });
});
