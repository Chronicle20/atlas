import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AddTeleportRockMapDialog } from "@/components/features/characters/AddTeleportRockMapDialog";

vi.mock("@/lib/hooks/api/useMaps", () => ({
  useMapsByName: () => ({
    data: [{ id: "100000000", attributes: { name: "Henesys" } }],
    isLoading: false,
  }),
}));
const addMap = vi.fn().mockResolvedValue({
  regular: [100000000],
  vip: [],
  regularCapacity: 5,
  vipCapacity: 10,
});
vi.mock("@/lib/hooks/api/useTeleportRocks", () => ({
  useAddTeleportRockMap: () => ({ mutateAsync: addMap, isPending: false }),
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

function renderDialog() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <AddTeleportRockMapDialog
        characterId="42"
        list="regular"
        existingMapIds={[]}
        open
        onOpenChange={vi.fn()}
      />
    </QueryClientProvider>,
  );
}

describe("AddTeleportRockMapDialog", () => {
  beforeEach(() => vi.clearAllMocks());

  it("adds the selected map", async () => {
    renderDialog();
    fireEvent.change(screen.getByPlaceholderText(/search maps/i), {
      target: { value: "hen" },
    });
    fireEvent.click(await screen.findByText("Henesys"));
    await waitFor(() =>
      expect(addMap).toHaveBeenCalledWith(
        expect.objectContaining({
          characterId: "42",
          list: "regular",
          mapId: 100000000,
        }),
      ),
    );
  });
});
