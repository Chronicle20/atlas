import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { GachaponsPage } from "@/pages/GachaponsPage";
import { gachaponsService } from "@/services/api/gachapons.service";
import type { GachaponData } from "@/types/models/gachapon";

vi.mock("@/services/api/gachapons.service", () => ({
  gachaponsService: {
    getPage: vi.fn(),
  },
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: { id: "test-tenant", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } },
  }),
}));

function makeGachapon(id: string, name: string, kind = "gachapon"): GachaponData {
  return {
    id,
    type: "gachapons",
    attributes: {
      name,
      kind,
      npcIds: [9010000],
      commonWeight: 70,
      uncommonWeight: 25,
      rareWeight: 5,
    },
  };
}

function renderAt(initial: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initial]}>
        <Routes>
          <Route path="/gachapons" element={<GachaponsPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("GachaponsPage", () => {
  beforeEach(() => {
    vi.mocked(gachaponsService.getPage).mockReset();
  });

  it("requests page 1 at the default page size on mount", async () => {
    vi.mocked(gachaponsService.getPage).mockResolvedValue({
      data: [makeGachapon("1", "Standard Gachapon")],
      meta: { total: 1, page: { number: 1, size: 50, last: 1 } },
    });

    renderAt("/gachapons");

    await waitFor(() => {
      expect(gachaponsService.getPage).toHaveBeenCalledWith(
        { number: 1, size: 50 },
        expect.anything(),
      );
    });
  });

  it("renders the pager off meta.total / meta.page.last and requests the next page on click", async () => {
    vi.mocked(gachaponsService.getPage).mockResolvedValue({
      data: [makeGachapon("1", "Standard Gachapon")],
      meta: { total: 120, page: { number: 1, size: 50, last: 3 } },
    });

    renderAt("/gachapons");

    await screen.findByText(/Page 1 of 3/i);
    expect(screen.getByText(/120 results/i)).toBeInTheDocument();

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: /next page/i }));

    await waitFor(() => {
      const lastCall = vi.mocked(gachaponsService.getPage).mock.calls.at(-1)!;
      expect(lastCall[0]).toEqual({ number: 2, size: 50 });
    });
  });

  it("renders the Kind column so incubator machines are distinguishable from gachapon machines", async () => {
    vi.mocked(gachaponsService.getPage).mockResolvedValue({
      data: [
        makeGachapon("1", "Standard Gachapon", "gachapon"),
        makeGachapon("2", "Snow Island Incubator", "incubator"),
      ],
      meta: { total: 2, page: { number: 1, size: 50, last: 1 } },
    });

    renderAt("/gachapons");

    await screen.findByText("Standard Gachapon");
    expect(screen.getByRole("columnheader", { name: "Kind" })).toBeInTheDocument();
    expect(screen.getByRole("cell", { name: "gachapon" })).toBeInTheDocument();
    expect(screen.getByRole("cell", { name: "incubator" })).toBeInTheDocument();
  });

  it("hydrates the page number from ?page= in the URL", async () => {
    vi.mocked(gachaponsService.getPage).mockResolvedValue({
      data: [makeGachapon("1", "Standard Gachapon")],
      meta: { total: 120, page: { number: 3, size: 50, last: 3 } },
    });

    renderAt("/gachapons?page=3");

    await waitFor(() => {
      expect(gachaponsService.getPage).toHaveBeenCalledWith(
        { number: 3, size: 50 },
        expect.anything(),
      );
    });
  });
});
