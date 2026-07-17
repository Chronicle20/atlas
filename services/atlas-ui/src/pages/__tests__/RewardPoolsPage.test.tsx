import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RewardPoolsPage } from "../RewardPoolsPage";

// vi.mock factories are hoisted above top-level const declarations, so the
// pool fixtures referenced inside the factory below must be wrapped in
// vi.hoisted (see MarketplacePage.test.tsx for the same idiom).
const { pools } = vi.hoisted(() => ({
  pools: [
    { id: "henesys", type: "gachapons", attributes: { name: "Henesys", kind: "gachapon", npcIds: [9100100], commonWeight: 70, uncommonWeight: 25, rareWeight: 5 } },
    { id: "4170001", type: "gachapons", attributes: { name: "Pigmy Egg (Victoria)", kind: "incubator", npcIds: [1012004], commonWeight: 0, uncommonWeight: 0, rareWeight: 0 } },
  ],
}));
vi.mock("@/services/api/reward-pools.service", () => ({
  rewardPoolsService: {
    getAllPools: vi.fn().mockResolvedValue(pools),
    getGlobalItems: vi.fn().mockResolvedValue([
      { id: "1", type: "global-gachapon-items", attributes: { itemId: 2000000, quantity: 1, tier: "common" } },
    ]),
  },
}));
vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } }),
}));
vi.mock("@/components/item-name-cell", () => ({
  ItemNameCell: ({ itemId }: { itemId: string }) => <span>item-{itemId}</span>,
}));
vi.mock("@/lib/hooks/api/useItemStrings", () => ({
  // Egg-name resolution falls back to the pool's seeded name when undefined —
  // the assertions below rely on that fallback.
  useItemName: () => ({ data: undefined }),
}));

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={qc}>
        <RewardPoolsPage />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("RewardPoolsPage", () => {
  it("shows both pools on the All tab with kind badges", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText("Henesys")).toBeInTheDocument());
    expect(screen.getByText("Pigmy Egg (Victoria)")).toBeInTheDocument();
    expect(screen.getAllByText(/gachapon/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/incubator/i).length).toBeGreaterThan(0);
  });

  it("Incubators tab filters out gachapon pools", async () => {
    const user = userEvent.setup();
    renderPage();
    await waitFor(() => expect(screen.getByText("Henesys")).toBeInTheDocument());
    await user.click(screen.getByRole("tab", { name: /incubators/i }));
    expect(screen.queryByText("Henesys")).not.toBeInTheDocument();
    expect(screen.getByText("Pigmy Egg (Victoria)")).toBeInTheDocument();
  });

  it("Global Pool tab lists global items", async () => {
    const user = userEvent.setup();
    renderPage();
    await waitFor(() => expect(screen.getByText("Henesys")).toBeInTheDocument());
    await user.click(screen.getByRole("tab", { name: /global pool/i }));
    await waitFor(() => expect(screen.getByText("item-2000000")).toBeInTheDocument());
  });
});
