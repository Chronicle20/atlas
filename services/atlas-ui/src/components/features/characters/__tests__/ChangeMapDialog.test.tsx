import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ChangeMapDialog } from "@/components/features/characters/ChangeMapDialog";

// Mock the tenant context the component consumes via useTenant().
vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t" } }),
}));

// Mock the location hook (current map) + its key helper.
vi.mock("@/lib/hooks/api/useCharacterLocation", () => ({
  useCharacterLocation: () => ({
    data: {
      id: "7",
      type: "character-locations",
      attributes: { worldId: 0, channelId: 1, mapId: 100000000, instance: "" },
    },
  }),
  characterLocationKeys: {
    all: ["character-location"],
    detail: (tenantId: string | undefined, characterId: string) => [
      "character-location",
      tenantId,
      characterId,
    ],
  },
}));

// Mock the location service write.
vi.mock("@/services/api/locations.service", () => ({
  locationsService: { changeMap: vi.fn().mockResolvedValue(undefined) },
}));

// Mock toast so sonner doesn't pull in side effects.
vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

import { locationsService } from "@/services/api/locations.service";

const character = {
  id: "7",
  type: "characters",
  attributes: { name: "Hero", mapId: 100000000 },
} as never;

function renderDialog(overrides: Record<string, unknown> = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <ChangeMapDialog
        character={character}
        open={true}
        onOpenChange={vi.fn()}
        {...overrides}
      />
    </QueryClientProvider>,
  );
}

describe("ChangeMapDialog", () => {
  beforeEach(() => vi.clearAllMocks());

  it("writes the warp via the location endpoint", async () => {
    renderDialog();

    const input = screen.getByLabelText(/new map id/i);
    fireEvent.change(input, { target: { value: "104000000" } });
    fireEvent.click(screen.getByRole("button", { name: /change map/i }));

    await waitFor(() =>
      expect(locationsService.changeMap).toHaveBeenCalledWith("7", {
        mapId: 104000000,
      }),
    );
  });

  it("shows the current map from the location query", () => {
    renderDialog();
    expect(screen.getByText("100000000")).toBeInTheDocument();
  });
});
