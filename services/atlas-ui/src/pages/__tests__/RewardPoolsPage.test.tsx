import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { rewardPoolsService } from "@/services/api/reward-pools.service";
import { RewardPoolsPage } from "../RewardPoolsPage";

// vi.mock factories are hoisted above top-level const declarations, so the
// pool fixtures referenced inside the factory below must be wrapped in
// vi.hoisted (see MarketplacePage.test.tsx for the same idiom).
const { pools } = vi.hoisted(() => ({
  pools: [
    { id: "henesys", type: "gachapons", attributes: { name: "Henesys", kind: "gachapon", npcIds: [9100100], commonWeight: 70, uncommonWeight: 25, rareWeight: 5 } },
    { id: "4170001", type: "gachapons", attributes: { name: "Pigmy Egg", kind: "incubator", npcIds: [1012004], commonWeight: 0, uncommonWeight: 0, rareWeight: 0 } },
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
  // Egg-name resolution falls back to the pool's seeded name when undefined.
  // Pool id "4170001" maps to the Ellinia egg region (egg-regions.ts), so the
  // fixture's "Pigmy Egg" fallback renders region-appended as
  // "Pigmy Egg (Ellinia)" — the assertions below expect that exact string.
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
    expect(screen.getByText("Pigmy Egg (Ellinia)")).toBeInTheDocument();
    expect(screen.getAllByText(/gachapon/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/incubator/i).length).toBeGreaterThan(0);
  });

  it("renders a single refresh control next to the tabs", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText("Henesys")).toBeInTheDocument());
    expect(screen.getByRole("button", { name: /refresh/i })).toBeInTheDocument();
  });

  it("Incubators tab filters out gachapon pools", async () => {
    const user = userEvent.setup();
    renderPage();
    await waitFor(() => expect(screen.getByText("Henesys")).toBeInTheDocument());
    await user.click(screen.getByRole("tab", { name: /incubators/i }));
    expect(screen.queryByText("Henesys")).not.toBeInTheDocument();
    expect(screen.getByText("Pigmy Egg (Ellinia)")).toBeInTheDocument();
  });

  it("Global Pool tab lists global items", async () => {
    const user = userEvent.setup();
    renderPage();
    await waitFor(() => expect(screen.getByText("Henesys")).toBeInTheDocument());
    await user.click(screen.getByRole("tab", { name: /global pool/i }));
    await waitFor(() => expect(screen.getByText("item-2000000")).toBeInTheDocument());
  });

  it("Global Pool tab surfaces a fetch error instead of a false empty state", async () => {
    // The global-items query fires at mount (tenant present), so the rejection
    // must be queued before renderPage(); Once keeps the other tests' default
    // resolved mock intact.
    vi.mocked(rewardPoolsService.getGlobalItems).mockRejectedValueOnce(new Error("global items unavailable"));
    const user = userEvent.setup();
    renderPage();
    await waitFor(() => expect(screen.getByText("Henesys")).toBeInTheDocument());
    await user.click(screen.getByRole("tab", { name: /global pool/i }));
    await waitFor(() => expect(screen.getByTestId("error-display")).toBeInTheDocument());
    expect(screen.getByText("global items unavailable")).toBeInTheDocument();
    expect(screen.queryByText("No global items.")).not.toBeInTheDocument();
  });
});
