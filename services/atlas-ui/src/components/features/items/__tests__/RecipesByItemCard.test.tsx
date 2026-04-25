import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { RecipesByItemCard } from "../RecipesByItemCard";
import type { Recipe } from "@/types/models/recipe";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } },
  }),
}));
vi.mock("@/lib/hooks/useNpcData", () => ({
  useNpcData: () => ({ name: "Tylus", iconUrl: undefined, isLoading: false }),
}));

const sample: Recipe = {
  id: "r1",
  npcId: 2040020,
  conversationId: "c1",
  stateId: "craft0",
  itemId: 1082007,
  materials: [],
  mesoCost: 18000,
  stimulatorId: 0,
  stimulatorFailChance: 0,
};

function wrap(children: React.ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={qc}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  );
}

describe("RecipesByItemCard", () => {
  it("renders the loading message while data loads", () => {
    render(wrap(<RecipesByItemCard recipes={undefined} isLoading={true} error={null} />));
    expect(screen.getByText("Loading craft recipes...")).toBeInTheDocument();
  });

  it("renders the empty-state copy when no recipes match", () => {
    render(wrap(<RecipesByItemCard recipes={[]} isLoading={false} error={null} />));
    expect(screen.getByText("No NPCs craft this item.")).toBeInTheDocument();
    expect(screen.getByText("Craftable At")).toBeInTheDocument();
  });

  it("renders one widget per recipe and shows the count in the title", () => {
    render(wrap(<RecipesByItemCard recipes={[sample, { ...sample, id: "r2", stateId: "craft1" }]} isLoading={false} error={null} />));
    expect(screen.getByText("Craftable At (2)")).toBeInTheDocument();
    expect(screen.getAllByText("Tylus").length).toBe(2);
  });

  it("renders the error inline when the query errors", () => {
    render(wrap(<RecipesByItemCard recipes={undefined} isLoading={false} error={new Error("boom")} />));
    expect(screen.getByText(/boom/)).toBeInTheDocument();
  });
});
