import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ItemsPage } from "@/pages/ItemsPage";
import { itemsService, type ItemSearchFilters } from "@/services/api/items.service";

vi.mock("@/services/api/items.service", async () => {
  return {
    itemsService: {
      searchItems: vi.fn(async () => []),
    },
    buildItemSearchQuery: (await vi.importActual<typeof import("@/services/api/items.service")>("@/services/api/items.service")).buildItemSearchQuery,
  };
});

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: { id: "test-tenant", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } },
  }),
}));

function renderAt(initial: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initial]}>
        <Routes>
          <Route path="/items" element={<ItemsPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("ItemsPage", () => {
  beforeEach(() => {
    vi.mocked(itemsService.searchItems).mockClear();
  });

  it("fires the default browse query on mount with no params", async () => {
    renderAt("/items");
    await waitFor(() => {
      expect(itemsService.searchItems).toHaveBeenCalled();
    });
    const lastCall = vi.mocked(itemsService.searchItems).mock.calls.at(-1)![0] as ItemSearchFilters;
    expect(lastCall).toEqual({});
  });

  it("hydrates compartment + class from URL params", async () => {
    renderAt("/items?comp=equipment&class=warrior%2Cbowman");
    await waitFor(() => {
      const lastCall = vi.mocked(itemsService.searchItems).mock.calls.at(-1)![0] as ItemSearchFilters;
      expect(lastCall.compartment).toBe("equipment");
      expect(lastCall.classes?.sort()).toEqual(["bowman", "warrior"]);
    });
    expect(screen.getByRole("button", { name: "Warrior" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: "Bowman" })).toHaveAttribute("aria-pressed", "true");
  });

  it("hides the class toggle group when compartment is not equipment", () => {
    renderAt("/items?comp=use");
    expect(screen.queryByRole("button", { name: "Warrior" })).toBeNull();
  });

  it("'All Classes' is mutually exclusive with per-class toggles", async () => {
    const user = userEvent.setup();
    renderAt("/items?comp=equipment&class=warrior");
    expect(screen.getByRole("button", { name: "Warrior" })).toHaveAttribute("aria-pressed", "true");

    await user.click(screen.getByRole("button", { name: "All Classes" }));
    expect(screen.getByRole("button", { name: "All Classes" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: "Warrior" })).toHaveAttribute("aria-pressed", "false");
    expect(screen.getByRole("button", { name: "Warrior" })).toBeDisabled();

    await waitFor(() => {
      const lastCall = vi.mocked(itemsService.searchItems).mock.calls.at(-1)![0] as ItemSearchFilters;
      expect(lastCall.classes).toEqual(["any"]);
    });
  });
});
