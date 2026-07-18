import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { TeleportRockListCard } from "@/components/features/characters/TeleportRockListCard";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t" } }),
}));
vi.mock("@/lib/hooks/useItemData", () => ({
  useItemData: () => ({ iconUrl: "icon.png", name: "Teleport Rock" }),
}));
vi.mock("@/lib/hooks/api/useMaps", () => ({
  useMap: (id: string) => ({ data: { attributes: { name: `Map ${id}` } } }),
}));
const removeMap = vi.fn().mockResolvedValue({
  regular: [],
  vip: [],
  regularCapacity: 5,
  vipCapacity: 10,
});
vi.mock("@/lib/hooks/api/useTeleportRocks", () => ({
  useRemoveTeleportRockMap: () => ({
    mutateAsync: removeMap,
    isPending: false,
  }),
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

function renderCard(maps = [100000000]) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <TeleportRockListCard
        characterId="42"
        list="regular"
        maps={maps}
        capacity={5}
      />
    </QueryClientProvider>,
  );
}

describe("TeleportRockListCard", () => {
  beforeEach(() => vi.clearAllMocks());

  it("shows used-of-capacity", () => {
    renderCard();
    expect(screen.getByText("1 of 5")).toBeInTheDocument();
  });

  it("disables Add at capacity", () => {
    renderCard([1, 2, 3, 4, 5].map((n) => n * 100000000));
    expect(screen.getByRole("button", { name: /add/i })).toBeDisabled();
  });

  it("removes a map", async () => {
    renderCard();
    fireEvent.click(
      screen.getByRole("button", { name: /remove map 100000000/i }),
    );
    await waitFor(() =>
      expect(removeMap).toHaveBeenCalledWith(
        expect.objectContaining({
          characterId: "42",
          list: "regular",
          mapId: 100000000,
        }),
      ),
    );
  });
});
