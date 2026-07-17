import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RewardPoolDetailPage } from "../RewardPoolDetailPage";

const henesys = { id: "henesys", type: "gachapons", attributes: { name: "Henesys", kind: "gachapon", npcIds: [9100100], commonWeight: 70, uncommonWeight: 25, rareWeight: 5 } };
const egg = { id: "4170001", type: "gachapons", attributes: { name: "Pigmy Egg (Victoria)", kind: "incubator", npcIds: [1012004], commonWeight: 0, uncommonWeight: 0, rareWeight: 0 } };

const mocks = vi.hoisted(() => ({
  getPoolById: vi.fn(),
  getItems: vi.fn(),
  getGlobalItems: vi.fn().mockResolvedValue([]),
}));
vi.mock("@/services/api/reward-pools.service", () => ({ rewardPoolsService: mocks }));
vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } }),
}));
vi.mock("@/components/item-name-cell", () => ({
  ItemNameCell: ({ itemId }: { itemId: string }) => <span>item-{itemId}</span>,
}));
vi.mock("@/lib/hooks/api/useItemStrings", () => ({
  useItemName: () => ({ data: "Pigmy Egg" }),
}));
vi.mock("@/lib/hooks/api/useNpcs", () => ({
  useNPC: () => ({ data: { attributes: { name: "Pigmy & Etran" } } }),
}));

function renderAt(id: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter initialEntries={[`/reward-pools/${id}`]}>
      <QueryClientProvider client={qc}>
        <Routes>
          <Route path="/reward-pools/:id" element={<RewardPoolDetailPage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("RewardPoolDetailPage", () => {
  it("gachapon: shows tier weights card and tier-grouped pool with global rows badged", async () => {
    mocks.getPoolById.mockResolvedValue(henesys);
    mocks.getItems.mockResolvedValue([
      { id: "1", type: "gachapon-items", attributes: { gachaponId: "henesys", itemId: 2000000, quantity: 1, tier: "common", weight: 0 } },
    ]);
    mocks.getGlobalItems.mockResolvedValue([
      { id: "9", type: "global-gachapon-items", attributes: { itemId: 2000001, quantity: 1, tier: "common" } },
    ]);
    renderAt("henesys");
    await waitFor(() => expect(screen.getByText("Henesys")).toBeInTheDocument());
    expect(screen.getByText(/tier weights/i)).toBeInTheDocument();
    expect(screen.getByText("item-2000001")).toBeInTheDocument();
    expect(screen.getByText(/global/i)).toBeInTheDocument();
    // two common rows, uniform within a 70% tier → 35.00% each
    expect(screen.getAllByText("35.00%").length).toBe(2);
  });

  it("incubator: shows egg card, weight column, weight-based chance; no tier weights card", async () => {
    mocks.getPoolById.mockResolvedValue(egg);
    mocks.getItems.mockResolvedValue([
      { id: "1", type: "gachapon-items", attributes: { gachaponId: "4170001", itemId: 2000000, quantity: 1, tier: "common", weight: 75 } },
      { id: "2", type: "gachapon-items", attributes: { gachaponId: "4170001", itemId: 1302000, quantity: 1, tier: "common", weight: 25 } },
    ]);
    mocks.getGlobalItems.mockResolvedValue([]);
    renderAt("4170001");
    await waitFor(() => expect(screen.getByText("Pigmy Egg (Victoria)")).toBeInTheDocument());
    expect(screen.queryByText(/tier weights/i)).not.toBeInTheDocument();
    expect(screen.getByText(/success npc/i)).toBeInTheDocument();
    expect(screen.getByText("75.00%")).toBeInTheDocument();
    expect(screen.getByText("25.00%")).toBeInTheDocument();
  });

  it("warns when a gachapon tier mixes weighted and zero-weight rows", async () => {
    mocks.getPoolById.mockResolvedValue(henesys);
    mocks.getItems.mockResolvedValue([
      { id: "1", type: "gachapon-items", attributes: { gachaponId: "henesys", itemId: 2000000, quantity: 1, tier: "rare", weight: 10 } },
      { id: "2", type: "gachapon-items", attributes: { gachaponId: "henesys", itemId: 2000001, quantity: 1, tier: "rare", weight: 0 } },
    ]);
    mocks.getGlobalItems.mockResolvedValue([]);
    renderAt("henesys");
    await waitFor(() => expect(screen.getByText("Henesys")).toBeInTheDocument());
    expect(screen.getByText(/exclude/i)).toBeInTheDocument();
  });
});
