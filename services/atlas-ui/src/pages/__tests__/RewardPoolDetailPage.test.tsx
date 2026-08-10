import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RewardPoolDetailPage } from "../RewardPoolDetailPage";

const henesys = {
  id: "henesys",
  type: "gachapons",
  attributes: {
    name: "Henesys",
    kind: "gachapon",
    npcIds: [9100100],
    commonWeight: 70,
    uncommonWeight: 25,
    rareWeight: 5,
  },
};
const egg = {
  id: "4170001",
  type: "gachapons",
  attributes: {
    name: "Pigmy Egg (Victoria)",
    kind: "incubator",
    npcIds: [1012004],
    commonWeight: 0,
    uncommonWeight: 0,
    rareWeight: 0,
  },
};
const surpriseBox = {
  id: "5910000",
  type: "gachapons",
  attributes: {
    name: "Surprise Style Box",
    kind: "cash-surprise",
    npcIds: [],
    commonWeight: 0,
    uncommonWeight: 0,
    rareWeight: 0,
  },
};

const mocks = vi.hoisted(() => ({
  getPoolById: vi.fn(),
  getItems: vi.fn(),
  getGlobalItems: vi.fn().mockResolvedValue([]),
}));
vi.mock("@/services/api/reward-pools.service", () => ({
  rewardPoolsService: mocks,
}));
vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
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
// NpcRow renders NpcImage, which lazy-loads via IntersectionObserver — not
// available in jsdom. Same stub pattern as InventoryGrid.test.tsx.
vi.mock("@/lib/hooks/useIntersectionObserver", () => ({
  useLazyLoad: () => ({ shouldLoad: true, ref: { current: null } }),
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
  it("gachapon: shows tier weights card and a flat item grid with global rows badged", async () => {
    mocks.getPoolById.mockResolvedValue(henesys);
    mocks.getItems.mockResolvedValue([
      {
        id: "1",
        type: "gachapon-items",
        attributes: {
          gachaponId: "henesys",
          itemId: 2000000,
          quantity: 1,
          tier: "common",
          weight: 0,
        },
      },
    ]);
    mocks.getGlobalItems.mockResolvedValue([
      {
        id: "9",
        type: "global-gachapon-items",
        attributes: { itemId: 2000001, quantity: 1, tier: "common" },
      },
    ]);
    renderAt("henesys");
    await waitFor(() =>
      expect(screen.getByText("Henesys")).toBeInTheDocument(),
    );
    expect(screen.getByText(/tier weights/i)).toBeInTheDocument();
    expect(screen.getByText("item-2000001")).toBeInTheDocument();
    expect(screen.getByText(/global/i)).toBeInTheDocument();
    // two common rows, uniform within a 70% tier → 35.00% each
    expect(screen.getAllByText("35.00%").length).toBe(2);
  });

  it("incubator: shows region-formatted header name, weight column, weight-based chance; no tier weights card", async () => {
    mocks.getPoolById.mockResolvedValue(egg);
    mocks.getItems.mockResolvedValue([
      {
        id: "1",
        type: "gachapon-items",
        attributes: {
          gachaponId: "4170001",
          itemId: 2000000,
          quantity: 1,
          tier: "common",
          weight: 75,
        },
      },
      {
        id: "2",
        type: "gachapon-items",
        attributes: {
          gachaponId: "4170001",
          itemId: 1302000,
          quantity: 1,
          tier: "common",
          weight: 25,
        },
      },
    ]);
    mocks.getGlobalItems.mockResolvedValue([]);
    renderAt("4170001");
    // useItemName is mocked to "Pigmy Egg"; formatIncubatorName appends the
    // region for id 4170001 (Ellinia), regardless of the pool's own attrs.name.
    await waitFor(() =>
      expect(screen.getByText("Pigmy Egg (Ellinia)")).toBeInTheDocument(),
    );
    expect(screen.queryByText(/tier weights/i)).not.toBeInTheDocument();
    expect(screen.getByText("75.00%")).toBeInTheDocument();
    expect(screen.getByText("25.00%")).toBeInTheDocument();
  });

  it("cash-surprise: flat item table with Commodity column, plain header name, item icon (not egg-formatted)", async () => {
    mocks.getPoolById.mockResolvedValue(surpriseBox);
    mocks.getItems.mockResolvedValue([
      {
        id: "1",
        type: "gachapon-items",
        attributes: {
          gachaponId: "5910000",
          itemId: 5510000,
          quantity: 1,
          tier: "common",
          weight: 100,
          commodityId: 5300000,
        },
      },
    ]);
    mocks.getGlobalItems.mockResolvedValue([]);
    const { container } = renderAt("5910000");
    await waitFor(() =>
      expect(screen.getByText("Surprise Style Box")).toBeInTheDocument(),
    );
    // Plain seeded name -- the egg-region formatting (e.g. "(Ellinia)") is
    // an incubator-only convention (task-207 F1) and must not apply here,
    // even though useItemName is globally mocked to return a truthy name.
    expect(
      screen.queryByText(/\(Ellinia\)|\(Victoria\)/),
    ).not.toBeInTheDocument();
    // No tier-weights/NPC cards -- cash-surprise uses the flat layout.
    expect(screen.queryByText(/tier weights/i)).not.toBeInTheDocument();
    // Flat table renders the Commodity column and its value.
    expect(screen.getByText("Commodity")).toBeInTheDocument();
    expect(screen.getByText("5300000")).toBeInTheDocument();
    // Header icon resolves via the "item" source (pool id), not the
    // gachapon NPC-icon branch (which would need npcIds populated). The
    // header <img alt=""> is presentational (no accessible "img" role), so
    // it's queried directly rather than via getByRole.
    const headerImg = container.querySelector("img[width='32']");
    expect(headerImg).toHaveAttribute(
      "src",
      expect.stringContaining("/item/5910000/icon.png"),
    );
  });

  it("warns when a gachapon tier mixes weighted and zero-weight rows", async () => {
    mocks.getPoolById.mockResolvedValue(henesys);
    mocks.getItems.mockResolvedValue([
      {
        id: "1",
        type: "gachapon-items",
        attributes: {
          gachaponId: "henesys",
          itemId: 2000000,
          quantity: 1,
          tier: "rare",
          weight: 10,
        },
      },
      {
        id: "2",
        type: "gachapon-items",
        attributes: {
          gachaponId: "henesys",
          itemId: 2000001,
          quantity: 1,
          tier: "rare",
          weight: 0,
        },
      },
    ]);
    mocks.getGlobalItems.mockResolvedValue([]);
    renderAt("henesys");
    await waitFor(() =>
      expect(screen.getByText("Henesys")).toBeInTheDocument(),
    );
    expect(screen.getByText(/exclude/i)).toBeInTheDocument();
  });
});
