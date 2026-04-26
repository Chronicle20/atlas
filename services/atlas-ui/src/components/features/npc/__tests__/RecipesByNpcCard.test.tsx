import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { RecipesByNpcCard } from "../RecipesByNpcCard";
import type { Recipe } from "@/types/models/recipe";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } },
  }),
}));
vi.mock("@/lib/hooks/api/useNpcRecipes", () => ({
  useNpcRecipes: vi.fn(),
}));
import { useNpcRecipes } from "@/lib/hooks/api/useNpcRecipes";

function wrap(children: React.ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={qc}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  );
}

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

describe("RecipesByNpcCard", () => {
  it("renders nothing when the NPC crafts nothing", () => {
    (useNpcRecipes as unknown as ReturnType<typeof vi.fn>).mockReturnValue({ data: [], isLoading: false, error: null });
    const { container } = render(wrap(<RecipesByNpcCard npcId={2040020} />));
    expect(container.firstChild).toBeNull();
  });

  it("renders one row per craftable item with the count in the title", () => {
    (useNpcRecipes as unknown as ReturnType<typeof vi.fn>).mockReturnValue({
      data: [sample, { ...sample, id: "r2", itemId: 1082008 }],
      isLoading: false,
      error: null,
    });
    render(wrap(<RecipesByNpcCard npcId={2040020} />));
    expect(screen.getByText("Crafts (2)")).toBeInTheDocument();
    expect(screen.getAllByRole("link").length).toBeGreaterThanOrEqual(2);
  });

  it("renders the With Stimulator badge for stimulator recipes", () => {
    (useNpcRecipes as unknown as ReturnType<typeof vi.fn>).mockReturnValue({
      data: [{ ...sample, stimulatorId: 4020009, stimulatorFailChance: 0.1 }],
      isLoading: false,
      error: null,
    });
    render(wrap(<RecipesByNpcCard npcId={2040020} />));
    expect(screen.getByText(/With Stimulator/i)).toBeInTheDocument();
  });
});
