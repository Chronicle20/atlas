import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { RecipeWidget } from "../RecipeWidget";
import type { Recipe } from "@/types/models/recipe";

vi.mock("@/lib/hooks/useNpcData", () => ({
  useNpcData: () => ({ name: "Tylus", iconUrl: undefined, isLoading: false }),
}));
vi.mock("@/lib/hooks/useItemData", () => ({
  useItemData: (id: number) => ({ name: `Item Name ${id}`, iconUrl: undefined, isLoading: false }),
}));
vi.mock("@/services/api/items.service", () => ({
  itemsService: {
    getItemName: async (id: string) => `Item Name ${id}`,
  },
}));
vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } },
  }),
}));

function wrap(children: React.ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={qc}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  );
}

const baseRecipe: Recipe = {
  id: "r1",
  npcId: 2040020,
  conversationId: "c1",
  stateId: "craft0",
  itemId: 1082007,
  materials: [{ itemId: 4011000, quantity: 3 }],
  mesoCost: 18000,
  stimulatorId: 0,
  stimulatorFailChance: 0,
};

describe("RecipeWidget", () => {
  it("renders the NPC name and meso cost (locale formatted)", () => {
    render(wrap(<RecipeWidget recipe={baseRecipe} />));
    expect(screen.getByText("Tylus")).toBeInTheDocument();
    expect(screen.getByText(/18,000\s+mesos/)).toBeInTheDocument();
  });

  it("renders one widget per material with the resolved name and quantity", () => {
    render(wrap(<RecipeWidget recipe={baseRecipe} />));
    expect(screen.getByText("Item Name 4011000")).toBeInTheDocument();
    expect(screen.getByText(/× 3/)).toBeInTheDocument();
  });

  it("does not render the With Stimulator badge when stimulatorId is 0", () => {
    render(wrap(<RecipeWidget recipe={baseRecipe} />));
    expect(screen.queryByText(/With Stimulator/i)).not.toBeInTheDocument();
  });

  it("renders the With Stimulator badge when stimulatorId > 0", () => {
    render(wrap(<RecipeWidget recipe={{ ...baseRecipe, stimulatorId: 4020009, stimulatorFailChance: 0.1 }} />));
    expect(screen.getByText(/With Stimulator/i)).toBeInTheDocument();
  });
});
